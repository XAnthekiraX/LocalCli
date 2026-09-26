package tui

// Tests de T-F007: el selector momentáneo de sesiones. Una prueba, una regla
// (frontend/05-quality/TESTING.md §3). La apertura con Ctrl+S, el cierre con
// Esc, la navegación y el cambio de sesión ya tenían pruebas de T-B014 en
// tui_test.go; aquí van las reglas nuevas de esta tarea.
//
//	T-F007-02 → cerrado no deja rastro en la vista
//	T-F007-03 → cada fila pinta el estado tal cual los enums de store
//	T-F007-04 → elegir no cancela nada de lo que sigue corriendo
//	T-F007-05 → con el selector abierto, estado_sesion refresca su fila

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"localcli/internal/session"
)

// --- T-F007-03: las filas usan los enums --------------------------------------

func TestLasFilasDelSelectorPintanLosEnumsDeSesión(t *testing.T) {
	p := &puertoStub{sesiones: []session.Sesion{
		{ID: "s1", Nombre: "primera", Estado: session.EstadoInactiva},
		{ID: "s2", Nombre: "segunda", Estado: session.EstadoEsperandoPermiso},
		{ID: "s3", Nombre: "tercera", Estado: session.EstadoError},
	}}
	a := Nuevo(p)
	a.Vista = VistaPrincipal
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})
	tecla(t, a, tea.KeyCtrlS)

	v := sinEstilo(a.View())
	for _, enum := range []string{session.EstadoInactiva, session.EstadoEsperandoPermiso, session.EstadoError} {
		if !strings.Contains(v, enum) {
			t.Errorf("el estado %q del enum debe verse en su fila:\\n%s", enum, v)
		}
	}
}

// --- T-F007-02: cerrado no deja rastro -----------------------------------------

func TestElSelectorCerradoNoDejaRastroEnLaVista(t *testing.T) {
	p := &puertoStub{sesiones: []session.Sesion{{ID: "s1", Nombre: "una"}}}
	a := Nuevo(p)
	a.Vista = VistaPrincipal
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})

	tecla(t, a, tea.KeyCtrlS)
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

	tecla(t, a, tea.KeyCtrlS)
	tecla(t, a, tea.KeyDown)
	tecla(t, a, tea.KeyEnter)

	if a.Panel.SesionID != "s2" {
		t.Fatalf("la activa pasa a la elegida: %q", a.Panel.SesionID)
	}
	if len(p.cancelado) != 0 || len(p.pausadas) != 0 {
		t.Errorf("cambiar de sesión no cancela ni pausa nada: %v, %v", p.cancelado, p.pausadas)
	}
}

// --- T-F007-05: estados en vivo con el selector abierto --------------------------

func TestConElSelectorAbiertoLosEstadosSeRefrescanEnVivo(t *testing.T) {
	p := &puertoStub{sesiones: []session.Sesion{
		{ID: "s1", Nombre: "primera", Estado: session.EstadoInactiva},
		{ID: "s2", Nombre: "segunda", Estado: session.EstadoInactiva},
	}}
	a := Nuevo(p)
	a.Vista = VistaPrincipal
	a.Panel.SesionID = "s1"
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})
	tecla(t, a, tea.KeyCtrlS)

	pulsa(t, a, eventoMsg{Evento: Evento{
		Nombre: session.EventoEstadoSesion,
		Datos:  map[string]string{"sesion": "s2", "estado": session.EstadoTrabajando},
	}})
	if !a.Selector.Abierto {
		t.Fatal("el evento no cierra el selector")
	}
	if !strings.Contains(sinEstilo(a.View()), "trabajando") {
		t.Errorf("la fila de s2 se refresca con el selector abierto:\\n%s", sinEstilo(a.View()))
	}
}
