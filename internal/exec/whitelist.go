package exec

// whitelist.go — T-B009-01: la lista blanca cerrada de comandos.
//
// Fuente de verdad: ai/docs/backend/02-interfaces/TOOLS.md §5 (control 1) y
// ai/docs/backend/01-domain/BUSINESS_RULES.md §Herramientas y terminal.
// "Se ejecutan sin preguntar, porque son los que se repiten en cada iteración:
// compilar el proyecto, correr las pruebas, revisar estilo y tipos, ver estado
// y diferencias del repositorio." Cualquier otro comando pide aprobación.
//
// La lista es cerrada: se identifica el programa por su ejecutable y, cuando
// aplica, por su subcomando. No se inspecciona el texto para decidir si es
// seguro —eso sería frágil—, solo para saber si está en la lista.

import (
	"path/filepath"
	"strings"
)

// regla es un programa permitido y, opcionalmente, los subcomandos permitidos.
// Sin subcomandos, el programa entero está permitido (con los flags que la
// regla admita).
type regla struct {
	programa    string
	subcomandos []string
}

// listaBlanca es la lista cerrada. `go` cubre compilar, probar y revisar tipos;
// `gofmt` cubre el estilo; `git` cubre estado, diferencias e historial.
var listaBlanca = []regla{
	{programa: "go", subcomandos: []string{"build", "test", "vet"}},
	{programa: "gofmt"},
	{programa: "git", subcomandos: []string{"status", "diff", "log"}},
}

// EnListaBlanca informa si el comando se ejecuta sin preguntar. `argv` ya viene
// validado (sin shell ni composición).
func EnListaBlanca(argv []string) bool {
	if len(argv) == 0 {
		return false
	}
	programa := filepath.Base(argv[0])
	for _, r := range listaBlanca {
		if programa != r.programa {
			continue
		}
		if len(r.subcomandos) == 0 {
			// gofmt escribe con -w/-i; cualquier otra forma solo revisa.
			return !escribeGofmt(argv[1:])
		}
		return primerSubcomando(argv[1:]) != "" && contiene(r.subcomandos, primerSubcomando(argv[1:]))
	}
	return false
}

// primerSubcomando devuelve el primer argumento que no es un flag.
func primerSubcomando(args []string) string {
	for _, a := range args {
		if !strings.HasPrefix(a, "-") {
			return a
		}
	}
	return ""
}

// escribeGofmt informa si `gofmt` va a reescribir archivos (-w) o aplicar -i.
func escribeGofmt(args []string) bool {
	for _, a := range args {
		if a == "-w" || a == "-i" || strings.HasPrefix(a, "-w=") {
			return true
		}
		if strings.HasPrefix(a, "-") && !strings.HasPrefix(a, "--") {
			// Combinaciones cortas tipo -lw incluyen la w.
			if strings.Contains(strings.TrimPrefix(a, "-"), "w") {
				return true
			}
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
