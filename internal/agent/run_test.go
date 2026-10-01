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

// TestPeticionLlevaElPromptYElCanalDeHerramientas — lo que se envía al modelo
// lleva el prompt del agente como mensaje de sistema y sus herramientas en el
// campo `tools`, no dentro del prompt.
func TestPeticionLlevaElPromptYElCanalDeHerramientas(t *testing.T) {
	capturado := make(chan map[string]any, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var cuerpo map[string]any
		_ = json.NewDecoder(r.Body).Decode(&cuerpo)
		capturado <- cuerpo
		w.Header().Set("Content-Type", "application/x-ndjson")
		_, _ = io.WriteString(w, `{"model":"m","message":{"content":"hola"},"done":true}`+"\n")
	}))
	defer srv.Close()

	reg := registroStub(nil)
	defs := NuevoDespachador(reg).Definiciones(Agente{
		Nombre:      "lector",
		Permissions: map[string]string{"read": "allow"},
	})
	a := Agente{Nombre: "lector", Prompt: "PROMPT-DEL-JSON"}
	runner := Runner{Cliente: ollama.NewClient(srv.URL)}
	ch, err := runner.Generar(context.Background(), a, "m", []ollama.Mensaje{{Role: "user", Content: "hola"}}, defs, 0, nil)
	if err != nil {
		t.Fatalf("Generar: %v", err)
	}
	for range ch {
	}

	cuerpo := <-capturado
	mensajes, ok := cuerpo["messages"].([]any)
	if !ok || len(mensajes) < 2 {
		t.Fatalf("messages = %v", cuerpo["messages"])
	}
	primero := mensajes[0].(map[string]any)
	contenido, _ := primero["content"].(string)
	if primero["role"] != RolSistema || contenido != "PROMPT-DEL-JSON" {
		t.Errorf("primer mensaje = %v, quiero el prompt del agente como sistema", primero)
	}
	// El catálogo ya NO se inyecta en el mensaje de sistema.
	if strings.Contains(contenido, "leer_archivo") {
		t.Errorf("el prompt no puede llevar el catálogo: %q", contenido)
	}
	// Viaja por el canal de herramientas: un objeto por herramienta.
	herramientas, ok := cuerpo["tools"].([]any)
	if !ok || len(herramientas) != len(tools.HerramientasDePlan()) {
		t.Fatalf("tools = %v, quiero las herramientas de lectura del agente", cuerpo["tools"])
	}
	primeraHerr, _ := herramientas[0].(map[string]any)
	if primeraHerr["type"] != "function" {
		t.Errorf("cada definición es una función: %v", primeraHerr)
	}
}

// TestPromptNoLlevaCatalogo — T-B024-15: el prompt del agente se dedica a lo
// que el agente es; las herramientas van por su canal.
func TestPromptNoLlevaCatalogo(t *testing.T) {
	a := Agente{
		Nombre:       "plan",
		Prompt:       "Eres plan.",
		Permissions:  map[string]string{"read": "allow"},
		Herramientas: tools.HerramientasDePlan(),
	}
	p := PromptDeSistema(a)
	if p != "Eres plan." {
		t.Errorf("prompt = %q, quiero solo el prompt del agente", p)
	}
	for _, nombre := range tools.HerramientasDePlan() {
		if strings.Contains(p, nombre) {
			t.Errorf("el prompt no puede listar %s", nombre)
		}
	}
	if strings.Contains(p, "```herramienta") {
		t.Error("no hay formato en prosa que imitar")
	}
}

// TestConstruirPeticionNoMutaLosMensajes — armar la petición no debe pisar el
// slice del llamador.
func TestConstruirPeticionNoMutaLosMensajes(t *testing.T) {
	mensajes := []ollama.Mensaje{{Role: "user", Content: "hola"}}
	_ = ConstruirPeticion(Agente{Nombre: "x", Prompt: "p"}, "m", mensajes, nil, 0, nil)
	if len(mensajes) != 1 || mensajes[0].Content != "hola" {
		t.Errorf("la petición mutó los mensajes de entrada: %+v", mensajes)
	}
}

// TestConstruirPeticionLlevaLaVentana — la ventana de contexto (num_ctx) viaja
// en la petición al modelo: sin ella, Ollama usa un valor pequeño y corta los
// turnos con herramientas.
func TestConstruirPeticionLlevaLaVentana(t *testing.T) {
	req := ConstruirPeticion(Agente{Nombre: "x", Prompt: "p"}, "m", nil, nil, 16384, nil)
	if req.NumCtx != 16384 {
		t.Errorf("NumCtx = %d, quiero 16384", req.NumCtx)
	}
}

// Sin decisión sobre el razonamiento, el campo `think` NO se manda: Ollama
// decide, que es lo que quiere un modelo que no razona.
func TestConstruirPeticionSinRazonamientoNoMandaThink(t *testing.T) {
	req := ConstruirPeticion(Agente{Nombre: "x", Prompt: "p"}, "m", nil, nil, 0, nil)
	bruto, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(bruto), "think") {
		t.Errorf("sin decisión no viaja `think`: %s", bruto)
	}
}

// Con decisión, el valor viaja tal cual: es lo que apaga el razonamiento de un
// modelo local, que Ollama deja encendido por defecto.
func TestConstruirPeticionMandaElRazonamientoDecidido(t *testing.T) {
	no, si := false, true
	casos := []struct {
		nombre string
		pensar *bool
		quiere string
	}{
		{"apagado", &no, `"think":false`},
		{"encendido", &si, `"think":true`},
	}
	for _, tc := range casos {
		req := ConstruirPeticion(Agente{Nombre: "x", Prompt: "p"}, "m", nil, nil, 0, tc.pensar)
		bruto, err := json.Marshal(req)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(bruto), tc.quiere) {
			t.Errorf("%s: %s debe llevar %s", tc.nombre, bruto, tc.quiere)
		}
	}
}

// TestRegresionElTurnoMandaLaVentana — reproduce el fallo real: Ollama responde
// 500 `no user query found in messages` cuando la petición no declara `num_ctx`
// (usa su contexto por defecto, demasiado pequeño para un turno con
// herramientas). El bucle debe mandar la ventana y el turno debe completar.
func TestRegresionElTurnoMandaLaVentana(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var cuerpo struct {
			Options map[string]any `json:"options"`
		}
		_ = json.NewDecoder(r.Body).Decode(&cuerpo)
		if _, ok := cuerpo.Options["num_ctx"]; !ok {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = io.WriteString(w, `{"error":"no user query found in messages"}`)
			return
		}
		w.Header().Set("Content-Type", "application/x-ndjson")
		_, _ = io.WriteString(w, `{"model":"m","message":{"content":"ok"},"done":true}`+"\n")
	}))
	defer srv.Close()

	e := &Ejecutor{Runner: Runner{Cliente: ollama.NewClient(srv.URL)}}
	ag := Agente{Nombre: "plan", Prompt: "p"}

	// Sin ventana, el servidor simula el corte por defecto de Ollama.
	if _, err := e.Ejecutar(context.Background(), ag, "m", "ctx", nil, nil, 0, false, nil); err == nil {
		t.Fatal("sin num_ctx debe aparecer el corte simulado de Ollama")
	}
	// Con ventana, el turno completa igual que un modelo sin herramientas.
	res, err := e.Ejecutar(context.Background(), ag, "m", "ctx", nil, nil, 16384, false, nil)
	if err != nil {
		t.Fatalf("con num_ctx no debe fallar: %v", err)
	}
	if res.Texto != "ok" {
		t.Errorf("texto = %q, quiero ok", res.Texto)
	}
}
