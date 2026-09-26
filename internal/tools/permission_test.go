package tools

import (
	"errors"
	"testing"
)

// TestPlanNoTieneHerramientasDeEscritura — el catálogo de `plan` no incluye
// ninguna de escritura. No es que se bloqueen: sencillamente no están.
func TestPlanNoTieneHerramientasDeEscritura(t *testing.T) {
	for _, nombre := range HerramientasDePlan() {
		h, ok := Buscar(nombre)
		if !ok {
			t.Fatalf("plan declara %s, que no está en el catálogo", nombre)
		}
		if h.SoloBuild() {
			t.Errorf("plan declara %s, que es de escritura", nombre)
		}
	}
	if len(HerramientasDePlan()) != 7 {
		t.Errorf("plan tiene %d herramientas, quiero 7 de lectura", len(HerramientasDePlan()))
	}
}

// TestPlanPidiendoEscrituraSeRechaza — la verificación de T-B007-05: `plan`
// pidiendo una herramienta de escritura produce E_TOOL_NOT_ALLOWED.
func TestPlanPidiendoEscrituraSeRechaza(t *testing.T) {
	for _, nombre := range []string{"crear_archivo", "escribir_archivo", "editar_archivo", "eliminar_archivo", "crear_carpeta", "eliminar_carpeta"} {
		err := ComprobarPermiso(HerramientasDePlan(), nombre)
		if !errors.Is(err, ErrHerramientaNoPermitida) {
			t.Errorf("plan pidiendo %s: err = %v, quiero E_TOOL_NOT_ALLOWED", nombre, err)
		}
	}
}

// TestBuildTieneElCatalogoCompleto — `build` declara las trece y no se le
// rechaza ninguna.
func TestBuildTieneElCatalogoCompleto(t *testing.T) {
	if len(HerramientasDeBuild()) != 13 {
		t.Fatalf("build tiene %d herramientas, quiero 13", len(HerramientasDeBuild()))
	}
	for _, nombre := range NombresCatalogo() {
		if err := ComprobarPermiso(HerramientasDeBuild(), nombre); err != nil {
			t.Errorf("build pidiendo %s: %v", nombre, err)
		}
	}
}

// TestHerramientaDesconocidaSeRechaza — el catálogo es cerrado, sin importar
// el agente.
func TestHerramientaDesconocidaSeRechaza(t *testing.T) {
	err := ComprobarPermiso(HerramientasDeBuild(), "inventada")
	if !errors.Is(err, ErrHerramientaDesconocida) {
		t.Errorf("err = %v, quiero E_TOOL_UNKNOWN", err)
	}
}
