// store.go — T-B013-02: persistir sesiones y mensajes a través de `store`.
//
// Fuente de verdad: DOMAIN.md §3 ("`session` contiene `messages`… `session`
// contiene `approvals`"), RELATIONSHIPS.md §3 (borrar una sesión arrastra
// messages, reasoning, approvals y context_audit, y deja vivo change_history) y
// DECISIONS.md ("store es el único que escribe en SQLite").
//
// El módulo no puede importar `database/sql` (invariante
// TestStoreEsElUnicoEscritorDeSQLite), así que depende de esta interfaz, que
// implementa `store.Sesiones`. La consecuencia práctica es doble: `session` no
// sabe de SQL, y sus tests corren con un almacén en memoria.
package session

import "localcli/internal/store"

// Almacen es lo que `session` necesita persistir. Lo implementa
// `store.Sesiones`.
type Almacen interface {
	// Crear inserta una sesión nueva (nace inactiva).
	Crear(nombre, capa string) (*store.Session, error)
	// CrearConMotor inserta una sesión nueva con su par motor/modelo asignado:
	// una sesión nueva parte del último motor y modelo usados (SPEC-SESIONES).
	CrearConMotor(nombre, capa, motorID, modelo string) (*store.Session, error)
	// CambiarMotorModelo fija el par motor/modelo de la sesión sin tocar su
	// identidad ni su historial. Un valor vacío deja NULL (sin asignar).
	CambiarMotorModelo(id, motorID, modelo string) error
	// Obtener lee una sesión por id.
	Obtener(id string) (*store.Session, error)
	// Listar lista las sesiones del proyecto, filtradas por estado si no es "".
	Listar(estado string) ([]store.Session, error)
	// CambiarEstado aplica una transición de estado ya validada.
	CambiarEstado(id, estado string) error
	// Borrar elimina la sesión y todo lo que cuelga de ella.
	Borrar(id string) error
	// Renombrar cambia el nombre visible de la sesión (su título). El id sigue
	// siendo la identidad: renombrar nunca toca los mensajes ni las referencias.
	Renombrar(id, nombre string) error
	// Historial devuelve la conversación en orden, con su razonamiento.
	Historial(sessionID string) ([]store.MensajeConRazonamiento, error)
	// EscribirMensaje inserta un turno de conversación.
	EscribirMensaje(m *store.Message) error
	// CerrarTurno cierra la respuesta del agente: mensaje, razonamiento y
	// estado de la sesión, en la misma transacción. Los tokens y la duración
	// llegan como -1 cuando no se midieron.
	CerrarTurno(sessionID, contenido string, inputTokens, outputTokens, duracionMs int, razonamiento, estadoSesion string) (*store.Message, error)
	// PendientesDeAprobacion devuelve lo que espera decisión, de cualquier
	// sesión del proyecto.
	PendientesDeAprobacion() ([]store.AprobacionPendiente, error)
}

// El adaptador de producción cumple la interfaz: si deja de cumplirla, no
// compila.
var _ Almacen = store.Sesiones{}
