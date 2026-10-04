package tui

// Tests de T-F014-03, ampliados por T-F044: el enrutado de los CUATRO modales.
// Son reglas del `app`, no de un componente: qué acción abre cada modal, que
// solo puede haber uno abierto y que `dismiss` (Esc) cierra cualquiera
// (SPEC-INTERFAZ §Modales, SPEC-KEYBINDS §Acción y §Resolución por contexto,
// DOMAIN §1 `modals`).
//
//	T-F014-03 → abrir modelos, sesiones y motores deja uno solo visible
//	T-F014-03 → Esc desde cualquiera de los cuatro devuelve la vista previa

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// appConModales monta la interfaz principal con sesiones y modelos de prueba,
// lista para abrir y cerrar modales.
func appConModales(t *testing.T) (*App, *puertoStub) {
	t.Helper()
	p := &puertoStub{sesiones: sesionesDePrueba(), modelos: modelosDePrueba()}
	a := Nuevo(p)
	a.Vista = VistaPrincipal
	a.Panel.SesionID = "s1"
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 100})
	return a, p
}

func TestSoloPuedeHaberUnModalAbierto(t *testing.T) {
	a, _ := appConModales(t)

	// Modelos → sesiones: el primero deja de estar visible.
	ejecuta(t, a, abreElModalDeModelos(t, a))
	if !strings.Contains(sinEstilo(a.View()), "MODELOS") {
		t.Fatal("ctrl+x m abre el modal de modelos")
	}
	ejecuta(t, a, abreElModalDeSesiones(t, a))
	v := sinEstilo(a.View())
	if !strings.Contains(v, "SESIONES") {
		t.Errorf("ctrl+x l abre el de sesiones:\n%s", v)
	}
	if strings.Contains(v, "MODELOS") || a.Modelos.Abierto {
		t.Errorf("abrir un modal cierra el que hubiera: %+v", a.Modelos)
	}

	// Sesiones → atajos: igual.
	tecla(t, a, tea.KeyCtrlP)
	v = sinEstilo(a.View())
	if !strings.Contains(v, "ATAJOS") {
		t.Errorf("ctrl+p abre el modal de atajos:\n%s", v)
	}
	if strings.Contains(v, "SESIONES") || a.Sesiones.Abierto {
		t.Errorf("abrir un modal cierra el que hubiera: %+v", a.Sesiones)
	}
	if a.modalAbierto() != true {
		t.Error("el modal abierto es el que se ve")
	}

	// Atajos → motores: igual, el cuarto modal comparte la mecánica (T-F044).
	ejecuta(t, a, abreElModalDeMotores(t, a))
	v = sinEstilo(a.View())
	if !strings.Contains(v, "MOTORES") {
		t.Errorf("ctrl+x i abre el modal de motores:\n%s", v)
	}
	if strings.Contains(v, "ATAJOS") || a.AtajosModal.Abierto {
		t.Errorf("abrir un modal cierra el que hubiera: %+v", a.AtajosModal)
	}
	if !a.Motores.Abierto {
		t.Error("el modal de motores es el que se ve")
	}
}

func TestEscCierraCualquierModalYDevuelveLaVistaPrevia(t *testing.T) {
	a, _ := appConModales(t)
	escribe(t, a, "una petición en curso")
	previo := sinEstilo(a.View())

	for _, abre := range []struct {
		nombre   string
		abrir    func() tea.Cmd
		esperado string
	}{
		{"modelos", func() tea.Cmd { return abreElModalDeModelos(t, a) }, "MODELOS"},
		{"sesiones", func() tea.Cmd { return abreElModalDeSesiones(t, a) }, "SESIONES"},
		{"atajos", func() tea.Cmd { return tecla(t, a, tea.KeyCtrlP) }, "ATAJOS"},
		{"motores", func() tea.Cmd { return abreElModalDeMotores(t, a) }, "MOTORES"},
	} {
		ejecuta(t, a, abre.abrir())
		if !strings.Contains(sinEstilo(a.View()), abre.esperado) {
			t.Fatalf("el modal de %s se ve abierto:\n%s", abre.nombre, sinEstilo(a.View()))
		}
		tecla(t, a, tea.KeyEsc)
		if a.modalAbierto() {
			t.Errorf("esc cierra el modal de %s", abre.nombre)
		}
		if v := sinEstilo(a.View()); v != previo {
			t.Errorf("esc desde el modal de %s devuelve la vista previa:\n%s", abre.nombre, v)
		}
	}
}

func TestConUnModalAbiertoLasTeclasNoAlcanzanLaVistaDeAbajo(t *testing.T) {
	a, p := appConModales(t)
	escribe(t, a, "media línea")
	ejecuta(t, a, abreElModalDeSesiones(t, a))

	// Escribir y enviar son de la vista: con el modal abierto no ocurren
	// (SPEC-KEYBINDS §Resolución por contexto).
	escribe(t, a, " más")
	tecla(t, a, tea.KeyEnter)
	if a.Entrada.Texto() != "media línea" {
		t.Errorf("con el modal abierto no se escribe: %q", a.Entrada.Texto())
	}
	if len(p.enviados) != 0 {
		t.Errorf("con el modal abierto no se envía: %v", p.enviados)
	}
	// Cerrado, la vista vuelve a escribir y a enviar.
	tecla(t, a, tea.KeyEsc)
	escribe(t, a, " más")
	if a.Entrada.Texto() != "media línea más" {
		t.Errorf("al cerrar se escribe otra vez: %q", a.Entrada.Texto())
	}
}
