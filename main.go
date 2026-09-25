// localcli — binario único del backend de LocalCli.
// La carpeta desde la que se ejecuta es el proyecto (CONFIGURATION.md §1 y §4).
package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	_ "modernc.org/sqlite" // driver puro Go, sin cgo (DECISIONS.md)
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

	// Verificación mínima del stack: el driver puro Go abre SQLite sin cgo.
	// El esquema completo llega con store (T-B002); aquí solo se confirma
	// que la ruta derivada .localcli/state.db funciona.
	dbPath := filepath.Join(cwd, ".localcli", "state.db")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "error al preparar .localcli/:", err)
		os.Exit(1)
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error al abrir la base:", err)
		os.Exit(1)
	}
	if err := db.Ping(); err != nil {
		fmt.Fprintln(os.Stderr, "error al verificar la base:", err)
		db.Close()
		os.Exit(1)
	}
	db.Close()

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
