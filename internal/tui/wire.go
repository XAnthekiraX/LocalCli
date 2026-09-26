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
	EventoCambioAplicado     = "cambio_aplicado"
	EventoColaActualizada    = "cola_actualizada"
	EventoElementoBloqueado  = "elemento_bloqueado"
	EventoEtapaIniciada      = "etapa_iniciada"
	EventoEtapaTerminada     = "etapa_terminada"
	EventoEtapaFallida       = "etapa_fallida"
)

// Puerto es lo que la vista necesita de `session`. Todo lo que no esté aquí, la
// vista no lo puede hacer.
type Puerto interface {
	// ResolverActiva devuelve la sesión activa del proyecto: la retoma si
	// existe o crea una nueva (SPEC-INTERFAZ §Pantalla de bienvenida).
	ResolverActiva() (*session.Sesion, error)
	// Listar lista las sesiones del proyecto para el selector.
	Listar() ([]session.Sesion, error)
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
	eventoMsg       struct{ Evento Evento }
	sesionesMsg     struct{ Sesiones []session.Sesion }
	aprobacionesMsg struct{ Items []Aprobacion }
	enviadoMsg      struct{ Sesion, Texto string }
	errorMsg        struct{ err error }
)

func (e errorMsg) Error() string { return e.err.Error() }

// Suscribir devuelve el comando que entrega el siguiente evento del canal. Se
// vuelve a lanzar tras cada evento, que es como se queda a la escucha sin un
// bucle propio.
func Suscribir(ch <-chan Evento) tea.Cmd {
	return func() tea.Msg {
		e, ok := <-ch
		if !ok {
			return nil
		}
		return eventoMsg{Evento: e}
	}
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
		if e.Datos["sesion"] != a.Panel.SesionID {
			// Es de otra sesión: su estado se ve en el selector, no en el panel
			// (el panel refleja la sesión activa, y solo esa).
			return
		}
		a.Panel.Estado = e.Datos["estado"]
		if e.Datos["estado"] == session.EstadoTerminada || e.Datos["estado"] == session.EstadoError {
			a.Chat.CerrarTurno(a.Razon.Texto())
			a.Razon.CerrarTurno()
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
	case EventoEtapaIniciada:
		a.Chat.AñadirSistema("etapa iniciada: " + e.Datos["etapa"])
	case EventoEtapaTerminada:
		a.Chat.AñadirSistema("etapa terminada: " + e.Datos["etapa"])
	case EventoEtapaFallida:
		a.Chat.AñadirSistema("etapa fallida: " + e.Datos["etapa"])
	case EventoCambioAplicado:
		a.Chat.AñadirSistema("cambio aplicado en " + e.Datos["archivo"])
	case EventoElementoBloqueado:
		a.Chat.AñadirSistema("elemento bloqueado: " + e.Datos["elemento"])
	case EventoColaActualizada:
		if e.Datos["elemento"] != "" {
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

func enteroDe(s string) (int, error) {
	var n int
	if s == "" {
		return 0, fmt.Errorf("tui: sin número")
	}
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}
