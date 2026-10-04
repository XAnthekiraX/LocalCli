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
	pulsa(t, a, capacidadesMsg{Nombre: "llama3.2", Herramientas: CapacidadSoportada})
	if !strings.Contains(sinEstilo(a.View()), "tool [*]") || !strings.Contains(a.View(), verde) {
		t.Errorf("con herramientas, la chapa va en verde:\n%s", sinEstilo(a.View()))
	}

	// Sin herramientas, en rojo.
	pulsa(t, a, capacidadesMsg{Nombre: "llama3.2", Herramientas: CapacidadNoSoportada})
	if !strings.Contains(sinEstilo(a.View()), "tool [*]") || !strings.Contains(a.View(), rojo) {
		t.Errorf("sin herramientas, la chapa va en rojo:\n%s", sinEstilo(a.View()))
	}

	// Una respuesta de otro modelo no pisa la del que está en uso.
	pulsa(t, a, capacidadesMsg{Nombre: "otro", Herramientas: CapacidadSoportada, Vision: CapacidadSoportada})
	if strings.Contains(a.View(), verde) && !strings.Contains(a.View(), rojo) {
		t.Error("la capacidad de otro modelo no cambia la del modelo en uso")
	}
}

// La capacidad en estado `desconocida` se pinta con `?` —herramientas y
// visión— y la `no soportada` sin `?`, porque ahí sí hay dato; el `?` va junto
// al nombre de la capacidad (SPEC-MODELO-MOTOR §Capacidades, T-F045-05).
func TestCapacidadDesconocidaSePintaConInterrogacionYNoSoportadaSin(t *testing.T) {
	p := &puertoStub{modelo: "llama3.2"}
	a := Nuevo(p)
	a.Vista = VistaPrincipal
	a.Panel.SesionID = "s1"
	pulsa(t, a, tea.WindowSizeMsg{Width: 120, Height: 30})

	// Sin dato: herramientas y visión llevan `?` junto a su nombre.
	v := sinEstilo(a.View())
	if !strings.Contains(v, "tool [?]") {
		t.Fatalf("herramientas desconocidas se marcan con ?:\n%s", v)
	}
	if !strings.Contains(v, "[v?]") {
		t.Errorf("visión desconocida se marca con ?:\n%s", v)
	}

	// Con dato negativo: ninguna `?`; las chapas dicen lo que se sabe.
	pulsa(t, a, capacidadesMsg{Nombre: "llama3.2", Herramientas: CapacidadNoSoportada, Vision: CapacidadNoSoportada})
	v = sinEstilo(a.View())
	if strings.Contains(v, "tool [?]") || strings.Contains(v, "[v?]") {
		t.Errorf("una capacidad no soportada no lleva ?:\n%s", v)
	}
	if !strings.Contains(v, "tool [*]") {
		t.Errorf("herramientas no soportadas siguen mostrando su chapa:\n%s", v)
	}
	if strings.Contains(v, "[v]") {
		t.Errorf("visión no soportada no se ofrece:\n%s", v)
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
	pulsa(t, a, capacidadesMsg{Nombre: "llava", Herramientas: CapacidadSoportada, Vision: CapacidadSoportada})
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

// TestLaLineaDeEstadoAvisaDelMotorAusente — T-F044-09: si el motor de la sesión
// está desactivado o ya no existe, la línea de estado lo dice —con el camino
// para arreglarlo, `Ctrl+X i`— sin bloquear nada: la sesión sigue escribiéndose
// y esperando (SPEC-MODELO-MOTOR §Desactivar, §Eliminar).
func TestLaLineaDeEstadoAvisaDelMotorAusente(t *testing.T) {
	for _, caso := range []struct {
		nombre  string
		motores []MotorLocal
		aviso   string
	}{
		{"desactivado", []MotorLocal{{ID: "m1", Nombre: "Ollama local", Activo: false}}, "está desactivado"},
		{"eliminado", []MotorLocal{{ID: "otro", Nombre: "otro motor", Activo: true}}, "ya no existe"},
	} {
		p := &puertoStub{
			modelo:  "qwen3:8b",
			motores: caso.motores,
			pares:   map[string][2]string{"s1": {"m1", "qwen3:8b"}},
		}
		a := Nuevo(p)
		a.Vista = VistaPrincipal
		a.Panel.SesionID = "s1"
		// La ventana es generosa a propósito: el aviso va junto al modelo en la
		// misma línea, que se recorta al ancho de la caja.
		pulsa(t, a, tea.WindowSizeMsg{Width: 200, Height: 40})
		ejecuta(t, a, a.cmdMotorDeSesion("s1"))

		v := sinEstilo(a.View())
		if !strings.Contains(v, caso.aviso) {
			t.Errorf("%s: la línea de estado avisa de que %s:\n%s", caso.nombre, caso.aviso, v)
		}
		if !strings.Contains(v, "Ctrl+X i") {
			t.Errorf("%s: el aviso dice cómo elegir otro motor:\n%s", caso.nombre, v)
		}
		// No bloquea: la sesión sigue escribiéndose y esperando.
		escribe(t, a, "hola")
		if a.Entrada.Texto() != "hola" {
			t.Errorf("%s: el aviso no bloquea la entrada: %q", caso.nombre, a.Entrada.Texto())
		}
	}
}

// TestLaLineaDeEstadoRotulaElMotorDeLaSesion — T-F044-09: junto al modelo va
// el nombre de la INSTANCIA del motor de la sesión, que es lo que la distingue
// de otra del mismo tipo (SPEC-INTERFAZ §Caja de entrada).
func TestLaLineaDeEstadoRotulaElMotorDeLaSesion(t *testing.T) {
	p := &puertoStub{
		modelo:  "qwen3:8b",
		motores: []MotorLocal{{ID: "m1", Nombre: "Ollama local", Activo: true}},
		pares:   map[string][2]string{"s1": {"m1", "qwen3:8b"}},
	}
	a := Nuevo(p)
	a.Vista = VistaPrincipal
	a.Panel.SesionID = "s1"
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})
	ejecuta(t, a, a.cmdMotorDeSesion("s1"))

	v := sinEstilo(a.View())
	if !strings.Contains(v, "qwen3:8b (Ollama local)") {
		t.Errorf("la línea de estado rotula el modelo con su motor:\n%s", v)
	}
	if strings.Contains(v, "desactivado") || strings.Contains(v, "ya no existe") {
		t.Errorf("un motor presente y activo no avisa de nada:\n%s", v)
	}
}
