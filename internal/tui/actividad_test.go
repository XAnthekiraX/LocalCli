package tui

// actividad_test.go — el indicador en vivo ([⠋ Pensando], [⠋ Usando
// herramienta: X]) y el contador de tokens del turno. Una prueba, una regla
// (frontend/05-quality/TESTING.md §3):
//
//	el conteo se acorta en la unidad legible
//	el glifo del indicador recorre el ciclo
//	el indicador dice Pensando/Generando y nombra la herramienta
//	la línea de tokens muestra el consumo del turno
//	el valor exacto del turno corrige la aproximación viva

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// --- el formato del conteo (función pura) ------------------------------------

func TestFormatearTokensAcortaElConteoEnLaUnidadLegible(t *testing.T) {
	casos := []struct {
		n      int
		quiero string
	}{
		{0, "0"},
		{999, "999"},
		{1000, "1.0k"},
		{4200, "4.2k"},
		{9999, "10.0k"},
		{10000, "10k"},
		{54000, "54k"},
		{128000, "128k"},
		{-5, "0"}, // nunca un conteo negativo
	}
	for _, c := range casos {
		if got := formatearTokens(c.n); got != c.quiero {
			t.Errorf("formatearTokens(%d) = %q, quiero %q", c.n, got, c.quiero)
		}
	}
}

// --- el indicador (funciones puras) ------------------------------------------

func TestGlifoActividadRecorreElCiclo(t *testing.T) {
	for i := 0; i < len(framesActividad)*3; i++ {
		got := glifoActividad(i)
		if got != framesActividad[i%len(framesActividad)] {
			t.Fatalf("glifoActividad(%d) = %q, quiero el frame %d", i, got, i%len(framesActividad))
		}
	}
	// Un frame negativo se envuelve en lugar de romper.
	if got := glifoActividad(-1); got != framesActividad[len(framesActividad)-1] {
		t.Errorf("glifoActividad(-1) = %q, quiero el último frame", got)
	}
}

func TestRenderActividadComponeElIndicador(t *testing.T) {
	got := renderActividad(0, "Pensando")
	if !strings.Contains(got, "Pensando") {
		t.Errorf("el indicador lleva su etiqueta: %q", got)
	}
	if !strings.HasPrefix(got, "[") || !strings.HasSuffix(got, "]") {
		t.Errorf("el indicador va entre corchetes: %q", got)
	}
	if strings.Contains(got, "Generando") {
		t.Errorf("no debe inventar etiqueta: %q", got)
	}
}

// --- el indicador en la vista -------------------------------------------------

func TestElIndicadorDicePensandoYGenerando(t *testing.T) {
	h := nuevoArnes(t)
	h.escribe("hola")
	ejecuta(h.t, h.app, pulsa(h.t, h.app, tea.KeyMsg{Type: tea.KeyEnter}))
	// Enviado el turno y sin respuesta todavía: el modelo piensa.
	h.veSiContiene("Pensando")
	// Llega la respuesta: el indicador pasa a Generando.
	h.evento(EventoToken, map[string]string{"texto": "listo"})
	if strings.Contains(h.ve(), "Pensando") {
		t.Errorf("con la respuesta en marcha el indicador ya no dice Pensando:\n%s", h.ve())
	}
	h.veSiContiene("Generando")
}

// --- el contador de tokens ----------------------------------------------------

func TestLaLineaDeTokensMuestraElConsumoDelTurno(t *testing.T) {
	h := nuevoArnes(t)
	if strings.Contains(h.ve(), "tokens:") {
		t.Error("sin consumo no se pinta la línea de tokens")
	}
	h.app.Panel.Tokens = 54000
	h.veSiContiene("tokens: 54k")
}

func TestElConsumoVivoSeCorrigeConElValorExactoDelTurno(t *testing.T) {
	h := nuevoArnes(t)
	h.evento(EventoToken, map[string]string{"texto": "uno"})
	h.evento(EventoToken, map[string]string{"texto": "dos"})
	if h.app.Panel.Tokens != 2 {
		t.Fatalf("los fragmentos cuentan para el consumo vivo: %d", h.app.Panel.Tokens)
	}
	h.evento(EventoTokensTurno, map[string]string{"entrada": "120", "salida": "54000"})
	if h.app.Panel.Tokens != 54000 {
		t.Fatalf("el valor exacto corrige la aproximación: %d", h.app.Panel.Tokens)
	}
	h.veSiContiene("tokens: 54k")
}

func TestUnTurnoNuevoReiniciaElConsumo(t *testing.T) {
	h := nuevoArnes(t)
	h.evento(EventoTokensTurno, map[string]string{"entrada": "1", "salida": "4500"})
	if h.app.Panel.Tokens != 4500 {
		t.Fatalf("el consumo del turno se guarda: %d", h.app.Panel.Tokens)
	}
	h.escribe("otra")
	ejecuta(h.t, h.app, pulsa(h.t, h.app, tea.KeyMsg{Type: tea.KeyEnter}))
	if h.app.Panel.Tokens != 0 {
		t.Fatalf("un turno nuevo reinicia el consumo: %d", h.app.Panel.Tokens)
	}
}
