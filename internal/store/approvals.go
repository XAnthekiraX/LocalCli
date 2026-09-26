package store

import (
	"database/sql"
	"fmt"
)

// Estados de aprobación (ENUMS.md §2). pendiente → aprobada/declinada/obsoleta;
// las tres son finales.
const (
	ApprovalPendiente = "pendiente"
	ApprovalAprobada  = "aprobada"
	ApprovalDeclinada = "declinada"
	ApprovalObsoleta  = "obsoleta"
)

// Approval es una fila de approvals (TABLES.md §3).
type Approval struct {
	ID          string
	SessionID   string
	Description string
	Status      string
	CreatedAt   string
	ResolvedAt  string // "" = NULL mientras siga pendiente
}

// Permisos es el adaptador de PedirPermiso para módulos que no pueden importar
// `database/sql` (invariante TestStoreEsElUnicoEscritorDeSQLite): `fileops` pide
// la aprobación a través de un método, no de la conexión.
type Permisos struct{ DB *sql.DB }

// PedirPermiso registra la aprobación y deja la sesión esperando la decisión.
func (p Permisos) PedirPermiso(sessionID, descripcion string) (*Approval, error) {
	return PedirPermiso(p.DB, sessionID, descripcion)
}

// PedirAprobación registra lo que una sesión necesita que decidas antes de
// seguir. Nace pendiente, con resolved_at NULL (el CHECK de la base lo exige).
// DATA_FLOW.md exige que esta insercion vaya en la misma transaccion que el
// cambio de estado de la sesion a esperando_permiso; para eso esta PedirPermiso,
// que compone las dos. Esta funcion es el caso de una fila sin transicion
// asociada.
func PedirAprobacion(db *sql.DB, sessionID, descripcion string) (*Approval, error) {
	return pedirAprobacion(db, sessionID, descripcion)
}

func pedirAprobacion(e ejecutor, sessionID, descripcion string) (*Approval, error) {
	now := nowISO()
	id := newID()
	_, err := e.Exec(
		`INSERT INTO approvals (id, session_id, description, status, created_at, resolved_at)
 VALUES (?, ?, ?, ?, ?, NULL)`,
		id, sessionID, descripcion, ApprovalPendiente, now,
	)
	if err != nil {
		return nil, fmt.Errorf("no se pudo registrar la aprobación: %w", traducirError(err))
	}
	return &Approval{ID: id, SessionID: sessionID, Description: descripcion, Status: ApprovalPendiente, CreatedAt: now}, nil
}

// ResolverAprobacion pasa una aprobación pendiente a aprobada, declinada u
// obsoleta y rellena resolved_at en el mismo UPDATE. La transición contraria
// (resolver dos veces) se rechaza en código: los CHECK solo miran la fila nueva
// (CONSTRAINTS.md §2), así que la legalidad la aplica quien escribe
// (BUSINESS_RULES.md).
func ResolverAprobacion(db *sql.DB, id, estado string) error {
	return resolverAprobacion(db, id, estado)
}

func resolverAprobacion(e ejecutor, id, estado string) error {
	switch estado {
	case ApprovalAprobada, ApprovalDeclinada, ApprovalObsoleta:
	default:
		return fmt.Errorf("%w: %q no es un estado final de aprobación", ErrEstadoIlegal, estado)
	}
	res, err := e.Exec(
		`UPDATE approvals SET status = ?, resolved_at = ?
 WHERE id = ? AND status = 'pendiente'`,
		estado, nowISO(), id,
	)
	if err != nil {
		return traducirError(err)
	}
	return filasAfectadas(res, 1, fmt.Sprintf("aprobación %s (¿ya resuelta?)", id))
}

// ObtenerAprobacion lee una aprobación por id.
func ObtenerAprobacion(db *sql.DB, id string) (*Approval, error) {
	return obtenerAprobacion(db, id)
}

// obtenerAprobacion acepta cualquier ejecutor (DB o Tx): ResolverPermiso lo usa
// para leer la sesion duena de la aprobacion dentro de la misma transaccion que
// la resuelve.
func obtenerAprobacion(e ejecutor, id string) (*Approval, error) {
	row := e.QueryRow(
		`SELECT id, session_id, description, status, created_at, COALESCE(resolved_at, '')
 FROM approvals WHERE id = ?`, id)
	return escanearAprobacion(row)
}

func escanearAprobacion(row rowScanner) (*Approval, error) {
	var a Approval
	err := row.Scan(&a.ID, &a.SessionID, &a.Description, &a.Status, &a.CreatedAt, &a.ResolvedAt)
	if err != nil {
		return nil, traducirError(err)
	}
	return &a, nil
}

// AprobacionesPendientes es la consulta del panel global (QUERIES.md §1):
// pendientes de CUALQUIER sesión, con el nombre de la sesión vía JOIN, ordenadas
// por created_at y aceleradas por el índice parcial idx_approvals_pending.
type AprobacionPendiente struct {
	Approval
	SessionName string
}

func AprobacionesPendientes(db *sql.DB) ([]AprobacionPendiente, error) {
	rows, err := db.Query(
		`SELECT a.id, a.session_id, s.name AS session_name, a.description, a.status, a.created_at,
        COALESCE(a.resolved_at, '')
 FROM approvals a
 JOIN sessions s ON s.id = a.session_id
 WHERE a.status = 'pendiente'
 ORDER BY a.created_at`)
	if err != nil {
		return nil, traducirError(err)
	}
	defer rows.Close()
	var out []AprobacionPendiente
	for rows.Next() {
		var p AprobacionPendiente
		err := rows.Scan(&p.ID, &p.SessionID, &p.SessionName, &p.Description, &p.Status, &p.CreatedAt, &p.ResolvedAt)
		if err != nil {
			return nil, traducirError(err)
		}
		out = append(out, p)
	}
	return out, traducirError(rows.Err())
}

// PendientesDeAprobacion es el método del adaptador Sesiones (sessions.go) para
// la consulta del panel global: lo que espera decisión, de cualquier sesión.
// `session` lo usa para avisar de que una sesión está esperando permiso.
func (s Sesiones) PendientesDeAprobacion() ([]AprobacionPendiente, error) {
	return AprobacionesPendientes(s.DB)
}

// ContarAprobacionesPendientes mantiene el contador del panel en cada
// redibujado. QUERIES.md §5 prohíbe contar sobre la tabla entera: se filtra por
// status = 'pendiente' para aprovechar el índice parcial.
func ContarAprobacionesPendientes(db *sql.DB) (int, error) {
	var n int
	err := db.QueryRow(`SELECT COUNT(*) FROM approvals WHERE status = 'pendiente'`).Scan(&n)
	if err != nil {
		return 0, traducirError(err)
	}
	return n, nil
}

// ListarAprobacionesDeSesion ve las aprobaciones de una sesión filtradas por
// estado (patrón de QUERIES.md §2, acelerado por idx_approvals_session_status).
// estado vacío lista todas.
func ListarAprobacionesDeSesion(db *sql.DB, sessionID, estado string) ([]Approval, error) {
	q := `SELECT id, session_id, description, status, created_at, COALESCE(resolved_at, '') FROM approvals WHERE session_id = ?`
	args := []any{sessionID}
	if estado != "" {
		q += ` AND status = ?`
		args = append(args, estado)
	}
	q += ` ORDER BY created_at`
	rows, err := db.Query(q, args...)
	if err != nil {
		return nil, traducirError(err)
	}
	defer rows.Close()
	var out []Approval
	for rows.Next() {
		a, err := escanearAprobacion(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *a)
	}
	return out, traducirError(rows.Err())
}
