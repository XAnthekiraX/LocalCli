-- todo.sql — la lista de pasos de una sesión (migración 002-crear-todo).
--
-- A diferencia de la cola, que es una proyección de `ai/tasks/` y no se
-- almacena (DECISIONS.md), la lista de pasos del AGENTE sí vive en la base: es
-- estado de ejecución de la sesión, se reescribe entera en cada llamada a
-- `actualizar_todo` y desaparece en cascada con su sesión.
--
-- El orden es la `position`: la lista llega completa y se numera por su índice,
-- sin identificadores de fila que puedan quedar obsoletos. La clave primaria
-- compuesta (session_id, position) ya indexa las lecturas por sesión.

CREATE TABLE todos (
    session_id TEXT NOT NULL
               REFERENCES sessions(id) ON DELETE CASCADE ON UPDATE CASCADE,
    position   INTEGER NOT NULL,
    content    TEXT NOT NULL,
    status     TEXT NOT NULL
               CHECK (status IN ('pendiente', 'en_progreso', 'completada', 'cancelada')),
    priority   TEXT NOT NULL DEFAULT 'media'
               CHECK (priority IN ('alta', 'media', 'baja')),
    created_at TEXT NOT NULL,                   -- ISO 8601 UTC
    updated_at TEXT NOT NULL,                   -- ISO 8601 UTC
    PRIMARY KEY (session_id, position)
);
