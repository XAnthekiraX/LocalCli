package internal_test

import (
	"os"
	"path/filepath"
	"testing"
)

// modulos son los quince módulos del motor documentados en BACKEND.md §2: los
// trece originales más `llm` (la frontera neutra) y `openai` (el adaptador de
// llama.cpp) que introduce T-B036.
var modulos = []string{
	"tui", "session", "flow", "queue", "context", "agent", "ollama",
	"openai", "llm",
	"task", "tools", "fileops", "exec", "store", "docs",
}

// TestEstructuraInternal verifica que existen los 15 paquetes bajo internal/
// y que cada uno declara su límite en doc.go (T-B001).
func TestEstructuraInternal(t *testing.T) {
	if len(modulos) != 15 {
		t.Fatalf("se esperan 15 módulos, lista tiene %d", len(modulos))
	}
	for _, m := range modulos {
		doc := filepath.Join(m, "doc.go")
		f, err := os.ReadFile(doc)
		if err != nil {
			t.Errorf("falta internal/%s/doc.go: %v", m, err)
			continue
		}
		if len(f) == 0 {
			t.Errorf("internal/%s/doc.go está vacío", m)
		}
	}
}
