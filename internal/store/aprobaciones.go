package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Aprobacion representa una aprobación pendiente.
type Aprobacion struct {
	ID          string
	SessionID   string
	Description string
	Status      string
	CreatedAt   string
	ResolvedAt  *string
}

// CrearAprobacion crea una aprobación para una sesión del proyecto.
func CrearAprobacion(db *sql.DB, projectID, sessionID, description string) (*Aprobacion, error) {
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
	status := "pendiente"

	_, err = db.Exec(
		"INSERT INTO approvals (id, session_id, description, status, created_at, resolved_at) VALUES (?, ?, ?, ?, ?, ?)",
		id, sessionID, description, status, now, nil,
	)
	if err != nil {
		return nil, err
	}

	return &Aprobacion{
		ID:          id,
		SessionID:   sessionID,
		Description: description,
		Status:      status,
		CreatedAt:   now,
		ResolvedAt:  nil,
	}, nil
}

// ResolverAprobacion cierra una aprobación pendiente con su desenlace. El estado
// solo puede ser `aprobada` o `declinada`; una aprobación ya resuelta no se toca.
func ResolverAprobacion(db *sql.DB, projectID, sessionID, id, estado string) error {
	if estado != "aprobada" && estado != "declinada" {
		return fmt.Errorf("store: estado de aprobación no válido: %s", estado)
	}
	if err := sesionEnProyecto(db, projectID, sessionID); err != nil {
		return err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	res, err := db.Exec(
		"UPDATE approvals SET status = ?, resolved_at = ? WHERE id = ? AND session_id = ? AND status = 'pendiente'",
		estado, now, id, sessionID,
	)
	if err != nil {
		return fmt.Errorf("store: no se pudo resolver la aprobación: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("store: la aprobación no existe o ya estaba resuelta")
	}
	return nil
}

// ListarAprobacionesPendientesProyecto lista aprobaciones pendientes de cualquier sesión del proyecto.
func ListarAprobacionesPendientesProyecto(db *sql.DB, projectID string) ([]Aprobacion, error) {
	rows, err := db.Query(
		`SELECT a.id, a.session_id, a.description, a.status, a.created_at, a.resolved_at 
		 FROM approvals a 
		 JOIN sessions s ON s.id = a.session_id 
		 WHERE s.project_id = ? AND a.status = 'pendiente' 
		 ORDER BY a.created_at ASC`,
		projectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var aprobaciones []Aprobacion
	for rows.Next() {
		var a Aprobacion
		var resolvedAt sql.NullString
		if err := rows.Scan(&a.ID, &a.SessionID, &a.Description, &a.Status, &a.CreatedAt, &resolvedAt); err != nil {
			return nil, err
		}
		if resolvedAt.Valid {
			a.ResolvedAt = &resolvedAt.String
		}
		aprobaciones = append(aprobaciones, a)
	}
	return aprobaciones, nil
}
