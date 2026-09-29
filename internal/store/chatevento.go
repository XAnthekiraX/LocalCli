// chatevento.go — el hilo de procesamiento del chat (migración 004).
//
// Fuente de verdad: ai/docs/database/01-schema/TABLES.md (la tabla
// `chat_evento`) y ai/docs/database/02-rules/DATA_FLOW.md (qué se guarda de un
// turno). Son las líneas que la TUI pinta además de la conversación —el
// sub-proceso de una etapa y las líneas de herramienta— guardadas como parte
// del hilo.
//
// No son contexto: el contexto que se entrega al modelo se arma solo de
// `messages` (HistorialSesion, en messages.go). Por eso viven en su propia
// tabla: la frontera «se muestra / se envía al modelo» es la frontera de la
// tabla, no un filtro que alguien pueda olvidar.
package store

import "database/sql"

// Tipos de línea de procesamiento (chat_evento.tipo). Cerrado por CHECK.
const (
	ChatTipoProceso     = "proceso"     // un sub-proceso de un flujo
	ChatTipoHerramienta = "herramienta" // una línea de herramienta
)

// ChatEvento es una fila de chat_evento (TABLES.md §3).
type ChatEvento struct {
	ID        string
	SessionID string
	Tipo      string // "proceso" | "herramienta"
	Content   string
	CreatedAt string
}

// LineaChat es una línea del hilo de una sesión, ya fusionada para la vista:
// un mensaje de la conversación o una línea de procesamiento, en orden.
type LineaChat struct {
	Rol          string // "user" | "agent" | "proceso"
	Content      string
	Razonamiento string // "" si no hay
	DuracionMS   int    // -1 = no se midió
	CreatedAt    string
}

// InsertarChatEvento guarda una línea de procesamiento del hilo.
func InsertarChatEvento(db *sql.DB, e *ChatEvento) error {
	now := e.CreatedAt
	if now == "" {
		now = nowISO()
	}
	id := e.ID
	if id == "" {
		id = newID()
	}
	_, err := db.Exec(
		`INSERT INTO chat_evento (id, session_id, tipo, content, created_at)
		 VALUES (?, ?, ?, ?, ?)`,
		id, e.SessionID, e.Tipo, e.Content, now,
	)
	if err != nil {
		return traducirError(err)
	}
	e.ID, e.CreatedAt = id, now
	return nil
}

// ChatEventosSesion lee las líneas de procesamiento de una sesión, en orden.
func ChatEventosSesion(db *sql.DB, sessionID string) ([]ChatEvento, error) {
	rows, err := db.Query(
		`SELECT id, session_id, tipo, content, created_at
		 FROM chat_evento WHERE session_id = ?
		 ORDER BY created_at, rowid`, sessionID)
	if err != nil {
		return nil, traducirError(err)
	}
	defer rows.Close()
	var out []ChatEvento
	for rows.Next() {
		var e ChatEvento
		if err := rows.Scan(&e.ID, &e.SessionID, &e.Tipo, &e.Content, &e.CreatedAt); err != nil {
			return nil, traducirError(err)
		}
		out = append(out, e)
	}
	return out, traducirError(rows.Err())
}

// HiloSesion devuelve el hilo completo de una sesión para pintarlo: los turnos
// de la conversación y las líneas de procesamiento, fusionados en orden. La
// conversación sale de `messages`; el procesamiento, de `chat_evento`.
//
// El orden es por `created_at` (segundos) y, dentro del mismo segundo, por
// `fuente` (0 = conversación, 1 = procesamiento) y luego por `rowid`: sin el
// desempate, dos filas del mismo segundo de tablas distintas quedarían en un
// orden arbitrario. La restricción de segundos es la del esquema (SCHEMA.md §1).
func HiloSesion(db *sql.DB, sessionID string) ([]LineaChat, error) {
	rows, err := db.Query(
		`SELECT m.role AS rol, m.content AS contenido,
		        COALESCE(r.content, '') AS razonamiento,
		        COALESCE(m.duration_ms, -1) AS duracion,
		        m.created_at AS creado, 0 AS fuente, m.rowid AS seq
		 FROM messages m
		 LEFT JOIN reasoning r ON r.message_id = m.id
		 WHERE m.session_id = ?
		 UNION ALL
		 SELECT 'proceso' AS rol, e.content AS contenido, '' AS razonamiento,
		        -1 AS duracion, e.created_at AS creado, 1 AS fuente, e.rowid AS seq
		 FROM chat_evento e
		 WHERE e.session_id = ?
		 ORDER BY creado, fuente, seq`, sessionID, sessionID)
	if err != nil {
		return nil, traducirError(err)
	}
	defer rows.Close()
	var out []LineaChat
	for rows.Next() {
		var (
			l  LineaChat
			du any
			// Columnas de orden: se leen para consumir la fila, no se usan.
			creado string
			fuente int
			seq    int64
		)
		if err := rows.Scan(&l.Rol, &l.Content, &l.Razonamiento, &du, &creado, &fuente, &seq); err != nil {
			return nil, traducirError(err)
		}
		l.DuracionMS = intNull(du)
		l.CreatedAt = creado
		out = append(out, l)
	}
	return out, traducirError(rows.Err())
}

// Chat es la fachada de la tabla chat_evento y del hilo para el arranque, que
// no importa `database/sql` (invariante de DECISIONS.md).
type Chat struct{ db *sql.DB }

// NuevoChat envuelve la conexión para operar sobre el hilo de procesamiento.
func NuevoChat(db *sql.DB) *Chat { return &Chat{db: db} }

// Registrar guarda una línea de procesamiento de la sesión.
func (c *Chat) Registrar(sessionID, tipo, content string) error {
	return InsertarChatEvento(c.db, &ChatEvento{SessionID: sessionID, Tipo: tipo, Content: content})
}

// Hilo devuelve el hilo fusionado (conversación + procesamiento) de una sesión.
func (c *Chat) Hilo(sessionID string) ([]LineaChat, error) {
	return HiloSesion(c.db, sessionID)
}
