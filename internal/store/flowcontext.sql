-- flowcontext.sql — el bloque de contexto de un flujo (migración
-- 003-crear-flow-context).
--
-- Un flujo con el flag `bloque_contexto` (hoy solo el resolver) no descarta el
-- resultado de sus etapas: cada una deja aquí su aportación ya optimizada por el
-- modelo y la última etapa compone la entrega a partir de TODO el bloque, sin
-- herramientas (SPEC-MOTOR-FLUJOS §Bloque de contexto, DECISIONS.md).
--
-- Es estado de ejecución de la sesión: se reemplaza en cada ejecución del mismo
-- flujo (una aportación por etapa) y cae en cascada con su sesión.

CREATE TABLE flow_context (
    id         TEXT PRIMARY KEY NOT NULL,               -- UUID v4
    session_id TEXT NOT NULL
               REFERENCES sessions(id) ON DELETE CASCADE ON UPDATE CASCADE,
    flow       TEXT NOT NULL,                   -- nombre del flujo (p. ej. "resolver")
    stage      TEXT NOT NULL,                   -- id de la etapa
    stage_name TEXT NOT NULL,                   -- nombre visible de la etapa
    position   INTEGER NOT NULL CHECK (position >= 0), -- orden en la secuencia
    content    TEXT NOT NULL,                   -- aportación optimizada
    created_at TEXT NOT NULL                    -- ISO 8601 UTC
);

-- Una aportación por etapa y flujo dentro de la sesión: re-ejecutar el flujo
-- reemplaza la fila en vez de duplicarla.
CREATE UNIQUE INDEX idx_flow_context_unico ON flow_context (session_id, flow, stage);
