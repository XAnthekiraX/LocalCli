// localcli — binario único del backend de LocalCli.
// La carpeta desde la que se ejecuta es el proyecto (CONFIGURATION.md §1 y §4).
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"localcli/internal/store"
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

	// store es el ÚNICO punto de apertura y escritura de SQLite
	// (DECISIONS.md: "store es el único que escribe en SQLite"). Aquí no se
	// importa database/sql ni se abre el driver: store.Open aplica los PRAGMA
	// de conexión, crea .localcli/ y ejecuta las migraciones por user_version.
	db, err := store.Open(cwd)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error al preparar la base del proyecto:", err)
		os.Exit(1)
	}
	defer db.Close()

	// El módulo tui completo llega en T-B014; aquí solo se confirma que
	// Bubble Tea + Lip Gloss compilan y corren en este stack.
	title := lipgloss.NewStyle().Bold(true).Render("LocalCli")
	p := tea.NewProgram(newRootModel(title))
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error en la interfaz:", err)
		os.Exit(1)
	}
}

// newRootModel devuelve el modelo raíz provisional de la TUI.
func newRootModel(title string) tea.Model {
	return rootModel{title: title}
}

type rootModel struct {
	title string
}

func (m rootModel) Init() tea.Cmd { return nil }

func (m rootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m rootModel) View() string {
	return m.title + "\n\nPulsa q para salir.\n"
}
