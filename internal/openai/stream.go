// stream.go — T-B036-06: parser SSE con acumulación de tool_calls por índice.
//
// Fuente de verdad: ai/docs/backend/04-infrastructure/INTEGRATIONS.md §llama.cpp
// («SSE, con `data: {…}` y un `data: [DONE]` que cierra. El razonamiento llega
// en `delta.reasoning_content` y el texto en `delta.content`. Las llamadas de
// herramienta llegan en `delta.tool_calls` por índice: una misma llamada puede
// repartirse en varias deltas, así que hay que acumularlas por su índice»).
//
// El SSE se lee con `bufio` de la biblioteca estándar: sin dependencias nuevas.
package openai

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"sort"
	"strings"

	"localcli/internal/llm"
)

// trozo es la forma cruda de una línea `data:` del stream. Un chunk puede traer
// texto, razonamiento o un fragmento de tool_calls (varias deltas por índice).
type trozo struct {
	Model   string `json:"model"`
	Choices []struct {
		Delta struct {
			Content          string `json:"content"`
			ReasoningContent string `json:"reasoning_content"`
			ToolCalls        []struct {
				Index    int    `json:"index"`
				ID       string `json:"id"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"delta"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens     int64 `json:"prompt_tokens"`
		CompletionTokens int64 `json:"completion_tokens"`
	} `json:"usage"`
}

// toolAcum acumula las deltas de una llamada de herramienta repartida en varios
// chunks, identificada por su índice.
type toolAcum struct {
	id     string
	nombre strings.Builder
	args   strings.Builder
}

// acumulador reconstruye la respuesta final a partir de los chunks, en orden.
type acumulador struct {
	modelo string
	texto  strings.Builder
	razon  strings.Builder
	tools  map[int]*toolAcum
	tokens struct {
		entrada int64
		salida  int64
	}
}

func nuevoAcumulador() *acumulador { return &acumulador{tools: map[int]*toolAcum{}} }

// procesarSSE convierte una línea de datos en cero o más eventos de texto y
// actualiza el acumulador. Se expone como función pura para poder testear el
// parseo sin HTTP.
func procesarSSE(datos []byte, acc *acumulador) ([]llm.Evento, error) {
	if len(datos) == 0 {
		return nil, nil
	}
	var t trozo
	if err := json.Unmarshal(datos, &t); err != nil {
		return nil, llm.NuevoErrorNoDisponible("respuesta ilegible del modelo", err.Error())
	}
	if t.Model != "" {
		acc.modelo = t.Model
	}
	if t.Usage != nil {
		acc.tokens.entrada = t.Usage.PromptTokens
		acc.tokens.salida = t.Usage.CompletionTokens
	}
	var evs []llm.Evento
	for _, ch := range t.Choices {
		if ch.Delta.ReasoningContent != "" {
			acc.razon.WriteString(ch.Delta.ReasoningContent)
			evs = append(evs, llm.Evento{Tipo: llm.EventoRazonamiento, Texto: ch.Delta.ReasoningContent})
		}
		if ch.Delta.Content != "" {
			acc.texto.WriteString(ch.Delta.Content)
			evs = append(evs, llm.Evento{Tipo: llm.EventoToken, Texto: ch.Delta.Content})
		}
		for _, tc := range ch.Delta.ToolCalls {
			a := acc.tools[tc.Index]
			if a == nil {
				a = &toolAcum{}
				acc.tools[tc.Index] = a
			}
			if tc.ID != "" {
				a.id = tc.ID
			}
			a.nombre.WriteString(tc.Function.Name)
			a.args.WriteString(tc.Function.Arguments)
		}
	}
	return evs, nil
}

// finalizar construye la RespuestaFinal con lo acumulado: texto, razonamiento y
// las tool_calls ordenadas por su índice.
func (acc *acumulador) finalizar() llm.RespuestaFinal {
	final := llm.RespuestaFinal{
		Model:      acc.modelo,
		Texto:      acc.texto.String(),
		Razonamien: acc.razon.String(),
		Done:       true,
		TokensEntr: uint64(acc.tokens.entrada),
		TokensSal:  uint64(acc.tokens.salida),
	}
	indices := make([]int, 0, len(acc.tools))
	for i := range acc.tools {
		indices = append(indices, i)
	}
	sort.Ints(indices)
	for _, i := range indices {
		a := acc.tools[i]
		nombre := a.nombre.String()
		if nombre == "" {
			continue
		}
		var c llm.ToolCall
		c.Function.Name = nombre
		args := a.args.String()
		if strings.TrimSpace(args) == "" {
			args = "{}"
		}
		c.Function.Arguments = json.RawMessage(args)
		final.ToolCalls = append(final.ToolCalls, c)
	}
	return final
}

// leerSSE lee el cuerpo SSE y emite eventos por out. Único writer de out; cierra
// out siempre.
func leerSSE(ctx context.Context, cuerpo io.ReadCloser, out chan<- llm.Evento) {
	defer close(out)
	defer cuerpo.Close()

	esc := bufio.NewScanner(cuerpo)
	esc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	acc := nuevoAcumulador()
	vioDone := false

	for esc.Scan() {
		select {
		case <-ctx.Done():
			enviar(ctx, out, llm.Evento{Tipo: llm.EventoError, Error: ctx.Err()})
			return
		default:
		}
		linea := strings.TrimSpace(esc.Text())
		if linea == "" || !strings.HasPrefix(linea, "data:") {
			continue
		}
		datos := strings.TrimSpace(strings.TrimPrefix(linea, "data:"))
		if datos == "[DONE]" {
			enviar(ctx, out, llm.Evento{Tipo: llm.EventoDone, Done: ptrFinal(acc.finalizar())})
			vioDone = true
			break
		}
		evs, err := procesarSSE([]byte(datos), acc)
		if err != nil {
			enviar(ctx, out, llm.Evento{Tipo: llm.EventoError, Error: err})
			return
		}
		for _, e := range evs {
			if !enviar(ctx, out, e) {
				return
			}
		}
	}
	if err := esc.Err(); err != nil && err != io.EOF {
		if tipado, ok := clasificarFalloRed(err); ok {
			enviar(ctx, out, llm.Evento{Tipo: llm.EventoError, Error: tipado})
			return
		}
		enviar(ctx, out, llm.Evento{Tipo: llm.EventoError, Error: llm.NuevoErrorNoDisponible("el stream se cortó antes de terminar", err.Error())})
		return
	}
	if vioDone {
		return
	}
	// Sin `[DONE]` no hay respuesta completa: se avisa en vez de inventar un
	// cierre limpio con texto a medias.
	enviar(ctx, out, llm.Evento{Tipo: llm.EventoError, Error: llm.NuevoErrorNoDisponible("el stream terminó sin la señal de fin", "")})
}

func ptrFinal(f llm.RespuestaFinal) *llm.RespuestaFinal { return &f }

// enviar entrega ev respetando la cancelación. Devuelve false si el contexto
// murió antes de poder entregar.
func enviar(ctx context.Context, out chan<- llm.Evento, ev llm.Evento) bool {
	select {
	case out <- ev:
		return true
	case <-ctx.Done():
		return false
	}
}
