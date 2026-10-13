package arranque

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

var ErrNoIniciada = errors.New("localcli no iniciada")

// RaizMaquina devuelve la ruta de la raíz de la máquina: ~/.config/localcli/
// Se resuelve usando os.UserConfigDir() que respeta XDG_CONFIG_HOME.
func RaizMaquina() (string, error) {
	if override := os.Getenv("LOCALCLI_CONFIG_DIR"); override != "" {
		return filepath.Join(override, "localcli"), nil
	}
	configDir, err := os.UserConfigDir()
	if err != nil {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("arranque: no se pudo resolver la carpeta del usuario: %w", err)
		}
		return filepath.Join(home, ".config", "localcli"), nil
	}
	return filepath.Join(configDir, "localcli"), nil
}

// EstaInicializada comprueba si la raíz de la máquina existe.
func EstaInicializada() bool {
	raiz, err := RaizMaquina()
	if err != nil {
		return false
	}
	info, err := os.Stat(raiz)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// RaizProyecto devuelve la ruta del directorio .localcli en el proyecto actual.
func RaizProyecto(cwd string) string {
	if cwd == "" {
		cwd, _ = os.Getwd()
	}
	return filepath.Join(cwd, ".localcli")
}
