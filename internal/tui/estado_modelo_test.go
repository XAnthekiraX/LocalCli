package tui

// Tests de la línea de estado del modelo bajo el input (app.go).

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestLaLineaDeEstadoMuestraModeloYHerramientas(t *testing.T) {
	p := &puertoStub{modelo: "llama3.2", capHerramientas: true}
	a := Nuevo(p)
	a.Vista = VistaPrincipal
	a.Panel.SesionID = "s1"
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})

	// Antes de saber la capacidad, se muestra «?».
	v := sinEstilo(a.View())
	if !strings.Contains(v, "modelo: llama3.2") || !strings.Contains(v, "herramientas: ?") {
		t.Fatalf("la línea de estado muestra el modelo y la duda:\n%s", v)
	}

	// Llega la respuesta: con herramientas.
	pulsa(t, a, capacidadesMsg{Nombre: "llama3.2", Herramientas: true})
	if !strings.Contains(sinEstilo(a.View()), "herramientas: sí") {
		t.Errorf("con herramientas:\n%s", sinEstilo(a.View()))
	}

	// Sin herramientas.
	pulsa(t, a, capacidadesMsg{Nombre: "llama3.2", Herramientas: false})
	if !strings.Contains(sinEstilo(a.View()), "herramientas: no") {
		t.Errorf("sin herramientas:\n%s", sinEstilo(a.View()))
	}

	// Una respuesta de otro modelo no pisa la del que está en uso.
	pulsa(t, a, capacidadesMsg{Nombre: "otro", Herramientas: true})
	if !strings.Contains(sinEstilo(a.View()), "herramientas: no") {
		t.Error("la capacidad de otro modelo no cambia la del modelo en uso")
	}
}

// La línea de estado también refleja la capacidad de visión del modelo.
func TestLaLineaDeEstadoMuestraVision(t *testing.T) {
	p := &puertoStub{modelo: "llava", capHerramientas: true, capVision: true}
	a := Nuevo(p)
	a.Vista = VistaPrincipal
	a.Panel.SesionID = "s1"
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})

	// Antes de saberlo, «?».
	if !strings.Contains(sinEstilo(a.View()), "visión: ?") {
		t.Fatalf("sin dato, la visión se muestra como duda:\n%s", sinEstilo(a.View()))
	}
	pulsa(t, a, capacidadesMsg{Nombre: "llava", Herramientas: true, Vision: true})
	if !strings.Contains(sinEstilo(a.View()), "visión: sí") {
		t.Errorf("con visión:\n%s", sinEstilo(a.View()))
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
	if !strings.Contains(v, "modelo: sin-tools") || !strings.Contains(v, "herramientas: no") {
		t.Errorf("al elegir un modelo sin herramientas, la línea de estado lo dice:\n%s", v)
	}
}
