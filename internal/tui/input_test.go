package tui

// Tests de T-F004: la línea de entrada de la interfaz principal. Una prueba,
// una regla (frontend/05-quality/TESTING.md §3):
//
//	T-F004-01 → modelo de una sola línea con placeholder
//	T-F004-02 → escribir mientras la sesión genera no bloquea ni cancela
//	T-F004-03 → Enter compone el envío solo con texto; vacío no envía
//	T-F004-04 → el ancho se adapta al layout recibido

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"localcli/internal/session"
)

// --- T-F004-01: el modelo ---------------------------------------------------

func TestLaEntradaNaceEnfocadaConSuPlaceholder(t *testing.T) {
	e := NuevaEntrada()
	if e.Texto() != "" {
		t.Errorf("la línea nace vacía: %q", e.Texto())
	}
	if v := e.View(); !strings.Contains(sinEstilo(v), "Escribe tu petición…") {
		t.Errorf("vacía muestra su placeholder: %q", v)
	}
}

// --- T-F004-02: escribir mientras la sesión genera ---------------------------

func TestEscribirMientrasGeneraNoBloqueaNiCancela(t *testing.T) {
	p := &puertoStub{}
	a := Nuevo(p)
	a.Vista = VistaPrincipal
	a.Panel.SesionID = "s1"
	a.Panel.Estado = session.EstadoTrabajando
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})

	escribe(t, a, "otra cosa")
	if !strings.Contains(sinEstilo(a.View()), "otra cosa") {
		t.Fatal("la entrada sigue operativa mientras la sesión genera (INTERFACES §5)")
	}
	// La generación sigue su curso mientras se escribe.
	pulsa(t, a, eventoMsg{Evento: Evento{Nombre: EventoToken, Datos: map[string]string{"texto": "respuesta"}}})
	if a.Chat.EnCurso() == "" {
		t.Error("escribir no detiene la generación")
	}
	if len(p.cancelado) != 0 || len(p.pausadas) != 0 {
		t.Errorf("escribir no cancela ni pausa nada: %v, %v", p.cancelado, p.pausadas)
	}
	// Confirmar durante la generación también funciona.
	ejecuta(t, a, pulsa(t, a, tea.KeyMsg{Type: tea.KeyEnter}))
	if len(p.enviados) != 1 {
		t.Errorf("enviar mientras genera no se bloquea: %v", p.enviados)
	}
}

// --- T-F004-03: confirmar el envío -------------------------------------------

func TestEnterConTextoEnviaYVacioNoEnvia(t *testing.T) {
	p := &puertoStub{}
	a := Nuevo(p)
	a.Vista = VistaPrincipal
	a.Panel.SesionID = "s1"

	// Vacía: no hay petición que componer.
	ejecuta(t, a, pulsa(t, a, tea.KeyMsg{Type: tea.KeyEnter}))
	if len(p.enviados) != 0 {
		t.Fatalf("con la línea vacía no se envía nada: %v", p.enviados)
	}
	// Con texto: una petición y la línea limpia para lo siguiente.
	escribe(t, a, "arregla el test")
	ejecuta(t, a, pulsa(t, a, tea.KeyMsg{Type: tea.KeyEnter}))
	if len(p.enviados) != 1 || p.enviados[0] != "s1|arregla el test" {
		t.Fatalf("Enter compone y envía la petición: %v", p.enviados)
	}
	if a.Entrada.Texto() != "" {
		t.Error("tras enviar, la línea queda limpia")
	}
	// Solo espacios: tampoco hay petición.
	escribe(t, a, "   ")
	ejecuta(t, a, pulsa(t, a, tea.KeyMsg{Type: tea.KeyEnter}))
	if len(p.enviados) != 1 {
		t.Errorf("una línea de espacios no es petición: %v", p.enviados)
	}
}

// --- T-F004-05: la edición del cursor ----------------------------------------

func TestElCursorSeMueveConLasFlechasYHomeEnd(t *testing.T) {
	e := NuevaEntrada()
	escribir := func(e *Entrada, k tea.KeyType) {
		e2, _ := e.Update(tea.KeyMsg{Type: k})
		*e = e2
	}
	for _, r := range "hole como estas" {
		e2, _ := e.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		e = e2
	}
	if e.Texto() != "hole como estas" {
		t.Fatalf("texto escrito: %q", e.Texto())
	}

	// La flecha izquierda mueve el caret sin tocar el texto; escribir en medio
	// inserta donde está el cursor (el caso «hole| como estas»).
	escribir(&e, tea.KeyLeft)
	if e.Texto() != "hole como estas" {
		t.Errorf("mover el cursor no cambia el texto: %q", e.Texto())
	}
	e2, _ := e.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("X")})
	e = e2
	if e.Texto() != "hole como estaXs" {
		t.Errorf("se escribe donde está el cursor: %q", e.Texto())
	}

	// Home lleva al principio y End al final.
	escribir(&e, tea.KeyHome)
	e2, _ = e.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(">")})
	e = e2
	if e.Texto() != ">hole como estaXs" {
		t.Errorf("home lleva el cursor al principio: %q", e.Texto())
	}
	escribir(&e, tea.KeyEnd)
	e2, _ = e.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("<")})
	e = e2
	if e.Texto() != ">hole como estaXs<" {
		t.Errorf("end lleva el cursor al final: %q", e.Texto())
	}

	// Ctrl+B / Ctrl+E son los equivalentes de emacs y también editan.
	escribir(&e, tea.KeyCtrlB)
	e2, _ = e.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("!")})
	e = e2
	if e.Texto() != ">hole como estaXs!<" {
		t.Errorf("ctrl+b mueve el cursor: %q", e.Texto())
	}
}

// --- T-F004-04: el ancho ------------------------------------------------------

func TestElAnchoDeLaEntradaSeAdaptaAlLayout(t *testing.T) {
	a := Nuevo(&puertoStub{})
	a.Vista = VistaPrincipal
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})
	if a.Entrada.Ancho != 100 {
		t.Errorf("con el panel cerrado la línea toma todo el ancho: %d", a.Entrada.Ancho)
	}
	a.Panel.Abierto = true
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})
	if a.Entrada.Ancho != 100-AnchoPanel {
		t.Errorf("con el panel abierto la línea cede su ancho: %d", a.Entrada.Ancho)
	}
	// Un ancho desconocido no resetea la medida anterior.
	a.Entrada.FijarAncho(0)
	if a.Entrada.Ancho != 100-AnchoPanel {
		t.Errorf("ancho 0 no debe tocar la medida: %d", a.Entrada.Ancho)
	}
}
