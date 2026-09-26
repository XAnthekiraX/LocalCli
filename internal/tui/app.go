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
	Puerto Puerto
	// Mapa es el keymap central (T-F012-05) y TeclaRes el resolver con la
	// máquina de estados del líder (T-F012-02/-03). Los componentes reciben
	// acciones ya resueltas: aquí abajo ya no se compara ninguna string de
	// tecla suelta (SPEC-KEYBINDS §Regla de oro).
	Mapa     *Keymap
	TeclaRes *KeyResolver
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
	// El mapa de teclas: el del usuario si su keys.json carga bien; el de
	// fábrica si el archivo no existe o es inválido (nunca un arranque roto
	// por una preferencia, T-F010-04). T-F012-05: se carga como Keymap
	// completo —líder, timeout y bindings múltiples— ya validado.
	mapa, err := CargarKeymap()
	if err != nil || ValidarAtajos(mapa.Entradas()) != nil {
		mapa = KeymapPorDefecto()
	}
	return &App{
		Puerto:   p,
		Mapa:     mapa,
		TeclaRes: NuevoKeyResolver(mapa),
		Atajos:   mapa.Entradas(),
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
// eventos sin goroutines propias, como manda la arquitectura Elm. Además pide
// de una vez la lista del selector de modelos: llega asíncrona, porque la
// bienvenida se pinta sin esperar a Ollama (SPEC-INTERFAZ §Reglas).
func (a *App) Init() tea.Cmd {
	// El orden importa: escucharCmd abre la suscripción y deja el canal listo;
	// si cargarModelos se enlazara primero, su cmd() leería el canal todavía
	// nil y devolvería nil — la escucha quedaría agotada tras el primer evento
	// (T-F010-06). Batch encadena de izquierda a derecha.
	return tea.Batch(a.cargarModelos(), a.escucharCmd())
}

// Update maneja las teclas, el tamaño y los mensajes que llegan por eventos.
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case tea.WindowSizeMsg:
		a.Ancho, a.Alto = m.Width, m.Height
		a.Entrada.FijarAncho(a.anchoChat())
		// El teclado propio del campo queda subordinado al KeyResolver
		// (T-F012-06): flechas, home/end, tab o ctrl+u/k son acciones del mapa,
		// no edición interna de bubbles. Se reaplica en cada cambio de tamaño
		// para que un textinput reconstruido nunca recupere sus atajos viejos.
		a.Entrada.DesactivarTeclasPropias()
		return a, nil
	case tea.KeyMsg:
		// Cada vista tiene su teclado: en la bienvenida solo se escribe, envía
		// y sale (welcome.go, T-F003); los atajos del panel, el selector y las
		// aprobaciones no existen allí.
		if a.Vista == VistaBienvenida {
			return a.teclaBienvenida(m)
		}
		return a.tecla(m)
	case LiderExpiradoMsg:
		// T-F012-03: el temporizador de la líder venció. Se vuelve a NORMAL y
		// se quita el indicador; si para entonces la espera ya terminó (la
		// secuencia se resolvió o se canceló), el mensaje sobrante se ignora.
		if a.TeclaRes != nil && a.TeclaRes.EsperandoLeader() {
			a.TeclaRes.Cancelar()
		}
		return a, nil
	case eventoMsg:
		a.AplicarEvento(m.Evento)
		// La escucha se re-arma: sin esto, tras el primer evento la vista
		// quedaría sorda (T-F010-06).
		return a, a.escucharCmd()
	case sesionesMsg:
		a.Selector.Abrir(m.Sesiones, a.Panel.SesionID)
		return a, nil
	case modelosMsg:
		// La lista del selector arriba cuando llega; si el puerto falló, queda
		// el aviso «sin modelos» y se puede escribir igual (SPEC-INTERFAZ).
		a.Bienvenida.fijarModelos(m.Modelos, m.Err)
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

// tecla resuelve una pulsación de la interfaz principal. T-F012-06: la tecla
// llega al KeyResolver (que conoce el mapa, la líder y el contexto) y desde
// aquí se despachan SOLO acciones — ningún switch compara ya strings de tecla
// suelta (SPEC-KEYBINDS §Regla de oro: "los componentes no comparan strings").
func (a *App) tecla(m tea.KeyMsg) (tea.Model, tea.Cmd) {
	// La ayuda se cierra con cualquier tecla y no deja rastro (T-F010-07):
	// es una superposición modal sin navegación propia.
	if a.AyudaAbierta {
		a.AyudaAbierta = false
		a.TeclaRes.Cancelar()
		return a, nil
	}

	// Cancelar pide confirmación: el texto escrito no se pierde mientras
	// decide (T-F010-08). Es un mini-modal propio: captura el teclado hasta
	// que el usuario responde s/n/esc. La líder y sus secuencias siguen
	// vivas incluso aquí (escape hatch de SPEC-KEYBINDS).
	if a.PidiendoCancelar {
		if nombre := strings.ToLower(strings.TrimSpace(m.String())); nombre == a.Mapa.Lider() ||
			a.TeclaRes.EsperandoLeader() {
			accion, cmdLider := a.TeclaRes.Resolver(m, ContextoModal)
			if a.TeclaRes.EsperandoLeader() {
				return a, cmdLider
			}
			if accion != AccionNinguna {
				return a.despachar(accion, m)
			}
			return a, nil
		}
		switch m.Type {
		case tea.KeyRunes:
			switch string(m.Runes) {
			case "s", "S":
				a.PidiendoCancelar = false
				a.Puerto.Cancelar(a.Panel.SesionID)
			case "n", "N":
				a.PidiendoCancelar = false
			}
		case tea.KeyEsc:
			a.PidiendoCancelar = false
		}
		return a, nil
	}

	// El contexto de resolución (T-F012-04): modal > aprobaciones > input >
	// vista. Con el selector o las aprobaciones abiertos las teclas simples
	// las consume el componente; la líder sigue funcionando desde cualquier
	// contexto (SPEC-KEYBINDS §Tecla líder: escape hatch).
	ctx := ContextoVista
	switch {
	case a.Selector.Abierto:
		ctx = ContextoModal
	case a.Aprobs.Abierto:
		ctx = ContextoAprobaciones
	}

	// Regla de oro de SPEC-KEYBINDS §Resolución por contexto: "con el input
	// enfocado, las letras sueltas son texto, nunca atajo". En la interfaz
	// principal el input está enfocado salvo cuando un modal o el panel de
	// aprobaciones reclama el teclado, así que el texto puro se entrega al
	// editor antes de resolver acciones — con una excepción: durante una
	// espera de líder, todas las pulsaciones van al resolver (el literal
	// «ctrl+x» no deja texto huérfano y la segunda tecla cierra la secuencia).
	if ctx == ContextoVista && !a.TeclaRes.EsperandoLeader() && EsEntradaDeTexto(m) {
		_, cmd := a.Entrada.Update(m)
		return a, cmd
	}

	accion, cmdLider := a.TeclaRes.Resolver(m, ctx)
	if accion == AccionNinguna && a.TeclaRes.EsperandoLeader() {
		// El resolver entró en LEADER: la pulsación no hace nada más; el
		// comando arma el temporizador que produce LiderExpiradoMsg
		// (T-F012-03).
		return a, cmdLider
	}

	// Con un modal o el panel de aprobaciones abierto, las teclas simples que
	// el mapa no reclama pertenecen al componente con el foco: se resuelven
	// aquí mismo contra su acción propia (esc cierra, ↑/↓ navegan, a/d
	// resuelven una línea) sin tocar el resolver (T-F012-04). En vista normal
	// manda el mapa tal cual, para que un mapa reasignado siga disparando la
	// misma acción (T-F010-04).
	if accion == AccionNinguna && (ctx == ContextoModal || ctx == ContextoAprobaciones) {
		accion = a.accionDelComponente(strings.ToLower(strings.TrimSpace(m.String())), ctx)
	}

	return a.despachar(accion, m)
}

// accionDelComponente traduce una tecla simple dentro del modal o del panel de
// aprobaciones a la acción de ese componente, usando el propio Keymap como
// tabla (las entradas «cerrar selector», «subir», «bajar», «aprobar»,
// «declinar»). No avanza la máquina de líder: es la parte del contexto que le
// toca al componente abierto.
func (a *App) accionDelComponente(nombre string, ctx Contexto) Accion {
	for _, e := range a.Mapa.Entradas() {
		for _, sec := range e.Secuencias {
			if sec.Paso2 != "" || sec.Paso1 != nombre {
				continue
			}
			switch e.Accion {
			case AccionCerrarSelector, AccionSubir, AccionBajar:
				return e.Accion
			case AccionAprobar, AccionDeclinar:
				if ctx == ContextoAprobaciones {
					return e.Accion
				}
			}
		}
	}
	return AccionNinguna
}

// despachar ejecuta el efecto de una acción resuelta por el KeyResolver. Es el
// único sitio de la app que traduce acciones a comportamiento: los componentes
// ya no comparan strings de tecla (T-F012-06).
func (a *App) despachar(accion Accion, m tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch accion {
	case AccionSalir:
		return a, tea.Quit
	case AccionPanel:
		a.Panel.Abierto = !a.Panel.Abierto
		// Plegar o desplegar cambia el reparto del ancho: la línea se
		// adapta al layout resultante, sin interrumpir nada (T-F006-02).
		a.Entrada.FijarAncho(a.anchoChat())
		return a, nil
	case AccionSelector, AccionModalSesiones:
		if a.Selector.Abierto {
			a.Selector.Cerrar()
			return a, nil
		}
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
		a.Aprobs.Abierto = false
		return a, nil
	case AccionEnviar:
		return a, a.enviar()
	case AccionSubir:
		// La navegación pertenece al componente con el foco (T-F012-04).
		switch {
		case a.Selector.Abierto:
			a.Selector.Mover(-1)
		case a.Aprobs.Abierto:
			a.Aprobs.Mover(-1)
		default:
			a.Bienvenida.MoverModelo(-1)
			a.fijarModeloEnPuerto()
		}
		return a, nil
	case AccionBajar:
		switch {
		case a.Selector.Abierto:
			a.Selector.Mover(1)
		case a.Aprobs.Abierto:
			a.Aprobs.Mover(1)
		default:
			a.Bienvenida.MoverModelo(1)
			a.fijarModeloEnPuerto()
		}
		return a, nil
	case AccionCiclarAgente:
		// plan ↔ build (SPEC-KEYBINDS §Acción agent_cycle). El puerto aún no
		// expone el cambio de agente: se registra como sistema para que el
		// usuario vea que la acción existe y está casada.
		a.Chat.AñadirSistema("cambio de agente: todavía no disponible")
		return a, nil
	case AccionModalModelos:
		// El modal de modelos vive hoy en la bienvenida; desde la interfaz
		// principal refresca la lista mientras exista el modal propio (T-F013).
		return a, a.cargarModelos()
	}

	// Lo que no es acción se entrega al componente con el foco: mientras el
	// selector está abierto, sus teclas propias (las que el mapa no reclama)
	// lo mueven a él; con las aprobaciones abiertas pasa igual; si no, lo
	// escrito lo compone la línea de entrada (INTERFACES §5).
	if a.Selector.Abierto {
		switch m.Type {
		case tea.KeyEnter:
			return a, a.elegirSesion()
		}
		return a, nil
	}
	if a.Aprobs.Abierto {
		return a, nil
	}
	_, cmd := a.Entrada.Update(m)
	return a, cmd
}

// accionGlobal marca las acciones que siguen vivas con un modal o el panel de
// aprobaciones abierto: salir, los toggles, cerrar y la navegación del
// componente con el foco. Las demás (aprobar/declinar/enviar…) pertenecen al
// contexto del input y quedan bloqueadas mientras otro zona manda.
func accionGlobal(a Accion) bool {
	switch a {
	case AccionSalir, AccionPanel, AccionSelector, AccionRazonamiento,
		AccionAprobaciones, AccionCancelar, AccionCerrarSelector, AccionAyuda,
		AccionModalModelos, AccionModalSesiones, AccionSubir, AccionBajar:
		return true
	}
	return false
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

// FijarMapa sustituye el mapa de teclas en caliente (reasignar un atajo no
// exige reiniciar, INTERFACES §4) y re-sincroniza el resolver y la lista que
// consumen ayuda y pruebas. El mapa nuevo viene ya validado por quien lo
// construyó; si llegara inválido, se rechaza entero y el anterior sigue vivo.
func (a *App) FijarMapa(mapa *Keymap) error {
	if err := ValidarAtajos(mapa.Entradas()); err != nil {
		return err
	}
	a.Mapa = mapa
	a.Atajos = mapa.Entradas()
	resolver := NuevoKeyResolver(mapa)
	if a.TeclaRes != nil {
		resolver.Estado = a.TeclaRes.Estado // si había una espera a medias, se hereda
	}
	a.TeclaRes = resolver
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
	// El indicador de líder pendiente (T-F012-06): mientras el resolver está
	// en LEADER se muestra «lider » junto a la entrada; al resolverse o
	// expirar la espera desaparece solo.
	if a.TeclaRes != nil && a.TeclaRes.EsperandoLeader() {
		cuerpo += " " + estiloAviso.Render("lider ")
	}
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
