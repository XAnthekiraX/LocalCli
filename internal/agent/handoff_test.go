package agent

import (
	"testing"

	"localcli/internal/tools"
)

// TestRelevoNoAplicaSinAprobacion — el relevo empieza en plan y `build` no
// puede escribir hasta que haya una propuesta aprobada.
func TestRelevoNoAplicaSinAprobacion(t *testing.T) {
	r := NuevoRelevo()
	if r.Agente() != tools.AgentePlan {
		t.Errorf("agente inicial = %q, quiero plan", r.Agente())
	}
	if r.PuedeEscribir() {
		t.Error("no se puede escribir antes de proponer y aprobar")
	}
	if err := r.Aprobar(); err == nil {
		t.Error("aprobar sin propuesta previa debe fallar")
	}
	if err := r.Aplicar(); err == nil {
		t.Error("aplicar sin aprobación debe fallar")
	}
}

// TestRelevoTrasAprobacion — el ciclo documentado: plan propone, se aprueba,
// build queda habilitado para aplicar.
func TestRelevoTrasAprobacion(t *testing.T) {
	r := NuevoRelevo()
	if err := r.Proponer(); err != nil {
		t.Fatalf("Proponer: %v", err)
	}
	if r.PuedeEscribir() {
		t.Error("proponer no habilita la escritura")
	}
	if err := r.Aprobar(); err != nil {
		t.Fatalf("Aprobar: %v", err)
	}
	if r.Agente() != tools.AgenteBuild {
		t.Errorf("agente = %q, quiero build tras la aprobación", r.Agente())
	}
	if !r.PuedeEscribir() {
		t.Error("tras aprobar, build debe poder aplicar")
	}
}

// TestAprobacionValeSoloParaLoPropuesto — tras aplicar, el relevo vuelve a
// plan: el siguiente cambio necesita su propia propuesta y su aprobación.
func TestAprobacionValeSoloParaLoPropuesto(t *testing.T) {
	r := NuevoRelevo()
	_ = r.Proponer()
	_ = r.Aprobar()
	if err := r.Aplicar(); err != nil {
		t.Fatalf("Aplicar: %v", err)
	}
	if r.Fase() != FasePlan || r.Agente() != tools.AgentePlan || r.PuedeEscribir() {
		t.Fatalf("tras aplicar quedo en fase %s, agente %q, escribir=%v; quiero volver a plan",
			r.Fase(), r.Agente(), r.PuedeEscribir())
	}
	if err := r.Aplicar(); err == nil {
		t.Error("no se puede aplicar dos veces la misma aprobación")
	}
}
