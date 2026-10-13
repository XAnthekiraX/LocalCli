package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// servidorChat responde con el razonamiento y el texto dados, más el uso.
func servidorChat(t *testing.T, razonamiento, texto string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		emite := func(v any) {
			b, _ := json.Marshal(v)
			fmt.Fprintf(w, "data: %s\n\n", b)
			if flusher != nil {
				flusher.Flush()
			}
		}
		emite(map[string]any{"choices": []any{map[string]any{"delta": map[string]any{"role": "assistant"}}}})
		if razonamiento != "" {
			emite(map[string]any{"choices": []any{map[string]any{"delta": map[string]any{"reasoning_content": razonamiento}}}})
		}
		emite(map[string]any{"choices": []any{map[string]any{"delta": map[string]any{"content": texto}}}})
		emite(map[string]any{"choices": []any{}, "usage": map[string]any{"prompt_tokens": 7, "completion_tokens": 3}})
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestChatEntregaRazonamientoYTextoYUso(t *testing.T) {
	srv := servidorChat(t, "pienso", "hola")
	c := &Cliente{baseURL: srv.URL, http: srv.Client()}

	var texto strings.Builder
	var razonamiento strings.Builder
	uso, err := c.Chat(context.Background(), "m", []Mensaje{{Rol: "user", Contenido: "hola"}}, func(d Delta) error {
		if d.Razonamiento {
			razonamiento.WriteString(d.Texto)
			return nil
		}
		texto.WriteString(d.Texto)
		return nil
	})
	if err != nil {
		t.Fatalf("chat: %v", err)
	}
	if texto.String() != "hola" {
		t.Fatalf("texto = %q", texto.String())
	}
	if razonamiento.String() != "pienso" {
		t.Fatalf("razonamiento = %q", razonamiento.String())
	}
	if uso != (Uso{Prompt: 7, Completion: 3}) {
		t.Fatalf("uso = %+v", uso)
	}
}

func TestCompletarEnviaElCanalYRecomponeLasLlamadas(t *testing.T) {
	var cuerpo string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		cuerpo = string(b)
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		emite := func(v any) {
			bb, _ := json.Marshal(v)
			fmt.Fprintf(w, "data: %s\n\n", bb)
			if flusher != nil {
				flusher.Flush()
			}
		}
		// El nombre llega entero en el primer fragmento; los argumentos, troceados.
		emite(map[string]any{"choices": []any{map[string]any{"delta": map[string]any{"tool_calls": []any{
			map[string]any{"index": 0, "id": "call_1", "type": "function", "function": map[string]any{"name": "leer_archivo", "arguments": `{"ru`}},
		}}}}})
		emite(map[string]any{"choices": []any{map[string]any{"delta": map[string]any{"tool_calls": []any{
			map[string]any{"index": 0, "function": map[string]any{"arguments": `ta":"a.go"}`}},
		}}}}})
		emite(map[string]any{"choices": []any{}, "usage": map[string]any{"prompt_tokens": 11, "completion_tokens": 5}})
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	t.Cleanup(srv.Close)

	c := &Cliente{baseURL: srv.URL, http: srv.Client()}
	respuesta, err := c.Completar(context.Background(), "m",
		[]Mensaje{{Rol: "user", Contenido: "lee"}},
		[]Herramienta{{Tipo: "function", Funcion: Definicion{Nombre: "leer_archivo", Descripcion: "lee", Parametros: json.RawMessage(`{"type":"object"}`)}}},
		Opciones{}, nil)
	if err != nil {
		t.Fatalf("completar: %v", err)
	}
	if len(respuesta.Llamadas) != 1 {
		t.Fatalf("llamadas = %+v", respuesta.Llamadas)
	}
	l := respuesta.Llamadas[0]
	if l.ID != "call_1" || l.Funcion.Nombre != "leer_archivo" || l.Funcion.Argumentos != `{"ruta":"a.go"}` {
		t.Fatalf("la llamada no se recompuso: %+v", l)
	}
	if respuesta.Uso != (Uso{Prompt: 11, Completion: 5}) {
		t.Fatalf("uso = %+v", respuesta.Uso)
	}
	if !strings.Contains(cuerpo, `"tools"`) || !strings.Contains(cuerpo, "leer_archivo") {
		t.Fatalf("la petición debería llevar el canal de herramientas: %s", cuerpo)
	}
}

func TestCompletarEnviaLaEleccionDeHerramienta(t *testing.T) {
	var cuerpo string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		cuerpo = string(b)
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	t.Cleanup(srv.Close)

	c := &Cliente{baseURL: srv.URL, http: srv.Client()}
	if _, err := c.Completar(context.Background(), "m", nil,
		[]Herramienta{{Tipo: "function", Funcion: Definicion{Nombre: "leer_archivo"}}},
		Opciones{EleccionHerramienta: "required"}, nil); err != nil {
		t.Fatalf("completar: %v", err)
	}
	if !strings.Contains(cuerpo, `"tool_choice":"required"`) {
		t.Fatalf("la petición debería llevar la elección de herramienta: %s", cuerpo)
	}
}

func TestModelosLeeVentanaYCapacidades(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			http.NotFound(w, r)
			return
		}
		fmt.Fprint(w, `{"data":[{"id":"m.gguf","meta":{"n_ctx":8192}}],"models":[{"model":"m.gguf","capabilities":["completion"]}]}`)
	}))
	t.Cleanup(srv.Close)

	c := &Cliente{baseURL: srv.URL, http: srv.Client()}
	modelos, err := c.Modelos(context.Background())
	if err != nil {
		t.Fatalf("modelos: %v", err)
	}
	if len(modelos) != 1 || modelos[0].ID != "m.gguf" || modelos[0].Ventana != 8192 || modelos[0].Herramientas {
		t.Fatalf("modelos = %+v", modelos)
	}
}

func TestStreamCortadoEsError(t *testing.T) {
	// El servidor cierra la conexión sin enviar `data: [DONE]`.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"a medias\"}}]}\n\n")
	}))
	defer srv.Close()

	c := &Cliente{baseURL: srv.URL, http: srv.Client()}
	_, err := c.Chat(context.Background(), "m", nil, nil)
	if err == nil || !strings.Contains(err.Error(), "cortó") {
		t.Fatalf("un stream sin [DONE] debería ser error: %v", err)
	}
}

func TestErrorDeConexionEsClaro(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	url := srv.URL
	srv.Close()

	c := &Cliente{baseURL: url, http: &http.Client{}}
	_, err := c.Chat(context.Background(), "m", nil, nil)
	if err == nil || !strings.Contains(err.Error(), "no se pudo conectar") {
		t.Fatalf("error = %v", err)
	}
}

func TestModelosLeeVision(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"data":[{"id":"vision.gguf"}],"models":[{"model":"vision.gguf","capabilities":["completion","vision"]}]}`)
	}))
	t.Cleanup(srv.Close)

	c := &Cliente{baseURL: srv.URL, http: srv.Client()}
	modelos, err := c.Modelos(context.Background())
	if err != nil {
		t.Fatalf("modelos: %v", err)
	}
	if len(modelos) != 1 || !modelos[0].Vision {
		t.Fatalf("el modelo debería declarar visión: %+v", modelos)
	}
}

func TestMensajeConImagenViajaComoPartesImageURL(t *testing.T) {
	var cuerpo string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		cuerpo = string(b)
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	t.Cleanup(srv.Close)

	c := &Cliente{baseURL: srv.URL, http: srv.Client()}
	_, err := c.Completar(context.Background(), "m", []Mensaje{{
		Rol:       "user",
		Contenido: "describe esto",
		Imagenes:  []Imagen{{MIME: "image/png", Datos: []byte{0x89, 0x50, 0x4e, 0x47}}},
	}}, nil, Opciones{}, nil)
	if err != nil {
		t.Fatalf("completar: %v", err)
	}
	if !strings.Contains(cuerpo, `"type":"image_url"`) || !strings.Contains(cuerpo, `data:image/png;base64,`) {
		t.Fatalf("la petición debería llevar image_url con data URI: %s", cuerpo)
	}
	if !strings.Contains(cuerpo, `"type":"text"`) || !strings.Contains(cuerpo, "describe esto") {
		t.Fatalf("la petición debería llevar el texto como parte: %s", cuerpo)
	}
}

func TestMensajeSinImagenViajaComoCadena(t *testing.T) {
	datos, err := json.Marshal(Mensaje{Rol: "user", Contenido: "hola"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(datos), `"content":"hola"`) {
		t.Fatalf("sin imágenes el content debería ser una cadena: %s", datos)
	}
}
