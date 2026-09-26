package flow

import (
	"testing"

	"localcli/internal/tools"
)

// TestTransicionesValidas — la máquina de estados del flujo: pendiente→en
// curso, pausado por permiso se retoma, y los estados finales no vuelven.
func TestTransicionesValidas(t *testing.T) {
	validas := []struct{ desde, hasta EstadoFlujo }{
		{EstadoPendiente, EstadoEnCurso},
		{EstadoEnCurso, EstadoPausadoPermiso},
		{EstadoPausadoPermiso, EstadoEnCurso},
		{EstadoEnCurso, EstadoTerminado},
		{EstadoEnCurso, EstadoDetenido},
		{EstadoEnCurso, EstadoConError},
	}
	for _, c := range validas {
		if !TransicionValida(c.desde, c.hasta) {
			t.Errorf("%s → %s debería ser válida", c.desde, c.hasta)
		}
	}
	invalidas := []struct{ desde, hasta EstadoFlujo }{
		{EstadoTerminado, EstadoEnCurso},
		{EstadoPendiente, EstadoPausadoPermiso},
		{EstadoPausadoPermiso, EstadoTerminado},
		{EstadoDetenido, EstadoEnCurso},
	}
	for _, c := range invalidas {
		if TransicionValida(c.desde, c.hasta) {
			t.Errorf("%s → %s no debería ser válida", c.desde, c.hasta)
		}
	}
}

// TestEstadoTrasDecisionRechazaInvalida — una decisión que produce una
// transición inválida se rechaza.
func TestEstadoTrasDecisionRechazaInvalida(t *testing.T) {
	if _, err := EstadoTrasDecision(EstadoTerminado, Seguir); err == nil {
		t.Error("un flujo terminado no puede seguir")
	}
	if _, err := EstadoTrasDecision(EstadoEnCurso, Decision(99)); err == nil {
		t.Error("una decisión desconocida debe rechazarse")
	}
}

// TestFlujoValidar — un flujo sin etapas o con un agente fuera del reparto no
// es encadenable.
func TestFlujoValidar(t *testing.T) {
	if err := (Flujo{Nombre: "x"}).Validar(); err == nil {
		t.Error("un flujo sin etapas no es válido")
	}
	malo := Flujo{Nombre: "x", Etapas: []Etapa{{ID: "a", Nombre: "A", Agente: "otro"}}}
	if err := malo.Validar(); err == nil {
		t.Error("un agente fuera de plan/build debe rechazarse")
	}
	bueno := Flujo{Nombre: "x", Etapas: []Etapa{{ID: "a", Nombre: "A", Agente: tools.AgentePlan}}}
	if err := bueno.Validar(); err != nil {
		t.Errorf("flujo válido rechazado: %v", err)
	}
}
