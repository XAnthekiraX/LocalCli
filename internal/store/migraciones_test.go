package store

import (
	"path/filepath"
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
		`PRAGMA user_version = 6;`,
	}
	for _, paso := range pasos {
		if _, err := db.Exec(paso); err != nil {
			t.Fatalf("preparar base v6 (%q): %v", paso, err)
		}
	}
}
