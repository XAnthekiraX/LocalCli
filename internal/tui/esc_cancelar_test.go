package tui

// Tests del doble `esc` para cancelar el trabajo en curso (app.go).

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"localcli/internal/session"
)

func TestElDobleEscCancelaElTrabajoEnCurso(t *testing.T) {
	p := &puertoStub{}
	a := Nuevo(p)
	a.Vista = VistaPrincipal
	a.Panel.SesionID = "s1"
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})
	pulsa(t, a, eventoMsg{Evento: Evento{
		Nombre: session.EventoEstadoSesion,
		Datos:  map[string]string{"sesion": "s1", "estado": session.EstadoTrabajando},
	}})

	// Primer esc: pide confirmación, no cancela.
	tecla(t, a, tea.KeyEsc)
	if len(p.cancelado) != 0 {
		t.Fatalf("el primer esc no cancela: %v", p.cancelado)
	}
	if !a.PidiendoCancelarEsc {
		t.Fatal("el primer esc abre la confirmación")
	}
	if !strings.Contains(sinEstilo(a.View()), "presiona esc otra vez para cancelar razonamiento") {
		t.Errorf("la confirmación se ve bajo el input:\n%s", sinEstilo(a.View()))
	}

	// Segundo esc: cancela.
	tecla(t, a, tea.KeyEsc)
	if len(p.cancelado) != 1 || p.cancelado[0] != "s1" {
		t.Errorf("el segundo esc cancela el trabajo: %v", p.cancelado)
	}
	if a.PidiendoCancelarEsc {
		t.Error("tras cancelar, la confirmación se cierra")
	}
}

func TestElDobleEscSeDescartaConOtraTecla(t *testing.T) {
	p := &puertoStub{}
	a := Nuevo(p)
	a.Vista = VistaPrincipal
	a.Panel.SesionID = "s1"
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})
	pulsa(t, a, eventoMsg{Evento: Evento{
		Nombre: session.EventoEstadoSesion,
		Datos:  map[string]string{"sesion": "s1", "estado": session.EstadoTrabajando},
	}})

	tecla(t, a, tea.KeyEsc)
	// Cualquier otra tecla descarta la confirmación en lugar de cancelar.
	escribe(t, a, "h")
	if a.PidiendoCancelarEsc {
		t.Error("otra tecla descarta la confirmación")
	}
	if len(p.cancelado) != 0 {
		t.Errorf("descartar no cancela: %v", p.cancelado)
	}
}

func TestSinTrabajoElEscNoPideCancelar(t *testing.T) {
	a := Nuevo(&puertoStub{})
	a.Vista = VistaPrincipal
	a.Panel.SesionID = "s1"
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})
	tecla(t, a, tea.KeyEsc)
	if a.PidiendoCancelarEsc {
		t.Error("sin trabajo en curso, esc no pide cancelar")
	}
}
