package exec

import (
	"path/filepath"
	"strings"
)

type regla struct {
	programa    string
	subcomandos []string
}

// ListaBlanca es la lista cerrada de comandos que corren sin preguntar.
var ListaBlanca = []regla{
	{programa: "go", subcomandos: []string{"build", "test", "vet"}},
	{programa: "gofmt"},
	{programa: "git", subcomandos: []string{"status", "diff", "log"}},
}

// EnListaBlanca dice si un comando ya partido en argv se ejecuta sin aprobación.
func EnListaBlanca(argv []string) bool {
	if len(argv) == 0 {
		return false
	}
	programa := filepath.Base(argv[0])
	for _, r := range ListaBlanca {
		if programa != r.programa {
			continue
		}
		if len(r.subcomandos) == 0 {
			return !escribeGofmt(argv[1:])
		}
		sub := primerSubcomando(argv[1:])
		return sub != "" && contiene(r.subcomandos, sub)
	}
	return false
}

func primerSubcomando(args []string) string {
	for _, a := range args {
		if !strings.HasPrefix(a, "-") {
			return a
		}
	}
	return ""
}

func escribeGofmt(args []string) bool {
	for _, a := range args {
		if a == "-w" || a == "-i" || strings.HasPrefix(a, "-w=") {
			return true
		}
		if strings.HasPrefix(a, "-") && !strings.HasPrefix(a, "--") &&
			strings.Contains(strings.TrimPrefix(a, "-"), "w") {
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
