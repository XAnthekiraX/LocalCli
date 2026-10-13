package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// pegado es un KeyRunes con `Paste` en alto, como lo entrega el bracketed paste
// (Ctrl+Shift+V) al bucle de Bubble Tea.
func pegado(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Paste: true, Runes: []rune(s)}
}

// En la bienvenida, un pegado de varias líneas se tokeniza y, al enviar, viaja el
// texto real expandido, no el token.
func TestLaBienvenidaTokenizaElPegado(t *testing.T) {
	p := &puertoFalso{}
	app := appDePrueba(p)
	app.vista = VistaBienvenida

	pulsar(app, pegado("primera\nsegunda"))

	if !strings.Contains(app.bienvenida.Texto, "[PEGADO 2 líneas]") {
		t.Fatalf("el pegado debería verse como token: %q", app.bienvenida.Texto)
	}
	pulsar(app, tea.KeyMsg{Type: tea.KeyEnter})
	if len(p.enviados) != 1 || p.enviados[0] != "primera\nsegunda" {
		t.Fatalf("lo enviado = %v, quiero el texto expandido", p.enviados)
	}
}

// En la bienvenida, una ruta pegada se ve como su nombre y se envía entera.
func TestLaBienvenidaTokenizaUnaRuta(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "foto.png")
	if err := os.WriteFile(ruta, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	app := appDePrueba(&puertoFalso{})
	app.vista = VistaBienvenida

	pulsar(app, pegado(ruta))

	if strings.Contains(app.bienvenida.Texto, ruta) || !strings.Contains(app.bienvenida.Texto, "[foto.png]") {
		t.Fatalf("la ruta debería verse como token: %q", app.bienvenida.Texto)
	}
	if got := app.bienvenida.Peticion(); got != ruta {
		t.Fatalf("la petición = %q, quiero %q", got, ruta)
	}
}

// En la vista principal, el pegado también se tokeniza en la caja de entrada.
func TestLaPrincipalTokenizaElPegado(t *testing.T) {
	app := appDePrueba(&puertoFalso{})
	app.vista = VistaPrincipal

	pulsar(app, pegado("uno\ndos"))

	if !strings.Contains(app.entrada.Texto, "[PEGADO 2 líneas]") {
		t.Fatalf("el pegado debería verse como token: %q", app.entrada.Texto)
	}
	if got := app.entrada.Peticion(); got != "uno\ndos" {
		t.Fatalf("la petición = %q", got)
	}
}
