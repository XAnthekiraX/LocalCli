-- schema.sql — DDL del esquema v1 de LocalCli (T-B002-03).
-- Seis tablas, sin séptima: la versión vive en PRAGMA user_version
-- (MIGRATIONS.md). Convenciones de SCHEMA.md §1: id TEXT UUID v4, fechas
-- TEXT ISO 8601 UTC, booleanos enteros 0/1, ausencia = NULL.

CREATE TABLE sessions (
    id         TEXT PRIMARY KEY,                -- UUID v4
    name       TEXT NOT NULL,                   -- nombre visible de la sesión
    layer      TEXT CHECK (layer IN ('backend', 'frontend')), -- NULL = sesión general
    status     TEXT NOT NULL DEFAULT 'inactiva'
               CHECK (status IN ('inactiva', 'trabajando', 'esperando_permiso',
                                  'terminada', 'error')),
    created_at TEXT NOT NULL,                   -- ISO 8601 UTC
    updated_at TEXT NOT NULL                   -- ISO 8601 UTC
);

CREATE TABLE messages (
    id           TEXT PRIMARY KEY,              -- UUID v4
    session_id   TEXT NOT NULL
                 REFERENCES sessions(id) ON DELETE CASCADE ON UPDATE CASCADE,
    role         TEXT NOT NULL CHECK (role IN ('user', 'agent')),
    content      TEXT NOT NULL,
    input_tokens  INTEGER CHECK (input_tokens IS NULL OR input_tokens >= 0),
    output_tokens INTEGER CHECK (output_tokens IS NULL OR output_tokens >= 0),
    created_at   TEXT NOT NULL                  -- ISO 8601 UTC
);

-- El razonamiento va en tabla propia (DECISIONS.md): llega en streaming y el
-- mensaje es inmutable una vez completo.
CREATE TABLE reasoning (
    id         TEXT PRIMARY KEY,                -- UUID v4
    message_id TEXT NOT NULL                    -- 1:1 con messages (UNIQUE abajo)
               REFERENCES messages(id) ON DELETE CASCADE ON UPDATE CASCADE,
    content    TEXT NOT NULL,
    created_at TEXT NOT NULL                    -- ISO 8601 UTC
);

CREATE TABLE approvals (
    id          TEXT PRIMARY KEY,               -- UUID v4
    session_id  TEXT NOT NULL
                REFERENCES sessions(id) ON DELETE CASCADE ON UPDATE CASCADE,
    description TEXT NOT NULL,
    status      TEXT NOT NULL DEFAULT 'pendiente'
                CHECK (status IN ('pendiente', 'aprobada', 'declinada', 'obsoleta')),
    created_at  TEXT NOT NULL,                  -- ISO 8601 UTC
    resolved_at TEXT,                           -- NULL mientras siga pendiente
    -- CONSTRAINTS.md §2: resolved_at consistente con el estado.
    CHECK ((status = 'pendiente') = (resolved_at IS NULL))
);

CREATE TABLE context_audit (
    id         TEXT PRIMARY KEY,                -- UUID v4
    session_id TEXT NOT NULL
               REFERENCES sessions(id) ON DELETE CASCADE ON UPDATE CASCADE,
    stage      TEXT NOT NULL,                   -- etapa del flujo
    document   TEXT NOT NULL,                   -- ruta relativa al proyecto
    decision   TEXT NOT NULL CHECK (decision IN ('incluido', 'descartado')),
    reason     TEXT,                            -- obligatorio al descartar, nulo al incluir
    tokens     INTEGER CHECK (tokens IS NULL OR tokens >= 0),
    created_at TEXT NOT NULL,                   -- ISO 8601 UTC
    -- CONSTRAINTS.md §2: todo descarte queda justificado.
    CHECK ((decision = 'descartado') = (reason IS NOT NULL))
);

-- change_history nunca se borra: su session_id usa SET NULL para sobrevivir
-- a la sesión (RELATIONSHIPS.md §3).
CREATE TABLE change_history (
    id             TEXT PRIMARY KEY,            -- UUID v4
    session_id     TEXT
                   REFERENCES sessions(id) ON DELETE SET NULL ON UPDATE CASCADE,
    operation      TEXT NOT NULL
                   CHECK (operation IN ('crear_archivo', 'escribir_archivo',
                                        'editar_archivo', 'eliminar_archivo',
                                        'crear_carpeta', 'eliminar_carpeta')),
    file_path      TEXT NOT NULL,               -- ruta relativa al proyecto
    before_content TEXT,
    after_content  TEXT,
    created_at     TEXT NOT NULL,               -- ISO 8601 UTC
    -- CONSTRAINTS.md §2: contenido según la operación.
    CHECK (
        (operation IN ('crear_archivo', 'crear_carpeta') AND before_content IS NULL)
        OR (operation IN ('eliminar_archivo', 'eliminar_carpeta') AND after_content IS NULL)
        OR (operation IN ('escribir_archivo', 'editar_archivo')
            AND before_content IS NOT NULL AND after_content IS NOT NULL)
    )
);

-- Índices explícitos de INDEXES.md §1 (las PK generan los suyos propios).
CREATE INDEX idx_messages_session_created ON messages (session_id, created_at);
-- UNIQUE: es lo que hace efectiva la relación 1:1 messages↔reasoning
-- (INDEXES.md §4) y a la vez acelera leer el razonamiento de un mensaje.
CREATE UNIQUE INDEX idx_reasoning_message ON reasoning (message_id);
CREATE INDEX idx_approvals_session_status ON approvals (session_id, status);
-- Parcial: solo filas pendientes; ordena por created_at, no por status
-- (INDEXES.md §3). Es la consulta del contador en cada redibujado.
CREATE INDEX idx_approvals_pending ON approvals (created_at) WHERE status = 'pendiente';
CREATE INDEX idx_context_audit_session_stage ON context_audit (session_id, stage);
CREATE INDEX idx_change_history_file ON change_history (file_path);
CREATE INDEX idx_sessions_status ON sessions (status);
CREATE INDEX idx_sessions_updated ON sessions (updated_at);
