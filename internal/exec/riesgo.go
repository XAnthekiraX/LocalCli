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

// programasEgreso sacan información de la máquina, que es justo lo que se
// quiere revisar antes de que ocurra: sin ellos, la única puerta de salida
// son las herramientas de internet ([[specs/SPEC-TOOLS]] §Herramientas de
// internet).
var programasEgreso = []string{
	"curl", "wget", "nc", "ncat", "netcat", "telnet",
	"ssh", "scp", "sftp", "rsync",
}

// Peligroso dice si un comando ya partido en argv es destructivo o hace salir
// información de la máquina, y por tanto exige confirmación explícita. Un
// programa que se nombra con prefijo `mkfs` lo es sin figurar en la lista
// ([[specs/SPEC-TOOLS]] §Control 1).
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
	return contiene(programasPeligrosos, programa) || contiene(programasEgreso, programa)
}

// gitPeligroso marca los subcomandos de git que descartan trabajo o lo suben a
// la fuerza, y `push` —publicar no se deshace—: el resto de git (`status`,
// `diff`, `log`, `commit`) no lo es.
func gitPeligroso(args []string) bool {
	switch primerSubcomando(args) {
	case "push":
		return true
	case "reset", "clean", "checkout", "restore":
		return true
	case "branch":
		return tiene(args, "-D")
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

// PideConfirmacion es la regla única de la terminal: destructivo, con salida de
// red o fuera de la carpeta; cualquier otro comando corre sin preguntar.
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
