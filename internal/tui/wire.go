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
	"fmt"

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
)

// Agentes que puede cyclical la acción `agent_cycle` (SPEC-KEYBINDS §Acción
// `agent_cycle`: "plan ↔ build"). Son los dos valores de `tools`, pero la vista
// no puede importarlos: `tui` no conoce `tools` (tests/arquitectura_test.go
// §TestLimitesDeImporteEntreModulos). El puerto los viaja como texto y es el
// adaptador quien los reconoce.
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
	// CapacidadesModelo dice si el modelo indicado declara capacidad de usar
	// herramientas, para la línea de estado bajo el input (SPEC-OLLAMA-PERFIL).
	// La vista no importa `ollama`: el booleano llega ya resuelto.
	CapacidadesModelo(nombre string) (bool, error)
	// AgenteRecordado devuelve el último agente con el que se trabajó, para que
	// la vista arranque en él (SPEC-OLLAMA-PERFIL: la preferencia se recuerda).
	// Vacío significa «sin preferencia»: la vista cae en `plan`. Lo lee el motor,
	// que es quien conoce el archivo de preferencias del usuario.
	AgenteRecordado() string
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
	// Historial devuelve la conversación de una sesión, ya con su razonamiento
	// cerrado, para pintarla al cambiar de sesión (INTERFACES §3). Es una
	// lectura que llega como dato; la vista nunca consulta la base.
	Historial(sesionID string) ([]MensajeHistorial, error)
	// Enviar manda un mensaje a una sesión indicando con qué agente se pide: el
	// que el usuario tiene elegido en el indicador, no el que la vista adivine.
	// `agente` es `AgentePlan` o `AgenteBuild`; la vista nunca envía otro
	// (SPEC-INTERFAZ §Zonas 2, "Indicador de agente": el agente activo es el que
	// manda).
	Enviar(ctx context.Context, sesionID, agente, texto string) error
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
	// capacidadesMsg trae si el modelo en uso declara capacidad de herramientas,
	// para la línea de estado bajo el input. Un fallo deja el dato como
	// desconocido («?»): no bloquea nada.
	capacidadesMsg struct {
		Nombre       string
		Herramientas bool
		Err          error
	}
	aprobacionesMsg struct{ Items []Aprobacion }
	historialMsg    struct {
		Sesion   string
		Mensajes []MensajeHistorial
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

// AplicarEvento traduce un evento del motor a lo que se ve. Es el único sitio
// donde la vista "entiende" el motor, y lo hace sin decidir nada: pinta.
func (a *App) AplicarEvento(e Evento) {
	switch e.Nombre {
	case EventoToken:
		texto := e.Datos["texto"]
		if texto == "" {
			texto = e.Datos["content"]
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
		// Llega aunque no se esté viendo esa sesión (SPEC-SESIONES).
		a.Chat.AñadirSistema(notificacionEnTexto(e.Datos))
	case EventoPeticionAprobacion:
		a.Aprobs.Fijar(append(a.Aprobs.Items, Aprobacion{
			ID:          e.Datos["aprobacion"],
			Sesion:      e.Datos["sesion"],
			Descripcion: e.Datos["descripcion"],
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
		a.Chat.AñadirSistema("etapa iniciada: " + e.Datos["etapa"])
	case EventoEtapaTerminada:
		a.Chat.AñadirSistema("etapa terminada: " + e.Datos["etapa"])
	case EventoEtapaFallida:
		a.Chat.AñadirSistema("etapa fallida: " + e.Datos["etapa"])
	case EventoFlujoPausado, EventoFlujoReanudado, EventoFlujoCancelado:
		// El estado del flujo se refleja en el hilo (EVENTS.md §2, T-F010-05).
		a.Chat.AñadirSistema(textoDeFlujo(e.Nombre))
	case EventoCambioAplicado:
		a.Chat.AñadirSistema("cambio aplicado en " + e.Datos["archivo"])
		// El cambio ya está en change_history (EVENTS.md §4): el dato de git del
		// panel pasa a mostrar el árbol con cambios.
		a.Panel.GitLimpio = false
	case EventoElementoBloqueado:
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

// cerrarTurnoDeVista cierra el turno de la sesión activa: pasa la respuesta en
// curso al historial (con el tiempo que tardó), limpia el razonamiento y apaga
// el contador. Con `obsoletas` marca las aprobaciones pendientes como tales, que
// es lo que corresponde cuando la sesión terminó, no cuando quedó pausada.
func (a *App) cerrarTurnoDeVista(obsoletas bool) {
	a.enTurno = false
	a.PidiendoCancelarEsc = false
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
