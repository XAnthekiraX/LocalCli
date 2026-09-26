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

	// Bienvenida es el estado de la primera pantalla; solo vive mientras esa
	// vista está activa (welcome.go, T-F003).
	Bienvenida Bienvenida

	// Entrada es la línea de texto de la interfaz principal (input.go, T-F004).
	Entrada Entrada

	// AyudaAbierta muestra la superposición con los atajos (T-F010-07). Sin
	// estado dentro: se abre, se lee y se cierra, sin efectos colaterales.
	AyudaAbierta bool

	// PidiendoCancelar espera la confirmación de ctrl+f (T-F010-08). El texto
	// escrito se conserva mientras tanto.
	PidiendoCancelar bool

	// eventos es el canal del motor, suscrito UNA vez en Init: cada evento
	// re-arma el comando sobre el MISMO canal, sin abrir suscripciones nuevas
	// (T-F010-06).
	eventos <-chan Evento
	baja    func()

	ctx context.Context
}

// Nuevo construye la vista. No lee la base, no habla con Ollama y no abre
// canales: solo deja la pantalla lista para pintarse.
func Nuevo(p Puerto) *App {
	// Los atajos: los del usuario si su keys.json carga bien; los de fábrica
	// si el archivo no existe o es inválido (nunca un arranque roto por una
	// preferencia, T-F010-04).
	atajos, err := CargarKeys()
	if err != nil {
		atajos = AtajosPorDefecto()
	}
	if err := ValidarAtajos(atajos); err != nil {
		// Los atajos de fábrica son válidos por construcción; si dejan de
		// serlo, es un error de programación y conviene verlo pronto.
		panic(err)
	}
	return &App{
		Puerto: p,
		Atajos: atajos,
		Vista:  VistaBienvenida,
		Panel:  NuevoPanel(),
		// El razonamiento se muestra por defecto (SPEC-INTERFAZ: "se puede
		// ocultar", luego visible es el estado normal).
		Razon:      NuevoRazonamiento(),
		Aprobs:     Aprobaciones{},
		Bienvenida: NuevaBienvenida(),
		Entrada:    NuevaEntrada(),
		ctx:        context.Background(),
	}
}

// Init arma la escucha del canal del motor (T-F010-06): el comando espera el
// siguiente evento y el bucle lo relanza tras cada uno. Así la vista recibe los
// eventos sin goroutines propias, como manda la arquitectura Elm.
func (a *App) Init() tea.Cmd { return a.escucharCmd() }

// Update maneja las teclas, el tamaño y los mensajes que llegan por eventos.
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case tea.WindowSizeMsg:
		a.Ancho, a.Alto = m.Width, m.Height
		a.Entrada.FijarAncho(a.anchoChat())
		return a, nil
	case tea.KeyMsg:
		// Cada vista tiene su teclado: en la bienvenida solo se escribe, envía
		// y sale (welcome.go, T-F003); los atajos del panel, el selector y las
		// aprobaciones no existen allí.
		if a.Vista == VistaBienvenida {
			return a.teclaBienvenida(m)
		}
		return a.tecla(m)
	case eventoMsg:
		a.AplicarEvento(m.Evento)
		// La escucha se re-arma: sin esto, tras el primer evento la vista
		// quedaría sorda (T-F010-06).
		return a, a.escucharCmd()
	case sesionesMsg:
		a.Selector.Abrir(m.Sesiones, a.Panel.SesionID)
		return a, nil
	case aprobacionesMsg:
		a.Aprobs.Fijar(m.Items)
		// La instantánea de pendientes alimenta también el contador del panel.
		a.Panel.Aprobaciones = a.Aprobs.Pendientes()
		return a, nil
	case historialMsg:
		// La carga llega cuando la sesión pedida sigue siendo la activa; si el
		// usuario cambió de sesión mientras volaba, se descarta: el chat nunca
		// muestra el historial de otra sesión (T-F005-05).
		if m.Sesion == a.Panel.SesionID {
			a.Chat.Cargar(m.Mensajes)
		}
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

	// La ayuda se cierra con cualquier tecla y no deja rastro (T-F010-07).
	if a.AyudaAbierta {
		a.AyudaAbierta = false
		return a, nil
	}

	// Cancelar pide confirmación: el texto escrito no se pierde mientras
	// decide (T-F010-08).
	if a.PidiendoCancelar {
		switch m.String() {
		case "s", "S":
			a.PidiendoCancelar = false
			a.Puerto.Cancelar(a.Panel.SesionID)
			return a, nil
		case "n", "N", "esc":
			a.PidiendoCancelar = false
			return a, nil
		}
		return a, nil
	}

	// Con el panel de aprobaciones abierto, a/d resuelven la línea
	// seleccionada y esc lo cierra (INTERFACES §4, T-F008-05).
	if a.Aprobs.Abierto {
		switch m.String() {
		case "up":
			a.Aprobs.Mover(-1)
			return a, nil
		case "down":
			a.Aprobs.Mover(1)
			return a, nil
		case "a":
			return a, a.resolverAprobacion(true)
		case "d":
			return a, a.resolverAprobacion(false)
		case "esc", "ctrl+a":
			a.Aprobs.Abierto = false
			return a, nil
		}
	}

	if accion, ok := AccionDe(a.Atajos, m.String()); ok {
		switch accion {
		case AccionSalir:
			return a, tea.Quit
		case AccionPanel:
			a.Panel.Abierto = !a.Panel.Abierto
			// Plegar o desplegar cambia el reparto del ancho: la línea se
			// adapta al layout resultante, sin interrumpir nada (T-F006-02).
			a.Entrada.FijarAncho(a.anchoChat())
			return a, nil
		case AccionSelector:
			return a, a.abrirSelector()
		case AccionRazonamiento:
			// Ocultar no detiene la generación: solo deja de pintarse.
			a.Razon.Alternar()
			return a, nil
		case AccionAprobaciones:
			a.Aprobs.Abierto = !a.Aprobs.Abierto
			return a, nil
		case AccionAprobar:
			return a, a.resolverAprobacion(true)
		case AccionDeclinar:
			return a, a.resolverAprobacion(false)
		case AccionPausar:
			return a, a.pausar()
		case AccionCancelar:
			// Cancelar pide confirmación antes de cortar el trabajo (T-F010-08).
			a.PidiendoCancelar = true
			return a, nil
		case AccionAyuda:
			a.AyudaAbierta = !a.AyudaAbierta
			return a, nil
		case AccionCerrarSelector:
			a.Selector.Cerrar()
			return a, nil
		case AccionEnviar:
			return a, a.enviar()
		}
	}

	// Lo que no es atajo se escribe: la línea de entrada lo compone, también
	// mientras la sesión está generando (INTERFACES §5).
	_, cmd := a.Entrada.Update(m)
	return a, cmd
}

// enviar manda lo escrito a la sesión activa. En la bienvenida, además, cambia
// de vista: la primera petición es la que abre la interfaz principal
// (SPEC-INTERFAZ §Pantalla de bienvenida).
func (a *App) enviar() tea.Cmd {
	texto := strings.TrimSpace(a.Entrada.Texto())
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
	a.Entrada.Limpiar()
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
// una sesión no se mezcla con el de otra (SPEC-SESIONES). El historial no está
// en memoria: se pide a la sesión y llega por su comando (T-F005-05).
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

// cmdHistorial pide la conversación de la sesión activa. La lectura llega como
// mensaje: la vista no consulta la base (INTERFACES §3).
func (a *App) cmdHistorial(sesionID string) tea.Cmd {
	return func() tea.Msg {
		ms, err := a.Puerto.Historial(sesionID)
		if err != nil {
			return errorMsg{err: err}
		}
		return historialMsg{Sesion: sesionID, Mensajes: ms}
	}
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
	// El chat cambia a esa sesión: su historial llega como dato y se pinta al
	// llegar, sin detener lo que siga corriendo en segundo plano (T-F005-05).
	return a.cmdHistorial(copia.ID)
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

// View pinta la pantalla: la bienvenida, la interfaz principal o la ayuda.
func (a *App) View() string {
	if a.AyudaAbierta {
		return a.viewAyuda()
	}
	if a.Vista == VistaBienvenida {
		return a.viewBienvenida()
	}
	return a.viewPrincipal()
}

// viewAyuda lista los atajos disponibles y la acción de cada uno
// (SPEC-INTERFAZ-ATAJOS, criterio: "Se listan los atajos disponibles y la
// acción de cada uno"). Cualquier tecla la cierra y no deja efectos.
func (a *App) viewAyuda() string {
	var b strings.Builder
	b.WriteString(estiloTitulo.Render("ATAJOS") + "\n\n")
	b.WriteString(AyudaAtajos(a.Atajos))
	b.WriteString("\n(cualquier tecla cierra la ayuda)")
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
	if h := recortarAlto(a.Chat.Render(anchoChat), a.Alto); h != "" {
		partes = append(partes, h)
	}
	// El intercambio en curso: razonamiento arriba de la respuesta y separado
	// de ella (SPEC-INTERFAZ §Razonamiento del modelo). El dibujo vive en
	// styles.go (T-F002).
	if x := renderIntercambio(a.Razon.Texto(), a.Chat.EnCurso(), a.Razon.Visible, a.Razon.NoDisponible, anchoChat); x != "" {
		partes = append(partes, x)
	}
	// Las propuestas pendientes de esta sesión se ven dentro del chat
	// (SPEC-INTERFAZ §Zonas 1, T-F005-06).
	if prop := a.Chat.RenderPropuestas(); prop != "" {
		partes = append(partes, prop)
	}
	if ap := a.Aprobs.Render(); ap != "" {
		partes = append(partes, ap)
	}

	cuerpo := strings.Join(partes, "\n\n")
	cuerpo += "\n\n" + estiloUsuario.Render("› ") + a.Entrada.View()
	// La confirmación de cancelar se pregunta en línea; mientras tanto, lo
	// escrito queda a salvo (T-F010-08).
	if a.PidiendoCancelar {
		cuerpo += "\n" + estiloAviso.Render("¿cancelar el trabajo en curso? (s/n)")
	}
	// El aviso de aprobaciones pendientes es la única información fuera del
	// panel de datos, y se ve con el panel CERRADO: abierto, el dato ya está
	// en su fila y el aviso no se repite (T-F009-03).
	if !a.Panel.Abierto {
		if aviso := a.Panel.AvisoAprobaciones(); aviso != "" {
			cuerpo += "\n" + aviso
		}
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
