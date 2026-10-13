package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"localcli/internal/tui/render"
)

// Enviar desde la bienvenida crea la sesión y pasa a la vista principal con la
// petición como primer mensaje, sin repetirla ni pedir confirmación.
func TestEnviarDesdeLaBienvenidaCreaLaSesion(t *testing.T) {
	p := &puertoFalso{}
	app := appDePrueba(p)
	app.vista = VistaBienvenida
	app.bienvenida.Texto = "hola"

	pulsar(app, tea.KeyMsg{Type: tea.KeyEnter})

	if p.creadas != 1 {
		t.Fatalf("debería crear una sola sesión: %d", p.creadas)
	}
	if app.sesion == nil || app.vista != VistaPrincipal {
		t.Fatalf("no pasó a la principal con su sesión: vista=%v sesión=%+v", app.vista, app.sesion)
	}
	if app.bienvenida.Texto != "" {
		t.Fatalf("la bienvenida debería quedar limpia: %q", app.bienvenida.Texto)
	}
	if len(p.enviados) != 1 || p.enviados[0] != "hola" {
		t.Fatalf("la petición no se envió a la sesión: %v", p.enviados)
	}
	if hilo := render.SinANSI(app.chat.Hilo(80)); !strings.Contains(hilo, "hola") {
		t.Fatalf("la petición debería estar en el hilo:\n%s", hilo)
	}
}
