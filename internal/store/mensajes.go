package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Mensaje es un turno de la conversación.
type Mensaje struct {
	ID           string
	SessionID    string
	Role         string
	Content      string
	InputTokens  *int
	OutputTokens *int
	CreatedAt    string
	DurationMs   *int
}

// MensajeConRazonamiento es un mensaje con el razonamiento de su turno, si lo tiene.
type MensajeConRazonamiento struct {
	Mensaje
	Razonamiento *string
}

// CrearMensaje crea un mensaje de una sesión del proyecto.
func CrearMensaje(db *sql.DB, projectID, sessionID, role, content string, inputTokens, outputTokens, durationMs *int) (*Mensaje, error) {
	if err := sesionEnProyecto(db, projectID, sessionID); err != nil {
		return nil, err
	}

	id := uuid.New().String()
	now := time.Now().UTC().Format(time.RFC3339)

	_, err := db.Exec(
		"INSERT INTO messages (id, session_id, role, content, input_tokens, output_tokens, created_at, duration_ms) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		id, sessionID, role, content, inputTokens, outputTokens, now, durationMs,
	)
	if err != nil {
		return nil, fmt.Errorf("store: no se pudo crear mensaje: %w", err)
	}

	return &Mensaje{
		ID:           id,
		SessionID:    sessionID,
		Role:         role,
		Content:      content,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		CreatedAt:    now,
		DurationMs:   durationMs,
	}, nil
}

// ListarMensajes lista los mensajes de una sesión del proyecto, en orden.
func ListarMensajes(db *sql.DB, projectID, sessionID string) ([]Mensaje, error) {
	if err := sesionEnProyecto(db, projectID, sessionID); err != nil {
		return nil, err
	}

	rows, err := db.Query(
		"SELECT id, session_id, role, content, input_tokens, output_tokens, created_at, duration_ms FROM messages WHERE session_id = ? ORDER BY created_at, rowid",
		sessionID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var mensajes []Mensaje
	for rows.Next() {
		m, err := leerMensaje(rows)
		if err != nil {
			return nil, err
		}
		mensajes = append(mensajes, m)
	}
	return mensajes, rows.Err()
}

// Historial devuelve la conversación de una sesión con el razonamiento de cada mensaje.
func Historial(db *sql.DB, projectID, sessionID string) ([]MensajeConRazonamiento, error) {
	if err := sesionEnProyecto(db, projectID, sessionID); err != nil {
		return nil, err
	}

	rows, err := db.Query(
		`SELECT m.id, m.session_id, m.role, m.content, m.input_tokens, m.output_tokens, m.created_at, m.duration_ms, r.content
		 FROM messages m
		 LEFT JOIN reasoning r ON r.message_id = m.id
		 WHERE m.session_id = ?
		 ORDER BY m.created_at, m.rowid`,
		sessionID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var historial []MensajeConRazonamiento
	for rows.Next() {
		h, err := leerMensajeConRazonamiento(rows)
		if err != nil {
			return nil, err
		}
		historial = append(historial, h)
	}
	return historial, rows.Err()
}

// leerMensajeConRazonamiento lee una fila del historial: el mensaje y, si lo
// tiene, el razonamiento de su turno.
func leerMensajeConRazonamiento(rows *sql.Rows) (MensajeConRazonamiento, error) {
	var h MensajeConRazonamiento
	var inputTokens, outputTokens, durationMs sql.NullInt64
	var razonamiento sql.NullString
	if err := rows.Scan(&h.ID, &h.SessionID, &h.Role, &h.Content, &inputTokens, &outputTokens, &h.CreatedAt, &durationMs, &razonamiento); err != nil {
		return MensajeConRazonamiento{}, err
	}
	h.InputTokens = enteroOpcional(inputTokens)
	h.OutputTokens = enteroOpcional(outputTokens)
	h.DurationMs = enteroOpcional(durationMs)
	if razonamiento.Valid {
		h.Razonamiento = &razonamiento.String
	}
	return h, nil
}

// leerMensaje lee una fila de `messages`.
func leerMensaje(rows *sql.Rows) (Mensaje, error) {
	var m Mensaje
	var inputTokens, outputTokens, durationMs sql.NullInt64
	if err := rows.Scan(&m.ID, &m.SessionID, &m.Role, &m.Content, &inputTokens, &outputTokens, &m.CreatedAt, &durationMs); err != nil {
		return Mensaje{}, err
	}
	m.InputTokens = enteroOpcional(inputTokens)
	m.OutputTokens = enteroOpcional(outputTokens)
	m.DurationMs = enteroOpcional(durationMs)
	return m, nil
}

// enteroOpcional distingue NULL («no lo sé») de 0.
func enteroOpcional(v sql.NullInt64) *int {
	if !v.Valid {
		return nil
	}
	n := int(v.Int64)
	return &n
}
