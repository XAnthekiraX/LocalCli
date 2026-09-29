-- chatevento.sql — las líneas de procesamiento del chat (migración
-- 004-crear-chat-evento).
--
-- El hilo del chat muestra más que conversación: los sub-procesos de un flujo
-- (`[Sub Proceso] <etapa>`) y las líneas de herramienta («✓ LEER [ruta] · 70
-- líneas»). Hasta ahora vivían solo en memoria y se perdían al cambiar de
-- sesión; aquí quedan guardadas como parte del hilo.
--
-- Van en su propia tabla, y no en `messages`, a propósito: son LÍNEAS DE
-- PANTALLA, no turnos de conversación. El contexto que se le entrega al modelo
-- se arma solo de `messages` (`session.historialPara`), así que por construcción
-- una línea de procesamiento nunca entra al contexto. Ver ENUMS.md y
-- MIGRATIONS.md.
--
-- Es estado de la sesión: cae en cascada con ella, como su conversación.

CREATE TABLE chat_evento (
    id         TEXT PRIMARY KEY NOT NULL,               -- UUID v4
    session_id TEXT NOT NULL
               REFERENCES sessions(id) ON DELETE CASCADE ON UPDATE CASCADE,
    tipo       TEXT NOT NULL CHECK (tipo IN ('proceso', 'herramienta')),
    content    TEXT NOT NULL,                   -- la misma línea que se pinta
    created_at TEXT NOT NULL                    -- ISO 8601 UTC
);

CREATE INDEX idx_chat_evento_session_created ON chat_evento (session_id, created_at);

-- La duración del turno del agente, en milisegundos: lo que tardó de la
-- petición al cierre. Aditiva y nullable (filas anteriores quedan NULL, que es
-- "no se midió"). Sin CHECK: el valor lo valida el código, y así la migración no
-- depende del soporte de CHECK en ALTER TABLE (MIGRATIONS.md §4).
ALTER TABLE messages ADD COLUMN duration_ms INTEGER;
