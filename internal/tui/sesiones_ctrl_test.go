package tui

// Tests de T-F016 y T-F017: el modal de sesiones en la bienvenida y los atajos
// `Ctrl+X n` / `Ctrl+D` de la vista principal (SPEC-KEYBINDS §Acción
// `session_new` y `session_delete`, SPEC-INTERFAZ §Pantalla de bienvenida,
// SPEC-SESIONES).
//
//	T-F016 → abrir el modal desde la bienvenida y elegir una sesión
//	T-F017 → Ctrl+X n crea la sesión y la deja activa
//	T-F017 → Ctrl+D borra sin confirmar si no trabaja
//	T-F017 → Ctrl+D pide confirmación si trabaja

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"localcli/internal/session"
)

// T-F016: la bienvenida abre el modal de sesiones y elegir una de ellas lleva a
// la vista principal con esa sesión y su historial.
func TestLaBienvenidaAbreElModalDeSesionesYAlElegirPasaALaPrincipal(t *testing.T) {
	p := &puertoStub{
		sesiones:  sesionesDePrueba(),
		historial: []MensajeHistorial{{Rol: "user", Texto: "hola"}},
	}
	a := Nuevo(p)
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})

	ejecuta(t, a, abreElModalDeSesiones(t, a))
	if !a.Sesiones.Abierto {
		t.Fatal("ctrl+x l abre el modal de sesiones en la bienvenida")
	}
	v := sinEstilo(a.View())
	if !strings.Contains(v, "SESIONES") || !strings.Contains(v, "segunda") {
		t.Fatalf("el modal pinta la lista de sesiones:\n%s", v)
	}

	// La lista resalta la activa; una flecha abajo marca la segunda y Enter la abre.
	tecla(t, a, tea.KeyDown)
	ejecuta(t, a, tecla(t, a, tea.KeyEnter))

	if a.Vista != VistaPrincipal {
		t.Error("elegir una sesión desde la bienvenida lleva a la vista principal")
	}
	if a.Sesiones.Abierto {
		t.Error("enter cierra el modal")
	}
	if a.Panel.SesionID != "s2" {
		t.Errorf("la sesión elegida queda activa: %q", a.Panel.SesionID)
	}
	if p.peticionesChat == 0 {
		t.Error("el historial de la sesión elegida se pide")
	}
}

// T-F017: Ctrl+X n crea una sesión y la deja activa, con el chat empezando en
// blanco (una sesión nueva no hereda el hilo de la anterior).
func TestCtrlXNCreaUnaSesionNuevaYLaDejaActiva(t *testing.T) {
	a, p := appConModales(t)
	a.Chat.AñadirUsuario("lo que había antes")

	secuencia(t, a, tea.KeyCtrlX, "n")

	if p.creadas != 1 {
		t.Fatalf("ctrl+x n crea una sesión: %d", p.creadas)
	}
	if a.Panel.SesionID != "nueva" {
		t.Errorf("la sesión creada queda activa: %q", a.Panel.SesionID)
	}
	if a.Vista != VistaPrincipal {
		t.Error("la vista sigue siendo la principal")
	}
	if v := strings.TrimSpace(sinEstilo(a.Chat.Render(80))); v != "" {
		t.Errorf("el chat de la sesión nueva empieza en blanco:\n%s", v)
	}
	if a.modalAbierto() {
		t.Error("crear una sesión no deja ningún modal abierto")
	}
}

// T-F017: con el modal abierto, Ctrl+D borra la resaltada sin confirmar
// cuando no está trabajando.
func TestCtrlDBorraSinConfirmarSiNoTrabaja(t *testing.T) {
	a, p := appConModales(t)
	// La resaltada es la activa (s1) y está inactiva: se borra sin preguntar.
	ejecuta(t, a, abreElModalDeSesiones(t, a))

	tecla(t, a, tea.KeyCtrlD)

	if len(p.eliminadas) != 1 || p.eliminadas[0] != "s1" {
		t.Fatalf("ctrl+d borra la sesión resaltada: %v", p.eliminadas)
	}
	if a.PidiendoEliminarSesion != "" {
		t.Error("una sesión que no trabaja se borra sin confirmación")
	}
	if !a.Sesiones.Abierto {
		t.Error("el modal sigue abierto con la lista refrescada")
	}
	// Al borrar la activa el chat retoma otra viva del proyecto en lugar de
	// quedarse apuntando a una sesión que ya no existe.
	if a.Panel.SesionID == "s1" || a.Panel.SesionID == "" {
		t.Errorf("la sesión activa eliminada deja paso a otra: %q", a.Panel.SesionID)
	}
}

// T-F017: Ctrl+D sobre una sesión que trabaja pide confirmación; `s` la
// confirma y la borra, `n` y Esc cancelan.
func TestCtrlDPideConfirmacionCuandoLaSesionTrabaja(t *testing.T) {
	t.Run("s confirma", func(t *testing.T) {
		a, p := appConModales(t)
		ejecuta(t, a, abreElModalDeSesiones(t, a))
		tecla(t, a, tea.KeyDown) // resalta la segunda, que está trabajando

		tecla(t, a, tea.KeyCtrlD)
		if a.PidiendoEliminarSesion != "s2" {
			t.Fatalf("ctrl+d sobre una sesión que trabaja pide confirmación: %q", a.PidiendoEliminarSesion)
		}
		if len(p.eliminadas) != 0 {
			t.Errorf("sin confirmar todavía no se borra: %v", p.eliminadas)
		}
		v := sinEstilo(a.View())
		if !strings.Contains(v, "seguro que deseas eliminarla") {
			t.Errorf("la confirmación se pinta:\n%s", v)
		}

		// Mientras espera la respuesta, ninguna otra tecla pasa a la vista.
		escribe(t, a, "zzz")
		tecla(t, a, tea.KeyEnter)
		if len(p.enviados) != 0 || len(p.eliminadas) != 0 {
			t.Errorf("la confirmación captura el teclado: %v / %v", p.enviados, p.eliminadas)
		}

		escribe(t, a, "s")
		if a.PidiendoEliminarSesion != "" {
			t.Error("s cierra la confirmación")
		}
		if len(p.eliminadas) != 1 || p.eliminadas[0] != "s2" {
			t.Errorf("s confirma el borrado: %v", p.eliminadas)
		}
	})

	t.Run("n cancela", func(t *testing.T) {
		a, p := appConModales(t)
		ejecuta(t, a, abreElModalDeSesiones(t, a))
		tecla(t, a, tea.KeyDown)

		tecla(t, a, tea.KeyCtrlD)
		escribe(t, a, "n")

		if a.PidiendoEliminarSesion != "" {
			t.Error("n cierra la confirmación")
		}
		if len(p.eliminadas) != 0 {
			t.Errorf("n cancela el borrado: %v", p.eliminadas)
		}
	})

	t.Run("esc cancela", func(t *testing.T) {
		a, p := appConModales(t)
		ejecuta(t, a, abreElModalDeSesiones(t, a))
		tecla(t, a, tea.KeyDown)

		tecla(t, a, tea.KeyCtrlD)
		tecla(t, a, tea.KeyEsc)

		if a.PidiendoEliminarSesion != "" {
			t.Error("esc cierra la confirmación")
		}
		if len(p.eliminadas) != 0 {
			t.Errorf("esc cancela el borrado: %v", p.eliminadas)
		}
	})
}

// SPEC-SESIONES / T-F027: al borrar la última sesión del proyecto la vista
// vuelve a la bienvenida de inmediato, sin sesión activa ni modal abierto.
func TestCtrlDSobreLaUltimaSesionVuelveALaBienvenida(t *testing.T) {
	p := &puertoStub{sesiones: []session.Sesion{
		{ID: "s1", Nombre: "única", Estado: session.EstadoInactiva},
	}}
	a := Nuevo(p)
	a.Vista = VistaPrincipal
	a.Panel.SesionID = "s1"
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})

	ejecuta(t, a, abreElModalDeSesiones(t, a))
	tecla(t, a, tea.KeyCtrlD)

	if len(p.eliminadas) != 1 || p.eliminadas[0] != "s1" {
		t.Fatalf("ctrl+d borra la última sesión: %v", p.eliminadas)
	}
	if a.Vista != VistaBienvenida {
		t.Errorf("sin sesiones la vista vuelve a la bienvenida: %v", a.Vista)
	}
	if a.Panel.SesionID != "" {
		t.Errorf("no queda sesión activa: %q", a.Panel.SesionID)
	}
	if a.modalAbierto() {
		t.Error("al volver a la bienvenida no queda ningún modal abierto")
	}
}
