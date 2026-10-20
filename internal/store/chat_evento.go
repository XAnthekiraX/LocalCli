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

// Tipos de bloque visual que acompañan a una línea del hilo
// (`chat_evento.bloque_tipo`): el diagrama ya renderizado o la diferencia de un
// cambio aplicado.
const (
	BloqueDiagrama   = "diagrama"
	BloqueDiferencia = "diferencia"
)

// Bloque es el bloque visual que se pinta bajo una línea del hilo. `Texto` llega
// ya renderizado —la TUI no lo interpreta— y `Tipo` dice cómo pintarlo. El cero
// (texto vacío) significa «sin bloque».
type Bloque struct {
	Texto string
	Tipo  string
}

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
	// Bloque es el bloque visual ya renderizado que se pinta bajo la línea
	// —un diagrama o una diferencia—; vacío si la línea no tiene.
	Bloque string
	// BloqueTipo dice cómo pintarlo (`diagrama` o `diferencia`); vacío sin bloque.
	BloqueTipo string
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
	return CrearChatEventoConBloque(db, projectID, sessionID, tipo, content, durationMs, Bloque{})
}

// CrearChatEventoConBloque guarda una línea del hilo con su bloque visual: el
// diagrama ya renderizado o la diferencia de un cambio. El bloque se persiste con
// la línea para que al retomar la sesión se pinte igual, pero no entra al
// contexto del modelo (EVENTS §3, DATA_FLOW §Creación).
func CrearChatEventoConBloque(db *sql.DB, projectID, sessionID, tipo, content string, durationMs *int, bloque Bloque) (*ChatEvento, error) {
	if tipo != ChatEventoProceso && tipo != ChatEventoHerramienta && tipo != ChatEventoRespuesta {
		return nil, fmt.Errorf("store: tipo de evento de chat inválido: %q", tipo)
	}
	if err := sesionEnProyecto(db, projectID, sessionID); err != nil {
		return nil, err
	}

	id := uuid.New().String()
	now := ahora()

	_, err := db.Exec(
		`INSERT INTO chat_evento (id, session_id, tipo, content, created_at, duration_ms, bloque, bloque_tipo)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		id, sessionID, tipo, content, now, durationMs, textoOpcional(bloque.Texto), textoOpcional(bloque.Tipo),
	)
	if err != nil {
		return nil, fmt.Errorf("store: no se pudo crear evento de chat: %w", err)
	}

	return &ChatEvento{
		ID:         id,
		SessionID:  sessionID,
		Tipo:       tipo,
		Content:    content,
		Bloque:     bloque.Texto,
		BloqueTipo: bloque.Tipo,
		CreatedAt:  now,
		DurationMs: durationMs,
	}, nil
}

// CrearChatEventoTool guarda una línea de herramienta con su ejecución completa:
// la línea para pintarla y los datos para reenviarla al modelo en los próximos
// turnos. Es best-effort para quien la llama: un fallo no debe tumbar el turno.
func CrearChatEventoTool(db *sql.DB, projectID, sessionID, content string, durationMs *int, e EjecucionHerramienta) (*ChatEvento, error) {
	return CrearChatEventoToolConBloque(db, projectID, sessionID, content, durationMs, Bloque{}, e)
}

// CrearChatEventoToolConBloque guarda una línea de herramienta con su ejecución
// y su bloque visual, si lo tiene.
func CrearChatEventoToolConBloque(db *sql.DB, projectID, sessionID, content string, durationMs *int, bloque Bloque, e EjecucionHerramienta) (*ChatEvento, error) {
	if err := sesionEnProyecto(db, projectID, sessionID); err != nil {
		return nil, err
	}

	id := uuid.New().String()
	now := ahora()

	_, err := db.Exec(
		`INSERT INTO chat_evento
		 (id, session_id, tipo, content, created_at, duration_ms, bloque, bloque_tipo,
		  call_id, herramienta, argumentos, salida, error, ok, truncado)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, sessionID, ChatEventoHerramienta, content, now, durationMs,
		textoOpcional(bloque.Texto), textoOpcional(bloque.Tipo),
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
		Bloque:     bloque.Texto,
		BloqueTipo: bloque.Tipo,
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

// textoOpcional guarda un texto vacío como NULL: el bloque ausente no es un
// bloque vacío, y el CHECK de `bloque_tipo` solo admite el vocabulario.
func textoOpcional(v string) any {
	if v == "" {
		return nil
	}
	return v
}

// ListarChatEventos lista las líneas guardadas de una sesión del proyecto, en
// orden de creación.
func ListarChatEventos(db *sql.DB, projectID, sessionID string) ([]ChatEvento, error) {
	if err := sesionEnProyecto(db, projectID, sessionID); err != nil {
		return nil, err
	}

	rows, err := db.Query(
		`SELECT id, session_id, tipo, content, created_at, duration_ms, bloque, bloque_tipo,
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
		`SELECT id, session_id, tipo, content, created_at, duration_ms, bloque, bloque_tipo,
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
	var bloque, bloqueTipo, callID, herramienta, argumentos, salida, fallo sql.NullString
	var ok, truncado sql.NullInt64
	if err := rows.Scan(
		&e.ID, &e.SessionID, &e.Tipo, &e.Content, &e.CreatedAt, &durationMs,
		&bloque, &bloqueTipo,
		&callID, &herramienta, &argumentos, &salida, &fallo, &ok, &truncado,
	); err != nil {
		return ChatEvento{}, err
	}
	e.Bloque = bloque.String
	e.BloqueTipo = bloqueTipo.String
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
		 (id, session_id, tipo, content, created_at, duration_ms, bloque, bloque_tipo,
		  call_id, herramienta, argumentos, salida, error, ok, truncado)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.ID, sessionID, e.Tipo, e.Content, e.CreatedAt, e.DurationMs,
		textoOpcional(e.Bloque), textoOpcional(e.BloqueTipo),
		callID, herramienta, argumentos, salida, fallo, ok, truncado,
	)
	if err != nil {
		return fmt.Errorf("store: no se pudo escribir una línea del hilo: %w", err)
	}
	return nil
}
