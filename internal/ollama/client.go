// client.go — T-B005-01 / T-B036-04: cliente HTTP local de Ollama como
// adaptador de `llm.Proveedor`.
//
// Fuente de verdad: ai/docs/backend/04-infrastructure/INTEGRATIONS.md §Ollama.
//   - API HTTP en local, por defecto http://localhost:11434.
//   - Sin credenciales (§4): no se envía ninguna clave.
//   - Streaming obligatorio: Chat abre el stream NDJSON; el parseo vive en
//     stream.go/reasoning.go.
//
// Si el contrato de streaming de Ollama cambiara, solo cambia este módulo: el
// resto del harness habla con `llm.Proveedor`.
package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"localcli/internal/llm"
)

// DirPorDefecto es la dirección por defecto de Ollama (INTEGRATIONS §1).
const DirPorDefecto = "http://localhost:11434"

// Client habla con un servidor Ollama local. Es seguro para uso concurrente
// (http.Client lo es); la serialización de inferencia vive en `llm`.
type Client struct {
	baseURL string
	http    *http.Client
}

// NewClient crea un cliente contra baseURL (sin barra final). Si baseURL está
// vacío usa DirPorDefecto.
func NewClient(baseURL string) *Client {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = DirPorDefecto
	}
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{},
	}
}

// Nombre es la clave del runtime para la frontera neutra.
func (c *Client) Nombre() string { return "ollama" }

// BaseURL devuelve la dirección configurada (visible para tests/TUI).
func (c *Client) BaseURL() string { return c.baseURL }

// VentanaDeContexto: Ollama declara la ventana por petición (`options.num_ctx`),
// así que no la lee del servidor. Devuelve `declarada=true` y una ventana 0: el
// valor efectivo lo calcula el arranque a partir de la lista de modelos y el
// tope.
func (c *Client) VentanaDeContexto(_ context.Context, _ string) (int, bool, error) {
	return 0, true, nil
}

// GenerarRequest es el cuerpo de POST /api/generate o /api/chat en modo
// stream:true. Es el tipo de cable de Ollama: el resto del harness no lo ve.
type GenerarRequest struct {
	Model    string         `json:"model"`
	Prompt   string         `json:"prompt"`
	Stream   bool           `json:"stream"`
	Context  []byte         `json:"context,omitempty"`
	Raw      bool           `json:"raw,omitempty"`
	Options  map[string]any `json:"options,omitempty"`
	Think    any            `json:"think,omitempty"`
	Messages []Mensaje      `json:"messages,omitempty"`
	// Tools es el canal nativo de herramientas de /api/chat.
	Tools []Herramienta `json:"tools,omitempty"`
	// NumCtx fija el tamaño de contexto del modelo (`num_ctx`). 0 = el del
	// servidor. No viaja como campo suelto: se inyecta en `Options`.
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

// peticionDe traduce la petición neutra al cuerpo de `/api/chat` de Ollama. El
// `think` solo viaja si el usuario lo decidió (`Pensar != nil`).
func peticionDe(req llm.Peticion) GenerarRequest {
	r := GenerarRequest{
		Model:    req.Modelo,
		Messages: req.Mensajes,
		Tools:    req.Herramientas,
		NumCtx:   req.NumCtx,
	}
	if req.Pensar != nil {
		r.Think = *req.Pensar
	}
	return r
}

// Generar lanza POST /api/generate con stream:true (variante sin chat).
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

// Chat implementa `llm.Proveedor`: traduce la petición neutra y lanza
// POST /api/chat (stream:true).
func (c *Client) Chat(ctx context.Context, req llm.Peticion) (<-chan Evento, error) {
	cable := peticionDe(req)
	cable.Stream = true
	cable.conNumCtx()
	resp, err := c.post(ctx, "/api/chat", cable)
	if err != nil {
		return nil, err
	}
	salida := make(chan Evento, 64)
	go bombear(ctx, resp.Body, salida)
	return salida, nil
}

// post hace el POST JSON común. Separa fallos de red (→ error tipado) de fallos
// de status HTTP (→ error legible con el mensaje del servidor).
func (c *Client) post(ctx context.Context, ruta string, payload any) (*http.Response, error) {
	cuerpo, err := json.Marshal(payload)
	if err != nil {
		return nil, llm.NuevoErrorNoDisponible("no se pudo codificar la petición al modelo", err.Error())
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+ruta, bytes.NewReader(cuerpo))
	if err != nil {
		return nil, llm.NuevoErrorNoDisponible(mensajeLevantarOllama, err.Error())
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(httpReq)
	if err != nil {
		if tipado, ok := clasificarFalloRed(err); ok {
			return nil, tipado
		}
		return nil, llm.NuevoErrorNoDisponible(mensajeLevantarOllama, err.Error())
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
		return nil, llm.NuevoErrorNoDisponible(mensajeLevantarOllama, err.Error())
	}
	resp, err := c.http.Do(httpReq)
	if err != nil {
		if tipado, ok := clasificarFalloRed(err); ok {
			return nil, tipado
		}
		return nil, llm.NuevoErrorNoDisponible(mensajeLevantarOllama, err.Error())
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errorDeStatus(resp)
	}
	return resp, nil
}

// Ping comprueba si Ollama responde. Devuelve el error tipado o nil.
func (c *Client) Ping(ctx context.Context) error {
	resp, err := c.get(ctx, "/api/tags")
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// timeoutTransport deja constancia del límite defensivo: conexiones nuevas en
// 5 s. No hay timeout de lectura total porque el streaming puede ser largo.
var _ = 5 * time.Second
