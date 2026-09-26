package context

import (
	"strings"
	"testing"
)

// TestEnsamblarSoloIncluidos — el bloque lleva la cabecera con el objetivo y un
// apartado por documento incluido, nunca lo descartado.
func TestEnsamblarSoloIncluidos(t *testing.T) {
	incluidos := []DocumentoSeleccionado{
		{Ruta: "backend/BACKEND.md", Contenido: "## Cuerpo\n\ntexto\n"},
		{Ruta: "backend/DECISIONS.md", Contenido: "decisiones"},
	}
	bloque := Ensamblar("implementar la API", "etapa-3", incluidos)

	if !strings.Contains(bloque, "implementar la API") {
		t.Errorf("el bloque no menciona el objetivo: %q", bloque)
	}
	if !strings.Contains(bloque, "etapa-3") {
		t.Errorf("el bloque no menciona la etapa: %q", bloque)
	}
	for _, d := range incluidos {
		if !strings.Contains(bloque, "## "+d.Ruta) {
			t.Errorf("el bloque no aparta %s: %q", d.Ruta, bloque)
		}
		if !strings.Contains(bloque, strings.TrimRight(d.Contenido, "\n")) {
			t.Errorf("el bloque no lleva el contenido de %s", d.Ruta)
		}
	}
	if strings.Contains(bloque, "## descartado.md") {
		t.Error("el bloque no debe incluir documentos descartados")
	}
}

// TestEnsamblarSinEtapa — sin etapa no se añade esa línea.
func TestEnsamblarSinEtapa(t *testing.T) {
	bloque := Ensamblar("objetivo", "", nil)
	if strings.Contains(bloque, "Etapa:") {
		t.Errorf("no debe haber línea de etapa: %q", bloque)
	}
}
