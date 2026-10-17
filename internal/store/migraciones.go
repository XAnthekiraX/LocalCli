package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// versionEsquema es la versión actual del esquema ([[database/03-operations/MIGRATIONS]]).
const versionEsquema = 10

// Migrar lleva la base a la versión objetivo aplicando, en orden y una a una, las
// migraciones que falten ([[database/03-operations/MIGRATIONS]] §1 y §2).
//
// Un archivo recién creado (user_version = 0, sin esquema) nace ya en la versión
// actual: se crea el esquema completo de una vez en lugar de encadenar las
// migraciones 001..010, que dejarían el archivo a medias si algo se interrumpe.
// El 001 no puede distinguir «archivo vacío» de «v1» por el número solo, así que
// el 0 se resuelve como «sin esquema» y se crea todo.
func Migrar(db *sql.DB, versionObjetivo int) error {
	var userVersion int
	if err := db.QueryRow("PRAGMA user_version").Scan(&userVersion); err != nil {
		return fmt.Errorf("store: no se pudo leer user_version: %w", err)
	}

	if userVersion >= versionObjetivo {
		return nil
	}

	if userVersion == 0 {
		if err := initSchema(db); err != nil {
			return fmt.Errorf("store: no se pudo crear el esquema: %w", err)
		}
		return nil
	}

	for v := userVersion; v < versionObjetivo; v++ {
		if v == 6 {
			// El 007 reconstruye `sessions` y necesita gobernar las claves
			// foráneas fuera de transacción, así que gestiona su propia conexión.
			if err := migracion007(db); err != nil {
				return fmt.Errorf("store: migración %d falló: %w", v+1, err)
			}
			continue
		}
		if err := aplicarMigracion(db, v); err != nil {
			return fmt.Errorf("store: migración %d falló: %w", v+1, err)
		}
	}

	return nil
}

// aplicarMigracion ejecuta la migración que lleva la base de la versión `desde` a
// `desde+1` en su propia transacción y sube `user_version` dentro de ella: si algo
// falla se revierte la migración y la versión no avanza (MIGRATIONS.md §2).
func aplicarMigracion(db *sql.DB, desde int) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var migrar func(*sql.Tx) error
	switch desde {
	case 1:
		migrar = migracion002
	case 2:
		migrar = migracion003
	case 3:
		migrar = migracion004
	case 4:
		migrar = migracion005
	case 5:
		migrar = migracion006
	case 7:
		migrar = migracion008
	case 8:
		migrar = migracion009
	case 9:
		migrar = migracion010
	default:
		return fmt.Errorf("store: migración desde versión %d no soportada", desde)
	}

	if err := migrar(tx); err != nil {
		return err
	}
	if _, err := tx.Exec(fmt.Sprintf("PRAGMA user_version = %d;", desde+1)); err != nil {
		return err
	}
	return tx.Commit()
}

// migracion002 (`002-crear-todo`) es aditiva: añade la tabla `todos` y su clave
// primaria `(session_id, position)`, sin tocar ninguna fila existente.
func migracion002(tx *sql.Tx) error {
	_, err := tx.Exec(`
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
    PRIMARY KEY (session_id, position)
);
`)
	return err
}

// migracion003 (`003-crear-flow-context`) es aditiva: añade la tabla `flow_context`
// y su índice único `(session_id, flow, stage)`.
func migracion003(tx *sql.Tx) error {
	if _, err := tx.Exec(`
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
`); err != nil {
		return err
	}
	_, err := tx.Exec(`CREATE UNIQUE INDEX idx_flow_context_unico ON flow_context (session_id, flow, stage);`)
	return err
}

// migracion004 (`004-crear-chat-evento`) es aditiva: añade la tabla `chat_evento`
// con su índice, y la columna `messages.duration_ms` (nullable: las filas
// anteriores quedan con NULL, que significa «no se midió»).
func migracion004(tx *sql.Tx) error {
	if _, err := tx.Exec(`
CREATE TABLE chat_evento (
    id         TEXT PRIMARY KEY NOT NULL,
    session_id TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE ON UPDATE CASCADE,
    tipo       TEXT NOT NULL CHECK (tipo IN ('proceso', 'herramienta')),
    content    TEXT NOT NULL,
    created_at TEXT NOT NULL
);
`); err != nil {
		return err
	}
	if _, err := tx.Exec(`CREATE INDEX idx_chat_evento_session_created ON chat_evento (session_id, created_at);`); err != nil {
		return err
	}
	_, err := tx.Exec(`ALTER TABLE messages ADD COLUMN duration_ms INTEGER;`)
	return err
}

// migracion005 (`005-duracion-linea-herramienta`) es aditiva y solo añade
// `chat_evento.duration_ms`, nullable.
func migracion005(tx *sql.Tx) error {
	_, err := tx.Exec(`ALTER TABLE chat_evento ADD COLUMN duration_ms INTEGER;`)
	return err
}

// migracion006 (`006-motor-y-modelo-de-sesion`) es aditiva y solo añade a
// `sessions` las columnas `motor_id` y `modelo`, ambas nullable.
func migracion006(tx *sql.Tx) error {
	if _, err := tx.Exec(`ALTER TABLE sessions ADD COLUMN motor_id TEXT;`); err != nil {
		return err
	}
	_, err := tx.Exec(`ALTER TABLE sessions ADD COLUMN modelo TEXT;`)
	return err
}

// migracion007 (`007-crear-proyectos`) hace global la base: crea `projects` y
// añade `sessions.project_id`. SQLite no admite añadir con ALTER TABLE una columna
// NOT NULL con clave foránea, así que se reconstruye `sessions`. Eso obliga a
// apagar las claves foráneas: con ellas activas, el DROP de `sessions` borraría en
// cascada `messages`, `reasoning` y el resto del historial. Las sesiones existentes
// se reasignan a un proyecto «heredado» en vez de perderlas (MIGRATIONS.md §4 y §5).
func migracion007(db *sql.DB) error {
	ctx := context.Background()
	conn, err := db.Conn(ctx)
	if err != nil {
		return err
	}
	defer func() {
		// Devolver la conexión al pool en su estado normal.
		conn.ExecContext(ctx, "PRAGMA foreign_keys = ON")
		conn.Close()
	}()

	if _, err := conn.ExecContext(ctx, "PRAGMA foreign_keys = OFF"); err != nil {
		return err
	}

	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`
CREATE TABLE projects (
    id         TEXT PRIMARY KEY NOT NULL,
    path       TEXT NOT NULL UNIQUE,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
`); err != nil {
		return err
	}
	if _, err := tx.Exec(`CREATE UNIQUE INDEX idx_projects_path ON projects (path);`); err != nil {
		return err
	}

	// Solo hace falta un proyecto heredado si hay sesiones que reasignar.
	const heredado = "heredado"
	var sesiones int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM sessions`).Scan(&sesiones); err != nil {
		return err
	}
	if sesiones > 0 {
		ahora := time.Now().UTC().Format(time.RFC3339)
		if _, err := tx.Exec(
			`INSERT INTO projects (id, path, created_at, updated_at) VALUES (?, '', ?, ?)`,
			heredado, ahora, ahora,
		); err != nil {
			return err
		}
	}

	if _, err := tx.Exec(`
CREATE TABLE sessions_nuevo (
    id         TEXT PRIMARY KEY NOT NULL,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE ON UPDATE CASCADE,
    name       TEXT NOT NULL,
    layer      TEXT CHECK (layer IN ('backend', 'frontend')),
    status     TEXT NOT NULL DEFAULT 'inactiva'
               CHECK (status IN ('inactiva', 'trabajando', 'esperando_permiso',
                                  'terminada', 'error')),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    motor_id   TEXT,
    modelo     TEXT
);
`); err != nil {
		return err
	}
	if _, err := tx.Exec(`
INSERT INTO sessions_nuevo (id, project_id, name, layer, status, created_at, updated_at, motor_id, modelo)
SELECT id, ?, name, layer, status, created_at, updated_at, motor_id, modelo FROM sessions;
`, heredado); err != nil {
		return err
	}
	if _, err := tx.Exec(`DROP TABLE sessions;`); err != nil {
		return err
	}
	if _, err := tx.Exec(`ALTER TABLE sessions_nuevo RENAME TO sessions;`); err != nil {
		return err
	}
	if _, err := tx.Exec(`CREATE INDEX idx_sessions_project ON sessions (project_id, updated_at);`); err != nil {
		return err
	}
	if _, err := tx.Exec(`DROP INDEX IF EXISTS idx_sessions_status;`); err != nil {
		return err
	}
	if _, err := tx.Exec(`CREATE INDEX idx_sessions_status ON sessions (project_id, status);`); err != nil {
		return err
	}
	if _, err := tx.Exec(`CREATE INDEX idx_sessions_updated ON sessions (updated_at);`); err != nil {
		return err
	}
	if _, err := tx.Exec("PRAGMA user_version = 7;"); err != nil {
		return err
	}
	return tx.Commit()
}

// migracion008 (`008-resumen-sesion`) es aditiva y solo añade a `sessions` las
// columnas `summary` (el resumen compactado del historial) y `summary_upto` (el
// id del último mensaje que ese resumen abarca), ambas nullable: las sesiones
// anteriores quedan con NULL, que significa «sin resumen». Ver
// [[specs/SPEC-HISTORIAL-CONVERSACION]] y [[database/03-operations/MIGRATIONS]].
func migracion008(tx *sql.Tx) error {
	if _, err := tx.Exec(`ALTER TABLE sessions ADD COLUMN summary TEXT;`); err != nil {
		return err
	}
	_, err := tx.Exec(`ALTER TABLE sessions ADD COLUMN summary_upto TEXT;`)
	return err
}

// migracion009 (`009-quitar-motor-id`) elimina la columna vestigial
// `sessions.motor_id`: con un único proveedor (llama.cpp) el motor no se elige
// por sesión, la columna siempre estaba en NULL y el modelo ya vive en
// `sessions.modelo`. Solo dropea la columna; no toca filas.
func migracion009(tx *sql.Tx) error {
	_, err := tx.Exec(`ALTER TABLE sessions DROP COLUMN motor_id;`)
	return err
}

// migracion010 (`010-respuesta-etapa`) amplía el `CHECK` de `chat_evento.tipo`
// para admitir `respuesta`, el tipo de la respuesta completa de una etapa de
// flujo con `respuesta_en_chat` (SPEC-ORQUESTADOR-FLUJOS §Visibilidad de una
// respuesta). SQLite no permite cambiar un `CHECK` con `ALTER TABLE`, así que la
// tabla se reconstruye: se renombra la vieja, se crea la nueva con el mismo
// esquema y sus claves, se copian las filas y se dropea la vieja junto a su
// índice, que se vuelve a crear sobre la tabla nueva. Las filas existentes —
// `proceso` y `herramienta`— siguen siendo válidas: no se pierde nada.
func migracion010(tx *sql.Tx) error {
	if _, err := tx.Exec(`ALTER TABLE chat_evento RENAME TO chat_evento_vieja;`); err != nil {
		return err
	}
	if _, err := tx.Exec(`
CREATE TABLE chat_evento (
    id         TEXT PRIMARY KEY NOT NULL,
    session_id TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE ON UPDATE CASCADE,
    tipo       TEXT NOT NULL CHECK (tipo IN ('proceso', 'herramienta', 'respuesta')),
    content    TEXT NOT NULL,
    created_at TEXT NOT NULL,
    duration_ms INTEGER
);
`); err != nil {
		return err
	}
	if _, err := tx.Exec(`
INSERT INTO chat_evento (id, session_id, tipo, content, created_at, duration_ms)
SELECT id, session_id, tipo, content, created_at, duration_ms FROM chat_evento_vieja;
`); err != nil {
		return err
	}
	if _, err := tx.Exec(`DROP TABLE chat_evento_vieja;`); err != nil {
		return err
	}
	_, err := tx.Exec(`CREATE INDEX idx_chat_evento_session_created ON chat_evento (session_id, created_at);`)
	return err
}
