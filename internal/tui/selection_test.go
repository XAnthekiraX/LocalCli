package tui

// Tests de la selección con el ratón y su copia (selection.go) y del scroll con
// la rueda.

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestTextoSeleccionadoExtraeElRangoSinCódigos(t *testing.T) {
	// Los códigos de color no cuentan como columnas.
	vista := "\x1b[31mhola\x1b[0m mundo\nsegunda línea"
	if got := textoSeleccionado(vista, posicion{X: 0, Y: 0}, posicion{X: 5, Y: 0}); got != "hola" {
		t.Errorf("selección de una línea: %q", got)
	}
	// Varias líneas y extremos en cualquier orden (se ordenan solos).
	got := textoSeleccionado(vista, posicion{X: 3, Y: 1}, posicion{X: 0, Y: 0})
	if got != "hola mundo\nseg" {
		t.Errorf("selección multilínea con extremos invertidos: %q", got)
	}
	// Un clic sin arrastrar no copia nada.
	if got := textoSeleccionado(vista, posicion{X: 2, Y: 0}, posicion{X: 2, Y: 0}); got != "" {
		t.Errorf("un clic sin arrastre no selecciona: %q", got)
	}
	// Un marco vacío no revienta.
	if got := textoSeleccionado("", posicion{X: 0, Y: 0}, posicion{X: 1, Y: 1}); got != "" {
		t.Errorf("vista vacía: %q", got)
	}
}

func TestElArrastreDelRatónCopiaLaSelección(t *testing.T) {
	var copiado []string
	original := copiarFunc
	copiarFunc = func(s string) { copiado = append(copiado, s) }
	defer func() { copiarFunc = original }()

	a := Nuevo(&puertoStub{})
	a.Vista = VistaPrincipal
	a.Panel.SesionID = "s1"
	pulsa(t, a, tea.WindowSizeMsg{Width: 60, Height: 12})
	a.ultimaVista = "hola mundo\nsegunda línea"

	pulsa(t, a, tea.MouseMsg{Button: tea.MouseButtonLeft, Action: tea.MouseActionPress, X: 0, Y: 0})
	pulsa(t, a, tea.MouseMsg{Button: tea.MouseButtonLeft, Action: tea.MouseActionMotion, X: 5, Y: 0})
	pulsa(t, a, tea.MouseMsg{Button: tea.MouseButtonLeft, Action: tea.MouseActionRelease, X: 5, Y: 0})
	if len(copiado) != 1 || copiado[0] != "hola" {
		t.Fatalf("al soltar se copia el texto seleccionado: %v", copiado)
	}
	if a.ratonSelec {
		t.Error("al soltar termina la selección")
	}

	// Un clic sin arrastrar no copia.
	pulsa(t, a, tea.MouseMsg{Button: tea.MouseButtonLeft, Action: tea.MouseActionPress, X: 2, Y: 0})
	pulsa(t, a, tea.MouseMsg{Button: tea.MouseButtonLeft, Action: tea.MouseActionRelease, X: 2, Y: 0})
	if len(copiado) != 1 {
		t.Error("un clic sin arrastre no copia")
	}
}

func TestLaRuedaDesplazaElHistorial(t *testing.T) {
	a := Nuevo(&puertoStub{})
	a.Vista = VistaPrincipal
	a.Panel.SesionID = "s1"
	pulsa(t, a, tea.WindowSizeMsg{Width: 60, Height: 12})
	for i := 1; i <= 40; i++ {
		a.Chat.AñadirSistema(fmt.Sprintf("mensaje %d", i))
	}
	_ = a.View()
	if a.Chat.OcultasAbajo() != 0 {
		t.Fatalf("al pie no queda nada abajo, quedan %d", a.Chat.OcultasAbajo())
	}

	// La rueda hacia arriba sube por el historial.
	pulsa(t, a, tea.MouseMsg{Button: tea.MouseButtonWheelUp, Action: tea.MouseActionPress})
	_ = a.View()
	if a.Chat.OcultasAbajo() == 0 {
		t.Error("la rueda hacia arriba sube por el historial")
	}

	// La rueda hacia abajo vuelve al final.
	for i := 0; i < 40; i++ {
		pulsa(t, a, tea.MouseMsg{Button: tea.MouseButtonWheelDown, Action: tea.MouseActionPress})
	}
	if a.Chat.OcultasAbajo() != 0 || !strings.Contains(sinEstilo(a.View()), "mensaje 40") {
		t.Error("la rueda hacia abajo vuelve al final")
	}
}
