// client.go — T-B005-01: cliente HTTP local de Ollama.
//
// Fuente de verdad: ai/docs/backend/04-infrastructure/INTEGRATIONS.md §Ollama.
//   - API HTTP en local, por defecto http://localhost:11434.
//   - Sin credenciales (§4): no se envía ninguna clave.
//   - Base URL configurable internamente (para tests con httptest y para
//     usuarios con Ollama en otro puerto), no expuesta como superficie nueva.
//   - Streaming obligatorio: Generate abre el stream; el parseo vive en
//     stream.go/reasoning.go.
//
// Si el contrato de streaming de Ollama cambiara, solo cambia este módulo
// (INTEGRATIONS §5): el resto del harness ve Client, Request y Evento.
package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

// DirPorDefecto es la dirección por defecto de Ollama (INTEGRATIONS §1).
const DirPorDefecto = "http://localhost:11434"

// Client habla con un servidor Ollama local. Es seguro para uso concurrente
// (http.Client lo es); la serialización de inferencia NO vive aquí, sino en
// fifo.go.
type Client struct {
	baseURL string
	http    *http.Client
}

// NewClient crea un cliente contra baseURL (sin barra final). Si baseURL está
// vacío usa DirPorDefecto. El timeout del transporte cubre la recepción
// COMPLETA de la respuesta; para streams largos que pueden durar minutos se
// usa un cliente sin Timeout global y cancelación por context (el contexto es
// el mecanismo idiomático de cancelar generación).
func NewClient(baseURL string) *Client {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = DirPorDefecto
	}
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{},
	}
}

// BaseURL devuelve la dirección configurada (visible para tests/TUI).
func (c *Client) BaseURL() string { return c.baseURL }

// GenerarRequest es el cuerpo de POST /api/generate en modo stream:true.
// Campos según la API pública de Ollama; Options admite think para modelos
// con razonamiento separable (reasoning.go).
type GenerarRequest struct {
	Model    string         `json:"model"`
	Prompt   string         `json:"prompt"`
	Stream   bool           `json:"stream"`
	Context  []byte         `json:"context,omitempty"`  // reanudar conversación
	Raw      bool           `json:"raw,omitempty"`      // sin plantilla de chat
	Options  map[string]any `json:"options,omitempty"`  // num_ctx, temperature…
	Think    any            `json:"think,omitempty"`    // true/false/"low"/"medium"/"high"
	Messages []Mensaje      `json:"messages,omitempty"` // variante /api/chat
}

// Mensaje es un turno de /api/chat (se reutiliza en el contrato de agent).
type Mensaje struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// RespuestaFinal resume lo que queda al terminar un stream: texto completo,
// razonamiento completo y métricas para el nodo de contexto (tokens usados…).
type RespuestaFinal struct {
	Model      string `json:"model"`
	Texto      string `json:"response"`
	Razonamien string `json:"thinking"`
	Done       bool   `json:"done"`
	TokensEntr uint64 `json:"prompt_eval_count,omitempty"`
	TokensSal  uint64 `json:"eval_count,omitempty"`
	DuracionNs uint64 `json:"total_duration,omitempty"`
}

// Generar lanza POST /api/generate con stream:true y devuelve los eventos por
// el canal hasta done o error. El canal se cierra SIEMPRE (invariantes de
// stream.go). Errores de conexión llegan como *ErrorOllama con código
// E_OLLAMA_UNAVAILABLE (errors.Is(ErrOllamaNoDisponible)); nunca panic.
func (c *Client) Generar(ctx context.Context, req GenerarRequest) (<-chan Evento, error) {
	req.Stream = true
	resp, err := c.post(ctx, "/api/generate", req)
	if err != nil {
		return nil, err
	}
	salida := make(chan Evento, 64)
	go bombear(ctx, resp.Body, salida)
	return salida, nil
}

// Chat lanza POST /api/chat (stream:true) con historial de mensajes. Mismo
// contrato de eventos y errores que Generar.
func (c *Client) Chat(ctx context.Context, req GenerarRequest) (<-chan Evento, error) {
	req.Stream = true
	resp, err := c.post(ctx, "/api/chat", req)
	if err != nil {
		return nil, err
	}
	salida := make(chan Evento, 64)
	go bombear(ctx, resp.Body, salida)
	return salida, nil
}

// post hace el POST JSON común. Separa fallos de red (→ error tipado) de
// fallos de status HTTP (→ error legible con el mensaje del servidor).
func (c *Client) post(ctx context.Context, ruta string, payload any) (*http.Response, error) {
	cuerpo, err := json.Marshal(payload)
	if err != nil {
		return nil, &ErrorOllama{
			Codigo:   CodigoOllamaNoDisponible,
			Mensaje:  "no se pudo codificar la petición al modelo",
			Detalle:  err.Error(),
			sentinel: ErrOllamaNoDisponible,
		}
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+ruta, bytes.NewReader(cuerpo))
	if err != nil {
		return nil, &ErrorOllama{
			Codigo:   CodigoOllamaNoDisponible,
			Mensaje:  mensajeLevantarOllama,
			Detalle:  err.Error(),
			sentinel: ErrOllamaNoDisponible,
		}
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(httpReq)
	if err != nil {
		if tipado, ok := clasificarFalloRed(err); ok {
			return nil, tipado
		}
		return nil, &ErrorOllama{
			Codigo:   CodigoOllamaNoDisponible,
			Mensaje:  mensajeLevantarOllama,
			Detalle:  err.Error(),
			sentinel: ErrOllamaNoDisponible,
		}
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, &ErrorOllama{
			Codigo:  CodigoOllamaNoDisponible,
			Mensaje: "Ollama respondió con un error (" + itob(int64(resp.StatusCode)) + ")",
			Detalle: http.StatusText(resp.StatusCode),
		}
	}
	return resp, nil
}

// get hace un GET simple (para /api/tags en models.go).
func (c *Client) get(ctx context.Context, ruta string) (*http.Response, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+ruta, nil)
	if err != nil {
		return nil, &ErrorOllama{Codigo: CodigoOllamaNoDisponible, Mensaje: mensajeLevantarOllama, Detalle: err.Error(), sentinel: ErrOllamaNoDisponible}
	}
	resp, err := c.http.Do(httpReq)
	if err != nil {
		if tipado, ok := clasificarFalloRed(err); ok {
			return nil, tipado
		}
		return nil, &ErrorOllama{Codigo: CodigoOllamaNoDisponible, Mensaje: mensajeLevantarOllama, Detalle: err.Error(), sentinel: ErrOllamaNoDisponible}
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, &ErrorOllama{Codigo: CodigoOllamaNoDisponible, Mensaje: "Ollama respondió con un error (" + itob(int64(resp.StatusCode)) + ")"}
	}
	return resp, nil
}

// Ping comprueba si Ollama responde (SPEC-OLLAMA-PERFIL, flujo alternativo
// "Ollama no está corriendo: avisa y explica cómo levantarlo"). Devuelve el
// error tipado correspondiente o nil.
func (c *Client) Ping(ctx context.Context) error {
	resp, err := c.get(ctx, "/api/tags")
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// timeoutTransport deja constancia del límite defensivo: conexiones nuevas en
// 5 s. No hay timeout de lectura total porque el streaming puede ser largo;
// la cancelación manda por ctx.
var _ = 5 * time.Second
