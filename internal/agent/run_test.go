package agent

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"localcli/internal/ollama"
	"localcli/internal/tools"
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
	contenido, _ := primero["content"].(string)
	if primero["role"] != RolSistema || !strings.HasPrefix(contenido, "PROMPT-DEL-JSON") {
		t.Errorf("primer mensaje = %v, quiero el prompt del agente como sistema", primero)
	}
	// El mensaje de sistema se completa con el catálogo del agente.
	if !strings.Contains(contenido, "leer_archivo") {
		t.Errorf("el sistema debe listar las herramientas del agente: %q", contenido)
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

// TestPromptDeSistemaInyectaCatalogoYFormato — el modelo recibe, en el mensaje
// de sistema, sus herramientas y el formato exacto de llamada: sin esto, un
// modelo local sin function-calling nativo no sabe qué puede pedir ni cómo.
func TestPromptDeSistemaInyectaCatalogoYFormato(t *testing.T) {
	a := Agente{Nombre: "plan", Prompt: "Eres plan.", Herramientas: tools.HerramientasDePlan()}
	p := PromptDeSistema(a)

	if !strings.HasPrefix(p, "Eres plan.") {
		t.Errorf("el prompt del agente debe ir primero: %q", p)
	}
	if !strings.Contains(p, "leer_archivo") {
		t.Error("el catálogo inyectado debe listar las herramientas del agente")
	}
	if !strings.Contains(p, "```herramienta") {
		t.Error("el mensaje de sistema debe explicar el formato de llamada")
	}
	if strings.Contains(p, "crear_archivo") {
		t.Error("`plan` no debe ver herramientas de escritura en su catálogo")
	}
}

// TestPromptDeSistemaSoloConversacion — un agente sin herramientas recibe
// únicamente su prompt, sin catálogo ni formato.
func TestPromptDeSistemaSoloConversacion(t *testing.T) {
	a := Agente{Nombre: "charlatan", Prompt: "Conversa sin tocar nada."}
	if got := PromptDeSistema(a); got != "Conversa sin tocar nada." {
		t.Errorf("prompt = %q, quiero solo el prompt del agente", got)
	}
}
