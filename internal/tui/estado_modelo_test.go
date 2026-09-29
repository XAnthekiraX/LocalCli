package tui

// Tests de la línea de estado del modelo bajo el input (app.go): el agente, el
// modelo y sus chapas de capacidad.

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// secuenciaDe devuelve la secuencia ANSI que enciende el color de un estilo.
func secuenciaDe(estilo lipgloss.Style) string {
	const marca = "\uE000"
	s := estilo.Render(marca)
	if i := strings.Index(s, marca); i >= 0 {
		return s[:i]
	}
	return ""
}

func TestLaLineaDeEstadoMuestraModeloYHerramientas(t *testing.T) {
	p := &puertoStub{modelo: "llama3.2", capHerramientas: true}
	a := Nuevo(p)
	a.Vista = VistaPrincipal
	a.Panel.SesionID = "s1"
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})

	verde, rojo := secuenciaDe(estiloCapaz), secuenciaDe(estiloIncapaz)
	if verde == "" || rojo == "" {
		t.Fatal("el perfil de prueba debe pintar color")
	}

	// Antes de saber la capacidad, la chapa queda atenuada.
	v := sinEstilo(a.View())
	if !strings.Contains(v, "* llama3.2") || !strings.Contains(v, "tool [?]") {
		t.Fatalf("la línea de estado muestra el modelo y la duda:\n%s", v)
	}

	// Llega la respuesta: con herramientas, `tool [*]` en verde.
	pulsa(t, a, capacidadesMsg{Nombre: "llama3.2", Herramientas: true})
	if !strings.Contains(sinEstilo(a.View()), "tool [*]") || !strings.Contains(a.View(), verde) {
		t.Errorf("con herramientas, la chapa va en verde:\n%s", sinEstilo(a.View()))
	}

	// Sin herramientas, en rojo.
	pulsa(t, a, capacidadesMsg{Nombre: "llama3.2", Herramientas: false})
	if !strings.Contains(sinEstilo(a.View()), "tool [*]") || !strings.Contains(a.View(), rojo) {
		t.Errorf("sin herramientas, la chapa va en rojo:\n%s", sinEstilo(a.View()))
	}

	// Una respuesta de otro modelo no pisa la del que está en uso.
	pulsa(t, a, capacidadesMsg{Nombre: "otro", Herramientas: true, Vision: true})
	if strings.Contains(a.View(), verde) && !strings.Contains(a.View(), rojo) {
		t.Error("la capacidad de otro modelo no cambia la del modelo en uso")
	}
}

// La línea de estado también refleja la capacidad de visión del modelo: `[v]`
// cuando la tiene, y `[T]` (texto) siempre.
func TestLaLineaDeEstadoMuestraVision(t *testing.T) {
	p := &puertoStub{modelo: "llava", capHerramientas: true, capVision: true}
	a := Nuevo(p)
	a.Vista = VistaPrincipal
	a.Panel.SesionID = "s1"
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})

	v := sinEstilo(a.View())
	if strings.Contains(v, "[v]") || !strings.Contains(v, "[T]") {
		t.Fatalf("sin dato no hay [v] y siempre está [T]:\n%s", v)
	}
	pulsa(t, a, capacidadesMsg{Nombre: "llava", Herramientas: true, Vision: true})
	if !strings.Contains(sinEstilo(a.View()), "[v]") {
		t.Errorf("con visión aparece [v]:\n%s", sinEstilo(a.View()))
	}
}

func TestElegirModeloActualizaLaLineaDeEstado(t *testing.T) {
	p := &puertoStub{
		modelo: "con-tools",
		modelos: []ModeloLocal{
			{Nombre: "con-tools"},
			{Nombre: "sin-tools", SinHerramientas: true},
		},
	}
	a := Nuevo(p)
	a.Vista = VistaPrincipal
	a.Panel.SesionID = "s1"
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})
	ejecuta(t, a, abreElModalDeModelos(t, a))
	tecla(t, a, tea.KeyDown)
	tecla(t, a, tea.KeyEnter)

	v := sinEstilo(a.View())
	if !strings.Contains(v, "* sin-tools") || !strings.Contains(v, "tool [*]") {
		t.Errorf("al elegir un modelo sin herramientas, la línea de estado lo dice:\n%s", v)
	}
	if !strings.Contains(a.View(), secuenciaDe(estiloIncapaz)) {
		t.Errorf("sin herramientas, la chapa va en rojo:\n%s", v)
	}
}

// Elegir un modelo cuya ficha no se pudo leer deja la capacidad como desconocida:
// `tool [?]` y sin avisos, en vez de afirmar que no puede.
func TestElegirUnModeloSinFichaNoAfirmaNada(t *testing.T) {
	p := &puertoStub{
		modelo:  "con-ficha",
		modelos: []ModeloLocal{{Nombre: "sin-ficha", CapacidadesSinDato: true}},
	}
	a := Nuevo(p)
	a.Vista = VistaPrincipal
	a.Panel.SesionID = "s1"
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})
	ejecuta(t, a, abreElModalDeModelos(t, a))
	tecla(t, a, tea.KeyEnter)

	v := sinEstilo(a.View())
	if !strings.Contains(v, "* sin-ficha") || !strings.Contains(v, "tool [?]") {
		t.Errorf("sin ficha la capacidad queda desconocida:\n%s", v)
	}
	if a.Aviso != "" {
		t.Errorf("una ficha ilegible no debe avisar de nada: %q", a.Aviso)
	}
}
