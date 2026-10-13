package session

import (
	"os"
	"path/filepath"
	"strings"

	"localcli/internal/tui"
)

// leerGit devuelve la rama del proyecto leyendo `.git/HEAD`. Una rama vacía dice
// que no hay git inicializado.
func leerGit(cwd string) tui.Git {
	datos, err := os.ReadFile(filepath.Join(cwd, ".git", "HEAD"))
	if err != nil {
		return tui.Git{}
	}
	ref := strings.TrimSpace(string(datos))
	if rama, ok := strings.CutPrefix(ref, "ref: refs/heads/"); ok {
		return tui.Git{Rama: rama}
	}
	if len(ref) >= 7 {
		return tui.Git{Rama: ref[:7]}
	}
	return tui.Git{}
}
