package tui

// Tests del estado vivo al cambiar o crear sesión y del aislamiento entre
// sesiones (SPEC-SESIONES, SPEC-PANEL-CONTEXTO, DOMAIN §1: "No hay relación
// entre sesiones"). Cubren el arreglo del bug reportado:
//
//   - el CONTEXTO del sidebar se refresca con la sesión activa;
//   - crear sesión no hereda el contexto de la anterior;
//   - el razonamiento revelado (Ctrl+R) se conserva al cambiar;
//   - retomar una sesión que trabaja reanuda el reloj y el glifo;
//   - los eventos de otra sesión no se cuelan en la visible.

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"localcli/internal/session"
)

// ejecutaLote corre un comando y, si es un lote (`tea.Batch`), corre también
// cada comando que lo compone entregando su mensaje. El bucle real de Bubble Tea
// despacha un `BatchMsg` solo; `Update` no lo entiende, así que en el arnés hay
// que aplanarlo a mano.
func ejecutaLote(t *testing.T, a *App, cmd tea.Cmd) {
	t.Helper()
	if cmd == nil {
		return
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, c := range batch {
			ejecuta(t, a, c)
		}
		return
	}
	if msg != nil {
		pulsa(t, a, msg)
	}
}

// Cambiar de sesión refresca el CONTEXTO del sidebar con los números de la
// sesión cargada, en lugar de conservar los de la anterior.
func TestCambiarDeSesionRefrescaElContexto(t *testing.T) {
	a, p := appConModales(t)
	// La sesión activa (s1) tiene un contexto cargado...
	a.Panel.ContextoTokens = 123
	a.Panel.LimiteTokens = 1000
	a.Panel.TokensEstimados = true
	// ...y el puerto devuelve otros números para la sesión a la que se cambia.
	p.contextoTokens = 7
	p.limiteTokens = 8000

	ejecuta(t, a, abreElModalDeSesiones(t, a))
	tecla(t, a, tea.KeyDown) // resalta s2
	ejecuta(t, a, tecla(t, a, tea.KeyEnter))

	if a.Panel.SesionID != "s2" {
		t.Fatalf("la sesión activa es la elegida: %q", a.Panel.SesionID)
	}
	if a.Panel.ContextoTokens != 7 || a.Panel.LimiteTokens != 8000 {
		t.Fatalf("el CONTEXTO se refresca con la sesión cargada: %d/%d",
			a.Panel.ContextoTokens, a.Panel.LimiteTokens)
	}
	if !a.Panel.TokensEstimados {
		t.Error("el contexto cargado es una estimación")
	}
}

// Crear una sesión no conserva el contexto de la anterior: se pide su historial
// —vacío— y el CONTEXTO queda a cero con el límite del modelo.
func TestCrearSesionRefrescaElContexto(t *testing.T) {
	a, p := appConModales(t)
	a.Panel.ContextoTokens = 123
	a.Panel.LimiteTokens = 1000
	a.Panel.TokensEstimados = true
	p.contextoTokens = 0
	p.limiteTokens = 8000

	ejecutaLote(t, a, secuencia(t, a, tea.KeyCtrlX, "n"))

	if a.Panel.SesionID != "nueva" {
		t.Fatalf("la sesión creada queda activa: %q", a.Panel.SesionID)
	}
	if a.Panel.ContextoTokens != 0 || a.Panel.LimiteTokens != 8000 {
		t.Fatalf("una sesión nueva no conserva el contexto de la anterior: %d/%d",
			a.Panel.ContextoTokens, a.Panel.LimiteTokens)
	}
	if p.peticionesChat == 0 {
		t.Error("crear una sesión pide su historial para refrescar el contexto")
	}
}

// El razonamiento revelado es una preferencia de la vista: cambiar de sesión no
// la pierde (antes se apagaba a la fuerza).
func TestCambiarDeSesionConservaElRazonamientoRevelado(t *testing.T) {
	a, _ := appConModales(t)
	a.Razon.Alternar() // revela el texto crudo
	a.Chat.MostrarRazonamiento = true

	ejecuta(t, a, abreElModalDeSesiones(t, a))
	tecla(t, a, tea.KeyDown)
	ejecuta(t, a, tecla(t, a, tea.KeyEnter))

	if !a.Razon.Visible {
		t.Error("el Ctrl+R se conserva al cambiar de sesión")
	}
	if !a.Chat.MostrarRazonamiento {
		t.Error("el historial sigue mostrando el razonamiento revelado")
	}
}

// Retomar una sesión que ya estaba trabajando reanuda su reloj y arma el latido
// del glifo: la animación no se queda congelada tras el cambio.
func TestRetomarUnaSesionQueTrabajaReanudaElRelojYElGlifo(t *testing.T) {
	a := Nuevo(&puertoStub{})
	a.Vista = VistaPrincipal
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})
	a.activar(&session.Sesion{ID: "s2", Nombre: "segunda", Estado: session.EstadoTrabajando})

	cmd := pulsa(t, a, historialMsg{Sesion: "s2"})
	if !a.Chat.HayTurno() {
		t.Fatal("retomar una sesión que trabaja reanuda su reloj")
	}
	if cmd == nil {
		t.Fatal("el reloj reanudado arma el latido")
	}
	antes := a.frame
	pulsa(t, a, tickMsg{gen: a.latido})
	if a.frame == antes {
		t.Error("el glifo avanza con el latido")
	}
}

// Un token de otra sesión no entra al chat de la activa (ni suma a su consumo).
func TestUnTokenDeOtraSesionNoSePintaEnLaActiva(t *testing.T) {
	a := Nuevo(&puertoStub{})
	a.Vista = VistaPrincipal
	a.Panel.SesionID = "s1"

	pulsa(t, a, eventoMsg{Evento: Evento{Nombre: EventoToken, Datos: map[string]string{"sesion": "s2", "texto": "ajena"}}})
	if a.Chat.EnCurso() != "" || a.Panel.Tokens != 0 {
		t.Fatalf("el token de otra sesión no entra: enCurso=%q tokens=%d", a.Chat.EnCurso(), a.Panel.Tokens)
	}
	pulsa(t, a, eventoMsg{Evento: Evento{Nombre: EventoToken, Datos: map[string]string{"sesion": "s1", "texto": "mía"}}})
	if a.Chat.EnCurso() != "mía" {
		t.Fatalf("el token de la activa sí entra: %q", a.Chat.EnCurso())
	}
}

// Una herramienta de otra sesión no enciende el indicador ni deja su línea en el
// chat de la activa.
func TestUnaHerramientaDeOtraSesionNoSePintaEnLaActiva(t *testing.T) {
	a := Nuevo(&puertoStub{})
	a.Vista = VistaPrincipal
	a.Panel.SesionID = "s1"

	pulsa(t, a, eventoMsg{Evento: Evento{Nombre: EventoHerramientaInvocada, Datos: map[string]string{
		"sesion": "s2", "herramienta": "leer_archivo", "verbo": "LEER", "tema": "x.go",
	}}})
	if a.herramientaEnCurso != "" {
		t.Fatalf("la herramienta de otra sesión no enciende el indicador: %q", a.herramientaEnCurso)
	}
	if len(a.Chat.Mensajes()) != 0 {
		t.Fatalf("la línea de otra sesión no entra al chat: %+v", a.Chat.Mensajes())
	}
}

// La notificación de otra sesión se ve (SPEC-SESIONES) pero no cierra el
// razonamiento vivo de la sesión que se está mirando.
func TestLaNotificacionDeOtraSesionNoCierraElRazonamientoVivo(t *testing.T) {
	a := Nuevo(&puertoStub{})
	a.Vista = VistaPrincipal
	a.Panel.SesionID = "s1"
	a.Razon.Añadir("pensando en la activa")

	pulsa(t, a, eventoMsg{Evento: Evento{Nombre: session.EventoNotificacion, Datos: map[string]string{
		"sesion": "s2", "motivo": "otra terminó",
	}}})
	if !a.Razon.Hay() {
		t.Error("la notificación de otra sesión no puede vaciar el razonamiento vivo")
	}

	// La de la sesión activa sí cierra el segmento en vivo.
	pulsa(t, a, eventoMsg{Evento: Evento{Nombre: session.EventoNotificacion, Datos: map[string]string{
		"sesion": "s1", "motivo": "la mía terminó",
	}}})
	if a.Razon.Hay() {
		t.Error("la notificación de la activa cierra el razonamiento vivo")
	}
}

// El consumo de un turno de otra sesión no pisa el contador del panel.
func TestTokensTurnoDeOtraSesionNoPisaElContador(t *testing.T) {
	a := Nuevo(&puertoStub{})
	a.Vista = VistaPrincipal
	a.Panel.SesionID = "s1"
	a.Panel.Tokens = 3
	a.Panel.ContextoTokens = 5

	pulsa(t, a, eventoMsg{Evento: Evento{Nombre: EventoTokensTurno, Datos: map[string]string{
		"sesion": "s2", "salida": "999", "contexto": "999", "limite": "999",
	}}})
	if a.Panel.Tokens != 3 || a.Panel.ContextoTokens != 5 {
		t.Fatalf("el consumo de otra sesión no toca el panel: tokens=%d contexto=%d",
			a.Panel.Tokens, a.Panel.ContextoTokens)
	}
}
