package store

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

// Turno es un turno de la conversación capturado al deshacerlo: sus mensajes con
// su razonamiento y sus marcas de tiempo, para poder rehacerlo tal cual.
type Turno struct {
	Mensajes []MensajeConRazonamiento
}

// DeshacerTurno saca de la conversación el último turno de la sesión —el último
// mensaje del usuario y lo que vino después— y lo devuelve para poder rehacerlo.
// Devuelve el turno capturado (vacío si no había ninguno) y cuántos mensajes se
// sacaron. No toca los archivos del proyecto.
func DeshacerTurno(db *sql.DB, projectID, sessionID string) (Turno, int, error) {
	if err := sesionEnProyecto(db, projectID, sessionID); err != nil {
		return Turno{}, 0, err
	}
	var desde int64
	err := db.QueryRow(
		`SELECT rowid FROM messages
		 WHERE session_id = ? AND role = 'user'
		 ORDER BY created_at DESC, rowid DESC LIMIT 1`,
		sessionID,
	).Scan(&desde)
	if err == sql.ErrNoRows {
		return Turno{}, 0, nil
	}
	if err != nil {
		return Turno{}, 0, fmt.Errorf("store: no se pudo localizar el último turno: %w", err)
	}
	turno, err := turnoDesde(db, sessionID, desde)
	if err != nil {
		return Turno{}, 0, err
	}
	if _, err := db.Exec(`DELETE FROM messages WHERE session_id = ? AND rowid >= ?`, sessionID, desde); err != nil {
		return Turno{}, 0, fmt.Errorf("store: no se pudo deshacer el turno: %w", err)
	}
	return turno, len(turno.Mensajes), nil
}

// RehacerTurno vuelve a escribir un turno deshecho, con sus mensajes, su
// razonamiento y sus marcas de tiempo, en una sola transacción: o vuelve el turno
// entero, o no vuelve nada.
func RehacerTurno(db *sql.DB, turno Turno) error {
	if len(turno.Mensajes) == 0 {
		return nil
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, m := range turno.Mensajes {
		if _, err := tx.Exec(
			`INSERT INTO messages (id, session_id, role, content, input_tokens, output_tokens, created_at, duration_ms)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			m.ID, m.SessionID, m.Role, m.Content, m.InputTokens, m.OutputTokens, m.CreatedAt, m.DurationMs,
		); err != nil {
			return fmt.Errorf("store: no se pudo rehacer el turno: %w", err)
		}
		if m.Razonamiento != nil {
			if _, err := tx.Exec(
				`INSERT INTO reasoning (id, message_id, content, created_at) VALUES (?, ?, ?, ?)`,
				uuid.New().String(), m.ID, *m.Razonamiento, m.CreatedAt,
			); err != nil {
				return fmt.Errorf("store: no se pudo rehacer el razonamiento: %w", err)
			}
		}
	}
	return tx.Commit()
}

// turnoDesde lee los mensajes de la sesión a partir de `desde` (su rowid), con su
// razonamiento, en orden.
func turnoDesde(db *sql.DB, sessionID string, desde int64) (Turno, error) {
	rows, err := db.Query(
		`SELECT m.id, m.session_id, m.role, m.content, m.input_tokens, m.output_tokens, m.created_at, m.duration_ms, r.content
		 FROM messages m
		 LEFT JOIN reasoning r ON r.message_id = m.id
		 WHERE m.session_id = ? AND m.rowid >= ?
		 ORDER BY m.created_at, m.rowid`,
		sessionID, desde,
	)
	if err != nil {
		return Turno{}, err
	}
	defer rows.Close()
	var mensajes []MensajeConRazonamiento
	for rows.Next() {
		m, err := leerMensajeConRazonamiento(rows)
		if err != nil {
			return Turno{}, err
		}
		mensajes = append(mensajes, m)
	}
	if err := rows.Err(); err != nil {
		return Turno{}, err
	}
	return Turno{Mensajes: mensajes}, nil
}
