package context

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"localcli/internal/ollama"
)

type modeloStub struct {
	devuelve []string
	err      error
	objetivo string
}

func (m *modeloStub) Seleccionar(ctx context.Context, objetivo string, candidatos []string) ([]string, error) {
	m.objetivo = objetivo
	return m.devuelve, m.err
}

// TestInterpretarSeleccion — solo cuentan las rutas candidatas nombradas.
func TestInterpretarSeleccion(t *testing.T) {
	candidatos := []string{"backend/BACKEND.md", "backend/DECISIONS.md"}
	respuesta := "Necesito:\n- backend/BACKEND.md\n`backend/DECISIONS.md`\n- ruta/inexistente.md\nBACKEND.md\n"
	got := InterpretarSeleccion(respuesta, candidatos)
	quiero := []string{"backend/BACKEND.md", "backend/DECISIONS.md"}
	if !reflect.DeepEqual(got, quiero) {
		t.Fatalf("selección = %v, quiero %v", got, quiero)
	}
}

// TestModeloOllamaSeleccionaPorHTTP — con un servidor falso, el modelo devuelve
// la selección que nombró.
func TestModeloOllamaSeleccionaPorHTTP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			t.Errorf("ruta inesperada %s", r.URL.Path)
		}
		// El cuerpo lleva el objetivo y los candidatos, nunca el contenido.
		var cuerpo map[string]any
		_ = json.NewDecoder(r.Body).Decode(&cuerpo)
		w.Header().Set("Content-Type", "application/x-ndjson")
		_, _ = io.WriteString(w, `{"message":{"content":"backend/BACKEND.md\n"},"done":false}`+"\n")
		_, _ = io.WriteString(w, `{"message":{"content":""},"done":true}`+"\n")
	}))
	defer srv.Close()

	m := &ModeloOllama{Cliente: ollama.NewClient(srv.URL), Modelo: "m"}
	got, err := m.Seleccionar(context.Background(), "objetivo", []string{"backend/BACKEND.md", "backend/DECISIONS.md"})
	if err != nil {
		t.Fatalf("Seleccionar: %v", err)
	}
	if !reflect.DeepEqual(got, []string{"backend/BACKEND.md"}) {
		t.Fatalf("selección = %v", got)
	}
}
