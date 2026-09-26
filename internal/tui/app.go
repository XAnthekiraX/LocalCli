// app.go — T-B014-01: el programa Bubble Tea raíz.
//
// Fuente de verdad: SPEC-INTERFAZ §Disposición (chat a pantalla completa, panel
// plegado a la derecha), §Pantalla de bienvenida ("la primera vista al ejecutar
// `localcli`… solo hay logotipo, nombre con su versión y una línea de entrada")
// y §Reglas ("La bienvenida se pinta sin esperar a Ollama ni a la base: no
// depende de nada externo para mostrarse").
//
// De ahí que `Nuevo` no abra nada ni consulte nada: construye la vista y ya. Lo
// que espere a la base o al modelo es el arranque, no la pantalla.
//
// La bienvenida y la interfaz principal son la misma estructura en dos estados,
// no dos programas: cambiar de una a otra no puede perder la entrada escrita ni
// el canal de eventos ya abierto.
package tui

import (
	"context"
	_ "embed"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"localcli/internal/session"
)

// Nombre y Version de la aplicación, los dos datos que el panel muestra siempre.
const (
	Nombre  = "LocalCli"
	Version = "v0.1"
)

// logo.txt es la salida dorada del logotipo (SPEC-INTERFAZ §Arte canónico del
// logotipo): seis líneas sin espacios finales. Se incrusta para que la vista no
// dependa de la carpeta de trabajo.
//
//go:embed testdata/logo.txt
var LogoCanonico string

// Vista es dónde está la pantalla.
type Vista int

const (
	// VistaBienvenida: logotipo, nombre y una línea de entrada. Nada más.
	VistaBienvenida Vista = iota
	// VistaPrincipal: chat, panel, selector y aprobaciones.
	VistaPrincipal
)

// App es el modelo raíz de la TUI.
type App struct {
	Puerto   Puerto
	Atajos   []Atajo
	Vista    Vista
	Ancho    int
	Alto     int
	Chat     Chat
	Panel    Panel
	Razon    Razonamiento
	Selector Selector
	Aprobs   Aprobaciones

	entrada string
	ctx     context.Context
}

// Nuevo construye la vista. No lee la base, no habla con Ollama y no abre
// canales: solo deja la pantalla lista para pintarse.
func Nuevo(p Puerto) *App {
	if err := ValidarAtajos(AtajosPorDefecto()); err != nil {
		// Los atajos de fábrica son válidos por construcción; si dejan de
		// serlo, es un error de programación y conviene verlo pronto.
		panic(err)
	}
	return &App{
		Puerto: p,
		Atajos: AtajosPorDefecto(),
		Vista:  VistaBienvenida,
		Panel:  NuevoPanel(),
		// El razonamiento se muestra por defecto (SPEC-INTERFAZ: "se puede
		// ocultar", luego visible es el estado normal).
		Razon:  NuevoRazonamiento(),
		Aprobs: Aprobaciones{},
		ctx:    context.Background(),
	}
}

// Init no lanza nada: la suscripción a los eventos la monta el arranque, que es
// quien tiene el canal.
func (a *App) Init() tea.Cmd { return nil }

// Update maneja las teclas, el tamaño y los mensajes que llegan por eventos.
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case tea.WindowSizeMsg:
		a.Ancho, a.Alto = m.Width, m.Height
		return a, nil
	case tea.KeyMsg:
		return a.tecla(m)
	case eventoMsg:
		a.AplicarEvento(m.Evento)
		return a, nil
	case sesionesMsg:
		a.Selector.Abrir(m.Sesiones, a.Panel.SesionID)
		return a, nil
	case aprobacionesMsg:
		a.Aprobs.Fijar(m.Items)
		return a, nil
	case enviadoMsg:
		return a, nil
	case errorMsg:
		a.Chat.AñadirSistema(m.Error())
		return a, nil
	}
	return a, nil
}

// tecla resuelve una pulsación: primero el selector (que se lleva las flechas y
// el enter mientras está abierto), después el mapa de atajos, y solo si la tecla
// no es un atajo se escribe.
func (a *App) tecla(m tea.KeyMsg) (tea.Model, tea.Cmd) {
	if a.Selector.Abierto {
		switch m.String() {
		case "up":
			a.Selector.Mover(-1)
			return a, nil
		case "down":
			a.Selector.Mover(1)
			return a, nil
		case "enter":
			return a, a.elegirSesion()
		case "esc", "ctrl+s":
			a.Selector.Cerrar()
			return a, nil
		}
	}

	if accion, ok := AccionDe(a.Atajos, m.String()); ok {
		switch accion {
		case AccionSalir:
			return a, tea.Quit
		case AccionPanel:
			a.Panel.Abierto = !a.Panel.Abierto
			return a, nil
		case AccionSelector:
			return a, a.abrirSelector()
		case AccionRazonamiento:
			// Ocultar no detiene la generación: solo deja de pintarse.
			a.Razon.Alternar()
			return a, nil
		case AccionAprobar:
			return a, a.resolverAprobacion(true)
		case AccionDeclinar:
			return a, a.resolverAprobacion(false)
		case AccionPausar:
			return a, a.pausar()
		case AccionCancelar:
			a.Puerto.Cancelar(a.Panel.SesionID)
			return a, nil
		case AccionCerrarSelector:
			a.Selector.Cerrar()
			return a, nil
		case AccionEnviar:
			return a, a.enviar()
		}
	}

	switch m.Type {
	case tea.KeyRunes:
		a.entrada += string(m.Runes)
	case tea.KeySpace:
		a.entrada += " "
	case tea.KeyBackspace:
		if r := []rune(a.entrada); len(r) > 0 {
			a.entrada = string(r[:len(r)-1])
		}
	}
	return a, nil
}

// enviar manda lo escrito a la sesión activa. En la bienvenida, además, cambia
// de vista: la primera petición es la que abre la interfaz principal
// (SPEC-INTERFAZ §Pantalla de bienvenida).
func (a *App) enviar() tea.Cmd {
	texto := strings.TrimSpace(a.entrada)
	if texto == "" {
		return nil
	}
	id := a.Panel.SesionID
	if id == "" {
		ses, err := a.Puerto.ResolverActiva()
		if err != nil {
			a.Chat.AñadirSistema("no se pudo abrir la sesión: " + err.Error())
			return nil
		}
		a.activar(ses)
		id = ses.ID
	}
	a.entrada = ""
	a.Chat.AñadirUsuario(texto)
	a.Vista = VistaPrincipal
	return a.enviarCmd(id, texto)
}

func (a *App) enviarCmd(sesionID, texto string) tea.Cmd {
	return func() tea.Msg {
		if err := a.Puerto.Enviar(a.ctx, sesionID, texto); err != nil {
			return errorMsg{err: err}
		}
		return enviadoMsg{Sesion: sesionID, Texto: texto}
	}
}

// activar fija la sesión activa y limpia lo que era de la anterior: el chat de
// una sesión no se mezcla con el de otra (SPEC-SESIONES).
func (a *App) activar(ses *session.Sesion) {
	if ses == nil {
		return
	}
	a.Panel.SesionID = ses.ID
	a.Panel.Sesion = ses.Nombre
	a.Panel.Estado = ses.Estado
	a.Panel.Capa = ses.Capa
	if a.Panel.Sesion == "" {
		a.Panel.Sesion = ses.ID
	}
	a.Chat.Vaciar()
	a.Razon = NuevoRazonamiento()
}

func (a *App) abrirSelector() tea.Cmd {
	sesiones, err := a.Puerto.Listar()
	if err != nil {
		a.Chat.AñadirSistema("no se pudo listar las sesiones: " + err.Error())
		return nil
	}
	a.Selector.Abrir(sesiones, a.Panel.SesionID)
	return nil
}

// elegirSesion cambia a la sesión seleccionada sin detener nada.
func (a *App) elegirSesion() tea.Cmd {
	ses, ok := a.Selector.Elegida()
	if !ok {
		a.Selector.Cerrar()
		return nil
	}
	copia := ses
	a.activar(&copia)
	a.Selector.Cerrar()
	return nil
}

// resolverAprobacion manda la decisión de la línea seleccionada a la sesión
// dueña. Solo afecta a esa sesión (SPEC-INTERFAZ-ATAJOS).
func (a *App) resolverAprobacion(aprobar bool) tea.Cmd {
	ap, ok := a.Aprobs.Seleccionada()
	if !ok {
		return nil
	}
	if ap.Obsoleta {
		a.Aprobs.Resolver(ap.ID)
		return nil
	}
	if err := a.Puerto.Resolver(ap.ID, aprobar); err != nil {
		a.Chat.AñadirSistema("no se pudo resolver la aprobación: " + err.Error())
		return nil
	}
	a.Aprobs.Resolver(ap.ID)
	verbo := "declinada"
	if aprobar {
		verbo = "aprobada"
	}
	a.Chat.AñadirSistema("aprobación " + verbo + ": " + ap.Descripcion)
	return nil
}

// pausar detiene la cola en curso después del elemento actual.
func (a *App) pausar() tea.Cmd {
	if a.Panel.SesionID == "" {
		return nil
	}
	if err := a.Puerto.Pausar(a.Panel.SesionID); err != nil {
		a.Chat.AñadirSistema(err.Error())
	}
	return nil
}

// View pinta la pantalla: la bienvenida o la interfaz principal.
func (a *App) View() string {
	if a.Vista == VistaBienvenida {
		return a.viewBienvenida()
	}
	return a.viewPrincipal()
}

// viewBienvenida pinta el logotipo, el nombre con versión y la línea de entrada.
// Sin panel, sin selector y sin aprobaciones: esta pantalla no tiene más.
func (a *App) viewBienvenida() string {
	var b strings.Builder
	b.WriteString(strings.TrimRight(LogoCanonico, "\n"))
	b.WriteString("\n\n")
	b.WriteString(estiloMarca.Render(Nombre + " · " + Version))
	b.WriteString("\n\n")
	b.WriteString("En qué te ayudo hoy: " + a.entrada + "▌")
	b.WriteString("\n")
	return b.String()
}

// viewPrincipal pinta chat, razonamiento, respuesta en curso, aprobaciones y
// avisos; y el panel a la derecha cuando está abierto.
func (a *App) viewPrincipal() string {
	if a.Selector.Abierto {
		return a.Selector.Render() + "\n"
	}
	anchoChat := a.anchoChat()

	var partes []string
	if h := a.Chat.Render(anchoChat); h != "" {
		partes = append(partes, h)
	}
	// El razonamiento va encima de la respuesta, y separado de ella.
	if r := a.Razon.Render(anchoChat); r != "" {
		partes = append(partes, r)
	}
	if enCurso := a.Chat.EnCurso(); enCurso != "" {
		partes = append(partes, estiloAgente.Render(recortar(enCurso, anchoChat)))
	}
	if ap := a.Aprobs.Render(); ap != "" {
		partes = append(partes, ap)
	}

	cuerpo := strings.Join(partes, "\n\n")
	cuerpo += "\n\n" + estiloUsuario.Render("› ") + a.entrada + "▌"
	// La línea de aprobaciones pendientes se ve SIEMPRE, con el panel abierto o
	// cerrado: es la única que no se puede ocultar (SPEC-INTERFAZ).
	if aviso := a.Panel.AvisoAprobaciones(); aviso != "" {
		cuerpo += "\n" + aviso
	}

	if !a.Panel.Abierto {
		return cuerpo + "\n"
	}
	panel := a.Panel.Render(AnchoPanel - 4)
	return lipgloss.JoinHorizontal(lipgloss.Top, cuerpo, "  ", panel) + "\n"
}

// anchoChat es el ancho disponible para el chat: el total menos el panel cuando
// está abierto.
func (a *App) anchoChat() int {
	if a.Ancho <= 0 {
		return 80
	}
	if a.Panel.Abierto {
		return a.Ancho - AnchoPanel
	}
	return a.Ancho
}
