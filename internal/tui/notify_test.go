package tui

// Tests de T-F009: la línea de aviso de aprobaciones pendientes. Una prueba,
// una regla (frontend/05-quality/TESTING.md §3):
//
//	T-F009-01 → formato del contador: nada con 0, singular, plural
//	T-F009-02 → el recuento sube y baja con los eventos, de cualquier sesión
//	T-F009-03 → se ve con el panel de datos cerrado y no con el abierto

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// --- T-F009-01: el formato ----------------------------------------------------

func TestElFormatoDelContadorConCeroUnoYVarios(t *testing.T) {
	if got := formatoAvisoPendientes(0); got != "" {
		t.Errorf("sin pendientes no hay aviso: %q", got)
	}
	if got := formatoAvisoPendientes(-2); got != "" {
		t.Errorf("un recuento imposible tampoco avisa: %q", got)
	}
	if got := formatoAvisoPendientes(1); got != "1 aprobación esperando tu decisión" {
		t.Errorf("singular: %q", got)
	}
	if got := formatoAvisoPendientes(3); got != "3 aprobaciones esperando tu decisión" {
		t.Errorf("plural: %q", got)
	}
}

// --- T-F009-02: el recuento por eventos ----------------------------------------

func TestElRecuentoSubeYBajaConLosEventosDeCualquierSesión(t *testing.T) {
	a := Nuevo(&puertoStub{})
	a.Vista = VistaPrincipal
	a.Panel.SesionID = "s1"

	// Una pendiente de la sesión activa y otra de una de segundo plano: las
	// dos cuentan, porque las dos detienen trabajo que espera decisión.
	pulsa(t, a, eventoMsg{Evento: Evento{
		Nombre: EventoPeticionAprobacion,
		Datos:  map[string]string{"aprobacion": "a1", "sesion": "s1", "descripcion": "crear archivo"},
	}})
	pulsa(t, a, eventoMsg{Evento: Evento{
		Nombre: EventoPeticionAprobacion,
		Datos:  map[string]string{"aprobacion": "a2", "sesion": "s2", "descripcion": "borrar carpeta"},
	}})
	if a.Panel.Aprobaciones != 2 {
		t.Fatalf("dos pendientes de dos sesiones: %d", a.Panel.Aprobaciones)
	}

	pulsa(t, a, eventoMsg{Evento: Evento{
		Nombre: EventoAprobacionResuelta,
		Datos:  map[string]string{"aprobacion": "a2"},
	}})
	if a.Panel.Aprobaciones != 1 {
		t.Fatalf("resuelta la de segundo plano, queda una: %d", a.Panel.Aprobaciones)
	}
	pulsa(t, a, eventoMsg{Evento: Evento{
		Nombre: EventoAprobacionResuelta,
		Datos:  map[string]string{"aprobacion": "a1"},
	}})
	if a.Panel.Aprobaciones != 0 {
		t.Fatalf("sin pendientes el recuento vuelve a cero: %d", a.Panel.Aprobaciones)
	}
}

// --- T-F009-03: la visibilidad ---------------------------------------------------

func TestElAvisoSeVeConElPanelCerradoYNoConElAbierto(t *testing.T) {
	a := Nuevo(&puertoStub{})
	a.Vista = VistaPrincipal
	a.Panel.SesionID = "s1"
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})
	a.Panel.Aprobaciones = 2

	// Panel cerrado: el aviso es la única forma de saberlo.
	a.Panel.Abierto = false
	v := sinEstilo(a.View())
	if !strings.Contains(v, "2 aprobaciones esperando tu decisión") {
		t.Errorf("con el panel cerrado el aviso se ve:\n%s", v)
	}

	// Panel abierto: el dato ya está en su fila del panel y el aviso de la
	// barra no se repite.
	a.Panel.Abierto = true
	v = sinEstilo(a.View())
	if n := strings.Count(v, "esperando tu decisión"); n != 0 {
		t.Errorf("con el panel abierto el aviso no se repite fuera del panel: %d veces\n%s", n, v)
	}
	if !strings.Contains(v, "2 esperando decisión") {
		t.Errorf("con el panel abierto el dato vive en su fila:\n%s", v)
	}
}
