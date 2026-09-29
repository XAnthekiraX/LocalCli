package store

// flowcontext.go — el bloque de contexto de un flujo (migración 003).
//
// Fuente de verdad: ai/docs/specs/SPEC-MOTOR-FLUJOS.md §Bloque de contexto y
// ai/docs/database/01-schema/TABLES.md §flow_context. Cada etapa de un flujo con
// `bloque_contexto` deja aquí su aportación optimizada; la última etapa compone
// la entrega a partir de todo el bloque. Es estado de ejecución de la sesión: se
// reemplaza en cada ejecución del flujo y cae en cascada con su sesión.
//
// El orden es la `position`, que es el índice de la etapa en la secuencia del
// flujo; la clave única (session_id, flow, stage) hace idempotente volver a
// correr el mismo flujo.

import "database/sql"

// FlowContext es una fila de `flow_context`: la aportación de una etapa al
// bloque de contexto de un flujo.
type FlowContext struct {
	ID        string
	SessionID string
	Flow      string // nombre del flujo (p. ej. "resolver")
	Stage     string // id de la etapa
	StageName string // nombre visible de la etapa
	Position  int    // orden en la secuencia, 0-indexado
	Content   string // aportación optimizada
	CreatedAt string
}

// LimpiarBloque deja vacío el bloque de un flujo en una sesión. Se llama al
// arrancar una ejecución: el bloque nuevo no hereda las aportaciones del
// anterior.
func LimpiarBloque(db *sql.DB, sessionID, flujo string) error {
	_, err := db.Exec(`DELETE FROM flow_context WHERE session_id = ? AND flow = ?`, sessionID, flujo)
	return traducirError(err)
}

// GuardarEntradaBloque inserta o reemplaza la aportación de una etapa. La clave
// única (session_id, flow, stage) hace que re-ejecutar el flujo actualice la
// fila en vez de duplicarla.
func GuardarEntradaBloque(db *sql.DB, sessionID string, x *FlowContext) error {
	now := x.CreatedAt
	if now == "" {
		now = nowISO()
	}
	id := x.ID
	if id == "" {
		id = newID()
	}
	_, err := db.Exec(
		`INSERT INTO flow_context (id, session_id, flow, stage, stage_name, position, content, created_at)
 VALUES (?, ?, ?, ?, ?, ?, ?, ?)
 ON CONFLICT (session_id, flow, stage) DO UPDATE SET
   stage_name = excluded.stage_name,
   position   = excluded.position,
   content    = excluded.content,
   created_at = excluded.created_at`,
		id, sessionID, x.Flow, x.Stage, x.StageName, x.Position, x.Content, now,
	)
	if err != nil {
		return traducirError(err)
	}
	x.ID, x.SessionID, x.CreatedAt = id, sessionID, now
	return nil
}

// EntradasDelBloque devuelve las aportaciones de un flujo en una sesión, en el
// orden de la secuencia. Acelera idx_flow_context_unico.
func EntradasDelBloque(db *sql.DB, sessionID, flujo string) ([]FlowContext, error) {
	rows, err := db.Query(
		`SELECT id, session_id, flow, stage, stage_name, position, content, created_at
 FROM flow_context
 WHERE session_id = ? AND flow = ?
 ORDER BY position`, sessionID, flujo)
	if err != nil {
		return nil, traducirError(err)
	}
	defer rows.Close()
	var out []FlowContext
	for rows.Next() {
		var x FlowContext
		if err := rows.Scan(&x.ID, &x.SessionID, &x.Flow, &x.Stage, &x.StageName, &x.Position, &x.Content, &x.CreatedAt); err != nil {
			return nil, traducirError(err)
		}
		out = append(out, x)
	}
	return out, traducirError(rows.Err())
}
