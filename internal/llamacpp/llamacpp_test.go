package llamacpp

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"localcli/internal/llm"
)

func nuevoServidor(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return NewClient(srv.URL)
}

func TestNombreYBaseURL(t *testing.T) {
	if got := NewClient("").BaseURL(); got != DirPorDefecto {
		t.Errorf("vacío → %q, quiero el default", got)
	}
	if got := NewClient("").Nombre(); got != "llamacpp" {
		t.Errorf("Nombre() = %q, quiero llamacpp", got)
	}
	if got := NewClient("http://otro:9999/").BaseURL(); got != "http://otro:9999" {
		t.Errorf("trim barra final: %q", got)
	}
}

// TestElSSEAcumulaToolCallsPorIndice — T-B036-06: una misma llamada puede
// repartirse en varias deltas; hay que acumularlas por índice antes de poder
// ejecutarlas.
func TestElSSEAcumulaToolCallsPorIndice(t *testing.T) {
	c := nuevoServidor(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("ruta inesperada %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		// El texto llega en dos deltas; la tool_call 0 se parte en tres.
		lineas := []string{
			`data: {"model":"m","choices":[{"delta":{"content":"Miro "}}]}`,
			`data: {"model":"m","choices":[{"delta":{"reasoning_content":"pienso"}}]}`,
			`data: {"model":"m","choices":[{"delta":{"content":"el archivo"}}]}`,
			`data: {"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_1","function":{"name":"leer_","arguments":"{\"r"}}]}}]}`,
			`data: {"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"name":"archivo","arguments":"uta\":\"a.md\"}"}}]}}]}`,
			`data: {"choices":[{"finish_reason":"tool_calls"}]}`,
			`data: {"choices":[],"usage":{"prompt_tokens":12,"completion_tokens":4}}`,
			`data: [DONE]`,
		}
		io.WriteString(w, strings.Join(lineas, "\n")+"\n")
	})

	events, err := c.Chat(context.Background(), llm.Peticion{Modelo: "m", Mensajes: []llm.Mensaje{{Role: "user", Content: "hola"}}})
	if err != nil {
		t.Fatal(err)
	}
	var texto, razon strings.Builder
	var final *llm.RespuestaFinal
	for ev := range events {
		switch ev.Tipo {
		case llm.EventoToken:
			texto.WriteString(ev.Texto)
		case llm.EventoRazonamiento:
			razon.WriteString(ev.Texto)
		case llm.EventoDone:
			final = ev.Done
		case llm.EventoError:
			t.Fatalf("error inesperado: %v", ev.Error)
		}
	}
	if texto.String() != "Miro el archivo" {
		t.Errorf("texto=%q", texto.String())
	}
	if razon.String() != "pienso" {
		t.Errorf("razonamiento=%q", razon.String())
	}
	if final == nil || len(final.ToolCalls) != 1 {
		t.Fatalf("tool_calls = %+v, quiero una acumulada", final)
	}
	if final.ToolCalls[0].Nombre() != "leer_archivo" {
		t.Errorf("nombre acumulado = %q", final.ToolCalls[0].Nombre())
	}
	if got := string(final.ToolCalls[0].ArgumentosJSON()); got != `{"ruta":"a.md"}` {
		t.Errorf("argumentos acumulados = %s", got)
	}
	if final.TokensEntr != 12 || final.TokensSal != 4 {
		t.Errorf("usage no leído: %+v", final)
	}
}

// TestLaVentanaSeLeeYNuncaSeMutaElServidor — T-B036-07: la ventana se lee de
// /props; nunca se hace POST /props ni POST /models.
func TestLaVentanaSeLeeYNuncaSeMutaElServidor(t *testing.T) {
	var mu sync.Mutex
	var mutaciones []string
	c := nuevoServidor(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			mu.Lock()
			mutaciones = append(mutaciones, r.Method+" "+r.URL.Path)
			mu.Unlock()
		}
		if r.URL.Path != "/props" {
			t.Errorf("ruta inesperada %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"default_generation_settings":{"n_ctx":8192},"chat_template_caps":{"supports_tool_calls":true}}`))
	})

	n, declarada, err := c.VentanaDeContexto(context.Background(), "m")
	if err != nil {
		t.Fatal(err)
	}
	if n != 8192 {
		t.Errorf("ventana leída = %d, quiero 8192", n)
	}
	if declarada {
		t.Error("llama.cpp lee la ventana: declarada debe ser false")
	}
	// Capacidades también usa /props, sin mutar.
	if _, err := c.Capacidades(context.Background(), "m"); err != nil {
		t.Fatal(err)
	}
	if len(mutaciones) != 0 {
		t.Errorf("el harness nunca muta el servidor; hubo %v", mutaciones)
	}
}

// Si /props no se puede leer, la ventana se trata como desconocida sin error.
func TestVentanaDesconocidaSiPropsFalla(t *testing.T) {
	c := nuevoServidor(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	n, declarada, err := c.VentanaDeContexto(context.Background(), "m")
	if err != nil || n != 0 || declarada {
		t.Errorf("ventana desconocida = (%d,%v,%v)", n, declarada, err)
	}
}

func TestCapacidadesNormalizaProps(t *testing.T) {
	c := nuevoServidor(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"chat_template_caps":{"supports_tool_calls":true,"supports_reasoning_effort":true},"modalities":{"vision":true}}`))
	})
	caps, err := c.Capacidades(context.Background(), "m")
	if err != nil {
		t.Fatal(err)
	}
	if !llm.PuedeUsarHerramientas(caps) || !llm.PuedeVer(caps) || !llm.PuedePensar(caps) {
		t.Errorf("capacidades mal normalizadas: %v", caps)
	}
}

func TestCapacidadesSinFichasEsError(t *testing.T) {
	c := nuevoServidor(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{}`))
	})
	if _, err := c.Capacidades(context.Background(), "m"); err == nil {
		t.Error("sin chat_template_caps ni modalities no se puede afirmar nada")
	}
}

// TestListarModelosConReservaV1 — el router sirve `/models`; si no, `/v1/models`.
func TestListarModelosConReservaV1(t *testing.T) {
	c := nuevoServidor(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/models" {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":"not found"}`))
			return
		}
		if r.URL.Path != "/v1/models" {
			t.Errorf("ruta inesperada %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"data":[{"id":"qwen3:8b","meta":{"n_ctx_train":32768}}]}`))
	})
	modelos, err := c.ListarModelos(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(modelos) != 1 || modelos[0].Nombre != "qwen3:8b" || modelos[0].ContextLength != 32768 {
		t.Errorf("modelos = %+v", modelos)
	}
}

// La imagen base64 sin prefijo se traduce a un `image_url` con data URI.
func TestChatEnviaImagenComoDataURI(t *testing.T) {
	var crudo []byte
	c := nuevoServidor(t, func(w http.ResponseWriter, r *http.Request) {
		crudo, _ = io.ReadAll(r.Body)
		io.WriteString(w, "data: [DONE]\n")
	})
	// PNG de 1x1 en base64.
	const png = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAAC0lEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="
	events, err := c.Chat(context.Background(), llm.Peticion{
		Modelo:   "m",
		Mensajes: []llm.Mensaje{{Role: "user", Content: "mira", Images: []string{png}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	for range events {
	}
	var cuerpo struct {
		Messages []struct {
			Content []struct {
				Type     string `json:"type"`
				ImageURL *struct {
					URL string `json:"url"`
				} `json:"image_url"`
			} `json:"content"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(crudo, &cuerpo); err != nil {
		t.Fatalf("petición no JSON: %v (%s)", err, crudo)
	}
	var ok bool
	for _, p := range cuerpo.Messages[0].Content {
		if p.Type == "image_url" && p.ImageURL != nil && strings.HasPrefix(p.ImageURL.URL, "data:image/png;base64,") {
			ok = true
		}
	}
	if !ok {
		t.Errorf("no viajó la imagen como data URI: %s", crudo)
	}
}
