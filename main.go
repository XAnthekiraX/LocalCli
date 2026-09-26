// localcli — binario único del backend de LocalCli.
// La carpeta desde la que se ejecuta es el proyecto (CONFIGURATION.md §1 y §4).
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--help" {
		fmt.Println("uso: localcli\n\nArranca LocalCli en la carpeta actual.")
		return
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error al resolver la carpeta actual:", err)
		os.Exit(1)
	}

	// El cableado completo vive en arranque.go: ahí se abre la base (por
	// `store`, el único escritor SQLite), se monta el motor contra Ollama
	// local, se crea o retoma la sesión activa del proyecto y se ata todo al
	// `tui.Puerto` de producción. main solo levanta y baja el telón.
	a, err := nuevoArranque(cwd)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error al preparar la base del proyecto:", err)
		os.Exit(1)
	}
	defer a.Cerrar()

	p := tea.NewProgram(nuevaApp(a), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error en la interfaz:", err)
		os.Exit(1)
	}
}
