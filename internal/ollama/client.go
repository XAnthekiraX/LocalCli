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
	"io"
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
	// Tools es el canal nativo de herramientas de /api/chat: las definiciones
	// —nombre, descripción y esquema de argumentos— que el modelo puede pedir.
	// El catálogo NO viaja en el mensaje de sistema (SPEC-TOOLS).
	Tools []Herramienta `json:"tools,omitempty"`
	// NumCtx fija el tamaño de contexto del modelo (`num_ctx`). 0 = el del
	// servidor, que es pequeño y corta los turnos con herramientas (window.go).
	// No viaja como campo suelto: se inyecta en `Options` antes del POST.
	NumCtx int `json:"-"`
}

// conNumCtx pasa NumCtx a `options.num_ctx`, que es donde Ollama lo espera.
func (req *GenerarRequest) conNumCtx() {
	if req.NumCtx <= 0 {
		return
	}
	if req.Options == nil {
		req.Options = map[string]any{}
	}
	req.Options["num_ctx"] = req.NumCtx
}

// Mensaje es un turno de /api/chat (se reutiliza en el contrato de agent).
// Images lleva imágenes ya codificadas en base64 (sin prefijo `data:`), que es
// la forma que espera /api/chat para los modelos multimodales. El harness no
// interpreta el contenido: es un []string opaco que solo este módulo entiende.
//
// ToolCalls son las peticiones de herramienta del turno; un mensaje de rol
// `tool` lleva el resultado de una. Ninguno de los dos se persiste: solo viven
// en el bucle del turno (DECISIONS.md).
type Mensaje struct {
	Role      string     `json:"role"`
	Content   string     `json:"content"`
	Images    []string   `json:"images,omitempty"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
	ToolName  string     `json:"tool_name,omitempty"`
}

// Herramienta es una definición de herramienta en el formato que espera
// /api/chat: `{type: "function", function: {name, description, parameters}}`.
type Herramienta struct {
	Type     string     `json:"type"`
	Function Definicion `json:"function"`
}

// Definicion describe una función para el modelo.
type Definicion struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Parameters  any    `json:"parameters,omitempty"`
}

// ToolCall es una petición de herramienta del modelo. `Arguments` es un JSON
// crudo porque Ollama lo entrega como objeto y algunos modelos lo mandan como
// cadena: `ArgumentosJSON` normaliza las dos formas.
type ToolCall struct {
	Function struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	} `json:"function"`
}

// Nombre devuelve la herramienta pedida.
func (t ToolCall) Nombre() string { return t.Function.Name }

// ArgumentosJSON normaliza los argumentos a JSON crudo. Acepta tanto un objeto
// (la forma habitual de Ollama) como la cadena que algunos modelos producen.
func (t ToolCall) ArgumentosJSON() json.RawMessage {
	a := t.Function.Arguments
	trim := strings.TrimSpace(string(a))
	if len(trim) >= 2 && trim[0] == '"' && trim[len(trim)-1] == '"' {
		var s string
		if err := json.Unmarshal(a, &s); err == nil {
			if strings.TrimSpace(s) == "" {
				return json.RawMessage("{}")
			}
			return json.RawMessage(s)
		}
	}
	if trim == "" {
		return json.RawMessage("{}")
	}
	return a
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
	// ToolCalls son las herramientas que el modelo pidió en este turno. Se
	// acumulan durante el streaming y llegan con la señal de fin.
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

// Generar lanza POST /api/generate con stream:true y devuelve los eventos por
// el canal hasta done o error. El canal se cierra SIEMPRE (invariantes de
// stream.go). Errores de conexión llegan como *ErrorOllama con código
// E_OLLAMA_UNAVAILABLE (errors.Is(ErrOllamaNoDisponible)); nunca panic.
func (c *Client) Generar(ctx context.Context, req GenerarRequest) (<-chan Evento, error) {
	req.Stream = true
	req.conNumCtx()
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
	req.conNumCtx()
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
		return nil, errorDeStatus(resp)
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
		return nil, errorDeStatus(resp)
	}
	return resp, nil
}

// errorDeStatus construye el error tipado de una respuesta HTTP no-200. Lee el
// cuerpo de Ollama —que trae el motivo real en `{"error": …}`— y lo deja en el
// Detalle: sin él, un fallo como «no user query found in messages» llegaba a la
// pantalla como un «(500)» pelado. El cuerpo se acota para no arrastrar una
// respuesta entera a un mensaje.
func errorDeStatus(resp *http.Response) error {
	detalle := cuerpoDeError(resp)
	if detalle == "" {
		detalle = http.StatusText(resp.StatusCode)
	}
	return &ErrorOllama{
		Codigo:  CodigoOllamaNoDisponible,
		Mensaje: "Ollama respondió con un error (" + itob(int64(resp.StatusCode)) + ")",
		Detalle: detalle,
	}
}

// cuerpoDeError lee el motivo de una respuesta de error: si el cuerpo es el
// `{"error": "…"}` de Ollama, devuelve ese texto; si no, el cuerpo crudo
// colapsado. Vacío si no hay nada legible.
func cuerpoDeError(resp *http.Response) string {
	defer resp.Body.Close()
	datos, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	texto := strings.TrimSpace(string(datos))
	var envoltura struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal([]byte(texto), &envoltura); err == nil && envoltura.Error != "" {
		texto = envoltura.Error
	}
	texto = strings.Join(strings.Fields(texto), " ")
	const maximo = 300
	if r := []rune(texto); len(r) > maximo {
		texto = string(r[:maximo]) + "…"
	}
	return texto
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
