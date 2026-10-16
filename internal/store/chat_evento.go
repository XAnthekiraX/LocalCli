package store

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

// Tipos de línea guardada del hilo (chat_evento.tipo): una etapa de un flujo o el
// cierre de una herramienta.
const (
	ChatEventoProceso     = "proceso"
	ChatEventoHerramienta = "herramienta"
)

// ChatEvento es una línea de pantalla guardada de una sesión: una herramienta o
// una etapa de un flujo. No es conversación: se pinta al retomar la sesión y
// nunca viaja al modelo (DATA_FLOW §chat_evento).
type ChatEvento struct {
	ID        string
	SessionID string
	// Tipo es `proceso` o `herramienta`.
	Tipo string
	// Content es la línea ya compuesta, tal como se vio en pantalla.
	Content string
	// CreatedAt distingue el orden dentro del hilo; alta resolución para
	// intercalarse con los mensajes del mismo turno.
	CreatedAt string
	// DurationMs es lo que tardó la herramienta; NULL en una etapa de flujo.
	DurationMs *int
}

// CrearChatEvento guarda una línea del hilo de una sesión del proyecto. La línea
// de herramienta se guarda ya cerrada, con su duración; la de un flujo no lleva
// duración. Es best-effort para quien la llama: un fallo no debe tumbar el turno.
func CrearChatEvento(db *sql.DB, projectID, sessionID, tipo, content string, durationMs *int) (*ChatEvento, error) {
	if tipo != ChatEventoProceso && tipo != ChatEventoHerramienta {
		return nil, fmt.Errorf("store: tipo de evento de chat inválido: %q", tipo)
	}
	if err := sesionEnProyecto(db, projectID, sessionID); err != nil {
		return nil, err
	}

	id := uuid.New().String()
	now := ahora()

	_, err := db.Exec(
		"INSERT INTO chat_evento (id, session_id, tipo, content, created_at, duration_ms) VALUES (?, ?, ?, ?, ?, ?)",
		id, sessionID, tipo, content, now, durationMs,
	)
	if err != nil {
		return nil, fmt.Errorf("store: no se pudo crear evento de chat: %w", err)
	}

	return &ChatEvento{
		ID:         id,
		SessionID:  sessionID,
		Tipo:       tipo,
		Content:    content,
		CreatedAt:  now,
		DurationMs: durationMs,
	}, nil
}

// ListarChatEventos lista las líneas guardadas de una sesión del proyecto, en
// orden de creación.
func ListarChatEventos(db *sql.DB, projectID, sessionID string) ([]ChatEvento, error) {
	if err := sesionEnProyecto(db, projectID, sessionID); err != nil {
		return nil, err
	}

	rows, err := db.Query(
		"SELECT id, session_id, tipo, content, created_at, duration_ms FROM chat_evento WHERE session_id = ? ORDER BY created_at, rowid",
		sessionID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var eventos []ChatEvento
	for rows.Next() {
		var e ChatEvento
		var durationMs sql.NullInt64
		if err := rows.Scan(&e.ID, &e.SessionID, &e.Tipo, &e.Content, &e.CreatedAt, &durationMs); err != nil {
			return nil, err
		}
		e.DurationMs = enteroOpcional(durationMs)
		eventos = append(eventos, e)
	}
	return eventos, rows.Err()
}
