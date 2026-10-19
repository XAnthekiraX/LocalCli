package store

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

// Tipos de línea guardada del hilo (chat_evento.tipo): una etapa de un flujo, el
// cierre de una herramienta o la respuesta de una etapa con `respuesta_en_chat`.
const (
	ChatEventoProceso     = "proceso"
	ChatEventoHerramienta = "herramienta"
	// ChatEventoRespuesta es la respuesta completa de una etapa de flujo que
	// pidió verse en el chat: se pinta como bloque y se guarda para que el hilo
	// se recupere igual al retomar la sesión. Nunca viaja al modelo
	// (SPEC-ORQUESTADOR-FLUJOS §Visibilidad de una respuesta).
	ChatEventoRespuesta = "respuesta"
)

// ChatEvento es una línea de pantalla guardada de una sesión: una herramienta, un
// sub-proceso o la respuesta de una etapa. Las herramientas guardan además su
// ejecución (`Ejecucion`) para poder reenviarla al modelo en turnos posteriores;
// el resto de líneas solo se pintan.
type ChatEvento struct {
	ID        string
	SessionID string
	// Tipo es `proceso`, `herramienta` o `respuesta`.
	Tipo string
	// Content es la línea ya compuesta, tal como se vio en pantalla.
	Content string
	// CreatedAt distingue el orden dentro del hilo; alta resolución para
	// intercalarse con los mensajes del mismo turno.
	CreatedAt string
	// DurationMs es lo que tardó la herramienta; NULL en una etapa de flujo.
	DurationMs *int
	// Ejecucion es la llamada y su desenlace; nil en las líneas que no son una
	// herramienta ejecutada (sub-procesos y respuestas de etapa).
	Ejecucion *EjecucionHerramienta
}

// EjecucionHerramienta es lo que hace falta para reconstruir el par
// `assistant(tool_calls)` + `tool(resultado)` que el modelo ve en el turno en que
// la herramienta corrió. Se persiste junto a la línea para reenviarla después.
type EjecucionHerramienta struct {
	CallID      string
	Herramienta string
	Argumentos  string
	Salida      string
	Error       string
	OK          bool
	Truncado    bool
}

// CrearChatEvento guarda una línea del hilo de una sesión del proyecto. La línea
// de herramienta se guarda ya cerrada, con su duración; la de un flujo no lleva
// duración. Es best-effort para quien la llama: un fallo no debe tumbar el turno.
func CrearChatEvento(db *sql.DB, projectID, sessionID, tipo, content string, durationMs *int) (*ChatEvento, error) {
	if tipo != ChatEventoProceso && tipo != ChatEventoHerramienta && tipo != ChatEventoRespuesta {
		return nil, fmt.Errorf("store: tipo de evento de chat inválido: %q", tipo)
	}
	if err := sesionEnProyecto(db, projectID, sessionID); err != nil {
		return nil, err
	}

	id := uuid.New().String()
	now := ahora()

	_, err := db.Exec(
		"INSERT INTO chat_evento (id, session_id, tipo, content, created_at, duration_ms) VALUES (?, ?, ?, ?, ?, ?)",
		id, sessionID, tipo, content, now, durationMs,
	)
	if err != nil {
		return nil, fmt.Errorf("store: no se pudo crear evento de chat: %w", err)
	}

	return &ChatEvento{
		ID:         id,
		SessionID:  sessionID,
		Tipo:       tipo,
		Content:    content,
		CreatedAt:  now,
		DurationMs: durationMs,
	}, nil
}

// CrearChatEventoTool guarda una línea de herramienta con su ejecución completa:
// la línea para pintarla y los datos para reenviarla al modelo en los próximos
// turnos. Es best-effort para quien la llama: un fallo no debe tumbar el turno.
func CrearChatEventoTool(db *sql.DB, projectID, sessionID, content string, durationMs *int, e EjecucionHerramienta) (*ChatEvento, error) {
	if err := sesionEnProyecto(db, projectID, sessionID); err != nil {
		return nil, err
	}

	id := uuid.New().String()
	now := ahora()

	_, err := db.Exec(
		`INSERT INTO chat_evento
		 (id, session_id, tipo, content, created_at, duration_ms, call_id, herramienta, argumentos, salida, error, ok, truncado)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, sessionID, ChatEventoHerramienta, content, now, durationMs,
		e.CallID, e.Herramienta, e.Argumentos, e.Salida, e.Error, booleano(e.OK), booleano(e.Truncado),
	)
	if err != nil {
		return nil, fmt.Errorf("store: no se pudo crear evento de chat: %w", err)
	}

	return &ChatEvento{
		ID:         id,
		SessionID:  sessionID,
		Tipo:       ChatEventoHerramienta,
		Content:    content,
		CreatedAt:  now,
		DurationMs: durationMs,
		Ejecucion:  &e,
	}, nil
}

// booleano guarda un bool como entero 0/1 para SQLite.
func booleano(v bool) int {
	if v {
		return 1
	}
	return 0
}

// ListarChatEventos lista las líneas guardadas de una sesión del proyecto, en
// orden de creación.
func ListarChatEventos(db *sql.DB, projectID, sessionID string) ([]ChatEvento, error) {
	if err := sesionEnProyecto(db, projectID, sessionID); err != nil {
		return nil, err
	}

	rows, err := db.Query(
		`SELECT id, session_id, tipo, content, created_at, duration_ms,
		        call_id, herramienta, argumentos, salida, error, ok, truncado
		 FROM chat_evento WHERE session_id = ? ORDER BY created_at, rowid`,
		sessionID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var eventos []ChatEvento
	for rows.Next() {
		e, err := leerChatEvento(rows)
		if err != nil {
			return nil, err
		}
		eventos = append(eventos, e)
	}
	return eventos, rows.Err()
}

// ListarEjecucionesHerramienta lista las herramientas ejecutadas de una sesión
// que tienen llamada asociada, para reenviarlas al modelo. Las líneas antiguas
// —anteriores a la persistencia de la ejecución— no tienen `call_id` y se omiten.
func ListarEjecucionesHerramienta(db *sql.DB, projectID, sessionID string) ([]ChatEvento, error) {
	if err := sesionEnProyecto(db, projectID, sessionID); err != nil {
		return nil, err
	}

	rows, err := db.Query(
		`SELECT id, session_id, tipo, content, created_at, duration_ms,
		        call_id, herramienta, argumentos, salida, error, ok, truncado
		 FROM chat_evento
		 WHERE session_id = ? AND tipo = ?
		   AND call_id IS NOT NULL AND call_id <> ''
		 ORDER BY created_at, rowid`,
		sessionID, ChatEventoHerramienta,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var eventos []ChatEvento
	for rows.Next() {
		e, err := leerChatEvento(rows)
		if err != nil {
			return nil, err
		}
		eventos = append(eventos, e)
	}
	return eventos, rows.Err()
}

// leerChatEvento arma una línea desde una fila con todas sus columnas.
func leerChatEvento(rows *sql.Rows) (ChatEvento, error) {
	var e ChatEvento
	var durationMs sql.NullInt64
	var callID, herramienta, argumentos, salida, fallo sql.NullString
	var ok, truncado sql.NullInt64
	if err := rows.Scan(
		&e.ID, &e.SessionID, &e.Tipo, &e.Content, &e.CreatedAt, &durationMs,
		&callID, &herramienta, &argumentos, &salida, &fallo, &ok, &truncado,
	); err != nil {
		return ChatEvento{}, err
	}
	e.DurationMs = enteroOpcional(durationMs)
	if callID.Valid && callID.String != "" {
		e.Ejecucion = &EjecucionHerramienta{
			CallID:      callID.String,
			Herramienta: herramienta.String,
			Argumentos:  argumentos.String,
			Salida:      salida.String,
			Error:       fallo.String,
			OK:          ok.Valid && ok.Int64 != 0,
			Truncado:    truncado.Valid && truncado.Int64 != 0,
		}
	}
	return e, nil
}

// insertarChatEvento escribe una línea ya construida en la sesión indicada. Lo
// usan rehacer y bifurcar, que conservan la ejecución de la herramienta junto a
// la línea para que el historial reconstruido siga teniendo la llamada.
func insertarChatEvento(tx *sql.Tx, sessionID string, e ChatEvento) error {
	var callID, herramienta, argumentos, salida, fallo any
	var ok, truncado any
	if e.Ejecucion != nil {
		callID = e.Ejecucion.CallID
		herramienta = e.Ejecucion.Herramienta
		argumentos = e.Ejecucion.Argumentos
		salida = e.Ejecucion.Salida
		fallo = e.Ejecucion.Error
		ok = booleano(e.Ejecucion.OK)
		truncado = booleano(e.Ejecucion.Truncado)
	}
	_, err := tx.Exec(
		`INSERT INTO chat_evento
		 (id, session_id, tipo, content, created_at, duration_ms, call_id, herramienta, argumentos, salida, error, ok, truncado)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.ID, sessionID, e.Tipo, e.Content, e.CreatedAt, e.DurationMs,
		callID, herramienta, argumentos, salida, fallo, ok, truncado,
	)
	if err != nil {
		return fmt.Errorf("store: no se pudo escribir una línea del hilo: %w", err)
	}
	return nil
}
