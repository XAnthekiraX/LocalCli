package tui

// Test de la edición del cursor en la petición de bienvenida (welcome.go).

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestLaBienvenidaEditaEnCualquierPunto(t *testing.T) {
	a := Nuevo(&puertoStub{})
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})
	escribe(t, a, "hole como estas")

	// Retroceder seis caracteres y escribir en medio.
	for i := 0; i < 6; i++ {
		tecla(t, a, tea.KeyLeft)
	}
	escribe(t, a, "X")
	if a.Bienvenida.Texto != "hole comoX estas" {
		t.Fatalf("inserta donde está el cursor: %q", a.Bienvenida.Texto)
	}
	if !strings.Contains(sinEstilo(a.View()), "comoX▌") {
		t.Errorf("el cursor se pinta en su posición:\n%s", sinEstilo(a.View()))
	}

	// Home y End.
	tecla(t, a, tea.KeyHome)
	escribe(t, a, ">")
	if a.Bienvenida.Texto != ">hole comoX estas" {
		t.Errorf("home lleva el cursor al principio: %q", a.Bienvenida.Texto)
	}
	tecla(t, a, tea.KeyEnd)
	tecla(t, a, tea.KeyBackspace)
	if a.Bienvenida.Texto != ">hole comoX esta" {
		t.Errorf("end y retroceso borran el final: %q", a.Bienvenida.Texto)
	}

	// Suprimir borra el carácter del cursor.
	tecla(t, a, tea.KeyHome)
	tecla(t, a, tea.KeyDelete)
	if a.Bienvenida.Texto != "hole comoX esta" {
		t.Errorf("suprimir borra el carácter del cursor: %q", a.Bienvenida.Texto)
	}
}
