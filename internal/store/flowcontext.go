package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// AporteFlow es la contribución de una etapa al bloque de contexto de su flujo.
type AporteFlow struct {
	Etapa     string
	Nombre    string
	Posicion  int
	Contenido string
}

// GuardarFlowContext guarda (o reemplaza) la contribución de una etapa al bloque
// de contexto de un flujo de la sesión.
func GuardarFlowContext(db *sql.DB, projectID, sessionID, flujo, etapa, nombre string, posicion int, contenido string) error {
	if err := sesionEnProyecto(db, projectID, sessionID); err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := db.Exec(
		`INSERT INTO flow_context (id, session_id, flow, stage, stage_name, position, content, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT (session_id, flow, stage) DO UPDATE SET
		   stage_name = excluded.stage_name,
		   position   = excluded.position,
		   content    = excluded.content`,
		uuid.New().String(), sessionID, flujo, etapa, nombre, posicion, contenido, now,
	)
	if err != nil {
		return fmt.Errorf("store: no se pudo guardar el bloque del flujo: %w", err)
	}
	return nil
}

// ListarFlowContext devuelve las contribuciones de un flujo, en orden de etapa.
func ListarFlowContext(db *sql.DB, projectID, sessionID, flujo string) ([]AporteFlow, error) {
	if err := sesionEnProyecto(db, projectID, sessionID); err != nil {
		return nil, err
	}
	rows, err := db.Query(
		`SELECT stage, stage_name, position, content FROM flow_context
		 WHERE session_id = ? AND flow = ? ORDER BY position`,
		sessionID, flujo,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var aportes []AporteFlow
	for rows.Next() {
		var a AporteFlow
		if err := rows.Scan(&a.Etapa, &a.Nombre, &a.Posicion, &a.Contenido); err != nil {
			return nil, err
		}
		aportes = append(aportes, a)
	}
	return aportes, rows.Err()
}

// VaciarFlowContext deja el bloque de un flujo en blanco: cada corrida no hereda
// la anterior.
func VaciarFlowContext(db *sql.DB, projectID, sessionID, flujo string) error {
	if err := sesionEnProyecto(db, projectID, sessionID); err != nil {
		return err
	}
	if _, err := db.Exec("DELETE FROM flow_context WHERE session_id = ? AND flow = ?", sessionID, flujo); err != nil {
		return fmt.Errorf("store: no se pudo vaciar el bloque del flujo: %w", err)
	}
	return nil
}
