package exec

import (
	"fmt"
	"path/filepath"
	"strings"

	"localcli/internal/tools"
)

// Salida es el resultado estructurado de un comando.
type Salida struct {
	Salida   string
	Error    string
	Codigo   int
	Truncado bool
	Termino  bool
}

// Texto compone lo que ve el modelo: la salida, el error si lo hay y, cuando el
// comando no terminó bien, su código.
func (s Salida) Texto() string {
	var b strings.Builder
	b.WriteString(s.Salida)
	if s.Error != "" {
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		b.WriteString(s.Error)
	}
	if s.Codigo != 0 {
		fmt.Fprintf(&b, "\n[código %d]", s.Codigo)
	}
	if s.Truncado {
		b.WriteString("\n[la salida se cortó por tamaño o tiempo]")
	}
	return b.String()
}

var shells = []string{
	"sh", "bash", "dash", "zsh", "ksh", "csh", "tcsh", "fish", "ash",
	"cmd", "cmd.exe", "powershell", "powershell.exe", "pwsh",
}

const metacaracteres = "|&;<>()`$"

// PartirComando convierte el texto en argv sin usar un shell. Rechaza lo que
// necesita un intérprete, que serían argumentos literales de todos modos.
func PartirComando(comando string) ([]string, error) {
	limpio := strings.TrimSpace(comando)
	if limpio == "" {
		return nil, nuevaError(tools.CodBadArgs, "el comando está vacío")
	}
	if strings.ContainsAny(limpio, metacaracteres) {
		return nil, nuevaError(tools.CodBadArgs,
			"el comando usa un shell (tubería, redirección o sustitución); la terminal no interpreta un shell")
	}
	argv := strings.Fields(limpio)
	if len(argv) == 0 {
		return nil, nuevaError(tools.CodBadArgs, "el comando está vacío")
	}
	if esShell(argv[0]) {
		return nil, nuevaError(tools.CodBadArgs, "no se ejecuta un shell: pide el programa directamente")
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
