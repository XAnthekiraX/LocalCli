package tui

import tea "github.com/charmbracelet/bubbletea"

// Correr abre la interfaz en pantalla completa hasta que el usuario sale. Carga
// la paleta de estilos, las preferencias del usuario y su mapa de teclas antes
// de arrancar el bucle.
func Correr(puerto Puerto) error {
	app := Nueva(puerto)
	app.cargarEstilos()
	app.cargarTeclas()
	app.cargarPreferencias()
	_, err := tea.NewProgram(app, tea.WithAltScreen(), tea.WithMouseCellMotion()).Run()
	return err
}
