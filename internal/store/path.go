package store

import (
	"fmt"
	"os"
	"path/filepath"
)

// dbDirName es la carpeta de estado del proyecto, fuera de git (DECISIONS.md:
// "Archivo SQLite en .localcli/state.db dentro del proyecto").
const dbDirName = ".localcli"

// dbFileName es el nombre fijo del archivo SQLite (DATABASE.md: "cada proyecto
// tiene su propio archivo SQLite, con nombre fijo y ubicación dentro de la
// carpeta de estado del proyecto").
const dbFileName = "state.db"

// EnvDBPath sobrescribe la ruta del archivo SQLite del proyecto
// (CONFIGURATION.md §2). En el uso normal no hace falta.
const EnvDBPath = "LOCALCLI_DB_PATH"

// DBPath deriva la ruta del archivo SQLite desde la carpeta del proyecto
// (T-B002-02). Si LOCALCLI_DB_PATH está definida, esa es la ruta; si no,
// es <proyecto>/.localcli/state.db.
func DBPath(projectDir string) (string, error) {
	if p := os.Getenv(EnvDBPath); p != "" {
		return p, nil
	}
	info, err := os.Stat(projectDir)
	if err != nil {
		return "", fmt.Errorf("carpeta del proyecto inaccesible: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("la carpeta del proyecto no es un directorio: %s", projectDir)
	}
	return filepath.Join(projectDir, dbDirName, dbFileName), nil
}
