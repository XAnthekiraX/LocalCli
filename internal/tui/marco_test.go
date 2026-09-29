package tui

// Tests de la capa común de render (marco.go): el marco es una superficie
// rectangular continua —Ancho×Alto celdas con fondo— y sin glifos de adorno,
// que es lo que hace que la selección cubra el área y la copia traiga solo el
// texto (frontend/05-quality/TESTING.md: funciones puras, sin pantalla).

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattn/go-runewidth"
)

func TestElMarcoTieneAnchoYAltoExactos(t *testing.T) {
	bloque := "hola\n\nadios"
	marco := marcoCompleto(bloque, 20, 5)
	lineas := strings.Split(marco, "\n")
	if len(lineas) != 5 {
		t.Fatalf("el marco tiene %d filas y deberían ser 5", len(lineas))
	}
	for i, l := range lineas {
		if w := runewidth.StringWidth(sinANSI(l)); w != 20 {
			t.Errorf("la fila %d mide %d columnas y debería medir 20: %q", i, w, sinANSI(l))
		}
	}
	// El contenido no se pierde: sigue al principio del marco.
	if v := sinANSI(marco); !strings.HasPrefix(v, "hola") || !strings.Contains(v, "adios") {
		t.Errorf("el marco conserva el contenido:\n%q", v)
	}
}

func TestElMarcoSinGeometriaNoCambia(t *testing.T) {
	// Antes de la primera WindowSizeMsg no hay tamaño: el bloque se devuelve tal
	// cual para no alterar lo que ya se veía sin terminal.
	if got := marcoCompleto("hola", 0, 5); got != "hola" {
		t.Errorf("sin ancho no se toca nada: %q", got)
	}
	if got := marcoCompleto("hola", 20, 0); got != "hola" {
		t.Errorf("sin alto no se toca nada: %q", got)
	}
}

func TestElMarcoRecortaPorArribaSiSobra(t *testing.T) {
	// Si el contenido no cabe, se conservan las últimas filas, igual que el
	// renderer de Bubble Tea: así el marco guardado coincide con la pantalla.
	marco := marcoCompleto("uno\ndos\ntres\ncuatro", 10, 2)
	if v := sinANSI(marco); !strings.HasPrefix(v, "tres") || !strings.Contains(v, "cuatro") {
		t.Errorf("el marco conserva las últimas filas:\n%q", v)
	}
	if n := len(strings.Split(marco, "\n")); n != 2 {
		t.Errorf("el marco tiene %d filas y deberían ser 2", n)
	}
}

func TestPintarFondoCubreElAnchoYConservaElTexto(t *testing.T) {
	bloque := pintarFondo("hola", 8, fondoApp)
	if w := runewidth.StringWidth(sinANSI(bloque)); w != 8 {
		t.Errorf("la línea con fondo mide %d columnas y debería medir 8", w)
	}
	if v := sinANSI(bloque); v != "hola    " {
		t.Errorf("el fondo rellena a la derecha sin tocar el texto: %q", v)
	}
	// El perfil de las pruebas pinta color: el fondo deja su secuencia.
	if !strings.Contains(bloque, "\x1b[") {
		t.Errorf("el fondo deja la secuencia ANSI con el perfil de color: %q", bloque)
	}
}

// El requisito del usuario: al pintar no debe quedar ningún glifo de adorno
// (líneas, columnas, esquinas) que la selección pueda arrastrar.
func TestElMarcoNoTieneGlifosDeAdorno(t *testing.T) {
	a := Nuevo(&puertoStub{modelo: "m"})
	a.Vista = VistaPrincipal
	a.Panel.SesionID = "s1"
	pulsa(t, a, tea.WindowSizeMsg{Width: 80, Height: 24})
	a.Chat.AñadirUsuario("hola")
	a.Chat.AñadirAgente("buenas")

	if v := sinEstilo(a.View()); strings.ContainsAny(v, "─│╭╮╰╯") {
		t.Errorf("el marco no debe llevar glifos de adorno:\n%s", v)
	}
}
