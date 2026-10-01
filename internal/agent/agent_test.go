package agent

import (
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"localcli/internal/tools"
)

// TestAgenteSinSkills — el contrato tiene solo tres campos. Una codificación
// del agente no produce `skills` (ni `prompt` ni `herramientas`): el campo
// desapareció del contrato.
func TestAgenteSinSkills(t *testing.T) {
	a := Agente{
		Nombre:      "x",
		Descripcion: "d",
		Permissions: map[string]string{"read": "allow"},
	}
	b, err := yaml.Marshal(a)
	if err != nil {
		t.Fatalf("no codifica: %v", err)
	}
	var m map[string]any
	if err := yaml.Unmarshal(b, &m); err != nil {
		t.Fatalf("YAML inválido: %v", err)
	}
	if len(m) != len(CamposDelAgente) {
		t.Fatalf("el agente tiene %d campos, quiero %d: %s", len(m), len(CamposDelAgente), b)
	}
	for _, campo := range CamposDelAgente {
		if _, ok := m[campo]; !ok {
			t.Errorf("falta el campo %q", campo)
		}
	}
	for _, muerto := range []string{"skills", "prompt", "herramientas"} {
		if _, ok := m[muerto]; ok {
			t.Errorf("el contrato no admite %q: %s", muerto, b)
		}
	}
}

// TestYamlConCampoDesconocidoSeRechaza — un campo desconocido rompe la carga
// nombrando el campo, no se ignora.
func TestYamlConCampoDesconocidoSeRechaza(t *testing.T) {
	datos := []byte("name: x\ndescription: d\nskills: []\n")
	if _, err := DecodificarAgente(datos); err == nil {
		t.Fatal("un agente con un campo desconocido no puede cargarse")
	} else if !strings.Contains(err.Error(), "skills") {
		t.Errorf("el error debe nombrar el campo: %v", err)
	}
}

// TestPromptEnElYamlSeRechaza — el prompt vive en `prompt.md`, no en el YAML.
func TestPromptEnElYamlSeRechaza(t *testing.T) {
	datos := []byte("name: x\ndescription: d\nprompt: hola\n")
	if _, err := DecodificarAgente(datos); err == nil {
		t.Fatal("un `prompt:` en el YAML no puede cargarse")
	}
}

// TestCampoHerramientasSeRechaza — el catálogo se deriva de `permissions`; un
// `herramientas:` escrito por costumbre rompe la carga.
func TestCampoHerramientasSeRechaza(t *testing.T) {
	datos := []byte("name: x\ndescription: d\npermissions:\n  read: allow\nherramientas:\n  - leer_archivo\n")
	if _, err := DecodificarAgente(datos); err == nil {
		t.Fatal("un agente con `herramientas:` no puede cargarse")
	}
}

// TestHeredaDeSeRechaza — no hay herencia entre agentes: `hereda_de` rompe la
// carga.
func TestHeredaDeSeRechaza(t *testing.T) {
	datos := []byte("name: x\ndescription: d\nhereda_de: plan\n")
	if _, err := DecodificarAgente(datos); err == nil {
		t.Fatal("un agente con `hereda_de` no puede cargarse")
	}
}

// TestDefaultAllowSeRechaza — `default` solo admite `deny`: un `allow` donde se
// quería un `deny` concedería todo en silencio.
func TestDefaultAllowSeRechaza(t *testing.T) {
	if _, err := HerramientasDe(map[string]string{"default": "allow", "read": "allow"}); err == nil {
		t.Fatal("`default: allow` no puede cargarse")
	}
}

// TestPermisoDesconocidoSeRechaza — un permiso fuera del vocabulario se rechaza
// nombrandolo.
func TestPermisoDesconocidoSeRechaza(t *testing.T) {
	err := ValidarPermisos(Agente{Permissions: map[string]string{"borrar_el_mundo": "allow"}})
	if err == nil {
		t.Fatal("un permiso inventado no puede cargarse")
	}
	if !strings.Contains(err.Error(), "borrar_el_mundo") {
		t.Errorf("el error debe nombrar el permiso: %v", err)
	}
}

// TestAgenteSinPermissionsSoloConversa — sin `permissions` el agente es válido
// y de solo conversación, sin error.
func TestAgenteSinPermissionsSoloConversa(t *testing.T) {
	hs, err := HerramientasDe(nil)
	if err != nil {
		t.Fatalf("un agente sin permisos no es un error: %v", err)
	}
	if len(hs) != 0 {
		t.Fatalf("sin permisos no hay herramientas: %v", hs)
	}
	a := Agente{Nombre: "charla", Prompt: "p", Herramientas: hs}
	if !a.SoloConversacion() {
		t.Error("un agente sin permisos debe ser de solo conversación")
	}
	if err := tools.ComprobarPermiso(a.Permisos(), "leer_archivo"); err == nil {
		t.Error("un agente de solo conversación no puede pedir herramientas")
	}
}

// TestCarpetaSinPromptNoCarga — los dos archivos son obligatorios.
func TestCarpetaSinPromptNoCarga(t *testing.T) {
	if _, err := Cargar(filepath.Join("testdata", "sin_prompt")); err == nil {
		t.Fatal("una carpeta sin `prompt.md` no puede cargarse")
	}
}

// TestElNombreVieneDeName — el nombre del agente lo decide `name`, no el de la
// carpeta.
func TestElNombreVieneDeName(t *testing.T) {
	a, err := Cargar(filepath.Join("testdata", "renombrado"))
	if err != nil {
		t.Fatalf("Cargar: %v", err)
	}
	if a.Nombre != "research" {
		t.Errorf("nombre = %q, quiero `research` (el campo `name`)", a.Nombre)
	}
}

// TestCargarAgenteValido — un fixture válido carga, lee su prompt entero y
// deriva el catálogo efectivo de sus permisos.
func TestCargarAgenteValido(t *testing.T) {
	a, err := Cargar(filepath.Join("testdata", "valido"))
	if err != nil {
		t.Fatalf("Cargar: %v", err)
	}
	if a.Nombre != "lector" {
		t.Fatalf("nombre = %q", a.Nombre)
	}
	if a.Prompt != "Eres un agente de prueba que solo lee." {
		t.Errorf("prompt = %q, quiero el de `prompt.md`", a.Prompt)
	}
	if len(a.Herramientas) != len(tools.HerramientasDePlan()) {
		t.Errorf("un agente con `read: allow` tiene %d herramientas, quiero %d", len(a.Herramientas), len(tools.HerramientasDePlan()))
	}
	for _, h := range a.Herramientas {
		if !a.Declara(h) {
			t.Errorf("la herramienta %s debería estar en el catálogo derivado", h)
		}
	}
	if a.TieneEscritura() {
		t.Error("un agente con solo `read` no puede tener escritura")
	}
}

// TestElYamlRotoDaErrorLocalizado — un archivo ilegible falla nombrando la
// carpeta, sin tumbar el proceso.
func TestElYamlRotoDaErrorLocalizado(t *testing.T) {
	if _, err := Cargar(filepath.Join("testdata", "roto")); err == nil {
		t.Fatal("un YAML roto no puede cargarse")
	}
}

// TestElPlanDelRepoNoTieneHerramientasDeEscritura y
// TestElBuildDelRepoTieneElCatalogoCompleto — los agentes base versionados en
// `.localcli/agents/` cargan desde sus carpetas y conservan el reparto.
func TestElPlanDelRepoNoTieneHerramientasDeEscritura(t *testing.T) {
	plan := agenteDelRepo(t, "plan")
	if plan.TieneEscritura() {
		t.Error("plan no puede tener ninguna herramienta de escritura")
	}
	for _, h := range plan.Herramientas {
		if o, ok := tools.Buscar(h); ok && o.SoloBuild() {
			t.Errorf("plan no puede declarar %s", h)
		}
	}
}

func TestElBuildDelRepoTieneElCatalogoCompleto(t *testing.T) {
	build := agenteDelRepo(t, "build")
	if !build.TieneEscritura() {
		t.Error("build debe tener herramientas de escritura")
	}
	if len(build.Herramientas) != len(tools.NombresCatalogo()) {
		t.Errorf("build declara %d herramientas, quiero %d", len(build.Herramientas), len(tools.NombresCatalogo()))
	}
}

func TestAgentesBaseDocumentados(t *testing.T) {
	dir := filepath.Join("..", "..", ".localcli", "agents")
	agentes, err := CargarCarpeta(dir)
	if err != nil {
		t.Fatalf("CargarCarpeta: %v", err)
	}
	if _, ok := PorNombre(agentes, tools.AgentePlan); !ok {
		t.Fatal("falta el agente base plan")
	}
	if _, ok := PorNombre(agentes, tools.AgenteBuild); !ok {
		t.Fatal("falta el agente base build")
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

func agenteDelRepo(t *testing.T, nombre string) Agente {
	t.Helper()
	a, err := Cargar(filepath.Join("..", "..", ".localcli", "agents", nombre))
	if err != nil {
		t.Fatalf("Cargar(%s): %v", nombre, err)
	}
	return a
}
