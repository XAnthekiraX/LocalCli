package tui

// Tests de T-F014-02: el modal de atajos. Una prueba, una regla
// (frontend/05-quality/TESTING.md §3), contra SPEC-INTERFAZ §Modales (fila
// Atajos), SPEC-KEYBINDS §Acción y §Configuración y DOMAIN §1 (`keysmodal`):
//
//	T-F014-02 → la tabla lista cada acción con su tecla, incluidas las
//	             secuencias con líder y las deshabilitadas
//	T-F014-02 → la tabla sale del keymap vigente: un atajo reasignado es el que
//	             se ve
//	T-F014-02 → es de solo lectura: no navega, Enter no aplica y no escribe

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// --- T-F014-02: el contenido de la tabla -------------------------------------

func TestElModalDeAtajosListaCadaAcciónConSuTecla(t *testing.T) {
	km := &KeysModal{}
	km.AbrirAtajos(KeymapPorDefecto().Entradas())

	v := sinEstilo(km.Render(100, 30))
	if !strings.Contains(v, "ATAJOS") {
		t.Fatalf("el modal lleva su cabecera:\n%s", v)
	}
	for _, a := range KeymapPorDefecto().Entradas() {
		literales := []string{}
		for _, sec := range a.Secuencias {
			literales = append(literales, sec.Describir())
		}
		for _, lit := range literales {
			if !strings.Contains(v, lit) {
				t.Errorf("falta el literal %q de la acción %q:\n%s", lit, a.Descripcion, v)
			}
		}
		if !strings.Contains(v, a.Descripcion) {
			t.Errorf("falta la acción %q:\n%s", a.Descripcion, v)
		}
	}
	// Las dos secuencias con líder salen con su forma completa, no con la tecla
	// líder expandida (SPEC-INTERFAZ §Modales: "incluidas las secuencias con
	// líder").
	for _, sec := range []string{"<leader>m", "<leader>l"} {
		if !strings.Contains(v, sec) {
			t.Errorf("la tabla muestra la secuencia %q tal cual:\n%s", sec, v)
		}
	}
	// La acción sin tecla de fábrica se lista marcada como deshabilitada
	// (SPEC-KEYBINDS §Binding: "Cero significa deshabilitada").
	if !strings.Contains(v, "(deshabilitada)") {
		t.Errorf("las acciones deshabilitadas se marcan:\n%s", v)
	}
}

func TestElModalDeAtajosMuestraElKeymapVigente(t *testing.T) {
	p := &puertoStub{}
	a := Nuevo(p)
	a.Vista = VistaPrincipal
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})
	porAccion := MapasPorDefecto()
	porAccion[AccionPanel] = []string{"ctrl+k"}
	km, err := NuevoKeymap(LíderPorDefecto, TimeoutPorDefectoMs, porAccion)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.FijarMapa(km); err != nil {
		t.Fatal(err)
	}

	tecla(t, a, tea.KeyCtrlP)
	v := sinEstilo(a.View())
	filaVigente := fmt.Sprintf("%-16s %s", "ctrl+k", "abrir o cerrar el panel de datos")
	filaFabrica := fmt.Sprintf("%-16s %s", "ctrl+d", "abrir o cerrar el panel de datos")
	if !strings.Contains(v, filaVigente) || strings.Contains(v, filaFabrica) {
		t.Errorf("la tabla sale del mapa vigente, no de una copia:\n%s", v)
	}
}

func TestElModalDeAtajosEsDeSoloLectura(t *testing.T) {
	p := &puertoStub{}
	a := Nuevo(p)
	a.Vista = VistaPrincipal
	a.Panel.SesionID = "s1"
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})
	escribe(t, a, "algo pendiente")
	tecla(t, a, tea.KeyCtrlP)
	antes := sinEstilo(a.View())

	// Ni navegar ni aplicar cambian nada: el modal es una tabla que se lee
	// (SPEC-INTERFAZ §Modales, fila Atajos: "No aplica nada").
	tecla(t, a, tea.KeyDown)
	tecla(t, a, tea.KeyUp)
	tecla(t, a, tea.KeyEnter)
	if despues := sinEstilo(a.View()); despues != antes {
		t.Errorf("navegar o aplicar no alteran la tabla:\n%s", despues)
	}
	if a.AtajosModal.Indice >= 0 {
		t.Errorf("una tabla de solo lectura no tiene fila resaltada: %d", a.AtajosModal.Indice)
	}
	if len(p.enviados) != 0 {
		t.Errorf("con el modal abierto enter no envía: %v", p.enviados)
	}
	if a.Entrada.Texto() != "algo pendiente" {
		t.Errorf("con el modal abierto no se escribe: %q", a.Entrada.Texto())
	}
	// Cerrarlo con Esc no deja rastro.
	tecla(t, a, tea.KeyEsc)
	if a.AtajosModal.Abierto {
		t.Error("esc cierra el modal de atajos")
	}
	if a.Entrada.Texto() != "algo pendiente" {
		t.Errorf("cerrar no pierde lo escrito: %q", a.Entrada.Texto())
	}
}
