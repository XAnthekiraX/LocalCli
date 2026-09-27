package tui

// Tests del modal de sesiones: las reglas de T-F007 (el selector momentáneo, ya
// convertido en modal) y las nuevas de T-F014. Una prueba, una regla
// (frontend/05-quality/TESTING.md §3):
//
//	T-F007-02 → cerrado no deja rastro en la vista
//	T-F007-03 → cada fila pinta el estado tal cual los enums de store
//	T-F007-04 → elegir no cancela nada de lo que sigue corriendo
//	T-F007-05 → con el modal abierto, estado_sesion refresca su fila
//	T-F014-01 → abre centrado y pidiendo la lista; ↑/↓ mueven, Enter abre la
//	             elegida y cierra, Esc cierra sin cambiar nada
//	T-F014-04 → la lista se pide al abrir (una lectura), y mientras no llega se
//	             ve «cargando…»; sin sesiones, «sin sesiones»

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"localcli/internal/session"
)

// --- T-F007-03: las filas usan los enums --------------------------------------

func TestLasFilasDelModalDeSesionesPintanLosEnumsDeSesión(t *testing.T) {
	p := &puertoStub{sesiones: []session.Sesion{
		{ID: "s1", Nombre: "primera", Estado: session.EstadoInactiva},
		{ID: "s2", Nombre: "segunda", Estado: session.EstadoEsperandoPermiso},
		{ID: "s3", Nombre: "tercera", Estado: session.EstadoError},
	}}
	a := Nuevo(p)
	a.Vista = VistaPrincipal
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})
	ejecuta(t, a, abreElModalDeSesiones(t, a))

	v := sinEstilo(a.View())
	for _, enum := range []string{session.EstadoInactiva, session.EstadoEsperandoPermiso, session.EstadoError} {
		if !strings.Contains(v, enum) {
			t.Errorf("el estado %q del enum debe verse en su fila:\n%s", enum, v)
		}
	}
}

// --- T-F007-02: cerrado no deja rastro -----------------------------------------

func TestElModalDeSesionesCerradoNoDejaRastroEnLaVista(t *testing.T) {
	p := &puertoStub{sesiones: []session.Sesion{{ID: "s1", Nombre: "una"}}}
	a := Nuevo(p)
	a.Vista = VistaPrincipal
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})

	ejecuta(t, a, abreElModalDeSesiones(t, a))
	if !strings.Contains(sinEstilo(a.View()), "SESIONES") {
		t.Fatal("abierto, el selector se ve")
	}
	tecla(t, a, tea.KeyEsc)
	if strings.Contains(sinEstilo(a.View()), "SESIONES") {
		t.Error("cerrado no pinta nada: no ocupa espacio permanente")
	}
}

// --- T-F007-04: elegir no cancela nada ------------------------------------------

func TestElegirSesiónNoCancelaLoQueCorre(t *testing.T) {
	p := &puertoStub{sesiones: []session.Sesion{
		{ID: "s1", Nombre: "primera"},
		{ID: "s2", Nombre: "segunda", Estado: session.EstadoTrabajando},
	}}
	a := Nuevo(p)
	a.Vista = VistaPrincipal
	a.Panel.SesionID = "s1"
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})

	ejecuta(t, a, abreElModalDeSesiones(t, a))
	tecla(t, a, tea.KeyDown)
	tecla(t, a, tea.KeyEnter)

	if a.Panel.SesionID != "s2" {
		t.Fatalf("la activa pasa a la elegida: %q", a.Panel.SesionID)
	}
	if len(p.cancelado) != 0 || len(p.pausadas) != 0 {
		t.Errorf("cambiar de sesión no cancela ni pausa nada: %v, %v", p.cancelado, p.pausadas)
	}
}

// --- T-F007-05: estados en vivo con el modal abierto --------------------------

func TestConElModalDeSesionesAbiertoLosEstadosSeRefrescanEnVivo(t *testing.T) {
	p := &puertoStub{sesiones: []session.Sesion{
		{ID: "s1", Nombre: "primera", Estado: session.EstadoInactiva},
		{ID: "s2", Nombre: "segunda", Estado: session.EstadoInactiva},
	}}
	a := Nuevo(p)
	a.Vista = VistaPrincipal
	a.Panel.SesionID = "s1"
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})
	ejecuta(t, a, abreElModalDeSesiones(t, a))

	pulsa(t, a, eventoMsg{Evento: Evento{
		Nombre: session.EventoEstadoSesion,
		Datos:  map[string]string{"sesion": "s2", "estado": session.EstadoTrabajando},
	}})
	if !a.Sesiones.Abierto {
		t.Fatal("el evento no cierra el modal")
	}
	if !strings.Contains(sinEstilo(a.View()), "trabajando") {
		t.Errorf("la fila de s2 se refresca con el modal abierto:\n%s", sinEstilo(a.View()))
	}
}

// --- T-F014-01: el modal como componente -------------------------------------

func TestElModalDeSesionesAbreCentradoYPidiendoLaLista(t *testing.T) {
	sm := &SessionsModal{}
	sm.AbrirSesiones("s1")
	if !sm.Abierto {
		t.Fatal("el modal se abre")
	}
	if sm.Aviso != AvisoCargando {
		t.Errorf("mientras la lectura no llega se ve «cargando…»: %q", sm.Aviso)
	}
	if len(sm.Lineas) != 0 {
		t.Errorf("la lista todavía no ha llegado: %v", sm.Lineas)
	}
	// Se pinta centrado, como los otros dos modales (SPEC-INTERFAZ §Modales:
	// "Tres modales centrados").
	v := sinEstilo(sm.Render(100, 30))
	lineas := strings.Split(strings.TrimRight(v, "\n"), "\n")
	if !strings.Contains(v, "SESIONES") || !strings.Contains(v, AvisoCargando) {
		t.Fatalf("el modal pinta cabecera y aviso:\n%s", v)
	}
	if !strings.HasPrefix(lineas[0], " ") {
		t.Errorf("el modal se centra en la ventana:\n%s", v)
	}
}

func TestAbrirElModalDeSesionesResaltaLaSesiónActiva(t *testing.T) {
	sm := &SessionsModal{}
	sm.AbrirSesiones("s2")
	sm.FijarSesiones(sesionesDePrueba())

	if ses, ok := sm.SesionElegida(); !ok || ses.ID != "s2" {
		t.Errorf("el resaltado arranca en la sesión activa: %+v", ses)
	}
	sm.Mover(1)
	if ses, _ := sm.SesionElegida(); ses.ID != "s3" {
		t.Errorf("↓ mueve el resaltado: %q", ses.ID)
	}
	sm.Mover(-1)
	if ses, _ := sm.SesionElegida(); ses.ID != "s2" {
		t.Errorf("↑ vuelve a la anterior: %q", ses.ID)
	}
	// La activa se distingue de la resaltada: es la que se está viendo.
	if !strings.Contains(sm.Lineas[sm.IndiceDe("s2")], "←") {
		t.Errorf("la fila de la activa va marcada:\n%v", sm.Lineas)
	}
}

func TestEscEnElModalDeSesionesNoCambiaNada(t *testing.T) {
	p := &puertoStub{sesiones: sesionesDePrueba()}
	a := Nuevo(p)
	a.Vista = VistaPrincipal
	a.Panel.SesionID = "s1"
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})

	ejecuta(t, a, abreElModalDeSesiones(t, a))
	tecla(t, a, tea.KeyDown)
	tecla(t, a, tea.KeyEsc)

	if a.Sesiones.Abierto {
		t.Fatal("esc cierra el modal")
	}
	if a.Panel.SesionID != "s1" {
		t.Errorf("descartar no cambia de sesión: %q", a.Panel.SesionID)
	}
	if p.peticionesChat != 0 {
		t.Errorf("descartar no pide el historial de nadie: %d", p.peticionesChat)
	}
}

// --- T-F014-04: la carga es bajo demanda --------------------------------------

func TestAbrirElModalDeSesionesEmiteUnaSolaLectura(t *testing.T) {
	p := &puertoStub{sesiones: sesionesDePrueba()}
	a := Nuevo(p)
	a.Vista = VistaPrincipal
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})

	// La lista se pide al abrir, no antes: con el modal ya en pantalla y
	// «cargando…» a la vista, la pantalla no está bloqueada (SPEC-INTERFAZ
	// §Cambiar de sesión).
	cmd := abreElModalDeSesiones(t, a)
	v := sinEstilo(a.View())
	if !strings.Contains(v, AvisoCargando) {
		t.Errorf("el modal aparece antes de que la lectura termine:\n%s", v)
	}
	ejecuta(t, a, cmd)
	if p.lecturas != 1 {
		t.Errorf("abrir el modal lee la lista una vez: %d", p.lecturas)
	}
	if v := sinEstilo(a.View()); strings.Contains(v, AvisoCargando) {
		t.Errorf("con la lista ya, el aviso de carga desaparece:\n%s", v)
	}

	// Cada apertura vuelve a leer: lo que se ve es lo que hay ahora.
	tecla(t, a, tea.KeyEsc)
	ejecuta(t, a, abreElModalDeSesiones(t, a))
	if p.lecturas != 2 {
		t.Errorf("volver a abrir es una lectura nueva: %d", p.lecturas)
	}
}

func TestSinSesionesElModalDeSesionesAvisaYSeCierraSinBloquear(t *testing.T) {
	p := &puertoStub{} // el proyecto todavía no tiene sesiones
	a := Nuevo(p)
	a.Vista = VistaPrincipal
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})
	ejecuta(t, a, abreElModalDeSesiones(t, a))

	v := sinEstilo(a.View())
	if !strings.Contains(v, AvisoSinSesiones) {
		t.Errorf("sin sesiones el modal avisa:\n%s", v)
	}
	// No hay nada que elegir ni que aplicar: Enter no abre ninguna sesión.
	tecla(t, a, tea.KeyEnter)
	if a.Panel.SesionID != "" || p.peticionesChat != 0 {
		t.Errorf("sin lista no hay sesión a la que ir: %q", a.Panel.SesionID)
	}
	tecla(t, a, tea.KeyEsc)
	if a.Sesiones.Abierto {
		t.Error("el aviso no bloquea: el modal se cierra igual")
	}
}

func TestLaRespuestaTardíaNoRellenaElModalDeSesionesYaCerrado(t *testing.T) {
	sm := &SessionsModal{}
	sm.AbrirSesiones("s1")
	sm.Cerrar()
	sm.FijarSesiones(sesionesDePrueba())

	if sm.Abierto || len(sm.Sesiones) != 0 {
		t.Errorf("una lista que llega tarde se descarta: %v", sm.Sesiones)
	}
}

// T-F027: el evento `titulo_sesion` (el título que el modelo generó con la
// primera petición) renombra la sesión en el panel —si es la activa— y en la
// lista del modal, sin cambiar su id.
func TestElEventoTituloRenombraLaSesionEnPanelYModal(t *testing.T) {
	a, _ := appConModales(t)
	ejecuta(t, a, abreElModalDeSesiones(t, a))

	// El evento se inyecta como llega del motor; su comando re-arma la escucha,
	// así que no se ejecuta aquí (como el resto de pruebas de eventos).
	pulsa(t, a, eventoMsg{Evento: Evento{
		Nombre: EventoTituloSesion,
		Datos:  map[string]string{"sesion": "s1", "nombre": "Documentar sesiones"},
	}})

	if a.Panel.Sesion != "Documentar sesiones" {
		t.Errorf("el panel refleja el título de la sesión activa: %q", a.Panel.Sesion)
	}
	if v := sinEstilo(a.View()); !strings.Contains(v, "Documentar sesiones") {
		t.Errorf("el modal refleja el título:\n%s", v)
	}
	if a.Panel.SesionID != "s1" {
		t.Errorf("el id no cambia al renombrar: %q", a.Panel.SesionID)
	}
}

// El título de otra sesión no toca el panel de la activa: solo su fila del
// modal (SPEC-INTERFAZ: el panel refleja la sesión activa, y solo esa).
func TestUnTituloDeOtraSesionNoTocaElPanel(t *testing.T) {
	a, _ := appConModales(t)
	a.Panel.Sesion = "primera"

	pulsa(t, a, eventoMsg{Evento: Evento{
		Nombre: EventoTituloSesion,
		Datos:  map[string]string{"sesion": "s2", "nombre": "Otra cosa"},
	}})

	if a.Panel.Sesion != "primera" {
		t.Errorf("un título ajeno no cambia el panel: %q", a.Panel.Sesion)
	}
}

// sesionesDePrueba es la lista que devuelve el doble: tres sesiones del
// proyecto con sus estados.
func sesionesDePrueba() []session.Sesion {
	return []session.Sesion{
		{ID: "s1", Nombre: "primera", Estado: session.EstadoInactiva},
		{ID: "s2", Nombre: "segunda", Estado: session.EstadoTrabajando},
		{ID: "s3", Nombre: "tercera", Estado: session.EstadoEsperandoPermiso},
	}
}
