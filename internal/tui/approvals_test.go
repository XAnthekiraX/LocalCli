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

// --- T-F008-02: abrir y cerrar no toca nada ------------------------------------

func TestAbrirYCerrarElPanelDeAprobacionesNoDetieneNada(t *testing.T) {
	a, p := nuevoConPanelDeAprobaciones(t)
	a.Aprobs.Fijar([]Aprobacion{{ID: "a1", Sesion: "s1", Descripcion: "crear archivo"}})
	a.Aprobs.Abierto = false // Fijar lo abre; lo cerramos para probar el atajo

	tecla(t, a, tea.KeyCtrlA)
	if !a.Aprobs.Abierto {
		t.Fatal("ctrl+a abre el panel de aprobaciones")
	}
	tecla(t, a, tea.KeyCtrlA)
	if a.Aprobs.Abierto {
		t.Fatal("ctrl+a lo cierra otra vez")
	}
	if len(p.cancelado) != 0 || len(p.pausadas) != 0 || len(p.resueltas) != 0 {
		t.Errorf("abrir y cerrar no emite ningún control: %v, %v, %v", p.cancelado, p.pausadas, p.resueltas)
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
	// SPEC-INTERFAZ-ATAJOS: `sesión | acción propuesta | aprobar | declinar |
	// <opción por definir>`. La tercera opción aún no está definida: sale como
	// un hueco, no se inventa. La seleccionada lleva además su marca de cursor.
	quiere := "› api | crear archivo | aprobar | declinar | —"
	if lineas[0] != quiere {
		t.Errorf("línea = %q, quiero %q", lineas[0], quiere)
	}
}

// --- T-F008-05: a/d resuelven solo su línea ---------------------------------------

func TestAprobarYDeclinarResuelvenSoloLaLíneaSeleccionada(t *testing.T) {
	a, p := nuevoConPanelDeAprobaciones(t)
	a.Aprobs.Fijar([]Aprobacion{
		{ID: "a1", Sesion: "s1", Descripcion: "crear archivo"},
		{ID: "a2", Sesion: "s2", Descripcion: "borrar carpeta"},
	})

	// Fijar deja el panel abierto: no hace falta volver a abrirlo.
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

// --- T-F008-06: resuelta sale; sesión terminada → obsoleta -------------------------

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
	if !strings.Contains(sinEstilo(a.View()), "obsoleta") {
		t.Error("la línea obsoleta sigue visible, marcada como tal")
	}
	// Y sobre una obsoleta no se manda ninguna decisión.
	tecla(t, a, tea.KeyCtrlA)
	pulsa(t, a, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if len(p.resueltas) != 0 {
		t.Errorf("lo obsoleto no se manda a la sesión: %v", p.resueltas)
	}
}
