package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// El interruptor de razonamiento llega a la sesión: apagado responde directo,
// encendido deja pensar al modelo.
func TestElInterruptorDePensarLlegaAlPuerto(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	p := &puertoFalso{}
	app := appDePrueba(p)

	if app.entrada.Pensar {
		t.Fatal("debería empezar apagado")
	}
	app.alternarPensar()
	if !app.entrada.Pensar || !p.pensado {
		t.Fatalf("encender no llegó al puerto: entrada=%v puerto=%v", app.entrada.Pensar, p.pensado)
	}
	app.alternarPensar()
	if app.entrada.Pensar || p.pensado {
		t.Fatalf("apagar no llegó al puerto: entrada=%v puerto=%v", app.entrada.Pensar, p.pensado)
	}
}

// El chip de la bienvenida es clicable igual que el de la vista principal.
func TestElInterruptorDePensarFuncionaEnLaBienvenida(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	p := &puertoFalso{}
	app := appDePrueba(p)
	app.vista = VistaBienvenida
	_ = app.View()

	z := app.zonaPensar
	if z.Alto == 0 || z.Ancho == 0 {
		t.Fatal("la bienvenida debería exponer el chip de razonamiento")
	}
	_, _ = app.raton(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonLeft, X: z.Columna, Y: z.Fila})
	if !app.entrada.Pensar || !p.pensado {
		t.Fatalf("el clic en la bienvenida no encendió el razonamiento: entrada=%v puerto=%v", app.entrada.Pensar, p.pensado)
	}
}
