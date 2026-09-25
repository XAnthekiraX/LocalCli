package store

import (
	"database/sql"
	"fmt"
)

// Estados de sesión (ENUMS.md §2). Son texto cerrado validado por CHECK en la
// base; estas constantes dan un único nombre en Go para cada valor.
const (
	StatusInactiva         = "inactiva"
	StatusTrabajando       = "trabajando"
	StatusEsperandoPermiso = "esperando_permiso"
	StatusTerminada        = "terminada"
	StatusError            = "error"
)

// Capas conocidas (ENUMS.md, TABLES.md): layer es NULL para una sesión general.
const (
	LayerBackend  = "backend"
	LayerFrontend = "frontend"
)

// Session es una fila de sessions (TABLES.md §3).
type Session struct {
	ID        string
	Name      string
	Layer     string // "" = sesión general (NULL en la base)
	Status    string
	CreatedAt string
	UpdatedAt string
}

// ValidarTransicionSesion comprueba si pasar de `from` a `to` es una
// transición legal según ENUMS.md §3. La legacidad no la impone un CHECK
// (CONSTRAINTS.md §2: "CHECK solo mira la fila nueva"), así que se aplica en
// el código, aquí, en el módulo que escribe.
func ValidarTransicionSesion(from, to string) bool {
	switch from {
	case StatusInactiva:
		return to == StatusTrabajando || to == StatusError
	case StatusTrabajando:
		return to == StatusInactiva || to == StatusEsperandoPermiso ||
			to == StatusTerminada || to == StatusError
	case StatusEsperandoPermiso:
		return to == StatusTrabajando || to == StatusError
	case StatusTerminada:
		// terminal: solo vuelve a actividad pasando antes por inactiva
		return to == StatusInactiva || to == StatusError
	case StatusError:
		// "una sesión no vuelve a trabajando sin pasar antes por inactiva"
		return to == StatusInactiva || to == StatusError
	}
	return false
}

// CrearSesion inserta una sesión nueva en estado inactiva (su valor por
// defecto en el esquema) y devuelve la fila creada.
func CrearSesion(db *sql.DB, nombre, capa string) (*Session, error) {
	now := nowISO()
	id := newID()
	_, err := db.Exec(
		`INSERT INTO sessions (id, name, layer, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		id, nombre, nullStr(capa), StatusInactiva, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("no se pudo crear la sesión: %w", traducirError(err))
	}
	return &Session{ID: id, Name: nombre, Layer: capa, Status: StatusInactiva, CreatedAt: now, UpdatedAt: now}, nil
}

// ObtenerSesion lee una sesión por id. Devuelve ErrNoEncontrado si no existe.
var ErrNoEncontrado = sql.ErrNoRows

func ObtenerSesion(db *sql.DB, id string) (*Session, error) {
	row := db.QueryRow(
		`SELECT id, name, COALESCE(layer, ''), status, created_at, updated_at
		 FROM sessions WHERE id = ?`, id)
	return escanearSesion(row)
}

func escanearSesion(row rowScanner) (*Session, error) {
	var s Session
	err := row.Scan(&s.ID, &s.Name, &s.Layer, &s.Status, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, traducirError(err)
	}
	return &s, nil
}

// rowScanner permite compartir el escaneo entre QueryRow y las filas de un Rows.
type rowScanner interface {
	Scan(dest ...any) error
}

// ListarSesiones lista sesiones ordenadas por actividad reciente (patrón
// "Listar por estado y actividad" de QUERIES.md §2, acelerado por
// idx_sessions_updated). Si estado es vacío, lista todas; si carpeta/proyecto
// no filtra (la base es por proyecto), se ignora el concepto de carpeta: cada
// archivo SQLite pertenece ya a una carpeta.
func ListarSesiones(db *sql.DB, estado string) ([]Session, error) {
	q := `SELECT id, name, COALESCE(layer, ''), status, created_at, updated_at FROM sessions`
	args := []any{}
	if estado != "" {
		q += ` WHERE status = ?`
		args = append(args, estado)
	}
	q += ` ORDER BY updated_at DESC`
	rows, err := db.Query(q, args...)
	if err != nil {
		return nil, traducirError(err)
	}
	defer rows.Close()
	var out []Session
	for rows.Next() {
		s, err := escanearSesion(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *s)
	}
	return out, traducirError(rows.Err())
}

// ActualizarEstadoSesion cambia el estado de una sesión y toca updated_at
// (patrón "Escribir estado" de QUERIES.md §2: el motor de flujos en cada
// transición). Rechaza en código la transición ilegal (CONSTRAINTS.md §2).
func ActualizarEstadoSesion(db *sql.DB, id, estadoNuevo string) error {
	actual, err := ObtenerSesion(db, id)
	if err != nil {
		return err
	}
	if !ValidarTransicionSesion(actual.Status, estadoNuevo) {
		return fmt.Errorf("%w: sesión %s no puede pasar de %q a %q", ErrEstadoIlegal, id, actual.Status, estadoNuevo)
	}
	res, err := db.Exec(`UPDATE sessions SET status = ?, updated_at = ? WHERE id = ?`,
		estadoNuevo, nowISO(), id)
	if err != nil {
		return traducirError(err)
	}
	return filasAfectadas(res, 1, fmt.Sprintf("sesión %s", id))
}

// BorrarSesion elimina una sesión de forma definitiva. messages, reasoning
// (vía messages), approvals y context_audit caen en cascada; change_history
// sobrevive con session_id a NULL (RELATIONSHIPS.md §3).
func BorrarSesion(db *sql.DB, id string) error {
	res, err := db.Exec(`DELETE FROM sessions WHERE id = ?`, id)
	if err != nil {
		return traducirError(err)
	}
	return filasAfectadas(res, 1, fmt.Sprintf("sesión %s", id))
}
