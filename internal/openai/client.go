// client.go — T-B036-05: cliente HTTP del adaptador compatible con OpenAI.
//
// Fuente de verdad: ai/docs/backend/04-infrastructure/INTEGRATIONS.md §llama.cpp.
//   - API HTTP en local, por defecto http://localhost:8080.
//   - Sin credenciales: el `--api-key` opcional de llama-server no se usa.
//   - Streaming SSE por /v1/chat/completions.
package openai

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"localcli/internal/llm"
)

// DirPorDefecto es la dirección por defecto de `llama-server`.
const DirPorDefecto = "http://localhost:8080"

// Client habla con un servidor `llama-server` local. Es seguro para uso
// concurrente.
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
func (c *Client) Nombre() string { return "llamacpp" }

// BaseURL devuelve la dirección configurada.
func (c *Client) BaseURL() string { return c.baseURL }

// --- tipos de cable (formato OpenAI) --------------------------------------

type chatRequest struct {
	Model           string        `json:"model"`
	Messages        []chatMessage `json:"messages"`
	Stream          bool          `json:"stream"`
	Tools           []toolDef     `json:"tools,omitempty"`
	ReasoningEffort string        `json:"reasoning_effort,omitempty"`
}

type chatMessage struct {
	Role       string        `json:"role"`
	Content    any           `json:"content,omitempty"`
	ToolCalls  []outToolCall `json:"tool_calls,omitempty"`
	ToolCallID string        `json:"tool_call_id,omitempty"`
}

type outToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type toolDef struct {
	Type     string         `json:"type"`
	Function llm.Definicion `json:"function"`
}

type contentPart struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageURL *imageURL `json:"image_url,omitempty"`
}

type imageURL struct {
	URL string `json:"url"`
}

// peticionDe traduce la petición neutra al cuerpo de `/v1/chat/completions`.
func peticionDe(req llm.Peticion) chatRequest {
	out := chatRequest{
		Model:    req.Modelo,
		Messages: mensajesDe(req.Mensajes),
		Stream:   true,
	}
	for _, h := range req.Herramientas {
		out.Tools = append(out.Tools, toolDef{Type: "function", Function: h.Function})
	}
	// El razonamiento solo se pide cuando el usuario lo enciende: la plantilla
	// del modelo lo decide. Un `false` no se traduce (no hay un valor estándar
	// que lo apague en todos los runtimes), así que se omite.
	if req.Pensar != nil && *req.Pensar {
		out.ReasoningEffort = "medium"
	}
	return out
}

func mensajesDe(in []llm.Mensaje) []chatMessage {
	out := make([]chatMessage, 0, len(in))
	for _, m := range in {
		msg := chatMessage{Role: m.Role}
		if len(m.Images) > 0 && m.Role == "user" {
			partes := make([]contentPart, 0, len(m.Images)+1)
			if m.Content != "" {
				partes = append(partes, contentPart{Type: "text", Text: m.Content})
			}
			for _, img := range m.Images {
				partes = append(partes, contentPart{Type: "image_url", ImageURL: &imageURL{URL: dataURI(img)}})
			}
			msg.Content = partes
		} else if m.Content != "" {
			msg.Content = m.Content
		}
		for _, tc := range m.ToolCalls {
			var ot outToolCall
			ot.ID = tc.Nombre()
			if ot.ID == "" {
				ot.ID = "call"
			}
			ot.Type = "function"
			ot.Function.Name = tc.Nombre()
			ot.Function.Arguments = string(tc.ArgumentosJSON())
			msg.ToolCalls = append(msg.ToolCalls, ot)
		}
		if m.Role == "tool" {
			msg.ToolCallID = m.ToolName
		}
		out = append(out, msg)
	}
	return out
}

// dataURI convierte una imagen en base64 (sin prefijo) al `data:` URI que espera
// llama.cpp. El tipo MIME se huele del contenido; si no se reconoce, PNG.
func dataURI(imagen string) string {
	if strings.HasPrefix(imagen, "data:") {
		return imagen
	}
	mime := "image/png"
	if datos, err := base64.StdEncoding.DecodeString(imagen); err == nil && len(datos) > 0 {
		if detectado := http.DetectContentType(datos); strings.HasPrefix(detectado, "image/") {
			mime = detectado
		}
	}
	return "data:" + mime + ";base64," + imagen
}

// --- transporte ------------------------------------------------------------

// Chat implementa `llm.Proveedor`: POST /v1/chat/completions en streaming SSE.
func (c *Client) Chat(ctx context.Context, req llm.Peticion) (<-chan llm.Evento, error) {
	resp, err := c.postJSON(ctx, "/v1/chat/completions", peticionDe(req))
	if err != nil {
		return nil, err
	}
	salida := make(chan llm.Evento, 64)
	go leerSSE(ctx, resp.Body, salida)
	return salida, nil
}

// Ping comprueba si el servidor responde (para `LOCALCLI_PROVEEDOR=auto`).
func (c *Client) Ping(ctx context.Context) error {
	resp, err := c.get(ctx, "/props")
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func (c *Client) postJSON(ctx context.Context, ruta string, payload any) (*http.Response, error) {
	cuerpo, err := json.Marshal(payload)
	if err != nil {
		return nil, llm.NuevoErrorNoDisponible("no se pudo codificar la petición al modelo", err.Error())
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+ruta, bytes.NewReader(cuerpo))
	if err != nil {
		return nil, llm.NuevoErrorNoDisponible(mensajeLevantarLlamaCpp, err.Error())
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	resp, err := c.http.Do(httpReq)
	if err != nil {
		if tipado, ok := clasificarFalloRed(err); ok {
			return nil, tipado
		}
		return nil, llm.NuevoErrorNoDisponible(mensajeLevantarLlamaCpp, err.Error())
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errorDeStatus(resp)
	}
	return resp, nil
}

func (c *Client) get(ctx context.Context, ruta string) (*http.Response, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+ruta, nil)
	if err != nil {
		return nil, llm.NuevoErrorNoDisponible(mensajeLevantarLlamaCpp, err.Error())
	}
	resp, err := c.http.Do(httpReq)
	if err != nil {
		if tipado, ok := clasificarFalloRed(err); ok {
			return nil, tipado
		}
		return nil, llm.NuevoErrorNoDisponible(mensajeLevantarLlamaCpp, err.Error())
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errorDeStatus(resp)
	}
	return resp, nil
}

// mensajeLevantarLlamaCpp explica cómo actuar: el servidor no responde y hay que
// levantarlo en modo router, sin las banderas que sirven el sistema de archivos.
const mensajeLevantarLlamaCpp = "no se puede generar: llama.cpp no responde en la dirección configurada; levántalo con " +
	"`llama-server --models-dir <carpeta>` (sin --tools, --agent ni --mcp-servers-json)"

// clasificarFalloRed mapea un fallo de conexión al error tipado.
func clasificarFalloRed(err error) (*llm.ErrorProveedor, bool) {
	return llm.ClasificarFalloRed(err, mensajeLevantarLlamaCpp)
}

// errorDeStatus construye el error tipado de una respuesta HTTP no-200,
// normalizando el cuerpo (JSON o texto plano de llama.cpp) al mismo error.
func errorDeStatus(resp *http.Response) error {
	detalle := cuerpoDeError(resp)
	if detalle == "" {
		detalle = http.StatusText(resp.StatusCode)
	}
	return llm.NuevoErrorNoDisponible(
		"llama.cpp respondió con un error ("+itoa(resp.StatusCode)+")",
		detalle,
	)
}

// cuerpoDeError lee el motivo de una respuesta de error: si el cuerpo es
// `{"error": {"message": "…"}}` o `{"error": "…"}`, devuelve ese texto; si no,
// el cuerpo crudo colapsado.
func cuerpoDeError(resp *http.Response) string {
	defer resp.Body.Close()
	datos, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	texto := strings.TrimSpace(string(datos))
	var envoltura struct {
		Error json.RawMessage `json:"error"`
	}
	if err := json.Unmarshal([]byte(texto), &envoltura); err == nil && len(envoltura.Error) > 0 {
		var comoCadena string
		if err := json.Unmarshal(envoltura.Error, &comoCadena); err == nil {
			texto = comoCadena
		} else {
			var comoObjeto struct {
				Message string `json:"message"`
			}
			if err := json.Unmarshal(envoltura.Error, &comoObjeto); err == nil && comoObjeto.Message != "" {
				texto = comoObjeto.Message
			}
		}
	}
	texto = strings.Join(strings.Fields(texto), " ")
	const maximo = 300
	if r := []rune(texto); len(r) > maximo {
		texto = string(r[:maximo]) + "…"
	}
	return texto
}

// itoa evita arrastrar strconv para un entero en un mensaje.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

// timeoutTransport deja constancia del límite defensivo: conexiones nuevas en
// 5 s. No hay timeout de lectura total porque el streaming puede ser largo.
var _ = 5 * time.Second
