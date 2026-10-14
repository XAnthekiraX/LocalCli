package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Sesion representa una sesión almacenada.
type Sesion struct {
	ID        string
	ProjectID string
	Name      string
	Layer     *string
	Status    string
	CreatedAt string
	UpdatedAt string
	MotorID   *string
	Modelo    *string
}

// CrearSesion crea una nueva sesión para el proyecto dado.
func CrearSesion(db *sql.DB, projectID, name string, layer *string) (*Sesion, error) {
	id := uuid.New().String()
	now := time.Now().UTC().Format(time.RFC3339)
	status := "inactiva"

	_, err := db.Exec(
		"INSERT INTO sessions (id, project_id, name, layer, status, created_at, updated_at, motor_id, modelo) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		id, projectID, name, layer, status, now, now, nil, nil,
	)
	if err != nil {
		return nil, fmt.Errorf("store: no se pudo crear sesión: %w", err)
	}

	return &Sesion{
		ID:        id,
		ProjectID: projectID,
		Name:      name,
		Layer:     layer,
		Status:    status,
		CreatedAt: now,
		UpdatedAt: now,
		MotorID:   nil,
		Modelo:    nil,
	}, nil
}

// ListarSesiones lista todas las sesiones de un proyecto, ordenadas por updated_at desc.
func ListarSesiones(db *sql.DB, projectID string) ([]Sesion, error) {
	rows, err := db.Query(
		"SELECT id, project_id, name, layer, status, created_at, updated_at, motor_id, modelo FROM sessions WHERE project_id = ? ORDER BY updated_at DESC",
		projectID,
	)
	if err != nil {
		return nil, fmt.Errorf("store: no se pudo listar sesiones: %w", err)
	}
	defer rows.Close()

	var sesiones []Sesion
	for rows.Next() {
		var s Sesion
		var layer, motorID, modelo sql.NullString
		if err := rows.Scan(&s.ID, &s.ProjectID, &s.Name, &layer, &s.Status, &s.CreatedAt, &s.UpdatedAt, &motorID, &modelo); err != nil {
			return nil, err
		}
		if layer.Valid {
			s.Layer = &layer.String
		}
		if motorID.Valid {
			s.MotorID = &motorID.String
		}
		if modelo.Valid {
			s.Modelo = &modelo.String
		}
		sesiones = append(sesiones, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return sesiones, nil
}

// ObtenerSesion obtiene una sesión específica del proyecto.
func ObtenerSesion(db *sql.DB, projectID, sessionID string) (*Sesion, error) {
	var s Sesion
	var layer, motorID, modelo sql.NullString
	err := db.QueryRow(
		"SELECT id, project_id, name, layer, status, created_at, updated_at, motor_id, modelo FROM sessions WHERE id = ? AND project_id = ?",
		sessionID, projectID,
	).Scan(&s.ID, &s.ProjectID, &s.Name, &layer, &s.Status, &s.CreatedAt, &s.UpdatedAt, &motorID, &modelo)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("store: sesión no encontrada: %w", err)
		}
		return nil, err
	}
	if layer.Valid {
		s.Layer = &layer.String
	}
	if motorID.Valid {
		s.MotorID = &motorID.String
	}
	if modelo.Valid {
		s.Modelo = &modelo.String
	}
	return &s, nil
}

// EliminarSesion elimina una sesión del proyecto.
func EliminarSesion(db *sql.DB, projectID, sessionID string) error {
	result, err := db.Exec("DELETE FROM sessions WHERE id = ? AND project_id = ?", sessionID, projectID)
	if err != nil {
		return fmt.Errorf("store: no se pudo eliminar sesión: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("store: sesión no encontrada")
	}
	return nil
}

// ActualizarEstadoSesion actualiza el estado de una sesión del proyecto.
func ActualizarEstadoSesion(db *sql.DB, projectID, sessionID, status string) error {
	_, err := db.Exec("UPDATE sessions SET status = ?, updated_at = ? WHERE id = ? AND project_id = ?", status, time.Now().UTC().Format(time.RFC3339), sessionID, projectID)
	if err != nil {
		return fmt.Errorf("store: no se pudo actualizar estado: %w", err)
	}
	return nil
}

// sesionEnProyecto comprueba que la sesión existe dentro del proyecto.
func sesionEnProyecto(db *sql.DB, projectID, sessionID string) error {
	var existe int
	if err := db.QueryRow("SELECT COUNT(*) FROM sessions WHERE id = ? AND project_id = ?", sessionID, projectID).Scan(&existe); err != nil {
		return err
	}
	if existe == 0 {
		return fmt.Errorf("store: sesión no encontrada en el proyecto")
	}
	return nil
}

// ActualizarNombreSesion actualiza el nombre de una sesión del proyecto.
func ActualizarNombreSesion(db *sql.DB, projectID, sessionID, name string) error {
	_, err := db.Exec("UPDATE sessions SET name = ?, updated_at = ? WHERE id = ? AND project_id = ?", name, time.Now().UTC().Format(time.RFC3339), sessionID, projectID)
	if err != nil {
		return fmt.Errorf("store: no se pudo actualizar nombre: %w", err)
	}
	return nil
}

// ResumenHistorial es el resumen compactado del historial de una sesión y el id
// del último mensaje que abarca. Un `HastaID` vacío con texto dice que el resumen
// cubre todo el historial hasta el momento de guardarlo.
type ResumenHistorial struct {
	Texto   string
	HastaID string
}

// ObtenerResumenHistorial lee el resumen compactado de una sesión del proyecto.
// Devuelve `false` cuando no hay resumen guardado (columnas NULL o texto vacío).
func ObtenerResumenHistorial(db *sql.DB, projectID, sessionID string) (ResumenHistorial, bool, error) {
	var texto, hasta sql.NullString
	err := db.QueryRow(
		"SELECT summary, summary_upto FROM sessions WHERE id = ? AND project_id = ?",
		sessionID, projectID,
	).Scan(&texto, &hasta)
	if err != nil {
		if err == sql.ErrNoRows {
			return ResumenHistorial{}, false, nil
		}
		return ResumenHistorial{}, false, fmt.Errorf("store: no se pudo leer el resumen: %w", err)
	}
	if !texto.Valid || texto.String == "" {
		return ResumenHistorial{}, false, nil
	}
	return ResumenHistorial{Texto: texto.String, HastaID: hasta.String}, true, nil
}

// GuardarResumenHistorial persiste el resumen compactado de una sesión y hasta
// qué mensaje abarca.
func GuardarResumenHistorial(db *sql.DB, projectID, sessionID, texto, hastaID string) error {
	_, err := db.Exec(
		"UPDATE sessions SET summary = ?, summary_upto = ? WHERE id = ? AND project_id = ?",
		texto, hastaID, sessionID, projectID,
	)
	if err != nil {
		return fmt.Errorf("store: no se pudo guardar el resumen: %w", err)
	}
	return nil
}

// LimpiarResumenHistorial borra el resumen compactado de una sesión: el historial
// vuelve a entregarse entero.
func LimpiarResumenHistorial(db *sql.DB, projectID, sessionID string) error {
	_, err := db.Exec(
		"UPDATE sessions SET summary = NULL, summary_upto = NULL WHERE id = ? AND project_id = ?",
		sessionID, projectID,
	)
	if err != nil {
		return fmt.Errorf("store: no se pudo limpiar el resumen: %w", err)
	}
	return nil
}
