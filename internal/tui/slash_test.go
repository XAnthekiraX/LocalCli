package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"localcli/internal/tui/render"
)

// TestUnFlowNoEsUnComandoLocal: `/flow:…` no se consume en la vista; desde la
// bienvenida arranca una sesión normal y su texto llega al orquestador tal cual.
func TestUnFlowNoEsUnComandoLocal(t *testing.T) {
	p := &puertoFalso{}
	app := appDePrueba(p)
	app.bienvenida.Texto = "/flow:planificar crear la lista"

	pulsar(app, tea.KeyMsg{Type: tea.KeyEnter})

	if p.creadas != 1 {
		t.Fatalf("`/flow:` debería crear la sesión: %d", p.creadas)
	}
	if len(p.enviados) != 1 || p.enviados[0] != "/flow:planificar crear la lista" {
		t.Fatalf("`/flow:` debería llegar entero al orquestador: %v", p.enviados)
	}
}

// TestComandosLocalesNoLleganAlAgente: `/models` y `/sessions` se resuelven en
// la vista, abriendo su modal; a la sesión no llega ninguno.
func TestComandosLocalesNoLleganAlAgente(t *testing.T) {
	casos := []struct {
		texto string
		modal ModalVista
	}{
		{"/models", ModalModelos},
		{"/sessions", ModalSesiones},
	}
	for _, c := range casos {
		p := &puertoFalso{}
		app := appDePrueba(p)
		app.sesion = &Sesion{ID: "activa", Nombre: "S", Estado: EstadoTrabajando}
		app.vista = VistaPrincipal
		app.entrada.Texto = c.texto

		pulsar(app, tea.KeyMsg{Type: tea.KeyEnter})

		if len(p.enviados) != 0 {
			t.Fatalf("`%s` no debería llegar a la sesión: %v", c.texto, p.enviados)
		}
		if app.modal != c.modal {
			t.Fatalf("`%s` no abrió su modal: %v", c.texto, app.modal)
		}
	}
}

// TestRetryRepiteLaUltimaPeticion: `/retry` reenvía lo último sin volver a
// teclearlo, y sin petición previa solo avisa.
func TestRetryRepiteLaUltimaPeticion(t *testing.T) {
	p := &puertoFalso{}
	app := appDePrueba(p)
	app.sesion = &Sesion{ID: "activa", Nombre: "S", Estado: EstadoTrabajando}
	app.vista = VistaPrincipal
	app.ultimaPeticion = "implementa la tarea T-1"

	app.entrada.Texto = "/retry"
	pulsar(app, tea.KeyMsg{Type: tea.KeyEnter})

	if len(p.enviados) != 1 || p.enviados[0] != "implementa la tarea T-1" {
		t.Fatalf("`/retry` debería reenviar la última petición: %v", p.enviados)
	}
	if hilo := render.SinANSI(app.chat.Hilo(80)); !strings.Contains(hilo, "implementa la tarea T-1") {
		t.Fatalf("la petición repetida debería dejarse en el hilo:\n%s", hilo)
	}

	app.sesion = &Sesion{ID: "otra", Nombre: "O", Estado: EstadoTrabajando}
	app.ultimaPeticion = ""
	app.entrada.Texto = "/retry"
	pulsar(app, tea.KeyMsg{Type: tea.KeyEnter})
	if len(p.enviados) != 1 {
		t.Fatalf("`/retry` sin petición previa no debería enviar nada: %v", p.enviados)
	}
	hilo := render.SinANSI(app.chat.Hilo(80))
	if !strings.Contains(hilo, "petición previa") {
		t.Fatalf("`/retry` sin petición debería dejar el aviso en el hilo:\n%s", hilo)
	}
}

// TestUndoDeshaceElTurno: `/undo` deshace el último turno de la sesión activa,
// recarga el hilo y deja el aviso.
func TestUndoDeshaceElTurno(t *testing.T) {
	p := &puertoFalso{deshacerN: 2}
	app := appDePrueba(p)
	app.sesion = &Sesion{ID: "activa", Nombre: "S", Estado: EstadoTrabajando}
	app.vista = VistaPrincipal
	app.entrada.Texto = "/undo"
	pulsar(app, tea.KeyMsg{Type: tea.KeyEnter})

	if len(p.deshechos) != 1 || p.deshechos[0] != "activa" {
		t.Fatalf("`/undo` debería deshacer en la sesión activa: %v", p.deshechos)
	}
	if p.historialPedido != "activa" {
		t.Fatalf("`/undo` debería recargar el historial: %q", p.historialPedido)
	}
	if hilo := render.SinANSI(app.chat.Hilo(80)); !strings.Contains(hilo, "turno deshecho") {
		t.Fatalf("`/undo` debería dejar el aviso en el hilo:\n%s", hilo)
	}
}

// TestUndoSinTurnoAvisa: sin nada que deshacer, `/undo` avisa y no recarga.
func TestUndoSinTurnoAvisa(t *testing.T) {
	p := &puertoFalso{}
	app := appDePrueba(p)
	app.sesion = &Sesion{ID: "activa", Nombre: "S", Estado: EstadoTrabajando}
	app.vista = VistaPrincipal
	app.entrada.Texto = "/undo"
	pulsar(app, tea.KeyMsg{Type: tea.KeyEnter})

	if p.historialPedido != "" {
		t.Fatalf("sin nada que deshacer no debería recargar el historial: %q", p.historialPedido)
	}
	if hilo := render.SinANSI(app.chat.Hilo(80)); !strings.Contains(hilo, "nada que deshacer") {
		t.Fatalf("`/undo` vacío debería avisar:\n%s", hilo)
	}
}

// TestRedoRehaceElTurno: `/redo` vuelve a poner en pie el turno deshecho.
func TestRedoRehaceElTurno(t *testing.T) {
	p := &puertoFalso{rehacerN: 2}
	app := appDePrueba(p)
	app.sesion = &Sesion{ID: "activa", Nombre: "S", Estado: EstadoTrabajando}
	app.vista = VistaPrincipal
	app.entrada.Texto = "/redo"
	pulsar(app, tea.KeyMsg{Type: tea.KeyEnter})

	if len(p.rehechos) != 1 || p.rehechos[0] != "activa" {
		t.Fatalf("`/redo` debería rehacer en la sesión activa: %v", p.rehechos)
	}
	if hilo := render.SinANSI(app.chat.Hilo(80)); !strings.Contains(hilo, "turno rehecho") {
		t.Fatalf("`/redo` debería dejar el aviso en el hilo:\n%s", hilo)
	}
}

// TestSkillYComandosDesconocidosSoloAvisan: `/skill:…` y un `/comando` que no se
// reconoce no envían nada al agente: dejan un aviso en la vista.
func TestSkillYComandosDesconocidosSoloAvisan(t *testing.T) {
	p := &puertoFalso{}
	app := appDePrueba(p)

	app.bienvenida.Texto = "/skill:contexto"
	pulsar(app, tea.KeyMsg{Type: tea.KeyEnter})
	if app.aviso == "" || !strings.Contains(app.aviso, "no están implementadas") {
		t.Fatalf("`/skill:` debería avisar en la bienvenida: %q", app.aviso)
	}

	app.bienvenida.Texto = "/nada"
	pulsar(app, tea.KeyMsg{Type: tea.KeyEnter})
	if !strings.Contains(app.aviso, "comando desconocido") {
		t.Fatalf("`/nada` debería avisar el comando desconocido: %q", app.aviso)
	}
	if len(p.enviados) != 0 || p.creadas != 0 {
		t.Fatalf("los avisos no deberían crear sesión ni enviar: enviados=%v creadas=%d", p.enviados, p.creadas)
	}
}
