package tui

// Tests de T-F015: el indicador del agente a la izquierda del input y su ciclo
// con Tab (acción `agent_cycle`).
//
//	T-F015-01 → la línea de entrada de la vista principal empieza por el agente
//	T-F015-02 → la bienvenida incluye `[plan] >` por defecto
//	T-F015-03 → Tab alterna plan↔build en las dos vistas, con un modal abierto
//	            no cicla y Tab nunca escribe un tabulador
//	T-F015-04 → el agente elegido viaja con la petición enviada
//
// Una prueba, una regla (TESTING.md §3), contra SPEC-INTERFAZ §Zonas 2 y
// SPEC-KEYBINDS §Acción `agent_cycle`.

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// --- T-F015-01: el indicador en la línea de entrada ------------------------

func TestLaLineaDeEntradaEmpiezaPorElAgenteActivo(t *testing.T) {
	a := Nuevo(&puertoStub{})
	a.Vista = VistaPrincipal
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})

	if v := sinEstilo(a.Entrada.View()); !strings.HasPrefix(v, "[plan] > ") {
		t.Errorf("la línea de entrada debe empezar por el indicador: %q", v)
	}
	tecla(t, a, tea.KeyTab)
	if v := sinEstilo(a.Entrada.View()); !strings.HasPrefix(v, "[build] > ") {
		t.Errorf("tras Tab el indicador cambia al instante: %q", v)
	}
}

// --- T-F015-02: el indicador en la bienvenida ------------------------------

func TestLaBienvenidaMuestraElIndicadorDelAgente(t *testing.T) {
	a := Nuevo(&puertoStub{})
	if v := sinEstilo(a.View()); !strings.Contains(v, "[plan] > En qué te ayudo hoy:") {
		t.Errorf("la bienvenida muestra el indicador del agente:\n%s", v)
	}
}

// --- T-F015-03: el ciclo con Tab -------------------------------------------

func TestTabAlternaElAgenteEnLaVistaPrincipal(t *testing.T) {
	a := Nuevo(&puertoStub{})
	a.Vista = VistaPrincipal
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})

	if a.Agente != AgentePlan {
		t.Fatalf("la vista arranca en plan: %q", a.Agente)
	}
	tecla(t, a, tea.KeyTab)
	if a.Agente != AgenteBuild || a.Entrada.Agente != AgenteBuild {
		t.Errorf("Tab pasa a build en la vista y en el indicador: %q / %q", a.Agente, a.Entrada.Agente)
	}
	tecla(t, a, tea.KeyTab)
	if a.Agente != AgentePlan {
		t.Errorf("Tab vuelve a plan: %q", a.Agente)
	}
	if a.Entrada.Texto() != "" {
		t.Errorf("Tab nunca escribe un tabulador: %q", a.Entrada.Texto())
	}
}

func TestTabAlternaElAgenteEnLaBienvenida(t *testing.T) {
	a := Nuevo(&puertoStub{})
	tecla(t, a, tea.KeyTab)
	if a.Agente != AgenteBuild {
		t.Errorf("Tab cicla el agente en la bienvenida: %q", a.Agente)
	}
	if v := sinEstilo(a.View()); !strings.Contains(v, "[build] > En qué te ayudo hoy:") {
		t.Errorf("la bienvenida pinta el nuevo agente:\n%s", v)
	}
	if a.Bienvenida.Texto != "" {
		t.Errorf("Tab no escribe en la bienvenida: %q", a.Bienvenida.Texto)
	}
	tecla(t, a, tea.KeyTab)
	if a.Agente != AgentePlan {
		t.Errorf("Tab vuelve a plan: %q", a.Agente)
	}
}

func TestConUnModalAbiertoElTabNoCicla(t *testing.T) {
	// Vista principal: modal de sesiones.
	a, _ := appConModales(t)
	ejecuta(t, a, abreElModalDeSesiones(t, a))
	tecla(t, a, tea.KeyTab)
	if a.Agente != AgentePlan {
		t.Errorf("con el modal abierto Tab no cicla: %q", a.Agente)
	}

	// Bienvenida: modal de modelos.
	b := Nuevo(&puertoStub{modelos: modelosDePrueba()})
	ejecuta(t, b, abreElModalDeModelos(t, b))
	tecla(t, b, tea.KeyTab)
	if b.Agente != AgentePlan {
		t.Errorf("con el modal abierto Tab no cicla en la bienvenida: %q", b.Agente)
	}
}

// --- T-F015-04: el agente viaja con la petición ----------------------------

func TestElAgenteViajaConLaPeticionDesdeLaVistaPrincipal(t *testing.T) {
	p := &puertoStub{}
	a := Nuevo(p)
	a.Vista = VistaPrincipal
	a.Panel.SesionID = "s1"
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})

	escribe(t, a, "revisa esto")
	tecla(t, a, tea.KeyTab)
	ejecuta(t, a, tecla(t, a, tea.KeyEnter))
	if len(p.agentes) != 1 || p.agentes[0] != AgenteBuild {
		t.Fatalf("la petición sale con el agente activo: %v", p.agentes)
	}
}

func TestElAgenteViajaConLaPrimeraPeticionDesdeLaBienvenida(t *testing.T) {
	p := &puertoStub{}
	a := Nuevo(p)
	tecla(t, a, tea.KeyTab)
	escribe(t, a, "documenta la capa")
	ejecuta(t, a, tecla(t, a, tea.KeyEnter))
	if len(p.agentes) != 1 || p.agentes[0] != AgenteBuild {
		t.Fatalf("la primera petición sale con el agente activo: %v", p.agentes)
	}
}

// --- T-F031: el ciclo recorre todos los agentes disponibles -----------------

// Con un catálogo propio, Tab recorre todos los agentes, no solo plan y build.
func TestTabRecorreTodosLosAgentesDisponibles(t *testing.T) {
	p := &puertoStub{agentesDisponibles: []string{AgentePlan, AgenteBuild, "revisor"}}
	a := Nuevo(p)
	a.Vista = VistaPrincipal
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})

	if a.Agente != AgentePlan {
		t.Fatalf("arranca en el primero de la lista: %q", a.Agente)
	}
	tecla(t, a, tea.KeyTab)
	if a.Agente != AgenteBuild {
		t.Errorf("Tab pasa al segundo: %q", a.Agente)
	}
	tecla(t, a, tea.KeyTab)
	if a.Agente != "revisor" || a.Entrada.Agente != "revisor" {
		t.Errorf("Tab llega al agente propio en la vista y el indicador: %q / %q", a.Agente, a.Entrada.Agente)
	}
	if v := sinEstilo(a.Entrada.View()); !strings.HasPrefix(v, "[revisor] > ") {
		t.Errorf("el indicador pinta el agente propio: %q", v)
	}
	tecla(t, a, tea.KeyTab)
	if a.Agente != AgentePlan {
		t.Errorf("Tab vuelve al primero: %q", a.Agente)
	}
}

// Un agente recordado que ya no está en la lista cae en el primero disponible.
func TestElAgenteRecordadoSeValidaContraLaLista(t *testing.T) {
	p := &puertoStub{agenteRecordado: AgenteBuild, agentesDisponibles: []string{AgentePlan, "revisor"}}
	a := Nuevo(p)
	if a.Agente != AgentePlan {
		t.Errorf("un agente recordado que ya no existe cae en el primero: %q", a.Agente)
	}
}
