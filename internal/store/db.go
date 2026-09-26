package store

import (
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite" // driver puro Go, sin cgo (DATABASE.md: binario portable)
)

// ErrNotAProject es el error interno equivalente a E_NOT_A_PROJECT: la carpeta
// abierta no es un proyecto válido y no se abre una base en un sitio que no toca
// (ERRORS.md §2 y §3).
var ErrNotAProject = errors.New("E_NOT_A_PROJECT: la carpeta abierta no es un proyecto válido")

// isProjectReported reporta si una ruta de documento hace de esta carpeta un
// proyecto LocalCli. Los documentos vivos del arranque son ai/docs/PROJECT.md y
// el TODO de alguna capa, ai/tasks/<capa>/MAIN-TASKS.md
// (CONFIGURATION.md §4, DATABASE.md "Archivos (fuente de verdad)").
func isProjectReported(projectDir string) bool {
	if _, err := os.Stat(filepath.Join(projectDir, "ai", "docs", "PROJECT.md")); err == nil {
		return true
	}
	capas, err := os.ReadDir(filepath.Join(projectDir, "ai", "tasks"))
	if err != nil {
		return false
	}
	for _, capa := range capas {
		if !capa.IsDir() {
			continue
		}
		if _, err := os.Stat(filepath.Join(projectDir, "ai", "tasks", capa.Name(), "MAIN-TASKS.md")); err == nil {
			return true
		}
	}
	return false
}

// dsn construye la cadena de conexión del driver modernc.org/sqlite con los
// PRAGMA obligatorios de cada conexión: foreign_keys=ON (DECISIONS.md: "sin
// este pragma SQLite ignora las cascadas") y journal_mode=WAL (SCHEMA.md §1),
// más busy_timeout para que dos sesiones concurrentes esperen en vez de fallar.
//
// El driver espera una sentencia pragma por parámetro "_pragma" (repetido);
// un valor con ";" lo interpreta como separador de sentencias SQL y rechaza
// la consulta ("invalid semicolon separator in query"), así que cada pragma
// viaja en su propio parámetro.
func dsn(dbPath string) string {
	u := url.URL{Scheme: "file", Path: filepath.ToSlash(dbPath)}
	q := url.Values{}
	q.Add("_pragma", "foreign_keys(1)")
	q.Add("_pragma", "journal_mode(WAL)")
	q.Add("_pragma", fmt.Sprintf("busy_timeout(%d)", busyTimeoutMS))
	u.RawQuery = q.Encode()
	return u.String()
}

// busyTimeoutMS es cuánto espera una escritura bloqueada antes de rendirse.
// No lo fija la documentación; se fija aquí por necesidad técnica de WAL con
// varias sesiones escribiendo (BUSINESS_RULES.md §3: "dos sesiones en segundo
// plano escriben a la vez").
const busyTimeoutMS = 5000

// Open abre (y crea si no existe) el archivo SQLite del proyecto aplicando el
// orden de arranque documentado (SEEDING.md §4): comprobar que la carpeta es un
// proyecto, crear .localcli/, abrir la base con sus PRAGMA y aplicar las
// migraciones pendientes. Devuelve la conexión lista para usar.
func Open(projectDir string) (*sql.DB, error) {
	if !isProjectReported(projectDir) {
		return nil, ErrNotAProject
	}
	dbPath, err := DBPath(projectDir)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("%w: no se pudo crear la carpeta de estado: %v", ErrNoDisponible, err)
	}
	db, err := openPath(dbPath)
	if err != nil {
		return nil, err
	}
	if err := migrate(db); err != nil {
		db.Close()
		return nil, err
	}
	if err := verifyEsquema(db); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

// maxOpenConns es cuántas conexiones simultáneas puede abrir el pool.
//
// El esquema está en modo WAL precisamente para que la interfaz lea mientras
// las sesiones de segundo plano escriben (DATABASE.md, DATA_FLOW.md §2,
// BUSINESS_RULES.md §3 "dos sesiones en segundo plano escriben a la vez"). Con
// una única conexión, una transacción abierta retiene el único connection del
// pool y cualquier lectura concurrente se queda esperando: WAL no aportaría
// nada. El pool deja pasar varias conexiones; WAL coordina a un escritor con
// varios lectores, y busy_timeout (db.go) serializa a los escritores entre sí.
const maxOpenConns = 4

// openPath abre una ruta concreta de base de datos con los PRAGMA de conexión y
// verifica que quedaron activos. Es el punto único donde se comprueban los
// pragmas (T-B002-01).
func openPath(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dsn(dbPath))
	if err != nil {
		return nil, fmt.Errorf("%w: no se pudo abrir el archivo: %v", ErrNoDisponible, err)
	}
	// Los PRAGMA viajan en el DSN, así que el driver los aplica en cada
	// conexión que el pool abra (no solo en la primera): por eso el pool
	// puede tener varias sin perder foreign_keys ni WAL.
	db.SetMaxOpenConns(maxOpenConns)
	db.SetMaxIdleConns(maxOpenConns)
	db.SetConnMaxLifetime(0)
	if err := verifyPragmas(db); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

// verifyPragmas comprueba tras abrir que journal_mode es wal y foreign_keys
// está ON. Si el modo WAL no se pudo activar (por ejemplo un sistema de
// archivos que no lo soporta), se falla en vez de funcionar en silencio.
func verifyPragmas(db *sql.DB) error {
	var mode string
	if err := db.QueryRow("PRAGMA journal_mode").Scan(&mode); err != nil {
		return fmt.Errorf("%w: no se pudo leer journal_mode: %v", ErrNoDisponible, err)
	}
	if mode != "wal" {
		return fmt.Errorf("%w: el modo WAL no está activo (journal_mode=%q)", ErrNoDisponible, mode)
	}
	var fk int
	if err := db.QueryRow("PRAGMA foreign_keys").Scan(&fk); err != nil {
		return fmt.Errorf("%w: no se pudo leer foreign_keys: %v", ErrNoDisponible, err)
	}
	if fk != 1 {
		return fmt.Errorf("%w: foreign_keys no está activo; las cascadas no se sostendrían", ErrNoDisponible)
	}
	return nil
}
