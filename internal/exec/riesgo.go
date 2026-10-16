package exec

import (
	"path/filepath"
	"strings"
)

// programasPeligrosos son los programas que destruyen datos: los que borran,
// machacan o apagan. El resto de la terminal no necesita confirmación.
var programasPeligrosos = []string{
	"rm", "rmdir", "shred", "unlink", "truncate", "mv", "dd", "wipefs",
	"kill", "pkill", "shutdown", "reboot",
}

// Peligroso dice si un comando ya partido en argv es destructivo y por tanto
// exige confirmación explícita. Un programa que se nombra con prefijo `mkfs`
// lo es sin figurar en la lista ([[specs/SPEC-TOOLS]] §Control 1).
func Peligroso(argv []string) bool {
	if len(argv) == 0 {
		return false
	}
	programa := filepath.Base(argv[0])
	if strings.HasPrefix(programa, "mkfs") {
		return true
	}
	if programa == "git" {
		return gitPeligroso(argv[1:])
	}
	return contiene(programasPeligrosos, programa)
}

// gitPeligroso marca los subcomandos de git que descartan trabajo o lo suben a
// la fuerza: el resto de git (`status`, `diff`, `log`, `commit`) no lo es.
func gitPeligroso(args []string) bool {
	switch primerSubcomando(args) {
	case "reset", "clean", "checkout", "restore":
		return true
	case "branch":
		return tiene(args, "-D")
	case "push":
		return tiene(args, "-f") || tiene(args, "--force") || tiene(args, "--force-with-lease")
	}
	return false
}

// FueraDelProyecto dice si algún argumento del comando es una ruta absoluta que
// queda fuera de la carpeta del proyecto: ahí sí se pide confirmación.
func FueraDelProyecto(raiz string, argv []string) bool {
	if raiz == "" {
		return false
	}
	for _, a := range argv[1:] {
		if !filepath.IsAbs(a) {
			continue
		}
		if !dentro(raiz, a) {
			return true
		}
	}
	return false
}

// PideConfirmacion es la regla única de la terminal: destructivo o fuera de la
// carpeta; cualquier otro comando corre sin preguntar.
func PideConfirmacion(raiz string, argv []string) bool {
	return Peligroso(argv) || FueraDelProyecto(raiz, argv)
}

// dentro dice si una ruta absoluta cuelga de la raíz.
func dentro(raiz, abs string) bool {
	rel, err := filepath.Rel(raiz, abs)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

func primerSubcomando(args []string) string {
	for _, a := range args {
		if !strings.HasPrefix(a, "-") {
			return a
		}
	}
	return ""
}

func tiene(args []string, valor string) bool {
	for _, a := range args {
		if a == valor {
			return true
		}
	}
	return false
}

func contiene(lista []string, valor string) bool {
	for _, v := range lista {
		if v == valor {
			return true
		}
	}
	return false
}
