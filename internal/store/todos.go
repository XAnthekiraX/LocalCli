package store

import (
	"database/sql"
	"fmt"
	"time"
)

// Todo es un paso de la lista de la sesión. La jerarquía es de un nivel: un paso
// raíz lleva `PosicionPadre` nulo y una subtarea apunta a la posición de su tarea
// principal dentro de la misma sesión.
type Todo struct {
	Posicion      int
	PosicionPadre *int
	Contenido     string
	Estado        string
	Prioridad     string
}

// ListarTodos devuelve la lista de pasos de una sesión del proyecto, en orden.
func ListarTodos(db *sql.DB, projectID, sessionID string) ([]Todo, error) {
	if err := sesionEnProyecto(db, projectID, sessionID); err != nil {
		return nil, err
	}
	return leerTodos(db, sessionID)
}

// CrearTodo añade al final de la lista un paso suelto o una tarea principal con
// sus subtareas, sin tocar los que ya había, y devuelve la lista resultante. El
// grupo se inserta en una sola transacción: o queda entero, o no queda nada.
func CrearTodo(db *sql.DB, projectID, sessionID string, principal Todo, subtareas []Todo) ([]Todo, error) {
	if err := sesionEnProyecto(db, projectID, sessionID); err != nil {
		return nil, err
	}

	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var posicion int
	if err := tx.QueryRow("SELECT COUNT(*) FROM todos WHERE session_id = ?", sessionID).Scan(&posicion); err != nil {
		return nil, fmt.Errorf("store: no se pudo contar los pasos: %w", err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	if err := insertarTodo(tx, sessionID, posicion, nil, principal, now); err != nil {
		return nil, fmt.Errorf("store: no se pudo añadir el paso: %w", err)
	}
	for i, subtarea := range subtareas {
		padre := posicion
		if err := insertarTodo(tx, sessionID, posicion+1+i, &padre, subtarea, now); err != nil {
			return nil, fmt.Errorf("store: no se pudo añadir la subtarea: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return leerTodos(db, sessionID)
}

// ReemplazarTodos sustituye la lista entera de la sesión por la dada, en un solo
// paso atómico: o queda la nueva entera, o queda la anterior. La posición de cada
// elemento es su índice en la lista; `PosicionPadre` debe apuntar a un elemento
// raíz de esa misma lista.
func ReemplazarTodos(db *sql.DB, projectID, sessionID string, elementos []Todo) ([]Todo, error) {
	if err := sesionEnProyecto(db, projectID, sessionID); err != nil {
		return nil, err
	}
	for i := range elementos {
		elementos[i].Posicion = i
	}
	if err := validarJerarquia(elementos); err != nil {
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
		if err := insertarTodo(tx, sessionID, i, t.PosicionPadre, t, now); err != nil {
			return nil, fmt.Errorf("store: no se pudo escribir la lista: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return leerTodos(db, sessionID)
}

func insertarTodo(tx *sql.Tx, sessionID string, posicion int, padre *int, t Todo, now string) error {
	_, err := tx.Exec(
		`INSERT INTO todos (session_id, position, parent_position, content, status, priority, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		sessionID, posicion, padre, t.Contenido, t.Estado, t.Prioridad, now, now,
	)
	return err
}

func leerTodos(db *sql.DB, sessionID string) ([]Todo, error) {
	rows, err := db.Query(
		"SELECT position, parent_position, content, status, priority FROM todos WHERE session_id = ? ORDER BY position",
		sessionID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []Todo
	for rows.Next() {
		var t Todo
		var padre sql.NullInt64
		if err := rows.Scan(&t.Posicion, &padre, &t.Contenido, &t.Estado, &t.Prioridad); err != nil {
			return nil, err
		}
		if padre.Valid {
			p := int(padre.Int64)
			t.PosicionPadre = &p
		}
		lista = append(lista, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := validarJerarquia(lista); err != nil {
		return nil, err
	}
	return lista, nil
}

// validarJerarquia impone lo que SQLite no puede declarar con `ALTER TABLE`: que
// cada `PosicionPadre` apunte a una fila existente de la sesión y que la
// jerarquía no pase de un nivel.
func validarJerarquia(lista []Todo) error {
	porPosicion := make(map[int]int, len(lista))
	for i, t := range lista {
		porPosicion[t.Posicion] = i
	}
	for i, t := range lista {
		if t.PosicionPadre == nil {
			continue
		}
		padre, existe := porPosicion[*t.PosicionPadre]
		if !existe {
			return fmt.Errorf("store: la posición padre %d del paso %d no existe en la lista", *t.PosicionPadre, i)
		}
		if padre == i {
			return fmt.Errorf("store: el paso %d no puede ser padre de sí mismo", i)
		}
		if lista[padre].PosicionPadre != nil {
			return fmt.Errorf("store: la jerarquía de pasos es de un solo nivel: la posición %d ya es una subtarea", *t.PosicionPadre)
		}
	}
	return nil
}
