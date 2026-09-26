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
	enviados         []string
	resueltas        []string
	pausadas         []string
	cancelado        []string
	activasResueltas int
	suscripciones    int
	canal            chan Evento
	err              error
}

func (p *puertoStub) ResolverActiva() (*session.Sesion, error) {
	if p.err != nil {
		return nil, p.err
	}
	p.activasResueltas++
	if p.activa == nil {
		p.activa = &session.Sesion{ID: "s1", Nombre: "primera", Estado: session.EstadoInactiva}
	}
	copia := *p.activa
	return &copia, nil
}

func (p *puertoStub) Listar() ([]session.Sesion, error) {
	if p.err != nil {
		return nil, p.err
	}
	return p.sesiones, nil
}

func (p *puertoStub) Historial(sesionID string) ([]MensajeHistorial, error) {
	if p.err != nil {
		return nil, p.err
	}
	return p.historial, nil
}

func (p *puertoStub) Enviar(ctx context.Context, sesionID, texto string) error {
	if p.err != nil {
		return p.err
	}
	p.enviados = append(p.enviados, sesionID+"|"+texto)
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
	if len(p.enviados) != 1 || p.enviados[0] != "s1|documentar la capa" {
		t.Fatalf("la petición se envía una vez a la sesión activa: %v", p.enviados)
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

func TestElRazonamientoApareceProgresivamenteYSePuedeOcultar(t *testing.T) {
	a := Nuevo(&puertoStub{})
	a.Vista = VistaPrincipal
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})
	pulsa(t, a, eventoMsg{Evento: Evento{Nombre: EventoToken, Datos: map[string]string{"texto": "primero ", "razonamiento": "true"}}})
	if !strings.Contains(sinEstilo(a.View()), "primero") {
		t.Fatalf("el primer token debe verse:\n%s", sinEstilo(a.View()))
	}
	pulsa(t, a, eventoMsg{Evento: Evento{Nombre: EventoToken, Datos: map[string]string{"texto": "segundo", "razonamiento": "true"}}})
	if !strings.Contains(sinEstilo(a.View()), "primero segundo") {
		t.Fatalf("los tokens se acumulan en orden:\n%s", sinEstilo(a.View()))
	}

	// Ocultarlo lo quita de la vista sin borrar lo acumulado.
	tecla(t, a, tea.KeyCtrlR)
	if strings.Contains(sinEstilo(a.View()), "primero segundo") {
		t.Error("oculto no debe pintarse")
	}
	if a.Razon.Texto() != "primero segundo" {
		t.Errorf("ocultar no puede borrar el razonamiento: %q", a.Razon.Texto())
	}
	tecla(t, a, tea.KeyCtrlR)
	if !strings.Contains(sinEstilo(a.View()), "primero segundo") {
		t.Error("volver a mostrarlo debe recuperarlo entero")
	}
}

func TestLaRespuestaNoSeMezclaConElRazonamiento(t *testing.T) {
	a := Nuevo(&puertoStub{})
	a.Vista = VistaPrincipal
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})
	pulsa(t, a, eventoMsg{Evento: Evento{Nombre: EventoToken, Datos: map[string]string{"texto": "estoy pensando", "razonamiento": "true"}}})
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
		Tokens:             1234,
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
		"Sesión", "api de pedidos", "trabajando",
		"Contexto", "1234 tokens", "(estimado)", "61%",
		"TODO", "T-B014", "quedan 3",
		"Ruta", "/tmp/proyecto",
		"Git", "master",
		"Capa y cola", "backend", "2 tareas grandes",
		"Aprobaciones", "1 esperando decisión",
		"Agente", "build",
		"Proyecto", Nombre, Version,
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
	a.Panel.Tokens, a.Panel.LimiteTokens = 1700, 2000
	pulsa(t, a, tea.WindowSizeMsg{Width: 120, Height: 30})
	if !strings.Contains(sinEstilo(a.View()), "acercando a su límite") {
		t.Errorf("al 85%% del contexto debe avisarse:\n%s", sinEstilo(a.View()))
	}
}

func TestElPanelSeAbreYCierraSinTocarLaEntrada(t *testing.T) {
	a := Nuevo(&puertoStub{})
	a.Vista = VistaPrincipal
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})
	escribe(t, a, "a medio escribir")
	tecla(t, a, tea.KeyCtrlD)
	if !a.Panel.Abierto || !strings.Contains(sinEstilo(a.View()), "PANEL") {
		t.Fatal("ctrl+d abre el panel")
	}
	if a.Entrada.Texto() != "a medio escribir" {
		t.Error("abrir el panel no puede perder lo escrito")
	}
	tecla(t, a, tea.KeyCtrlD)
	if a.Panel.Abierto || strings.Contains(sinEstilo(a.View()), "PANEL") {
		t.Fatal("ctrl+d cierra el panel")
	}
	if a.Entrada.Texto() != "a medio escribir" {
		t.Error("cerrar el panel no puede perder lo escrito")
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

func TestElSelectorListaYCambiaDeSesión(t *testing.T) {
	p := &puertoStub{
		activa: &session.Sesion{ID: "s1", Nombre: "primera", Estado: session.EstadoInactiva},
		sesiones: []session.Sesion{
			{ID: "s1", Nombre: "primera", Estado: session.EstadoInactiva},
			{ID: "s2", Nombre: "segunda", Estado: session.EstadoTrabajando},
		},
	}
	a := Nuevo(p)
	// El selector existe en la interfaz principal, no en la bienvenida.
	a.Vista = VistaPrincipal
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})
	// Se abre la sesión activa con una primera petición.
	escribe(t, a, "abre la sesión")
	ejecuta(t, a, pulsa(t, a, tea.KeyMsg{Type: tea.KeyEnter}))
	if a.Panel.SesionID != "s1" {
		t.Fatalf("sesión activa = %q", a.Panel.SesionID)
	}
	a.Chat.AñadirUsuario("de la primera")

	tecla(t, a, tea.KeyCtrlS)
	if !a.Selector.Abierto {
		t.Fatal("ctrl+s abre el selector")
	}
	v := sinEstilo(a.View())
	if !strings.Contains(v, "primera") || !strings.Contains(v, "segunda") || !strings.Contains(v, "trabajando") {
		t.Fatalf("el selector muestra nombre y estado de cada sesión:\n%s", v)
	}
	tecla(t, a, tea.KeyDown)
	tecla(t, a, tea.KeyEnter)

	if a.Selector.Abierto {
		t.Error("elegir una sesión cierra el selector")
	}
	if a.Panel.SesionID != "s2" {
		t.Errorf("la sesión activa debe ser la elegida: %q", a.Panel.SesionID)
	}
	if len(a.Chat.Mensajes()) != 0 {
		t.Error("cambiar de sesión empieza un chat limpio: los chats no se mezclan")
	}
}

func TestElSelectorSeCierraConEsc(t *testing.T) {
	p := &puertoStub{sesiones: []session.Sesion{{ID: "s1", Nombre: "una"}}}
	a := Nuevo(p)
	// El selector solo existe en la interfaz principal (INTERFACES §4: en la
	// bienvenida no hay atajos de selector ni panel).
	a.Vista = VistaPrincipal
	tecla(t, a, tea.KeyCtrlS)
	if !a.Selector.Abierto {
		t.Fatal("el selector debería estar abierto")
	}
	tecla(t, a, tea.KeyEsc)
	if a.Selector.Abierto {
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
	if !strings.Contains(sinEstilo(a.View()), "obsoleta") {
		t.Error("una aprobación que ya no aplica se marca como obsoleta")
	}
	tecla(t, a, tea.KeyCtrlA)
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
		"ctrl+q": AccionSalir,
		"ctrl+d": AccionPanel,
		"ctrl+s": AccionSelector,
		"ctrl+r": AccionRazonamiento,
		"ctrl+a": AccionAprobaciones,
		"ctrl+f": AccionCancelar,
		"?":      AccionAyuda,
		"esc":    AccionCerrarSelector,
	}
	for tecla, quiere := range casos {
		got, ok := AccionDe(atajos, tecla)
		if !ok || got != quiere {
			t.Errorf("AccionDe(%q) = %d,%v; quería %d", tecla, got, ok, quiere)
		}
	}
	if _, ok := AccionDe(atajos, "j"); ok {
		t.Error("una letra suelta no puede ser atajo: se está escribiendo")
	}
	// La ayuda lista cada atajo con su acción.
	ayuda := AyudaAtajos(atajos)
	for _, a := range atajos {
		if !strings.Contains(ayuda, a.Tecla) || !strings.Contains(ayuda, a.Descripcion) {
			t.Errorf("la ayuda no lista %s", a.Tecla)
		}
	}
	// Dos atajos con la misma tecla no valen.
	mala := []Atajo{{Tecla: "ctrl+o", Accion: AccionPanel}, {Tecla: "ctrl+o", Accion: AccionSalir}}
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
// (INTERFACES §4: reasignar solo cambia la forma de invocar, T-F010-04).
func TestUnAtajoReasignadoDisparaLaMismaAcción(t *testing.T) {
	p := &puertoStub{}
	a := Nuevo(p)
	a.Vista = VistaPrincipal
	a.Panel.SesionID = "s1"
	a.Atajos = []Atajo{{Tecla: "ctrl+k", Accion: AccionPanel, Descripcion: "panel"}}

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
