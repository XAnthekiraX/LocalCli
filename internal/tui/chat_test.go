package tui

// Tests de T-F005: el historial de la sesión activa. Una prueba, una regla
// (frontend/05-quality/TESTING.md §3):
//
//	T-F005-02 → los tokens llenan el bloque según su marca
//	T-F005-03 → razonamiento arriba de la respuesta, sin mezclar
//	T-F005-04 → ocultar no borra ni detiene la acumulación
//	T-F005-05 → al cambiar de sesión llega el historial de esa sesión
//	T-F005-06 → las propuestas se ven en el chat y salen al resolverse
//	T-F005-07 → el chat recorta por arriba conservando el final visible

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"localcli/internal/session"
)

// --- T-F005-03: el intercambio cerrado ---------------------------------------

func TestElHistorialCierraElIntercambioRazonamientoArribaDeLaRespuesta(t *testing.T) {
	c := Chat{}
	c.Token("la respuesta es 4")
	c.CerrarTurno("estuve pensando")

	msgs := c.Mensajes()
	if len(msgs) != 1 {
		t.Fatalf("un intercambio cerrado: %+v", msgs)
	}
	plano := sinEstilo(c.Render(80))
	if !strings.Contains(plano, "estuve pensando") || !strings.Contains(plano, "la respuesta es 4") {
		t.Fatalf("deben verse el razonamiento y la respuesta:\n%s", plano)
	}
	if strings.Index(plano, "estuve pensando") >= strings.Index(plano, "la respuesta es 4") {
		t.Error("en el historial también va el razonamiento arriba de la respuesta")
	}
	if !strings.Contains(plano, "estuve pensando\n\nla respuesta es 4") {
		t.Error("razonamiento y respuesta cerrados van separados, sin mezclarse")
	}
}

// --- T-F005-02: la acumulación en vivo ---------------------------------------

func TestLosTokensLlenanElBloqueQueToca(t *testing.T) {
	a := Nuevo(&puertoStub{})
	a.Vista = VistaPrincipal
	a.Panel.SesionID = "s1"
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})

	pulsa(t, a, eventoMsg{Evento: Evento{Nombre: EventoToken, Datos: map[string]string{"texto": "uno ", "razonamiento": "true"}}})
	pulsa(t, a, eventoMsg{Evento: Evento{Nombre: EventoToken, Datos: map[string]string{"texto": "dos"}}})
	if a.Razon.Texto() != "uno " {
		t.Errorf("el razonamiento acumula los marcados: %q", a.Razon.Texto())
	}
	if a.Chat.EnCurso() != "dos" {
		t.Errorf("la respuesta acumula los demás: %q", a.Chat.EnCurso())
	}
}

// --- T-F005-04: ocultar no borra ----------------------------------------------

func TestOcultarElRazonamientoNoBorraNiDetieneLaAcumulación(t *testing.T) {
	a := Nuevo(&puertoStub{})
	a.Vista = VistaPrincipal
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})

	pulsa(t, a, eventoMsg{Evento: Evento{Nombre: EventoToken, Datos: map[string]string{"texto": "parte uno ", "razonamiento": "true"}}})
	tecla(t, a, tea.KeyCtrlR) // ocultar
	if strings.Contains(sinEstilo(a.View()), "parte uno") {
		t.Error("oculto no debe verse")
	}
	// Mientras está oculto sigue llegando razonamiento...
	pulsa(t, a, eventoMsg{Evento: Evento{Nombre: EventoToken, Datos: map[string]string{"texto": "y parte dos", "razonamiento": "true"}}})
	if a.Razon.Texto() != "parte uno y parte dos" {
		t.Errorf("ocultar no detiene la acumulación: %q", a.Razon.Texto())
	}
	tecla(t, a, tea.KeyCtrlR) // mostrar de nuevo
	if !strings.Contains(sinEstilo(a.View()), "parte uno y parte dos") {
		t.Error("al mostrar aparece todo lo acumulado, también lo llegado oculto")
	}
	// ...y la respuesta sigue creciendo mientras tanto.
	pulsa(t, a, eventoMsg{Evento: Evento{Nombre: EventoToken, Datos: map[string]string{"texto": "resp"}}})
	if a.Chat.EnCurso() != "resp" {
		t.Errorf("la generación sigue su curso: %q", a.Chat.EnCurso())
	}
}

// --- T-F005-05: la carga del historial ----------------------------------------

func TestAlCambiarDeSesiónLlegaElHistorialDeEsaSesión(t *testing.T) {
	p := &puertoStub{
		sesiones: []session.Sesion{
			{ID: "s1", Nombre: "primera"},
			{ID: "s2", Nombre: "segunda"},
		},
		historial: []MensajeHistorial{
			{Rol: "user", Texto: "pregunta vieja"},
			{Rol: "agent", Texto: "respuesta vieja", Razonamiento: "razonamiento viejo"},
		},
	}
	a := Nuevo(p)
	a.Vista = VistaPrincipal
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})

	tecla(t, a, tea.KeyCtrlS)
	tecla(t, a, tea.KeyDown)
	// Elegir la sesión dispara la carga: el comando la pide y la entrega.
	cmd := pulsa(t, a, tea.KeyMsg{Type: tea.KeyEnter})
	if a.Panel.SesionID != "s2" {
		t.Fatalf("sesión activa = %q", a.Panel.SesionID)
	}
	if len(a.Chat.Mensajes()) != 0 {
		t.Fatal("el historial no está en memoria: llega como dato")
	}
	ejecuta(t, a, cmd)

	if len(a.Chat.Mensajes()) != 2 {
		t.Fatalf("el historial de la sesión elegida se pinta: %+v", a.Chat.Mensajes())
	}
	v := sinEstilo(a.View())
	if !strings.Contains(v, "pregunta vieja") || !strings.Contains(v, "respuesta vieja") {
		t.Fatalf("el historial se ve:\n%s", v)
	}
	if !strings.Contains(v, "razonamiento viejo") {
		t.Error("el razonamiento cerrado del historial también se ve")
	}
	// Lo nuevo de esta sesión convive con su historial cargado.
	pulsa(t, a, eventoMsg{Evento: Evento{Nombre: EventoToken, Datos: map[string]string{"texto": "nuevo"}}})
	v = sinEstilo(a.View())
	if !strings.Contains(v, "nuevo") || !strings.Contains(v, "respuesta vieja") {
		t.Errorf("el chat pinta el historial y lo nuevo de la sesión:\n%s", v)
	}
}

func TestUnHistorialQueLlegaTardeSeDescarta(t *testing.T) {
	p := &puertoStub{}
	a := Nuevo(p)
	a.Vista = VistaPrincipal
	a.Panel.SesionID = "s1"
	// Llega el historial de otra sesión (el usuario ya cambió mientras volaba).
	pulsa(t, a, historialMsg{Sesion: "s2", Mensajes: []MensajeHistorial{{Rol: "user", Texto: "de otra sesión"}}})
	if len(a.Chat.Mensajes()) != 0 {
		t.Error("el chat nunca muestra el historial de otra sesión")
	}
}

// --- T-F005-06: las propuestas dentro del chat ---------------------------------

func TestLaPropuestaDeLaSesiónActivaSeVeEnElChat(t *testing.T) {
	p := &puertoStub{}
	a := Nuevo(p)
	a.Vista = VistaPrincipal
	a.Panel.SesionID = "s1"
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})

	pulsa(t, a, eventoMsg{Evento: Evento{
		Nombre: EventoPeticionAprobacion,
		Datos:  map[string]string{"aprobacion": "a1", "sesion": "s1", "descripcion": "crear archivo"},
	}})
	if !strings.Contains(sinEstilo(a.View()), "propuesta pendiente: crear archivo") {
		t.Fatalf("la propuesta de la sesión activa se ve en su chat:\n%s", sinEstilo(a.View()))
	}

	// Resuelta (aprobacion_resuelta): la línea sale del chat.
	pulsa(t, a, eventoMsg{Evento: Evento{
		Nombre: EventoAprobacionResuelta,
		Datos:  map[string]string{"aprobacion": "a1"},
	}})
	if a.Chat.RenderPropuestas() != "" {
		t.Error("la propuesta resuelta sale del chat")
	}
}

func TestLaPropuestaDeOtraSesiónNoSeVeEnEsteChat(t *testing.T) {
	a := Nuevo(&puertoStub{})
	a.Vista = VistaPrincipal
	a.Panel.SesionID = "s1"
	pulsa(t, a, eventoMsg{Evento: Evento{
		Nombre: EventoPeticionAprobacion,
		Datos:  map[string]string{"aprobacion": "a2", "sesion": "s2", "descripcion": "borrar carpeta"},
	}})
	if a.Chat.RenderPropuestas() != "" {
		t.Error("el chat de la activa no muestra propuestas de otra sesión")
	}
}

// --- T-F005-07: el recorte por alto ---------------------------------------------

func TestElChatRecortaPorArribaConservandoElFinal(t *testing.T) {
	var lineas []string
	for i := 1; i <= 30; i++ {
		lineas = append(lineas, fmt.Sprintf("línea %d", i))
	}
	texto := strings.Join(lineas, "\n")
	got := recortarAlto(texto, 10)
	if n := len(strings.Split(got, "\n")); n != 10 {
		t.Fatalf("quedan las últimas 10 líneas, hay %d", n)
	}
	// Lo que se recorta es el principio: el final visible es lo último dicho.
	if strings.Contains(got, lineas[0]) {
		t.Error("el principio es lo que sale de la ventana")
	}
	if !strings.Contains(got, lineas[29]) {
		t.Error("lo último dicho debe seguir visible")
	}
	if recortarAlto(texto, 0) != texto {
		t.Error("sin alto conocido no se recorta")
	}
	if recortarAlto("corto", 10) != "corto" {
		t.Error("lo que cabe no se toca")
	}
}
