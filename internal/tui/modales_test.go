package tui

import (
	"context"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"localcli/internal/tui/render"
)

// puertoFalso es un doble del puerto: valores, sin base ni red. Solo existe en
// las pruebas, que es donde el doble de motor tiene sentido (TESTING §3).
type puertoFalso struct {
	modelos         []ModeloLocal
	sesiones        []Sesion
	flujos          []FlujoVista
	errModelos      error
	modeloFijado    string
	eliminadas      []string
	enviados        []string
	creadas         int
	pensado         bool
	pedido          bool
	historial       []PiezaHistorial
	historialPedido string
	respondidas     []string
	capacidades     Capacidades
	descargado      string
	cancelado       bool
	deshechos       []string
	rehechos        []string
	deshacerN       int
	rehacerN        int
	desdeIDs        []string
	bifurcadas      []string
	compactadas     []string
}

func (p *puertoFalso) Agentes() []string                  { return []string{AgentePlan, AgenteBuild} }
func (p *puertoFalso) AgenteRecordado() string            { return "" }
func (p *puertoFalso) ModeloActual() string               { return "qwen2.5" }
func (p *puertoFalso) MotorActual() string                { return "Ollama local" }
func (p *puertoFalso) Capacidades() Capacidades           { return p.capacidades }
func (p *puertoFalso) Carpeta() string                    { return "/tmp/proyecto" }
func (p *puertoFalso) Git() Git                           { return Git{Rama: "main"} }
func (p *puertoFalso) Suscribir() (<-chan Evento, func()) { return nil, func() {} }
func (p *puertoFalso) Enviar(_ context.Context, _, _, texto string) error {
	p.enviados = append(p.enviados, texto)
	return nil
}

func (p *puertoFalso) ResponderAprobacion(_, aprobacionID string, _ bool) error {
	p.respondidas = append(p.respondidas, aprobacionID)
	return nil
}

func (p *puertoFalso) Crear() (*Sesion, error) {
	p.creadas++
	return &Sesion{ID: "nueva", Nombre: "Nueva sesión", Estado: EstadoInactiva}, nil
}

func (p *puertoFalso) Modelos() ([]ModeloLocal, error) {
	p.pedido = true
	return p.modelos, p.errModelos
}

func (p *puertoFalso) DescargarModelo(ggufURL, mmprojURL string) error {
	p.descargado = ggufURL
	return nil
}
func (p *puertoFalso) CancelarDescarga() { p.cancelado = true }
func (p *puertoFalso) NombreDeURL(url string) string {
	base := url
	if i := strings.IndexByte(base, '?'); i >= 0 {
		base = base[:i]
	}
	if i := strings.LastIndexByte(base, '/'); i >= 0 {
		base = base[i+1:]
	}
	return strings.TrimSuffix(base, ".gguf")
}

func (p *puertoFalso) FijarModelo(nombre string) { p.modeloFijado = nombre }
func (p *puertoFalso) FijarPensar(pensar bool)   { p.pensado = pensar }
func (p *puertoFalso) Flujos() []FlujoVista      { return p.flujos }
func (p *puertoFalso) Listar() ([]Sesion, error) { return p.sesiones, nil }

func (p *puertoFalso) Historial(sesionID string) ([]PiezaHistorial, error) {
	p.historialPedido = sesionID
	return p.historial, nil
}

func (p *puertoFalso) Eliminar(sesionID string) error {
	p.eliminadas = append(p.eliminadas, sesionID)
	return nil
}

func (p *puertoFalso) Deshacer(sesionID string) (int, error) {
	p.deshechos = append(p.deshechos, sesionID)
	return p.deshacerN, nil
}

func (p *puertoFalso) Rehacer(sesionID string) (int, error) {
	p.rehechos = append(p.rehechos, sesionID)
	return p.rehacerN, nil
}

func (p *puertoFalso) DeshacerDesde(sesionID, mensajeID string) (int, error) {
	p.desdeIDs = append(p.desdeIDs, mensajeID)
	return p.deshacerN, nil
}

func (p *puertoFalso) Bifurcar(sesionID, mensajeID string) (*Sesion, error) {
	p.bifurcadas = append(p.bifurcadas, mensajeID)
	return &Sesion{ID: "fork", Nombre: "FORK:primera", Estado: EstadoInactiva}, nil
}

func (p *puertoFalso) SincronizarContexto(sesionID string) tea.Cmd {
	return nil
}

func (p *puertoFalso) Compactar(_ context.Context, sesionID string) (bool, error) {
	p.compactadas = append(p.compactadas, sesionID)
	return true, nil
}

func teclaRuna(s string) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)} }

// appDePrueba deja la app con geometría puesta y el reloj de la líder apagado:
// esperar el timeout no aporta nada a una prueba.
func appDePrueba(p Puerto) *App {
	app := Nueva(p)
	app.ancho, app.alto = 100, 24
	app.teclaRes.esperar = func(time.Duration) tea.Cmd { return nil }
	return app
}

// pulsar entrega una tecla y ejecuta lo que devuelva, como el bucle de Bubble Tea.
func pulsar(app *App, m tea.KeyMsg) {
	_, cmd := app.Update(m)
	correrCmds(app, cmd)
}

// correrCmds ejecuta un comando y entrega su mensaje al Update, en cadena. La
// cadena del latido del indicador se corta: aquí no hay nada que animar.
func correrCmds(app *App, cmd tea.Cmd) {
	for cmd != nil {
		msg := cmd()
		if msg == nil {
			return
		}
		if lote, ok := msg.(tea.BatchMsg); ok {
			for _, sub := range lote {
				correrCmds(app, sub)
			}
			return
		}
		if _, latido := msg.(frameMsg); latido {
			return
		}
		_, cmd = app.Update(msg)
	}
}

// abrirAtajo pulsa `ctrl+x` y la segunda tecla del atajo.
func abrirAtajo(app *App, segunda string) {
	pulsar(app, tea.KeyMsg{Type: tea.KeyCtrlX})
	pulsar(app, teclaRuna(segunda))
}

// El modal de modelos se abre con la secuencia líder, pide la lista al motor en
// ese momento y aplica con `enter` el modelo resaltado.
func TestElModalDeModelosSeAbreAplicaYDescarta(t *testing.T) {
	p := &puertoFalso{modelos: []ModeloLocal{
		{Nombre: "qwen2.5", Motor: "Ollama local"},
		{Nombre: "llama3.2", Motor: "Ollama local", SinHerramientas: true},
	}}
	app := appDePrueba(p)

	abrirAtajo(app, "m")
	if !p.pedido || app.modal != ModalModelos {
		t.Fatalf("`ctrl+x m` no abrió el modal de modelos: %v", app.modal)
	}
	if !strings.Contains(render.SinANSI(app.View()), "MODELOS") {
		t.Fatal("el modal no se pintó")
	}

	pulsar(app, tea.KeyMsg{Type: tea.KeyDown})
	pulsar(app, tea.KeyMsg{Type: tea.KeyEnter})
	if app.modal != SinModal {
		t.Fatal("`enter` no cerró el modal")
	}
	if p.modeloFijado != "llama3.2" || app.entrada.Modelo != "llama3.2" {
		t.Fatalf("el modelo no se aplicó: %q %q", p.modeloFijado, app.entrada.Modelo)
	}
	if !strings.Contains(render.SinANSI(app.chat.Hilo(80)), "no declara capacidad de herramientas") {
		t.Fatal("un modelo sin herramientas debería avisarse")
	}

	// `esc` cierra sin cambiar nada.
	abrirAtajo(app, "m")
	pulsar(app, tea.KeyMsg{Type: tea.KeyEsc})
	if app.modal != SinModal || app.modelo != "llama3.2" {
		t.Fatal("`esc` no descartó el modal de modelos")
	}
}

// Abrir un modal cierra el anterior: solo puede haber uno a la vez.
func TestSoloHayUnModalAbiertoALaVez(t *testing.T) {
	p := &puertoFalso{sesiones: []Sesion{{ID: "s1", Nombre: "primera", Estado: EstadoInactiva}}}
	app := appDePrueba(p)

	abrirAtajo(app, "l")
	abrirAtajo(app, "m")
	if app.modal != ModalModelos || app.sesiones.Abierto {
		t.Fatalf("los modales no son excluyentes: %v", app.modal)
	}
	pulsar(app, tea.KeyMsg{Type: tea.KeyCtrlP})
	if app.modal != ModalAtajos || app.modelos.Abierto {
		t.Fatalf("`ctrl+p` no abrió los atajos en lugar del otro: %v", app.modal)
	}
}

// Con un modal abierto, sus teclas no llegan a la vista de abajo: no se escribe
// en la caja, `ctrl+d` no pliega el panel y `tab` no cicla el agente. `ctrl+c`
// sigue saliendo.
func TestConUnModalAbiertoElTecladoEsSuyo(t *testing.T) {
	p := &puertoFalso{sesiones: []Sesion{{ID: "s1", Nombre: "primera", Estado: "inactiva"}}}
	app := appDePrueba(p)
	app.entrada.Escribir("texto a medio escribir")
	antes := app.entrada.Texto
	agenteAntes := app.agente

	abrirAtajo(app, "l")
	pulsar(app, teclaRuna("x"))
	pulsar(app, teclaRuna("y"))
	pulsar(app, tea.KeyMsg{Type: tea.KeyTab})
	pulsar(app, tea.KeyMsg{Type: tea.KeyCtrlD})

	if app.entrada.Texto != antes {
		t.Fatalf("el modal dejó escribir en la caja: %q", app.entrada.Texto)
	}
	if app.agente != agenteAntes {
		t.Fatalf("`tab` cicló el agente con un modal abierto: %q", app.agente)
	}
	if !app.panel.Abierto() {
		t.Fatal("`ctrl+d` plegó el panel con el modal de sesiones abierto")
	}
	if _, cmd := app.Update(tea.KeyMsg{Type: tea.KeyCtrlC}); cmd == nil {
		t.Fatal("`ctrl+c` debería salir con un modal abierto")
	}
	// Sin modal, `tab` sí cicla: lo que cambia es el foco, no la acción.
	app.cerrarModales()
	pulsar(app, tea.KeyMsg{Type: tea.KeyTab})
	if app.agente == agenteAntes {
		t.Fatal("sin modal, `tab` debería ciclar el agente")
	}
}

// Elegir una sesión con `enter` la deja activa sin detener nada, y borrarla pide
// confirmación si está trabajando.
func TestElModalDeSesionesEligeYConfirmaElBorrado(t *testing.T) {
	p := &puertoFalso{sesiones: []Sesion{
		{ID: "s1", Nombre: "primera", Estado: "inactiva"},
		{ID: "s2", Nombre: "segunda", Estado: EstadoTrabajando},
	}}
	app := appDePrueba(p)
	app.sesion = &Sesion{ID: "s1", Nombre: "primera", Estado: "inactiva"}

	abrirAtajo(app, "l")
	pulsar(app, tea.KeyMsg{Type: tea.KeyDown})
	pulsar(app, tea.KeyMsg{Type: tea.KeyCtrlD})
	if len(p.eliminadas) != 0 {
		t.Fatalf("borró sin confirmar: %v", p.eliminadas)
	}
	if !strings.Contains(render.SinANSI(app.View()), "¿seguro que deseas eliminar?") {
		t.Fatal("no se pidió confirmación")
	}
	pulsar(app, tea.KeyMsg{Type: tea.KeyCtrlD})
	if len(p.eliminadas) != 1 || p.eliminadas[0] != "s2" {
		t.Fatalf("la confirmación no expuso la eliminación: %v", p.eliminadas)
	}

	// `enter` entrega la sesión resaltada al `app`, que la abre.
	pulsar(app, tea.KeyMsg{Type: tea.KeyDown})
	pulsar(app, tea.KeyMsg{Type: tea.KeyEnter})
	if app.modal != SinModal || app.sesion == nil || app.sesion.ID != "s2" {
		t.Fatalf("la sesión elegida no quedó activa: %+v", app.sesion)
	}
	if !app.chat.HayTurno() {
		t.Fatal("una sesión que trabaja debería reanudar su contador")
	}
}

// Al retomar una sesión se pide y se pinta su historial guardado, para que la
// conversación no desaparezca al reabrir la carpeta (SPEC-SESIONES: «al reabrir
// una carpeta se listan sus sesiones y se puede retomar cualquiera con su
// historial»).
func TestRetomarUnaSesionPintaSuHistorial(t *testing.T) {
	p := &puertoFalso{
		sesiones: []Sesion{{ID: "s2", Nombre: "segunda", Estado: EstadoInactiva}},
		historial: []PiezaHistorial{
			&MensajeHistorial{Rol: "user", Texto: "hola viejo"},
			&MensajeHistorial{Rol: "agent", Texto: "buenas, viejo", Razonamiento: "pensaba"},
		},
	}
	app := appDePrueba(p)

	abrirAtajo(app, "l")
	pulsar(app, tea.KeyMsg{Type: tea.KeyEnter})

	if p.historialPedido != "s2" {
		t.Fatalf("no se pidió el historial de la sesión elegida: %q", p.historialPedido)
	}
	hilo := render.SinANSI(app.chat.Hilo(80))
	if !strings.Contains(hilo, "hola viejo") || !strings.Contains(hilo, "buenas, viejo") {
		t.Fatalf("el historial de la sesión no se pintó: %q", hilo)
	}
}

// El hilo guardado incluye las líneas del sistema —herramientas y sub-procesos—,
// que se repintan al retomar la sesión igual que se vieron en vivo: antes no se
// guardaban y desaparecían (DATA_FLOW §Creación, INTERFACES §1.1).
func TestRetomarUnaSesionRepintaSusLineas(t *testing.T) {
	p := &puertoFalso{
		sesiones: []Sesion{{ID: "s2", Nombre: "segunda", Estado: EstadoInactiva}},
		historial: []PiezaHistorial{
			&MensajeHistorial{Rol: "user", Texto: "hazlo"},
			&LineaHistorial{Tipo: "proceso", Texto: "[Sub Proceso] Diagnosticar"},
			&LineaHistorial{Tipo: "herramienta", Texto: "✓ LEER [AGENTS.md] · 93 líneas · 0.4 s"},
			&MensajeHistorial{Rol: "agent", Texto: "listo"},
		},
	}
	app := appDePrueba(p)

	abrirAtajo(app, "l")
	pulsar(app, tea.KeyMsg{Type: tea.KeyEnter})

	hilo := render.SinANSI(app.chat.Hilo(80))
	if !strings.Contains(hilo, "[Sub Proceso] Diagnosticar") {
		t.Fatalf("la línea del sub-proceso no se repintó: %q", hilo)
	}
	if !strings.Contains(hilo, "LEER [AGENTS.md]") {
		t.Fatalf("la línea de la herramienta no se repintó: %q", hilo)
	}
}

// `ctrl+x n` desde la vista principal crea una sesión nueva, la deja activa y
// vacía el hilo: la sesión recién creada no arrastra la conversación anterior
// (SPEC-SESIONES, SPEC-KEYBINDS, SPEC-INTERFAZ).
func TestElAtajoDeSesionNuevaCreaYActiva(t *testing.T) {
	p := &puertoFalso{sesiones: []Sesion{{ID: "s1", Nombre: "primera", Estado: EstadoInactiva}}}
	app := appDePrueba(p)
	app.vista = VistaPrincipal
	app.sesion = &Sesion{ID: "s1", Nombre: "primera", Estado: EstadoInactiva}
	app.chat.AñadirUsuario("algo previo")

	abrirAtajo(app, "n")

	if p.creadas != 1 {
		t.Fatalf("`ctrl+x n` no creó la sesión: %d", p.creadas)
	}
	if app.sesion == nil || app.sesion.ID != "nueva" {
		t.Fatalf("la sesión nueva no quedó activa: %+v", app.sesion)
	}
	if app.vista != VistaPrincipal {
		t.Fatal("la sesión nueva no deja la vista en la principal")
	}
	if !app.chat.Vacio() {
		t.Fatal("el hilo de la sesión anterior no se vació")
	}
}

// Con un modal abierto, `ctrl+x n` no crea nada: el atajo es de la vista
// principal y el modal captura su teclado.
func TestElAtajoDeSesionNuevaNoDisparaConUnModalAbierto(t *testing.T) {
	p := &puertoFalso{sesiones: []Sesion{{ID: "s1", Nombre: "primera", Estado: EstadoInactiva}}}
	app := appDePrueba(p)

	abrirAtajo(app, "l")
	abrirAtajo(app, "n")

	if p.creadas != 0 {
		t.Fatalf("`ctrl+x n` creó una sesión con un modal abierto: %d", p.creadas)
	}
}

// El modal de atajos es de solo lectura: `enter` no aplica nada y `ctrl+p` lo
// alterna.
func TestElModalDeAtajosNoAplicaNada(t *testing.T) {
	app := appDePrueba(nil)
	// La tabla entera —seis categorías y veintidós acciones— solo cabe en una
	// ventana alta; con una baja el modal recorta por diseño, así que la
	// comprobación del contenido se hace donde no hay recorte.
	app.alto = 40

	pulsar(app, tea.KeyMsg{Type: tea.KeyCtrlP})
	if app.modal != ModalAtajos {
		t.Fatalf("`ctrl+p` no abrió los atajos: %v", app.modal)
	}
	pantalla := render.SinANSI(app.View())
	for _, quiero := range []string{"ATAJOS", "General", "Modales", "ctrl+x m", "modal de modelos"} {
		if !strings.Contains(pantalla, quiero) {
			t.Errorf("la tabla de atajos no trae %q", quiero)
		}
	}
	pulsar(app, tea.KeyMsg{Type: tea.KeyEnter})
	if app.modal != ModalAtajos {
		t.Fatal("`enter` cerró el modal de atajos")
	}
	pulsar(app, tea.KeyMsg{Type: tea.KeyCtrlP})
	if app.modal != SinModal {
		t.Fatal("`ctrl+p` debería volver a cerrarlo")
	}
}

// Escribir `/` despliega la paleta encima de la entrada; lo escrito filtra,
// `tab` autocompleta y un espacio la retira para dejar paso a la petición.
func TestLaPaletaDeComandosSeDespliegaConLaBarra(t *testing.T) {
	app := appDePrueba(&puertoFalso{flujos: []FlujoVista{
		{Comando: "planificar"},
		{Comando: "resolver"},
	}})

	pulsar(app, teclaRuna("/"))
	if !app.paleta.Visible() {
		t.Fatal("escribir `/` no desplegó la paleta")
	}
	pantalla := render.SinANSI(app.View())
	if !strings.Contains(pantalla, "/flow:planificar") || !strings.Contains(pantalla, "/flow:resolver") {
		t.Fatalf("la paleta no lista los flujos:\n%s", pantalla)
	}

	for _, r := range "resol" {
		pulsar(app, teclaRuna(string(r)))
	}
	filtrados := app.paleta.Filtrados()
	if len(filtrados) != 1 || filtrados[0].Nombre != "flow:resolver" {
		t.Fatalf("el filtro no redujo la lista a `flow:resolver`: %+v", filtrados)
	}

	pulsar(app, tea.KeyMsg{Type: tea.KeyTab})
	if got := app.bienvenida.Texto; got != "/flow:resolver " {
		t.Fatalf("`tab` no autocompletó el comando: %q", got)
	}
	pulsar(app, teclaRuna(" "))
	if app.paleta.Visible() {
		t.Fatal("un espacio debería retirar la paleta")
	}
}

// Con un comando ya escrito, el primer `enter` lo ejecuta: la paleta no lo
// secuestra para autocompletar —eso es lo que hace `tab`— (SPEC-INTERFAZ §2).
func TestLaPaletaDejaEjecutarConEnter(t *testing.T) {
	p := &puertoFalso{}
	app := appDePrueba(p)
	app.sesion = &Sesion{ID: "activa", Nombre: "S", Estado: EstadoTrabajando}
	app.vista = VistaPrincipal

	for _, r := range "/compactar" {
		pulsar(app, teclaRuna(string(r)))
	}
	if !app.paleta.Visible() {
		t.Fatal("escribir `/compactar` no desplegó la paleta")
	}

	pulsar(app, tea.KeyMsg{Type: tea.KeyEnter})
	if len(p.compactadas) != 1 || p.compactadas[0] != "activa" {
		t.Fatalf("el primer `enter` debería ejecutar `/compactar`: %v", p.compactadas)
	}
	if app.paleta.Visible() {
		t.Fatal("ejecutar el comando debería retirar la paleta")
	}
	if got := app.entrada.Texto; got != "" {
		t.Fatalf("la entrada debería quedar vacía tras ejecutar: %q", got)
	}
}

// La paleta es un overlay real: al desplegarse no añade ni desplaza filas y no
// pisa el sidebar (la columna de la derecha queda idéntica). Es la regresión de
// SPEC-INTERFAZ §2: sobreponer sin empujar el logo ni el chat.
func TestLaPaletaNoEmpujaLaVistaNiCortaElSidebar(t *testing.T) {
	app := appEnPrincipal(100, 30)
	app.paleta.EstablecerMaxAlto(app.alto)
	app.paleta.EstablecerCatalogo(ComandosDeLaEntrada(nil))
	app.paleta.Mostrar()

	conPaleta := render.SinANSI(app.View())
	app.paleta.Ocultar()
	sinPaleta := render.SinANSI(app.View())

	lineasCon := strings.Split(conPaleta, "\n")
	lineasSin := strings.Split(sinPaleta, "\n")
	if len(lineasCon) != len(lineasSin) {
		t.Fatalf("la paleta cambió el alto de la vista: %d vs %d filas",
			len(lineasCon), len(lineasSin))
	}
	if !strings.Contains(conPaleta, "/undo") {
		t.Fatalf("la paleta no lista sus comandos:\n%s", conPaleta)
	}

	// El sidebar empieza tras la columna principal y sus dos espacios de separación.
	inicioSidebar := app.anchoColumna() + 2
	for i := range lineasSin {
		izq := []rune(lineasSin[i])
		der := []rune(lineasCon[i])
		desde := min(inicioSidebar, min(len(izq), len(der)))
		if string(izq[desde:]) != string(der[desde:]) {
			t.Fatalf("la paleta cortó el sidebar en la fila %d:\n antes: %q\n después: %q",
				i, string(izq[desde:]), string(der[desde:]))
		}
	}
}

// La paleta muestra como máximo 5 elementos a la vez: con más comandos, recorta
// la ventana visible y ofrece scroll (SPEC-INTERFAZ §2).
func TestLaPaletaMuestraComoMaximoCincoElementos(t *testing.T) {
	app := appDePrueba(&puertoFalso{flujos: []FlujoVista{
		{Comando: "uno"}, {Comando: "dos"}, {Comando: "tres"}, {Comando: "cuatro"},
		{Comando: "cinco"}, {Comando: "seis"}, {Comando: "siete"},
	}})
	pulsar(app, teclaRuna("/"))

	if got := app.paleta.MaxAlto(); got != 5 {
		t.Fatalf("el máximo de elementos visibles es %d, quiero 5", got)
	}
	lista := app.paleta.Filtrados()
	if len(lista) <= 5 {
		t.Fatalf("el catálogo de prueba debe superar 5 elementos, tiene %d", len(lista))
	}
	pantalla := render.SinANSI(app.View())
	visibles := 0
	for _, c := range lista {
		if strings.Contains(pantalla, "/"+c.Nombre) {
			visibles++
		}
	}
	if visibles != 5 {
		t.Fatalf("la paleta muestra %d elementos, quiero 5:\n%s", visibles, pantalla)
	}
}
