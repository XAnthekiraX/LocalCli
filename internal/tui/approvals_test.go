package tui

// Tests de T-F008: el panel global de aprobaciones. Una prueba, una regla
// (frontend/05-quality/TESTING.md §3). La resolución por líneas y el aviso de
// pendientes ya tenían pruebas de T-B014; aquí van las reglas nuevas de esta
// tarea: el panel se abre con Ctrl+A, se cierra sin parar nada, se resuelve
// con a/d línea a línea y las obsoletas no se mandan.
//
//	T-F008-02 → abrir y cerrar no emite ningún control de sesión
//	T-F008-03 → dos sesiones: dos líneas independientes
//	T-F008-04 → formato de línea documentado, con la opción por definir
//	T-F008-05 → a/d resuelven solo la línea seleccionada
//	T-F008-06 → resuelta sale; sesión terminada → obsoleta, no se manda

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"localcli/internal/session"
)

func nuevoConPanelDeAprobaciones(t *testing.T) (*App, *puertoStub) {
	t.Helper()
	a := Nuevo(&puertoStub{})
	a.Vista = VistaPrincipal
	a.Panel.SesionID = "s1"
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})
	return a, a.Puerto.(*puertoStub)
}

// --- T-F008-02 / T-F028-01: mostrar y enfocar no toca nada ----------------------

// Una aprobación pendiente se MUESTRA (para que la decisión se vea) pero no le
// quita el teclado al input; el foco se pide con Ctrl+A y se suelta igual. Ni
// mostrarlo ni enfocarlo emite ningún control de sesión.
func TestMostrarYEnfocarElPanelDeAprobacionesNoDetieneNada(t *testing.T) {
	a, p := nuevoConPanelDeAprobaciones(t)
	a.Aprobs.Fijar([]Aprobacion{{ID: "a1", Sesion: "s1", Descripcion: "crear archivo"}})
	if !a.Aprobs.Abierto {
		t.Fatal("una aprobación pendiente debe verse")
	}
	if a.Aprobs.Enfocado {
		t.Fatal("una aprobación pendiente no debe enfocar el teclado")
	}

	tecla(t, a, tea.KeyCtrlA)
	if !a.Aprobs.Enfocado {
		t.Fatal("ctrl+a enfoca el panel para decidir con a/d")
	}
	tecla(t, a, tea.KeyCtrlA)
	if a.Aprobs.Enfocado {
		t.Fatal("ctrl+a vuelve a soltar el foco")
	}
	if len(p.cancelado) != 0 || len(p.pausadas) != 0 || len(p.resueltas) != 0 {
		t.Errorf("mostrar y enfocar no emite ningún control: %v, %v, %v", p.cancelado, p.pausadas, p.resueltas)
	}
}

// --- T-F008-03: líneas independientes por sesión ---------------------------------

func TestDosSesionesEsperanComoDosLíneasIndependientes(t *testing.T) {
	a, _ := nuevoConPanelDeAprobaciones(t)

	pulsa(t, a, eventoMsg{Evento: Evento{
		Nombre: EventoPeticionAprobacion,
		Datos:  map[string]string{"aprobacion": "a1", "sesion": "s1", "descripcion": "crear archivo"},
	}})
	pulsa(t, a, eventoMsg{Evento: Evento{
		Nombre: EventoPeticionAprobacion,
		Datos:  map[string]string{"aprobacion": "a2", "sesion": "s2", "descripcion": "borrar carpeta"},
	}})
	if len(a.Aprobs.Items) != 2 {
		t.Fatalf("dos sesiones, dos líneas: %+v", a.Aprobs.Items)
	}
	if a.Aprobs.Items[0].Sesion == a.Aprobs.Items[1].Sesion {
		t.Error("las líneas son de sesiones distintas, independientes")
	}
}

// --- T-F008-04: el formato documentado -------------------------------------------

func TestElFormatoDeLíneaEsElDocumentado(t *testing.T) {
	a, _ := nuevoConPanelDeAprobaciones(t)
	a.Aprobs.Fijar([]Aprobacion{{ID: "a1", Sesion: "api", Descripcion: "crear archivo"}})

	lineas := a.Aprobs.Lineas()
	if len(lineas) != 1 {
		t.Fatalf("una línea: %v", lineas)
	}
	// SPEC-INTERFAZ-ATAJOS: `sesión | acción propuesta | aprobar | declinar`.
	// La tercera opción sigue por definir y no se reserva un hueco vacío. La
	// seleccionada lleva además su marca de cursor.
	quiere := "› api | crear archivo | aprobar | declinar"
	if lineas[0] != quiere {
		t.Errorf("línea = %q, quiero %q", lineas[0], quiere)
	}
}

// La línea va en corto: la sesión es su identificador abreviado y la carpeta del
// usuario se abrevia a `~`, para que quepa de un vistazo.
func TestLaLineaDeAprobacionVaEnCorto(t *testing.T) {
	largo := "24afd397-ccdd-45df-a8df-90bf3ad0974a"
	fila := FilaDe(Aprobacion{ID: "a1", Sesion: largo, Descripcion: "leer el archivo /home/x/foto.png"})
	if strings.Contains(fila, largo) {
		t.Errorf("el identificador de sesión va abreviado: %q", fila)
	}
	if !strings.HasPrefix(fila, "24afd397 | ") {
		t.Errorf("la línea empieza por la sesión corta: %q", fila)
	}
	home := rutaDelUsuario()
	if home != "" {
		larga := FilaDe(Aprobacion{ID: "a1", Sesion: "s1", Descripcion: "leer el archivo " + home + "/foto.png"})
		if !strings.Contains(larga, "~/foto.png") {
			t.Errorf("la carpeta del usuario se abrevia a ~: %q", larga)
		}
	}
}

// Con una decisión esperando, el indicador lo dice: el turno no avanza hasta que
// se resuelva, y «Pensando» haría creer que sigue trabajando.
func TestElIndicadorDiceQueEsperaTuPermiso(t *testing.T) {
	a := Nuevo(&puertoStub{})
	a.Vista = VistaPrincipal
	a.Panel.SesionID = "s1"
	pulsa(t, a, tea.WindowSizeMsg{Width: 80, Height: 24})
	a.Chat.AñadirUsuario("describe esto") // el turno queda vivo
	pulsa(t, a, eventoMsg{Evento: Evento{Nombre: EventoPeticionAprobacion, Datos: map[string]string{
		"aprobacion": "a1", "sesion": "s1", "descripcion": "leer foto.png",
	}}})
	if v := sinEstilo(a.View()); !strings.Contains(v, "Esperando tu permiso") {
		t.Errorf("con una aprobación pendiente el indicador lo dice:\n%s", v)
	}
}

// --- T-F008-05: a/d resuelven solo su línea ---------------------------------------

func TestAprobarYDeclinarResuelvenSoloLaLíneaSeleccionada(t *testing.T) {
	a, p := nuevoConPanelDeAprobaciones(t)
	a.Aprobs.Fijar([]Aprobacion{
		{ID: "a1", Sesion: "s1", Descripcion: "crear archivo"},
		{ID: "a2", Sesion: "s2", Descripcion: "borrar carpeta"},
	})

	// Con el foco en el panel (Ctrl+A), a/d deciden la línea seleccionada.
	tecla(t, a, tea.KeyCtrlA)
	pulsa(t, a, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if len(p.resueltas) != 1 || p.resueltas[0] != "a1:aprobar" {
		t.Fatalf("a aprueba la línea seleccionada: %v", p.resueltas)
	}
	if len(a.Aprobs.Items) != 1 || a.Aprobs.Items[0].ID != "a2" {
		t.Fatalf("solo sale la línea resuelta: %+v", a.Aprobs.Items)
	}

	pulsa(t, a, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if len(p.resueltas) != 2 || p.resueltas[1] != "a2:declinar" {
		t.Fatalf("d declina la siguiente línea: %v", p.resueltas)
	}
	if len(a.Aprobs.Items) != 0 {
		t.Errorf("no queda nada pendiente: %+v", a.Aprobs.Items)
	}
}

// --- T-F028-01: la aprobación no secuestra el teclado ----------------------------

// Una petición de aprobación muestra la decisión SIN bloquear el input: se ven
// la línea y sus opciones, se sigue escribiendo y el teclado no decide nada.
func TestUnaAprobaciónPendienteNoBloqueaElInput(t *testing.T) {
	a, p := nuevoConPanelDeAprobaciones(t)
	pulsa(t, a, eventoMsg{Evento: Evento{
		Nombre: EventoPeticionAprobacion,
		Datos:  map[string]string{"aprobacion": "a1", "sesion": "s1", "descripcion": "crear archivo"},
	}})
	if !a.Aprobs.Abierto || a.Aprobs.Enfocado {
		t.Fatalf("la decisión se muestra sin enfocar el teclado: abierto=%v enfocado=%v",
			a.Aprobs.Abierto, a.Aprobs.Enfocado)
	}
	// La decisión que faltaba: la línea con sus opciones está a la vista.
	if v := sinEstilo(a.View()); !strings.Contains(v, "s1 | crear archivo | aprobar | declinar") {
		t.Fatalf("la decisión debe verse con sus opciones:\n%s", v)
	}
	escribe(t, a, "sigo escribiendo")
	if a.Entrada.Texto() != "sigo escribiendo" {
		t.Errorf("el input no puede quedar bloqueado: %q", a.Entrada.Texto())
	}
	if len(p.resueltas) != 0 {
		t.Errorf("escribir no resuelve ninguna aprobación: %v", p.resueltas)
	}
	// Con el panel de datos cerrado, el aviso de pendientes también se ve.
	a.Panel.Abierto = false
	if !strings.Contains(sinEstilo(a.View()), "1 aprobación esperando tu decisión") {
		t.Errorf("debe verse el aviso de pendientes:\n%s", sinEstilo(a.View()))
	}
}

// --- T-F028-02: el ratón decide sobre la opción pulsada --------------------------

// celdaDeOpción localiza en el marco pintado la celda (columna, fila) donde
// empieza `opcion` dentro de la fila que contiene `marca`.
func celdaDeOpción(t *testing.T, a *App, marca, opcion string) (int, int) {
	t.Helper()
	lineas := strings.Split(sinEstilo(a.View()), "\n")
	for y, l := range lineas {
		if !strings.Contains(l, marca) {
			continue
		}
		if i := strings.Index(l, opcion); i >= 0 {
			return len([]rune(l[:i])), y
		}
	}
	t.Fatalf("no encontré %q en una fila con %q:\n%s", opcion, marca, sinEstilo(a.View()))
	return 0, 0
}

// clic manda la pulsación y el soltado del botón izquierdo en una celda, como
// llega un clic real del ratón.
func clic(t *testing.T, a *App, x, y int) {
	t.Helper()
	pulsa(t, a, tea.MouseMsg{X: x, Y: y, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
	pulsa(t, a, tea.MouseMsg{X: x, Y: y, Button: tea.MouseButtonLeft, Action: tea.MouseActionRelease})
}

func TestClicEnAprobarResuelveLaFilaPulsada(t *testing.T) {
	a, p := nuevoConPanelDeAprobaciones(t)
	a.Aprobs.Fijar([]Aprobacion{
		{ID: "a1", Sesion: "s1", Descripcion: "crear archivo"},
		{ID: "a2", Sesion: "s2", Descripcion: "borrar carpeta"},
	})
	// El ratón decide sin necesidad de enfocar el panel: el input conserva el teclado.
	if a.Aprobs.Enfocado {
		t.Fatal("el ratón no necesita que el panel tenga el foco")
	}

	x, y := celdaDeOpción(t, a, "s2 | borrar carpeta", "aprobar")
	clic(t, a, x, y)

	if len(p.resueltas) != 1 || p.resueltas[0] != "a2:aprobar" {
		t.Fatalf("el clic en «aprobar» decide esa fila: %v", p.resueltas)
	}
	if len(a.Aprobs.Items) != 1 || a.Aprobs.Items[0].ID != "a1" {
		t.Fatalf("solo sale la fila pulsada: %+v", a.Aprobs.Items)
	}
}

func TestClicEnDeclinarDeclinaLaAprobacion(t *testing.T) {
	a, p := nuevoConPanelDeAprobaciones(t)
	a.Aprobs.Fijar([]Aprobacion{{ID: "a1", Sesion: "s1", Descripcion: "crear archivo"}})

	x, y := celdaDeOpción(t, a, "s1 | crear archivo", "declinar")
	clic(t, a, x, y)

	if len(p.resueltas) != 1 || p.resueltas[0] != "a1:declinar" {
		t.Fatalf("el clic en «declinar» declina esa línea: %v", p.resueltas)
	}
	// Al resolver la última, el panel se cierra solo.
	if a.Aprobs.Abierto {
		t.Error("sin pendientes, el panel se cierra")
	}
}

func TestClicFueraDeLasOpcionesNoResuelve(t *testing.T) {
	a, p := nuevoConPanelDeAprobaciones(t)
	a.Aprobs.Fijar([]Aprobacion{{ID: "a1", Sesion: "s1", Descripcion: "crear archivo"}})

	// Un clic sobre la descripción (a la izquierda de las opciones) no decide.
	x, y := celdaDeOpción(t, a, "s1 | crear archivo", "crear archivo")
	clic(t, a, x, y)

	if len(p.resueltas) != 0 {
		t.Errorf("un clic fuera de aprobar/declinar no decide nada: %v", p.resueltas)
	}
}

func TestLaAprobaciónResueltaSaleDelPanel(t *testing.T) {
	a, _ := nuevoConPanelDeAprobaciones(t)
	a.Aprobs.Fijar([]Aprobacion{
		{ID: "a1", Sesion: "s1", Descripcion: "crear archivo"},
		{ID: "a2", Sesion: "s2", Descripcion: "borrar carpeta"},
	})
	pulsa(t, a, eventoMsg{Evento: Evento{
		Nombre: EventoAprobacionResuelta,
		Datos:  map[string]string{"aprobacion": "a1"},
	}})
	if len(a.Aprobs.Items) != 1 || a.Aprobs.Items[0].ID != "a2" {
		t.Errorf("solo queda la línea no resuelta: %+v", a.Aprobs.Items)
	}
}

func TestLaSesiónTerminadaMarcaSuLíneaComoObsoleta(t *testing.T) {
	a, p := nuevoConPanelDeAprobaciones(t)
	a.Aprobs.Fijar([]Aprobacion{{ID: "a1", Sesion: "s1", Descripcion: "crear archivo"}})

	pulsa(t, a, eventoMsg{Evento: Evento{
		Nombre: session.EventoEstadoSesion,
		Datos:  map[string]string{"sesion": "s1", "estado": session.EstadoTerminada},
	}})
	if !a.Aprobs.Items[0].Obsoleta {
		t.Fatal("la sesión terminó mientras esperaba: la línea se marca obsoleta")
	}
	// Abrir el panel para ver la línea marcada; sobre una obsoleta no se manda
	// ninguna decisión.
	tecla(t, a, tea.KeyCtrlA)
	if !strings.Contains(sinEstilo(a.View()), "obsoleta") {
		t.Error("la línea obsoleta sigue visible, marcada como tal")
	}
	pulsa(t, a, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if len(p.resueltas) != 0 {
		t.Errorf("lo obsoleto no se manda a la sesión: %v", p.resueltas)
	}
}
