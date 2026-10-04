package tui

// El razonamiento de un modelo local cuesta minutos hasta para lo trivial: el
// interruptor del pie llega apagado, se pulsa con el ratón y se recuerda
// (SPEC-OLLAMA-PERFIL).

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// appConRazonamiento arma la vista principal con un modelo que razona —o que no—
// y sus capacidades ya resueltas, como las deja el mensaje del puerto.
func appConRazonamiento(t *testing.T, razona bool) (*App, *puertoStub) {
	t.Helper()
	// La preferencia se guarda en el config del usuario: se aísla en un HOME de
	// mentira para no tocar el de verdad.
	t.Setenv("HOME", t.TempDir())
	p := &puertoStub{modelo: "qwen3:4b", capPensar: razona}
	a := Nuevo(p)
	a.Vista = VistaPrincipal
	a.Panel.SesionID = "s1"
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 28})
	pulsa(t, a, capacidadesMsg{Nombre: "qwen3:4b", Herramientas: CapacidadSoportada, Pensar: CapacidadDe(razona)})
	return a, p
}

// celdaDeChapa localiza en el marco pintado la celda del interruptor.
func celdaDeChapa(t *testing.T, a *App) (int, int) {
	t.Helper()
	return celdaDeOpción(t, a, "pensar [", "pensar [")
}

// La chapa sale solo si el modelo declara que razona, y sale apagada: es el
// estado por defecto.
func TestLaChapaDeRazonamientoSoloSaleSiElModeloRazona(t *testing.T) {
	a, _ := appConRazonamiento(t, false)
	if v := sinEstilo(a.View()); strings.Contains(v, "pensar") {
		t.Errorf("un modelo que no razona no enseña el interruptor:\n%s", v)
	}

	a, _ = appConRazonamiento(t, true)
	if v := sinEstilo(a.View()); !strings.Contains(v, "pensar [ ]") {
		t.Errorf("la chapa sale apagada por defecto:\n%s", v)
	}
	if a.Pensar {
		t.Error("por defecto no se razona")
	}
}

// El clic en la chapa enciende y apaga el razonamiento, y el motor se entera.
func TestElClicEnLaChapaEnciendeYApagaElRazonamiento(t *testing.T) {
	a, p := appConRazonamiento(t, true)

	x, y := celdaDeChapa(t, a)
	clic(t, a, x, y)
	if !a.Pensar {
		t.Fatal("el clic enciende el razonamiento")
	}
	if p.pensarLlamadas != 1 || !p.pensarRecordado {
		t.Errorf("la vista avisa al motor: llamadas=%d recordado=%v", p.pensarLlamadas, p.pensarRecordado)
	}
	if v := sinEstilo(a.View()); !strings.Contains(v, "pensar [x]") {
		t.Errorf("el pie lo enseña encendido:\n%s", v)
	}

	// El mismo clic lo vuelve a apagar: no es de un solo sentido.
	x, y = celdaDeChapa(t, a)
	clic(t, a, x, y)
	if a.Pensar {
		t.Error("el segundo clic apaga el razonamiento")
	}
	if p.pensarLlamadas != 2 || p.pensarRecordado {
		t.Errorf("el motor recibe el valor apagado: llamadas=%d recordado=%v", p.pensarLlamadas, p.pensarRecordado)
	}
}

// Un clic fuera de la chapa no la toca: sigue siendo una selección de texto.
func TestElClicFueraDeLaChapaNoCambiaElRazonamiento(t *testing.T) {
	a, p := appConRazonamiento(t, true)
	x, y := celdaDeChapa(t, a)
	clic(t, a, x+len("pensar [x]")+3, y)
	if a.Pensar || p.pensarLlamadas != 0 {
		t.Errorf("fuera de la chapa no se cambia nada: %v %d", a.Pensar, p.pensarLlamadas)
	}
}

// El razonamiento en estado `desconocida` no ofrece el interruptor y avisa de
// que queda desactivado porque no se pudo comprobar; es el único de los tres
// estados desconocidos que avisa, porque le cambia el turno al usuario
// (SPEC-MODELO-MOTOR §Capacidades, T-F045-06).
func TestElThinkDesconocidoSeDesactivaYAvisa(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	// La preferencia está encendida, pero eso no basta: sin dato, la TUI no
	// ofrece el `think` y avisa.
	p := &puertoStub{modelo: "qwen3:4b", pensarRecordado: true}
	a := Nuevo(p)
	a.Vista = VistaPrincipal
	a.Panel.SesionID = "s1"
	pulsa(t, a, tea.WindowSizeMsg{Width: 120, Height: 30})

	pulsa(t, a, capacidadesMsg{Nombre: "qwen3:4b", Herramientas: CapacidadSoportada, Pensar: CapacidadDesconocida})
	v := sinEstilo(a.View())
	if strings.Contains(v, "pensar [") {
		t.Errorf("sin dato no se ofrece el interruptor de razonamiento:\n%s", v)
	}
	if !strings.Contains(v, "razonamiento está desactivado") {
		t.Errorf("el razonamiento desconocido avisa:\n%s", v)
	}

	// Con dato (razona): el aviso se retira y la chapa vuelve a ofrecerse.
	pulsa(t, a, capacidadesMsg{Nombre: "qwen3:4b", Herramientas: CapacidadSoportada, Pensar: CapacidadSoportada})
	v = sinEstilo(a.View())
	if strings.Contains(v, "razonamiento está desactivado") {
		t.Errorf("con dato no hay aviso:\n%s", v)
	}
	if !strings.Contains(v, "pensar [") {
		t.Errorf("con dato la chapa vuelve a ofrecerse:\n%s", v)
	}
}

// El interruptor arranca donde lo dejó el usuario.
func TestElRazonamientoSeRecuerdaEntreEjecuciones(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	p := &puertoStub{modelo: "qwen3:4b", capPensar: true, pensarRecordado: true}
	if a := Nuevo(p); !a.Pensar {
		t.Error("la vista arranca con el valor recordado")
	}
}
