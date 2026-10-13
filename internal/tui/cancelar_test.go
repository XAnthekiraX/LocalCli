package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"localcli/internal/tui/render"
)

// Esc detiene el agente en dos pasos: el primero avisa debajo de la caja y el
// segundo cancela de verdad.
func TestEscPideConfirmacionAntesDeDetener(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	app := appDePrueba(&puertoFalso{})
	app.vista = VistaPrincipal
	app.sesion = &Sesion{ID: "s1", Nombre: "s", Estado: EstadoTrabajando}
	app.chat.AñadirUsuario("hola")

	cancelado := false
	app.cancelarTurno = func() { cancelado = true }

	app.detenerTurno()
	if !app.cancelarArmado || cancelado {
		t.Fatalf("el primer esc debería solo avisar: armado=%v cancelado=%v", app.cancelarArmado, cancelado)
	}
	if !strings.Contains(render.SinANSI(app.View()), "presiona esc para detener al agente") {
		t.Fatal("debería avisar debajo de la caja de cómo detenerlo")
	}

	app.detenerTurno()
	if !cancelado || app.cancelarArmado {
		t.Fatalf("el segundo esc debería detener: armado=%v cancelado=%v", app.cancelarArmado, cancelado)
	}
	if app.chat.HayTurno() {
		t.Fatal("el turno debería quedar cortado")
	}
	if app.cancelarTurno != nil {
		t.Fatal("el cancelador debería soltarse")
	}
}

// El camino real del teclado: dos `esc` por Update() detienen el turno.
func TestEscPorElTecladoDetieneElTurno(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	app := appDePrueba(&puertoFalso{})
	app.vista = VistaPrincipal
	app.sesion = &Sesion{ID: "s1", Nombre: "s", Estado: EstadoTrabajando}
	app.chat.AñadirUsuario("hola")

	cancelado := false
	app.cancelarTurno = func() { cancelado = true }

	app.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if !app.cancelarArmado || cancelado {
		t.Fatalf("el primer esc debería armar: armado=%v cancelado=%v", app.cancelarArmado, cancelado)
	}
	app.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if !cancelado {
		t.Fatal("el segundo esc debería cancelar el turno")
	}
}

// Un keymap como el del usuario —`cancel` solo en `ctrl+f`— no deja sin parada:
// Esc detiene igual porque es del núcleo.
func TestEscDetieneAunqueElKeymapNoLoTraiga(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	app := appDePrueba(&puertoFalso{})
	km, err := app.teclaRes.Kmap.Reasignar(AccionCancelar, "ctrl+f")
	if err != nil {
		t.Fatal(err)
	}
	app.teclaRes = NuevoKeyResolver(km)

	app.vista = VistaPrincipal
	app.sesion = &Sesion{ID: "s1", Nombre: "s", Estado: EstadoTrabajando}
	app.chat.AñadirUsuario("hola")
	cancelado := false
	app.cancelarTurno = func() { cancelado = true }

	app.Update(tea.KeyMsg{Type: tea.KeyEsc})
	app.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if !cancelado {
		t.Fatal("Esc debería detener aunque el keymap no nombre cancel")
	}
}

// `esc` cancela en la vista y cierra el modal cuando hay uno abierto.
func TestEscCancelaEnLaVistaYCierraEnElModal(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	app := appDePrueba(&puertoFalso{})

	if accion, _ := app.teclaRes.Resolver(tea.KeyMsg{Type: tea.KeyEsc}, ContextoVista); accion != AccionCancelar {
		t.Fatalf("esc en la vista debería cancelar, no %d", accion)
	}
	if accion, _ := app.teclaRes.Resolver(tea.KeyMsg{Type: tea.KeyEsc}, ContextoModal); accion != AccionCerrarModal {
		t.Fatalf("esc en el modal debería cerrar, no %d", accion)
	}
}

// Detener el turno no borra lo que el modelo ya entregó: el segundo `esc` corta la
// generación pero el texto parcial se queda en pantalla.
func TestEscConservaLaRespuestaParcial(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	app := appDePrueba(&puertoFalso{})
	app.vista = VistaPrincipal
	app.sesion = &Sesion{ID: "s1", Nombre: "s", Estado: EstadoTrabajando}
	app.chat.AñadirUsuario("hola")
	app.chat.Fragmento("esto ya lo entregué")
	app.cancelarTurno = func() {}

	app.Update(tea.KeyMsg{Type: tea.KeyEsc})
	app.Update(tea.KeyMsg{Type: tea.KeyEsc})

	if app.chat.HayTurno() {
		t.Fatal("el turno debería quedar cortado")
	}
	if !strings.Contains(render.SinANSI(app.View()), "esto ya lo entregué") {
		t.Fatal("lo que el modelo ya entregó debería quedarse en pantalla")
	}
}
