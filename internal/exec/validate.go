package exec

// validate.go — T-B009-02: validar el comando parseando argv.
//
// Fuente de verdad: ai/docs/backend/03-security/SECURITY.md §6 ("Confiar en el
// texto del comando es un error") y ai/docs/backend/02-interfaces/TOOLS.md §5
// (el bloqueo no depende de revisar el texto; la lista blanca decide qué pide
// aprobación).
//
// La ejecución no usa un shell: el comando se parte en argv y se lanza el
// programa directamente. Eso ya impide que `|`, `>` o `tee` hagan algo: serían
// argumentos literales. Aun así, si el comando pide un shell o usa composición,
// se rechaza con claridad en vez de ejecutar algo distinto de lo que el modelo
// pretendía. `sh -c`, tuberías y redirecciones no pasan.

import (
	"path/filepath"
	"strings"
)

// shells son los intérpretes que podrían recomponer un comando arbitrario. Si
// el ejecutable es uno de estos, no se ejecuta: la lista blanca no lo cubre.
var shells = []string{
	"sh", "bash", "dash", "zsh", "ksh", "csh", "tcsh", "fish", "ash",
	"cmd", "cmd.exe", "powershell", "powershell.exe", "pwsh",
}

// metacaracteres son los que implican composición de shell. No se usan para
// decidir si el comando es seguro, solo para rechazar lo que necesita un shell
// y no se puede ejecutar tal cual.
const metacaracteres = "|&;<>()`$"

// Validar convierte el comando en argv y rechaza lo que no se puede ejecutar
// sin un shell. Devuelve E_BAD_ARGS si el comando está vacío o necesita shell.
func Validar(comando string) ([]string, error) {
	limpio := strings.TrimSpace(comando)
	if limpio == "" {
		return nil, nuevoError(CodigoArgumentosInvalidos, "el comando está vacío")
	}
	if strings.ContainsAny(limpio, metacaracteres) {
		return nil, nuevoError(CodigoArgumentosInvalidos,
			"el comando usa un shell (tubería, redirección o sustitución); la terminal no interpreta un shell")
	}
	argv := strings.Fields(limpio)
	if len(argv) == 0 {
		return nil, nuevoError(CodigoArgumentosInvalidos, "el comando está vacío")
	}
	if esShell(argv[0]) {
		return nil, nuevoError(CodigoArgumentosInvalidos,
			"no se ejecuta un shell: pide el programa directamente")
	}
	return argv, nil
}

func esShell(programa string) bool {
	base := strings.ToLower(filepath.Base(programa))
	for _, s := range shells {
		if base == s {
			return true
		}
	}
	return false
}
