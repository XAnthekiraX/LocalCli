package store

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

// DeshacerDesde saca de la conversación todo lo que va desde el mensaje indicado
// hasta el final de la sesión —el mensaje y lo que vino después— y lo devuelve
// para poder rehacerlo. El mensaje debe ser del usuario: revertir a un globo del
// agente no tendría un corte claro. Devuelve el turno capturado (vacío si no
// había nada) y cuántos mensajes se sacaron. No toca los archivos del proyecto.
func DeshacerDesde(db *sql.DB, projectID, sessionID, mensajeID string) (Turno, int, error) {
	if err := sesionEnProyecto(db, projectID, sessionID); err != nil {
		return Turno{}, 0, err
	}
	var desde int64
	var role, desdeMarca string
	err := db.QueryRow(
		`SELECT rowid, role, created_at FROM messages WHERE session_id = ? AND id = ?`,
		sessionID, mensajeID,
	).Scan(&desde, &role, &desdeMarca)
	if err == sql.ErrNoRows {
		return Turno{}, 0, nil
	}
	if err != nil {
		return Turno{}, 0, fmt.Errorf("store: no se pudo localizar el mensaje: %w", err)
	}
	if role != "user" {
		return Turno{}, 0, nil
	}
	turno, err := turnoDesde(db, sessionID, desde)
	if err != nil {
		return Turno{}, 0, err
	}
	turno.Eventos, err = eventosDesde(db, sessionID, desdeMarca)
	if err != nil {
		return Turno{}, 0, err
	}
	if _, err := db.Exec(`DELETE FROM messages WHERE session_id = ? AND rowid >= ?`, sessionID, desde); err != nil {
		return Turno{}, 0, fmt.Errorf("store: no se pudo revertir el turno: %w", err)
	}
	if _, err := db.Exec(`DELETE FROM chat_evento WHERE session_id = ? AND created_at >= ?`, sessionID, desdeMarca); err != nil {
		return Turno{}, 0, fmt.Errorf("store: no se pudo revertir el turno: %w", err)
	}
	return turno, len(turno.Mensajes), nil
}

// BifurcarSesion crea una sesión nueva en el proyecto y copia en ella todo el
// historial de la sesión de origen hasta el mensaje indicado, incluido, con su
// razonamiento: el punto de partida de la bifurcación es ese mensaje del usuario.
// La nueva sesión nace con el nombre «FORK:<nombre original>` y sin proyector ni
// modelo propios: hereda del proyecto como cualquier sesión recién creada.
func BifurcarSesion(db *sql.DB, projectID, sessionID, mensajeID string) (*Sesion, error) {
	original, err := ObtenerSesion(db, projectID, sessionID)
	if err != nil {
		return nil, err
	}
	var hasta int64
	var hastaMarca string
	err = db.QueryRow(
		`SELECT rowid, created_at FROM messages WHERE session_id = ? AND id = ?`,
		sessionID, mensajeID,
	).Scan(&hasta, &hastaMarca)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("store: el mensaje no pertenece a la sesión")
	}
	if err != nil {
		return nil, fmt.Errorf("store: no se pudo localizar el mensaje: %w", err)
	}

	nueva, err := CrearSesion(db, projectID, "FORK:"+original.Name, nil)
	if err != nil {
		return nil, err
	}
	if err := copiarHistorialHasta(db, sessionID, nueva.ID, hasta, hastaMarca); err != nil {
		_ = EliminarSesion(db, projectID, nueva.ID)
		return nil, err
	}
	return nueva, nil
}

// copiarHistorialHasta copia a `destino` los mensajes de `origen` con rowid menor
// o igual a `hasta`, junto con su razonamiento, y las líneas del hilo creadas
// hasta la marca `hastaMarca`, en una sola transacción: o entra el historial
// entero, o no entra nada. Cada mensaje copiado recibe un id nuevo, porque los
// ids son la clave del mensaje y no pueden repetirse entre sesiones; el
// razonamiento se reasocia al id nuevo.
func copiarHistorialHasta(db *sql.DB, origen, destino string, hasta int64, hastaMarca string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	rows, err := tx.Query(
		`SELECT m.id, m.role, m.content, m.input_tokens, m.output_tokens, m.created_at, m.duration_ms, r.content
		 FROM messages m
		 LEFT JOIN reasoning r ON r.message_id = m.id
		 WHERE m.session_id = ? AND m.rowid <= ?
		 ORDER BY m.rowid`,
		origen, hasta,
	)
	if err != nil {
		return fmt.Errorf("store: no se pudo leer el historial a bifurcar: %w", err)
	}
	defer rows.Close()

	type copia struct {
		role, content, createdAt string
		input, output, duration  sql.NullInt64
		razonamiento             sql.NullString
	}
	var copias []copia
	for rows.Next() {
		var c copia
		var id string
		if err := rows.Scan(&id, &c.role, &c.content, &c.input, &c.output, &c.createdAt, &c.duration, &c.razonamiento); err != nil {
			return err
		}
		copias = append(copias, c)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	rows.Close()

	for _, c := range copias {
		nuevoID := uuid.New().String()
		if _, err := tx.Exec(
			`INSERT INTO messages (id, session_id, role, content, input_tokens, output_tokens, created_at, duration_ms)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			nuevoID, destino, c.role, c.content, c.input, c.output, c.createdAt, c.duration,
		); err != nil {
			return fmt.Errorf("store: no se pudo copiar el mensaje: %w", err)
		}
		if c.razonamiento.Valid {
			if _, err := tx.Exec(
				`INSERT INTO reasoning (id, message_id, content, created_at) VALUES (?, ?, ?, ?)`,
				uuid.New().String(), nuevoID, c.razonamiento.String, c.createdAt,
			); err != nil {
				return fmt.Errorf("store: no se pudo copiar el razonamiento: %w", err)
			}
		}
	}

	// Las líneas del hilo hasta el punto de corte acompañan al historial: la
	// sesión bifurcada se lee igual que la original hasta ahí.
	eventos, err := eventosHasta(tx, origen, hastaMarca)
	if err != nil {
		return err
	}
	for _, e := range eventos {
		e.ID = uuid.New().String()
		if err := insertarChatEvento(tx, destino, e); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// eventosHasta lee las líneas del hilo de la sesión creadas hasta `hasta`
// (incluida), en orden. `hasta` es la marca del mensaje del corte.
func eventosHasta(q queryer, sessionID, hasta string) ([]ChatEvento, error) {
	rows, err := q.Query(
		`SELECT id, session_id, tipo, content, created_at, duration_ms,
		        call_id, herramienta, argumentos, salida, error, ok, truncado
		 FROM chat_evento
		 WHERE session_id = ? AND created_at <= ?
		 ORDER BY created_at, rowid`,
		sessionID, hasta,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var eventos []ChatEvento
	for rows.Next() {
		e, err := leerChatEvento(rows)
		if err != nil {
			return nil, err
		}
		eventos = append(eventos, e)
	}
	return eventos, rows.Err()
}

// queryer es lo común a *sql.DB y *sql.Tx para leer dentro o fuera de una
// transacción.
type queryer interface {
	Query(query string, args ...any) (*sql.Rows, error)
}
