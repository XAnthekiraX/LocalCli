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
	if v := e.Caja(e.Ancho, ""); !strings.Contains(sinEstilo(v), "Escribe tu petición…") {
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
	a.Panel.Abierto = false
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})
	if a.Entrada.Ancho != 100 {
		t.Errorf("con el panel cerrado la línea toma todo el ancho: %d", a.Entrada.Ancho)
	}
	a.Panel.Abierto = true
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})
	if a.Entrada.Ancho != 100-AnchoPanel-1 {
		t.Errorf("con el panel abierto la línea cede su ancho: %d", a.Entrada.Ancho)
	}
	// Un ancho desconocido no resetea la medida anterior.
	a.Entrada.FijarAncho(0)
	if a.Entrada.Ancho != 100-AnchoPanel-1 {
		t.Errorf("ancho 0 no debe tocar la medida: %d", a.Entrada.Ancho)
	}
}

// --- T-F035: el salto de línea de la entrada --------------------------------

func TestLaEntradaEnvuelveElTextoLargoEnVezDeRecortarlo(t *testing.T) {
	a := Nuevo(&puertoStub{})
	a.Vista = VistaPrincipal
	a.Panel.SesionID = "s1"
	a.Panel.Abierto = false
	pulsa(t, a, tea.WindowSizeMsg{Width: 40, Height: 20})
	escribe(t, a, "este es un texto de prueba bastante largo que debe saltar de linea")

	v := sinEstilo(a.Entrada.Caja(a.Entrada.Ancho, ""))
	// El texto está entero y repartido: saltó de línea en vez de recortarse al
	// inicio o desbordar la terminal.
	if filas := len(strings.Split(v, "\n")); filas < 2 {
		t.Fatalf("un texto largo debe saltar de línea:\n%s", v)
	}
	// Se compara sin los caracteres de la caja, sin espacios ni saltos: las
	// filas de continuación se reparten, pero el texto no se pierde.
	sinCaja := strings.NewReplacer("│", "", "╭", "", "╮", "", "╰", "", "╯", "", "─", "").Replace(v)
	compacto := strings.Join(strings.Fields(sinCaja), "")
	quiero := strings.Join(strings.Fields("este es un texto de prueba bastante largo que debe saltar de linea"), "")
	if !strings.Contains(compacto, quiero) {
		t.Errorf("el texto se conserva entero, repartido en filas:\n%s", v)
	}
	// Ninguna fila pasa del ancho de la terminal: la línea ya no desborda.
	for _, l := range strings.Split(v, "\n") {
		if n := len([]rune(l)); n > 40 {
			t.Errorf("ninguna fila puede pasar del ancho (40), hay una de %d: %q", n, l)
		}
	}
}

func TestElSaltoDeLineaSeReajustaAlRedimensionar(t *testing.T) {
	a := Nuevo(&puertoStub{})
	a.Vista = VistaPrincipal
	a.Panel.Abierto = false
	pulsa(t, a, tea.WindowSizeMsg{Width: 30, Height: 20})
	escribe(t, a, "texto suficientemente largo para envolver en varias lineas")
	if filas := a.Entrada.campo.Height(); filas < 2 {
		t.Fatalf("a ancho 30 el texto ocupa varias filas: %d", filas)
	}
	// Al ensanchar la terminal, el texto vuelve a caber en una sola fila.
	pulsa(t, a, tea.WindowSizeMsg{Width: 200, Height: 20})
	if filas := a.Entrada.campo.Height(); filas != 1 {
		t.Errorf("al ensanchar, el texto vuelve a una fila: %d", filas)
	}
}

func TestElAltoDeLaEntradaNoPasaDelTope(t *testing.T) {
	a := Nuevo(&puertoStub{})
	a.Vista = VistaPrincipal
	pulsa(t, a, tea.WindowSizeMsg{Width: 20, Height: 20})
	escribe(t, a, strings.Repeat("palabra ", 100))
	if filas := a.Entrada.campo.Height(); filas > altoMáximoEntrada {
		t.Errorf("la entrada no pasa del tope de %d filas: %d", altoMáximoEntrada, filas)
	}
}

func TestFilasEnvueltasCuentaElSoftWrap(t *testing.T) {
	// Sin texto siempre hay al menos una fila.
	if got := filasEnvueltas("", 10); got != 1 {
		t.Errorf("un texto vacío ocupa una fila: %d", got)
	}
	// El ancho manda: cuanto más estrecho, más filas.
	texto := "un texto de varias palabras para envolver"
	if ancho := filasEnvueltas(texto, 100); ancho != 1 {
		t.Errorf("con ancho de sobra el texto cabe en una fila: %d", ancho)
	}
	if estrecho, ancho := filasEnvueltas(texto, 15), filasEnvueltas(texto, 100); estrecho <= ancho {
		t.Errorf("estrechar el ancho reparte el texto en más filas: %d vs %d", estrecho, ancho)
	}
}
