package tui

// Tests de T-F003: la pantalla de bienvenida. Una prueba, una regla
// (frontend/05-quality/TESTING.md §3), contra las reglas de DOMAIN §3 y
// SPEC-INTERFAZ §Pantalla de bienvenida:
//
//	T-F003-01/03 → modelo, logotipo y composición de la vista
//	T-F003-02    → el arte se pinta byte a byte, tal cual el dorado
//	T-F003-04    → en la bienvenida solo se escribe, envía y sale
//	T-F003-05    → el envío resuelve la sesión y manda el mensaje, una vez

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"localcli/internal/session"
)

// --- T-F003-01: el modelo ---------------------------------------------------

func TestLaBienvenidaArrancaConElFocoEnSuUnicaEntrada(t *testing.T) {
	a := Nuevo(&puertoStub{})
	if !a.Bienvenida.Foco {
		t.Error("la entrada de la bienvenida nace enfocada: es la única línea de la pantalla")
	}
	v := sinEstilo(a.View())
	if !strings.Contains(v, "En qué te ayudo hoy: ▌") {
		t.Errorf("la línea de entrada con su cursor debe verse:\n%s", v)
	}
}

// --- T-F003-02: el logotipo dorado -------------------------------------------

func TestElLogotipoSePintaTalCualSinReescalarNiCentrar(t *testing.T) {
	a := Nuevo(&puertoStub{})
	// El arte va sin estilo, así que la comparación es directa contra el
	// dorado embebido: byte a byte y encabezando la vista, sin relleno de
	// centrado delante (SPEC-INTERFAZ: "se pinta tal cual").
	arte := strings.TrimRight(LogoCanonico, "\n")
	v := a.View()
	if !strings.HasPrefix(v, arte) {
		t.Error("la bienvenida empieza por el logotipo, sin reescalar ni centrar")
	}
}

// --- T-F003-03: la composición ----------------------------------------------

func TestLaComposicionPoneElArtePrimeroYElRestoFuera(t *testing.T) {
	a := Nuevo(&puertoStub{})
	v := sinEstilo(a.View())
	arte := strings.TrimRight(LogoCanonico, "\n")
	iArte := strings.Index(v, arte)
	iNombre := strings.Index(v, Nombre+" · "+Version)
	iEntrada := strings.Index(v, "En qué te ayudo hoy:")
	if iArte != 0 || iNombre < 0 || iEntrada < 0 {
		t.Fatalf("falta el arte, el nombre o la entrada:\n%s", v)
	}
	if iNombre <= iArte || iEntrada <= iNombre {
		t.Errorf("el orden es arte, nombre y entrada:\n%s", v)
	}
	// Nombre y entrada se componen alrededor pero fuera del arte.
	if iNombre < iArte+len(arte) {
		t.Error("el nombre no puede entrar dentro del arte canónico")
	}
}

// --- T-F003-04: el teclado de la bienvenida ---------------------------------

func TestEnLaBienvenidaLosAtajosDeLaPrincipalNoExisten(t *testing.T) {
	p := &puertoStub{sesiones: []session.Sesion{{ID: "s1", Nombre: "una"}}}
	a := Nuevo(p)
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})

	// T-F016: el modal de sesiones SÍ vive en la bienvenida; Esc lo cierra.
	ejecuta(t, a, abreElModalDeSesiones(t, a))
	if !a.Sesiones.Abierto {
		t.Error("en la bienvenida se abre el modal de sesiones")
	}
	tecla(t, a, tea.KeyEsc)
	if a.Sesiones.Abierto {
		t.Error("esc cierra el modal de sesiones en la bienvenida")
	}
	tecla(t, a, tea.KeyCtrlO)
	if a.Panel.Abierto {
		t.Error("en la bienvenida no hay panel de datos")
	}
	tecla(t, a, tea.KeyCtrlR)
	if !a.Razon.Visible {
		t.Error("en la bienvenida no hay razonamiento que ocultar")
	}
	if len(p.enviados) != 0 || p.activasResueltas != 0 {
		t.Errorf("esas teclas no pueden lanzar nada: %v", p.enviados)
	}

	// La única salida es Ctrl+C.
	cmd := tecla(t, a, tea.KeyCtrlC)
	if cmd == nil {
		t.Fatal("ctrl+c es la única salida desde la bienvenida")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Error("ctrl+c debe pedir salir del programa")
	}
}

func TestLaBienvenidaEscribeYBorra(t *testing.T) {
	a := Nuevo(&puertoStub{})
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})

	escribe(t, a, "qué")
	pulsa(t, a, tea.KeyMsg{Type: tea.KeySpace})
	escribe(t, a, "tal")
	if a.Bienvenida.Texto != "qué tal" {
		t.Fatalf("lo escrito se compone en la entrada: %q", a.Bienvenida.Texto)
	}
	if !strings.Contains(sinEstilo(a.View()), "qué tal▌") {
		t.Error("lo escrito debe verse con su cursor")
	}
	tecla(t, a, tea.KeyBackspace)
	if a.Bienvenida.Texto != "qué ta" {
		t.Errorf("borrar quita el último carácter: %q", a.Bienvenida.Texto)
	}
}

// --- T-F003-05: el envío de la primera petición ------------------------------

// T-F003-05 / SPEC-SESIONES: la primera petición desde la bienvenida CREA una
// sesión nueva (no hay ninguna previa) y luego envía el mensaje. Las dos
// operaciones, cada una exactamente una vez.
func TestEnviarDesdeLaBienvenidaCreaLaSesionYEnvíaUnaVez(t *testing.T) {
	p := &puertoStub{}
	a := Nuevo(p)
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})
	escribe(t, a, "documentar la capa")
	ejecuta(t, a, pulsa(t, a, tea.KeyMsg{Type: tea.KeyEnter}))

	if a.Vista != VistaPrincipal {
		t.Fatal("enviar la primera petición cambia a la interfaz principal")
	}
	// Las dos operaciones del envío (INTERFACES §2), cada una exactamente una
	// vez: crear la sesión (con su nombre provisional) y luego el mensaje.
	if p.creadas != 1 {
		t.Errorf("la primera petición desde la bienvenida crea una sesión: %d", p.creadas)
	}
	if p.activasResueltas != 0 {
		t.Errorf("no hay sesión previa que retomar: %d", p.activasResueltas)
	}
	if len(p.enviados) != 1 || p.enviados[0] != "nueva|documentar la capa" {
		t.Fatalf("el mensaje sale una sola vez, después de crear la sesión: %v", p.enviados)
	}
	if msgs := a.Chat.Mensajes(); len(msgs) != 1 || msgs[0].Texto != "documentar la capa" {
		t.Errorf("la petición es el primer mensaje del chat: %+v", msgs)
	}
	if a.Bienvenida.Texto != "" {
		t.Error("tras enviar, la entrada de la bienvenida queda limpia")
	}

	// La transición no repite la petición: lo que llegue después no reenvía.
	pulsa(t, a, tea.KeyMsg{Type: tea.KeyEsc})
	pulsa(t, a, eventoMsg{Evento: Evento{Nombre: EventoToken, Datos: map[string]string{"texto": "x"}}})
	if len(p.enviados) != 1 {
		t.Errorf("la petición no debe repetirse: %v", p.enviados)
	}
}

func TestEnviarVacioDesdeLaBienvenidaNoHaceNada(t *testing.T) {
	p := &puertoStub{}
	a := Nuevo(p)
	ejecuta(t, a, pulsa(t, a, tea.KeyMsg{Type: tea.KeyEnter}))
	if a.Vista != VistaBienvenida {
		t.Error("sin petición no hay transición de vista")
	}
	if p.creadas != 0 || p.activasResueltas != 0 || len(p.enviados) != 0 {
		t.Errorf("vacío no crea sesión ni envía: %d, %d, %v", p.creadas, p.activasResueltas, p.enviados)
	}
}
