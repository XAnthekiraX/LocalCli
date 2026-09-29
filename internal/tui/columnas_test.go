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

		// La superficie es continua: cada fila mide exactamente el ancho de la
		// terminal. El panel arranca en su columna, tras los dos espacios de
		// separación.
		filas := strings.Split(sinANSI(a.ultimaVista), "\n")
		for i, l := range filas {
			if w := runewidth.StringWidth(l); w != a.Ancho {
				t.Errorf("texto %q: la fila %d mide %d columnas y debería medir %d: %q",
					texto, i, w, a.Ancho, l)
			}
		}
		col := a.anchoColumna()
		inicio := -1
		for _, l := range filas {
			if j := strings.Index(l, "una sesión"); j >= 0 {
				inicio = runewidth.StringWidth(l[:j])
				break
			}
		}
		if inicio != col+4 {
			t.Errorf("texto %q: el sidebar debe empezar en la columna %d y empieza en la %d",
				texto, col+4, inicio)
		}
	}
}

// Cada mensaje ocupa el ancho del chat, sin icono: el del usuario pegado a la
// derecha (su color termina en el borde de la columna) y el del agente a la
// izquierda.
func TestLosMensajesOcupanElAnchoDelChat(t *testing.T) {
	a := Nuevo(&puertoStub{})
	a.Vista = VistaPrincipal
	a.Panel.SesionID = "s1"
	pulsa(t, a, tea.WindowSizeMsg{Width: 80, Height: 24})
	a.Chat.AñadirUsuario("hola")
	a.Chat.AñadirAgente("buenas")
	v := sinEstilo(a.View())

	var agente, usuario string
	for _, l := range strings.Split(v, "\n") {
		if strings.Contains(l, "buenas") {
			agente = l
		}
		if strings.Contains(l, "hola") {
			usuario = l
		}
	}
	if agente == "" || usuario == "" {
		t.Fatalf("no encontré los mensajes:\n%s", v)
	}
	if strings.Contains(v, "▣") {
		t.Error("los mensajes ya no llevan icono")
	}
	if got := columnaPrincipal(a, agente); !strings.HasPrefix(strings.TrimLeft(got, " "), "buenas") {
		t.Errorf("el mensaje del agente va pegado a la izquierda: %q", got)
	}
	if got := columnaPrincipal(a, usuario); !strings.HasSuffix(got, "hola") {
		t.Errorf("el mensaje del usuario va pegado a la derecha: %q", got)
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
			if strings.Contains(l, "Escribe") {
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
