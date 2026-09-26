package context

import (
	"strings"
	"testing"
)

func docsDe(ids ...string) []DocumentoSeleccionado {
	out := make([]DocumentoSeleccionado, 0, len(ids))
	for _, id := range ids {
		out = append(out, DocumentoSeleccionado{Ruta: id, Contenido: "## " + id + "\n\n" + strings.Repeat("x", 40), Tokens: 12})
	}
	return out
}

// TestRecortarDentroDelLimite — una selección que excede el límite queda dentro
// sin vaciarse, y lo que no cabe se descarta con motivo.
func TestRecortarDentroDelLimite(t *testing.T) {
	incluidos, descartados, tokens := Recortar(docsDe("a", "b", "c", "d", "e"), 30)
	if tokens > 30 {
		t.Fatalf("tokens = %d, quiero <= 30", tokens)
	}
	if len(incluidos) == 0 {
		t.Fatal("el recorte no puede vaciar el contexto")
	}
	if len(incluidos)+len(descartados) != 5 {
		t.Fatalf("incluidos %d + descartados %d, quiero 5", len(incluidos), len(descartados))
	}
	for _, d := range descartados {
		if d.Motivo == "" {
			t.Errorf("el descarte de %s no tiene motivo", d.Ruta)
		}
	}
}

// TestRecortarSinLimiteIncluyeTodo — sin tope, entra todo.
func TestRecortarSinLimiteIncluyeTodo(t *testing.T) {
	incluidos, descartados, _ := Recortar(docsDe("a", "b"), 0)
	if len(incluidos) != 2 || len(descartados) != 0 {
		t.Fatalf("incluidos=%d descartados=%d, quiero 2/0", len(incluidos), len(descartados))
	}
}

// TestRecortarDocumentoGrandePorSeccion — un documento que no cabe entero se
// recorta por secciones y no se descarta si es el primero.
func TestRecortarDocumentoGrandePorSeccion(t *testing.T) {
	grande := strings.Repeat("linea larga de texto\n", 50)
	sel := []DocumentoSeleccionado{{Ruta: "grande.md", Contenido: grande, Tokens: 1000}}
	incluidos, descartados, tokens := Recortar(sel, 20)
	if len(incluidos) != 1 || len(descartados) != 0 {
		t.Fatalf("incluidos=%d descartados=%d, quiero 1/0", len(incluidos), len(descartados))
	}
	if tokens > 20 {
		t.Fatalf("tokens = %d, quiero <= 20", tokens)
	}
	if incluidos[0].Contenido == "" {
		t.Error("el documento recortado no puede quedar vacío")
	}
}
