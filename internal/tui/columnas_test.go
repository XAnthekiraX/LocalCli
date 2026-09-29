package tui

// Tests del layout de la vista principal (T-F041): alineación de columnas,
// márgenes de los globos y anclaje de la caja de entrada al pie.

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattn/go-runewidth"
)

// El chat, el divisor vertical y el sidebar se componen en una sola cadena por
// fila: cada fila debe tener la MISMA anchura de pantalla hasta el divisor, o el
// sidebar se desplaza. Los caracteres anchos (emoji, ideogramas) son el caso que
// rompía: hay que contar columnas de pantalla, no runas.
func TestLasColumnasQuedanAlineadasConAnchoDePantalla(t *testing.T) {
	casos := []string{
		"hola",
		"¡Hola! 👋 ¿En qué puedo ayudarte hoy?",
		strings.Repeat("palabra ", 40),
		strings.Repeat("日本語", 30),
	}
	for _, texto := range casos {
		a := Nuevo(&puertoStub{modelo: "qwen3.5:4b"})
		a.Vista = VistaPrincipal
		a.Panel = NuevoPanel()
		a.Panel.Abierto = true
		a.Panel.SesionID = "s1"
		a.Panel.Sesion = "una sesión"
		a.Panel.Ruta = "/tmp/proyecto"
		pulsa(t, a, tea.WindowSizeMsg{Width: 80, Height: 24})
		a.Chat.AñadirUsuario(texto)
		a.Chat.AñadirAgente(texto)
		_ = a.View()

		col := a.anchoColumna()
		for i, l := range strings.Split(a.ultimaVista, "\n") {
			plano := sinANSI(l)
			j := strings.LastIndex(plano, "│")
			if j < 0 {
				t.Fatalf("texto %q: la fila %d no tiene divisor: %q", texto, i, plano)
			}
			if w := runewidth.StringWidth(plano[:j]); w != col {
				t.Errorf("texto %q: en la fila %d el divisor cae en la columna %d y debería caer en la %d: %q",
					texto, i, w, col, plano)
			}
		}
	}
}

// Los globos dejan un margen a cada lado: el del agente no toca el borde
// izquierdo y el del usuario no toca el derecho.
func TestLosGlobosDejanMargenALosLados(t *testing.T) {
	a := Nuevo(&puertoStub{})
	a.Vista = VistaPrincipal
	a.Panel.SesionID = "s1"
	pulsa(t, a, tea.WindowSizeMsg{Width: 60, Height: 16})
	a.Chat.AñadirUsuario("hola")
	a.Chat.AñadirAgente("buenas")
	v := sinEstilo(a.View())

	var agente, usuario string
	for _, l := range strings.Split(v, "\n") {
		icono, abre, cierra := strings.Index(l, "▣"), strings.Index(l, "╭"), strings.Index(l, "╮")
		if icono >= 0 && abre > icono {
			agente = l
		}
		if icono >= 0 && cierra >= 0 && cierra < icono {
			usuario = l
		}
	}
	if agente == "" || usuario == "" {
		t.Fatalf("no encontré los globos:\n%s", v)
	}
	if !strings.HasPrefix(agente, " ") || strings.HasPrefix(agente, "  ") {
		t.Errorf("el globo del agente deja margen a la izquierda: %q", agente)
	}
	if w := runewidth.StringWidth(usuario); w > 60-margenChat {
		t.Errorf("el globo del usuario deja margen a la derecha (ancho %d): %q", w, usuario)
	}
}

// La caja de entrada queda pegada abajo aunque el historial sea corto.
func TestLaCajaDeEntradaQuedaPegadaAbajo(t *testing.T) {
	for _, alto := range []int{12, 20, 30, 40} {
		a := Nuevo(&puertoStub{modelo: "m"})
		a.Vista = VistaPrincipal
		a.Panel.SesionID = "s1"
		pulsa(t, a, tea.WindowSizeMsg{Width: 80, Height: alto})
		a.Chat.AñadirSistema("un mensaje corto")

		lineas := strings.Split(strings.TrimRight(sinEstilo(a.View()), "\n"), "\n")
		if len(lineas) > alto {
			t.Fatalf("alto %d: el marco tiene %d líneas", alto, len(lineas))
		}
		fila := -1
		for i, l := range lineas {
			if strings.Contains(l, "Escribe tu petición") {
				fila = i
				break
			}
		}
		if fila < 0 {
			t.Fatalf("alto %d: no se ve la caja de entrada:\n%s", alto, strings.Join(lineas, "\n"))
		}
		if fila < alto/2 {
			t.Errorf("alto %d: la caja de entrada flota arriba (fila %d)", alto, fila)
		}
	}
}
