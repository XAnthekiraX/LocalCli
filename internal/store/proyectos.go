package store

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

// ResolucionProyecto resuelve o crea la fila de projects para el path dado.
// Normaliza el path a absoluto y devuelve el ID del proyecto.
func ResolucionProyecto(db *sql.DB, path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("store: no se pudo normalizar el path: %w", err)
	}

	// Intentar buscar proyecto existente
	var id string
	err = db.QueryRow("SELECT id FROM projects WHERE path = ?", absPath).Scan(&id)
	if err == nil {
		// Actualizar updated_at
		_, err = db.Exec("UPDATE projects SET updated_at = ? WHERE id = ?", time.Now().UTC().Format(time.RFC3339), id)
		if err != nil {
			return "", fmt.Errorf("store: no se pudo actualizar proyecto: %w", err)
		}
		return id, nil
	}
	if err != sql.ErrNoRows {
		return "", fmt.Errorf("store: error al buscar proyecto: %w", err)
	}

	// Crear nuevo proyecto
	id = uuid.New().String()
	now := time.Now().UTC().Format(time.RFC3339)
	_, err = db.Exec("INSERT INTO projects (id, path, created_at, updated_at) VALUES (?, ?, ?, ?)", id, absPath, now, now)
	if err != nil {
		return "", fmt.Errorf("store: no se pudo crear proyecto: %w", err)
	}

	return id, nil
}
