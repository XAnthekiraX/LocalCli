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
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"localcli/internal/session"
)

// Nombre y Version de la aplicación, los dos datos que el panel muestra siempre.
const (
	Nombre  = "LocalCli"
	Version = "v0.1 alpha"
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
	// VistaPrincipal: chat, panel, modal de sesiones y aprobaciones.
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
	Aprobs   Aprobaciones

	// Bienvenida es el estado de la primera pantalla; solo vive mientras esa
	// vista está activa (welcome.go, T-F003).
	Bienvenida Bienvenida

	// Modelo es el nombre del modelo en uso: el detectado en el arranque o el
	// último elegido en el modal de modelos (SPEC-INTERFAZ §Línea de modelo).
	// Solo lo escribe quien lo elige —el modal— y `fijarModeloEnPuerto` lo
	// entrega al motor.
	Modelo string
	// Agente es el agente activo —`AgentePlan` o `AgenteBuild`—, el que pinta el
	// indicador a la izquierda del input y el que viaja con cada petición
	// (SPEC-INTERFAZ §Zonas 2, T-F015). Lo alterna `Tab`; con un modal abierto
	// la acción no llega a cambiarlo.
	Agente string
	// Los tres modales (SPEC-INTERFAZ §Modales: "Tres modales centrados
	// comparten el mismo comportamiento: uno abierto a la vez… Esc cierra sin
	// cambios"). Todos se abren con acciones globales del mapa —session_picker,
	// command_palette y model_picker— y se cierran con `dismiss`.
	Modelos     ModelsModal
	Sesiones    SessionsModal
	AtajosModal KeysModal

	// Entrada es la línea de texto de la interfaz principal (input.go, T-F004).
	Entrada Entrada

	// PidiendoCancelar espera la confirmación de ctrl+f (T-F010-08). El texto
	// escrito se conserva mientras tanto.
	PidiendoCancelar bool
	// PidiendoEliminarSesion guarda el id de la sesión que trabaja y que se
	// quiere borrar: la confirmación «¿seguro que deseas eliminarla?» captura el
	// teclado hasta que se responde s/n/esc (SPEC-SESIONES).
	PidiendoEliminarSesion string

	// enTurno dice si la sesión activa llegó a ponerse a trabajar para el turno
	// en curso. Distingue el `inactiva` de una cancelación —que cierra el
	// turno— del `inactiva` previo a arrancar, que no lo toca.
	enTurno bool
	// latido es la generación del contador en vivo: identifica la cadena de
	// ticks vigente. Al enviar se incrementa, y una cadena de una generación
	// anterior se detiene en su siguiente tick, así que nunca se acumulan.
	latido uint64

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
		Vista:    VistaBienvenida,
		Panel:    NuevoPanel(),
		// El modelo en uso se lee una vez del motor, que ya lo detectó al
		// arrancar: es un dato en memoria, no una llamada a Ollama, así que la
		// bienvenida se sigue pintando sin esperar a nada externo
		// (SPEC-INTERFAZ §Línea de modelo, DOMAIN §3).
		Modelo: p.ModeloActual(),
		// El razonamiento se muestra por defecto (SPEC-INTERFAZ: "se puede
		// ocultar", luego visible es el estado normal).
		Razon:      NuevoRazonamiento(),
		Aprobs:     Aprobaciones{},
		Bienvenida: NuevaBienvenida(),
		Entrada:    NuevaEntrada(),
		// El agente arranca en `plan`, el que solo analiza y propone
		// (SPEC-INTERFAZ §Zonas 2: el indicador muestra el agente activo, y el
		// reparto de etapas lo hace el motor).
		Agente: AgentePlan,
		ctx:    context.Background(),
	}
}

// Init arma la escucha del canal del motor (T-F010-06): el comando espera el
// siguiente evento y el bucle lo relanza tras cada uno. Así la vista recibe los
// eventos sin goroutines propias, como manda la arquitectura Elm.
//
// T-F013: aquí ya no se pide la lista de modelos. Se pide al abrir el modal
// (SPEC-INTERFAZ §Modal de modelos: "La lista se pide a Ollama al abrir el
// modal (no en el arranque)"), así que arrancar no toca Ollama.
func (a *App) Init() tea.Cmd {
	return a.escucharCmd()
}

// tickMsg es el latido del contador en vivo: cada segundo mientras hay una
// respuesta en camino, para que el tiempo corra en pantalla aunque el modelo
// todavía no haya emitido un token. `gen` es la generación del contador a la que
// pertenece el latido: solo la cadena vigente se re-arma.
type tickMsg struct{ gen uint64 }

// tickCmd programa el siguiente latido de la generación dada. El comando se
// re-arma a sí mismo mientras el turno siga vivo (uno por segundo, no más).
func tickCmd(gen uint64) tea.Cmd {
	return tea.Tick(time.Second, func(time.Time) tea.Msg { return tickMsg{gen: gen} })
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
		// La confirmación de borrado captura el teclado en cualquier vista:
		// mientras está activa, s/n/esc y nada más (SPEC-SESIONES).
		if a.PidiendoEliminarSesion != "" {
			return a.teclaConfirmarEliminar(m)
		}
		// Cada vista tiene su teclado: en la bienvenida solo se escribe, envía
		// y sale (welcome.go, T-F003); los atajos del panel, los modales y las
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
		// La lista llega cuando la lectura a `store` termina: rellena el modal
		// que la pidió. Si ya no está abierto, el mensaje se ignora (llegó
		// tarde). Un fallo de lectura no bloquea: el modal avisa «sin sesiones»
		// y el chat deja la línea para cuando se cierre (T-F014-04).
		if m.Err != nil {
			a.Chat.AñadirSistema("no se pudo listar las sesiones: " + m.Err.Error())
		}
		a.Sesiones.FijarSesiones(m.Sesiones)
		return a, nil
	case modelosMsg:
		// La lista llega cuando Ollama responde: rellena el modal que la pidió.
		// Si ya no está abierto, el mensaje se ignora (llegó tarde); si el
		// puerto falló, queda el aviso «sin modelos» y se puede seguir
		// escribiendo igual (SPEC-INTERFAZ §Modal de modelos).
		a.Modelos.FijarModelos(m.Modelos, m.Err)
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
		// La petición ya salió: arranca una cadena de latidos nueva. Incrementar
		// la generación invalida la anterior, de modo que un envío justo al
		// cerrar un turno no deja dos cadenas corriendo.
		a.latido++
		return a, tickCmd(a.latido)
	case tickMsg:
		// El contador late solo mientras hay un turno vivo y la cadena sea la
		// vigente; al cerrarse el turno (o quedar obsoleta la generación), el
		// siguiente latido ya no se re-arma y el contador se detiene solo.
		if a.Chat.HayTurno() && m.gen == a.latido {
			return a, tickCmd(m.gen)
		}
		return a, nil
	case errorMsg:
		// Un envío que no llegó a arrancar no deja turno que fechar; uno que ya
		// está trabajando se deja en paz (el error puede ser de otra lectura).
		if !a.enTurno {
			a.Chat.CancelarTurno()
		}
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
	// Cancelar pide confirmación: el texto escrito no se pierde mientras
	// decide (T-F010-08). Es un mini-modal propio: captura el teclado hasta
	// que el usuario responde s/n/esc. La líder y sus secuencias siguen
	// vivas incluso aquí (escape hatch de SPEC-KEYBINDS), porque al resolver
	// antes de mirar las letras, la secuencia se resuelve como en cualquier
	// otro contexto.
	if a.PidiendoCancelar {
		accion, cmdLider := a.TeclaRes.Resolver(m, ContextoModal)
		if a.TeclaRes.EsperandoLeader() {
			return a, cmdLider
		}
		if accion != AccionNinguna {
			return a.despachar(accion, m)
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
	// vista. Con un modal o el panel de aprobaciones abiertos las teclas
	// simples las consume el componente; la líder sigue funcionando desde
	// cualquier contexto (SPEC-KEYBINDS §Tecla líder: escape hatch).
	ctx := ContextoVista
	switch {
	case a.modalAbierto():
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
	return a.despachar(accion, m)
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
	case AccionSelector:
		// El modal de sesiones es global: se abre desde la interfaz principal
		// y pide la lista en el acto (SPEC-INTERFAZ §Cambiar de sesión).
		if a.Sesiones.Abierto {
			a.Sesiones.Cerrar()
			return a, nil
		}
		return a, a.abrirModalSesiones()
	case AccionRazonamiento:
		// Ocultar no detiene la generación: solo deja de pintarse.
		a.Razon.Alternar()
		return a, nil
	case AccionSesionNueva:
		// `Ctrl+X n` crea una sesión nueva y la deja activa; solo desde la
		// vista principal (SPEC-KEYBINDS `session_new`).
		return a, a.nuevaSesion()
	case AccionEliminarSesion:
		// `Ctrl+D` borra la sesión resaltada, y solo tiene sentido con el
		// modal de sesiones abierto (SPEC-KEYBINDS `session_delete`).
		if !a.Sesiones.Abierto {
			return a, nil
		}
		return a, a.eliminarSesion()
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
		// El modal de atajos sustituye a la ayuda clásica: se abre con Ctrl+P
		// desde cualquier contexto y es de solo lectura, así que no navega ni
		// aplica (SPEC-KEYBINDS §Acción: command_palette).
		if a.AtajosModal.Abierto {
			a.AtajosModal.Cerrar()
			return a, nil
		}
		return a, a.abrirModalAtajos()
	case AccionCerrarSelector:
		// Esc —`dismiss`— descarta lo que haya abierto: cualquiera de los tres
		// modales y el panel de aprobaciones. Sin nada abierto no hace nada
		// visible (SPEC-INTERFAZ §Modales, criterio: "Esc cierra cualquier
		// modal sin cambiar nada").
		a.cerrarModales()
		a.Aprobs.Abierto = false
		return a, nil
	case AccionEnviar:
		// Enter es ambiguo y lo resuelve lo que está abierto: con el modal de
		// modelos aplica el resaltado, con el de sesiones abre esa sesión y en
		// la interfaz principal envía la petición (INTERFACES §4: "Enter aplica
		// lo resaltado en el modal y lo cierra"). Con el de atajos no hay nada
		// que aplicar: es de solo lectura y la tecla no llega a la vista.
		switch {
		case a.Modelos.Abierto:
			return a, a.aplicarModelo()
		case a.Sesiones.Abierto:
			return a, a.elegirSesion()
		case a.AtajosModal.Abierto:
			return a, nil
		}
		return a, a.enviar()
	case AccionSubir:
		// La navegación pertenece al componente con el foco (T-F012-04). El
		// modal de atajos no aparece: es una tabla de solo lectura.
		switch {
		case a.Modelos.Abierto:
			a.Modelos.Mover(-1)
		case a.Sesiones.Abierto:
			a.Sesiones.Mover(-1)
		case a.Aprobs.Abierto:
			a.Aprobs.Mover(-1)
		}
		return a, nil
	case AccionBajar:
		switch {
		case a.Modelos.Abierto:
			a.Modelos.Mover(1)
		case a.Sesiones.Abierto:
			a.Sesiones.Mover(1)
		case a.Aprobs.Abierto:
			a.Aprobs.Mover(1)
		}
		return a, nil
	case AccionCiclarAgente:
		// plan ↔ build (SPEC-KEYBINDS §Acción `agent_cycle`; SPEC-INTERFAZ
		// §Zonas 2: "Cambia al instante con Tab"). La acción es de contexto de
		// vista, así que con un modal abierto no llega hasta aquí: el indicador
		// y lo que se envía son los del último `Tab` pulsado.
		a.ciclarAgente()
		return a, nil
	case AccionModalModelos:
		// El modal de modelos es global: se abre desde la bienvenida y desde la
		// interfaz principal (INTERFACES §4). Cerrarlo es lo mismo que abrirlo
		// cuando ya está abierto.
		if a.Modelos.Abierto {
			a.Modelos.Cerrar()
			return a, nil
		}
		return a, a.abrirModalModelos()
	}

	// Lo que no es acción se entrega al componente con el foco: con un modal
	// abierto solo viven sus teclas, ya tratadas arriba, y el resto se traga
	// aquí — nada llega a la línea de entrada (SPEC-INTERFAZ §Modal de modelos);
	// con las aprobaciones abiertas pasa igual; si no, lo escrito lo compone la
	// línea de entrada (INTERFACES §5).
	if a.modalAbierto() {
		return a, nil
	}
	if a.Aprobs.Abierto {
		return a, nil
	}
	_, cmd := a.Entrada.Update(m)
	return a, cmd
}

// nuevaSesion crea una sesión y la deja activa: el chat vuelve a empezar en
// blanco con esa sesión y la vista queda en la principal (SPEC-SESIONES,
// SPEC-KEYBINDS `session_new`).
func (a *App) nuevaSesion() tea.Cmd {
	ses, err := a.Puerto.Crear()
	if err != nil {
		a.Chat.AñadirSistema("no se pudo crear la sesión: " + err.Error())
		return nil
	}
	// activar ya vacía el chat y el razonamiento de la sesión anterior: una
	// sesión nueva no hereda nada de la que se estaba viendo.
	a.activar(ses)
	a.Vista = VistaPrincipal
	return nil
}

// eliminarSesion borra la sesión resaltada. Si está trabajando (o esperando
// permiso) no la borra al instante: pide confirmación primero, porque en su
// cascada se va todo lo suyo (SPEC-SESIONES).
func (a *App) eliminarSesion() tea.Cmd {
	ses, ok := a.Sesiones.SesionElegida()
	if !ok {
		return nil
	}
	if ses.EnCurso() {
		a.PidiendoEliminarSesion = ses.ID
		return nil
	}
	return a.ejecutarEliminar(ses.ID)
}

// ejecutarEliminar mata la sesión y refresca el modal. Si era la activa, se
// retoma otra del proyecto (o se crea) para que el chat no quede apuntando a una
// sesión que ya no existe.
func (a *App) ejecutarEliminar(id string) tea.Cmd {
	if err := a.Puerto.Eliminar(id); err != nil {
		a.Chat.AñadirSistema("no se pudo eliminar la sesión: " + err.Error())
		return nil
	}
	if a.Panel.SesionID == id {
		ses, err := a.Puerto.ResolverActiva()
		if err != nil {
			a.Chat.AñadirSistema("no se pudo abrir la sesión: " + err.Error())
			return nil
		}
		a.activar(ses)
		return tea.Batch(a.cmdHistorial(ses.ID), a.abrirModalSesiones())
	}
	// El modal sigue abierto con la lista ya refrescada.
	return a.abrirModalSesiones()
}

// teclaConfirmarEliminar atiende la confirmación de borrado: s confirma, n y
// esc cancelan. Mientras tanto, ninguna otra tecla pasa a la vista de abajo.
func (a *App) teclaConfirmarEliminar(m tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.Type == tea.KeyCtrlC {
		return a, tea.Quit
	}
	switch m.Type {
	case tea.KeyRunes:
		switch string(m.Runes) {
		case "s", "S":
			id := a.PidiendoEliminarSesion
			a.PidiendoEliminarSesion = ""
			return a, a.ejecutarEliminar(id)
		case "n", "N":
			a.PidiendoEliminarSesion = ""
		}
	case tea.KeyEsc:
		a.PidiendoEliminarSesion = ""
	}
	return a, nil
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

// enviarCmd entrega la petición al puerto con el agente que está activo. El
// indicador no es decorativo: es lo que el motor recibe, así que la petición
// sale con el agente elegido y no con el que la vista suponga (T-F015-04,
// SPEC-INTERFAZ §Zonas 2: "Cambia de comportamiento según el agente activo").
func (a *App) enviarCmd(sesionID, texto string) tea.Cmd {
	return func() tea.Msg {
		if err := a.Puerto.Enviar(a.ctx, sesionID, a.Agente, texto); err != nil {
			return errorMsg{err: err}
		}
		return enviadoMsg{Sesion: sesionID, Texto: texto}
	}
}

// ciclarAgente alterna `plan` ↔ `build` y lo propaga a lo único que lo pinta y
// a lo único que lo consume: el indicador de las dos vistas y la fila del
// panel. No toca el motor: qué hace cada agente lo decide él (DOMAIN §2, el
// input no valida reglas de negocio).
func (a *App) ciclarAgente() {
	if a.Agente == AgenteBuild {
		a.Agente = AgentePlan
	} else {
		a.Agente = AgenteBuild
	}
	a.Entrada.FijarAgente(a.Agente)
	a.Panel.Agente = a.Agente
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
	// El turno que se seguía era de la sesión anterior: al cambiar de sesión no
	// se sigue midiendo, para que un evento suyo no cierre el turno de esta.
	a.enTurno = false
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

// modalAbierto dice si hay un modal en pantalla. Los tres comparten la misma
// mecánica y solo uno puede estar abierto a la vez (SPEC-INTERFAZ §Modales:
// "uno abierto a la vez").
func (a *App) modalAbierto() bool {
	return a.Modelos.Abierto || a.Sesiones.Abierto || a.AtajosModal.Abierto
}

// cerrarModales cierra los tres. Es lo que hace `dismiss` (Esc) y también lo que
// hacen las aperturas: abrir un modal es siempre abrirlo en lugar del que
// hubiera (SPEC-INTERFAZ §Modales).
func (a *App) cerrarModales() {
	a.Modelos.Cerrar()
	a.Sesiones.Cerrar()
	a.AtajosModal.Cerrar()
}

// abrirModalSesiones muestra el modal de sesiones y pide la lista al puerto en
// el acto (SPEC-INTERFAZ §Cambiar de sesión: la lista se pide al abrir, no hay
// ninguna lista permanente). El modal aparece ya, con «cargando…», y se rellena
// cuando llega la respuesta: la pantalla no se bloquea mientras tanto.
func (a *App) abrirModalSesiones() tea.Cmd {
	a.cerrarModales()
	a.Sesiones.AbrirSesiones(a.Panel.SesionID)
	return a.cargarSesiones()
}

// cargarSesiones pregunta al puerto las sesiones del proyecto. Es un comando
// asíncrono: la respuesta llega como `sesionesMsg` y la atiende el modal
// (T-F014-04). La vista no consulta la base (INTERFACES §3).
func (a *App) cargarSesiones() tea.Cmd {
	return func() tea.Msg {
		sesiones, err := a.Puerto.Listar()
		return sesionesMsg{Sesiones: sesiones, Err: err}
	}
}

// abrirModalAtajos muestra la tabla de acciones y teclas del mapa vigente. No
// pide nada a nadie: el listado sale del keymap que ya está en memoria
// (SPEC-INTERFAZ §Modales, fila Atajos).
func (a *App) abrirModalAtajos() tea.Cmd {
	a.cerrarModales()
	a.AtajosModal.AbrirAtajos(a.Atajos)
	return nil
}

// abrirModalModelos muestra el modal y pide la lista a Ollama en el acto
// (SPEC-INTERFAZ §Modal de modelos: "La lista se pide a Ollama al abrir el
// modal (no en el arranque)"). El modal aparece ya, con «cargando…», y se
// rellena cuando llega la respuesta: la pantalla no se bloquea mientras tanto.
func (a *App) abrirModalModelos() tea.Cmd {
	a.cerrarModales()
	a.Modelos.AbrirModelos()
	return a.cargarModelos()
}

// cargarModelos pregunta al puerto la lista de modelos. Es un comando
// asíncrono: la respuesta llega como `modelosMsg` y quien la atiende es el
// modal (T-F013-03). Si Ollama no responde, el aviso «sin modelos» aparece y
// nada se bloquea.
func (a *App) cargarModelos() tea.Cmd {
	return func() tea.Msg {
		modelos, err := a.Puerto.Modelos()
		return modelosMsg{Modelos: modelos, Err: err}
	}
}

// aplicarModelo entrega al motor el modelo resaltado y lo deja como modelo en
// uso: la línea de modelo de la bienvenida lo refleja y la primera petición
// sale con él (SPEC-INTERFAZ §Modal de modelos, DOMAIN §3). El modal no decide
// nada: entrega la elección y se cierra (DOMAIN §1 `modals`). Sin lista —«sin
// modelos» o «cargando…»— no hay nada que aplicar: solo se cierra.
func (a *App) aplicarModelo() tea.Cmd {
	nombre := a.Modelos.ModeloElegido()
	a.Modelos.Cerrar()
	if nombre == "" {
		return nil
	}
	a.Modelo = nombre
	a.fijarModeloEnPuerto()
	return nil
}

// elegirSesion cambia a la sesión seleccionada sin detener nada.
func (a *App) elegirSesion() tea.Cmd {
	ses, ok := a.Sesiones.SesionElegida()
	if !ok {
		a.Sesiones.Cerrar()
		return nil
	}
	copia := ses
	a.activar(&copia)
	a.Sesiones.Cerrar()
	// El chat cambia a esa sesión: su historial llega como dato y se pinta al
	// llegar, sin detener lo que siga corriendo en segundo plano (T-F005-05).
	return a.cmdHistorial(copia.ID)
}

// FijarMapa sustituye el mapa de teclas en caliente (reasignar un atajo no
// exige reiniciar, INTERFACES §4) y re-sincroniza el resolver y la lista que
// consumen el modal de atajos y pruebas. El mapa nuevo viene ya validado por quien lo
// construyó; si llegara inválido, se rechaza entero y el anterior sigue vivo.
// Una espera de líder a medias no sobrevive al cambio: sus secuencias
// candidatas eran del mapa anterior, así que el resolver nuevo arranca en NORMAL.
func (a *App) FijarMapa(mapa *Keymap) error {
	if err := ValidarAtajos(mapa.Entradas()); err != nil {
		return err
	}
	a.Mapa = mapa
	a.Atajos = mapa.Entradas()
	a.TeclaRes = NuevoKeyResolver(mapa)
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

// View pinta la pantalla: un modal abierto si lo hay, y si no la bienvenida o la
// interfaz principal. Los tres modales se dibujan centrados sobre la ventana
// (SPEC-INTERFAZ §Modales: "Tres modales centrados").
func (a *App) View() string {
	// La confirmación de borrado se pinta encima de todo: mientras está activa
	// es lo único que el usuario puede responder.
	if a.PidiendoEliminarSesion != "" {
		return a.viewConfirmarEliminar()
	}
	if a.Modelos.Abierto {
		return a.Modelos.Render(a.Ancho, a.Alto)
	}
	if a.Sesiones.Abierto {
		return a.Sesiones.Render(a.Ancho, a.Alto)
	}
	if a.AtajosModal.Abierto {
		return a.AtajosModal.Render(a.Ancho, a.Alto)
	}
	if a.Vista == VistaBienvenida {
		return a.viewBienvenida()
	}
	return a.viewPrincipal()
}

// viewConfirmarEliminar pinta la pregunta de borrado de una sesión que está
// trabajando: nombre, aviso y las dos respuestas posibles (SPEC-SESIONES).
func (a *App) viewConfirmarEliminar() string {
	nombre := a.PidiendoEliminarSesion
	for _, s := range a.Sesiones.Sesiones {
		if s.ID == a.PidiendoEliminarSesion {
			if s.Nombre != "" {
				nombre = s.Nombre
			}
			break
		}
	}
	texto := estiloTitulo.Render("ELIMINAR SESIÓN") + "\n\n" +
		estiloAviso.Render("la sesión «"+nombre+"» está trabajando.") + "\n\n" +
		"¿seguro que deseas eliminarla? (s/n)"
	return centrar(texto, a.Ancho, a.Alto)
}

// viewPrincipal pinta chat, razonamiento, respuesta en curso, aprobaciones y
// avisos; y el panel a la derecha cuando está abierto.
func (a *App) viewPrincipal() string {
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
	// El contador en vivo: mientras hay un turno en camino corre el tiempo, que
	// es lo que dice que el modelo sigue trabajando aunque aún no emita nada.
	if a.Chat.HayTurno() {
		partes = append(partes, estiloSistema.Render("generando… "+formatearDuracion(a.Chat.Transcurrido())))
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
	// La línea de entrada lleva su indicador de agente a la izquierda
	// (`[plan] > …`); el `Entrada` los compone los dos (T-F015-01).
	cuerpo += "\n\n" + a.Entrada.View()
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
