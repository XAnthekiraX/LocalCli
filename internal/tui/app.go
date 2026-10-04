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
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

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
	// Agente es el agente activo —uno de `Agentes`—, el que pinta el indicador a
	// la izquierda del input y el que viaja con cada petición (SPEC-INTERFAZ
	// §Zonas 2, T-F015). Lo recorre `Tab`; con un modal abierto la acción no
	// llega a cambiarlo.
	Agente string
	// Agentes es la lista de agentes disponibles, en orden, que ofrece el motor
	// (cargados de `.localcli/agents/*.json`). `Tab` recorre esta lista; con una lista
	// vacía la vista cae en los base (`plan`, `build`).
	Agentes []string
	// Aviso es la línea transitoria de la vista (por ahora, el modelo elegido
	// que no puede usar herramientas). Se pinta al pie de la bienvenida y de la
	// principal; se limpia al cambiar a un modelo capaz o al enviar.
	Aviso string
	// MotorNombre es el nombre visible de la INSTANCIA del motor de la sesión
	// activa, y MotorAviso su aviso si ese motor falta o está desactivado. Los
	// resuelve el puerto al cargar la sesión y alimentan la línea de estado bajo
	// el input (SPEC-MODELO-MOTOR §Desactivar). MotorIDSesion es el `id` con el
	// que el modal de motores marca el de la sesión. MotorEtiqueta es lo que se
	// rotula junto al modelo en las dos líneas de estado: arranca en el motor por
	// defecto y pasa a ser el de la sesión al cargarla.
	MotorNombre   string
	MotorAviso    string
	MotorIDSesion string
	MotorEtiqueta string
	// ModeloHerramientas y CapHerramientasConocida dicen si el modelo en uso
	// tiene acceso a herramientas y si eso se sabe ya. La capacidad se consulta
	// una vez al arrancar y se refresca al elegir modelo; sin dato conocido, la
	// línea de estado muestra «?».
	ModeloHerramientas      bool
	CapHerramientasConocida bool
	// ModeloVision y CapVisionConocida hacen lo mismo para la capacidad de
	// interpretar imágenes. No bloquean nada: alimentan la línea de estado y el
	// aviso al enviar una imagen con un modelo que no la declara.
	ModeloVision      bool
	CapVisionConocida bool
	// Pensar es el interruptor de razonamiento del pie: apagado por defecto, se
	// cambia con un clic y vale para los turnos siguientes. ModeloPensar y
	// CapPensarConocida dicen si el modelo en uso razona: sin eso la chapa no se
	// enseña, porque a un modelo que no razona no se le puede mandar `think`
	// (SPEC-OLLAMA-PERFIL).
	Pensar            bool
	ModeloPensar      bool
	CapPensarConocida bool
	// RazonAviso es el aviso de que el razonamiento quedó desactivado porque no
	// se pudo comprobar: es el único de los tres estados desconocidos que avisa,
	// porque afecta al turno en curso y no basta con la marca
	// (SPEC-MODELO-MOTOR §Capacidades). Vacío cuando hay dato.
	RazonAviso string

	// PidiendoCancelarEsc es la confirmación de doble `esc` cuando la sesión
	// está trabajando: el primer esc pregunta, el segundo cancela.
	PidiendoCancelarEsc bool

	// Estado de la selección con el ratón y el último marco pintado, del que se
	// extrae el texto al soltar (selection.go). ratonIni es el ancla (donde se
	// pulsó) y ratonFin el puntero. Mientras difieren, la selección se pinta.
	ratonSelec  bool
	ratonIni    posicion
	ratonFin    posicion
	ultimaVista string
	// chatFilaIni/chatFilaFin son las filas que ocupa la ventana del historial
	// dentro del marco pintado, y iniEnChat/finEnChat dicen si el ancla y el
	// puntero viven en esa banda. Es lo que permite a la rueda reanclar la
	// selección al texto sin desplazar lo que no es chat (selection.go).
	chatFilaIni int
	chatFilaFin int
	iniEnChat   bool
	finEnChat   bool
	// copiado enciende el aviso transitorio [Copiado] (arriba a la derecha) tras
	// copiar una selección; copiadoGen invalida el temporizador de una copia
	// anterior para que no apague el aviso de una copia nueva.
	copiado    bool
	copiadoGen uint64
	// Los tres modales (SPEC-INTERFAZ §Modales: "Tres modales centrados
	// comparten el mismo comportamiento: uno abierto a la vez… Esc cierra sin
	// cambios"). Todos se abren con acciones globales del mapa —session_picker,
	// command_palette y model_picker— y se cierran con `dismiss`.
	Modelos     ModelsModal
	Sesiones    SessionsModal
	AtajosModal KeysModal
	Motores     MotorsModal

	// Entrada es la línea de texto de la interfaz principal (input.go, T-F004).
	Entrada Entrada

	// Comandos es el catálogo de comandos de flujo disponibles: los del proyecto
	// —oficiales más los de `.localcli/flows/*.json`— que entrega el puerto. La paleta
	// lo lista y la entrada reconoce contra él (comandos.go).
	Comandos []ComandoFlujo

	// Paleta es la lista de comandos de flujo que se despliega encima del input
	// mientras se escribe un comando (comandos.go). La alimenta el texto de la
	// línea de entrada.
	Paleta Paleta

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
	// ticks vigente. Al arrancar se incrementa, y una cadena de una generación
	// anterior se detiene en su siguiente tick, así que nunca se acumulan.
	latido uint64
	// animando dice si la cadena de latidos vigente sigue viva. Evita arrancar
	// una segunda cadena con cada evento: solo se arma cuando no hay ninguna.
	animando bool
	// frame es el frame del indicador en vivo ([⠋ Pensando]): avanza un paso
	// por latido mientras hay turno. El glifo lo elige glifoActividad.
	frame int
	// herramientaEnCurso es la herramienta que se está ejecutando, si la hay.
	// Mientras no esté vacía, el indicador en vivo dice «Usando herramienta: X»
	// en lugar de «Pensando». Se pone al invocarse y se limpia al terminar.
	herramientaEnCurso string

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
	// El catálogo de comandos sale del puerto (flujos oficiales + propios); sin
	// él queda el respaldo oficial de la vista (comandos.go).
	comandos := p.Comandos()
	if len(comandos) == 0 {
		comandos = comandosDeFlujo()
	}
	// La lista de agentes sale del puerto (los base + los propios de
	// `.localcli/agents/*.json`); sin lista, quedan los base. El agente activo arranca
	// en el último recordado si sigue disponible, o en el primero.
	agentes := p.Agentes()
	if len(agentes) == 0 {
		agentes = []string{AgentePlan, AgenteBuild}
	}
	a := &App{
		Puerto:   p,
		Mapa:     mapa,
		TeclaRes: NuevoKeyResolver(mapa),
		Atajos:   mapa.Entradas(),
		Vista:    VistaBienvenida,
		Panel:    NuevoPanel(),
		// El modelo y su motor se leen una vez del motor, que ya los detectó al
		// arrancar: son datos en memoria, no llamadas al servidor, así que la
		// bienvenida se sigue pintando sin esperar a nada externo
		// (SPEC-INTERFAZ §Línea de modelo, DOMAIN §3).
		// `ModeloActual()` trae el modelo ROTULADO con su motor; la vista guarda
		// el modelo solo y el rótulo aparte, porque el motor es de la sesión y
		// cambia al cambiar de ella (SPEC-MODELO-MOTOR §Motor y modelo por
		// sesión): así la etiqueta no queda pegada al nombre que se le manda
		// al motor con `FijarModelo`.
		MotorEtiqueta: p.MotorActual(),
		Modelo:        modeloSoloDe(p.ModeloActual(), p.MotorActual()),
		// El razonamiento se muestra por defecto (SPEC-INTERFAZ: "se puede
		// ocultar", luego visible es el estado normal).
		Razon:      NuevoRazonamiento(),
		Aprobs:     Aprobaciones{},
		Bienvenida: NuevaBienvenida(),
		Entrada:    NuevaEntrada(),
		// El agente arranca en el último recordado si sigue disponible; si no,
		// en el primero de la lista (SPEC-INTERFAZ §Zonas 2: el indicador
		// muestra el agente activo, y el reparto de etapas lo hace el motor).
		Agente:   ValidarAgente(p.AgenteRecordado(), agentes),
		Agentes:  agentes,
		Comandos: comandos,
		// El interruptor de razonamiento arranca donde lo dejó el usuario: la
		// vista es la dueña de la preferencia, pero el valor vigente lo tiene el
		// motor, que es quien lo aplica a cada turno.
		Pensar: p.PensarRecordado(),
		ctx:    context.Background(),
	}
	a.Paleta.FijarComandos(comandos)
	// El catálogo de extensiones de cada tipo lo decide el registro; el modal lo
	// pide al puerto en vez de llevar una copia (SPEC-MODELO-MOTOR §El catálogo
	// es cerrado).
	a.Motores.CatalogoExtensiones = p.ExtensionesValidas
	// La carpeta del proyecto y el estado de su repositorio son datos fijos del
	// arranque —una sesión trabaja siempre en la misma rama—, así que se leen
	// una vez aquí y se pintan en el pie del panel (SPEC-INTERFAZ §Zonas 3). La
	// pantalla no pregunta nada: no llama a git ni al disco.
	a.Panel.Ruta = p.Carpeta()
	a.Panel.GitRama, a.Panel.GitCambios = p.Git()
	return a
}

// Init arma la escucha del canal del motor (T-F010-06): el comando espera el
// siguiente evento y el bucle lo relanza tras cada uno. Así la vista recibe los
// eventos sin goroutines propias, como manda la arquitectura Elm.
//
// T-F013: aquí ya no se pide la lista de modelos. Se pide al abrir el modal
// (SPEC-INTERFAZ §Modal de modelos: "La lista se pide a Ollama al abrir el
// modal (no en el arranque)"), así que arrancar no toca Ollama.
func (a *App) Init() tea.Cmd {
	// Además de la escucha del motor, se pregunta una vez si el modelo en uso
	// tiene acceso a herramientas, para la línea de estado bajo el input.
	return tea.Batch(a.escucharCmd(), a.cmdCapacidades())
}

// cmdCapacidades pregunta al motor qué declara capaz de hacer el modelo en uso.
// Llega como `capacidadesMsg`; un fallo deja el dato como desconocido y no
// bloquea nada.
func (a *App) cmdCapacidades() tea.Cmd {
	if a.Modelo == "" {
		return nil
	}
	nombre := a.Modelo
	return func() tea.Msg {
		caps, err := a.Puerto.CapacidadesModelo(nombre)
		return capacidadesMsg{Nombre: nombre, Herramientas: caps.Herramientas, Vision: caps.Vision, Pensar: caps.Pensar, Err: err}
	}
}

// intervaloLatido es la cadencia del latido en vivo. Es rápida para que el
// indicador ([⠋ Pensando]) gire con fluidez mientras el modelo trabaja; el
// contador de tiempo se lee de `time.Since`, así que no depende del intervalo.
const intervaloLatido = 120 * time.Millisecond

// tickMsg es el latido del contador en vivo: cada `intervaloLatido` mientras
// hay una respuesta en camino, para que el tiempo corra y el indicador gire en
// pantalla aunque el modelo todavía no haya emitido un token. `gen` es la
// generación del contador a la que pertenece el latido: solo la cadena vigente
// se re-arma.
type tickMsg struct{ gen uint64 }

// tickCmd programa el siguiente latido de la generación dada. El comando se
// re-arma a sí mismo mientras el turno siga vivo.
func tickCmd(gen uint64) tea.Cmd {
	return tea.Tick(intervaloLatido, func(time.Time) tea.Msg { return tickMsg{gen: gen} })
}

// asegurarLatido arranca la cadena de latidos si hay algo en vivo y no hay ya
// una corriendo. Es el único punto que incrementa la generación: así el
// contador y el glifo se reanudan al activar una sesión que trabaja o al llegar
// un evento, no solo al enviar. Sin actividad —o con una cadena ya viva— no
// hace nada.
func (a *App) asegurarLatido() tea.Cmd {
	if !a.enActividad() || a.animando {
		return nil
	}
	a.animando = true
	a.latido++
	return tickCmd(a.latido)
}

// Update maneja las teclas, el tamaño y los mensajes que llegan por eventos.
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case tea.MouseMsg:
		return a.raton(m)
	case tea.WindowSizeMsg:
		a.Ancho, a.Alto = m.Width, m.Height
		a.Entrada.FijarAncho(a.anchoColumna())
		// La edición interna del campo (flechas, home/end, borrado por palabra)
		// sigue viva; el KeyResolver resuelve antes las teclas que el mapa
		// reclama. Se reaplica en cada cambio de tamaño para que un textinput
		// reconstruido nunca recupere sus atajos viejos.
		a.Entrada.AjustarTeclasPropias()
		return a, nil
	case tea.KeyMsg:
		// Una tecla es una interacción nueva: la selección del ratón deja de
		// estar vigente, para que su realce no quede pegado sobre lo que venga
		// después (un modal, otra vista). La rueda no pasa por aquí.
		a.limpiarSeleccion()
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
		// quedaría sorda (T-F010-06). Además se asegura el latido: un evento que
		// abre actividad (un token, una herramienta) reanuda el contador y el
		// glifo aunque el turno no lo haya arrancado esta vista.
		return a, tea.Batch(a.escucharCmd(), a.asegurarLatido())
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
	case motoresMsg:
		// La lista llega cuando el registro responde: rellena el modal que la
		// pidió. Si ya no está abierto, el mensaje se ignora (llegó tarde); un
		// fallo queda como aviso «sin motores» y nada se bloquea.
		a.Motores.FijarMotores(m.Motores, m.Sesiones)
		return a, nil
	case motorSesionMsg:
		// El motor de la sesión activa alimenta la línea de estado. El de otra
		// sesión se descarta: la línea refleja la activa, y solo esa.
		if m.Sesion == a.Panel.SesionID {
			a.MotorIDSesion = m.ID
			a.MotorNombre = m.Nombre
			a.MotorAviso = m.Aviso
			if m.Nombre != "" {
				// La etiqueta que acompaña al modelo pasa a ser la de ESTA sesión:
				// al volver a una se retoma contra el motor que tenía
				// (SPEC-MODELO-MOTOR §Motor y modelo por sesión).
				a.MotorEtiqueta = m.Nombre
			}
		}
		return a, nil
	case motorAplicadoMsg:
		// Un motor distinto ya es el de la sesión: el modelo en uso se vacía si el
		// motor nuevo no lo declara (SPEC-MODELO-MOTOR §Cambiar a mitad).
		if m.Sesion == a.Panel.SesionID {
			a.Modelo = ModeloTrasCambiarDeMotor(a.Modelo, m.Modelos)
		}
		return a, a.cmdMotorDeSesion(m.Sesion)
	case capacidadesMsg:
		// La respuesta llega para el modelo con el que se preguntó. Si para
		// entonces el usuario cambió de modelo, se descarta: la línea de estado
		// refleja el modelo en uso, no el consultado. El estado de tres valores
		// se recibe ya resuelto del puerto: `desconocida` deja la chapa en «?» y
		// `no soportada` la pinta sabiendo que no puede (INTERFACES §3.2).
		if m.Nombre == a.Modelo {
			a.CapHerramientasConocida = m.Err == nil && m.Herramientas != CapacidadDesconocida
			a.ModeloHerramientas = m.Herramientas == CapacidadSoportada
			a.CapVisionConocida = m.Err == nil && m.Vision != CapacidadDesconocida
			a.ModeloVision = m.Vision == CapacidadSoportada
			a.CapPensarConocida = m.Err == nil && m.Pensar != CapacidadDesconocida
			a.ModeloPensar = m.Pensar == CapacidadSoportada
			// El razonamiento desconocido no ofrece la chapa y, además, avisa: es
			// el único de los tres que el usuario no puede deducir del `?`, porque
			// le cambia el turno que está haciendo.
			if m.Err == nil && m.Pensar == CapacidadDesconocida {
				a.RazonAviso = "el razonamiento está desactivado: no se pudo comprobar si el modelo razona; el turno va sin `think`"
			} else {
				a.RazonAviso = ""
			}
		}
		return a, nil
	case aprobacionesMsg:
		a.Aprobs.Fijar(m.Items)
		// La instantánea de pendientes alimenta también el contador del panel.
		a.Panel.Aprobaciones = a.Aprobs.Pendientes()
		return a, nil
	case historialMsg:
		// La carga llega cuando la sesión pedida sigue siendo la activa; si el
		// usuario cambió de sesión mientras volaba, se descarta: el chat nunca
		// muestra el historial de otra sesión (T-F005-05), ni la lista de pasos
		// de otra sesión.
		if m.Sesion == a.Panel.SesionID {
			a.Chat.Cargar(m.Mensajes)
			a.Panel.Tareas = m.Tareas
			// Los números del contexto son de la sesión cargada: el panel deja
			// de mostrar los de la anterior (SPEC-PANEL-CONTEXTO).
			a.Panel.ContextoTokens = m.ContextoTokens
			a.Panel.LimiteTokens = m.LimiteTokens
			a.Panel.LimiteDelHarness = m.LimiteDelHarness
			a.Panel.TokensEstimados = true
			// El motor de la sesión recién cargada alimenta la línea de estado
			// (SPEC-MODELO-MOTOR §Motor y modelo por sesión).
			cmdMotor := a.cmdMotorDeSesion(m.Sesion)
			// Cargar vacía el chat y deja `inicio` a cero: si la sesión retomada
			// sigue trabajando, se reanuda su reloj para que el contador y el
			// glifo no se queden mudos tras el cambio.
			if a.Panel.Estado == session.EstadoTrabajando {
				a.Chat.ReanudarTurno()
				return a, tea.Batch(a.asegurarLatido(), cmdMotor)
			}
			return a, cmdMotor
		}
		return a, nil
	case enviadoMsg:
		// La petición ya salió: se asegura el latido del contador y del glifo.
		// Al enviar, el chat ya arrancó su turno (`AñadirUsuario`), así que hay
		// actividad y la cadena se arma.
		return a, a.asegurarLatido()
	case copiadoExpiradoMsg:
		// El aviso [Copiado] se apaga solo al vencer su tiempo; un temporizador
		// de una copia anterior no apaga el aviso de una copia nueva.
		if m.gen == a.copiadoGen {
			a.copiado = false
		}
		return a, nil
	case tickMsg:
		// El contador late mientras hay algo en vivo y la cadena sea la vigente;
		// al dejar de haber actividad (o quedar obsoleta la generación), el
		// siguiente latido ya no se re-arma y el contador se detiene solo. Cada
		// latido avanza un frame del indicador en vivo.
		if m.gen != a.latido {
			return a, nil
		}
		if a.enActividad() {
			a.frame++
			return a, tickCmd(m.gen)
		}
		a.animando = false
		return a, nil
	case errorMsg:
		// Un envío que no llegó a arrancar no deja turno que fechar; uno que ya
		// está trabajando se deja en paz (el error puede ser de otra lectura).
		if !a.enTurno {
			a.Chat.CancelarTurno()
		}
		// El aviso no puede adelantarse al texto en vuelo: se cierra el segmento
		// para que la línea de error quede debajo de lo ya dicho.
		a.cerrarSegmentoEnVivo()
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

	// El formulario de alta/edición y la confirmación de borrado del modal de
	// motores capturan el teclado mientras están en pantalla: sus campos reciben
	// el texto y ninguna tecla resuelve acciones de la vista de abajo
	// (SPEC-INTERFAZ §Modales: con un modal abierto, sus teclas son del modal).
	if a.Motores.Abierto && a.Motores.FormAbierto() {
		return a.teclaFormMotor(m)
	}
	if a.Motores.Abierto && a.Motores.ConfirmandoBorrado() {
		return a.teclaConfirmarBorrarMotor(m)
	}

	// Doble `esc` para cancelar el trabajo en curso: el primer esc pregunta, el
	// segundo cancela. Solo sin modal abierto (con modal, esc lo cierra) y solo
	// si la sesión está trabajando. Cualquier otra tecla descarta la
	// confirmación y sigue su curso normal.
	if a.PidiendoCancelarEsc {
		a.PidiendoCancelarEsc = false
		if m.Type == tea.KeyEsc {
			if id := a.Panel.SesionID; id != "" {
				a.Puerto.Cancelar(id)
			}
			return a, nil
		}
	}
	if m.Type == tea.KeyEsc && !a.modalAbierto() && a.trabajando() {
		a.PidiendoCancelarEsc = true
		return a, nil
	}

	// El contexto de resolución (T-F012-04): modal > aprobaciones > input >
	// vista. El panel de aprobaciones solo reclama el teclado cuando está
	// enfocado (`Ctrl+A`); visible sin foco, el input sigue siendo el dueño de
	// las letras. La líder funciona desde cualquier contexto (SPEC-KEYBINDS
	// §Tecla líder: escape hatch).
	ctx := ContextoVista
	switch {
	case a.modalAbierto():
		ctx = a.contextoModal()
	case a.Aprobs.Enfocado:
		ctx = ContextoAprobaciones
	}

	// Regla de oro de SPEC-KEYBINDS §Resolución por contexto: "con el input
	// enfocado, las letras sueltas son texto, nunca atajo". En la interfaz
	// principal el input está enfocado salvo cuando un modal o el panel de
	// aprobaciones ENFOCADO reclama el teclado, así que el texto puro se entrega
	// al editor antes de resolver acciones — con una excepción: durante una
	// espera de líder, todas las pulsaciones van al resolver (el literal
	// «ctrl+x» no deja texto huérfano y la segunda tecla cierra la secuencia).
	if ctx == ContextoVista && !a.TeclaRes.EsperandoLeader() && EsEntradaDeTexto(m) {
		return a, a.actualizarEntrada(m)
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
		a.Entrada.FijarAncho(a.anchoColumna())
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
		// Revelar u ocultar el texto crudo del razonamiento no detiene la
		// generación: solo cambia lo que se pinta. El historial cerrado usa la
		// misma bandera, para que el toggle valga en toda la pantalla.
		a.Razon.Alternar()
		a.Chat.MostrarRazonamiento = a.Razon.Visible
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
		// `Ctrl+A` enfoca o desenfoca el panel para decidir con el teclado. El
		// panel se ve solo cuando hay algo pendiente; el foco es lo que le da el
		// teclado, y solo puede tenerlo si hay algo que decidir.
		if a.Aprobs.Enfocado {
			a.Aprobs.Enfocado = false
		} else if len(a.Aprobs.Items) > 0 {
			a.Aprobs.Enfocado = true
			a.Aprobs.Abierto = true
		}
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
		// modales, y el panel de aprobaciones si tenía el foco (o si el usuario
		// lo cierra explícitamente). Sin nada abierto no hace nada visible
		// (SPEC-INTERFAZ §Modales, criterio: "Esc cierra cualquier modal sin
		// cambiar nada").
		a.cerrarModales()
		a.Aprobs.Enfocado = false
		a.Aprobs.Abierto = false
		return a, nil
	case AccionEnviar:
		// Enter es ambiguo y lo resuelve lo que está abierto: con el modal de
		// modelos aplica el resaltado, con el de sesiones abre esa sesión y en
		// la interfaz principal envía la petición (INTERFACES §4: "Enter aplica
		// lo resaltado en el modal y lo cierra"). Con el de atajos no hay nada
		// que aplicar: es de solo lectura y la tecla no llega a la vista.
		switch {
		case a.Motores.Abierto && a.Motores.FormAbierto():
			return a, a.guardarMotor()
		case a.Motores.Abierto:
			return a, a.aplicarMotor()
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
		// modal de atajos no aparece: es una tabla de solo lectura. Con un
		// formulario abierto las flechas cambian de campo.
		switch {
		case a.Motores.Abierto && a.Motores.FormAbierto():
			a.Motores.FormMoverCampo(-1)
		case a.Motores.Abierto:
			a.Motores.Mover(-1)
		case a.Modelos.Abierto:
			a.Modelos.Mover(-1)
		case a.Sesiones.Abierto:
			a.Sesiones.Mover(-1)
		case a.Aprobs.Enfocado:
			a.Aprobs.Mover(-1)
		}
		return a, nil
	case AccionBajar:
		switch {
		case a.Motores.Abierto && a.Motores.FormAbierto():
			a.Motores.FormMoverCampo(1)
		case a.Motores.Abierto:
			a.Motores.Mover(1)
		case a.Modelos.Abierto:
			a.Modelos.Mover(1)
		case a.Sesiones.Abierto:
			a.Sesiones.Mover(1)
		case a.Aprobs.Enfocado:
			a.Aprobs.Mover(1)
		}
		return a, nil
	case AccionCiclarAgente:
		// Recorre la lista de agentes disponibles (SPEC-KEYBINDS §Acción
		// `agent_cycle`; SPEC-INTERFAZ §Zonas 2: "Cambia al instante con Tab").
		// La acción es de contexto de vista, así que con un modal abierto no
		// llega hasta aquí: el indicador y lo que se envía son los del último
		// `Tab` pulsado.
		//
		// Con la paleta de comandos desplegada, Tab no cambia de agente:
		// autocompleta el comando resaltado para poder escribir la petición
		// detrás (`/comando [petición]`).
		if a.Paleta.Abierto {
			return a, a.autocompletarComando()
		}
		a.ciclarAgente()
		return a, nil
	case AccionChatSubir:
		// Con la paleta de comandos desplegada, las flechas la recorren a ella;
		// sin paleta, recorren el historial del chat.
		if a.Paleta.Abierto {
			a.Paleta.Mover(-1)
			return a, nil
		}
		a.Chat.Subir(1)
		return a, nil
	case AccionChatBajar:
		if a.Paleta.Abierto {
			a.Paleta.Mover(1)
			return a, nil
		}
		a.Chat.Bajar(1)
		return a, nil
	case AccionChatPaginaArriba:
		a.Chat.SubirPagina()
		return a, nil
	case AccionChatPaginaAbajo:
		a.Chat.BajarPagina()
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
	case AccionModalMotores:
		// El modal de motores es global, como el de modelos: se abre y se cierra
		// desde la bienvenida y desde la interfaz principal (INTERFACES §4).
		if a.Motores.Abierto {
			a.Motores.Cerrar()
			return a, nil
		}
		return a, a.abrirModalMotores()
	case AccionMotorNuevo:
		// `n` solo tiene sentido con el modal de motores abierto y sin el
		// formulario ya en pantalla (SPEC-KEYBINDS `motor_new`).
		if a.Motores.Abierto && !a.Motores.FormAbierto() {
			a.Motores.AbrirAlta()
		}
		return a, nil
	case AccionMotorEditar:
		// `e` edita el resaltado (SPEC-KEYBINDS `motor_edit`).
		if a.Motores.Abierto && !a.Motores.FormAbierto() {
			a.Motores.AbrirEdicion()
		}
		return a, nil
	case AccionEliminarMotor:
		// `Ctrl+D` elimina el resaltado; si alguna sesión lo usa, pide
		// confirmación antes de borrarlo (SPEC-MODELO-MOTOR §Eliminar).
		if !a.Motores.Abierto {
			return a, nil
		}
		return a, a.eliminarMotorResaltado()
	}

	// Lo que no es acción se entrega al componente con el foco: con un modal
	// abierto solo viven sus teclas, ya tratadas arriba, y el resto se traga
	// aquí — nada llega a la línea de entrada (SPEC-INTERFAZ §Modal de modelos);
	// con el panel de aprobaciones enfocado pasa igual; si no, lo escrito lo
	// compone la línea de entrada (INTERFACES §5). El panel visible sin foco no
	// interviene: el input sigue escribiendo.
	if a.modalAbierto() {
		return a, nil
	}
	if a.Aprobs.Enfocado {
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
	// sesión nueva no hereda nada de la que se estaba viendo. Se pide su
	// historial —vacío— para que los números del contexto dejen de ser los de la
	// sesión anterior (SPEC-PANEL-CONTEXTO).
	a.activar(ses)
	a.Vista = VistaPrincipal
	return tea.Batch(a.cmdHistorial(ses.ID), a.asegurarLatido())
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

// ejecutarEliminar mata la sesión y refresca el modal. Si era la activa y no
// queda ninguna otra, la vista vuelve a la bienvenida; si quedan, se retoma la
// más reciente para que el chat no apunte a una sesión que ya no existe.
func (a *App) ejecutarEliminar(id string) tea.Cmd {
	if err := a.Puerto.Eliminar(id); err != nil {
		a.Chat.AñadirSistema("no se pudo eliminar la sesión: " + err.Error())
		return nil
	}
	if a.Panel.SesionID != id {
		// El modal sigue abierto con la lista ya refrescada.
		return a.abrirModalSesiones()
	}
	ses, err := a.Puerto.ResolverActiva()
	if err != nil {
		a.Chat.AñadirSistema("no se pudo abrir la sesión: " + err.Error())
		return nil
	}
	if ses == nil {
		// No queda ninguna sesión: la vista cambia a la bienvenida de inmediato
		// (SPEC-SESIONES); la siguiente petición creará otra.
		a.volverABienvenida()
		return nil
	}
	a.activar(ses)
	return tea.Batch(a.cmdHistorial(ses.ID), a.abrirModalSesiones())
}

// volverABienvenida regresa a la pantalla inicial cuando el proyecto se quedó
// sin sesiones: limpia la sesión activa y el chat, y cierra cualquier modal. La
// bienvenida no muestra sesión, así que no queda nada apuntando a una que ya no
// existe. No toca el agente activo ni el modelo elegido: no son de la sesión.
func (a *App) volverABienvenida() {
	a.cerrarModales()
	a.Vista = VistaBienvenida
	a.Panel.SesionID = ""
	a.Panel.Sesion = ""
	a.Panel.Estado = ""
	a.Panel.Capa = ""
	a.Panel.Tareas = nil
	a.Chat.Vaciar()
	// El razonamiento revelado es una preferencia de la vista: no se pierde al
	// quedarse sin sesión.
	a.Chat.MostrarRazonamiento = a.Razon.Visible
	razon := NuevoRazonamiento()
	razon.Visible = a.Razon.Visible
	a.Razon = razon
	// Sin sesión no hay contexto que medir.
	a.Panel.ContextoTokens = 0
	a.Panel.LimiteTokens = 0
	a.Panel.LimiteDelHarness = false
	a.Panel.TokensEstimados = false
	a.enTurno = false
	a.PidiendoCancelarEsc = false
	a.iniciarTurno()
	a.Bienvenida = NuevaBienvenida()
	a.Paleta.Filtrar("")
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
	// Un comando de flujo no es una petición al modelo: se reconoce y se
	// responde en el chat (SPEC-INTERFAZ §Reglas de negocio: un flujo arranca
	// con su comando explícito).
	if c, ok := a.comandoAplicable(texto); ok {
		return a.ejecutarComando(c, texto)
	}
	id := a.Panel.SesionID
	if id == "" {
		ses, err := a.Puerto.Crear()
		if err != nil {
			a.Chat.AñadirSistema("no se pudo crear la sesión: " + err.Error())
			return nil
		}
		a.activar(ses)
		id = ses.ID
	}
	a.Entrada.Limpiar()
	a.Paleta.Filtrar("")
	a.Chat.AñadirUsuario(texto)
	a.iniciarTurno()
	a.Vista = VistaPrincipal
	imagenes := a.prepararAdjuntos(texto)
	return a.enviarCmd(id, texto, imagenes)
}

// prepararAdjuntos detecta imágenes en el texto del turno, deja en el chat los
// avisos de lo que no se pudo leer y, si el modelo en uso no declara visión,
// avisa sin bloquear. Devuelve las imágenes (base64) que viajarán con el turno.
func (a *App) prepararAdjuntos(texto string) []string {
	imagenes, avisos := AdjuntosDe(texto)
	for _, av := range avisos {
		a.Chat.AñadirSistema(av)
	}
	if len(imagenes) > 0 && a.CapVisionConocida && !a.ModeloVision {
		a.Chat.AñadirSistema("el modelo «" + a.Modelo + "» no declara visión; la imagen puede no interpretarse")
	}
	return imagenes
}

// enviarCmd entrega la petición al puerto con el agente que está activo. El
// indicador no es decorativo: es lo que el motor recibe, así que la petición
// sale con el agente elegido y no con el que la vista suponga (T-F015-04,
// SPEC-INTERFAZ §Zonas 2: "Cambia de comportamiento según el agente activo").
func (a *App) enviarCmd(sesionID, texto string, imagenes []string) tea.Cmd {
	return func() tea.Msg {
		if err := a.Puerto.Enviar(a.ctx, sesionID, a.Agente, texto, imagenes); err != nil {
			return errorMsg{err: err}
		}
		return enviadoMsg{Sesion: sesionID, Texto: texto}
	}
}

// actualizarEntrada entrega la pulsación al editor y refresca la paleta de
// comandos con lo que quedó escrito. Es el único punto por el que cambia el
// texto de la línea principal, así que la paleta nunca queda desincronizada.
//
// Un pegado o arrastre llega como una sola pulsación con `Paste`: sus rutas de
// imagen se convierten en tokens visibles ([foto.png]) antes de insertarlas, y
// la ruta real se recupera al enviar (Entrada.Texto).
func (a *App) actualizarEntrada(m tea.Msg) tea.Cmd {
	if k, ok := m.(tea.KeyMsg); ok && k.Paste {
		k.Runes = []rune(a.Entrada.AnotarPegado(string(k.Runes)))
		m = k
	}
	_, cmd := a.Entrada.Update(m)
	a.Paleta.Filtrar(a.Entrada.Texto())
	return cmd
}

// comandoAplicable decide qué comando, si alguno, corresponde a la línea. Con
// la paleta desplegada manda el comando resaltado —Enter y Tab actúan sobre
// él—; sin paleta, se reconoce el texto escrito.
func (a *App) comandoAplicable(texto string) (ComandoFlujo, bool) {
	if a.Paleta.Abierto {
		if c, ok := a.Paleta.Seleccionado(); ok {
			return c, true
		}
	}
	return ComandoFlujoDe(a.Comandos, texto)
}

// autocompletarComando deja en la línea el comando resaltado seguido de un
// espacio, listo para escribir la petición detrás (`/comando [petición]`). La
// paleta se retira sola: con el espacio ya no hay comandos que ofrecer.
func (a *App) autocompletarComando() tea.Cmd {
	c, ok := a.Paleta.Seleccionado()
	if !ok {
		return nil
	}
	a.Entrada.FijarTexto(c.Nombre + " ")
	a.Paleta.Filtrar(a.Entrada.Texto())
	return nil
}

// ejecutarComando arranca el flujo del comando. El comando va por el mismo
// camino que una petición —el puerto lo reconoce y el motor lo ejecuta
// (SPEC-MOTOR-FLUJOS: "un flujo no arranca solo: lo solicita el usuario con un
// comando explícito")—, así que la vista no corre etapas: el eco del comando y
// las etapas se ven en el chat, y las aprobaciones llegan por eventos.
//
// Desde la bienvenida no hay sesión todavía: se crea una, como con cualquier
// primera petición (SPEC-SESIONES). El modelo elegido en el modal viaja con el
// flujo, igual que con un mensaje de chat.
func (a *App) ejecutarComando(c ComandoFlujo, escrito string) tea.Cmd {
	texto := strings.TrimSpace(lineaDeComando(c, escrito))
	if texto == "" {
		return nil
	}
	id := a.Panel.SesionID
	if id == "" {
		ses, err := a.Puerto.Crear()
		if err != nil {
			a.Chat.AñadirSistema("no se pudo crear la sesión: " + err.Error())
			return nil
		}
		a.activar(ses)
		id = ses.ID
	}
	a.Entrada.Limpiar()
	a.Bienvenida.Limpiar()
	a.Paleta.Filtrar("")
	a.fijarModeloEnPuerto()
	// Las etapas de un flujo van sin imágenes: si el usuario adjuntó alguna, se
	// avisa en vez de perderla en silencio.
	if imagenes, _ := AdjuntosDe(texto); len(imagenes) > 0 {
		a.Chat.AñadirSistema("las imágenes no viajan con un comando de flujo; adjúntalas en una petición normal")
	}
	a.Chat.AñadirUsuario(texto)
	a.iniciarTurno()
	a.Vista = VistaPrincipal
	return a.enviarCmd(id, texto, nil)
}

// ciclarAgente avanza al siguiente agente de la lista disponible y lo propaga
// a lo único que lo pinta y a lo único que lo consume: el indicador de las dos
// vistas y la fila del panel. No toca el motor: qué hace cada agente lo decide
// él (DOMAIN §2, el input no valida reglas de negocio). La lista la ofrece el
// puerto (los agentes de `.localcli/agents/*.json`).
func (a *App) ciclarAgente() {
	a.Agente = siguienteAgente(a.Agentes, a.Agente)
	a.Panel.Agente = a.Agente
	a.guardarPreferencias()
}

// siguienteAgente devuelve el siguiente de la lista en ciclo. Si el activo no
// está en la lista —o la lista está vacía— devuelve el primero; sin lista,
// `plan`.
func siguienteAgente(agentes []string, actual string) string {
	if len(agentes) == 0 {
		return AgentePlan
	}
	for i, n := range agentes {
		if n == actual {
			return agentes[(i+1)%len(agentes)]
		}
	}
	return agentes[0]
}

// guardarPreferencias recuerda el último modelo y el último agente en
// ~/.config/localcli/config.json, para que la próxima ejecución arranque con
// ellos (SPEC-OLLAMA-PERFIL). Es best-effort: una preferencia que no se puede
// guardar no interrumpe nada. Se conservan los campos que la vista no gestiona
// (p. ej. el presupuesto de historial) releyendo lo que ya había.
func (a *App) guardarPreferencias() {
	previas, _ := CargarPreferencias()
	previas.Modelo = a.Modelo
	previas.Agente = a.Agente
	previas.Pensar = a.Pensar
	_ = GuardarPreferencias(previas)
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
	// El razonamiento revelado es una preferencia de la vista, no de la sesión:
	// al cambiar de sesión se conserva el `Ctrl+R` y solo se vacía lo acumulado.
	razon := NuevoRazonamiento()
	razon.Visible = a.Razon.Visible
	a.Razon = razon
	a.Chat.MostrarRazonamiento = a.Razon.Visible
	a.Panel.Tareas = nil
	// Los números del contexto son de la sesión cargada: se dejan a cero hasta
	// que llegue su historial, para no seguir mostrando los de la anterior
	// (SPEC-PANEL-CONTEXTO).
	a.Panel.ContextoTokens = 0
	a.Panel.LimiteTokens = 0
	a.Panel.LimiteDelHarness = false
	a.Panel.TokensEstimados = false
	// El turno que se seguía era de la sesión anterior. Si la destino ya está
	// trabajando, su turno se retoma: `enTurno` queda encendido para que su
	// cierre se procese, y el reloj se arranca al cargar su historial.
	a.enTurno = ses.Estado == session.EstadoTrabajando
	a.PidiendoCancelarEsc = false
	a.iniciarTurno()
}

// cmdHistorial pide la conversación y la lista de pasos de la sesión activa. Las
// dos lecturas llegan en un solo mensaje: la vista no consulta la base
// (INTERFACES §3).
func (a *App) cmdHistorial(sesionID string) tea.Cmd {
	return func() tea.Msg {
		h, err := a.Puerto.Historial(sesionID)
		if err != nil {
			return errorMsg{err: err}
		}
		// La lista de pasos es secundaria: si falla su lectura, el chat se pinta
		// igual y el panel queda sin checklist.
		ts, _ := a.Puerto.Tareas(sesionID)
		return historialMsg{
			Sesion:         sesionID,
			Mensajes:       h.Mensajes,
			Tareas:         ts,
			ContextoTokens: h.ContextoTokens,
			LimiteTokens:   h.LimiteTokens,
		}
	}
}

// contextoModal da el contexto de resolución del modal abierto. Los cuatro
// comparten navegación, pero el de sesiones y el de motores añaden teclas
// propias con literal compartido (`ctrl+d`), y cada una resuelve solo dentro de
// su modal (SPEC-KEYBINDS §Reglas de negocio).
func (a *App) contextoModal() Contexto {
	switch {
	case a.Motores.Abierto:
		return ContextoModalMotores
	case a.Sesiones.Abierto:
		return ContextoModalSesiones
	}
	return ContextoModal
}

// modalAbierto dice si hay un modal en pantalla. Los cuatro comparten la misma
// mecánica y solo uno puede estar abierto a la vez (SPEC-INTERFAZ §Modales:
// "uno abierto a la vez").
func (a *App) modalAbierto() bool {
	return a.Modelos.Abierto || a.Sesiones.Abierto || a.AtajosModal.Abierto || a.Motores.Abierto
}

// cerrarModales cierra los cuatro. Es lo que hace `dismiss` (Esc) y también lo
// que hacen las aperturas: abrir un modal es siempre abrirlo en lugar del que
// hubiera (SPEC-INTERFAZ §Modales). Un formulario o una confirmación a medias se
// descartan con él.
func (a *App) cerrarModales() {
	a.Modelos.Cerrar()
	a.Sesiones.Cerrar()
	a.AtajosModal.Cerrar()
	a.Motores.CancelarFormulario()
	a.Motores.BorrarCancelar()
	a.Motores.Cerrar()
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
	a.AtajosModal.AbrirAtajos(a.Atajos, a.liderVigente())
	return nil
}

// liderVigente devuelve la tecla líder del mapa en uso, para que el modal de
// atajos expanda `<leader>` con la combinación real (SPEC-INTERFAZ §Modales).
// Sin mapa —o con uno incompleto— cae a la líder de fábrica: nunca se pinta un
// literal a medias.
func (a *App) liderVigente() string {
	if a.Mapa != nil && strings.TrimSpace(a.Mapa.Lider()) != "" {
		return a.Mapa.Lider()
	}
	return LíderPorDefecto
}

// abrirModalModelos muestra el modal y pide la lista a Ollama en el acto
// (SPEC-INTERFAZ §Modal de modelos: "La lista se pide a Ollama al abrir el
// modal (no en el arranque)"). El modal aparece ya, con «cargando…», y se
// rellena cuando llega la respuesta: la pantalla no se bloquea mientras tanto.
func (a *App) abrirModalModelos() tea.Cmd {
	a.cerrarModales()
	a.Modelos.AbrirModelos(a.Modelo)
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

// abrirModalMotores muestra el modal de motores y pide el registro y las
// sesiones al puerto en el acto (SPEC-MODELO-MOTOR §Gestión de motores). El
// modal aparece con «cargando…» y se rellena al llegar la respuesta: la pantalla
// no se bloquea. El resaltado arranca en el motor de la sesión activa.
func (a *App) abrirModalMotores() tea.Cmd {
	a.cerrarModales()
	a.Motores.AbrirMotores(a.MotorIDSesion)
	return a.cargarMotores()
}

// cargarMotores pregunta al puerto el registro (y las sesiones, para saber qué
// motor usa alguna). La respuesta llega como `motoresMsg` y la atiende el modal.
func (a *App) cargarMotores() tea.Cmd {
	return func() tea.Msg {
		motores, err := a.Puerto.ListarMotores()
		if err != nil {
			return motoresMsg{Err: err}
		}
		sesiones, _ := a.Puerto.Listar()
		return motoresMsg{Motores: motores, Sesiones: sesiones}
	}
}

// aplicarMotor entrega al puerto el motor resaltado como motor de la sesión
// activa y cierra. No toca la identidad ni el historial de la sesión
// (SPEC-MODELO-MOTOR §Cambiar a mitad de conversación). Si el modal no tiene
// lista —«sin motores»— solo se cierra.
func (a *App) aplicarMotor() tea.Cmd {
	m, ok := a.Motores.MotorElegido()
	if !ok {
		a.Motores.Cerrar()
		return nil
	}
	sesion := a.Panel.SesionID
	a.Motores.Cerrar()
	if sesion == "" {
		return nil
	}
	return func() tea.Msg {
		if err := a.Puerto.CambiarMotor(sesion, m.ID); err != nil {
			return errorMsg{err: err}
		}
		modelos, _ := a.Puerto.Modelos()
		return motorAplicadoMsg{Sesion: sesion, MotorID: m.ID, Modelos: modelos}
	}
}

// guardarMotor persiste el alta o la edición del formulario. La validación la
// hizo el modal: si el motor es nuevo se registra, y si es una edición se
// actualiza y, solo si el estado cambió, se desactiva o reactiva al momento
// (SPEC-MODELO-MOTOR §Editar: el cambio de configuración espera al reinicio; el
// estado activo sí es inmediato).
func (a *App) guardarMotor() tea.Cmd {
	motor, ok := a.Motores.GuardarFormulario()
	if !ok {
		return nil
	}
	cambioActivo := a.Motores.ActivoCambio()
	if motor.ID == "" {
		if err := a.Puerto.RegistrarMotor(motor); err != nil {
			a.Chat.AñadirSistema("no se pudo registrar el motor: " + err.Error())
			return nil
		}
	} else {
		if err := a.Puerto.EditarMotor(motor); err != nil {
			a.Chat.AñadirSistema("no se pudo editar el motor: " + err.Error())
			return nil
		}
		if cambioActivo {
			if err := a.Puerto.AlternarMotor(motor.ID, motor.Activo); err != nil {
				a.Chat.AñadirSistema("no se pudo cambiar el estado del motor: " + err.Error())
				return nil
			}
		}
	}
	return a.cargarMotores()
}

// eliminarMotorResaltado pide borrar el motor resaltado. Si alguna sesión lo usa,
// el modal muestra la confirmación y el borrado espera a la respuesta; si no,
// se elimina al momento (SPEC-MODELO-MOTOR §Eliminar).
func (a *App) eliminarMotorResaltado() tea.Cmd {
	id, enUso := a.Motores.PedirBorrado()
	if id == "" || enUso {
		return nil
	}
	return a.eliminarMotor(id)
}

// eliminarMotor borra el motor y recarga la lista. No elimina ni invalida
// ninguna sesión: las que lo usaban siguen con su historial y avisan.
func (a *App) eliminarMotor(id string) tea.Cmd {
	if err := a.Puerto.EliminarMotor(id); err != nil {
		a.Chat.AñadirSistema("no se pudo eliminar el motor: " + err.Error())
		return nil
	}
	a.Motores.BorrarCancelar()
	return tea.Batch(a.cargarMotores(), a.cmdMotorDeSesion(a.Panel.SesionID))
}

// cmdMotorDeSesion resuelve el motor de la sesión activa a lo que pinta la
// línea de estado: su nombre visible y el aviso de que falta o está desactivado,
// sin bloquear nada (SPEC-MODELO-MOTOR §Desactivar, §Eliminar).
func (a *App) cmdMotorDeSesion(sesionID string) tea.Cmd {
	if sesionID == "" {
		return nil
	}
	return func() tea.Msg {
		motorID, _, err := a.Puerto.ParMotorModelo(sesionID)
		if err != nil || motorID == "" {
			return motorSesionMsg{Sesion: sesionID}
		}
		motores, err := a.Puerto.ListarMotores()
		if err != nil {
			return motorSesionMsg{Sesion: sesionID, ID: motorID}
		}
		for _, mt := range motores {
			if mt.ID != motorID {
				continue
			}
			aviso := ""
			if !mt.Activo {
				aviso = "el motor «" + mt.Nombre + "» está desactivado; reactívalo con Ctrl+X i"
			}
			return motorSesionMsg{Sesion: sesionID, ID: motorID, Nombre: mt.Nombre, Aviso: aviso}
		}
		return motorSesionMsg{Sesion: sesionID, ID: motorID, Nombre: motorID,
			Aviso: "el motor de esta sesión ya no existe; elige otro con Ctrl+X i"}
	}
}

// teclaFormMotor entrega la pulsación al formulario de alta/edición mientras
// está abierto: sus campos reciben el texto y el resto de teclas no resuelven
// acciones de la app (SPEC-KEYBINDS: con un modal abierto, la vista de abajo no
// recibe teclas).
func (a *App) teclaFormMotor(m tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.Type == tea.KeyCtrlC {
		return a, tea.Quit
	}
	switch m.Type {
	case tea.KeyEsc:
		a.Motores.CancelarFormulario()
	case tea.KeyEnter:
		return a.despachar(AccionEnviar, m)
	case tea.KeyTab, tea.KeyDown:
		a.Motores.FormMoverCampo(1)
	case tea.KeyUp:
		a.Motores.FormMoverCampo(-1)
	case tea.KeyBackspace:
		a.Motores.FormBorrar()
	case tea.KeyRunes:
		a.Motores.FormEscribir(string(m.Runes))
	case tea.KeySpace:
		a.Motores.FormEscribir(" ")
	}
	return a, nil
}

// teclaConfirmarBorrarMotor atiende la confirmación de borrado de un motor en
// uso: `s` confirma, `n` y `esc` cancelan. Ninguna otra tecla pasa a la vista.
func (a *App) teclaConfirmarBorrarMotor(m tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.Type == tea.KeyCtrlC {
		return a, tea.Quit
	}
	switch m.Type {
	case tea.KeyRunes:
		switch string(m.Runes) {
		case "s", "S":
			return a, a.eliminarMotor(a.Motores.BorrandoID())
		case "n", "N":
			a.Motores.BorrarCancelar()
		}
	case tea.KeyEsc:
		a.Motores.BorrarCancelar()
	}
	return a, nil
}

// aplicarModelo entrega al motor el modelo resaltado y lo deja como modelo en
// uso: la línea de modelo de la bienvenida lo refleja y la primera petición
// sale con él (SPEC-INTERFAZ §Modal de modelos, DOMAIN §3). El modal no decide
// nada: entrega la elección y se cierra (DOMAIN §1 `modals`). Sin lista —«sin
// modelos» o «cargando…»— no hay nada que aplicar: solo se cierra.
func (a *App) aplicarModelo() tea.Cmd {
	m, ok := a.Modelos.ModeloElegidoLocal()
	a.Modelos.Cerrar()
	if !ok || m.Nombre == "" {
		return nil
	}
	a.Modelo = m.Nombre
	a.fijarModeloEnPuerto()
	// La línea de estado bajo el input ya sabe si el modelo tiene herramientas:
	// lo eligió el usuario desde la lista, que trae esa marca. Si la ficha no se
	// pudo leer, la capacidad queda como desconocida y no se avisa de más.
	a.CapHerramientasConocida = !m.CapacidadesSinDato
	a.ModeloHerramientas = !m.SinHerramientas
	a.CapVisionConocida = !m.CapacidadesSinDato
	a.ModeloVision = !m.SinVision
	// El modelo elegido que no declara herramientas avisa sin bloquear: el
	// usuario decide si cambia (SPEC-OLLAMA-PERFIL). Al elegir uno capaz, el
	// aviso se retira.
	if !m.CapacidadesSinDato && m.SinHerramientas {
		a.Aviso = "el modelo «" + m.Nombre + "» no puede usar herramientas; cámbialo con Ctrl+X m"
	} else {
		a.Aviso = ""
	}
	a.guardarPreferencias()
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
	return a.decidirAprobacion(ap, aprobar)
}

// decidirAprobacion aplica la decisión sobre una aprobación concreta: la que
// resuelve el teclado (a/d) o la que se pulsa con el ratón sobre «aprobar» o
// «declinar» (selection.go). Una obsoleta solo se retira: no se manda a su
// sesión, que ya terminó.
func (a *App) decidirAprobacion(ap Aprobacion, aprobar bool) tea.Cmd {
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

// View pinta la pantalla y recuerda el marco: de él se extrae el texto de una
// selección con el ratón (selection.go). Si hay una selección (el ancla y el
// puntero difieren, solo durante el arrastre), se devuelve el marco con su
// tramo resaltado en video inverso; `ultimaVista` se guarda en crudo para que la
// extracción del texto que se copia no dependa del realce. Encima va el aviso
// transitorio [Copiado] cuando toca.
func (a *App) View() string {
	// La capa común (marco.go) convierte lo que pintan los componentes en una
	// superficie rectangular continua: exactamente Ancho×Alto celdas, cada una
	// con fondo. Se normaliza ANTES de guardar el marco, de modo que la
	// selección y la copia trabajen sobre celdas reales (incluidos los espacios
	// del padding y del fondo), no sobre líneas de ancho cero.
	v := marcoCompleto(a.view(), a.Ancho, a.Alto)
	a.ultimaVista = v
	out := v
	if a.ratonIni != a.ratonFin {
		ini, fin := a.limitarSeleccion(a.ratonIni, a.ratonFin)
		out = resaltarSeleccion(v, ini, fin)
	}
	if a.copiado {
		out = superponerDerecha(out, estiloCopiado.Render("[Copiado]"), a.Ancho)
	}
	return out
}

// view pinta la pantalla: un modal abierto si lo hay, y si no la bienvenida o la
// interfaz principal. Los tres modales se dibujan centrados sobre la ventana
// (SPEC-INTERFAZ §Modales: "Tres modales centrados").
func (a *App) view() string {
	// La banda del chat solo existe en la vista principal: en las demás se
	// neutraliza para que ningún extremo de una selección cuente como del chat.
	a.chatFilaIni, a.chatFilaFin = 0, 0
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
	if a.Motores.Abierto {
		return a.Motores.Render(a.Ancho, a.Alto)
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

// bloqueEnCurso pinta el intercambio del turno en vivo —razonamiento revelado y
// respuesta que va llegando— como un bloque más del chat: vive dentro de la
// ventana del historial, así crece sin empujar la entrada y se recorre con el
// scroll como cualquier mensaje.
func (a *App) bloqueEnCurso(ancho int) string {
	x := renderIntercambio(a.Razon.Texto(), a.Chat.EnCurso(), a.Razon.Visible, a.Razon.NoDisponible, ancho-4)
	if x == "" {
		return ""
	}
	return bloqueChat(RolAgente, x, ancho)
}

// bloqueInferior compone todo lo que va pegado al pie de la vista principal: el
// intercambio en curso, el contador, las propuestas, las aprobaciones, la línea
// de entrada, la línea de estado del modelo y los avisos. Se mide para
// reservarle su alto exacto y que el chat no empuje el marco fuera de pantalla.
func (a *App) bloqueInferior(anchoCol int) string {
	// Lo que se pinta como parte del hilo, arriba del divisor: las propuestas y
	// las aprobaciones pendientes. El intercambio en vivo va dentro del chat
	// (`Chat.VentanaCon`) y el indicador de actividad, justo encima de la caja.
	var cuerpo []string
	// Las propuestas pendientes de esta sesión se ven dentro del chat.
	if prop := a.Chat.RenderPropuestas(); prop != "" {
		cuerpo = append(cuerpo, prop)
	}
	if ap := a.Aprobs.Render(); ap != "" {
		cuerpo = append(cuerpo, ap)
	}

	var partes []string
	if arriba := strings.Join(cuerpo, "\n\n"); arriba != "" {
		partes = append(partes, arriba)
	}
	// La paleta de comandos se despliega encima de la caja (comandos.go).
	if pal := a.Paleta.Render(); pal != "" {
		partes = append(partes, pal)
	}
	// El indicador en vivo va pegado a la caja: debajo del divisor y justo encima
	// de la entrada. Mientras hay un turno en camino gira el glifo y corre el
	// tiempo —[⠋ Pensando], [⠋ Usando herramienta: X]—, que es lo que dice que el
	// modelo sigue trabajando.
	partes = append(partes, a.divisorHorizontal(anchoCol))
	if act := a.lineaDeActividad(); act != "" {
		partes = append(partes, act)
	}
	// La caja de la línea de entrada, con su pie (agente, modelo y capacidades)
	// y el indicador de líder pendiente (T-F012-06, T-F015-01).
	pie := a.lineaPieEntrada()
	if a.TeclaRes != nil && a.TeclaRes.EsperandoLeader() {
		pie += " " + estiloAviso.Render("lider ")
	}
	partes = append(partes, a.Entrada.Caja(anchoCol, pie))

	// Los avisos van debajo de la caja: el consumo de tokens del turno, cancelar
	// con ctrl+f (lo escrito queda a salvo), el doble esc para cancelar sin
	// escribir, el aviso transitorio (p. ej. el modelo sin herramientas) y el
	// aviso de aprobaciones pendientes con el panel cerrado (T-F009-03).
	var avisos []string
	if tok := a.lineaDeTokens(); tok != "" {
		avisos = append(avisos, tok)
	}
	if a.PidiendoCancelar {
		avisos = append(avisos, estiloAviso.Render("¿cancelar el trabajo en curso? (s/n)"))
	}
	if a.PidiendoCancelarEsc {
		avisos = append(avisos, estiloAviso.Render("presiona esc otra vez para cancelar razonamiento"))
	}
	if a.Aviso != "" {
		avisos = append(avisos, estiloAviso.Render(a.Aviso))
	}
	// El aviso de razonamiento desconocido solo aparece en la vista principal:
	// es un atributo del modelo en uso y afecta al turno en curso.
	if a.RazonAviso != "" {
		avisos = append(avisos, estiloAviso.Render(a.RazonAviso))
	}
	if !a.Panel.Abierto {
		if aviso := a.Panel.AvisoAprobaciones(); aviso != "" {
			avisos = append(avisos, aviso)
		}
	}
	if len(avisos) > 0 {
		partes = append(partes, strings.Join(avisos, "\n"))
	}
	return strings.Join(partes, "\n")
}

// divisorHorizontal separa el chat de la caja de entrada con el fondo de la
// pantalla: una fila de celdas sin glifos que, junto al borde superior de la
// caja, deja dos espacios de separación.
func (a *App) divisorHorizontal(ancho int) string {
	if ancho < 1 {
		ancho = 1
	}
	return pintarFondo("", ancho, fondoApp)
}

// lineaDeActividad pinta el indicador en vivo mientras hay un turno en camino:
// un glifo que gira y la etiqueta de lo que está pasando —«Usando herramienta:
// X» si corre una herramienta, «Generando» si ya llega respuesta, «Pensando» si
// el modelo aún no ha emitido nada—, más el tiempo transcurrido. Solo se pinta
// mientras hay algo en marcha (enActividad): no sobrevive al turno.
func (a *App) lineaDeActividad() string {
	if !a.enActividad() {
		return ""
	}
	etiqueta := "Pensando"
	switch {
	case a.Aprobs.Pendientes() > 0:
		// Hay una decisión esperando: el turno no avanza hasta que se resuelva,
		// así que el indicador lo dice en vez de fingir que sigue pensando.
		etiqueta = "Esperando tu permiso"
	case a.herramientaEnCurso != "":
		etiqueta = "Usando herramienta: " + a.herramientaEnCurso
	case a.Chat.EnCurso() != "":
		etiqueta = "Generando"
	}
	return estiloActividad.Render(renderActividad(a.frame, etiqueta)) +
		" " + estiloSistema.Render(formatearDuracion(a.Chat.Transcurrido()))
}

// enActividad dice si hay algo en marcha que justifique el indicador: un turno
// vivo, texto en curso, razonamiento acumulado o una herramienta corriendo. Basta
// que haya llegado cualquier fragmento, aunque el turno no se haya marcado como
// vivo (p. ej. un evento aislado), para no dejar mudo el indicador.
func (a *App) enActividad() bool {
	return a.Chat.HayTurno() || a.Chat.EnCurso() != "" || a.Razon.Hay() || a.herramientaEnCurso != ""
}

// lineaDeTokens muestra el consumo de tokens del turno bajo el input, en la
// unidad que se lee de un vistazo (54000 → «54k»). Sin consumo no pinta nada:
// un «0 tokens» de adorno solo estorba.
func (a *App) lineaDeTokens() string {
	if a.Panel.Tokens <= 0 {
		return ""
	}
	return estiloSistema.Render("tokens: " + formatearTokens(a.Panel.Tokens))
}

// iniciarTurno deja a cero lo que se acumula por turno: el consumo de tokens y
// la herramienta en curso. El razonamiento y la respuesta en curso los limpia el
// cierre del turno anterior; esto es lo que se cuenta de nuevo al enviar. El
// contexto de la sesión (ContextoTokens) no se toca: es del chat, no del turno.
func (a *App) iniciarTurno() {
	a.Panel.Tokens = 0
	a.herramientaEnCurso = ""
}

// lineaPieEntrada compone el pie de la caja de entrada: el agente activo, el
// modelo en uso rotulado con su motor y sus capacidades. Sin modelo muestra solo
// el agente. Si el motor de la sesión falta o está desactivado, el aviso va junto
// al modelo sin bloquear nada (INTERFACES §5, T-F044-09).
func (a *App) lineaPieEntrada() string {
	piezas := []string{estiloIndicador.Render("[" + a.Agente + "]")}
	if a.Modelo != "" {
		piezas = append(piezas, estiloBlanco.Render("* "+a.rotuloModelo()), a.capacidadesPie())
	}
	if a.MotorAviso != "" {
		piezas = append(piezas, estiloAviso.Render(a.MotorAviso))
	}
	return strings.Join(piezas, " · ")
}

// modeloSoloDe quita el rótulo « (motor)» que `ModeloActual()` devuelve ya
// montado, para que la vista guarde el nombre del modelo solo y lo rotule con la
// etiqueta vigente. Si el motor declarado no es el sufijo, se devuelve tal cual:
// la vista no corta a ciegas.
func modeloSoloDe(rotulado, etiqueta string) string {
	if rotulado == "" || etiqueta == "" {
		return rotulado
	}
	sufijo := " (" + etiqueta + ")"
	if strings.HasSuffix(rotulado, sufijo) {
		return strings.TrimSuffix(rotulado, sufijo)
	}
	return rotulado
}

// rotuloModelo compone lo que se pinta junto al nombre del modelo: la instancia
// del motor que lo sirve, cuando se conoce (SPEC-INTERFAZ §Caja de entrada:
// `[plan] • qwen3:8b (llama.cpp)`). Sin modelo no hay nada que rotular.
func (a *App) rotuloModelo() string {
	if a.Modelo == "" {
		return "—"
	}
	if a.MotorEtiqueta == "" {
		return a.Modelo
	}
	return a.Modelo + " (" + a.MotorEtiqueta + ")"
}

// capacidadesPie compone las chapas del pie: `tool [*]` en verde si usa
// herramientas y en rojo si no, `tool [?]` mientras no se sabe, el interruptor de
// razonamiento si el modelo lo declara, `[v]` si acepta visión, `[v?]` si no se
// sabe y `[T]` siempre (texto). El `?` va junto al nombre de la capacidad y solo
// marca el estado `desconocida`: la `no soportada` no lleva marca, porque ahí sí
// hay dato (SPEC-MODELO-MOTOR §Capacidades).
func (a *App) capacidadesPie() string {
	var herramienta string
	switch {
	case !a.CapHerramientasConocida:
		herramienta = estiloSutil.Render("tool [?]")
	case a.ModeloHerramientas:
		herramienta = estiloBlanco.Render("tool ") + estiloCapaz.Render("[*]")
	default:
		herramienta = estiloBlanco.Render("tool ") + estiloIncapaz.Render("[*]")
	}
	piezas := []string{herramienta}
	if chapa := a.chapaRazonamiento(); chapa != "" {
		piezas = append(piezas, chapa)
	}
	entradas := make([]string, 0, 2)
	switch {
	case !a.CapVisionConocida:
		entradas = append(entradas, estiloSutil.Render("[v?]"))
	case a.ModeloVision:
		entradas = append(entradas, estiloCapaz.Render("[v]"))
	}
	entradas = append(entradas, estiloCapaz.Render("[T]"))
	piezas = append(piezas, strings.Join(entradas, " "))
	return strings.Join(piezas, "  ")
}

// chapaRazonamiento compone el interruptor de razonamiento: `pensar [x]`
// encendido y `pensar [ ]` apagado. Solo se enseña si el modelo declara que
// razona —a los demás no se les puede mandar `think`— y está apagado por
// defecto: en un modelo local, razonar cuesta minutos hasta para lo trivial
// (SPEC-OLLAMA-PERFIL). Se pulsa con el ratón (selection.go, `toggleEnCelda`).
func (a *App) chapaRazonamiento() string {
	if !a.CapPensarConocida || !a.ModeloPensar {
		return ""
	}
	if a.Pensar {
		return estiloBlanco.Render("pensar ") + estiloCapaz.Render("[x]")
	}
	return estiloSutil.Render("pensar [ ]")
}

// alternarRazonamiento cambia el interruptor del pie. Vale para los turnos
// siguientes —el que esté corriendo ya salió con lo que decidió— y se recuerda
// entre ejecuciones, como el modelo y el agente.
func (a *App) alternarRazonamiento() {
	a.Pensar = !a.Pensar
	if a.Puerto != nil {
		a.Puerto.Pensar(a.Pensar)
	}
	a.guardarPreferencias()
}

// viewPrincipal pinta chat, razonamiento, respuesta en curso, aprobaciones y
// avisos; y el panel a la derecha cuando está abierto. El alto del pie se mide
// para que el marco entero quepa en la terminal: si el chat empujara la vista
// más allá del alto disponible, al pintar se recortaría la parte de arriba.
func (a *App) viewPrincipal() string {
	anchoCol := a.anchoColumna()
	abajo := a.bloqueInferior(anchoCol)

	construir := func(alto int) string {
		var lineas []string
		if h := a.Chat.VentanaCon(anchoCol, alto, a.bloqueEnCurso(anchoCol)); h != "" {
			if oa := a.Chat.OcultasArriba(); oa > 0 {
				lineas = append(lineas, estiloSistema.Render(fmt.Sprintf("↑ %d líneas arriba", oa)))
			}
			lineas = append(lineas, strings.Split(h, "\n")...)
			if ob := a.Chat.OcultasAbajo(); ob > 0 {
				lineas = append(lineas, estiloSistema.Render(fmt.Sprintf("↓ %d líneas abajo", ob)))
			}
		}
		// El chat ocupa todo el alto que quede hasta el pie: con un historial
		// corto se rellena de líneas en blanco, de modo que la caja de entrada
		// queda pegada abajo en vez de flotar en medio de la pantalla. Después va
		// el bloque inferior, sin salto final, para que el marco tenga exactamente
		// `Alto` filas.
		objetivo := a.Alto - altoDe(abajo)
		for len(lineas) < objetivo {
			lineas = append(lineas, "")
		}
		lineas = append(lineas, strings.Split(abajo, "\n")...)
		return strings.Join(lineas, "\n")
	}

	// Alto para el chat: lo que queda tras el pie y hasta dos líneas de
	// indicadores de scroll. Reservar los indicadores siempre (aunque no se
	// pinten) mantiene el alto del chat estable: si cambiara al subir, la
	// posición del scroll saltaría.
	altoChat := a.Alto - altoDe(abajo) - 2
	if altoChat < 3 {
		altoChat = 3
	}
	cuerpo := construir(altoChat)

	// La banda del chat del marco recién compuesto, para que la rueda pueda
	// reanclar la selección (selection.go). La línea «↑ N líneas arriba», si
	// está, ocupa la primera fila.
	a.chatFilaIni = cabecera(a.Chat.OcultasArriba())
	a.chatFilaFin = a.chatFilaIni + altoChat

	if !a.Panel.Abierto {
		return cuerpo
	}
	panel := a.Panel.Render(a.anchoPanel(), a.Alto)
	return a.componerColumnas(cuerpo, anchoCol, a.anchoPanel(), panel)
}

// componerColumnas une la columna principal con el sidebar dejando una banda de
// fondo como separación (una columna de celdas con un tono distinto, sin el
// glifo `│`). Rellena cada columna a su ancho y con su fondo, de modo que la
// separación quede recta y el marco siga siendo UNA sola cadena: la selección
// con el ratón lo recorre entero (chat y sidebar a la vez) sin arrastrar el
// carácter del divisor.
func (a *App) componerColumnas(izquierda string, anchoIzq, anchoDer int, derecha string) string {
	alto := a.Alto
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
	// Dos columnas de fondo separan el chat del sidebar: la separación es la de
	// la pantalla, no una línea de otro color.
	banda := pintarFondo("", 2, fondoApp)
	lineas := make([]string, alto)
	for i := 0; i < alto; i++ {
		l, r := "", ""
		if i < len(izq) {
			l = izq[i]
		}
		if i < len(der) {
			r = der[i]
		}
		lineas[i] = pintarFondo(recortarColumnas(l, anchoIzq), anchoIzq, fondoApp) + banda +
			pintarFondo(recortarColumnas(r, anchoDer), anchoDer, fondoPanel)
	}
	return strings.Join(lineas, "\n")
}

// lineasDe cuenta las líneas de un bloque (0 si está vacío).
func lineasDe(texto string) int {
	if texto == "" {
		return 0
	}
	return strings.Count(texto, "\n") + 1
}

// altoDe cuenta las líneas que ocupa un bloque ya compuesto; los saltos finales
// no cuentan, porque el cursor se queda en la última línea con contenido.
func altoDe(texto string) int { return lineasDe(strings.TrimRight(texto, "\n")) }

// trabajando dice si la sesión activa tiene un turno en curso, que es cuando
// tiene sentido cancelar con el doble esc.
func (a *App) trabajando() bool {
	return a.Chat.HayTurno() || a.Panel.Estado == session.EstadoTrabajando
}

// anchoColMin es el ancho que la columna principal conserva siempre: si la
// terminal es estrecha, quien cede es el sidebar, nunca el chat.
const anchoColMin = 20

// anchoPanel es el ancho del sidebar cuando está abierto, recortado si la
// terminal no da para el ancho fijo más la columna principal mínima.
func (a *App) anchoPanel() int {
	if a.Ancho <= 0 {
		return AnchoPanel
	}
	disponible := a.Ancho - anchoColMin - 2
	if disponible < 1 {
		disponible = 1
	}
	return min(AnchoPanel, disponible)
}

// anchoColumna es el ancho disponible para la columna principal (el chat y su
// caja de entrada): el total menos el sidebar y las dos columnas de separación
// cuando el panel está abierto.
func (a *App) anchoColumna() int {
	if a.Ancho <= 0 {
		return 80
	}
	if a.Panel.Abierto {
		return a.Ancho - a.anchoPanel() - 2
	}
	return a.Ancho
}
