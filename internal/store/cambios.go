package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Cambio es una escritura registrada en `change_history`.
type Cambio struct {
	ID        string
	SessionID *string
	Operacion string
	Ruta      string
	Antes     *string
	Despues   *string
}

// RegistrarCambio guarda una escritura aplicada con su antes y su después.
// `sessionID` vacío deja la fila sin sesión (el cambio sobrevive a su borrado).
func RegistrarCambio(db *sql.DB, sessionID, operacion, ruta string, antes, despues *string) error {
	var sid any
	if sessionID != "" {
		sid = sessionID
	}
	_, err := db.Exec(
		`INSERT INTO change_history (id, session_id, operation, file_path, before_content, after_content, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		uuid.New().String(), sid, operacion, ruta, antes, despues, time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("store: no se pudo registrar el cambio: %w", err)
	}
	return nil
}
