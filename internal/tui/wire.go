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
	EventoEtapaIniciada      = "etapa_iniciada"
	EventoEtapaTerminada     = "etapa_terminada"
	EventoEtapaFallida       = "etapa_fallida"
	EventoFlujoPausado       = "flujo_pausado"
	EventoFlujoReanudado     = "flujo_reanudado"
	EventoFlujoCancelado     = "flujo_cancelado"
)

// Puerto es lo que la vista necesita de `session`. Todo lo que no esté aquí, la
// vista no lo puede hacer.
type Puerto interface {
	// ResolverActiva devuelve la sesión activa del proyecto: la retoma si
	// existe o crea una nueva (SPEC-INTERFAZ §Pantalla de bienvenida).
	ResolverActiva() (*session.Sesion, error)
	// Listar lista las sesiones del proyecto para el selector.
	Listar() ([]session.Sesion, error)
	// Historial devuelve la conversación de una sesión, ya con su razonamiento
	// cerrado, para pintarla al cambiar de sesión (INTERFACES §3). Es una
	// lectura que llega como dato; la vista nunca consulta la base.
	Historial(sesionID string) ([]MensajeHistorial, error)
	// Enviar manda un mensaje a una sesión.
	Enviar(ctx context.Context, sesionID, texto string) error
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
	initMsg         struct{}
	eventoMsg       struct{ Evento Evento }
	sesionesMsg     struct{ Sesiones []session.Sesion }
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
		// El selector, si está abierto, ve el cambio sea de quien sea: es la
		// lista de TODAS las sesiones del proyecto (T-F007-05).
		a.Selector.ActualizarEstado(e.Datos["sesion"], e.Datos["estado"])
		if e.Datos["sesion"] != a.Panel.SesionID {
			// Es de otra sesión: su estado se ve en el selector, no en el panel
			// (el panel refleja la sesión activa, y solo esa).
			return
		}
		a.Panel.Estado = e.Datos["estado"]
		if e.Datos["estado"] == session.EstadoTerminada || e.Datos["estado"] == session.EstadoError {
			a.Chat.CerrarTurno(a.Razon.Texto())
			a.Razon.CerrarTurno()
			// La sesión terminó mientras esperaba: sus líneas quedan marcadas
			// como obsoletas, visibles pero ya sin decisión posible
			// (SPEC-INTERFAZ-ATAJOS, T-F008-06).
			a.Aprobs.MarcarObsoleta(e.Datos["sesion"])
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
