package store

import (
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Un archivo recién creado (user_version = 0) debe nacer con el esquema completo:
// antes se creaba a medias y el historial no se podía guardar.
func TestAbrirDBDesdeUnArchivoVacioCreaElEsquemaCompleto(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "state.db")

	db, err := AbrirDBDesde(ruta)
	if err != nil {
		t.Fatalf("abrir base vacía: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	var version int
	if err := db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != versionEsquema {
		t.Fatalf("user_version = %d, quiero %d", version, versionEsquema)
	}

	// La prueba de fuego: el historial se puede escribir.
	projectID, err := ResolucionProyecto(db, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	sesion, err := CrearSesion(db, projectID, "nueva", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := CrearMensaje(db, projectID, sesion.ID, "user", "hola", nil, nil, nil); err != nil {
		t.Fatalf("no se pudo guardar un mensaje en una base nueva: %v", err)
	}
}

// Una base v6 debe migrar a v7 sin perder el historial: las sesiones existentes se
// reasignan a un proyecto heredado y sus mensajes siguen ahí.
func TestMigrarDesdeV6ConservaElHistorial(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "state.db")
	crearBaseV6(t, ruta)

	db, err := AbrirDBDesde(ruta)
	if err != nil {
		t.Fatalf("migrar v6 a v7: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	var version int
	if err := db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != versionEsquema {
		t.Fatalf("user_version = %d, quiero %d", version, versionEsquema)
	}

	var projectID string
	if err := db.QueryRow("SELECT project_id FROM sessions WHERE id = 's-vieja'").Scan(&projectID); err != nil {
		t.Fatalf("la sesión vieja se perdió: %v", err)
	}
	if projectID == "" {
		t.Fatal("la sesión vieja quedó sin proyecto")
	}
	var existeProyecto int
	if err := db.QueryRow("SELECT COUNT(*) FROM projects WHERE id = ?", projectID).Scan(&existeProyecto); err != nil {
		t.Fatal(err)
	}
	if existeProyecto != 1 {
		t.Fatalf("el proyecto %q no existe", projectID)
	}

	var mensajes int
	if err := db.QueryRow("SELECT COUNT(*) FROM messages WHERE session_id = 's-vieja'").Scan(&mensajes); err != nil {
		t.Fatal(err)
	}
	if mensajes != 1 {
		t.Fatalf("mensajes conservados = %d, quiero 1", mensajes)
	}

	rows, err := db.Query("PRAGMA foreign_key_check")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	if rows.Next() {
		t.Fatal("la migración dejó claves foráneas rotas")
	}
}

// crearBaseV6 fabrica una base con el esquema v6 (sesiones sin `project_id` y sin
// tabla `projects`) más una sesión y un mensaje, para probar la migración 007.
func crearBaseV6(t *testing.T, ruta string) {
	t.Helper()
	db, err := CrearDB(ruta)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	// Una única conexión: las claves foráneas son por conexión.
	db.SetMaxOpenConns(1)
	if _, err := db.Exec("PRAGMA foreign_keys = OFF"); err != nil {
		t.Fatal(err)
	}

	ahora := time.Now().UTC().Format(time.RFC3339)
	pasos := []string{
		`CREATE TABLE sessions_v6 (
			id         TEXT PRIMARY KEY NOT NULL,
			name       TEXT NOT NULL,
			layer      TEXT,
			status     TEXT NOT NULL DEFAULT 'inactiva',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			motor_id   TEXT,
			modelo     TEXT
		);`,
		`INSERT INTO sessions_v6 (id, name, status, created_at, updated_at) VALUES ('s-vieja', 'vieja', 'inactiva', '` + ahora + `', '` + ahora + `');`,
		`DROP TABLE sessions;`,
		`ALTER TABLE sessions_v6 RENAME TO sessions;`,
		`INSERT INTO messages (id, session_id, role, content, created_at) VALUES ('m-viejo', 's-vieja', 'user', 'hola viejo', '` + ahora + `');`,
		`DROP TABLE projects;`,
		`DROP INDEX idx_todos_padre;`,
		`ALTER TABLE todos DROP COLUMN parent_position;`,
		`PRAGMA user_version = 6;`,
	}
	for _, paso := range pasos {
		if _, err := db.Exec(paso); err != nil {
			t.Fatalf("preparar base v6 (%q): %v", paso, err)
		}
	}
}

// proyectoYSesionDeLaBaseV9 lee el proyecto y la sesión a los que apuntan las
// líneas del hilo de la base fabricada, que es con lo que hay que leerlas.
func proyectoYSesionDeLaBaseV9(db *sql.DB) (string, string, error) {
	var projectID, sesionID string
	err := db.QueryRow(
		`SELECT s.project_id, e.session_id FROM chat_evento e JOIN sessions s ON s.id = e.session_id LIMIT 1`,
	).Scan(&projectID, &sesionID)
	if err != nil {
		return "", "", err
	}
	return projectID, sesionID, nil
}

// TestMigrarDesdeV9ConservaLasLineasDelHilo: la 010 reconstruye `chat_evento` para
// admitir el tipo `respuesta`. Las líneas ya guardadas —`proceso` y `herramienta`—
// se conservan enteras, con su duración, y su índice sigue ahí.
func TestMigrarDesdeV9ConservaLasLineasDelHilo(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "state.db")
	crearBaseV9(t, ruta)

	db, err := AbrirDBDesde(ruta)
	if err != nil {
		t.Fatalf("migrar v9 a v10: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	var version int
	if err := db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != versionEsquema {
		t.Fatalf("user_version = %d, quiero %d", version, versionEsquema)
	}

	// La base fabricada deja el proyecto y la sesión a los que apuntan sus líneas.
	projectID, sesionID, err := proyectoYSesionDeLaBaseV9(db)
	if err != nil {
		t.Fatal(err)
	}

	eventos, err := ListarChatEventos(db, projectID, sesionID)
	if err != nil {
		t.Fatalf("las líneas del hilo no se pudieron leer tras migrar: %v", err)
	}
	if len(eventos) != 2 {
		t.Fatalf("líneas conservadas = %d, quiero 2", len(eventos))
	}
	if eventos[0].Tipo != ChatEventoProceso || eventos[0].Content != "[Sub Proceso] Diagnosticar" {
		t.Fatalf("la línea de etapa cambió: %+v", eventos[0])
	}
	if eventos[1].Tipo != ChatEventoHerramienta || eventos[1].DurationMs == nil || *eventos[1].DurationMs != 400 {
		t.Fatalf("la línea de herramienta perdió su duración: %+v", eventos[1])
	}

	var indice int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM sqlite_master WHERE type = 'index' AND name = 'idx_chat_evento_session_created'`,
	).Scan(&indice); err != nil || indice != 1 {
		t.Fatalf("el índice del hilo no sobrevivió a la reconstrucción: %d (%v)", indice, err)
	}

	// El tipo nuevo ya cabe en el CHECK de la tabla migrada.
	if _, err := CrearChatEvento(db, projectID, sesionID, ChatEventoRespuesta, "la respuesta de la etapa", nil); err != nil {
		t.Fatalf("tras migrar debería admitirse el tipo `respuesta`: %v", err)
	}
	var clavesRotas int
	if err := db.QueryRow("SELECT COUNT(*) FROM pragma_foreign_key_check").Scan(&clavesRotas); err != nil {
		t.Fatal(err)
	}
	if clavesRotas != 0 {
		t.Fatal("la migración dejó claves foráneas rotas")
	}
}

// crearBaseV9 fabrica una base con el esquema v9 —la misma tabla `chat_evento`,
// pero con su `CHECK` limitado a `proceso` y `herramienta`— más una sesión y dos
// líneas del hilo, para probar la migración 010.
func crearBaseV9(t *testing.T, ruta string) {
	t.Helper()
	db, err := CrearDB(ruta)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	// Una única conexión: las claves foráneas son por conexión.
	db.SetMaxOpenConns(1)

	pasos := []string{
		`DROP TABLE chat_evento;`,
		`CREATE TABLE chat_evento (
			id         TEXT PRIMARY KEY NOT NULL,
			session_id TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE ON UPDATE CASCADE,
			tipo       TEXT NOT NULL CHECK (tipo IN ('proceso', 'herramienta')),
			content    TEXT NOT NULL,
			created_at TEXT NOT NULL,
			duration_ms INTEGER
		);`,
		`CREATE INDEX idx_chat_evento_session_created ON chat_evento (session_id, created_at);`,
		`DROP INDEX idx_todos_padre;`,
		`ALTER TABLE todos DROP COLUMN parent_position;`,
		`PRAGMA user_version = 9;`,
	}
	for _, paso := range pasos {
		if _, err := db.Exec(paso); err != nil {
			t.Fatalf("preparar base v9 (%q): %v", paso, err)
		}
	}

	projectID, err := ResolucionProyecto(db, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if projectID == "" {
		t.Fatal("la base de prueba quedó sin proyecto")
	}
	sesion, err := CrearSesion(db, projectID, "vieja", nil)
	if err != nil {
		t.Fatal(err)
	}
	if sesion.ID == "" {
		t.Fatal("la base de prueba quedó sin sesión")
	}
	ahora := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := db.Exec(
		`INSERT INTO chat_evento (id, session_id, tipo, content, created_at) VALUES
		 ('e-proceso', ?, ?, ?, ?)`,
		sesion.ID, ChatEventoProceso, "[Sub Proceso] Diagnosticar", ahora,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(
		`INSERT INTO chat_evento (id, session_id, tipo, content, created_at, duration_ms) VALUES
		 ('e-herramienta', ?, ?, ?, ?, 400)`,
		sesion.ID, ChatEventoHerramienta, "✓ LEER [AGENTS.md] · 93 líneas", ahora,
	); err != nil {
		t.Fatal(err)
	}
}

// TestMigrarDesdeV10AnadeLaJerarquia: la 011 añade `todos.parent_position` y su
// índice sin tocar las filas existentes. Una base v10 se lee como lista plana.
func TestMigrarDesdeV10AnadeLaJerarquia(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "state.db")
	crearBaseV10(t, ruta)

	db, err := AbrirDBDesde(ruta)
	if err != nil {
		t.Fatalf("migrar v10 a v11: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	var version int
	if err := db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != versionEsquema {
		t.Fatalf("user_version = %d, quiero %d", version, versionEsquema)
	}

	var projectID, sesionID string
	if err := db.QueryRow(`SELECT s.project_id, t.session_id FROM sessions s JOIN todos t ON t.session_id = s.id LIMIT 1`).
		Scan(&projectID, &sesionID); err != nil {
		t.Fatal(err)
	}
	lista, err := ListarTodos(db, projectID, sesionID)
	if err != nil {
		t.Fatalf("la lista no se pudo leer tras migrar: %v", err)
	}
	if len(lista) != 1 || lista[0].Contenido != "paso viejo" {
		t.Fatalf("la fila existente se perdió o cambió: %+v", lista)
	}
	if lista[0].PosicionPadre != nil {
		t.Fatalf("una fila de una base v10 debería leerse como raíz: %+v", lista[0])
	}

	var indice int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM sqlite_master WHERE type = 'index' AND name = 'idx_todos_padre'`,
	).Scan(&indice); err != nil || indice != 1 {
		t.Fatalf("falta el índice idx_todos_padre: %d (%v)", indice, err)
	}
}

// crearBaseV10 fabrica una base con el esquema v10 —sin `todos.parent_position`—
// más una sesión con un paso, para probar la migración 011.
func crearBaseV10(t *testing.T, ruta string) {
	t.Helper()
	db, err := CrearDB(ruta)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)

	projectID, err := ResolucionProyecto(db, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	sesion, err := CrearSesion(db, projectID, "vieja", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := CrearTodo(db, projectID, sesion.ID, Todo{Contenido: "paso viejo", Estado: "pendiente", Prioridad: "media"}, nil); err != nil {
		t.Fatal(err)
	}

	pasos := []string{
		`DROP INDEX idx_todos_padre;`,
		`ALTER TABLE todos DROP COLUMN parent_position;`,
		// La v10 todavía no tenía las columnas de ejecución de `chat_evento`: se
		// reconstruye con el esquema de entonces para que la 012 las añada.
		`DROP TABLE chat_evento;`,
		`CREATE TABLE chat_evento (
			id         TEXT PRIMARY KEY NOT NULL,
			session_id TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE ON UPDATE CASCADE,
			tipo       TEXT NOT NULL CHECK (tipo IN ('proceso', 'herramienta', 'respuesta')),
			content    TEXT NOT NULL,
			created_at TEXT NOT NULL,
			duration_ms INTEGER
		);`,
		`CREATE INDEX idx_chat_evento_session_created ON chat_evento (session_id, created_at);`,
		`PRAGMA user_version = 10;`,
	}
	for _, paso := range pasos {
		if _, err := db.Exec(paso); err != nil {
			t.Fatalf("preparar base v10 (%q): %v", paso, err)
		}
	}
}

// La creación nueva y la migración 011 deben dejar el mismo esquema de `todos`:
// las mismas columnas, en el mismo orden, y el índice del padre.
func TestElEsquemaDeTodosNuevoYMigradoCoincide(t *testing.T) {
	nuevo, err := CrearDB(filepath.Join(t.TempDir(), "nuevo.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = nuevo.Close() })

	ruta := filepath.Join(t.TempDir(), "migrado.db")
	crearBaseV10(t, ruta)
	migrado, err := AbrirDBDesde(ruta)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = migrado.Close() })

	columnasNuevo, err := columnasDeTodos(nuevo)
	if err != nil {
		t.Fatal(err)
	}
	columnasMigrado, err := columnasDeTodos(migrado)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(columnasNuevo, "|") != strings.Join(columnasMigrado, "|") {
		t.Fatalf("columnas distintas:\n nuevo:   %v\n migrado: %v", columnasNuevo, columnasMigrado)
	}

	for nombre, db := range map[string]*sql.DB{"nuevo": nuevo, "migrado": migrado} {
		var indice int
		if err := db.QueryRow(
			`SELECT COUNT(*) FROM sqlite_master WHERE type = 'index' AND name = 'idx_todos_padre'`,
		).Scan(&indice); err != nil || indice != 1 {
			t.Fatalf("%s sin idx_todos_padre: %d (%v)", nombre, indice, err)
		}
	}
}

func columnasDeTodos(db *sql.DB) ([]string, error) {
	rows, err := db.Query(`SELECT name FROM pragma_table_info('todos') ORDER BY cid`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var columnas []string
	for rows.Next() {
		var nombre string
		if err := rows.Scan(&nombre); err != nil {
			return nil, err
		}
		columnas = append(columnas, nombre)
	}
	return columnas, rows.Err()
}
