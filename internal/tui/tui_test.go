package tui

// Tests del módulo tui (T-B014-09). Cubren las verificaciones pedidas en
// 014-task-tui.md subtarea por subtarea:
//
//	01 → el modelo arranca y avanza sin pánico (Update/View directos: la vista
//	     no necesita terminal; `teatest` no está en el stack fijado)
//	02 → enviar texto añade las burbujas en orden
//	03 → los tokens del razonamiento aparecen progresivamente y se pueden ocultar
//	04 → el fixture del panel renderiza las filas esperadas
//	05 → navegar el selector con las teclas documentadas cambia de sesión
//	06 → aprobar y declinar mandan la decisión de esa línea
//	07 → cada tecla del mapa produce su acción
//	08 → la vista no importa flow, tools ni store
//	09 → este propio fichero: go test ./internal/tui/... pasa

import (
	"context"
	"errors"
	"go/parser"
	"go/token"
	"os"
	"regexp"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"localcli/internal/session"
)

// --- dobles ---------------------------------------------------------------

type puertoStub struct {
	activa           *session.Sesion
	sesiones         []session.Sesion
	historial        []MensajeHistorial
	tareas           []TareaPanel
	enviados         []string
	resueltas        []string
	pausadas         []string
	cancelado        []string
	activasResueltas int
	suscripciones    int
	canal            chan Evento
	err              error
	modelos          []ModeloLocal
	modelosErr       error
	modelo           string
	agenteRecordado  string
	// carpeta es la carpeta del proyecto que el arranque resolvió; es lo que la
	// vista pinta en el pie del panel.
	carpeta string
	// gitRama y gitLimpio son el estado del repositorio que el arranque leyó;
	// gitRama vacía es el proyecto sin git inicializado.
	gitRama   string
	gitLimpio bool
	// agentesDisponibles es la lista que ofrece el puerto; vacía deja los base
	// plan/build.
	agentesDisponibles []string
	// comandos es el catálogo que ofrece el puerto para la paleta; vacío deja el
	// respaldo oficial de la vista.
	comandos        []ComandoFlujo
	capHerramientas bool
	capVision       bool
	capErr          error
	capConsultas    int
	// imagenesEnviadas guarda, por envío, las imágenes (base64) que la vista
	// detectó en el texto, en el mismo orden que `enviados`.
	imagenesEnviadas  [][]string
	peticionesModelos int
	fijados           []string
	// Lecturas bajo demanda del modal de sesiones: cada apertura pide la lista
	// una vez (T-F014-04) y elegir una sesión pide su historial.
	lecturas       int
	peticionesChat int
	// agentes es el agente con el que salió cada petición, en el mismo orden
	// que `enviados` (T-F015-04).
	agentes []string
	// creadas y eliminadas registran las operaciones de `session_new` y
	// `session_delete` (T-F017).
	creadas    int
	eliminadas []string
	// contextoTokens y limiteTokens son los números del panel de contexto que
	// devuelve la carga del historial (SPEC-PANEL-CONTEXTO).
	contextoTokens int
	limiteTokens   int
}

func (p *puertoStub) ResolverActiva() (*session.Sesion, error) {
	if p.err != nil {
		return nil, p.err
	}
	p.activasResueltas++
	// Retoma sin crear: la activa si sigue viva, o la primera viva de la lista.
	// Sin sesiones devuelve nil (la bienvenida creará una al enviar).
	if p.activa != nil && !p.eliminada(p.activa.ID) {
		copia := *p.activa
		return &copia, nil
	}
	for i := range p.sesiones {
		s := p.sesiones[i]
		if s.Estado == session.EstadoTerminada || s.Estado == session.EstadoError {
			continue
		}
		copia := s
		p.activa = &copia
		return &copia, nil
	}
	return nil, nil
}

// eliminada dice si ese id ya no está en el listado del doble. Sin sesiones que
// listar no hay con qué decidir, así que la activa se da por vigente.
func (p *puertoStub) eliminada(id string) bool {
	for _, s := range p.sesiones {
		if s.ID == id {
			return false
		}
	}
	return len(p.sesiones) > 0
}

func (p *puertoStub) Listar() ([]session.Sesion, error) {
	p.lecturas++
	if p.err != nil {
		return nil, p.err
	}
	return p.sesiones, nil
}

func (p *puertoStub) Crear() (*session.Sesion, error) {
	if p.err != nil {
		return nil, p.err
	}
	p.creadas++
	ses := &session.Sesion{ID: "nueva", Nombre: session.NombreProvisional, Estado: session.EstadoInactiva}
	p.activa = ses
	return ses, nil
}

func (p *puertoStub) Eliminar(sesionID string) error {
	if p.err != nil {
		return p.err
	}
	p.eliminadas = append(p.eliminadas, sesionID)
	for i, s := range p.sesiones {
		if s.ID == sesionID {
			p.sesiones = append(p.sesiones[:i], p.sesiones[i+1:]...)
			break
		}
	}
	return nil
}

func (p *puertoStub) Historial(sesionID string) (HistorialSesion, error) {
	p.peticionesChat++
	if p.err != nil {
		return HistorialSesion{}, p.err
	}
	return HistorialSesion{
		Mensajes:       p.historial,
		ContextoTokens: p.contextoTokens,
		LimiteTokens:   p.limiteTokens,
	}, nil
}

// Tareas simula la lectura de la lista de pasos de una sesión.
func (p *puertoStub) Tareas(sesionID string) ([]TareaPanel, error) {
	if p.err != nil {
		return nil, p.err
	}
	return p.tareas, nil
}

// Enviar registra también el agente: lo que se pide con `build` sale con
// `build` (T-F015-04).
func (p *puertoStub) Enviar(ctx context.Context, sesionID, agente, texto string, imagenes []string) error {
	if p.err != nil {
		return p.err
	}
	p.enviados = append(p.enviados, sesionID+"|"+texto)
	p.agentes = append(p.agentes, agente)
	p.imagenesEnviadas = append(p.imagenesEnviadas, imagenes)
	return nil
}

func (p *puertoStub) Pendientes() ([]Aprobacion, error) { return nil, nil }

func (p *puertoStub) Resolver(aprobacionID string, aprobar bool) error {
	if p.err != nil {
		return p.err
	}
	verbo := "declinar"
	if aprobar {
		verbo = "aprobar"
	}
	p.resueltas = append(p.resueltas, aprobacionID+":"+verbo)
	return nil
}

func (p *puertoStub) Pausar(sesionID string) error {
	p.pausadas = append(p.pausadas, sesionID)
	return nil
}

func (p *puertoStub) Cancelar(sesionID string) { p.cancelado = append(p.cancelado, sesionID) }

func (p *puertoStub) Modelos() ([]ModeloLocal, error) {
	p.peticionesModelos++
	if p.modelosErr != nil {
		return nil, p.modelosErr
	}
	return p.modelos, nil
}

// ModeloActual es lo que el motor autodetectó al arrancar; el doble lo declara
// en `modelo` para que la línea de modelo de la bienvenida tenga algo que
// enseñar sin llamar a Ollama.
func (p *puertoStub) ModeloActual() string { return p.modelo }

// Carpeta simula la carpeta del proyecto con la que se arrancó la herramienta;
// el doble la declara en `carpeta` para que el pie del panel tenga una ruta que
// pintar.
func (p *puertoStub) Carpeta() string { return p.carpeta }

// Git simula el estado del repositorio del proyecto: rama vacía = sin git
// inicializado, que el pie pinta como «sin iniciar».
func (p *puertoStub) Git() (string, bool) { return p.gitRama, p.gitLimpio }

// AgenteRecordado simula la preferencia leída al arrancar; vacío = sin
// preferencia (la vista cae en plan).
func (p *puertoStub) AgenteRecordado() string { return p.agenteRecordado }

// Agentes simula la lista de agentes disponibles. Vacía deja los base
// (`plan`, `build`), que es lo que ofrece la vista sin catálogo propio.
func (p *puertoStub) Agentes() []string { return p.agentesDisponibles }

func (p *puertoStub) Comandos() []ComandoFlujo { return p.comandos }

// CapacidadesModelo simula la consulta de capacidades del modelo en uso.
func (p *puertoStub) CapacidadesModelo(nombre string) (Capacidades, error) {
	p.capConsultas++
	return Capacidades{Herramientas: p.capHerramientas, Vision: p.capVision}, p.capErr
}

func (p *puertoStub) FijarModelo(nombre string) {
	p.fijados = append(p.fijados, nombre)
	p.modelo = nombre
}

func (p *puertoStub) Suscribir() (<-chan Evento, func()) {
	p.suscripciones++
	p.canal = make(chan Evento, 8)
	return p.canal, func() {}
}

// --- helpers ---------------------------------------------------------------

var ansiRE = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// sinEstilo quita los códigos de color para poder comparar el texto que ve la
// persona, que es lo que la spec describe.
func sinEstilo(s string) string { return ansiRE.ReplaceAllString(s, "") }

func pulsa(t *testing.T, a *App, m tea.Msg) tea.Cmd {
	t.Helper()
	_, cmd := a.Update(m)
	return cmd
}

// ejecuta corre el comando que devolvió el Update y le entrega su mensaje, que
// es lo que hace el bucle de Bubble Tea en vivo.
func ejecuta(t *testing.T, a *App, cmd tea.Cmd) {
	t.Helper()
	if cmd == nil {
		return
	}
	if msg := cmd(); msg != nil {
		pulsa(t, a, msg)
	}
}

func escribe(t *testing.T, a *App, texto string) {
	t.Helper()
	for _, r := range texto {
		pulsa(t, a, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
}

func tecla(t *testing.T, a *App, k tea.KeyType) tea.Cmd {
	t.Helper()
	return pulsa(t, a, tea.KeyMsg{Type: k})
}

// secuencia pulsa la líder y su segunda tecla, como el teclado real: el
// indicador aparece y desaparece solo, sin que la prueba tenga que saber cómo
// está armado el resolver.
func secuencia(t *testing.T, a *App, k tea.KeyType, segunda string) tea.Cmd {
	t.Helper()
	tecla(t, a, tea.KeyCtrlX)
	return pulsa(t, a, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(segunda)})
}

// abreElModalDeSesiones pulsa la secuencia documentada del modal de sesiones
// (`<leader>l`) y entrega la lectura que devuelve el puerto, como hace el bucle
// real: el comando de la apertura produce `sesionesMsg` y el Update lo aplica
// (T-F014-04).
func abreElModalDeSesiones(t *testing.T, a *App) tea.Cmd {
	t.Helper()
	return secuencia(t, a, tea.KeyCtrlX, "l")
}

// abreElModalDeModelos pulsa la secuencia documentada del modal de modelos y
// entrega la lista que devuelve el puerto, como hace el bucle real: el comando
// de la apertura produce `modelosMsg` y el Update lo aplica (T-F013).
func abreElModalDeModelos(t *testing.T, a *App) tea.Cmd {
	t.Helper()
	return secuencia(t, a, tea.KeyCtrlX, "m")
}

// --- T-B014-01: arranque ---------------------------------------------------

func TestElModeloArrancaYSePintaSinPánico(t *testing.T) {
	a := Nuevo(&puertoStub{})
	// Init arma la escucha de eventos (T-F010-06): devuelve su comando, pero
	// no consulta la base ni el modelo, y la bienvenida se pinta igual.
	if a.Init() == nil {
		t.Error("Init arma la escucha del canal de eventos")
	}
	if a.Vista != VistaBienvenida {
		t.Fatal("la primera vista es la bienvenida (SPEC-INTERFAZ)")
	}
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})
	if v := a.View(); v == "" {
		t.Fatal("la bienvenida no puede quedar vacía")
	}
	// La bienvenida se pinta sin base, sin Ollama y sin sesiones.
	a2 := Nuevo(&puertoStub{err: errors.New("la base no responde")})
	if v := a2.View(); !strings.Contains(sinEstilo(v), Nombre) {
		t.Fatalf("la bienvenida debe pintarse aunque la base falle:\n%s", v)
	}
}

// --- T-B014-02: la bienvenida y el chat -----------------------------------

func TestLaBienvenidaSoloTieneLogotipoNombreYEntrada(t *testing.T) {
	a := Nuevo(&puertoStub{})
	v := sinEstilo(a.View())
	if !strings.Contains(v, strings.TrimRight(LogoCanonico, "\n")) {
		t.Error("la bienvenida debe llevar el logotipo")
	}
	if !strings.Contains(v, Nombre+" · "+Version) {
		t.Error("la bienvenida debe llevar el nombre con su versión")
	}
	if !strings.Contains(v, "En qué te ayudo hoy:") {
		t.Error("la bienvenida debe tener una línea de entrada")
	}
	for _, prohibido := range []string{"PANEL", "SESIONES", "APROBACIONES", "Contexto"} {
		if strings.Contains(v, prohibido) {
			t.Errorf("la bienvenida no debe mostrar %q", prohibido)
		}
	}
}

func TestEnviarEnLaBienvenidaAbreLaInterfazUnaVez(t *testing.T) {
	p := &puertoStub{}
	a := Nuevo(p)
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})
	escribe(t, a, "documentar la capa")
	if !strings.Contains(sinEstilo(a.View()), "documentar la capa") {
		t.Fatal("lo escrito debe verse en la línea de entrada antes de enviar")
	}
	ejecuta(t, a, pulsa(t, a, tea.KeyMsg{Type: tea.KeyEnter}))

	if a.Vista != VistaPrincipal {
		t.Fatal("enviar la primera petición cambia a la interfaz principal")
	}
	if len(p.enviados) != 1 || p.enviados[0] != "nueva|documentar la capa" {
		t.Fatalf("la petición se envía una vez a la sesión recién creada: %v", p.enviados)
	}
	if len(a.Chat.Mensajes()) != 1 || a.Chat.Mensajes()[0].Texto != "documentar la capa" {
		t.Fatalf("el chat debe tener el mensaje como primero: %+v", a.Chat.Mensajes())
	}
	if a.Entrada.Texto() != "" {
		t.Error("tras enviar, la entrada se vacía")
	}
	// No se repite ni se pide confirmación: la petición sale una sola vez.
	view := sinEstilo(a.View())
	if strings.Count(view, "documentar la capa") != 1 {
		t.Errorf("la petición no debe aparecer dos veces:\n%s", view)
	}
}

func TestElChatMuestraLosMensajesEnOrden(t *testing.T) {
	p := &puertoStub{}
	a := Nuevo(p)
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})
	escribe(t, a, "primera")
	ejecuta(t, a, pulsa(t, a, tea.KeyMsg{Type: tea.KeyEnter}))
	escribe(t, a, "segunda")
	ejecuta(t, a, pulsa(t, a, tea.KeyMsg{Type: tea.KeyEnter}))

	msgs := a.Chat.Mensajes()
	if len(msgs) != 2 || msgs[0].Texto != "primera" || msgs[1].Texto != "segunda" {
		t.Fatalf("orden del chat = %+v", msgs)
	}
	if len(p.enviados) != 2 {
		t.Fatalf("se esperaban dos envíos: %v", p.enviados)
	}
}

// --- T-B014-03: razonamiento en vivo --------------------------------------

func TestElTextoDelRazonamientoSeRevelaConElAtajoYElIndicadorVaAparte(t *testing.T) {
	a := Nuevo(&puertoStub{})
	a.Vista = VistaPrincipal
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})
	pulsa(t, a, eventoMsg{Evento: Evento{Nombre: EventoToken, Datos: map[string]string{"texto": "primero ", "razonamiento": "true"}}})
	v := sinEstilo(a.View())
	if strings.Contains(v, "primero") {
		t.Fatalf("por defecto no se vuelca el razonamiento:\n%s", v)
	}
	if !strings.Contains(v, "Pensando") {
		t.Fatalf("mientras piensa se ve el indicador:\n%s", v)
	}
	// Ctrl+R revela el texto acumulado.
	tecla(t, a, tea.KeyCtrlR)
	if !strings.Contains(sinEstilo(a.View()), "primero") {
		t.Fatalf("revelar muestra el razonamiento:\n%s", sinEstilo(a.View()))
	}
	pulsa(t, a, eventoMsg{Evento: Evento{Nombre: EventoToken, Datos: map[string]string{"texto": "segundo", "razonamiento": "true"}}})
	if !strings.Contains(sinEstilo(a.View()), "primero segundo") {
		t.Fatalf("los tokens se acumulan en orden:\n%s", sinEstilo(a.View()))
	}

	// Volver a ocultarlo lo quita de la vista sin borrar lo acumulado.
	tecla(t, a, tea.KeyCtrlR)
	if strings.Contains(sinEstilo(a.View()), "primero segundo") {
		t.Error("oculto no debe pintarse")
	}
	if a.Razon.Texto() != "primero segundo" {
		t.Errorf("ocultar no puede borrar el razonamiento: %q", a.Razon.Texto())
	}
}

func TestLaRespuestaNoSeMezclaConElRazonamiento(t *testing.T) {
	a := Nuevo(&puertoStub{})
	a.Vista = VistaPrincipal
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})
	pulsa(t, a, eventoMsg{Evento: Evento{Nombre: EventoToken, Datos: map[string]string{"texto": "estoy pensando", "razonamiento": "true"}}})
	tecla(t, a, tea.KeyCtrlR) // revelar el texto para comprobar que no se mezcla
	pulsa(t, a, eventoMsg{Evento: Evento{Nombre: EventoToken, Datos: map[string]string{"texto": "la respuesta es 4"}}})
	v := sinEstilo(a.View())
	if !strings.Contains(v, "estoy pensando") || !strings.Contains(v, "la respuesta es 4") {
		t.Fatalf("deben verse las dos cosas:\n%s", v)
	}
	if strings.Index(v, "estoy pensando") > strings.Index(v, "la respuesta es 4") {
		t.Error("el razonamiento va arriba de la respuesta")
	}
	if a.Chat.EnCurso() != "la respuesta es 4" {
		t.Errorf("la respuesta se acumula aparte: %q", a.Chat.EnCurso())
	}
}

// --- T-B014-04: el panel de datos -----------------------------------------

func TestElPanelMuestraLosNueveDatos(t *testing.T) {
	a := Nuevo(&puertoStub{})
	a.Vista = VistaPrincipal
	a.Panel = NuevoPanel()
	a.Panel.Abierto = true
	a.Panel = Panel{
		Abierto:            true,
		Sesion:             "api de pedidos",
		Estado:             session.EstadoTrabajando,
		ContextoTokens:     1234,
		TokensEstimados:    true,
		LimiteTokens:       2000,
		ElementoActual:     "T-B014",
		ElementosRestantes: 3,
		Ruta:               "/tmp/proyecto",
		GitRama:            "master",
		Capa:               "backend",
		TareasGrandes:      2,
		Aprobaciones:       1,
		Agente:             "build",
		Proyecto:           Nombre,
		Version:            Version,
	}
	pulsa(t, a, tea.WindowSizeMsg{Width: 120, Height: 30})
	v := sinEstilo(a.View())
	for _, esperado := range []string{
		"api de pedidos", "trabajando",
		"CONTEXTO", "1234 tokens", "(estimado)", "61% usada",
		"TODO", "T-B014", "quedan 3",
		"/tmp/proyecto",
		"Git", "master", "con cambios sin confirmar",
		"Capa y cola", "backend", "2 tareas grandes",
		"Aprobaciones", "1 esperando decisión",
		"Agente", "build",
		// El nombre y la versión van en el pie sin etiqueta: es la firma del
		// harness, como la ruta va entre corchetes y sin nombre. La etiqueta
		// «Proyecto» ya no se pinta; el dato, sí.
		Nombre, Version,
	} {
		if !strings.Contains(v, esperado) {
			t.Errorf("el panel no muestra %q:\n%s", esperado, v)
		}
	}
	if a.Panel.PorcentajeContexto() != 61 {
		t.Errorf("porcentaje = %d, quiero 61", a.Panel.PorcentajeContexto())
	}
	if a.Panel.ContextoApretado() {
		t.Error("al 61% del contexto todavía no toca avisar (el umbral es 80%)")
	}
}

func TestElPanelAvisaCuandoElContextoSeAcercaAlLímite(t *testing.T) {
	a := Nuevo(&puertoStub{})
	a.Vista = VistaPrincipal
	a.Panel.Abierto = true
	a.Panel.ContextoTokens, a.Panel.LimiteTokens = 1700, 2000
	pulsa(t, a, tea.WindowSizeMsg{Width: 120, Height: 30})
	if !strings.Contains(sinEstilo(a.View()), "cerca del límite") {
		t.Errorf("al 85%% del contexto debe avisarse:\n%s", sinEstilo(a.View()))
	}
}

func TestElPanelSeAbreYCierraSinTocarLaEntrada(t *testing.T) {
	a := Nuevo(&puertoStub{})
	a.Vista = VistaPrincipal
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})
	escribe(t, a, "a medio escribir")
	// El sidebar arranca visible: el recorrido va de plegar a desplegar.
	tecla(t, a, tea.KeyCtrlD)
	if a.Panel.Abierto || strings.Contains(sinEstilo(a.View()), "CONTEXTO") {
		t.Fatal("ctrl+d cierra el panel")
	}
	if a.Entrada.Texto() != "a medio escribir" {
		t.Error("cerrar el panel no puede perder lo escrito")
	}
	tecla(t, a, tea.KeyCtrlD)
	if !a.Panel.Abierto || !strings.Contains(sinEstilo(a.View()), "CONTEXTO") {
		t.Fatal("ctrl+d abre el panel")
	}
	if a.Entrada.Texto() != "a medio escribir" {
		t.Error("abrir el panel no puede perder lo escrito")
	}
}

// El panel es de la sesión activa y de nadie más.
func TestUnEventoDeOtraSesiónNoCambiaElPanel(t *testing.T) {
	a := Nuevo(&puertoStub{})
	a.Panel.SesionID = "s1"
	a.Panel.Estado = session.EstadoTrabajando
	pulsa(t, a, eventoMsg{Evento: Evento{
		Nombre: session.EventoEstadoSesion,
		Datos:  map[string]string{"sesion": "s2", "estado": session.EstadoTerminada},
	}})
	if a.Panel.Estado != session.EstadoTrabajando {
		t.Errorf("el panel refleja la sesión activa, no otra: %s", a.Panel.Estado)
	}
}

func TestLaNotificaciónLlegaAunqueNoSeVeaLaSesión(t *testing.T) {
	a := Nuevo(&puertoStub{})
	a.Vista = VistaPrincipal
	a.Panel.SesionID = "s1"
	pulsa(t, a, eventoMsg{Evento: Evento{
		Nombre: session.EventoNotificacion,
		Datos: map[string]string{
			"sesion":    "s2",
			"motivo":    "la sesión espera una aprobación",
			"siguiente": "revisa el panel de aprobaciones y decide",
		},
	}})
	v := sinEstilo(a.View())
	if !strings.Contains(v, "la sesión espera una aprobación") || !strings.Contains(v, "revisa el panel") {
		t.Fatalf("la notificación de otra sesión debe verse:\n%s", v)
	}
}

// --- T-B014-05: el selector de sesiones -----------------------------------

func TestElModalDeSesionesListaYCambiaDeSesión(t *testing.T) {
	p := &puertoStub{
		activa: &session.Sesion{ID: "s1", Nombre: "primera", Estado: session.EstadoInactiva},
		sesiones: []session.Sesion{
			{ID: "s1", Nombre: "primera", Estado: session.EstadoInactiva},
			{ID: "s2", Nombre: "segunda", Estado: session.EstadoTrabajando},
		},
	}
	a := Nuevo(p)
	// El selector existe en la interfaz principal, no en la bienvenida. La
	// interfaz principal siempre tiene una sesión activa, así que se fija.
	a.Vista = VistaPrincipal
	a.activar(p.activa)
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})
	// Se envía una primera petición a la sesión activa.
	escribe(t, a, "abre la sesión")
	ejecuta(t, a, pulsa(t, a, tea.KeyMsg{Type: tea.KeyEnter}))
	if a.Panel.SesionID != "s1" {
		t.Fatalf("sesión activa = %q", a.Panel.SesionID)
	}
	a.Chat.AñadirUsuario("de la primera")

	ejecuta(t, a, abreElModalDeSesiones(t, a))
	if !a.Sesiones.Abierto {
		t.Fatal("ctrl+s abre el selector")
	}
	v := sinEstilo(a.View())
	if !strings.Contains(v, "primera") || !strings.Contains(v, "segunda") || !strings.Contains(v, "trabajando") {
		t.Fatalf("el selector muestra nombre y estado de cada sesión:\n%s", v)
	}
	tecla(t, a, tea.KeyDown)
	tecla(t, a, tea.KeyEnter)

	if a.Sesiones.Abierto {
		t.Error("elegir una sesión cierra el selector")
	}
	if a.Panel.SesionID != "s2" {
		t.Errorf("la sesión activa debe ser la elegida: %q", a.Panel.SesionID)
	}
	if len(a.Chat.Mensajes()) != 0 {
		t.Error("cambiar de sesión empieza un chat limpio: los chats no se mezclan")
	}
}

func TestElModalDeSesionesSeCierraConEsc(t *testing.T) {
	p := &puertoStub{sesiones: []session.Sesion{{ID: "s1", Nombre: "una"}}}
	a := Nuevo(p)
	// El selector solo existe en la interfaz principal (INTERFACES §4: en la
	// bienvenida no hay atajos de selector ni panel).
	a.Vista = VistaPrincipal
	ejecuta(t, a, abreElModalDeSesiones(t, a))
	if !a.Sesiones.Abierto {
		t.Fatal("el selector debería estar abierto")
	}
	tecla(t, a, tea.KeyEsc)
	if a.Sesiones.Abierto {
		t.Error("esc cierra el selector")
	}
}

// --- T-B014-06: aprobaciones ----------------------------------------------

func TestLasAprobacionesMandanLaDecisiónDeSuLínea(t *testing.T) {
	p := &puertoStub{}
	a := Nuevo(p)
	a.Vista = VistaPrincipal
	pulsa(t, a, tea.WindowSizeMsg{Width: 120, Height: 30})
	a.Aprobs.Fijar([]Aprobacion{
		{ID: "a1", Sesion: "primera", Descripcion: "crear archivo"},
		{ID: "a2", Sesion: "segunda", Descripcion: "borrar carpeta"},
	})
	tecla(t, a, tea.KeyCtrlA) // enfocar el panel para decidir con el teclado
	v := sinEstilo(a.View())
	for _, esperado := range []string{"primera | crear archivo | aprobar | declinar", "segunda | borrar carpeta"} {
		if !strings.Contains(v, esperado) {
			t.Errorf("falta la línea %q:\n%s", esperado, v)
		}
	}
	pulsa(t, a, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if len(p.resueltas) != 1 || p.resueltas[0] != "a1:aprobar" {
		t.Fatalf("a aprueba la línea seleccionada: %v", p.resueltas)
	}
	if a.Aprobs.Pendientes() != 1 {
		t.Errorf("la línea resuelta sale del panel: quedan %d", a.Aprobs.Pendientes())
	}
	pulsa(t, a, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if len(p.resueltas) != 2 || p.resueltas[1] != "a2:declinar" {
		t.Fatalf("declinar debe resolver la siguiente línea: %v", p.resueltas)
	}
	if got := a.Aprobs.Pendientes(); got != 0 {
		t.Errorf("no debe quedar ninguna pendiente: %d", got)
	}
}

func TestUnaAprobaciónObsoletaNoSeManda(t *testing.T) {
	p := &puertoStub{}
	a := Nuevo(p)
	a.Vista = VistaPrincipal
	a.Aprobs.Fijar([]Aprobacion{{ID: "a1", Sesion: "vieja", Descripcion: "ya no aplica", Obsoleta: true}})
	tecla(t, a, tea.KeyCtrlA) // enfocar el panel: la línea ya se ve, se enfoca para decidir
	if !strings.Contains(sinEstilo(a.View()), "obsoleta") {
		t.Error("una aprobación que ya no aplica se marca como obsoleta")
	}
	pulsa(t, a, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if len(p.resueltas) != 0 {
		t.Errorf("lo obsoleto no se manda a la sesión: %v", p.resueltas)
	}
}

func TestElAvisoDeAprobacionesSeVeConElPanelCerrado(t *testing.T) {
	a := Nuevo(&puertoStub{})
	a.Vista = VistaPrincipal
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})
	a.Panel.Abierto = false
	a.Panel.Aprobaciones = 2
	v := sinEstilo(a.View())
	if !strings.Contains(v, "2 aprobaciones esperando tu decisión") {
		t.Fatalf("el aviso no se puede ocultar:\n%s", v)
	}
	a.Panel.Aprobaciones = 1
	if !strings.Contains(sinEstilo(a.View()), "1 aprobación esperando tu decisión") {
		t.Error("el aviso debe concordar en singular")
	}
}

// --- T-B014-07: atajos -----------------------------------------------------

func TestLosAtajosPorDefectoNoSeSolapan(t *testing.T) {
	atajos := AtajosPorDefecto()
	if err := ValidarAtajos(atajos); err != nil {
		t.Fatalf("los atajos de fábrica deben ser válidos: %v", err)
	}
	casos := map[string]Accion{
		"enter":  AccionEnviar,
		"ctrl+c": AccionSalir,
		"ctrl+d": AccionPanel,
		"ctrl+r": AccionRazonamiento,
		"ctrl+a": AccionAprobaciones,
		"ctrl+f": AccionCancelar,
		"ctrl+p": AccionAyuda,
		"tab":    AccionCiclarAgente,
		"esc":    AccionCerrarSelector,
		"up":     AccionSubir,
		"down":   AccionBajar,
		"a":      AccionAprobar,
		"d":      AccionDeclinar,
	}
	for tecla, quiere := range casos {
		got, ok := AccionDe(atajos, tecla)
		if !ok || got != quiere {
			t.Errorf("AccionDe(%q) = %d,%v; quería %d", tecla, got, ok, quiere)
		}
	}
	// Las dos secuencias con líder son las documentadas y ninguna más.
	if accion, ok := ResolverSecuencia(atajos, "<leader>m"); !ok || accion != AccionModalModelos {
		t.Errorf("<leader>m abre el modal de modelos: %d, %v", accion, ok)
	}
	if accion, ok := ResolverSecuencia(atajos, "<leader>l"); !ok || accion != AccionSelector {
		t.Errorf("<leader>l abre el modal de sesiones: %d, %v", accion, ok)
	}
	if _, ok := AccionDe(atajos, "j"); ok {
		t.Error("una letra suelta no puede ser atajo: se está escribiendo")
	}
	// La ayuda lista cada atajo con su acción.
	ayuda := AyudaAtajos(atajos)
	for _, a := range atajos {
		if !strings.Contains(ayuda, a.Tecla()) || !strings.Contains(ayuda, a.Descripcion) {
			t.Errorf("la ayuda no lista %s", a.Tecla())
		}
	}
	// Dos atajos con la misma tecla no valen.
	mala := []Atajo{{Secuencias: []Secuencia{{Paso1: "ctrl+o"}}, Accion: AccionPanel}, {Secuencias: []Secuencia{{Paso1: "ctrl+o"}}, Accion: AccionSalir}}
	if err := ValidarAtajos(mala); err == nil {
		t.Error("dos acciones con la misma tecla deben rechazarse")
	}
}

func TestLasTeclasDeAcciónFuncionan(t *testing.T) {
	p := &puertoStub{}
	a := Nuevo(p)
	// Cancelar es un atajo de la interfaz principal: en la bienvenida no
	// existe (INTERFACES §4).
	a.Vista = VistaPrincipal
	a.Panel.SesionID = "s1"
	// Ctrl+F pide confirmación; solo tras el sí se corta el trabajo
	// (T-F010-08).
	tecla(t, a, tea.KeyCtrlF)
	if len(p.cancelado) != 0 {
		t.Fatalf("cancelar no corta nada sin confirmación: %v", p.cancelado)
	}
	if !a.PidiendoCancelar {
		t.Fatal("ctrl+f abre la confirmación")
	}
	pulsa(t, a, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if a.PidiendoCancelar || len(p.cancelado) != 0 {
		t.Fatal("el no descarta la cancelación")
	}
	tecla(t, a, tea.KeyCtrlF)
	pulsa(t, a, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	if len(p.cancelado) != 1 || p.cancelado[0] != "s1" {
		t.Errorf("el sí cancela el trabajo en curso: %v", p.cancelado)
	}
}

// Un mapa reasignado por el usuario enruta la misma acción con otra tecla
// (INTERFACES §4: reasignar solo cambia la forma de invocar y surte efecto sin
// reiniciar, T-F010-04).
func TestUnAtajoReasignadoDisparaLaMismaAcción(t *testing.T) {
	p := &puertoStub{}
	a := Nuevo(p)
	a.Vista = VistaPrincipal
	a.Panel.SesionID = "s1"
	// El sidebar arranca visible; aquí se parte de plegado, que es el estado en
	// el que se distingue si una tecla pliega o despliega.
	a.Panel.Abierto = false
	porAccion := MapasPorDefecto()
	porAccion[AccionPanel] = []string{"ctrl+k"}
	km, err := NuevoKeymap(LíderPorDefecto, TimeoutPorDefectoMs, porAccion)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.FijarMapa(km); err != nil {
		t.Fatal(err)
	}

	tecla(t, a, tea.KeyCtrlO)
	if a.Panel.Abierto {
		t.Error("ctrl+o ya no es panel en este mapa")
	}
	tecla(t, a, tea.KeyCtrlK)
	if !a.Panel.Abierto {
		t.Error("ctrl+k dispara la acción reasignada")
	}
}

// --- T-B014-08: la vista no sabe de negocio -------------------------------

// La vista pinta; no importa el motor. Si `tui` empieza a importar `store` o
// `flow`, deja de ser una vista y se convierte en un segundo motor de negocio.
func TestLaVistaNoImportaModulosDeNegocio(t *testing.T) {
	entradas, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	prohibidos := map[string]bool{
		"localcli/internal/store": true,
		"localcli/internal/flow":  true,
		"localcli/internal/tools": true,
		"localcli/internal/agent": true,
		"database/sql":            true,
	}
	for _, e := range entradas {
		n := e.Name()
		if e.IsDir() || !strings.HasSuffix(n, ".go") || strings.HasSuffix(n, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), n, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parsear %s: %v", n, err)
		}
		for _, imp := range f.Imports {
			r := strings.Trim(imp.Path.Value, `"`)
			if prohibidos[r] {
				t.Errorf("internal/tui/%s importa %q: la vista no lleva lógica de negocio", n, r)
			}
		}
	}
}

// --- T-B014-09: la salida dorada del logotipo ------------------------------

func TestElLogotipoCoincideConLaSalidaDorada(t *testing.T) {
	b, err := os.ReadFile("testdata/logo.txt")
	if err != nil {
		t.Fatal(err)
	}
	lineas := strings.Split(strings.TrimRight(string(b), "\n"), "\n")
	if len(lineas) != 6 {
		t.Fatalf("el logotipo son seis líneas, tiene %d", len(lineas))
	}
	ancho := len([]rune(lineas[0]))
	for i, l := range lineas {
		if strings.HasSuffix(l, " ") {
			t.Errorf("la línea %d tiene espacios finales", i+1)
		}
		// El arte son bloques de ancho uniforme: una línea más corta deja el
		// logotipo torcido.
		if len([]rune(l)) != ancho {
			t.Errorf("la línea %d mide %d columnas y la primera %d", i+1, len([]rune(l)), ancho)
		}
	}
	if LogoCanonico != string(b) {
		t.Error("lo incrustado en la vista tiene que ser el archivo canónico, byte a byte")
	}
	a := Nuevo(&puertoStub{})
	if !strings.Contains(a.View(), strings.TrimRight(string(b), "\n")) {
		t.Error("la bienvenida pinta el logotipo tal cual")
	}
}
