package store

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
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

	// Bidireccional: no solo importan los que faltan, tambien sobraria uno.
	got := indicesPorNombre(t, db)
	for want := range indicesWant {
		if _, ok := got[want]; !ok {
			t.Errorf("falta el índice %q", want)
		}
	}
	for name := range got {
		if _, ok := indicesWant[name]; !ok {
			t.Errorf("índice sobrante: %q", name)
		}
	}
}

// indicesWant es el catálogo de INDEXES.md §1/§3: nombre -> columnas indexadas
// en orden, más si es UNIQUE y si es parcial.
var indicesWant = map[string]struct {
	columnas []string
	unique   bool
	parcial  string // predicado del WHERE, "" si no es parcial
}{
	"idx_messages_session_created":    {columnas: []string{"session_id", "created_at"}},
	"idx_reasoning_message":           {columnas: []string{"message_id"}, unique: true},
	"idx_approvals_session_status":    {columnas: []string{"session_id", "status"}},
	"idx_approvals_pending":           {columnas: []string{"created_at"}, parcial: "status = 'pendiente'"},
	"idx_context_audit_session_stage": {columnas: []string{"session_id", "stage"}},
	"idx_context_audit_unico":         {columnas: []string{"session_id", "stage", "document"}, unique: true},
	"idx_change_history_file":         {columnas: []string{"file_path"}},
	"idx_sessions_status":             {columnas: []string{"status"}},
	"idx_sessions_updated":            {columnas: []string{"updated_at"}},
}

// Un índice con las columnas en otro orden, sin su UNIQUE o sin su predicado
// parcial no sirve para la consulta que lo justifie: se comprueba cada cosa.
func TestIndiceCumpleColumnasOrdenUnicidadYParcialidad(t *testing.T) {
	db := abrirBaseTemporal(t)
	for nombre, want := range indicesWant {
		cols := columnasDeIndice(t, db, nombre)
		if len(cols) != len(want.columnas) {
			t.Errorf("%s: columnas = %v, queremos %v", nombre, cols, want.columnas)
			continue
		}
		for i := range cols {
			if cols[i] != want.columnas[i] {
				t.Errorf("%s: columna %d = %q, queremos %q (el orden importa)", nombre, i, cols[i], want.columnas[i])
			}
		}
		var unique int
		if err := db.QueryRow(
			`SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name=? AND sql LIKE '%UNIQUE%'`,
			nombre).Scan(&unique); err != nil {
			t.Fatal(err)
		}
		if (unique == 1) != want.unique {
			t.Errorf("%s: UNIQUE = %v, queremos %v", nombre, unique == 1, want.unique)
		}
		var sql string
		if err := db.QueryRow(`SELECT COALESCE(sql,'') FROM sqlite_master WHERE type='index' AND name=?`, nombre).Scan(&sql); err != nil {
			t.Fatal(err)
		}
		tieneParcial := strings.Contains(sql, " WHERE ")
		if tieneParcial != (want.parcial != "") {
			t.Errorf("%s: parcial = %v, queremos %v (sql: %s)", nombre, tieneParcial, want.parcial != "", sql)
		}
		if want.parcial != "" && !strings.Contains(sql, want.parcial) {
			t.Errorf("%s: el predicado parcial no contiene %q: %s", nombre, want.parcial, sql)
		}
	}
}

func indicesPorNombre(t *testing.T, db *sql.DB) map[string]bool {
	t.Helper()
	rows, err := db.Query(`SELECT name FROM sqlite_master WHERE type='index' AND name LIKE 'idx_%'`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			t.Fatal(err)
		}
		out[n] = true
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

func columnasDeIndice(t *testing.T, db *sql.DB, indice string) []string {
	t.Helper()
	rows, err := db.Query(`SELECT name FROM pragma_index_info(?)`, indice)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			t.Fatal(err)
		}
		out = append(out, n)
	}
	return out
}

// --- T-B002-04: user_version e idempotencia ---

func TestMigracionIdempotente(t *testing.T) {
	proyecto := proyectoTemporal(t)

	// El valor se compara contra el literal 1 que fija MIGRATIONS.md, no contra
	// la constante de producción: si ambas suben a 2, este test debe seguir
	// avisando de que la documentación y el código han divergido.
	const versionEsperada = 1
	if schemaVersion != versionEsperada {
		t.Errorf("schemaVersion = %d; MIGRATIONS.md sigue fijando la v1 como esquema actual. Si el cambio es real, actualiza la nota de MIGRATIONS.md y este literal.", schemaVersion)
	}

	db, err := Open(proyecto)
	if err != nil {
		t.Fatal(err)
	}
	var v int
	if err := db.QueryRow("PRAGMA user_version").Scan(&v); err != nil {
		t.Fatal(err)
	}
	if v != versionEsperada {
		t.Fatalf("user_version = %d, queremos %d", v, versionEsperada)
	}
	// Un segundo Open sobre el MISMO archivo, con un handle nuevo: es la
	// idempotencia que de verdad importa (arrancar dos veces el binario).
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	db2, err := Open(proyecto)
	if err != nil {
		t.Fatalf("segundo Open: %v", err)
	}
	defer db2.Close()
	if err := db2.QueryRow("PRAGMA user_version").Scan(&v); err != nil {
		t.Fatal(err)
	}
	if v != versionEsperada {
		t.Errorf("tras reabrir, user_version = %d, queremos %d", v, versionEsperada)
	}
	// Reabrir no debe duplicar el esquema: siguen siendo seis tablas.
	var n int
	if err := db2.QueryRow(
		`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 6 {
		t.Errorf("tablas tras reabrir = %d, queremos 6", n)
	}
}

// proyectoTemporal crea un proyecto mínimo y devuelve su ruta, sin abrir la
// base: para los tests que necesitan controlar el ciclo de apertura.
func proyectoTemporal(t *testing.T) string {
	t.Helper()
	proyecto := t.TempDir()
	if err := os.MkdirAll(filepath.Join(proyecto, "ai", "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(proyecto, "ai", "docs", "PROJECT.md"), []byte("# p\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return proyecto
}

// verifyEsquema: una base con el DDL viejo se rechaza en vez de usarse en
// silencio. `id TEXT PRIMARY KEY` sin NOT NULL es un esquema distinto del
// declarado aunque user_version coincida.
func TestEsquemaDesfasadoSeRechaza(t *testing.T) {
	proyecto := proyectoTemporal(t)
	dbPath, err := DBPath(proyecto)
	if err != nil {
		t.Fatal(err)
	}
	// Se fabrica a mano una base con el defecto: id sin NOT NULL y sin el
	// índice único de auditoría, y con user_version ya en 1. La carpeta de
	// estado la crea Open, así que hay que crearla aquí.
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		t.Fatal(err)
	}
	legacy, err := sql.Open("sqlite", dsn(dbPath))
	if err != nil {
		t.Fatal(err)
	}
	ddlViejo := `CREATE TABLE sessions (
		id TEXT PRIMARY KEY, name TEXT NOT NULL, layer TEXT, status TEXT NOT NULL,
		created_at TEXT NOT NULL, updated_at TEXT NOT NULL);
	CREATE TABLE messages (id TEXT PRIMARY KEY, session_id TEXT NOT NULL
		REFERENCES sessions(id) ON DELETE CASCADE, role TEXT NOT NULL, content TEXT NOT NULL,
		input_tokens INTEGER, output_tokens INTEGER, created_at TEXT NOT NULL);
	CREATE TABLE reasoning (id TEXT PRIMARY KEY, message_id TEXT NOT NULL
		REFERENCES messages(id) ON DELETE CASCADE, content TEXT NOT NULL, created_at TEXT NOT NULL);
	CREATE TABLE approvals (id TEXT PRIMARY KEY, session_id TEXT NOT NULL
		REFERENCES sessions(id) ON DELETE CASCADE, description TEXT NOT NULL,
		status TEXT NOT NULL, created_at TEXT NOT NULL, resolved_at TEXT);
	CREATE TABLE context_audit (id TEXT PRIMARY KEY, session_id TEXT NOT NULL
		REFERENCES sessions(id) ON DELETE CASCADE, stage TEXT NOT NULL, document TEXT NOT NULL,
		decision TEXT NOT NULL, reason TEXT, tokens INTEGER, created_at TEXT NOT NULL);
	CREATE TABLE change_history (id TEXT PRIMARY KEY, session_id TEXT
		REFERENCES sessions(id) ON DELETE SET NULL, operation TEXT NOT NULL, file_path TEXT NOT NULL,
		before_content TEXT, after_content TEXT, created_at TEXT NOT NULL);
	PRAGMA user_version = 1;`
	if _, err := legacy.Exec(ddlViejo); err != nil {
		t.Fatalf("no se pudo fabricar la base vieja: %v", err)
	}
	if err := legacy.Close(); err != nil {
		t.Fatal(err)
	}

	_, err = Open(proyecto)
	if !errors.Is(err, ErrEsquemaDesfasado) {
		t.Fatalf("err = %v, queremos ErrEsquemaDesfasado (no usar una base cuyo esquema no coincide)", err)
	}
	if !strings.Contains(err.Error(), dbFileName) {
		t.Errorf("el aviso debería decir qué borrar; dice: %v", err)
	}
}
