package tui

// Tests del contador de duración de cada respuesta. Una prueba, una regla
// (frontend/05-quality/TESTING.md §3):
//
//	el tiempo se formatea en la unidad legible
//	la respuesta cerrada lleva su tiempo y se pinta
//	mientras el turno está vivo corre el contador en vivo
//	al cerrarse el turno el contador se detiene

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"localcli/internal/session"
)

// --- la función pura de formato ----------------------------------------------

func TestFormatearDuracionEligeLaUnidadLegible(t *testing.T) {
	casos := []struct {
		d      time.Duration
		quiero string
	}{
		{0, "0 ms"},
		{840 * time.Millisecond, "840 ms"},
		{999 * time.Millisecond, "999 ms"},
		{time.Second, "1.0 s"},
		{4200 * time.Millisecond, "4.2 s"},
		{59 * time.Second, "59.0 s"},
		{time.Minute, "1 m 0 s"},
		{65 * time.Second, "1 m 5 s"},
		{-3 * time.Second, "0 ms"}, // nunca un tiempo negativo
	}
	for _, c := range casos {
		if got := formatearDuracion(c.d); got != c.quiero {
			t.Errorf("formatearDuracion(%v) = %q, quiero %q", c.d, got, c.quiero)
		}
	}
}

// Sin duración medida no hay sufijo: un historial recargado no trae el tiempo.
func TestSinDuracionNoSePintaSufijo(t *testing.T) {
	if got := sufijoDuracion(0); got != "" {
		t.Errorf("sin duración no se pinta nada: %q", got)
	}
}

// --- la línea de herramienta --------------------------------------------------

// La línea cerrada lleva su tiempo al final, con el separador de la línea y sin
// paréntesis: los paréntesis marcan el tiempo de una respuesta, y una línea de
// herramienta no lo es (INTERFACES.md §1.1).
func TestLaLineaDeHerramientaCerradaLlevaSuTiempo(t *testing.T) {
	linea := sinEstilo(LineaHerramientaCerrada("LEER [AGENTS.md]", true, false, "93 líneas", "", 400*time.Millisecond))
	if linea != "✓ LEER [AGENTS.md] · 93 líneas · 400 ms" {
		t.Fatalf("la línea cerrada lleva su duración al final: %q", linea)
	}
}

// El tiempo se pinta también cuando la ejecución falló: tardó igual.
func TestLaLineaDeHerramientaQueFalloLlevaSuTiempo(t *testing.T) {
	linea := sinEstilo(LineaHerramientaCerrada("CREAR [nuevo.txt]", false, false, "", "el archivo ya existe", 10*time.Millisecond))
	if linea != "✗ CREAR [nuevo.txt] · el archivo ya existe · 10 ms" {
		t.Fatalf("la línea fallida lleva su tiempo tras el motivo: %q", linea)
	}
}

// Sin duración medida no se pinta nada ni se deja hueco: la línea queda igual
// que antes de existir el tiempo.
func TestSinDuracionLaLineaDeHerramientaNoDejaHueco(t *testing.T) {
	linea := sinEstilo(LineaHerramientaCerrada("LEER [a]", true, false, "1 línea", "", -1))
	if linea != "✓ LEER [a] · 1 línea" {
		t.Fatalf("sin medición la línea no cambia: %q", linea)
	}
}

// El chat pinta el tiempo de la línea al cerrarla, igual que lo hace con la
// respuesta: los dos salen del mismo formateador.
func TestElChatPintaElTiempoDeLaLineaDeHerramienta(t *testing.T) {
	c := Chat{}
	c.AnotarInvocacion("LEER", "a.md")
	c.CerrarHerramienta("leer_archivo", true, false, "70 líneas", "", 400*time.Millisecond)
	if plano := sinEstilo(c.Render(80)); !strings.Contains(plano, "✓ LEER [a.md] · 70 líneas · 400 ms") {
		t.Fatalf("el tiempo se pinta al cerrar la línea:\n%s", plano)
	}
}

// --- el chat -----------------------------------------------------------------

func TestLaRespuestaCerradaLlevaSuTiempo(t *testing.T) {
	c := Chat{}
	c.AñadirUsuario("hola")
	if !c.HayTurno() {
		t.Fatal("enviado el mensaje, el turno está en curso")
	}
	c.Token("buenas")
	c.CerrarTurno("")

	if c.HayTurno() {
		t.Error("cerrado el turno, el contador se detiene")
	}
	msgs := c.Mensajes()
	if len(msgs) != 2 || msgs[1].Rol != RolAgente {
		t.Fatalf("esperaba usuario + agente: %+v", msgs)
	}
	if msgs[1].Duracion <= 0 {
		t.Fatalf("la respuesta cerrada debe llevar su duración: %+v", msgs[1])
	}
	if plano := sinEstilo(c.Render(80)); !strings.Contains(plano, formatearDuracion(msgs[1].Duracion)) {
		t.Fatalf("el tiempo se pinta junto a la respuesta:\n%s", plano)
	}
}

// --- la vista ------------------------------------------------------------------

func TestElContadorEnVivoCorreMientrasElTurnoEstáVivo(t *testing.T) {
	a, _ := appConTurnoEnviado(t)
	if !strings.Contains(sinEstilo(a.View()), "Pensando") {
		t.Fatalf("mientras se espera la respuesta se ve el indicador en vivo:\n%s", sinEstilo(a.View()))
	}
}

func TestElContadorSeDetieneAlTerminarElTurno(t *testing.T) {
	a, _ := appConTurnoEnviado(t)
	// El modelo entrega algo y luego el turno se cierra.
	pulsa(t, a, eventoMsg{Evento: Evento{Nombre: EventoToken, Datos: map[string]string{"texto": "buenas"}}})
	pulsa(t, a, eventoMsg{Evento: Evento{
		Nombre: session.EventoEstadoSesion,
		Datos:  map[string]string{"sesion": "s1", "estado": session.EstadoTerminada},
	}})

	if a.Chat.HayTurno() {
		t.Error("terminado el turno, el contador se detiene")
	}
	if strings.Contains(sinEstilo(a.View()), "Pensando") {
		t.Errorf("el indicador en vivo desaparece al cerrarse el turno:\n%s", sinEstilo(a.View()))
	}
	msgs := a.Chat.Mensajes()
	ultimo := msgs[len(msgs)-1]
	if ultimo.Rol != RolAgente || ultimo.Duracion <= 0 {
		t.Fatalf("la respuesta cerrada lleva su tiempo: %+v", ultimo)
	}
}

func TestUnLatidoDeOtraGeneraciónNoSeRearma(t *testing.T) {
	a, _ := appConTurnoEnviado(t)
	if a.latido == 0 {
		t.Fatal("enviar arranca una generación de latidos")
	}
	// Un latido de una cadena anterior se detiene: no compite con la vigente.
	if cmd := pulsa(t, a, tickMsg{gen: a.latido - 1}); cmd != nil {
		t.Error("un latido de otra generación no debe re-armarse")
	}
	// El de la generación vigente, con el turno vivo, sigue latiendo.
	if cmd := pulsa(t, a, tickMsg{gen: a.latido}); cmd == nil {
		t.Error("el latido vigente con turno vivo debe re-armarse")
	}
}

// appConTurnoEnviado deja la vista principal con un mensaje ya enviado, que es
// lo que arranca el contador.
func appConTurnoEnviado(t *testing.T) (*App, *puertoStub) {
	t.Helper()
	p := &puertoStub{}
	a := Nuevo(p)
	a.Vista = VistaPrincipal
	a.Panel.SesionID = "s1"
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})
	escribe(t, a, "hola")
	ejecuta(t, a, pulsa(t, a, tea.KeyMsg{Type: tea.KeyEnter}))
	return a, p
}
