package agent

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"localcli/internal/tools"
)

// TestAgenteTieneExactamenteCincoCampos — el round-trip del JSON solo produce
// los cinco campos del contrato (DECISIONS.md).
func TestAgenteTieneExactamenteCincoCampos(t *testing.T) {
	a := Agente{
		Nombre:       "x",
		Descripcion:  "d",
		Prompt:       "p",
		Herramientas: []string{"leer_archivo"},
		Skills:       []string{"s"},
	}
	b, err := json.Marshal(a)
	if err != nil {
		t.Fatalf("no codifica: %v", err)
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("JSON inválido: %v", err)
	}
	if len(m) != len(CamposDelAgente) {
		t.Fatalf("el agente tiene %d campos, quiero %d: %s", len(m), len(CamposDelAgente), b)
	}
	for _, campo := range CamposDelAgente {
		if _, ok := m[campo]; !ok {
			t.Errorf("falta el campo %q", campo)
		}
	}
}

// TestHeredaDeSeRechaza — no hay herencia entre agentes: un campo `hereda_de`
// no se ignora, rompe la carga.
func TestHeredaDeSeRechaza(t *testing.T) {
	if _, err := Cargar(filepath.Join("testdata", "hereda.json")); err == nil {
		t.Fatal("un agente con `hereda_de` no puede cargarse")
	}
}

// TestHerramientaInventadaSeRechaza — el catálogo es cerrado: un agente que
// declara una herramienta que no existe no se carga.
func TestHerramientaInventadaSeRechaza(t *testing.T) {
	if _, err := Cargar(filepath.Join("testdata", "inventada.json")); err == nil {
		t.Fatal("un agente con una herramienta inventada no puede cargarse")
	}
}

// TestJSONRotoDaErrorLocalizado — un archivo ilegible falla nombrando el
// archivo, sin tumbar el proceso.
func TestJSONRotoDaErrorLocalizado(t *testing.T) {
	ruta := filepath.Join("testdata", "roto.json")
	if _, err := Cargar(ruta); err == nil {
		t.Fatal("un JSON roto no puede cargarse")
	}
}

// TestCargarAgenteValido — un fixture con los cinco campos carga y conserva
// sus datos.
func TestCargarAgenteValido(t *testing.T) {
	a, err := Cargar(filepath.Join("testdata", "valido.json"))
	if err != nil {
		t.Fatalf("Cargar: %v", err)
	}
	if a.Nombre != "lector" || len(a.Herramientas) != 2 || len(a.Skills) != 1 {
		t.Fatalf("agente cargado inesperado: %+v", a)
	}
}

// TestAgenteSinHerramientasEsSoloConversacion — T-B006-08: `herramientas`
// vacío es válido y produce un agente sin ruta hacia `tools`.
func TestAgenteSinHerramientasEsSoloConversacion(t *testing.T) {
	a, err := Cargar(filepath.Join("testdata", "solo_conversacion.json"))
	if err != nil {
		t.Fatalf("Cargar: %v", err)
	}
	if !a.SoloConversacion() {
		t.Error("un agente sin herramientas debe ser de solo conversación")
	}
	// Sin herramientas, no puede pedir ninguna: tools lo rechaza.
	if err := tools.ComprobarPermiso(a.Herramientas, "leer_archivo"); err == nil {
		t.Error("un agente de solo conversación no puede pedir herramientas")
	}
}

// TestAgentesBaseDocumentados — T-B006-04: los dos agentes base cargan desde
// `ai/agents/` y `plan` no contiene ninguna herramienta de escritura.
func TestAgentesBaseDocumentados(t *testing.T) {
	dir := filepath.Join("..", "..", "ai", "agents")
	agentes, err := CargarCarpeta(dir)
	if err != nil {
		t.Fatalf("CargarCarpeta: %v", err)
	}
	if len(agentes) < 2 {
		t.Fatalf("se esperan al menos plan y build, hay %d", len(agentes))
	}

	plan, ok := PorNombre(agentes, "plan")
	if !ok {
		t.Fatal("falta el agente base plan")
	}
	if plan.TieneEscritura() {
		t.Error("plan no puede tener ninguna herramienta de escritura")
	}
	if len(plan.Herramientas) != 7 {
		t.Errorf("plan declara %d herramientas, quiero 7 de lectura", len(plan.Herramientas))
	}

	build, ok := PorNombre(agentes, "build")
	if !ok {
		t.Fatal("falta el agente base build")
	}
	if len(build.Herramientas) != 13 {
		t.Errorf("build declara %d herramientas, quiero 13", len(build.Herramientas))
	}
	if !build.TieneEscritura() {
		t.Error("build debe tener herramientas de escritura")
	}
}
