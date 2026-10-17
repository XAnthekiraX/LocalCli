package tui

import (
	"context"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"localcli/internal/tui/components/chat"
	"localcli/internal/tui/components/input"
	"localcli/internal/tui/components/modals"
	"localcli/internal/tui/components/panel"
	"localcli/internal/tui/components/welcome"
	"localcli/internal/tui/render"
)

// intervaloLatido es cada cuánto avanza el glifo del indicador en vivo mientras
// el turno está en marcha.
const intervaloLatido = 120 * time.Millisecond

// frameMsg avanza un frame del indicador en vivo.
type frameMsg struct{}

// Vista es dónde está la pantalla.
type Vista int

const (
	VistaBienvenida Vista = iota
	VistaPrincipal
)

// ModalVista es el modal que está abierto. El valor cero es «ninguno»: solo uno
// puede estar abierto a la vez y `Ctrl+C` sigue saliendo con cualquiera de ellos.
type ModalVista int

const (
	SinModal ModalVista = iota
	ModalModelos
	ModalDescargar
	ModalSesiones
	ModalAtajos
)

// Mensajes que devuelven las lecturas pedidas al abrir un modal y el borrado de
// una sesión. Ninguno sale del módulo: son la forma en que la respuesta de un
// comando vuelve al Update.
type (
	// modelosMsg trae la lista del modal de modelos. Un error no bloquea: el
	// modal avisa «sin modelos» y se cierra con `esc`.
	modelosMsg struct {
		Modelos []ModeloLocal
		Err     error
	}
	// sesionesMsg trae la lista del modal de sesiones. `VolverAlInicio` dice que
	// la lectura viene de haber borrado la sesión activa: si el proyecto se queda
	// sin sesiones, la vista vuelve a la bienvenida.
	sesionesMsg struct {
		Sesiones       []Sesion
		VolverAlInicio bool
		Err            error
	}
	// sesionEliminadaMsg confirma el borrado de una sesión y dispara el refresco
	// de la lista del modal.
	sesionEliminadaMsg struct {
		ID  string
		Err error
	}
	// historialMsg trae el hilo guardado de una sesión al retomarla —o al
	// deshacer o rehacer un turno—. `ID` ata la respuesta a su sesión: si
	// mientras tanto se cambió a otra, se descarta para no mezclar contextos
	// (SPEC-SESIONES). `Aviso`, si viene, se deja en el hilo tras pintarlo.
	historialMsg struct {
		ID     string
		Piezas []PiezaHistorial
		Aviso  string
		Err    error
	}
	// avisoTurnoMsg deja un aviso de un deshacer o rehacer que no cambió el hilo
	// (nada que deshacer, nada que rehacer, o un fallo): no recarga nada.
	avisoTurnoMsg struct {
		ID    string
		Texto string
	}
	// bifurcadaMsg trae la sesión nueva creada al bifurcar un globo, para que el
	// app la adopte. Un error deja el aviso sin cambiar de sesión.
	bifurcadaMsg struct {
		Sesion *Sesion
		Err    error
	}
)

// App es el modelo raíz de la TUI.
type App struct {
	puerto Puerto
	vista  Vista

	ancho, alto int
	bienvenida  welcome.Model
	agentes     []string
	agente      string
	modelo      string
	motor       string

	sesion *Sesion
	aviso  string

	// ultimaPeticion es el último texto enviado al agente desde esta vista; lo usa
	// `/retry` para repetirlo sin volver a teclearlo.
	ultimaPeticion string

	teclaRes *KeyResolver
	prefs    Preferencias

	entrada    input.Model
	origenCaja int

	// chat es el hilo de la sesión activa; frame es el frame del indicador en vivo
	// y animando dice si la cadena de latidos vigente sigue viva, para no arrancar
	// una segunda mientras la primera gira.
	chat     chat.Model
	frame    int
	animando bool

	paleta PaletaModel

	// notificacionBajoInput guarda el aviso que se muestra debajo de la caja de
	// entrada (no en el chat). Se limpia al enviar una nueva petición.
	notificacionBajoInput string

	// panel es el sidebar de datos de la sesión activa: solo se pinta en la vista
	// principal, y con los datos de la sesión activa, nunca los de otra.
	panel panel.Model
	// aprobaciones es el panel de decisiones pendientes de todas las sesiones:
	// las cuenta a todas, porque una sesión esperando no bloquea a las demás.
	// focoAprobaciones dice si el panel tiene el teclado (Ctrl+A lo alterna).
	aprobaciones     Aprobaciones
	focoAprobaciones bool

	// Las tres zonas de selección del ratón: la caja de entrada, el chat y el
	// sidebar. El arrastre queda acotado a la zona donde empezó y nunca mezcla
	// una con otra; se registran en cada repintado.
	// Modales: uno abierto a la vez. `modal` dice cuál y cada componente guarda
	// su lista, su resaltado y su aviso. Los abre y los cierra el `app`, que es
	// también quien aplica lo elegido: los modales no deciden nada.
	modelos  modals.Modelos
	descarga modals.Descarga
	sesiones modals.Sesiones
	atajos   modals.Atajos
	modal    ModalVista

	zonaEntrada      rect
	zonaChat         rect
	zonaPanel        rect
	zonaPensar       rect
	zonaAprobaciones rect
	zonaNotificacion rect

	sel        seleccion
	copiado    bool
	copiadoGen uint64

	// globo es el menú emergente del globo de un mensaje del usuario; clicGlobo
	// recuerda el globo pulsado mientras el botón sigue abajo, para distinguir un
	// clic simple (abre el menú) de un arrastre (selecciona).
	globo     MenuGlobo
	clicGlobo clicGlobo
	// scrollAntes recuerda si el chat seguía el final al empezar un arrastre, para
	// restaurarlo al soltar.
	scrollAntes bool

	// cancelarTurno corta el turno en curso; cancelarArmado marca que ya se avisó
	// una vez con «presiona esc para detener al agente».
	cancelarTurno  context.CancelFunc
	cancelarArmado bool

	eventos <-chan Evento
	cerrar  func()
}

// Nueva construye la TUI sin abrir nada: la bienvenida no espera a motor ni a base.
func Nueva(puerto Puerto) *App {
	app := &App{
		puerto:       puerto,
		agentes:      []string{AgentePlan, AgenteBuild},
		agente:       AgentePlan,
		bienvenida:   welcome.Nuevo(arteCanonico, arteCompacto),
		entrada:      input.Nuevo(AgentePlan, "", "", input.Capacidades{Razonamiento: input.Soportada}),
		teclaRes:     NuevoKeyResolver(KeymapPorDefecto()),
		paleta:       NuevoPaleta(),
		panel:        panel.Nuevo(),
		aprobaciones: NuevaAprobaciones(),
		globo:        NuevoMenuGlobo(),
	}
	app.paleta.EstablecerCatalogo(ComandosDeLaEntrada(puerto))
	if puerto == nil {
		return app
	}
	// El pie del panel se lee una sola vez: la carpeta del proyecto y el estado de
	// git en el momento del arranque no vuelven a consultarse (INTERFACES §3.1).
	app.panel.Ruta = puerto.Carpeta()
	if git := puerto.Git(); git.Rama != "" || git.Cambios > 0 {
		app.panel.GitRama, app.panel.GitCambios = git.Rama, git.Cambios
	}
	if agentes := puerto.Agentes(); len(agentes) > 0 {
		app.agentes = agentes
	}
	if recordado := puerto.AgenteRecordado(); app.tieneAgente(recordado) {
		app.agente = recordado
	}
	app.modelo = puerto.ModeloActual()
	app.motor = puerto.MotorActual()
	app.entrada.Agente = app.agente
	app.entrada.Modelo = app.modelo
	app.entrada.Motor = app.motor
	app.entrada.Capacidades = capacidadesDeVista(puerto.Capacidades())
	return app
}

// capacidadesDeVista traduce las capacidades del puerto al dato que pinta la caja
// de entrada. La visión es la única que el motor declara hoy; la de herramientas
// queda desconocida y el razonamiento se conserva soportado para no perder su
// interruptor.
func capacidadesDeVista(c Capacidades) input.Capacidades {
	return input.Capacidades{
		Herramientas: input.Desconocida,
		Vision:       capacidadDeVista(c.Vision),
		Razonamiento: input.Soportada,
	}
}

// capacidadDeVista traduce el estado de una capacidad de la frontera al que pinta
// la caja de entrada.
func capacidadDeVista(c Capacidad) input.Capacidad {
	switch c {
	case CapacidadSoportada:
		return input.Soportada
	case CapacidadNoSoportada:
		return input.NoSoportada
	default:
		return input.Desconocida
	}
}

// capacidadDeTexto lee el estado que trae el payload del evento de capacidades.
func capacidadDeTexto(s string) Capacidad {
	switch s {
	case "soportada":
		return CapacidadSoportada
	case "no_soportada":
		return CapacidadNoSoportada
	default:
		return CapacidadDesconocida
	}
}

func (a *App) Init() tea.Cmd {
	a.suscribir()
	return escuchar(a.eventos)
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case tea.WindowSizeMsg:
		a.ancho, a.alto = m.Width, m.Height
		a.paleta.EstablecerMaxAlto(a.alto)
		if a.sel.contenido {
			a.chat.ReanudarSeguimiento(a.scrollAntes)
		}
		a.sel.limpiar()
	case tea.KeyMsg:
		return a.tecla(m)
	case tea.MouseMsg:
		return a.raton(m)
	case input.EnviadoMsg:
		return a, a.enviarEntrada(m.Texto)
	case modelosMsg:
		a.modelos.Fijar(modelosDeLocal(m.Modelos), m.Err)
		return a, nil
	case sesionesMsg:
		a.sesiones.Fijar(sesionesDeLocal(m.Sesiones), m.Err)
		if m.VolverAlInicio && len(m.Sesiones) == 0 {
			// El proyecto se quedó sin sesiones: la vista vuelve a la bienvenida.
			a.cerrarModales()
			a.vista = VistaBienvenida
		}
		return a, nil
	case sesionEliminadaMsg:
		return a, a.trasEliminarSesion(m)
	case historialMsg:
		if m.Err != nil {
			if a.sesion != nil && a.sesion.ID == m.ID {
				a.aviso = "no se pudo leer el historial: " + m.Err.Error()
			}
			return a, nil
		}
		// Solo se pinta si sigue siendo la sesión activa: un cambio de sesión
		// mientras la lectura viajaba no debe volcar el hilo de otra.
		if a.sesion != nil && a.sesion.ID == m.ID {
			a.pintarHistorial(m.Piezas)
			// El aviso va después de pintar: el hilo se reemplaza entero, así
			// que uno puesto antes se perdería con la recarga.
			if m.Aviso != "" {
				a.chat.AnotarAviso(m.Aviso)
			}
			if a.puerto != nil {
				return a, tea.Batch(escuchar(a.eventos), a.puerto.SincronizarContexto(m.ID))
			}
		}
		return a, nil
	case avisoTurnoMsg:
		if a.sesion != nil && a.sesion.ID == m.ID {
			a.chat.AnotarAviso(m.Texto)
		}
		return a, nil
	case bifurcadaMsg:
		if m.Err != nil {
			a.aviso = "no se pudo bifurcar: " + m.Err.Error()
			return a, nil
		}
		if m.Sesion == nil {
			return a, nil
		}
		return a, a.adoptarSesion(modals.Sesion{ID: m.Sesion.ID, Nombre: m.Sesion.Nombre, Estado: m.Sesion.Estado})
	case frameMsg:
		return a, a.latido()
	case eventoMsg:
		return a, tea.Batch(a.aplicarEvento(m.evento), escuchar(a.eventos))
	case LiderExpiradoMsg:
		a.teclaRes.Cancelar()
	case copiadoExpiradoMsg:
		if m.gen == a.copiadoGen {
			a.copiado = false
		}
	case errorMsg:
		a.aviso = m.err.Error()
	}
	return a, nil
}

func (a *App) View() string {
	// Con un modal abierto, la pantalla es el modal: lo de abajo ni se pinta ni
	// recibe teclas (SPEC-INTERFAZ §Modales).
	if a.modalAbierto() {
		return a.vistaModal()
	}
	frame := a.conSeleccion(a.marco())
	// Overlay de la paleta de comandos (si está visible)
	if a.paleta.Visible() {
		frame = a.overlayPaleta(frame)
	}
	// Overlay del menú del globo de un mensaje del usuario
	if a.globo.Visible() {
		fila, col, _, _ := a.rectGlobo()
		frame = render.Superponer(frame, a.globo.View(), fila, col)
	}
	switch {
	case a.copiado:
		return conIndicadorSuperior(frame, render.EstiloBlanco.Render("[Copiado]"))
	case a.teclaRes.EsperandoLider():
		return conIndicadorSuperior(frame, render.EstiloSutil.Render("lider »"))
	}
	return frame
}

// overlayPaleta dibuja la paleta de comandos como overlay superpuesta sobre el
// frame base, alineada con la caja de entrada (misma columna y ancho).
// Reemplaza columnas de las filas existentes sin insertar líneas (verdadero
// overlay: no empuja el contenido de debajo).
func (a *App) overlayPaleta(frame string) string {
	fila, col, ancho, alto := a.rectPaleta()
	if alto <= 0 {
		return frame
	}
	return render.Superponer(frame, a.paleta.View(ancho), fila, col)
}

// rectPaleta resuelve la geometría en pantalla de la paleta: la fila superior
// (justo encima de la caja de entrada), la columna, el ancho de la caja y el alto
// real del bloque ya pintado. Con alto 0 la paleta no debe pintarse.
func (a *App) rectPaleta() (fila, col, ancho, alto int) {
	if !a.paleta.Visible() {
		return 0, 0, 0, 0
	}
	var anchoCaja, columnaCaja, filaCaja int
	if a.vista == VistaBienvenida {
		_, limites := a.bienvenida.VistaLimites(a.ancho, a.alto, welcome.Datos{
			Agente: a.agente,
			Modelo: a.modelo,
			Motor:  a.motor,
			Aviso:  a.aviso,
			Pensar: a.entrada.Pensar,
		})
		anchoCaja = limites.Ancho
		columnaCaja = limites.Columna
		filaCaja = limites.Fila
	} else {
		caja := a.entrada.RectCaja()
		anchoCaja = a.anchoColumna()
		columnaCaja = caja.Columna
		filaCaja = a.origenCaja + caja.Fila
	}
	if anchoCaja <= 0 || filaCaja <= 0 {
		return 0, 0, 0, 0
	}
	palStr := a.paleta.View(anchoCaja)
	if palStr == "" {
		return 0, 0, 0, 0
	}
	palAlto := altoDe(palStr)
	inicio := filaCaja - palAlto
	if inicio < 0 {
		inicio = 0
	}
	return inicio, columnaCaja, anchoCaja, palAlto
}

// vistaModal pinta el modal abierto centrado en la ventana, sobre el fondo de
// la pantalla. Mientras un modal está abierto, `lider »` sigue avisando de la
// espera de la secuencia.
func (a *App) vistaModal() string {
	var frame string
	switch a.modal {
	case ModalModelos:
		frame = a.modelos.Render(a.ancho, a.alto)
	case ModalDescargar:
		frame = a.descarga.Render(a.ancho, a.alto)
	case ModalSesiones:
		frame = a.sesiones.Render(a.ancho, a.alto)
	case ModalAtajos:
		frame = a.atajos.Render(a.ancho, a.alto)
	}
	if a.teclaRes.EsperandoLider() {
		frame = conIndicadorSuperior(frame, render.EstiloSutil.Render("lider »"))
	}
	return render.MarcoCompleto(frame, a.ancho, a.alto)
}

// marco pinta la vista vigente y registra en cada repintado los límites en pantalla
// de sus zonas de selección: la caja de entrada, el chat y el sidebar.
func (a *App) marco() string {
	if a.vista == VistaBienvenida {
		frame, limites := a.vistaBienvenida()
		a.zonaEntrada = rect{Fila: limites.Fila, Columna: limites.Columna, Ancho: limites.Ancho, Alto: limites.Alto}
		a.zonaPensar = rect{Fila: limites.Pensar.Fila, Columna: limites.Pensar.Columna, Ancho: limites.Pensar.Ancho, Alto: limites.Pensar.Alto}
		// La bienvenida es una sola columna con su caja de entrada: no hay chat ni
		// sidebar que acotar.
		a.zonaChat, a.zonaPanel = rect{}, rect{}
		return render.MarcoCompleto(frame, a.ancho, a.alto)
	}
	frame, arriba := a.vistaPrincipal()
	caja := a.entrada.RectCaja()
	a.origenCaja = arriba
	a.zonaEntrada = rect{Fila: arriba + caja.Fila, Columna: caja.Columna, Ancho: caja.Ancho, Alto: caja.Alto}
	a.zonaPensar = rect{}
	return render.MarcoCompleto(frame, a.ancho, a.alto)
}

// vistaPrincipal pinta el chat de la sesión activa con la caja de entrada pegada
// al pie de la columna y, a su derecha, el sidebar de datos cuando está abierto.
// El chat ocupa el alto que queda hasta la caja; con un historial corto se
// rellena de celdas del fondo de la pantalla, de modo que la caja quede al pie en
// vez de flotar en medio. Devuelve el frame y la fila donde empieza la caja.
func (a *App) vistaPrincipal() (string, int) {
	ancho := a.anchoColumna()
	a.sincronizarPanel()
	a.entrada.Agente = a.agente
	a.entrada.Modelo = a.modelo
	a.entrada.Motor = a.motor
	lineasCaja := strings.Split(a.entrada.View(ancho), "\n")
	if a.cancelarArmado {
		lineasCaja = append(lineasCaja, render.EstiloSutil.Render("  presiona esc para detener al agente"))
	}

	// Notificación debajo de la caja: se envuelve al ancho disponible.
	var lineasNotificacion []string
	if a.notificacionBajoInput != "" {
		envueltas := render.EnvolverTexto(a.notificacionBajoInput, ancho-2)
		for _, l := range envueltas {
			lineasNotificacion = append(lineasNotificacion, render.PintarFondo(" "+render.EstiloAviso.Render(l), ancho, render.FondoPantalla))
		}
	}

	lineasActividad := a.lineasActividad(ancho)
	lineasAprobacion := a.lineasAprobaciones(ancho)
	arriba := max(a.alto-len(lineasCaja)-len(lineasNotificacion)-len(lineasActividad)-len(lineasAprobacion), 0)
	filas := make([]string, 0, a.alto)
	if arriba > 0 && !a.chat.Vacio() {
		if hilo := a.chat.Vista(ancho, arriba); hilo != "" {
			filas = append(filas, strings.Split(hilo, "\n")...)
		}
	}
	vacio := render.PintarFondo("", ancho, render.FondoPantalla)
	for len(filas) < arriba {
		filas = append(filas, vacio)
	}
	filas = append(filas, lineasActividad...)
	filas = append(filas, lineasAprobacion...)
	filas = append(filas, lineasCaja...)
	filas = append(filas, lineasNotificacion...)
	cuerpo := strings.Join(filas, "\n")
	inicioCaja := arriba + len(lineasActividad) + len(lineasAprobacion)
	// Zonas del ratón: el chat es la columna principal hasta donde empieza la caja;
	// la fila fija de actividad y el panel de aprobaciones quedan fuera de su zona.
	a.zonaChat = rect{Fila: 0, Columna: 0, Ancho: ancho, Alto: arriba}
	a.zonaAprobaciones = rect{}
	if len(lineasAprobacion) > 0 {
		a.zonaAprobaciones = rect{
			Fila:    arriba + len(lineasActividad),
			Columna: 0, Ancho: ancho, Alto: len(lineasAprobacion),
		}
	}
	// Zona de notificación (para posible interacción futura).
	a.zonaNotificacion = rect{}
	if len(lineasNotificacion) > 0 {
		a.zonaNotificacion = rect{
			Fila:    arriba + len(lineasActividad) + len(lineasAprobacion) + len(lineasCaja),
			Columna: 0, Ancho: ancho, Alto: len(lineasNotificacion),
		}
	}
	a.zonaPanel = rect{}
	if !a.panel.Abierto() {
		return cuerpo, inicioCaja
	}
	anchoPanel := a.anchoPanel()
	a.zonaPanel = rect{Fila: 0, Columna: ancho + 2, Ancho: anchoPanel, Alto: a.alto}
	return a.componerColumnas(cuerpo, ancho, anchoPanel, a.panel.Render(anchoPanel, a.alto)), inicioCaja
}

// lineasActividad es el indicador en vivo como fila fija encima de la caja: no
// pertenece al chat, así que el scroll del historial no lo desplaza.
func (a *App) lineasActividad(ancho int) []string {
	actividad := a.chat.Actividad(a.frame)
	if actividad == "" {
		return nil
	}
	return []string{render.PintarFondo(" "+actividad, ancho, render.FondoPantalla)}
}

// lineasAprobaciones pinta el panel de decisiones encima de la caja de entrada
// cuando hay algo que decidir; cada línea va ajustada al ancho de la columna.
// Sin nada pendiente devuelve nil: el panel no reserva hueco.
func (a *App) lineasAprobaciones(ancho int) []string {
	texto := a.aprobaciones.Render(ancho)
	if texto == "" {
		return nil
	}
	crudas := strings.Split(texto, "\n")
	lineas := make([]string, len(crudas))
	for i, l := range crudas {
		lineas[i] = render.PintarFondo(l, ancho, render.FondoPantalla)
	}
	return lineas
}

// sincronizarPanel deja el panel con los datos de la sesión activa. El nombre de
// la conversación y su estado se leen al pintar —el título lo cambia el modelo
// por evento, y cambiar de sesión no interrumpe nada—, así que el panel refleja
// siempre la sesión activa y nunca los datos de otra.
func (a *App) sincronizarPanel() {
	if a.sesion != nil {
		a.panel.Sesion = a.sesion.Nombre
		a.panel.Estado = a.sesion.Estado
	}
	a.panel.Agente = a.agente
}

// modalAbierto dice si hay un modal en pantalla. Los tres comparten la misma
// mecánica y solo uno puede estar abierto a la vez (SPEC-INTERFAZ §Modales).
func (a *App) modalAbierto() bool { return a.modal != SinModal }

// contextoModal es el contexto de resolución del modal abierto: el de sesiones
// añade `session_delete` (`ctrl+d`), que en la vista pliega el panel.
func (a *App) contextoModal() Contexto {
	if a.modal == ModalSesiones {
		return ContextoModalSesiones
	}
	return ContextoModal
}

// soloModal abre el modal pedido y cierra cualquier otro: abrir un modal es
// siempre abrirlo en lugar del que hubiera, y su confirmación a medias se
// descarta. La paleta se retira: con un modal abierto no hay entrada que
// autocompletar.
func (a *App) soloModal(cual ModalVista) {
	a.cerrarModales()
	a.modal = cual
	a.paleta.Ocultar()
}

// cerrarModales cierra los tres. Es lo que hace `esc` —descartar no cambia
// nada— y lo que hace cualquier apertura antes de abrir el suyo.
func (a *App) cerrarModales() {
	a.modelos.Cerrar()
	a.descarga.Cerrar()
	a.sesiones.Cerrar()
	a.atajos.Cerrar()
	a.modal = SinModal
}

// cerrarModal descarta lo abierto. Si el modal de sesiones está esperando la
// confirmación de un borrado, el primer `esc` cancela la pregunta en vez de
// cerrar el modal: descartar la confirmación no cambia nada.
func (a *App) cerrarModal() {
	if a.sesiones.Confirmando() {
		a.sesiones.CancelarConfirmacion()
		return
	}
	a.cerrarModales()
}

// moverModal mueve el resaltado del modal abierto. El de atajos es de solo
// lectura y no tiene resaltado: no hace nada.
func (a *App) moverModal(delta int) {
	switch a.modal {
	case ModalModelos:
		a.modelos.Mover(delta)
	case ModalSesiones:
		a.sesiones.Mover(delta)
	}
}

// aplicarModal resuelve `enter` en el modal abierto: modelos y sesiones
// entregan su elección al `app`, que es quien la aplica; los atajos no aplican
// nada.
func (a *App) aplicarModal() tea.Cmd {
	switch a.modal {
	case ModalModelos:
		modelo, ok := a.modelos.Elegido()
		a.cerrarModales()
		if !ok {
			return nil
		}
		return a.aplicarModelo(modelo)
	case ModalSesiones:
		sesion, ok := a.sesiones.SesionElegida()
		a.cerrarModales()
		if !ok {
			return nil
		}
		return a.adoptarSesion(sesion)
	}
	return nil
}

// abrirModalModelos abre el modal y pide la lista al motor en el acto: se pide al
// abrir, no al arrancar, así que arrancar no toca el motor. El modal aparece ya,
// con «cargando…», y se rellena cuando llega la respuesta.
func (a *App) abrirModalModelos() tea.Cmd {
	a.soloModal(ModalModelos)
	a.modelos.Abrir(a.modelo)
	return pedirModelos(a.puerto)
}

// pedirModelos es la lectura del modal: un fallo no bloquea ni propaga, el modal
// queda con el aviso «sin modelos». Sin puerto —una TUI suelta en una prueba— no
// hay nada que listar y tampoco es un error.
func pedirModelos(p Puerto) tea.Cmd {
	return func() tea.Msg {
		if p == nil {
			return modelosMsg{}
		}
		lista, err := p.Modelos()
		return modelosMsg{Modelos: lista, Err: err}
	}
}

// abrirModalDescargar abre el modal de descarga en blanco. No pide nada a nadie:
// el usuario teclea la URL y la descarga se lanza con `enter`.
func (a *App) abrirModalDescargar() tea.Cmd {
	a.soloModal(ModalDescargar)
	a.descarga.Abrir()
	return nil
}

// iniciarDescarga lanza la descarga de la URL escrita. El modal pasa a «bajando»
// y el avance llega por los eventos de descarga; una URL vacía o un puerto
// ausente dejan el aviso en el propio modal.
func (a *App) iniciarDescarga() tea.Cmd {
	if !a.descarga.Empezar() {
		return nil
	}
	if a.puerto == nil {
		a.descarga.Fallida("no hay conexión con el motor")
		return nil
	}
	if err := a.puerto.DescargarModelo(a.descarga.URL, a.descarga.MMProjURL); err != nil {
		a.descarga.Fallida(err.Error())
		return nil
	}
	return nil
}

// abrirModalSesiones abre el modal y pide las sesiones del proyecto en el acto:
// no hay ninguna lista permanente que guardar, lo que se ve es lo que hay.
func (a *App) abrirModalSesiones() tea.Cmd {
	a.soloModal(ModalSesiones)
	a.sesiones.Abrir(a.sesionID())
	return pedirSesiones(a.puerto, false)
}

// pedirSesiones es la lectura del modal de sesiones. `volverAlInicio` la marca
// cuando viene de haber borrado la sesión activa: con el proyecto sin sesiones,
// la vista vuelve a la bienvenida.
func pedirSesiones(p Puerto, volverAlInicio bool) tea.Cmd {
	return func() tea.Msg {
		if p == nil {
			return sesionesMsg{VolverAlInicio: volverAlInicio}
		}
		lista, err := p.Listar()
		return sesionesMsg{Sesiones: lista, VolverAlInicio: volverAlInicio, Err: err}
	}
}

// alternarModalAtajos abre la tabla del mapa vigente. `Ctrl+P` con el modal
// abierto lo cierra: es el mismo atajo, así que alterna. No pide nada a nadie:
// el listado sale del mapa que ya está en memoria, y se recompone en cada
// apertura, de modo que una reasignación se ve sin reiniciar.
func (a *App) alternarModalAtajos() tea.Cmd {
	if a.modal == ModalAtajos {
		a.cerrarModales()
		return nil
	}
	a.soloModal(ModalAtajos)
	a.atajos.Abrir(FilasDeAtajos(a.teclaRes.Kmap.Entradas(), a.teclaRes.Kmap.Lider()))
	return nil
}

// aplicarModelo deja el modelo elegido como el de la sesión: se refleja en la
// línea de estado de la caja y se pide al motor que trabaje con él. El modelo
// activo lo persiste el catálogo (`models.json`), no las preferencias. Un
// modelo que no declara herramientas se avisa en el hilo sin bloquear nada: el
// usuario decide si cambia de modelo.
func (a *App) aplicarModelo(m modals.Modelo) tea.Cmd {
	a.modelo = m.Nombre
	a.motor = m.Motor
	a.entrada.Modelo = m.Nombre
	a.entrada.Motor = m.Motor
	// Las capacidades del modelo nuevo se desconocen hasta que el motor las
	// resuelva; el evento `capacidades_modelo` las traerá.
	a.entrada.Capacidades.Vision = input.Desconocida
	if a.puerto != nil {
		a.puerto.FijarModelo(m.Nombre)
	}
	if m.SinHerramientas {
		a.chat.AnotarAviso("aviso: " + m.Nombre + " no declara capacidad de herramientas")
	}
	return nil
}

// adoptarSesion pasa la vista a la sesión elegida en el modal. Solo cambia lo que
// se pinta: no interrumpe ninguna ejecución (DOMAIN §2). El hilo se vacía y se
// rellena con el historial guardado de esa sesión —su lectura es una lectura de
// `store` (INTERFACES §3)—, que llega por `historialMsg`; una sesión que ya estaba
// trabajando reanuda su contador y su glifo.
func (a *App) adoptarSesion(s modals.Sesion) tea.Cmd {
	a.sesion = &Sesion{ID: s.ID, Nombre: s.Nombre, Estado: s.Estado}
	a.chat.Vaciar()
	a.reiniciarPanel()
	a.vista = VistaPrincipal
	a.notificacionBajoInput = ""
	cargar := cargarHistorial(a.puerto, s.ID)
	syncCtx := func() tea.Msg {
		if a.puerto == nil {
			return nil
		}
		c := a.puerto.SincronizarContexto(s.ID)
		if c == nil {
			return nil
		}
		return c()
	}
	switch s.Estado {
	case EstadoTrabajando:
		a.chat.ReanudarTurno()
		return tea.Batch(cargar, syncCtx, a.arrancarLatido())
	case EstadoEsperandoPermiso:
		return tea.Batch(cargar, syncCtx, a.arrancarLatido())
	}
	return tea.Batch(cargar, syncCtx)
}

// nuevaSesion crea una sesión y la deja activa sin detener las demás
// (SPEC-SESIONES: `Ctrl+X n` desde la vista principal). Nace con el nombre
// provisional que da el puerto y sin historial que traer: recién creada, no
// tiene ninguno.
func (a *App) nuevaSesion() tea.Cmd {
	if a.puerto == nil {
		return nil
	}
	sesion, err := a.puerto.Crear()
	if err != nil {
		a.aviso = "no se pudo crear la sesión: " + err.Error()
		return nil
	}
	a.sesion = sesion
	a.chat.Vaciar()
	a.reiniciarPanel()
	a.vista = VistaPrincipal
	a.notificacionBajoInput = ""
	return nil
}

// cargarHistorial es la lectura del historial de una sesión: un fallo no tumba la
// vista, se avisa y el hilo queda vacío. Sin puerto —una TUI suelta en una prueba—
// no hay historial que traer.
func cargarHistorial(p Puerto, sesionID string) tea.Cmd {
	return func() tea.Msg {
		if p == nil {
			return historialMsg{ID: sesionID}
		}
		piezas, err := p.Historial(sesionID)
		return historialMsg{ID: sesionID, Piezas: piezas, Err: err}
	}
}

// pintarHistorial vuelca el hilo guardado en la sesión activa, en su orden:
// usuario y agente con su razonamiento y su duración, y las líneas del sistema
// —herramientas y sub-procesos— que se intercalaron entre ellos. Es lo que hace
// visible la conversación al retomar la sesión.
func (a *App) pintarHistorial(piezas []PiezaHistorial) {
	hilo := make([]chat.Pieza, 0, len(piezas))
	for _, p := range piezas {
		switch m := p.(type) {
		case *MensajeHistorial:
			hablante := chat.Usuario
			if m.Rol == "agent" {
				hablante = chat.Agente
			}
			hilo = append(hilo, &chat.Segmento{
				ID:              m.ID,
				Hablante:        hablante,
				Texto:           m.Texto,
				Razonamiento:    m.Razonamiento,
				SinRazonamiento: hablante == chat.Agente && m.Razonamiento == "",
				Duracion:        m.Duracion,
			})
		case *LineaHistorial:
			// La respuesta de una etapa con `respuesta_en_chat` se guardó como
			// pieza de pantalla, pero se pinta como bloque del agente —igual que
			// en vivo— y no como línea suelta del sistema
			// (SPEC-ORQUESTADOR-FLUJOS §Visibilidad de una respuesta).
			if m.Tipo == "respuesta" {
				hilo = append(hilo, &chat.Segmento{
					Hablante:        chat.Agente,
					Texto:           m.Texto,
					SinRazonamiento: true,
				})
				continue
			}
			hilo = append(hilo, &chat.Linea{Tipo: tipoDeLinea(m.Tipo), Texto: m.Texto, Bloque: m.Bloque})
		}
	}
	a.chat.ReemplazarHistorial(hilo)
}

// tipoDeLinea traduce el tipo guardado en `chat_evento` al tipo de línea de la
// vista. Un tipo desconocido cae en aviso, que se pinta suelto y sin adorno.
func tipoDeLinea(tipo string) chat.TipoLinea {
	switch tipo {
	case "herramienta":
		return chat.LineaHerramienta
	case "proceso":
		return chat.LineaSubProceso
	default:
		return chat.LineaAviso
	}
}

// eliminarResaltada resuelve `ctrl+d` en el modal de sesiones: la primera
// pulsación pide confirmación si la sesión trabaja —y no entrega nada—, y la
// segunda entrega la sesión para pedir su borrado al puerto junto con el
// refresco de la lista.
func (a *App) eliminarResaltada() tea.Cmd {
	if a.modal != ModalSesiones {
		return nil
	}
	sesion, adelante := a.sesiones.PedirEliminar()
	if !adelante {
		return nil
	}
	return eliminarSesionCmd(a.puerto, sesion.ID)
}

func eliminarSesionCmd(p Puerto, sesionID string) tea.Cmd {
	return func() tea.Msg {
		if p == nil {
			return sesionEliminadaMsg{ID: sesionID}
		}
		return sesionEliminadaMsg{ID: sesionID, Err: p.Eliminar(sesionID)}
	}
}

// trasEliminarSesion refresca la lista con la sesión ya borrada. Si la borrada
// era la activa, la vista se queda sin sesión —el hilo y los datos del panel eran
// suyos— y, si el proyecto se queda sin ninguna, la lectura siguiente devuelve la
// vista a la bienvenida (SPEC-INTERFAZ §Cambiar de sesión).
func (a *App) trasEliminarSesion(m sesionEliminadaMsg) tea.Cmd {
	if m.Err != nil {
		a.aviso = "no se pudo eliminar la sesión: " + m.Err.Error()
		return nil
	}
	eraActiva := a.sesion != nil && a.sesion.ID == m.ID
	if eraActiva {
		a.sesion = nil
		a.chat.Vaciar()
		a.reiniciarPanel()
	}
	return pedirSesiones(a.puerto, eraActiva)
}

// reiniciarPanel olvida los datos de la sesión que se deja: el panel refleja la
// sesión activa, y los suyos llegarán con los eventos de la que se abra.
func (a *App) reiniciarPanel() {
	a.panel.Sesion = ""
	a.panel.Estado = ""
	a.panel.ContextoTokens = 0
	a.panel.LimiteTokens = 0
	a.panel.LimiteDelHarness = false
	a.panel.TokensEstimados = false
	a.panel.Tareas = nil
}

// sesionID es el identificador de la sesión activa (vacío si no la hay).
func (a *App) sesionID() string {
	if a.sesion == nil {
		return ""
	}
	return a.sesion.ID
}

// modelosDeLocal y sesionesDeLocal traducen lo que devuelve el puerto al dato que
// pintan los modales: la vista no le pasa al componente el tipo de su frontera.
func modelosDeLocal(lista []ModeloLocal) []modals.Modelo {
	if len(lista) == 0 {
		return nil
	}
	modelos := make([]modals.Modelo, len(lista))
	for i, m := range lista {
		modelos[i] = modals.Modelo{Nombre: m.Nombre, Motor: m.Motor, SinHerramientas: m.SinHerramientas}
	}
	return modelos
}

func sesionesDeLocal(lista []Sesion) []modals.Sesion {
	if len(lista) == 0 {
		return nil
	}
	sesiones := make([]modals.Sesion, len(lista))
	for i, s := range lista {
		sesiones[i] = modals.Sesion{ID: s.ID, Nombre: s.Nombre, Estado: s.Estado}
	}
	return sesiones
}

// componerColumnas une la columna principal con el sidebar dejando una banda de
// fondo de dos columnas como separación, sin el glifo `│`. Rellena cada columna a
// su ancho y con su fondo, de modo que la separación quede recta y el marco siga
// siendo UNA sola cadena. La banda no entra en ninguna selección: chat y sidebar
// son zonas distintas y el arrastre no cruza de una a otra (INTERFACES §Ratón).
func (a *App) componerColumnas(izquierda string, anchoIzq, anchoDer int, derecha string) string {
	alto := a.alto
	if alto <= 0 {
		// Sin geometría conocida no se recorta el sidebar: se toma el alto del
		// bloque más largo para no perder contenido.
		alto = altoDe(izquierda)
		if d := altoDe(derecha); d > alto {
			alto = d
		}
	}
	izq := strings.Split(izquierda, "\n")
	der := strings.Split(derecha, "\n")
	banda := render.PintarFondo("", 2, render.FondoPantalla)
	lineas := make([]string, alto)
	for i := 0; i < alto; i++ {
		l, r := "", ""
		if i < len(izq) {
			l = izq[i]
		}
		if i < len(der) {
			r = der[i]
		}
		lineas[i] = render.PintarFondo(render.RecortarColumnas(l, anchoIzq), anchoIzq, render.FondoPantalla) + banda +
			render.PintarFondo(render.RecortarColumnas(r, anchoDer), anchoDer, render.FondoPanel)
	}
	return strings.Join(lineas, "\n")
}

// anchoPanel es el ancho del sidebar cuando está abierto, recortado si la
// terminal no da para el ancho fijo más la columna principal mínima.
func (a *App) anchoPanel() int {
	if a.ancho <= 0 {
		return panel.Ancho
	}
	disponible := a.ancho - panel.AnchoMinimoChat - 2
	if disponible < 1 {
		disponible = 1
	}
	return min(panel.Ancho, disponible)
}

// anchoColumna es el ancho disponible para la columna principal (el chat y su
// caja de entrada): el total menos el sidebar y las dos columnas de separación
// cuando el panel está abierto. El chat nunca baja de 20 columnas: si la terminal
// es estrecha, quien cede es el sidebar.
func (a *App) anchoColumna() int {
	if a.ancho <= 0 {
		return a.ancho
	}
	if !a.panel.Abierto() {
		return a.ancho
	}
	return max(a.ancho-a.anchoPanel()-2, panel.AnchoMinimoChat)
}

// altoDe cuenta las líneas que ocupa un bloque ya compuesto; los saltos finales
// no cuentan, porque el cursor se queda en la última línea con contenido.
func altoDe(texto string) int {
	texto = strings.TrimRight(texto, "\n")
	if texto == "" {
		return 0
	}
	return strings.Count(texto, "\n") + 1
}

// latido avanza el glifo del indicador en vivo y vuelve a agendar el siguiente
// mientras haya un turno en marcha: sin turno, la cadena se corta sola y el
// indicador se detiene.
func (a *App) latido() tea.Cmd {
	if !a.chat.HayTurno() {
		a.animando = false
		return nil
	}
	a.frame++
	return tea.Tick(intervaloLatido, func(time.Time) tea.Msg { return frameMsg{} })
}

// arrancarLatido enciende el latido del indicador si no estaba ya girando.
func (a *App) arrancarLatido() tea.Cmd {
	if a.animando {
		return nil
	}
	a.animando = true
	return a.latido()
}

// conSeleccion resalta en video inverso lo que abarca el arrastre.
func (a *App) conSeleccion(frame string) string {
	if !a.sel.arrastrando {
		return frame
	}
	lineas := strings.Split(frame, "\n")
	for _, t := range a.tramosSeleccion() {
		if t.Fila >= 0 && t.Fila < len(lineas) {
			lineas[t.Fila] = render.ResaltarTramo(lineas[t.Fila], t.Desde, t.Hasta)
		}
	}
	return strings.Join(lineas, "\n")
}

// tramosSeleccion resuelve el realce del arrastre vigente: contra las líneas del
// hilo (contenido) si nació en el chat, contra el texto anclado del marco si nació
// en la caja o el sidebar.
func (a *App) tramosSeleccion() []tramo {
	if a.sel.contenido {
		return a.sel.resolverContenido(a.chat.LineasContenido(), a.chat.VentanaDeContenido)
	}
	return a.sel.resolver(strings.Split(render.SinANSI(a.marco()), "\n"))
}

// vistaBienvenida pinta la bienvenida y devuelve el rectángulo de su caja de entrada.
func (a *App) vistaBienvenida() (string, welcome.Limites) {
	frame, limites := a.bienvenida.VistaLimites(a.ancho, a.alto, welcome.Datos{
		Agente: a.agente,
		Modelo: a.modelo,
		Motor:  a.motor,
		Aviso:  a.aviso,
		Pensar: a.entrada.Pensar,
	})
	// La paleta se renderiza como overlay en View(), no aquí.
	return frame, limites
}

// enviarEntrada manda lo escrito desde la vista principal a la sesión activa y
// vacía la caja; sin sesión no hay a quién mandarlo.
func (a *App) enviarEntrada(texto string) tea.Cmd {
	a.entrada.Limpiar()
	a.paleta.Ocultar()
	a.notificacionBajoInput = ""
	if cmd, consumido := a.ejecutarComandoLocal(texto); consumido {
		return cmd
	}
	if a.puerto == nil || a.sesion == nil {
		return nil
	}
	a.ultimaPeticion = texto
	a.chat.AñadirUsuario(texto)
	return tea.Batch(a.lanzarTurno(a.sesion.ID, a.agente, texto), a.arrancarLatido())
}

// enviarActual envía desde la vista vigente: la bienvenida crea la sesión con su
// primera petición; la principal manda lo escrito a la sesión activa.
func (a *App) enviarActual() tea.Cmd {
	if a.vista == VistaBienvenida {
		return a.enviar()
	}
	if texto := a.entrada.Peticion(); texto != "" {
		return a.enviarEntrada(texto)
	}
	return nil
}

// sobrePensar dice si una celda de la pantalla cae sobre el interruptor de
// razonamiento del pie de la caja de entrada.
func (a *App) sobrePensar(p punto) bool {
	return a.entrada.SobrePensar(p.Fila-a.origenCaja, p.Columna)
}

// sobreInterruptorPensar dice si la celda cae sobre el chip de razonamiento, en
// la bienvenida o en el pie de la caja de la vista principal.
func (a *App) sobreInterruptorPensar(p punto) bool {
	if a.vista == VistaBienvenida {
		return a.zonaPensar.contiene(p)
	}
	return a.sobrePensar(p)
}

// detenerTurno para el agente en dos pasos: la primera pulsación avisa debajo de
// la caja, la segunda cancela de verdad.
func (a *App) detenerTurno() {
	if a.cancelarTurno == nil || !a.chat.HayTurno() {
		return
	}
	if !a.cancelarArmado {
		a.cancelarArmado = true
		return
	}
	a.cancelarEjecucion()
}

// cancelarEjecucion corta el turno en curso y olvida lo parcial.
func (a *App) cancelarEjecucion() {
	if a.cancelarTurno == nil {
		return
	}
	a.cancelarTurno()
	a.olvidarCancelacion()
	a.chat.CancelarTurno()
}

// olvidarCancelacion suelta el cancelador del turno que ya terminó.
func (a *App) olvidarCancelacion() {
	a.cancelarTurno = nil
	a.cancelarArmado = false
}

// editar entrega una tecla de edición al componente con el foco: la línea de la
// bienvenida o la caja de la vista principal. No es una acción: mueve el cursor
// o escribe texto (SPEC-KEYBINDS §Resolución por contexto).
func (a *App) editar(m tea.KeyMsg) tea.Cmd {
	if a.vista == VistaBienvenida {
		a.editarBienvenida(m)
		return nil
	}
	modelo, cmd := a.entrada.Update(m)
	a.entrada = modelo
	a.paleta.ActualizarDesdeLinea(a.entrada.Texto)
	return cmd
}

// despachar ejecuta el efecto de una acción resuelta por el KeyResolver. Es el
// único sitio de la app que traduce acciones a comportamiento: los componentes
// ya no comparan strings de tecla.
func (a *App) despachar(accion Accion, m tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch accion {
	case AccionSalir:
		return a.salir()
	case AccionEnviar:
		return a, a.enviarActual()
	case AccionCiclarAgente:
		a.ciclarAgente()
		a.guardarPreferencias()
		return a, nil
	case AccionPanel:
		// El sidebar se pliega y se despliega sin interrumpir ninguna sesión.
		a.panel.Alternar()
		return a, nil
	case AccionCancelar:
		a.detenerTurno()
		return a, nil
	case AccionCerrarModal:
		a.cerrarModal()
		return a, nil
	case AccionModalModelos:
		return a, a.abrirModalModelos()
	case AccionDescargarModal:
		return a, a.abrirModalDescargar()
	case AccionSelector:
		return a, a.abrirModalSesiones()
	case AccionSesionNueva:
		return a, a.nuevaSesion()
	case AccionAyuda:
		return a, a.alternarModalAtajos()
	case AccionEliminarSesion:
		return a, a.eliminarResaltada()
	case AccionSubir:
		a.moverModal(-1)
		return a, nil
	case AccionBajar:
		a.moverModal(1)
		return a, nil
	case AccionConfirmar:
		return a, a.aplicarModal()
	case AccionRazonamiento:
		a.chat.AlternarRazonamiento()
		return a, nil
	case AccionChatSubir:
		a.chat.Subir(1)
		return a, nil
	case AccionChatBajar:
		a.chat.Bajar(1)
		return a, nil
	case AccionChatPaginaArriba:
		a.chat.SubirPagina()
		return a, nil
	case AccionChatPaginaAbajo:
		a.chat.BajarPagina()
		return a, nil
	case AccionAprobar:
		return a, a.decidirAprobacion(true)
	case AccionDeclinar:
		return a, a.decidirAprobacion(false)
	case AccionAprobaciones:
		if a.aprobaciones.Abierto {
			a.focoAprobaciones = !a.focoAprobaciones
		}
		return a, nil
	}
	// Las acciones de aprobaciones y modales llegan con sus tareas: ya viajan
	// como acciones, no como teclas.
	return a, nil
}

// decidirAprobacion entrega la decisión de la línea resaltada a su sesión y la
// retira del panel: la línea ya no espera nada.
func (a *App) decidirAprobacion(aprobar bool) tea.Cmd {
	item, ok := a.aprobaciones.Seleccionada()
	if !ok {
		return nil
	}
	a.decidirItem(item, aprobar)
	return nil
}

// decidirItem entrega la decisión de una aprobación concreta y la retira del
// panel.
func (a *App) decidirItem(item Aprobacion, aprobar bool) {
	if a.puerto == nil {
		return
	}
	if err := a.puerto.ResponderAprobacion(item.Sesion, item.ID, aprobar); err != nil {
		a.aviso = err.Error()
		return
	}
	a.aprobaciones.Retirar(item.ID)
	a.panel.Aprobaciones = a.aprobaciones.Pendientes()
	a.sincronizarFocoAprobaciones()
}

// sincronizarFocoAprobaciones quita el foco cuando ya no queda nada que decidir.
func (a *App) sincronizarFocoAprobaciones() {
	if !a.aprobaciones.Abierto {
		a.focoAprobaciones = false
	}
}

// guardarPreferencias recuerda el agente y el razonamiento. El modelo no se
// guarda aquí: vive en el catálogo (`models.json`).
func (a *App) guardarPreferencias() {
	a.prefs.Agente = a.agente
	a.prefs.Pensar = a.entrada.Pensar
	if err := GuardarPreferencias(a.prefs); err != nil {
		a.aviso = "no se pudieron guardar las preferencias: " + err.Error()
	}
}

// alternarPensar cambia el razonamiento del chat, lo recuerda y lo comunica a la
// sesión: apagado, el turno pide al modelo que no razone.
func (a *App) alternarPensar() {
	a.entrada.AlternarPensar()
	a.guardarPreferencias()
	if a.puerto != nil {
		a.puerto.FijarPensar(a.entrada.Pensar)
	}
}

// cargarPreferencias aplica el último agente recordado. Un archivo ausente o
// ilegible no rompe el arranque: se queda con los valores vigentes. El modelo y
// el motor no se releen de las preferencias: son del proveedor, que ya los
// resolvió al construirse a partir del catálogo.
func (a *App) cargarPreferencias() {
	p, err := CargarPreferencias()
	if err != nil {
		a.aviso = "no se pudieron leer las preferencias: " + err.Error()
		return
	}
	a.prefs = p
	if p.Agente != "" {
		a.agente = ValidarAgente(p.Agente, a.agentes)
	}
	a.entrada.Agente = a.agente
	a.entrada.Modelo = a.modelo
	a.entrada.Motor = a.motor
	a.entrada.Pensar = p.Pensar
	if a.puerto != nil {
		a.puerto.FijarPensar(p.Pensar)
	}
}

// cargarEstilos aplica la paleta del usuario (`~/.config/localcli/styles.json`).
// Si el archivo es inválido se rechaza entero y se arranca con los estilos de
// fábrica, avisando.
func (a *App) cargarEstilos() {
	if err := CargarEstilos(); err != nil {
		a.aviso = "no se pudo cargar styles.json; se usan los estilos de fábrica: " + err.Error()
	}
}

// cargarTeclas carga el mapa de atajos del usuario. Si el archivo es inválido se
// rechaza entero y se arranca con el de fábrica, avisando.
func (a *App) cargarTeclas() {
	km, err := CargarKeymap()
	if err != nil {
		a.aviso = "keys.json inválido; se usan los atajos de fábrica: " + err.Error()
		a.teclaRes = NuevoKeyResolver(KeymapPorDefecto())
		return
	}
	a.teclaRes = NuevoKeyResolver(km)
}

// ciclarAgente pasa al siguiente agente de la lista.
func (a *App) ciclarAgente() {
	if len(a.agentes) == 0 {
		return
	}
	a.agente = a.agentes[(a.posicionAgente(a.agente)+1)%len(a.agentes)]
}

func (a *App) posicionAgente(nombre string) int {
	for i, agente := range a.agentes {
		if agente == nombre {
			return i
		}
	}
	return 0
}

func (a *App) textoVigente() string {
	if a.vista == VistaBienvenida {
		return a.bienvenida.Texto
	}
	return a.entrada.Texto
}

func (a *App) tieneAgente(nombre string) bool {
	for _, agente := range a.agentes {
		if agente == nombre {
			return true
		}
	}
	return false
}

// salir cierra la suscripción y termina el bucle.
func (a *App) salir() (tea.Model, tea.Cmd) {
	if a.cerrar != nil {
		a.cerrar()
		a.cerrar = nil
	}
	return a, tea.Quit
}
