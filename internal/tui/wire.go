// wire.go — T-B014-08: la unión con `session` por canales.
//
// Fuente de verdad: BACKEND.md (la cadena `tui → session`), DOMAIN.md §3 ("El
// límite del módulo tui": no decide nada, solo pinta lo que llega y manda lo que
// se pulsa) y EVENTS.md §2 y §4 ("La TUI no pregunta nada: se le notifica", "los
// eventos son notificaciones, no comandos").
//
// La vista no importa `flow`, `tools` ni `store`: lo que necesita del motor está
// detrás de `Puerto`, y lo que le llega, por eventos. El adaptador de producción
// vive en el arranque, que es el único sitio que conoce todas las piezas a la
// vez. El efecto práctico: un test de la vista no necesita base de datos, ni
// modelo, ni flujo.
package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"localcli/internal/session"
)

// Evento es el evento del motor tal como lo entrega `session`.
type Evento = session.Evento

// Nombres de eventos que la vista consume (EVENTS.md §1). Los que emite
// `session` se toman de su paquete, para no tener dos literales del mismo evento.
const (
	EventoToken              = "token"
	EventoPeticionAprobacion = "peticion_aprobacion"
	EventoAprobacionResuelta = "aprobacion_resuelta"
	EventoCambioAplicado     = "cambio_aplicado"
	EventoColaActualizada    = "cola_actualizada"
	EventoElementoBloqueado  = "elemento_bloqueado"
	EventoTituloSesion       = "titulo_sesion"
	EventoEtapaIniciada      = "etapa_iniciada"
	EventoEtapaTerminada     = "etapa_terminada"
	EventoEtapaFallida       = "etapa_fallida"
	EventoFlujoPausado       = "flujo_pausado"
	EventoFlujoReanudado     = "flujo_reanudado"
	EventoFlujoCancelado     = "flujo_cancelado"
	// Eventos de herramienta (EVENTS.md §3). Los emite `tools` desde la capa
	// universal; la vista los pinta sin importar el módulo y sin volcar la
	// salida cruda: basta el nombre, el agente y el estado.
	EventoHerramientaInvocada  = "herramienta_invocada"
	EventoHerramientaResultado = "herramienta_resultado"
	// EventoTokensTurno lleva el consumo del turno para el panel de datos.
	EventoTokensTurno = "tokens_turno"
	// EventoTodoActualizada lleva la lista de pasos de la sesión (SPEC-TOOLS):
	// el panel la repinta sin preguntar nada.
	EventoTodoActualizada = "todo_actualizada"
)

// Agentes base que conoce la vista de fábrica. El puerto ofrece la lista real
// de agentes disponibles (`Agentes`, cargada de `.localcli/agents/*.json`); si no
// ofrece ninguna, la vista cae en estos dos. La vista no puede importar `tools`
// (tests/arquitectura_test.go §TestLimitesDeImporteEntreModulos), así que los
// recibe como texto.
const (
	AgentePlan  = "plan"
	AgenteBuild = "build"
)

// ModeloLocal es una entrada del selector de modelos de la bienvenida
// (SPEC-INTERFAZ §Reglas, "Selector de modelos"). La vista no importa el
// paquete `ollama`: lo que reporta Ollama llega traducido a este dato mínimo,
// igual que las sesiones llegan como `session.Sesion` y nada más.
type ModeloLocal struct {
	Nombre string
	// SinHerramientas marca los modelos que Ollama no declara capaces de usar
	// herramientas. El valor cero es «sí puede»: un fallo de detección no debe
	// alarmar. La vista solo lo pinta y avisa; no decide nada (DOMAIN §4).
	SinHerramientas bool
	// SinVision marca los modelos que Ollama no declara capaces de interpretar
	// imágenes. Mismo criterio: el valor cero es «sí puede» y un fallo no alarma.
	SinVision bool
	// CapacidadesSinDato marca las fichas que no se pudieron leer: no se sabe si
	// el modelo puede o no, así que la vista ni avisa ni asegura nada. El valor
	// cero es «sí se sabe».
	CapacidadesSinDato bool
}

// Capacidades es lo que la vista conoce del modelo en uso para su línea de
// estado. Los tres campos son «sí puede»: la vista solo los pinta y avisa.
type Capacidades struct {
	Herramientas bool
	Vision       bool
	// Pensar dice si el modelo razona antes de responder. Solo estos modelos
	// enseñan la chapa `pensar [x]`: a los demás no se les puede mandar `think`.
	Pensar bool
}

// TareaPanel es un paso de la lista de la sesión tal como lo pinta el panel
// (SPEC-TOOLS). Lleva etiqueta json porque el evento lo entrega serializado y la
// vista lo decodifica sin importar el motor.
type TareaPanel struct {
	Contenido string `json:"contenido"`
	Estado    string `json:"estado"`
}

// Puerto es lo que la vista necesita de `session`. Todo lo que no esté aquí, la
// vista no lo puede hacer.
type Puerto interface {
	// ResolverActiva retoma la sesión más reciente del proyecto para volver a
	// ella (por ejemplo tras borrar la activa). NO crea ninguna: devuelve nil
	// sin error si el proyecto no tiene sesiones, y entonces la bienvenida crea
	// una con la primera petición (SPEC-SESIONES).
	ResolverActiva() (*session.Sesion, error)
	// Modelos lista los modelos locales que reporta Ollama para el modal de
	// modelos (SPEC-INTERFAZ §Modal de modelos). Se pide al abrir el modal, no
	// al arrancar: si Ollama no responde, el modal avisa «sin modelos» y nada
	// se bloquea — nunca espera (DOMAIN §3).
	Modelos() ([]ModeloLocal, error)
	// ModeloActual devuelve el modelo con el que trabaja el motor ahora mismo: el
	// detectado en el arranque o el último elegido en el modal. Es la línea de
	// modelo de la bienvenida (SPEC-INTERFAZ §Línea de modelo), y lo lee la
	// pantalla una vez al construirse, sin llamar a Ollama.
	ModeloActual() string
	// Carpeta devuelve la carpeta del proyecto —la desde la que se ejecutó la
	// herramienta—, que el arranque ya resolvió al abrir la base. Es el dato de
	// «Ruta» del panel (SPEC-INTERFAZ §Zonas 3): lo lee la pantalla una vez al
	// construirse, porque no cambia mientras la sesión vive. No va por la base:
	// es un dato en memoria del arranque, como el modelo o el agente.
	Carpeta() string
	// Git devuelve el estado del repositorio del proyecto: la rama activa y si
	// el árbol tiene cambios sin confirmar. La rama vacía es el caso de «no hay
	// git aquí» —el proyecto sin inicializar, o sin git instalado—, y la vista
	// lo pinta como «sin iniciar» (SPEC-INTERFAZ §Zonas 3, dato «Git»). Es
	// lectura del arranque, como la carpeta: la pantalla no llama a git ni
	// calcula nada con lo que llega.
	Git() (rama string, cambios int)
	// CapacidadesModelo dice qué declara capaz de hacer el modelo indicado, para
	// la línea de estado bajo el input (SPEC-OLLAMA-PERFIL). La vista no importa
	// `ollama`: los booleanos llegan ya resueltos.
	CapacidadesModelo(nombre string) (Capacidades, error)
	// AgenteRecordado devuelve el último agente con el que se trabajó, para que
	// la vista arranque en él (SPEC-OLLAMA-PERFIL: la preferencia se recuerda).
	// Vacío significa «sin preferencia»: la vista cae en `plan`. Lo lee el motor,
	// que es quien conoce el archivo de preferencias del usuario.
	AgenteRecordado() string
	// Agentes devuelve los nombres de los agentes disponibles, en orden estable
	// (los base primero y el resto alfabético). Los carga el arranque de
	// `.localcli/agents/*.json`: el usuario añade un agente dejando su JSON ahí, sin
	// tocar el código. La vista cicla por esta lista con `Tab`.
	Agentes() []string
	// PensarRecordado es el estado con el que arranca el interruptor de
	// razonamiento del pie: la última elección del usuario, o apagado. Sin él,
	// razonar costaría minutos en cada turno de un modelo local
	// (SPEC-OLLAMA-PERFIL).
	PensarRecordado() bool
	// Pensar aplica el interruptor: a partir de aquí los turnos mandan `think`
	// con ese valor, si el modelo declara que razona.
	Pensar(v bool)
	// Comandos devuelve los comandos de flujo disponibles en el proyecto: los
	// oficiales más los que declara `.localcli/flows/*.json`
	// ([[specs/SPEC-FLUJO-PERSONALIZADO]]). La vista los lista en la paleta y
	// reconoce los que se escriben; no decide cuáles existen. Es un dato en
	// memoria —el catálogo cargado al arrancar—, así que la bienvenida lo puede
	// leer sin esperar a nada externo.
	Comandos() []ComandoFlujo
	// FijarModelo elige el modelo con el que trabajará el motor a partir de
	// ahora; lo elegido por el usuario prevalece sobre la autodetección del
	// arranque (SPEC-OLLAMA-PERFIL: el modelo lo elige el usuario). Se llama al
	// aplicar en el modal y otra vez al enviar desde la bienvenida, para que la
	// primera petición salga con ese modelo.
	FijarModelo(nombre string)
	// Listar lista las sesiones del proyecto para el modal de sesiones.
	Listar() ([]session.Sesion, error)
	// Crear deja una sesión nueva activa (SPEC-SESIONES). Es lo que hace
	// `Ctrl+X n` en la vista principal (SPEC-KEYBINDS `session_new`).
	Crear() (*session.Sesion, error)
	// Eliminar borra una sesión y todo lo suyo en cascada; lo usa el modal de
	// sesiones con `Ctrl+D` (SPEC-SESIONES). Si está trabajando, la vista
	// pide confirmación antes de llamar aquí.
	Eliminar(sesionID string) error
	// Historial devuelve la conversación de una sesión —turnos y líneas de
	// procesamiento, con su razonamiento y su tiempo— más los números del panel
	// de contexto, para pintarlos al cambiar de sesión (INTERFACES §3). Es una
	// lectura que llega como dato; la vista nunca consulta la base.
	Historial(sesionID string) (HistorialSesion, error)
	// Tareas devuelve la lista de pasos de una sesión, para pintarla al cargarla
	// (SPEC-TOOLS). Igual que el historial, es una lectura que llega como dato;
	// la vista nunca consulta la base.
	Tareas(sesionID string) ([]TareaPanel, error)
	// Enviar manda un mensaje a una sesión indicando con qué agente se pide: el
	// que el usuario tiene elegido en el indicador, no el que la vista adivine.
	// `agente` es `AgentePlan` o `AgenteBuild`; la vista nunca envía otro
	// (SPEC-INTERFAZ §Zonas 2, "Indicador de agente": el agente activo es el que
	// manda). `imagenes` son las imágenes (base64) detectadas en el texto del
	// turno; viajan efímeras con este mensaje y no forman parte del historial.
	Enviar(ctx context.Context, sesionID, agente, texto string, imagenes []string) error
	// Pendientes devuelve lo que espera decisión, de cualquier sesión.
	Pendientes() ([]Aprobacion, error)
	// Resolver aplica la decisión sobre una aprobación.
	Resolver(aprobacionID string, aprobar bool) error
	// Pausar y Cancelar actúan sobre el trabajo en curso de una sesión.
	Pausar(sesionID string) error
	Cancelar(sesionID string)
	// Suscribir da el canal de eventos del motor.
	Suscribir() (<-chan Evento, func())
}

// Mensajes internos de la vista. Ninguno sale del módulo: son la forma en que
// las respuestas de los comandos vuelven al Update.
type (
	// initMsg marca el arranque del bucle: al llegar, la vista arma la
	// escucha del canal del motor (T-F010-06).
	initMsg   struct{}
	eventoMsg struct{ Evento Evento }
	// sesionesMsg trae la lista del modal de sesiones. Si hay error, la vista no
	// se bloquea: `Err` se registra en el chat y el modal muestra el aviso «sin
	// sesiones», cerrándose igual con Esc (T-F014-04).
	sesionesMsg struct {
		Sesiones []session.Sesion
		Err      error
	}
	// modelosMsg trae la lista del modal de modelos. Si hay error, la vista no se
	// bloquea ni espera: `Err` se pinta como aviso «sin modelos» y el modal
	// sigue cerrándose con Esc (SPEC-INTERFAZ §Modal de modelos).
	modelosMsg struct {
		Modelos []ModeloLocal
		Err     error
	}
	// capacidadesMsg trae qué declara capaz de hacer el modelo en uso, para la
	// línea de estado bajo el input. Un fallo deja el dato como desconocido
	// («?»): no bloquea nada.
	capacidadesMsg struct {
		Nombre       string
		Herramientas bool
		Vision       bool
		Pensar       bool
		Err          error
	}
	aprobacionesMsg struct{ Items []Aprobacion }
	historialMsg    struct {
		Sesion   string
		Mensajes []MensajeHistorial
		// Tareas es la lista de pasos de la sesión, que llega en la misma carga
		// que el historial (SPEC-TOOLS).
		Tareas []TareaPanel
		// ContextoTokens y LimiteTokens son los números del panel de contexto
		// de la sesión cargada (SPEC-PANEL-CONTEXTO).
		ContextoTokens int
		LimiteTokens   int
	}
	enviadoMsg struct{ Sesion, Texto string }
	errorMsg   struct{ err error }
)

func (e errorMsg) Error() string { return e.err.Error() }

// Suscribir devuelve el comando que espera el siguiente evento del canal. El
// bucle lo relanza tras cada evento (y al arrancar con Init), que es como se
// queda a la escucha sin goroutines propias: la cola de Go entre el motor y el
// comando hace de búfer (T-F010-06).
func Suscribir(ch <-chan Evento) tea.Cmd {
	return func() tea.Msg {
		e, ok := <-ch
		if !ok {
			return nil
		}
		return eventoMsg{Evento: e}
	}
}

// escucharCmd arma la escucha continua. Se suscribe una sola vez; después
// cada evento re-encadena el comando sobre el mismo canal, que es como la
// vista permanece a la escucha sin goroutines propias ni fugas de
// suscripción (T-F010-06).
func (a *App) escucharCmd() tea.Cmd {
	if a.eventos == nil {
		a.eventos, a.baja = a.Puerto.Suscribir()
	}
	return Suscribir(a.eventos)
}

// esDeOtraSesion dice si un evento pertenece a una sesión distinta de la
// activa. La vista pinta la sesión activa y solo esa: un token, una herramienta
// o una etapa de otra sesión no se cuelan en su chat ni en su panel
// (DOMAIN §1: "No hay relación entre sesiones"). Un evento sin `sesion` no se
// puede atribuir —es de alcance de proyecto— y se acepta tal cual.
func (a *App) esDeOtraSesion(e Evento) bool {
	s := e.Datos["sesion"]
	return s != "" && s != a.Panel.SesionID
}

// AplicarEvento traduce un evento del motor a lo que se ve. Es el único sitio
// donde la vista "entiende" el motor, y lo hace sin decidir nada: pinta.
func (a *App) AplicarEvento(e Evento) {
	switch e.Nombre {
	case EventoToken:
		if a.esDeOtraSesion(e) {
			return
		}
		texto := e.Datos["texto"]
		if texto == "" {
			texto = e.Datos["content"]
		}
		// Cada fragmento con contenido que emite el modelo —de razonamiento o de
		// respuesta— cuenta para el consumo vivo del turno. Es una aproximación
		// que el valor exacto de `tokens_turno` corrige al cerrarse
		// (SPEC-PANEL-CONTEXTO).
		if texto != "" {
			a.Panel.Tokens++
		}
		if e.Datos["razonamiento"] == "true" {
			a.Razon.Añadir(texto)
			return
		}
		a.Chat.Token(texto)
	case session.EventoEstadoSesion:
		// El modal de sesiones, si está abierto, ve el cambio sea de quien sea:
		// es la lista de TODAS las sesiones del proyecto (T-F007-05).
		a.Sesiones.ActualizarEstado(e.Datos["sesion"], e.Datos["estado"])
		if e.Datos["sesion"] != a.Panel.SesionID {
			// Es de otra sesión: su estado se ve en el modal de sesiones, no en el panel
			// (el panel refleja la sesión activa, y solo esa).
			return
		}
		a.Panel.Estado = e.Datos["estado"]
		switch e.Datos["estado"] {
		case session.EstadoTrabajando:
			a.enTurno = true
		case session.EstadoTerminada, session.EstadoError:
			// La sesión terminó su turno: se cierra con el tiempo medido y, si
			// esperaba una decisión, sus líneas quedan obsoletas —visibles pero
			// ya sin decisión posible (SPEC-INTERFAZ-ATAJOS, T-F008-06)—.
			a.cerrarTurnoDeVista(true)
		case session.EstadoEsperandoPermiso, session.EstadoInactiva:
			// Parada (pausa de un flujo o cancelación): el contador deja de
			// correr, pero la aprobación pendiente sigue viva y no se marca
			// obsoleta. Solo cierra si el turno llegó a trabajar: el `inactiva`
			// previo a arrancar no toca nada.
			if a.enTurno {
				a.cerrarTurnoDeVista(false)
			}
		}
	case EventoTituloSesion:
		// El modelo generó el título de la sesión a partir de su primera
		// petición: se refleja en la lista del modal y, si es la activa, en el
		// panel. El id no cambia (SPEC-SESIONES).
		nombre := e.Datos["nombre"]
		a.Sesiones.ActualizarNombre(e.Datos["sesion"], nombre)
		if e.Datos["sesion"] == a.Panel.SesionID {
			a.Panel.Sesion = nombre
		}
	case session.EventoNotificacion:
		// Llega aunque no se esté viendo esa sesión (SPEC-SESIONES): la línea se
		// pinta igual. Pero solo cierra el segmento en vivo cuando es de la
		// sesión activa: una notificación de otra no puede cortar el razonamiento
		// o la respuesta que se están viendo.
		if !a.esDeOtraSesion(e) {
			a.cerrarSegmentoEnVivo()
		}
		a.Chat.AñadirSistema(notificacionEnTexto(e.Datos))
	case EventoPeticionAprobacion:
		a.Aprobs.Fijar(append(a.Aprobs.Items, Aprobacion{
			ID:          e.Datos["aprobacion"],
			Sesion:      e.Datos["sesion"],
			Descripcion: e.Datos["descripcion"],
			Motivo:      e.Datos["motivo"],
		}))
		// La propuesta de la sesión activa se ve también en su chat
		// (SPEC-INTERFAZ §Zonas 1, T-F005-06).
		if e.Datos["sesion"] == a.Panel.SesionID {
			a.Chat.AñadirPropuesta(Propuesta{
				ID:          e.Datos["aprobacion"],
				Sesion:      e.Datos["sesion"],
				Descripcion: e.Datos["descripcion"],
			})
		}
		// El contador del panel de datos cuenta las que siguen esperando
		// decisión, de todas las sesiones (SPEC-INTERFAZ §Zonas 3).
		a.Panel.Aprobaciones = a.Aprobs.Pendientes()
	case EventoAprobacionResuelta:
		// Resuelta: la línea se retira del panel y del chat, y el contador se
		// actualiza (INTERFACES §1: "retira o marca la línea y actualiza el
		// contador"). Si era de otra sesión, ni el chat de la activa la mostraba
		// y las llamadas no encuentran su ID: no tocan nada.
		id := e.Datos["aprobacion"]
		a.Aprobs.Resolver(id)
		a.Chat.RetirarPropuesta(id)
		a.Panel.Aprobaciones = a.Aprobs.Pendientes()
	case EventoEtapaIniciada:
		// Cada etapa de un flujo corre como un sub-proceso: la vista solo anuncia
		// su nombre. El texto de un paso intermedio no llega al chat (el motor lo
		// corre en silencio); la entrega final sí.
		if a.esDeOtraSesion(e) {
			return
		}
		a.cerrarSegmentoEnVivo()
		a.Chat.AñadirSistema(LineaProceso(nombreDeEtapa(e.Datos), false))
	case EventoEtapaTerminada:
		// Sin línea: el último paso del flujo escribe la entrega final, y los
		// intermedios no se muestran.
	case EventoEtapaFallida:
		if a.esDeOtraSesion(e) {
			return
		}
		a.cerrarSegmentoEnVivo()
		a.Chat.AñadirSistema(LineaProceso(nombreDeEtapa(e.Datos), true))
	case EventoFlujoPausado, EventoFlujoReanudado, EventoFlujoCancelado:
		// El estado del flujo se refleja en el hilo (EVENTS.md §2, T-F010-05).
		if a.esDeOtraSesion(e) {
			return
		}
		a.cerrarSegmentoEnVivo()
		a.Chat.AñadirSistema(textoDeFlujo(e.Nombre))
	case EventoHerramientaInvocada:
		// Mientras una herramienta se ejecuta, se ve cuál es y con qué agente
		// (SPEC-TOOLS: "Mientras una herramienta se ejecuta, se ve en pantalla
		// cuál es"). El indicador en vivo dice «Usando herramienta: X» y en el
		// hilo queda una línea compacta que se completa al terminar: el verbo y
		// el objetivo —la ruta, el patrón o el comando—, nunca el resto de
		// argumentos.
		if a.esDeOtraSesion(e) {
			return
		}
		a.herramientaEnCurso = e.Datos["herramienta"]
		// La herramienta se interpone en el turno: lo que el modelo haya dicho
		// hasta aquí se cierra como un intercambio para que la línea quede debajo,
		// en el orden en que ocurrió. Lo que diga después abre un globo nuevo.
		if e.Datos["verbo"] != "" {
			a.cerrarSegmentoEnVivo()
		}
		a.Chat.AnotarInvocacion(e.Datos["verbo"], e.Datos["tema"])
	case EventoHerramientaResultado:
		// El resultado no lleva la salida: solo si terminó bien, la medida y si
		// se recortó. Al cerrar, el indicador vuelve a «Pensando» o «Generando».
		if a.esDeOtraSesion(e) {
			return
		}
		a.herramientaEnCurso = ""
		// La duración viaja en milisegundos; sin ella (-1) la línea se pinta sin
		// tiempo, sin dejar hueco (INTERFACES.md §1.1).
		duracion := time.Duration(-1)
		if ms, err := enteroDe(e.Datos["duracion"]); err == nil {
			duracion = time.Duration(ms) * time.Millisecond
		}
		a.Chat.CerrarHerramienta(e.Datos["herramienta"], e.Datos["ok"] == "true",
			e.Datos["truncado"] == "true", e.Datos["medida"], e.Datos["error"], duracion)
	case EventoTokensTurno:
		// El consumo del turno alimenta la línea bajo la entrada
		// (SPEC-PANEL-CONTEXTO). El total del contexto del chat y su límite
		// alimentan la fila CONTEXTO del panel: el contexto es una estimación
		// sobre los mensajes de la sesión, no el consumo del turno. Es de la
		// sesión activa: el de otra no pisa sus números.
		if a.esDeOtraSesion(e) {
			return
		}
		if n, err := enteroDe(e.Datos["salida"]); err == nil {
			a.Panel.Tokens = n
		}
		if n, err := enteroDe(e.Datos["contexto"]); err == nil {
			a.Panel.ContextoTokens = n
			a.Panel.TokensEstimados = true
		}
		if n, err := enteroDe(e.Datos["limite"]); err == nil {
			a.Panel.LimiteTokens = n
		}
	case EventoTodoActualizada:
		// La lista de pasos es de la sesión activa: la de otra no entra al panel
		// (el panel refleja la sesión activa, y solo esa).
		if e.Datos["sesion"] != a.Panel.SesionID {
			return
		}
		a.Panel.Tareas = decodificarTareas(e.Datos["elementos"])
	case EventoCambioAplicado:
		a.cerrarSegmentoEnVivo()
		a.Chat.AñadirSistema("cambio aplicado en " + e.Datos["archivo"])
		// El cambio ya está en change_history (EVENTS.md §4): el dato de git del
		// panel suma un cambio al árbol.
		a.Panel.GitCambios++
	case EventoElementoBloqueado:
		a.cerrarSegmentoEnVivo()
		a.Chat.AñadirSistema("elemento bloqueado: " + e.Datos["elemento"])
	case EventoColaActualizada:
		// La cola es del proyecto, no de una sesión: se ve desde cualquiera
		// (EVENTS.md §2). Payload según EVENTS.md §3: capa y resumen
		// (cuántas pendientes, cuál activa).
		if e.Datos["capa"] != "" {
			a.Panel.Capa = e.Datos["capa"]
		}
		if e.Datos["activa"] != "" {
			a.Panel.ElementoActual = e.Datos["activa"]
		} else if e.Datos["elemento"] != "" {
			a.Panel.ElementoActual = e.Datos["elemento"]
		}
		if n, err := enteroDe(e.Datos["pendientes"]); err == nil {
			a.Panel.ElementosRestantes = n
		}
	}
}

// cerrarSegmentoEnVivo cierra el intercambio en curso —texto y razonamiento—
// antes de escribir en el chat una línea que se interpone: una herramienta, un
// aviso, una etapa. Así el hilo queda en el orden en que ocurrió: el texto que
// precedió a la línea queda arriba de ella y el que venga después abre un globo
// nuevo. Sin nada acumulado no hace nada.
func (a *App) cerrarSegmentoEnVivo() {
	if a.Chat.EnCurso() == "" && !a.Razon.Hay() {
		return
	}
	a.Chat.CerrarSegmento(a.Razon.Texto())
	a.Razon.CerrarTurno()
}

// cerrarTurnoDeVista cierra el turno de la sesión activa: pasa la respuesta en
// curso al historial (con el tiempo que tardó), limpia el razonamiento y apaga
// el contador. Con `obsoletas` marca las aprobaciones pendientes como tales, que
// es lo que corresponde cuando la sesión terminó, no cuando quedó pausada.
func (a *App) cerrarTurnoDeVista(obsoletas bool) {
	a.enTurno = false
	a.PidiendoCancelarEsc = false
	a.herramientaEnCurso = ""
	a.Chat.CerrarTurno(a.Razon.Texto())
	a.Razon.CerrarTurno()
	if obsoletas {
		a.Aprobs.MarcarObsoleta(a.Panel.SesionID)
	}
}

// notificacionEnTexto compone la línea del aviso con el motivo y qué hacer a
// continuación (EVENTS.md §3: sin el contenido, que está en su sitio).
func notificacionEnTexto(datos map[string]string) string {
	texto := datos["motivo"]
	if s := datos["siguiente"]; s != "" {
		texto += " — " + s
	}
	return texto
}

// nombreDeEtapa saca el nombre visible de una etapa del payload del evento. Si
// no viene `nombre`, cae al `etapa` (el id), para no pintar una línea vacía.
func nombreDeEtapa(datos map[string]string) string {
	if n := datos["nombre"]; n != "" {
		return n
	}
	return datos["etapa"]
}

// textoDeFlujo traduce el nombre del evento de flujo a la línea que se ve.
func textoDeFlujo(nombre string) string {
	switch nombre {
	case EventoFlujoPausado:
		return "flujo pausado: espera una aprobación"
	case EventoFlujoReanudado:
		return "flujo reanudado"
	default:
		return "flujo cancelado"
	}
}

func enteroDe(s string) (int, error) {
	var n int
	if s == "" {
		return 0, fmt.Errorf("tui: sin número")
	}
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}

// decodificarTareas lee la lista de pasos que viene en el evento. Un payload
// ilegible deja la lista vacía: es una notificación, no puede tumbar la vista.
func decodificarTareas(datos string) []TareaPanel {
	if datos == "" {
		return nil
	}
	var vista []TareaPanel
	if err := json.Unmarshal([]byte(datos), &vista); err != nil {
		return nil
	}
	return vista
}
