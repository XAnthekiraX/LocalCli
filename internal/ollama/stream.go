// stream.go — T-B005-02: streaming NDJSON token a token por canal.
//
// Fuente de verdad: INTEGRATIONS.md (streaming obligatorio, razonamiento
// distinguible) y DECISIONS.md [25] (la pantalla recibe tokens por eventos).
//
// Formato: Ollama responde en modo stream una secuencia de objetos JSON, uno
// por línea (NDJSON). Cada línea puede traer `response` (token de texto),
// `thinking` (token de razonamiento; ver reasoning.go) y la última trae
// `done:true` con métricas.
//
// Invariantes del canal devuelto por Client.Generar/Chat:
//   - Solo un emisor (el goroutine bombear).
//   - El canal se CIERRA siempre: tras EventoDone, tras EventoError, o al
//     cancelarse el contexto. El receptor nunca se bloquea para siempre.
//   - Los tokens se emiten EN ORDEN de llegada; nada se reordena ni se agrupa.
package ollama

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
)

// TipoEvento distingue qué lleva un Evento.
type TipoEvento int

const (
	EventoToken        TipoEvento = iota // token de texto final
	EventoRazonamiento                   // token de razonamiento (reasoning.go)
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

// lineaJSON es la forma cruda de una línea NDJSON de /api/generate o
// /api/chat (chat mete el token dentro de message.content).
type lineaJSON struct {
	Model    string `json:"model"`
	Response string `json:"response"` // generate
	Message  struct {
		Content string `json:"content"` // chat
	} `json:"message"`
	Thinking string `json:"thinking"` // razonamiento (generate y chat)
	Done     bool   `json:"done"`
	Context  []byte `json:"context,omitempty"`

	PromptEvalCount int64  `json:"prompt_eval_count,omitempty"`
	EvalCount       int64  `json:"eval_count,omitempty"`
	TotalDurationNs uint64 `json:"total_duration,omitempty"`
}

// acumulador reconstruye la respuesta final a partir de los tokens, en orden.
type acumulador struct {
	texto        []byte
	razonamiento []byte
	final        RespuestaFinal
}

// procesarLinea convierte una línea NDJSON en cero o más eventos. Se expone
// como función pura para poder testear el parseo sin HTTP (fixture NDJSON →
// tokens en orden, criterio T-B005-02).
func procesarLinea(linea []byte, acc *acumulador) ([]Evento, error) {
	if len(linea) == 0 {
		return nil, nil
	}
	var l lineaJSON
	if err := json.Unmarshal(linea, &l); err != nil {
		return nil, &ErrorOllama{
			Codigo:  CodigoOllamaNoDisponible,
			Mensaje: "respuesta ilegible del modelo",
			Detalle: err.Error(),
		}
	}
	var evs []Evento
	// Orden dentro de la línea: primero razonamiento, luego texto. Es el orden
	// natural de aparición (el modelo piensa antes de responder) y mantiene
	// separados ambos flujos (contrato §5: razonamiento distinguible).
	if r := separarRazonamiento(l.Thinking); r != "" {
		acc.razonamiento = append(acc.razonamiento, r...)
		evs = append(evs, Evento{Tipo: EventoRazonamiento, Texto: r})
	}
	t := l.Response
	if t == "" {
		t = l.Message.Content
	}
	if t != "" {
		acc.texto = append(acc.texto, t...)
		evs = append(evs, Evento{Tipo: EventoToken, Texto: t})
	}
	if l.Done {
		acc.final = RespuestaFinal{
			Model:      l.Model,
			Texto:      string(acc.texto),
			Razonamien: string(acc.razonamiento),
			Done:       true,
			TokensEntr: uint64(max64(l.PromptEvalCount, int64(acc.final.TokensEntr))),
			TokensSal:  uint64(max64(l.EvalCount, int64(acc.final.TokensSal))),
			DuracionNs: l.TotalDurationNs,
		}
		evs = append(evs, Evento{Tipo: EventoDone, Done: &acc.final})
	}
	return evs, nil
}

// bombear lee el cuerpo NDJSON y emite eventos por out. Único writer de out;
// cierra out siempre (ver invariantes arriba).
func bombear(ctx context.Context, cuerpo io.ReadCloser, out chan<- Evento) {
	defer close(out)
	defer cuerpo.Close()

	escaner := bufio.NewScanner(cuerpo)
	// Un token puede venir en líneas largas (prompts/citados); subimos el
	// límite del scanner a 1 MB por línea.
	escaner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var acc acumulador
	vioDone := false

	for escaner.Scan() {
		select {
		case <-ctx.Done():
			enviar(ctx, out, Evento{Tipo: EventoError, Error: ctx.Err()})
			return
		default:
		}
		evs, err := procesarLinea(escaner.Bytes(), &acc)
		if err != nil {
			enviar(ctx, out, Evento{Tipo: EventoError, Error: err})
			return
		}
		for _, e := range evs {
			if e.Tipo == EventoDone {
				vioDone = true
			}
			if !enviar(ctx, out, e) {
				return // contexto cancelado: cerramos (defer) y salimos
			}
		}
	}
	if err := escaner.Err(); err != nil && err != io.EOF {
		if tipado, ok := clasificarFalloRed(err); ok {
			enviar(ctx, out, Evento{Tipo: EventoError, Error: tipado})
			return
		}
		enviar(ctx, out, Evento{Tipo: EventoError, Error: &ErrorOllama{
			Codigo:  CodigoOllamaNoDisponible,
			Mensaje: "el stream se cortó antes de terminar",
			Detalle: err.Error(),
		}})
		return
	}
	if vioDone {
		return // cierre limpio tras la señal de fin: nada que añadir
	}
	// Stream terminado sin línea done (servidor cortó limpio): no inventamos
	// EventoDone; segnalamos error para que el consumidor no crea que hay
	// respuesta completa.
	enviar(ctx, out, Evento{Tipo: EventoError, Error: &ErrorOllama{
		Codigo:  CodigoOllamaNoDisponible,
		Mensaje: "el stream terminó sin la señal de fin",
	}})
}

// enviar entrega ev respetando la cancelación. Devuelve false si el contexto
// murió antes de poder entregar.
func enviar(ctx context.Context, out chan<- Evento, ev Evento) bool {
	select {
	case out <- ev:
		return true
	case <-ctx.Done():
		return false
	}
}

func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
