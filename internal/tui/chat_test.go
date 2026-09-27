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

	ejecuta(t, a, abreElModalDeSesiones(t, a))
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

// --- T-F005-07: la ventana del chat (scroll) ------------------------------------

func TestElChatMuestraElFinalYSePuedeSubir(t *testing.T) {
	var lineas []string
	for i := 1; i <= 30; i++ {
		lineas = append(lineas, fmt.Sprintf("línea %d", i))
	}
	c := Chat{}
	c.AñadirSistema(strings.Join(lineas, "\n"))

	// Sin subir, la ventana se pega al final: lo último dicho sigue a la vista.
	got := c.Ventana(200, 10)
	if n := len(strings.Split(got, "\n")); n != 10 {
		t.Fatalf("la ventana tiene 10 líneas, tiene %d", n)
	}
	if !strings.Contains(got, "línea 30") {
		t.Errorf("al final se ve lo último:\n%s", got)
	}
	if strings.Contains(got, "línea 1\n") {
		t.Errorf("el principio queda fuera de la ventana:\n%s", got)
	}
	if !c.HayArriba() {
		t.Error("quedan líneas arriba: se puede subir")
	}

	// Subir deja de seguir el final y muestra líneas anteriores.
	c.Subir(5)
	got = c.Ventana(200, 10)
	if !strings.Contains(got, "línea 25") {
		t.Errorf("tras subir se ven líneas anteriores:\n%s", got)
	}
	if c.OcultasArriba() != 15 || c.OcultasAbajo() != 5 {
		t.Errorf("ventana en 20 con 30 líneas: arriba %d, abajo %d (quiero 15 y 5)",
			c.OcultasArriba(), c.OcultasAbajo())
	}
	if !c.HayAbajo() {
		t.Error("con líneas por debajo, se puede bajar")
	}

	// Bajar hasta el final vuelve a seguir las respuestas nuevas.
	c.Bajar(5)
	if c.HayAbajo() {
		t.Error("al llegar al final ya no queda nada abajo")
	}
	got = c.Ventana(200, 10)
	if !strings.Contains(got, "línea 30") {
		t.Errorf("volver al final muestra lo último:\n%s", got)
	}
}

func TestLoQueCabeEnteroNoSeRecorta(t *testing.T) {
	c := Chat{}
	c.AñadirSistema("corto")
	if got := sinEstilo(c.Ventana(80, 10)); !strings.Contains(got, "corto") {
		t.Errorf("lo que cabe se ve entero: %q", got)
	}
	if c.HayArriba() || c.HayAbajo() {
		t.Error("si cabe entero no hay scroll")
	}
	if got := sinEstilo(c.Ventana(80, 0)); !strings.Contains(got, "corto") {
		t.Errorf("sin alto conocido se devuelve todo: %q", got)
	}
}

func TestElMarcoCabeEnLaTerminal(t *testing.T) {
	for _, alto := range []int{10, 15, 24, 30, 40} {
		a := Nuevo(&puertoStub{})
		a.Vista = VistaPrincipal
		a.Panel.SesionID = "s1"
		pulsa(t, a, tea.WindowSizeMsg{Width: 80, Height: alto})
		for i := 1; i <= 100; i++ {
			a.Chat.AñadirSistema(fmt.Sprintf("mensaje %d", i))
		}
		if got := altoDe(a.View()); got > alto {
			t.Errorf("alto %d: el marco mide %d líneas y se recortaría la parte de arriba", alto, got)
		}
	}
}

func TestElScrollAlcanzaElPrincipioDeLaConversación(t *testing.T) {
	var lineas []string
	for i := 1; i <= 200; i++ {
		lineas = append(lineas, fmt.Sprintf("mensaje %d", i))
	}
	c := Chat{}
	c.AñadirSistema(strings.Join(lineas, "\n"))

	// Al principio sigue el final.
	c.Ventana(200, 10)

	// Subiendo muchas veces se llega al primer mensaje.
	for i := 0; i < 500; i++ {
		c.Subir(1)
	}
	got := sinEstilo(c.Ventana(200, 10))
	if !strings.Contains(got, "mensaje 1\n") {
		t.Fatalf("subiendo hasta el tope se ve el principio:\n%s", got)
	}
	if c.HayArriba() {
		t.Errorf("en el tope ya no queda nada arriba (offset %d)", c.OcultasArriba())
	}
}

func TestLasFlechasRecorrenElHistorialDelChat(t *testing.T) {
	a := Nuevo(&puertoStub{})
	a.Vista = VistaPrincipal
	a.Panel.SesionID = "s1"
	pulsa(t, a, tea.WindowSizeMsg{Width: 60, Height: 12})
	for i := 1; i <= 20; i++ {
		a.Chat.AñadirSistema(fmt.Sprintf("mensaje %d", i))
	}
	// Al pie: la ventana sigue el final y no hay nada por debajo.
	_ = a.View()
	if a.Chat.OcultasAbajo() != 0 {
		t.Fatalf("al principio se sigue el final, quedan %d abajo", a.Chat.OcultasAbajo())
	}
	// ↑ sube por el historial y deja de seguir el final.
	tecla(t, a, tea.KeyUp)
	tecla(t, a, tea.KeyUp)
	tecla(t, a, tea.KeyUp)
	_ = a.View()
	if a.Chat.OcultasAbajo() != 3 {
		t.Errorf("tras tres ↑ quedan 3 líneas abajo, quedan %d", a.Chat.OcultasAbajo())
	}
	// ↓ hasta el final retoma el seguimiento.
	for i := 0; i < 5; i++ {
		tecla(t, a, tea.KeyDown)
	}
	if a.Chat.OcultasAbajo() != 0 {
		t.Errorf("bajar al final retoma el seguimiento, quedan %d", a.Chat.OcultasAbajo())
	}
}
