package tui

// Arnés de pruebas de componente (T-F011-02): programa Bubble Tea in-process
// al que se le inyectan mensajes sintéticos y del que se lee la vista. Sin
// base de datos, sin red y sin modelo: el puerto es el doble del paquete
// (frontend/05-quality/TESTING.md §1 y §3).
//
// Los helpers sueltos (pulsa, tecla, escribe, ejecuta, sinEstilo) viven en
// tui_test.go; este arnés los reúne detrás de una interfaz pequeña para los
// recorridos completos de T-F011-03.

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// arnes monta la aplicación en la interfaz principal, con tamaño conocido y
// la sesión activa puesta, listo para recorrer pantallas.
type arnes struct {
	t      *testing.T
	app    *App
	puerto *puertoStub
}

func nuevoArnes(t *testing.T) *arnes {
	t.Helper()
	p := &puertoStub{}
	a := Nuevo(p)
	a.Vista = VistaPrincipal
	a.Panel.SesionID = "s1"
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})
	return &arnes{t: t, app: a, puerto: p}
}

// tecla pulsa una tecla con nombre de Bubble Tea ("ctrl+d", "enter", "esc",
// "up", "down" o una letra suelta), igual que llegaría del teclado real.
func (h *arnes) tecla(nombre string) {
	h.t.Helper()
	pulsa(h.t, h.app, teclaConNombre(nombre))
}

// teclaConNombre traduce el nombre de la tecla al mensaje de Bubble Tea.
func teclaConNombre(nombre string) tea.KeyMsg {
	tipos := map[string]tea.KeyType{
		"enter": tea.KeyEnter, "esc": tea.KeyEsc, "up": tea.KeyUp,
		"down": tea.KeyDown, "backspace": tea.KeyBackspace, "space": tea.KeySpace,
		"ctrl+c": tea.KeyCtrlC, "ctrl+q": tea.KeyCtrlQ, "ctrl+d": tea.KeyCtrlD,
		"ctrl+s": tea.KeyCtrlS, "ctrl+r": tea.KeyCtrlR, "ctrl+a": tea.KeyCtrlA,
		"ctrl+f": tea.KeyCtrlF,
	}
	if tipo, ok := tipos[nombre]; ok {
		return tea.KeyMsg{Type: tipo}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(nombre)}
}

// escribe compone texto carácter a carácter, como el teclado real.
func (h *arnes) escribe(texto string) {
	h.t.Helper()
	escribe(h.t, h.app, texto)
}

// evento inyecta un evento del motor por el mismo camino que el bucle.
func (h *arnes) evento(nombre string, datos map[string]string) {
	h.t.Helper()
	pulsa(h.t, h.app, eventoMsg{Evento: Evento{Nombre: nombre, Datos: datos}})
}

// ve devuelve la vista tal como la lee el ojo: sin códigos de color.
func (h *arnes) ve() string {
	h.t.Helper()
	return sinEstilo(h.app.View())
}

// veSiContiene falla la prueba si la vista no contiene el texto.
func (h *arnes) veSiContiene(quiere string) {
	h.t.Helper()
	if !strings.Contains(h.ve(), quiere) {
		h.t.Errorf("la vista debe contener %q:\n%s", quiere, h.ve())
	}
}

// TestElArnesInyectaYLee comprueba el propio arnés: inyecta un mensaje y lee
// el efecto en la vista.
func TestElArnesInyectaYLee(t *testing.T) {
	h := nuevoArnes(t)
	h.escribe("hola arnés")
	h.veSiContiene("hola arnés")

	h.evento(EventoEtapaIniciada, map[string]string{"etapa": "prueba"})
	h.veSiContiene("etapa iniciada: prueba")

	h.tecla("ctrl+d")
	if !h.app.Panel.Abierto {
		t.Error("el arnés pulsa atajos del mapa real")
	}
}
