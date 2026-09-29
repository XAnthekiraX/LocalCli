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
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"localcli/internal/session"
)

// --- T-F005-03: el intercambio cerrado ---------------------------------------

func TestElHistorialCierraElIntercambioRazonamientoArribaDeLaRespuesta(t *testing.T) {
	c := Chat{}
	// El razonamiento cerrado solo se vuelca cuando está revelado (`Ctrl+R`).
	c.MostrarRazonamiento = true
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
	// Dentro del globo del agente, razonamiento y respuesta van separados por una
	// línea en blanco: no se mezclan visualmente (SPEC-INTERFAZ §Razonamiento).
	iRaz, iResp := -1, -1
	lineas := strings.Split(plano, "\n")
	for i, l := range lineas {
		if strings.Contains(l, "estuve pensando") {
			iRaz = i
		}
		if strings.Contains(l, "la respuesta es 4") {
			iResp = i
		}
	}
	separados := false
	for i := iRaz + 1; i < iResp; i++ {
		// El razonamiento y la respuesta se separan con una línea en blanco
		// (renderIntercambio los une con \n\n); no hay glifos de adorno.
		if strings.TrimSpace(lineas[i]) == "" {
			separados = true
		}
	}
	if !separados {
		t.Errorf("razonamiento y respuesta cerrados van separados, sin mezclarse:\n%s", plano)
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
	// Por defecto el texto no se vuelca; el razonamiento sigue acumulándose.
	if strings.Contains(sinEstilo(a.View()), "parte uno") {
		t.Error("sin revelar no debe verse el texto")
	}
	pulsa(t, a, eventoMsg{Evento: Evento{Nombre: EventoToken, Datos: map[string]string{"texto": "y parte dos", "razonamiento": "true"}}})
	if a.Razon.Texto() != "parte uno y parte dos" {
		t.Errorf("no revelar no detiene la acumulación: %q", a.Razon.Texto())
	}
	tecla(t, a, tea.KeyCtrlR) // revelar
	if !strings.Contains(sinEstilo(a.View()), "parte uno y parte dos") {
		t.Error("al revelar aparece todo lo acumulado, también lo llegado oculto")
	}
	// ...y la respuesta sigue creciendo mientras tanto.
	pulsa(t, a, eventoMsg{Evento: Evento{Nombre: EventoToken, Datos: map[string]string{"texto": "resp"}}})
	if a.Chat.EnCurso() != "resp" {
		t.Errorf("la generación sigue su curso: %q", a.Chat.EnCurso())
	}
	// Ocultarlo de nuevo lo retira sin borrar lo acumulado.
	tecla(t, a, tea.KeyCtrlR)
	if strings.Contains(sinEstilo(a.View()), "parte uno y parte dos") {
		t.Error("oculto no debe verse")
	}
	if a.Razon.Texto() != "parte uno y parte dos" {
		t.Errorf("ocultar no puede borrar el razonamiento: %q", a.Razon.Texto())
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
	tecla(t, a, tea.KeyCtrlR) // revelar el razonamiento del historial
	v := sinEstilo(a.View())
	if !strings.Contains(v, "pregunta vieja") || !strings.Contains(v, "respuesta vieja") {
		t.Fatalf("el historial se ve:\n%s", v)
	}
	if !strings.Contains(v, "razonamiento viejo") {
		t.Error("el razonamiento cerrado del historial se ve al revelarlo")
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

// El historial guardado incluye las líneas de procesamiento (sub-procesos y
// herramientas) y el tiempo de cada respuesta: al recargar una sesión se ven,
// aunque no formen parte del contexto del modelo.
func TestCargarPintaElProcesamientoYElTiempoGuardado(t *testing.T) {
	c := Chat{}
	c.Cargar([]MensajeHistorial{
		{Rol: "user", Texto: "hola"},
		{Rol: "proceso", Texto: "✓ LEER [a] · 1 línea"},
		{Rol: "agent", Texto: "respuesta", Duracion: 2400 * time.Millisecond},
	})
	msgs := c.Mensajes()
	if len(msgs) != 3 {
		t.Fatalf("esperaba usuario + proceso + agente: %+v", msgs)
	}
	if msgs[1].Rol != RolSistema {
		t.Errorf("una línea de proceso se pinta como el sistema: %q", msgs[1].Rol)
	}
	if msgs[2].Duracion != 2400*time.Millisecond {
		t.Errorf("el tiempo guardado se recupera: %v", msgs[2].Duracion)
	}
	plano := sinEstilo(c.Render(80))
	if !strings.Contains(plano, "✓ LEER [a] · 1 línea") {
		t.Errorf("la línea de procesamiento se pinta:\n%s", plano)
	}
	if !strings.Contains(plano, formatearDuracion(2400*time.Millisecond)) {
		t.Errorf("el tiempo guardado se pinta junto a la respuesta:\n%s", plano)
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

// Un razonamiento en vivo enorme no puede empujar el chat fuera de la pantalla
// —y con él bloquear su scroll—: el bloque en curso se acota.
func TestElRazonamientoEnVivoNoBloqueaElScroll(t *testing.T) {
	a := Nuevo(&puertoStub{})
	a.Vista = VistaPrincipal
	a.Panel.SesionID = "s1"
	pulsa(t, a, tea.WindowSizeMsg{Width: 80, Height: 24})
	for i := 1; i <= 40; i++ {
		a.Chat.AñadirSistema(fmt.Sprintf("mensaje %d", i))
	}
	// El razonamiento se pinta por defecto; llega en cantidades enormes.
	for i := 0; i < 80; i++ {
		pulsa(t, a, eventoMsg{Evento: Evento{Nombre: EventoToken, Datos: map[string]string{
			"texto": fmt.Sprintf("razón %d\n", i), "razonamiento": "true"}}})
	}
	if n := altoDe(a.View()); n != a.Alto {
		t.Fatalf("el marco tiene %d filas y la terminal %d", n, a.Alto)
	}
	// El historial se sigue pudiendo recorrer.
	a.Chat.Subir(10)
	if !a.Chat.HayArriba() {
		t.Fatal("el scroll del historial sigue vivo con el razonamiento en curso")
	}
	if strings.Contains(sinEstilo(a.View()), "mensaje 40") {
		t.Error("al subir, el final del historial sale de la ventana")
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

// --- T-F036: los globos del chat --------------------------------------------

func TestElChatPintaCadaMensajeEnSuGlobo(t *testing.T) {
	c := Chat{}
	c.AñadirEntrada("lo que escribo")
	c.Token("lo que responde")
	c.CerrarTurno("")

	crudo := c.Render(60)
	lineas := strings.Split(sinEstilo(crudo), "\n")
	iUsuario, iAgente := -1, -1
	for i, l := range lineas {
		switch {
		case iUsuario < 0 && strings.Contains(l, "lo que escribo"):
			iUsuario = i
		case iAgente < 0 && strings.Contains(l, "lo que responde"):
			iAgente = i
		}
	}
	if iUsuario < 0 || iAgente < 0 {
		t.Fatalf("cada mensaje va en su bloque:\n%s", strings.Join(lineas, "\n"))
	}
	if iUsuario >= iAgente {
		t.Errorf("el mensaje del usuario va antes de la respuesta: %d, %d", iUsuario, iAgente)
	}
	// Cada mensaje ocupa el ancho del chat y no lleva icono: el usuario va pegado
	// a la derecha y el agente a la izquierda.
	if !strings.HasSuffix(strings.TrimRight(lineas[iUsuario], " "), "lo que escribo") {
		t.Errorf("el mensaje del usuario va a la derecha: %q", lineas[iUsuario])
	}
	if !strings.HasPrefix(strings.TrimLeft(lineas[iAgente], " "), "lo que responde") {
		t.Errorf("la respuesta va a la izquierda: %q", lineas[iAgente])
	}
	if strings.Contains(sinEstilo(crudo), "▣") {
		t.Error("los mensajes ya no llevan icono")
	}
	if strings.ContainsAny(sinEstilo(crudo), "╭╮╰╯│─") {
		t.Errorf("los globos no llevan glifos de borde:\n%s", sinEstilo(crudo))
	}
}

// --- T-F039: el orden del texto y las líneas de herramienta -------------------

func TestElTextoAntesYDespuésDeLaHerramientaVanEnSuOrden(t *testing.T) {
	c := Chat{}
	c.AñadirUsuario("lee el archivo")
	c.Token("voy a leerlo")
	// La herramienta se interpone: lo dicho se cierra y su línea va debajo.
	c.CerrarSegmento("")
	c.AnotarInvocacion("LEER", "internal/tui/chat.go")
	c.CerrarHerramienta("leer_archivo", true, false, "70 líneas", "")
	// El modelo sigue tras el resultado: abre un globo nuevo.
	c.Token("ya lo leí")
	c.CerrarTurno("")

	msgs := c.Mensajes()
	if len(msgs) != 4 {
		t.Fatalf("usuario + dos globos del agente + una línea de herramienta: %+v", msgs)
	}
	if msgs[0].Rol != RolUsuario || msgs[0].Texto != "lee el archivo" {
		t.Errorf("primero lo del usuario: %+v", msgs[0])
	}
	if msgs[1].Rol != RolAgente || msgs[1].Texto != "voy a leerlo" {
		t.Errorf("el texto previo a la herramienta va antes de su línea: %+v", msgs[1])
	}
	if msgs[2].Rol != RolSistema || !strings.Contains(msgs[2].Texto, "LEER") {
		t.Errorf("la línea de herramienta va entre los dos globos: %+v", msgs[2])
	}
	if msgs[3].Rol != RolAgente || msgs[3].Texto != "ya lo leí" {
		t.Errorf("el texto posterior abre un globo nuevo: %+v", msgs[3])
	}

	plano := sinEstilo(c.Render(80))
	ip, il, is := strings.Index(plano, "voy a leerlo"), strings.Index(plano, "LEER ["), strings.Index(plano, "ya lo leí")
	if ip < 0 || il < 0 || is < 0 || !(ip < il && il < is) {
		t.Errorf("el render respeta el orden de ejecución (texto %d, herramienta %d, texto %d):\n%s", ip, il, is, plano)
	}
}

func TestCerrarSegmentoSinNadaNoDejaGloboVacio(t *testing.T) {
	c := Chat{}
	c.CerrarSegmento("")
	if len(c.Mensajes()) != 0 {
		t.Errorf("una invocación directa no deja globo vacío: %+v", c.Mensajes())
	}
}

func TestElRazonamientoSeCierraConSuSegmentoDeTexto(t *testing.T) {
	c := Chat{}
	c.MostrarRazonamiento = true
	c.AñadirUsuario("lee")
	c.Token("voy a leer")
	c.CerrarSegmento("pienso leerlo")
	c.AnotarInvocacion("LEER", "x")
	c.Token("listo")
	c.CerrarTurno("pienso responder")

	msgs := c.Mensajes()
	if len(msgs) != 4 {
		t.Fatalf("usuario + dos globos + línea: %+v", msgs)
	}
	if msgs[1].Razonamiento != "pienso leerlo" {
		t.Errorf("cada globo conserva su razonamiento: %+v", msgs[1])
	}
	if msgs[3].Razonamiento != "pienso responder" {
		t.Errorf("el globo final lleva el suyo: %+v", msgs[3])
	}
}

func TestLaDuraciónDelTurnoQueTerminaEnHerramientaNoSePierde(t *testing.T) {
	c := Chat{}
	c.AñadirUsuario("lee")
	c.Token("voy")
	c.CerrarSegmento("")
	c.AnotarInvocacion("LEER", "x")
	c.CerrarHerramienta("leer_archivo", true, false, "", "")
	c.CerrarTurno("") // termina sin texto final

	msgs := c.Mensajes()
	if len(msgs) != 3 {
		t.Fatalf("usuario + globo + línea: %+v", msgs)
	}
	if msgs[1].Duracion <= 0 {
		t.Errorf("la duración se cuelga del último segmento del agente: %+v", msgs[1])
	}
	if msgs[2].Duracion != 0 {
		t.Errorf("la línea de herramienta no lleva duración: %+v", msgs[2])
	}
}
