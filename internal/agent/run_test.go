package agent

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"localcli/internal/ollama"
)

// TestPeticionLlevaElPromptDelAgente — T-B006-05: lo que se envía al modelo
// lleva el prompt del JSON del agente como mensaje de sistema, antes de los
// turnos de la conversación.
func TestPeticionLlevaElPromptDelAgente(t *testing.T) {
	capturado := make(chan map[string]any, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var cuerpo map[string]any
		_ = json.NewDecoder(r.Body).Decode(&cuerpo)
		capturado <- cuerpo
		w.Header().Set("Content-Type", "application/x-ndjson")
		_, _ = io.WriteString(w, `{"model":"m","response":"hola","done":true}`+"\n")
	}))
	defer srv.Close()

	a := Agente{Nombre: "lector", Prompt: "PROMPT-DEL-JSON", Herramientas: []string{"leer_archivo"}}
	runner := Runner{Cliente: ollama.NewClient(srv.URL)}
	ch, err := runner.Generar(context.Background(), a, "m", []ollama.Mensaje{{Role: "user", Content: "hola"}})
	if err != nil {
		t.Fatalf("Generar: %v", err)
	}
	for range ch {
		// drenamos el stream
	}

	cuerpo := <-capturado
	mensajes, ok := cuerpo["messages"].([]any)
	if !ok || len(mensajes) < 2 {
		t.Fatalf("messages = %v, quiero al menos sistema + usuario", cuerpo["messages"])
	}
	primero := mensajes[0].(map[string]any)
	if primero["role"] != RolSistema || primero["content"] != "PROMPT-DEL-JSON" {
		t.Errorf("primer mensaje = %v, quiero el prompt del agente como sistema", primero)
	}
	segundo := mensajes[1].(map[string]any)
	if segundo["content"] != "hola" {
		t.Errorf("segundo mensaje = %v, quiero el turno del usuario", segundo)
	}
}

// TestConstruirPeticionNoMutaLosMensajes — armar la petición no debe pisar el
// slice del llamador.
func TestConstruirPeticionNoMutaLosMensajes(t *testing.T) {
	mensajes := []ollama.Mensaje{{Role: "user", Content: "hola"}}
	_ = ConstruirPeticion(Agente{Nombre: "x", Prompt: "p"}, "m", mensajes)
	if len(mensajes) != 1 || mensajes[0].Content != "hola" {
		t.Errorf("la petición mutó los mensajes de entrada: %+v", mensajes)
	}
}
