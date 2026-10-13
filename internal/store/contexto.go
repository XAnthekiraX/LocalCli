package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ContextAudit representa un registro de auditoría de contexto.
type ContextAudit struct {
	ID        string
	SessionID string
	Stage     string
	Document  string
	Decision  string
	Reason    *string
	Tokens    *int
	CreatedAt string
}

// CrearContextAudit crea un registro de auditoría para una sesión del proyecto.
func CrearContextAudit(db *sql.DB, projectID, sessionID, stage, document, decision string, reason *string, tokens *int) (*ContextAudit, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM sessions WHERE id = ? AND project_id = ?", sessionID, projectID).Scan(&count)
	if err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, fmt.Errorf("store: sesión no encontrada en el proyecto")
	}

	id := uuid.New().String()
	now := time.Now().UTC().Format(time.RFC3339)

	_, err = db.Exec(
		"INSERT INTO context_audit (id, session_id, stage, document, decision, reason, tokens, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		id, sessionID, stage, document, decision, reason, tokens, now,
	)
	if err != nil {
		return nil, err
	}

	return &ContextAudit{
		ID:        id,
		SessionID: sessionID,
		Stage:     stage,
		Document:  document,
		Decision:  decision,
		Reason:    reason,
		Tokens:    tokens,
		CreatedAt: now,
	}, nil
}

// ListarContextAuditProyecto lista auditoría de contexto de todas las sesiones del proyecto.
func ListarContextAuditProyecto(db *sql.DB, projectID string) ([]ContextAudit, error) {
	rows, err := db.Query(
		`SELECT ca.id, ca.session_id, ca.stage, ca.document, ca.decision, ca.reason, ca.tokens, ca.created_at
		 FROM context_audit ca
		 JOIN sessions s ON s.id = ca.session_id
		 WHERE s.project_id = ?
		 ORDER BY ca.created_at DESC`,
		projectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var audits []ContextAudit
	for rows.Next() {
		var ca ContextAudit
		var reason sql.NullString
		var tokens sql.NullInt64
		if err := rows.Scan(&ca.ID, &ca.SessionID, &ca.Stage, &ca.Document, &ca.Decision, &reason, &tokens, &ca.CreatedAt); err != nil {
			return nil, err
		}
		if reason.Valid {
			ca.Reason = &reason.String
		}
		if tokens.Valid {
			v := int(tokens.Int64)
			ca.Tokens = &v
		}
		audits = append(audits, ca)
	}
	return audits, nil
}
