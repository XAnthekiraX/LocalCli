// repos.go — fachadas de negocio sobre la conexión, para que el cableado de
// producción (el arranque) no importe `database/sql`: la invariante de
// DECISIONS.md ("store es el único que escribe en SQLite") se refuerza aquí —
// ni siquiera *lee* SQLite alguien más; todo pasa por estos repositorios.
//
// Cada tipo envuelve una conexión ya abierta y verificada por Open (con sus
// PRAGMA). Métodos = las operaciones compuestas de DATA_FLOW.md §2 (petición
// de permiso, resolución) y las lecturas/escrituras sueltas que usan los
// handlers del arranque. La lógica es la misma de las funciones de paquete:
// esto es un reenvío, no una segunda implementación.
package store

import "database/sql"

// Aprobaciones es el repositorio de approvals sobre una conexión viva.
type Aprobaciones struct{ db *sql.DB }

// NuevasAprobaciones envuelve la conexión para operar sobre approvals.
func NuevasAprobaciones(db *sql.DB) *Aprobaciones { return &Aprobaciones{db: db} }

// PedirPermiso registra la aprobación y pasa la sesión a esperando_permiso en
// la misma transacción (DATA_FLOW.md §2).
func (a *Aprobaciones) PedirPermiso(sessionID, descripcion string) (*Approval, error) {
	return PedirPermiso(a.db, sessionID, descripcion)
}

// ResolverPermiso resuelve la aprobación y devuelve la sesión a su estado en
// la misma transacción (DATA_FLOW.md §2).
func (a *Aprobaciones) ResolverPermiso(approvalID, estado, estadoSesion string) error {
	return ResolverPermiso(a.db, approvalID, estado, estadoSesion)
}

// Obtener lee una aprobación por id.
func (a *Aprobaciones) Obtener(id string) (*Approval, error) { return ObtenerAprobacion(a.db, id) }

// Resolver aplica una decisión final sobre una aprobación (una sola tabla).
func (a *Aprobaciones) Resolver(id, estado string) error { return ResolverAprobacion(a.db, id, estado) }

// Pendientes lista lo que espera decisión, de cualquier sesión.
func (a *Aprobaciones) Pendientes() ([]AprobacionPendiente, error) {
	return AprobacionesPendientes(a.db)
}

// Turnos es el repositorio del turno del agente (messages + razonamiento +
// estado de sesión, atómicos).
type Turnos struct{ db *sql.DB }

// NuevosTurnos envuelve la conexión para cerrar turnos de agente.
func NuevosTurnos(db *sql.DB) *Turnos { return &Turnos{db: db} }

// CerrarAgente persiste la respuesta del agente con su razonamiento en la
// misma transacción (DATA_FLOW.md §2).
func (t *Turnos) CerrarAgente(sessionID, contenido string, inputTokens, outputTokens int, razonamiento, estadoSesion string) (*Message, error) {
	return CerrarTurnoAgente(t.db, sessionID, contenido, inputTokens, outputTokens, razonamiento, estadoSesion)
}

// Cambios es el repositorio de change_history sobre una conexión viva: la
// bitácora de operaciones de archivo (nunca se borra, RELATIONSHIPS.md §3).
type Cambios struct{ db *sql.DB }

// NuevosCambios envuelve la conexión para registrar cambios de archivo.
func NuevosCambios(db *sql.DB) *Cambios { return &Cambios{db: db} }

// Registrar anota una operación de archivo en la bitácora.
func (c *Cambios) Registrar(h *ChangeHistory) error { return RegistrarCambio(c.db, h) }

// Auditorias es el repositorio de context_audit sobre una conexión viva.
type Auditorias struct{ db *sql.DB }

// NuevasAuditorias envuelve la conexión para auditar decisiones de contexto.
func NuevasAuditorias(db *sql.DB) *Auditorias { return &Auditorias{db: db} }

// Registrar anota la decisión del nodo de contexto sobre un documento.
func (a *Auditorias) Registrar(x *ContextAudit) error { return RegistrarAuditoria(a.db, x) }
