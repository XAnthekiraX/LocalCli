package store

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// abrirBaseTemporal crea un proyecto mínimo en un t.TempDir() y abre su base
// con Open, que aplica pragmas y migraciones. Modo Pruebas de CONFIGURATION.md:
// "Base temporal, aislada, que se destruye al terminar" — t.TempDir se encarga.
func abrirBaseTemporal(t *testing.T) *sql.DB {
	t.Helper()
	proyecto := t.TempDir()
	if err := os.MkdirAll(filepath.Join(proyecto, "ai", "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(proyecto, "ai", "docs", "PROJECT.md"), []byte("# p\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	db, err := Open(proyecto)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// --- T-B002-01: pragmas activos tras abrir ---

func TestPragmasActivosTrasAbrir(t *testing.T) {
	db := abrirBaseTemporal(t)
	var modo string
	if err := db.QueryRow("PRAGMA journal_mode").Scan(&modo); err != nil {
		t.Fatal(err)
	}
	if modo != "wal" {
		t.Errorf("journal_mode = %q, queremos wal", modo)
	}
	var fk int
	if err := db.QueryRow("PRAGMA foreign_keys").Scan(&fk); err != nil {
		t.Fatal(err)
	}
	if fk != 1 {
		t.Errorf("foreign_keys = %d, queremos 1", fk)
	}
}

func TestOpenRechazaCarpetaQueNoEsProyecto(t *testing.T) {
	noProyecto := t.TempDir()
	_, err := Open(noProyecto)
	if !errors.Is(err, ErrNotAProject) {
		t.Fatalf("err = %v, queremos E_NOT_A_PROJECT", err)
	}
}

// --- T-B002-02: ruta de la base ---

func TestDBPathSinVariable(t *testing.T) {
	proyecto := t.TempDir()
	got, err := DBPath(proyecto)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(proyecto, ".localcli", "state.db")
	if got != want {
		t.Errorf("DBPath = %q, queremos %q", got, want)
	}
}

func TestDBPathConVariable(t *testing.T) {
	t.Setenv(EnvDBPath, "/rara/otro.db")
	got, err := DBPath("/proyecto")
	if err != nil {
		t.Fatal(err)
	}
	if got != "/rara/otro.db" {
		t.Errorf("DBPath = %q, LOCALCLI_DB_PATH debía ganar", got)
	}
}

// --- T-B002-03: esquema coincide con SCHEMA.md (6 tablas + índices) ---

func TestEsquemaSeisTablasYIndices(t *testing.T) {
	db := abrirBaseTemporal(t)
	tablasWant := []string{"approvals", "change_history", "context_audit", "messages", "reasoning", "sessions"}
	rows, err := db.Query(`SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name`)
	if err != nil {
		t.Fatal(err)
	}
	var tablasGot []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			t.Fatal(err)
		}
		tablasGot = append(tablasGot, n)
	}
	rows.Close()
	if len(tablasGot) != len(tablasWant) {
		t.Fatalf("tablas = %v, queremos %v", tablasGot, tablasWant)
	}
	for i, w := range tablasWant {
		if tablasGot[i] != w {
			t.Fatalf("tablas = %v, queremos %v", tablasGot, tablasWant)
		}
	}

	indicesWant := map[string]bool{
		"idx_messages_session_created":    true,
		"idx_reasoning_message":           true,
		"idx_approvals_session_status":    true,
		"idx_approvals_pending":           true,
		"idx_context_audit_session_stage": true,
		"idx_change_history_file":         true,
		"idx_sessions_status":             true,
		"idx_sessions_updated":            true,
	}
	rows2, err := db.Query(`SELECT name FROM sqlite_master WHERE type='index' AND name LIKE 'idx_%'`)
	if err != nil {
		t.Fatal(err)
	}
	for rows2.Next() {
		var n string
		if err := rows2.Scan(&n); err != nil {
			t.Fatal(err)
		}
		delete(indicesWant, n)
	}
	rows2.Close()
	if len(indicesWant) != 0 {
		t.Errorf("faltan índices: %v", indicesWant)
	}
}

// --- T-B002-04: user_version e idempotencia ---

func TestMigracionIdempotente(t *testing.T) {
	db := abrirBaseTemporal(t)
	var v int
	if err := db.QueryRow("PRAGMA user_version").Scan(&v); err != nil {
		t.Fatal(err)
	}
	if v != schemaVersion {
		t.Fatalf("user_version = %d, queremos %d", v, schemaVersion)
	}
	// segunda apertura sobre el mismo archivo: sin error, sin duplicar nada
	if err := migrate(db); err != nil {
		t.Fatalf("segunda migrate: %v", err)
	}
	if err := db.QueryRow("PRAGMA user_version").Scan(&v); err != nil {
		t.Fatal(err)
	}
	if v != schemaVersion {
		t.Errorf("tras segunda migrate user_version = %d", v)
	}
}
