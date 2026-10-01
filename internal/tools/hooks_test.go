package tools

import (
	"context"
	"encoding/json"
	"testing"
)

// TestHooksSeInvocaAntesYDespues — T-B024-07: los ganchos opcionales se llaman
// alrededor de cada ejecución sin que el catálogo sepa quién los instala.
func TestHooksSeInvocaAntesYDespues(t *testing.T) {
	r := registroStub("ok", nil)
	var antes, despues int
	var visto string
	r.Hooks = Hooks{
		AntesDeEjecutar: func(nombre string, args any, meta map[string]any) {
			antes++
			visto = nombre
		},
		DespuesDeEjecutar: func(nombre string, res Resultado, err error, meta map[string]any) {
			despues++
		},
	}
	if _, err := r.Ejecutar(context.Background(), Peticion{
		Agente:      AgenteBuild,
		Permisos:    PermisosDeBuild(),
		Herramienta: "leer_archivo",
		Argumentos:  json.RawMessage(`{"ruta":"a.md"}`),
	}); err != nil {
		t.Fatal(err)
	}
	if antes != 1 || despues != 1 {
		t.Fatalf("hooks: antes=%d despues=%d, quiero 1 y 1", antes, despues)
	}
	if visto != "leer_archivo" {
		t.Errorf("el hook recibió %q, quiero el nombre de la herramienta", visto)
	}
}

// TestDefinirHerramientaAdaptaElCatalogo — el tercer gancho puede ajustar
// nombre, descripción y esquema antes de que lleguen al modelo.
func TestDefinirHerramientaAdaptaElCatalogo(t *testing.T) {
	r := registroStub("ok", nil)
	r.Hooks = Hooks{
		DefinirHerramienta: func(nombre, descripcion string, e *Esquema) (string, *Esquema) {
			return nombre + "_v2", e
		},
	}
	defs := r.Definiciones(PermisosDePlan())
	found := false
	for _, d := range defs {
		if d.Nombre == "leer_archivo_v2" {
			found = true
		}
	}
	if !found {
		t.Errorf("el hook debe poder adaptar el nombre que ve el modelo: %+v", defs)
	}
}

// TestDefinicionesRepartenPorPermiso — el catálogo efectivo sale de los permisos:
// `plan` no ve ninguna de escritura, `build` las ve todas.
func TestDefinicionesRepartenPorPermiso(t *testing.T) {
	r := registroStub("ok", nil)
	for _, d := range r.Definiciones(PermisosDePlan()) {
		if h, ok := Buscar(d.Nombre); ok && h.SoloBuild() {
			t.Errorf("plan no puede ver %s en sus definiciones", d.Nombre)
		}
	}
	if got := len(r.Definiciones(PermisosDeBuild())); got != len(NombresCatalogo()) {
		t.Errorf("build ve %d herramientas, quiero %d", got, len(NombresCatalogo()))
	}
}
