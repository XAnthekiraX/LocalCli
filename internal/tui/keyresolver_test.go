package tui

// Tests del KeyResolver (T-F012-02/03/04/05, TESTING.md §2: el resolver se
// prueba sin terminal real, con un reloj inyectado y sin esperas fijas).
// Una prueba, una regla: la máquina de estados de la líder, su timeout, la
// resolución por contexto y la validación del mapa.

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// --- T-F012-02: la máquina NORMAL/LEADER ---------------------------------------

func TestLaLiderSeguidaDeUnaTeclaResuelveLaAccion(t *testing.T) {
	for _, caso := range []struct {
		segunda string
		accion  Accion
	}{
		{"m", AccionModalModelos},
		{"l", AccionSelector},
	} {
		r := NuevoKeyResolver(KeymapPorDefecto())
		if accion, _ := r.Resolver(teclaConNombre("ctrl+x"), ContextoVista); accion != AccionNinguna {
			t.Errorf("la líder sola no emite acción: %d", accion)
		}
		accion, _ := r.Resolver(teclaConNombre(caso.segunda), ContextoVista)
		if accion != caso.accion {
			t.Errorf("<leader>%s resuelve la acción de su binding: %d", caso.segunda, accion)
		}
		if r.Estado != EstadoNormal {
			t.Errorf("<leader>%s deja el resolver en NORMAL", caso.segunda)
		}
	}
}

func TestLaLiderSolaEsperaSinEmitir(t *testing.T) {
	r := NuevoKeyResolver(KeymapPorDefecto())
	accion, cmd := r.Resolver(tea.KeyMsg{Type: tea.KeyCtrlX}, ContextoVista)
	if accion != AccionNinguna {
		t.Errorf("la líder por sí sola no abre nada: %d", accion)
	}
	if r.Estado != EstadoLider || !r.EsperandoLeader() {
		t.Fatal("la líder deja el resolver esperando la segunda tecla")
	}
	if cmd == nil {
		t.Error("esperar la segunda tecla arma su temporizador")
	}
}

func TestUnaSegundaTeclaSinBindingDescartaLaSecuencia(t *testing.T) {
	r := NuevoKeyResolver(KeymapPorDefecto())
	r.Resolver(tea.KeyMsg{Type: tea.KeyCtrlX}, ContextoVista)
	accion, _ := r.Resolver(teclaConNombre("z"), ContextoVista)
	if accion != AccionNinguna {
		t.Errorf("una secuencia inexistente no emite acción: %d", accion)
	}
	if r.Estado != EstadoNormal {
		t.Error("la secuencia inválida vuelve a NORMAL")
	}
}

func TestLaLiderPulsadaDosVecesReiniciaLaEspera(t *testing.T) {
	r := NuevoKeyResolver(KeymapPorDefecto())
	r.Resolver(tea.KeyMsg{Type: tea.KeyCtrlX}, ContextoVista)
	if accion, _ := r.Resolver(tea.KeyMsg{Type: tea.KeyCtrlX}, ContextoVista); accion != AccionNinguna {
		t.Errorf("la segunda pulsación de la líder no emite: %d", accion)
	}
	if r.Estado != EstadoLider {
		t.Error("la líder repetida reinicia la espera desde cero")
	}
	if accion, _ := r.Resolver(teclaConNombre("l"), ContextoVista); accion != AccionSelector {
		t.Errorf("tras reiniciar, la secuencia sigue viva: %d", accion)
	}
}

func TestEscCancelaLaEsperaDeLider(t *testing.T) {
	r := NuevoKeyResolver(KeymapPorDefecto())
	r.Resolver(tea.KeyMsg{Type: tea.KeyCtrlX}, ContextoVista)
	accion, _ := r.Resolver(tea.KeyMsg{Type: tea.KeyEsc}, ContextoVista)
	if accion != AccionNinguna {
		t.Errorf("esc cancela sin emitir acción: %d", accion)
	}
	if r.Estado != EstadoNormal || r.EsperandoLeader() {
		t.Error("esc devuelve el resolver a NORMAL")
	}
}

// --- T-F012-03: el timeout de la líder ------------------------------------------

func TestElTimeoutDevuelveElResolverANormal(t *testing.T) {
	a := Nuevo(&puertoStub{})
	a.Vista = VistaPrincipal
	// Reloj inyectado: el temporizador expira ya, sin esperas fijas.
	a.TeclaRes.esperar = func(time.Duration) tea.Cmd {
		return func() tea.Msg { return LiderExpiradoMsg{} }
	}

	cmd := pulsa(t, a, tea.KeyMsg{Type: tea.KeyCtrlX})
	if !a.TeclaRes.EsperandoLeader() {
		t.Fatal("la líder deja el resolver esperando")
	}
	if !strings.Contains(sinEstilo(a.View()), "lider") {
		t.Error("mientras espera, la barra de estado muestra el indicador")
	}

	if msg := cmd(); msg == nil {
		t.Fatal("el temporizador produce su mensaje de expiración")
	} else {
		pulsa(t, a, msg)
	}
	if a.TeclaRes.EsperandoLeader() || a.TeclaRes.Estado != EstadoNormal {
		t.Error("pasado el timeout el resolver vuelve a NORMAL")
	}
	if strings.Contains(sinEstilo(a.View()), "lider") {
		t.Error("pasado el timeout el indicador desaparece")
	}
}

func TestElTimeoutConfigurableEsElDelMapa(t *testing.T) {
	km, err := NuevoKeymap(LíderPorDefecto, 750, MapasPorDefecto())
	if err != nil {
		t.Fatal(err)
	}
	var waited time.Duration
	r := NuevoKeyResolver(km)
	r.esperar = func(d time.Duration) tea.Cmd {
		waited = d
		return func() tea.Msg { return LiderExpiradoMsg{} }
	}
	r.Resolver(tea.KeyMsg{Type: tea.KeyCtrlX}, ContextoVista)
	if waited != 750*time.Millisecond {
		t.Errorf("el temporizador usa el leader_timeout_ms del mapa: %v", waited)
	}
}

// --- T-F012-04: la resolución por contexto --------------------------------------

func TestConUnModalAbiertoSoloRespondenSusTeclasYLasGlobales(t *testing.T) {
	r := NuevoKeyResolver(KeymapPorDefecto())
	for _, caso := range []struct {
		tecla  string
		accion Accion
	}{
		{"up", AccionSubir},
		{"down", AccionBajar},
		{"esc", AccionCerrarSelector},
		{"enter", AccionEnviar}, // con un modal abierto, enter aplica lo resaltado
		{"ctrl+c", AccionSalir}, // app_exit es global, también con modales
		{"ctrl+p", AccionAyuda},
	} {
		accion, _ := r.Resolver(teclaConNombre(caso.tecla), ContextoModal)
		if accion != caso.accion {
			t.Errorf("con un modal abierto %s resuelve %d", caso.tecla, accion)
		}
	}
	// Las teclas de la vista de abajo no llegan: ni los toggles ni el tab que
	// cicla el agente. `ctrl+d` resuelve dentro del modal que lo reclama: en el
	// de sesiones borra la sesión resaltada y en el de motores, el motor
	// (SPEC-KEYBINDS §Reglas: el duplicado solo se rechaza dentro del mismo
	// ámbito). En un modal sin teclas propias no resuelve nada.
	if accion, _ := r.Resolver(teclaConNombre("ctrl+d"), ContextoModalSesiones); accion != AccionEliminarSesion {
		t.Errorf("en el modal de sesiones ctrl+d elimina la sesión resaltada: %d", accion)
	}
	// En el modal de motores, eliminar es `<leader>d`; `ctrl+d` ya no es suya.
	if accion, _ := r.Resolver(teclaConNombre("ctrl+d"), ContextoModalMotores); accion != AccionNinguna {
		t.Errorf("en el modal de motores ctrl+d ya no elimina (es <leader>d): %d", accion)
	}
	if accion, _ := r.Resolver(teclaConNombre("ctrl+d"), ContextoModal); accion != AccionNinguna {
		t.Errorf("en un modal sin teclas propias ctrl+d no resuelve: %d", accion)
	}
	// En el modal de motores, añadir y editar son secuencias con líder: la
	// letra suelta no basta (SPEC-KEYBINDS §Tecla líder).
	if accion, _ := r.Resolver(teclaConNombre("n"), ContextoModalMotores); accion != AccionNinguna {
		t.Errorf("en el modal de motores la letra n sola no da de alta: %d", accion)
	}
	r.Resolver(teclaConNombre("ctrl+x"), ContextoModalMotores)
	if accion, _ := r.Resolver(teclaConNombre("a"), ContextoModalMotores); accion != AccionMotorNuevo {
		t.Errorf("<leader>a en el modal de motores da de alta: %d", accion)
	}
	r.Resolver(teclaConNombre("ctrl+x"), ContextoModalMotores)
	if accion, _ := r.Resolver(teclaConNombre("e"), ContextoModalMotores); accion != AccionMotorEditar {
		t.Errorf("<leader>e en el modal de motores edita: %d", accion)
	}
	if accion, _ := r.Resolver(teclaConNombre("n"), ContextoModalSesiones); accion != AccionNinguna {
		t.Errorf("en el modal de sesiones n no da de alta motores: %d", accion)
	}
	for _, tecla := range []string{"ctrl+r", "ctrl+a", "tab"} {
		if accion, _ := r.Resolver(teclaConNombre(tecla), ContextoModal); accion != AccionNinguna {
			t.Errorf("con un modal abierto %s no debe resolver: %d", tecla, accion)
		}
	}
}

func TestConElInputEnfocadoLasLetrasNoDisparanAcciones(t *testing.T) {
	r := NuevoKeyResolver(KeymapPorDefecto())
	if accion, _ := r.Resolver(teclaConNombre("m"), ContextoInput); accion != AccionNinguna {
		t.Errorf("escribir una letra no activa acciones: %d", accion)
	}
	if accion, _ := r.Resolver(teclaConNombre("d"), ContextoInput); accion != AccionNinguna {
		t.Errorf("escribir una letra no aprueba nada: %d", accion)
	}
	// Los atajos con modificador siguen llegando al mapa: es la razón de que
	// los de fábrica no usen letras sueltas.
	if accion, _ := r.Resolver(teclaConNombre("ctrl+p"), ContextoInput); accion != AccionAyuda {
		t.Errorf("ctrl+p sí resuelve escribiendo: %d", accion)
	}
}

func TestLaLiderFuncionaDesdeCualquierContexto(t *testing.T) {
	for _, ctx := range []Contexto{ContextoVista, ContextoInput, ContextoModal, ContextoModalSesiones, ContextoModalMotores, ContextoAprobaciones} {
		r := NuevoKeyResolver(KeymapPorDefecto())
		r.Resolver(tea.KeyMsg{Type: tea.KeyCtrlX}, ctx)
		accion, _ := r.Resolver(teclaConNombre("m"), ctx)
		if accion != AccionModalModelos {
			t.Errorf("el contexto %d deja trabajar la líder: %d", ctx, accion)
		}
	}
}

func TestConElPanelDeAprobacionesEnfocadoSusTeclasResuelven(t *testing.T) {
	r := NuevoKeyResolver(KeymapPorDefecto())
	// El panel de aprobaciones no es un modal: sus toggles de vista siguen
	// vivos, y a/d resuelven la línea seleccionada. Su contexto es el que tiene
	// cuando está enfocado (Ctrl+A); visible sin foco, el input manda.
	if accion, _ := r.Resolver(teclaConNombre("ctrl+a"), ContextoAprobaciones); accion != AccionAprobaciones {
		t.Errorf("ctrl+a con el panel enfocado suelta el foco: %d", accion)
	}
	if accion, _ := r.Resolver(teclaConNombre("a"), ContextoAprobaciones); accion != AccionAprobar {
		t.Errorf("a con el panel abierto aprueba: %d", accion)
	}
	if accion, _ := r.Resolver(teclaConNombre("d"), ContextoAprobaciones); accion != AccionDeclinar {
		t.Errorf("d con el panel abierto declina: %d", accion)
	}
}

func TestLasFlechasDeLaVistaNoMuevenUnaLista(t *testing.T) {
	r := NuevoKeyResolver(KeymapPorDefecto())
	// Arriba/bajo en la vista principal recorren el historial del chat, no la
	// lista de un modal (que ni está abierto): la misma tecla tiene otro
	// significado por contexto.
	if accion, _ := r.Resolver(teclaConNombre("up"), ContextoVista); accion != AccionChatSubir {
		t.Errorf("up en la vista sube por el chat: %d", accion)
	}
	if accion, _ := r.Resolver(teclaConNombre("down"), ContextoVista); accion != AccionChatBajar {
		t.Errorf("down en la vista baja por el chat: %d", accion)
	}
	// Con un modal abierto, en cambio, up/down navegan su lista.
	if accion, _ := r.Resolver(teclaConNombre("up"), ContextoModal); accion != AccionSubir {
		t.Errorf("con un modal abierto up navega la lista: %d", accion)
	}
}

// --- T-F012-05: la validación del mapa ------------------------------------------

func TestUnaAccionAdmiteVariosAtajos(t *testing.T) {
	km, err := NuevoKeymap(LíderPorDefecto, TimeoutPorDefectoMs, map[Accion][]string{
		AccionSalir:    {"ctrl+c", "<leader>q"},
		AccionAprobar:  {},
		AccionDeclinar: {},
	})
	if err != nil {
		t.Fatalf("una acción con dos literales es válida: %v", err)
	}
	if accion, ok := AccionDe(km.Entradas(), "ctrl+c"); !ok || accion != AccionSalir {
		t.Errorf("el atajo directo se conserva: %d, %v", accion, ok)
	}
	r := NuevoKeyResolver(km)
	if accion, _ := r.Resolver(tea.KeyMsg{Type: tea.KeyCtrlX}, ContextoVista); accion != AccionNinguna {
		t.Fatalf("la líder sola no emite: %d", accion)
	}
	if accion, _ := r.Resolver(teclaConNombre("q"), ContextoVista); accion != AccionSalir {
		t.Errorf("la secuencia con líder también dispara la acción: %d", accion)
	}
}

func TestUnaListaVaciaDeshabilitaLaAccion(t *testing.T) {
	km, err := NuevoKeymap(LíderPorDefecto, TimeoutPorDefectoMs, map[Accion][]string{
		AccionPanel: {},
	})
	if err != nil {
		t.Fatalf("una lista vacía no es un error: %v", err)
	}
	if accion, ok := AccionDe(km.Entradas(), "ctrl+d"); ok || accion != AccionNinguna {
		t.Errorf("sin literales la acción no resuelve nada: %d, %v", accion, ok)
	}
	if accion, _ := NuevoKeyResolver(km).Resolver(teclaConNombre("ctrl+d"), ContextoVista); accion != AccionNinguna {
		t.Errorf("la acción deshabilitada no se dispara: %d", accion)
	}
}

func TestElDuplicadoSeRechazaConMensajeClaro(t *testing.T) {
	_, err := NuevoKeymap(LíderPorDefecto, TimeoutPorDefectoMs, map[Accion][]string{
		AccionPanel: {"ctrl+o"},
		AccionSalir: {"ctrl+o"},
	})
	if err == nil {
		t.Fatal("un literal en dos acciones se rechaza")
	}
	if !strings.Contains(err.Error(), "ctrl+o") {
		t.Errorf("el error dice qué literal choca: %v", err)
	}
}

// El mismo literal en dos ÁMBITOS distintos no choca: `ctrl+d` es panel en la
// vista y eliminar en el modal de sesiones (SPEC-KEYBINDS §Reglas de negocio:
// "en un mismo contexto"). El ámbito global sí choca con cualquiera, porque
// está presente en todos los contextos.
func TestElMismoLiteralEnDosContextosSePermite(t *testing.T) {
	if _, err := NuevoKeymap(LíderPorDefecto, TimeoutPorDefectoMs, MapasPorDefecto()); err != nil {
		t.Fatalf("el mapa de fábrica debe ser válido (ctrl+d en dos contextos): %v", err)
	}
	_, err := NuevoKeymap(LíderPorDefecto, TimeoutPorDefectoMs, map[Accion][]string{
		AccionPanel:        {"ctrl+d"},
		AccionRazonamiento: {"ctrl+d"},
	})
	if err == nil {
		t.Fatal("el mismo literal en dos acciones del mismo contexto se rechaza")
	}
	if !strings.Contains(err.Error(), "mismo contexto") {
		t.Errorf("el error dice que el choque es de contexto: %v", err)
	}
}

func TestElTimeoutFueraDeRangoSeRechaza(t *testing.T) {
	for _, ms := range []int{TimeoutMínimoMs - 1, 0 - 1, TimeoutMáximoMs + 1} {
		if _, err := NuevoKeymap(LíderPorDefecto, ms, MapasPorDefecto()); err == nil {
			t.Errorf("%d ms está fuera del rango 500–10000", ms)
		}
	}
	for _, ms := range []int{TimeoutMínimoMs, TimeoutPorDefectoMs, TimeoutMáximoMs} {
		if _, err := NuevoKeymap(LíderPorDefecto, ms, MapasPorDefecto()); err != nil {
			t.Errorf("%d ms es un timeout válido: %v", ms, err)
		}
	}
}

func TestUnLiteralVacioSeRechaza(t *testing.T) {
	if _, err := NuevoKeymap(LíderPorDefecto, TimeoutPorDefectoMs, map[Accion][]string{
		AccionPanel: {"  "},
	}); err == nil {
		t.Error("un literal vacío se rechaza al cargar el mapa")
	}
}

func TestElPrefacioLeaderDesconocidoSeRechaza(t *testing.T) {
	if _, err := NuevoKeymap(LíderPorDefecto, TimeoutPorDefectoMs, map[Accion][]string{
		AccionPanel: {"<capa>d"},
	}); err == nil {
		t.Error("un prefijo que no es <leader> se rechaza, no se adivina")
	}
}

// --- T-F044-03: las cuatro acciones de motor --------------------------------

// TestElKeymapResuelveLasCuatroAccionesDeMotor — motor_picker con la letra
// líder `i` (global) y motor_new/motor_edit/motor_delete dentro del modal de
// motores, con sus ids documentados en SPEC-KEYBINDS §Tabla. Las cuatro están
// en el mapa de fábrica y cada una resuelve donde le corresponde (SPEC-KEYBINDS
// §Resolución por contexto).
func TestElKeymapResuelveLasCuatroAccionesDeMotor(t *testing.T) {
	// Los ids que SPEC-KEYBINDS §Tabla pone a cada acción siguen atados a ellas.
	for accion, id := range map[Accion]string{
		AccionModalMotores:  "motor_picker",
		AccionMotorNuevo:    "motor_new",
		AccionMotorEditar:   "motor_edit",
		AccionEliminarMotor: "motor_delete",
	} {
		if got := nombreDeAccion(accion); got != id {
			t.Errorf("la acción %d se llama %q, quiero %q", accion, got, id)
		}
		encontrada := false
		for _, at := range AtajosPorDefecto() {
			if at.Accion == accion {
				encontrada = true
				break
			}
		}
		if !encontrada {
			t.Errorf("la acción %q falta del mapa de fábrica", id)
		}
	}

	r := NuevoKeyResolver(KeymapPorDefecto())
	// `<leader>i` abre el modal desde la vista, como el resto de modales.
	if accion, _ := r.Resolver(teclaConNombre("ctrl+x"), ContextoVista); accion != AccionNinguna {
		t.Fatalf("la líder sola no emite: %d", accion)
	}
	if accion, _ := r.Resolver(teclaConNombre("i"), ContextoVista); accion != AccionModalMotores {
		t.Errorf("<leader>i resuelve motor_picker: %d", accion)
	}
	// Dentro del modal: `<leader>a`, `<leader>e` y `<leader>d` son suyas.
	for _, caso := range []struct {
		segunda string
		accion  Accion
	}{
		{"a", AccionMotorNuevo},
		{"e", AccionMotorEditar},
		{"d", AccionEliminarMotor},
	} {
		r.Resolver(teclaConNombre("ctrl+x"), ContextoModalMotores)
		accion, _ := r.Resolver(teclaConNombre(caso.segunda), ContextoModalMotores)
		if accion != caso.accion {
			t.Errorf("<leader>%s en el modal de motores resuelve %d", caso.segunda, accion)
		}
	}
	// Y no se filtran fuera del modal: `<leader>a` desde la vista no da de alta.
	r.Resolver(teclaConNombre("ctrl+x"), ContextoVista)
	if accion, _ := r.Resolver(teclaConNombre("a"), ContextoVista); accion != AccionNinguna {
		t.Errorf("<leader>a fuera del modal no da de alta: %d", accion)
	}
	r.Resolver(teclaConNombre("ctrl+x"), ContextoModalSesiones)
	if accion, _ := r.Resolver(teclaConNombre("a"), ContextoModalSesiones); accion != AccionNinguna {
		t.Errorf("<leader>a en el modal de sesiones no da de alta motores: %d", accion)
	}
	// Y la líder sigue viva dentro del modal (escape hatch): `<leader>i` cierra.
	if accion, _ := r.Resolver(teclaConNombre("ctrl+x"), ContextoModalMotores); accion != AccionNinguna {
		t.Fatalf("la líder entra en espera también con el modal: %d", accion)
	}
	if accion, _ := r.Resolver(teclaConNombre("i"), ContextoModalMotores); accion != AccionModalMotores {
		t.Errorf("<leader>i resuelve desde dentro del modal: %d", accion)
	}
}
