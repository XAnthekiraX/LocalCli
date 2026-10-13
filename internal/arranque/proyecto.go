package arranque

import (
	"os"
	"path/filepath"
)

// Create crea la configuración del proyecto en la carpeta actual: <cwd>/.localcli/
func Create(cwd string) error {
	if cwd == "" {
		var err error
		cwd, err = os.Getwd()
		if err != nil {
			return err
		}
	}

	projRoot := filepath.Join(cwd, ".localcli")
	if err := os.MkdirAll(projRoot, 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(projRoot, "agents"), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(projRoot, "flows"), 0o755); err != nil {
		return err
	}
	return nil
}
