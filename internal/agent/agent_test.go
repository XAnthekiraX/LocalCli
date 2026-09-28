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
		Nombre:      "x",
		Descripcion: "d",
		Prompt:      "p",
		Permisos:    []Permiso{{Accion: "leer", Efecto: EfectoPermitir}},
		Skills:      []string{"s"},
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

// TestPermisoInventadoSeRechaza — el catálogo de acciones es cerrado: un
// agente que declara un permiso sobre una acción que no existe no se carga.
func TestPermisoInventadoSeRechaza(t *testing.T) {
	if _, err := Cargar(filepath.Join("testdata", "inventada.json")); err == nil {
		t.Fatal("un agente con una acción de permiso inventada no puede cargarse")
	}
}

// TestPermisosContradictoriosSeRechazan — `permitir` y `denegar` sobre la
// misma acción es un contrato que se contradice: no se carga.
func TestPermisosContradictoriosSeRechazan(t *testing.T) {
	if _, err := Cargar(filepath.Join("testdata", "contradictorio.json")); err == nil {
		t.Fatal("unos permisos contradictorios no pueden cargarse")
	}
}

// TestCampoHerramientasSeRechaza — el contrato ya no declara la lista de
// herramientas: `permisos` es la fuente. Un JSON viejo con `herramientas` no
// se ignora, rompe la carga.
func TestCampoHerramientasSeRechaza(t *testing.T) {
	datos := []byte(`{"nombre":"viejo","descripcion":"","prompt":"p","herramientas":["leer_archivo"],"skills":[]}`)
	if _, err := DecodificarAgente(datos); err == nil {
		t.Fatal("un agente con el campo viejo `herramientas` no puede cargarse")
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

// TestCargarAgenteValido — un fixture con los cinco campos carga, deriva el
// catálogo efectivo de sus permisos y conserva sus datos.
func TestCargarAgenteValido(t *testing.T) {
	a, err := Cargar(filepath.Join("testdata", "valido.json"))
	if err != nil {
		t.Fatalf("Cargar: %v", err)
	}
	// `leer` son las cuatro herramientas de lectura: el catálogo se deriva.
	if a.Nombre != "lector" || len(a.Herramientas) != 4 || len(a.Skills) != 1 {
		t.Fatalf("agente cargado inesperado: %+v", a)
	}
	for _, h := range a.Herramientas {
		if !a.Declara(h) {
			t.Errorf("la herramienta %s debería estar en el catálogo derivado", h)
		}
	}
	if a.TieneEscritura() {
		t.Error("un agente con solo `leer` no puede tener escritura")
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
	// Sin acciones, no puede pedir ninguna: tools lo rechaza.
	if err := tools.ComprobarPermiso(a.Acciones(), "leer_archivo"); err == nil {
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
	if len(plan.Herramientas) != 8 {
		t.Errorf("plan declara %d herramientas, quiero 8 (lectura, ejecución, internet y tareas)", len(plan.Herramientas))
	}

	build, ok := PorNombre(agentes, "build")
	if !ok {
		t.Fatal("falta el agente base build")
	}
	if len(build.Herramientas) != 14 {
		t.Errorf("build declara %d herramientas, quiero 14", len(build.Herramientas))
	}
	if !build.TieneEscritura() {
		t.Error("build debe tener herramientas de escritura")
	}
}

// TestOrdenarNombresPoneLosBasePrimero — el orden de presentación es estable:
// `plan` y `build` primero, el resto alfabético.
func TestOrdenarNombresPoneLosBasePrimero(t *testing.T) {
	agentes := map[string]Agente{
		"zeta":  {Nombre: "zeta"},
		"build": {Nombre: "build"},
		"alfa":  {Nombre: "alfa"},
		"plan":  {Nombre: "plan"},
	}
	quiere := []string{"plan", "build", "alfa", "zeta"}
	got := OrdenarNombres(agentes)
	if len(got) != len(quiere) {
		t.Fatalf("orden = %v, quiero %v", got, quiere)
	}
	for i := range quiere {
		if got[i] != quiere[i] {
			t.Fatalf("orden = %v, quiero %v", got, quiere)
		}
	}
	if len(OrdenarNombres(nil)) != 0 {
		t.Error("un catálogo vacío da una lista vacía")
	}
}
