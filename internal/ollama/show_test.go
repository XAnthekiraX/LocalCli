package ollama

// Test de las capacidades del modelo (/api/show). LocalCli no usa
// function-calling nativo —las herramientas van como texto—, así que esta
// capacidad solo alimenta el aviso al usuario (SPEC-OLLAMA-PERFIL).

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestCapacidadesLeeLaFichaDelModelo(t *testing.T) {
	var recibido struct {
		Model string `json:"model"`
	}
	c := nuevoServidorOllama(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/show" {
			t.Errorf("ruta inesperada %s", r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&recibido)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"capabilities":["completion","tools","vision"]}`))
	})

	caps, err := c.Capacidades(context.Background(), "llama3.2")
	if err != nil {
		t.Fatal(err)
	}
	if recibido.Model != "llama3.2" {
		t.Errorf("se pregunta por el modelo pedido: %q", recibido.Model)
	}
	if !PuedeUsarHerramientas(caps) {
		t.Errorf("tools está entre las capacidades: %v", caps)
	}
	if PuedeUsarHerramientas([]string{"completion", "vision"}) {
		t.Error("sin tools no puede usar herramientas")
	}
	if PuedeUsarHerramientas(nil) {
		t.Error("sin capacidades no puede usar herramientas")
	}
}
