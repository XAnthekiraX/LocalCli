package store

import (
	"database/sql"
	"fmt"
	"time"
)

// Todo es un paso de la lista de la sesión.
type Todo struct {
	Posicion  int
	Contenido string
	Estado    string
	Prioridad string
}

// ListarTodos devuelve la lista de pasos de una sesión del proyecto, en orden.
func ListarTodos(db *sql.DB, projectID, sessionID string) ([]Todo, error) {
	if err := sesionEnProyecto(db, projectID, sessionID); err != nil {
		return nil, err
	}
	return leerTodos(db, sessionID)
}

// CrearTodo añade un paso al final de la lista, sin tocar los que ya había, y
// devuelve la lista resultante.
func CrearTodo(db *sql.DB, projectID, sessionID, contenido, estado, prioridad string) ([]Todo, error) {
	if err := sesionEnProyecto(db, projectID, sessionID); err != nil {
		return nil, err
	}

	var posicion int
	if err := db.QueryRow("SELECT COUNT(*) FROM todos WHERE session_id = ?", sessionID).Scan(&posicion); err != nil {
		return nil, fmt.Errorf("store: no se pudo contar los pasos: %w", err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := db.Exec(
		`INSERT INTO todos (session_id, position, content, status, priority, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		sessionID, posicion, contenido, estado, prioridad, now, now,
	); err != nil {
		return nil, fmt.Errorf("store: no se pudo añadir el paso: %w", err)
	}
	return leerTodos(db, sessionID)
}

// ReemplazarTodos sustituye la lista entera de la sesión por la dada, en un solo
// paso atómico: o queda la nueva entera, o queda la anterior.
func ReemplazarTodos(db *sql.DB, projectID, sessionID string, elementos []Todo) ([]Todo, error) {
	if err := sesionEnProyecto(db, projectID, sessionID); err != nil {
		return nil, err
	}

	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM todos WHERE session_id = ?", sessionID); err != nil {
		return nil, fmt.Errorf("store: no se pudo vaciar la lista: %w", err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	for i, t := range elementos {
		if _, err := tx.Exec(
			`INSERT INTO todos (session_id, position, content, status, priority, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?)`,
			sessionID, i, t.Contenido, t.Estado, t.Prioridad, now, now,
		); err != nil {
			return nil, fmt.Errorf("store: no se pudo escribir la lista: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return leerTodos(db, sessionID)
}

func leerTodos(db *sql.DB, sessionID string) ([]Todo, error) {
	rows, err := db.Query(
		"SELECT position, content, status, priority FROM todos WHERE session_id = ? ORDER BY position",
		sessionID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []Todo
	for rows.Next() {
		var t Todo
		if err := rows.Scan(&t.Posicion, &t.Contenido, &t.Estado, &t.Prioridad); err != nil {
			return nil, err
		}
		lista = append(lista, t)
	}
	return lista, rows.Err()
}
