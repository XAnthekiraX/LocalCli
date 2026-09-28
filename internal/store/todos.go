package store

// todos.go — la lista de pasos de la sesión (migración 002-crear-todo).
//
// Fuente de verdad: TABLES.md §7 (la tabla `todos`) y SPEC-TOOLS (la lista de
// pasos del agente). Es estado de ejecución de la sesión, no un documento del
// proyecto: se reescribe entera en cada `actualizar_todo` y cae en cascada con
// su sesión.
//
// El orden es la `position`, que es el índice del elemento en la lista que manda
// el modelo; no hay identificadores de fila que puedan quedar obsoletos.

import "database/sql"

// Todo es un paso de la lista de la sesión. El estado y la prioridad viajan ya
// validados por la capa universal (`tools`); la base los vuelve a comprobar con
// sus CHECK.
type Todo struct {
	Contenido string
	Estado    string
	Prioridad string // "" se guarda como "media"
}

// prioridadPorDefecto es la prioridad de un paso que no la declara.
const prioridadPorDefecto = "media"

// LeerTodos devuelve la lista de pasos de una sesión, en el orden guardado.
func LeerTodos(db *sql.DB, sessionID string) ([]Todo, error) {
	rows, err := db.Query(
		`SELECT content, status, priority FROM todos WHERE session_id = ? ORDER BY position`,
		sessionID)
	if err != nil {
		return nil, traducirError(err)
	}
	defer rows.Close()
	var out []Todo
	for rows.Next() {
		var t Todo
		if err := rows.Scan(&t.Contenido, &t.Estado, &t.Prioridad); err != nil {
			return nil, traducirError(err)
		}
		out = append(out, t)
	}
	return out, traducirError(rows.Err())
}

// ReemplazarTodos sustituye la lista entera de una sesión: borra lo que hubiera
// e inserta la lista nueva numerada en el mismo paso atómico. Es lo que hace
// idempotente a `actualizar_todo`: no hay deltas ni estados a medias.
func ReemplazarTodos(db *sql.DB, sessionID string, items []Todo) error {
	return EjecutarTX(db, func(tx *sql.Tx) error {
		return reemplazarTodos(tx, sessionID, items)
	})
}

// reemplazarTodos es la versión que acepta un ejecutor, para poder correr dentro
// de la transacción que abre ReemplazarTodos.
func reemplazarTodos(e ejecutor, sessionID string, items []Todo) error {
	if _, err := e.Exec(`DELETE FROM todos WHERE session_id = ?`, sessionID); err != nil {
		return traducirError(err)
	}
	now := nowISO()
	for i, it := range items {
		prioridad := it.Prioridad
		if prioridad == "" {
			prioridad = prioridadPorDefecto
		}
		if _, err := e.Exec(
			`INSERT INTO todos (session_id, position, content, status, priority, created_at, updated_at)
 VALUES (?, ?, ?, ?, ?, ?, ?)`,
			sessionID, i, it.Contenido, it.Estado, prioridad, now, now,
		); err != nil {
			return traducirError(err)
		}
	}
	return nil
}
