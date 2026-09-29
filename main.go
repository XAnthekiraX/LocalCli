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
	// local, se deja listo el ciclo de sesiones (sin crear ninguna hasta la
	// primera petición) y se ata todo al `tui.Puerto` de producción. main solo
	// levanta y baja el telón.
	a, err := nuevoArranque(cwd)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error al preparar la base del proyecto:", err)
		os.Exit(1)
	}
	defer a.Cerrar()

	// Se captura el ratón para poder seleccionar texto y copiarlo al portapapeles
	// (selection.go) y para desplazar el chat con la rueda. La selección nativa
	// de la terminal sigue disponible manteniendo Shift. El compresor ANSI funde
	// las secuencias de color consecutivas: la superficie continua (marco.go)
	// repite el fondo en cada celda y así el repintado viaja más compacto.
	p := tea.NewProgram(nuevaApp(a), tea.WithAltScreen(), tea.WithMouseCellMotion(), tea.WithANSICompressor())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error en la interfaz:", err)
		os.Exit(1)
	}
}
