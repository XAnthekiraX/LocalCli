package store

import "database/sql"

// initSchema inicializa el esquema v10 de la base de datos global.
func initSchema(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Schema v10
	_, err = tx.Exec(`
CREATE TABLE projects (
    id         TEXT PRIMARY KEY NOT NULL,
    path       TEXT NOT NULL UNIQUE,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
CREATE TABLE sessions (
    id         TEXT PRIMARY KEY NOT NULL,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE ON UPDATE CASCADE,
    name       TEXT NOT NULL,
    layer      TEXT CHECK (layer IN ('backend', 'frontend')),
    status     TEXT NOT NULL DEFAULT 'inactiva'
               CHECK (status IN ('inactiva', 'trabajando', 'esperando_permiso',
                                  'terminada', 'error')),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    modelo     TEXT,
    summary      TEXT,
    summary_upto TEXT
);
`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
CREATE TABLE messages (
    id           TEXT PRIMARY KEY NOT NULL,
    session_id   TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE ON UPDATE CASCADE,
    role         TEXT NOT NULL CHECK (role IN ('user', 'agent')),
    content      TEXT NOT NULL,
    input_tokens  INTEGER CHECK (input_tokens IS NULL OR input_tokens >= 0),
    output_tokens INTEGER CHECK (output_tokens IS NULL OR output_tokens >= 0),
    created_at   TEXT NOT NULL,
    duration_ms  INTEGER
);
`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
CREATE TABLE reasoning (
    id         TEXT PRIMARY KEY NOT NULL,
    message_id TEXT NOT NULL UNIQUE REFERENCES messages(id) ON DELETE CASCADE ON UPDATE CASCADE,
    content    TEXT NOT NULL,
    created_at TEXT NOT NULL
);
`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
CREATE TABLE approvals (
    id          TEXT PRIMARY KEY NOT NULL,
    session_id  TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE ON UPDATE CASCADE,
    description TEXT NOT NULL,
    status      TEXT NOT NULL DEFAULT 'pendiente'
                CHECK (status IN ('pendiente', 'aprobada', 'declinada', 'obsoleta')),
    created_at  TEXT NOT NULL,
    resolved_at TEXT,
    CHECK ((status = 'pendiente') = (resolved_at IS NULL))
);
`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
CREATE TABLE context_audit (
    id         TEXT PRIMARY KEY NOT NULL,
    session_id TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE ON UPDATE CASCADE,
    stage      TEXT NOT NULL,
    document   TEXT NOT NULL,
    decision   TEXT NOT NULL CHECK (decision IN ('incluido', 'descartado')),
    reason     TEXT,
    tokens     INTEGER CHECK (tokens IS NULL OR tokens >= 0),
    created_at TEXT NOT NULL,
    CHECK ((decision = 'descartado') = (reason IS NOT NULL))
);
`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
CREATE TABLE change_history (
    id             TEXT PRIMARY KEY NOT NULL,
    session_id     TEXT REFERENCES sessions(id) ON DELETE SET NULL ON UPDATE CASCADE,
    operation      TEXT NOT NULL
                   CHECK (operation IN ('crear_archivo', 'escribir_archivo',
                                        'editar_archivo', 'eliminar_archivo',
                                        'crear_carpeta', 'eliminar_carpeta')),
    file_path      TEXT NOT NULL,
    before_content TEXT,
    after_content  TEXT,
    created_at     TEXT NOT NULL,
    CHECK (
        (operation IN ('crear_archivo', 'crear_carpeta') AND before_content IS NULL)
        OR (operation IN ('eliminar_archivo', 'eliminar_carpeta') AND after_content IS NULL)
        OR (operation IN ('escribir_archivo', 'editar_archivo')
            AND before_content IS NOT NULL AND after_content IS NOT NULL)
    )
);
`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
CREATE TABLE todos (
    session_id TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE ON UPDATE CASCADE,
    position   INTEGER NOT NULL,
    content    TEXT NOT NULL,
    status     TEXT NOT NULL
               CHECK (status IN ('pendiente', 'en_progreso', 'completada', 'cancelada')),
    priority   TEXT NOT NULL DEFAULT 'media'
               CHECK (priority IN ('alta', 'media', 'baja')),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    parent_position INTEGER,
    PRIMARY KEY (session_id, position)
);
`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
CREATE TABLE flow_context (
    id         TEXT PRIMARY KEY NOT NULL,
    session_id TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE ON UPDATE CASCADE,
    flow       TEXT NOT NULL,
    stage      TEXT NOT NULL,
    stage_name TEXT NOT NULL,
    position   INTEGER NOT NULL CHECK (position >= 0),
    content    TEXT NOT NULL,
    created_at TEXT NOT NULL
);
`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
CREATE TABLE chat_evento (
    id         TEXT PRIMARY KEY NOT NULL,
    session_id TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE ON UPDATE CASCADE,
    tipo       TEXT NOT NULL CHECK (tipo IN ('proceso', 'herramienta', 'respuesta')),
    content    TEXT NOT NULL,
    created_at TEXT NOT NULL,
    duration_ms INTEGER,
    call_id    TEXT,
    herramienta TEXT,
    argumentos TEXT,
    salida     TEXT,
    error      TEXT,
    ok         INTEGER,
    truncado   INTEGER
);
`)
	if err != nil {
		return err
	}

	// Indices
	_, err = tx.Exec(`CREATE UNIQUE INDEX idx_projects_path ON projects (path);`)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`CREATE INDEX idx_sessions_project ON sessions (project_id, updated_at);`)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`CREATE INDEX idx_sessions_status ON sessions (project_id, status);`)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`CREATE INDEX idx_messages_session_created ON messages (session_id, created_at);`)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`CREATE UNIQUE INDEX idx_reasoning_message ON reasoning (message_id);`)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`CREATE INDEX idx_approvals_session_status ON approvals (session_id, status);`)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`CREATE INDEX idx_approvals_pending ON approvals (created_at) WHERE status = 'pendiente';`)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`CREATE INDEX idx_context_audit_session_stage ON context_audit (session_id, stage);`)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`CREATE UNIQUE INDEX idx_context_audit_unico ON context_audit (session_id, stage, document);`)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`CREATE INDEX idx_change_history_file ON change_history (file_path);`)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`CREATE INDEX idx_sessions_updated ON sessions (updated_at);`)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`CREATE UNIQUE INDEX idx_flow_context_unico ON flow_context (session_id, flow, stage);`)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`CREATE INDEX idx_chat_evento_session_created ON chat_evento (session_id, created_at);`)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`CREATE INDEX idx_todos_padre ON todos (session_id, parent_position);`)
	if err != nil {
		return err
	}

	// user_version = 12
	_, err = tx.Exec("PRAGMA user_version = 12;")
	if err != nil {
		return err
	}

	return tx.Commit()
}
