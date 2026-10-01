// tipos.go — T-B036-01: los tipos neutros del contrato con el modelo.
//
// Movidos desde `ollama` para que `agent` y `context` no dependan del adaptador
// de un runtime. La forma de cada tipo es la que el harness entiende; cómo la
// serializa cada proveedor es asunto de su adaptador.
package llm

import (
	"encoding/json"
	"strings"
)

// --- Eventos de streaming --------------------------------------------------

// TipoEvento distingue qué lleva un Evento.
type TipoEvento int

const (
	EventoToken        TipoEvento = iota // token de texto final
	EventoRazonamiento                   // token de razonamiento
	EventoDone                           // último evento: respuesta completa + métricas
	EventoError                          // fallo del stream; cierra el canal
)

// Evento es lo que consume la TUI y el agente. Texto trae el fragmento; Done
// trae la RespuestaFinal acumulada.
type Evento struct {
	Tipo  TipoEvento
	Texto string
	Done  *RespuestaFinal
	Error error
}

// --- Mensajes y herramientas ----------------------------------------------

// Mensaje es un turno de la conversación. Images lleva imágenes ya codificadas
// en base64 SIN prefijo `data:`; cada adaptador las traduce al formato de su
// runtime (Ollama las manda en `images`, llama.cpp como `image_url` con data
// URI). ToolCalls son las peticiones de herramienta del turno; un mensaje de
// rol `tool` lleva el resultado de una.
type Mensaje struct {
	Role      string     `json:"role"`
	Content   string     `json:"content"`
	Images    []string   `json:"images,omitempty"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
	ToolName  string     `json:"tool_name,omitempty"`
}

// Herramienta es una definición de herramienta: `{type: "function", function:
// {name, description, parameters}}`.
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
// crudo porque los runtimes lo entregan de formas distintas y algunos modelos lo
// mandan como cadena: `ArgumentosJSON` normaliza las dos formas.
type ToolCall struct {
	Function struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	} `json:"function"`
}

// Nombre devuelve la herramienta pedida.
func (t ToolCall) Nombre() string { return t.Function.Name }

// ArgumentosJSON normaliza los argumentos a JSON crudo. Acepta tanto un objeto
// (la forma habitual) como la cadena que algunos modelos producen.
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

// --- Modelos ---------------------------------------------------------------

// Modelo es una entrada de la lista del proveedor. SizeBytes puede ser 0 si el
// servidor no lo reporta; context_length puede ser 0 si no declara ventana.
type Modelo struct {
	Nombre        string
	Familia       string
	Parametros    string
	TamanoBytes   int64
	ContextLength int
}

// --- Petición neutra -------------------------------------------------------

// Peticion es lo que `agent` construye y la frontera entrega al adaptador. Es
// lo que antes era `ollama.GenerarRequest` sin los campos que solo tienen
// sentido para un runtime: el adaptador decide cómo la traduce.
//
//   - Modelo y Mensajes son el contenido de la conversación.
//   - Herramientas son las definiciones que el modelo puede pedir.
//   - NumCtx es la ventana de contexto a pedir cuando el proveedor la declara
//     por petición (Ollama); un proveedor que la lee del servidor la ignora.
//   - Pensar es el interruptor de razonamiento: nil NO manda nada (sin ficha no
//     se decide), y un puntero manda ese valor a quien sepa entenderlo.
type Peticion struct {
	Modelo       string
	Mensajes     []Mensaje
	Herramientas []Herramienta
	NumCtx       int
	Pensar       *bool
}
