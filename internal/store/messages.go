package store

import (
	"database/sql"
	"fmt"
	"time"
)

// Message es una fila de messages (TABLES.md §3). El mensaje es inmutable una
// vez completo (DECISIONS.md): no hay función de actualización aquí, a
// propósito.
type Message struct {
	ID           string
	SessionID    string
	Role         string // "user" | "agent"
	Content      string
	InputTokens  int // -1 = NULL ("el modelo no lo reporta")
	OutputTokens int // -1 = NULL
	CreatedAt    string
}

// InsertarMensaje inserta un turno de conversación. Los tokens llegan como -1
// cuando el modelo no los reporta: se guardan NULL, nunca 0 ni cadena vacía
// (TABLES.md: "NULL significa 'no lo sé', 0 significa 'cero'").
func InsertarMensaje(db *sql.DB, m *Message) error {
	return insertarMensaje(db, m)
}

// insertarMensaje acepta cualquier ejecutor (DB o Tx).
func insertarMensaje(e ejecutor, m *Message) error {
	now := m.CreatedAt
	if now == "" {
		now = nowISO()
	}
	id := m.ID
	if id == "" {
		id = newID()
	}
	_, err := e.Exec(
		`INSERT INTO messages (id, session_id, role, content, input_tokens, output_tokens, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		id, m.SessionID, m.Role, m.Content, nullInt(m.InputTokens), nullInt(m.OutputTokens), now,
	)
	if err != nil {
		return traducirError(err)
	}
	m.ID, m.CreatedAt = id, now
	return nil
}

// HistorialSesion carga la conversación de una sesión en orden cronológico con
// el razonamiento de cada mensaje de agente: la consulta documentada en
// QUERIES.md §1 ("Historial de una sesión, en orden"), con LEFT JOIN porque
// "un mensaje de agente puede no tener razonamiento". Acelera
// idx_messages_session_created.
type MensajeConRazonamiento struct {
	Message
	Reasoning string // "" si el mensaje no tiene razonamiento
}

func HistorialSesion(db *sql.DB, sessionID string) ([]MensajeConRazonamiento, error) {
	rows, err := db.Query(
		`SELECT m.id, m.session_id, m.role, m.content, m.input_tokens, m.output_tokens, m.created_at,
		        r.content
		 FROM messages m
		 LEFT JOIN reasoning r ON r.message_id = m.id
		 WHERE m.session_id = ?
		 -- rowid desempata mensajes insertados dentro del mismo segundo
		-- (created_at es RFC3339 con precisión de segundos): el orden de
		-- inserción es el orden del historial.
		ORDER BY m.created_at, m.rowid`, sessionID)
	if err != nil {
		return nil, traducirError(err)
	}
	defer rows.Close()
	var out []MensajeConRazonamiento
	for rows.Next() {
		var (
			m    MensajeConRazonamiento
			in   any
			out_ any
			rz   sql.NullString
		)
		err := rows.Scan(&m.ID, &m.SessionID, &m.Role, &m.Content, &in, &out_, &m.CreatedAt, &rz)
		if err != nil {
			return nil, traducirError(err)
		}
		m.InputTokens = intNull(in)
		m.OutputTokens = intNull(out_)
		m.Reasoning = rz.String
		out = append(out, m)
	}
	return out, traducirError(rows.Err())
}

// UltimoMensajeDeSesión pide solo el último mensaje con
// ORDER BY created_at DESC LIMIT 1 (QUERIES.md §5: "No cargar el historial
// completo de una sesión para mostrar la última línea"). Sin JOIN de
// razonamiento: §5 también prohíbe traerlo si no se va a mostrar.
func UltimoMensaje(db *sql.DB, sessionID string) (*Message, error) {
	row := db.QueryRow(
		`SELECT id, session_id, role, content, input_tokens, output_tokens, created_at
		 FROM messages WHERE session_id = ? ORDER BY created_at DESC LIMIT 1`, sessionID)
	var (
		m        Message
		in, outV any
	)
	err := row.Scan(&m.ID, &m.SessionID, &m.Role, &m.Content, &in, &outV, &m.CreatedAt)
	if err != nil {
		return nil, traducirError(err)
	}
	m.InputTokens = intNull(in)
	m.OutputTokens = intNull(outV)
	return &m, nil
}

// RazonamientoPersistido es el límite de frecuencia con que el streaming
// escribe el razonamiento acumulado: cada 200 ms y al terminar la respuesta
// (DECISIONS.md, fila "Razonamiento en streaming"). Un UpsertRazonamiento por
// token está descartado; este helper lo hace esperable para el llamador.
const RazonamientoPersistido = 200 * time.Millisecond

// UpsertRazonamiento crea o actualiza la única fila de reasoning de un mensaje
// con el texto acumulado hasta ahora (DATA_FLOW.md §"Acumulación del
// razonamiento": "No se inserta una fila por token: es una fila por mensaje
// que crece"). El UNIQUE idx_reasoning_message garantiza máximo uno por
// mensaje; el INSERT ... ON CONFLICT lo mantiene idempotente.
func UpsertRazonamiento(db *sql.DB, messageID, content string) error {
	return upsertRazonamiento(db, messageID, content)
}

// upsertRazonamiento acepta cualquier ejecutor (DB o Tx).
func upsertRazonamiento(e ejecutor, messageID, content string) error {
	_, err := e.Exec(
		`INSERT INTO reasoning (id, message_id, content, created_at)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT(message_id) DO UPDATE SET content = excluded.content`,
		newID(), messageID, content, nowISO(),
	)
	if err != nil {
		return traducirError(err)
	}
	return nil
}

// RazonamientoDe lee el razonamiento de un mensaje concreto (patrón de
// QUERIES.md §2 "Leer por sesión"). Devuelve "" sin error si el mensaje no
// tiene razonamiento.
func RazonamientoDe(db *sql.DB, messageID string) (string, error) {
	var content string
	err := db.QueryRow(`SELECT content FROM reasoning WHERE message_id = ?`, messageID).Scan(&content)
	if errorsIsNoRows(err) {
		return "", nil
	}
	if err != nil {
		return "", traducirError(err)
	}
	return content, nil
}

func errorsIsNoRows(err error) bool {
	return err == sql.ErrNoRows || err == ErrNoEncontrado
}

// CerrarTurnoAgente es la escritura multi-tabla que fija DATA_FLOW.md al
// terminar una respuesta: razonamiento final + mensaje del agente + estado de
// la sesión, todo en la misma transacción. Si algo falla a mitad, no queda
// ninguna de las tres (rollback en EjecutarTX).
//
// razonamiento vacío significa "el modelo no devolvió razonamiento": no se
// crea fila en reasoning (BUSINESS_RULES.md).
func CerrarTurnoAgente(db *sql.DB, sessionID, contenido string, inputTokens, outputTokens int, razonamiento string, estadoSesion string) (*Message, error) {
	m := &Message{
		SessionID:    sessionID,
		Role:         "agent",
		Content:      contenido,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
	}
	err := EjecutarTX(db, func(tx *sql.Tx) error {
		if err := insertarMensaje(tx, m); err != nil {
			return err
		}
		if razonamiento != "" {
			if err := upsertRazonamiento(tx, m.ID, razonamiento); err != nil {
				return err
			}
		}
		if estadoSesion != "" {
			s, err := obtenerSesionTX(tx, sessionID)
			if err != nil {
				return err
			}
			if !ValidarTransicionSesion(s.Status, estadoSesion) {
				return fmt.Errorf("%w: sesión %s no puede pasar de %q a %q", ErrEstadoIlegal, sessionID, s.Status, estadoSesion)
			}
			res, err := tx.Exec(`UPDATE sessions SET status = ?, updated_at = ? WHERE id = ?`,
				estadoSesion, nowISO(), sessionID)
			if err != nil {
				return traducirError(err)
			}
			if err := filasAfectadas(res, 1, fmt.Sprintf("sesión %s", sessionID)); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return m, nil
}

// ejecutor abstracta lo común entre *sql.DB y *sql.Tx para que los helpers de
// inserción funcionen dentro y fuera de transacciones.
type ejecutor interface {
	Exec(query string, args ...any) (sql.Result, error)
	QueryRow(query string, args ...any) *sql.Row
}

func obtenerSesionTX(e ejecutor, id string) (*Session, error) {
	row := e.QueryRow(
		`SELECT id, name, COALESCE(layer, ''), status, created_at, updated_at
		 FROM sessions WHERE id = ?`, id)
	return escanearSesion(row)
}
