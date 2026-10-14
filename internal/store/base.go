package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// RutaGlobal devuelve la ruta del archivo SQLite global.
func RutaGlobal() (string, error) {
	if path := os.Getenv("LOCALCLI_DB_PATH"); path != "" {
		return path, nil
	}
	configDir, err := os.UserConfigDir()
	if err != nil {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("store: no se pudo resolver la carpeta del usuario: %w", err)
		}
		return filepath.Join(home, ".config", "localcli", "state.db"), nil
	}
	return filepath.Join(configDir, "localcli", "state.db"), nil
}

// CrearDB crea una nueva base de datos global con el esquema v8.
func CrearDB(ruta string) (*sql.DB, error) {
	dir := filepath.Dir(ruta)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", ruta+"?_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)")
	if err != nil {
		return nil, err
	}

	if err := initSchema(db); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

// AbrirDB abre la base de datos global.
func AbrirDB() (*sql.DB, error) {
	ruta, err := RutaGlobal()
	if err != nil {
		return nil, err
	}
	return AbrirDBDesde(ruta)
}

// AbrirDBDesde abre la base de datos desde una ruta específica.
func AbrirDBDesde(ruta string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", ruta+"?_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)")
	if err != nil {
		return nil, err
	}

	// Aplicar migraciones a la versión objetivo
	if err := Migrar(db, versionEsquema); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

// CrearDBTemporal crea una base de datos temporal aislada para pruebas.
// Devuelve la base de datos y una función para limpiar (eliminar el archivo).
func CrearDBTemporal() (*sql.DB, func() error, error) {
	dir, err := os.MkdirTemp("", "localcli-test-db-*")
	if err != nil {
		return nil, nil, err
	}

	ruta := filepath.Join(dir, "test.db")
	db, err := CrearDB(ruta)
	if err != nil {
		os.RemoveAll(dir)
		return nil, nil, err
	}

	cleanup := func() error {
		db.Close()
		return os.RemoveAll(dir)
	}

	return db, cleanup, nil
}
