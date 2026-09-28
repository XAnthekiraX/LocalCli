package agent

// loop.go — el ciclo conversacional único del agente, con el contrato nativo de
// herramientas.
//
// Fuente de verdad: ai/docs/specs/SPEC-AGENTE-BASE.md §El ciclo de un turno
// (pedir, ejecutar y volver a pedir, con un máximo de rondas) y
// ai/docs/specs/SPEC-TOOLS.md §Cómo se le pide una herramienta al modelo (el
// canal nativo, sin formato en prosa).
//
// Un agente NO responde «por un lado» y usa herramientas «por otro»: las
// herramientas forman parte del mismo ciclo de inferencia. El modelo recibe el
// contexto y las definiciones de sus herramientas; responde con texto, con una
// petición de herramienta, o con las dos cosas. Si pidió herramientas, se
// ejecutan **en el orden pedido** y sus resultados vuelven como mensajes de rol
// `tool`, y se vuelve a preguntar. `plan` y `build` corren ESTE mismo bucle:
// solo cambian su prompt y sus permisos.
//
// Un fallo corregible vuelve al modelo como resultado (`tools.Resultado.Error`);
// un fallo duro se le dice igual, pero no detiene el trabajo. Al agotar las
// rondas, el turno termina con lo conseguido.

import (
	"context"
	"strings"

	"localcli/internal/ollama"
)

// pasadasPorDefecto acota cuántas veces puede pedir herramientas un turno
// antes de cerrarse.
const pasadasPorDefecto = 3

// Sink recibe los fragmentos de texto y razonamiento a medida que el modelo
// los produce. Puede ser nil: el bucle trabaja igual sin consumidor.
type Sink interface {
	Token(texto string, esRazon bool)
}

// Avisador es un Sink que además recibe avisos del turno (por ejemplo, que el
// agente ha caído a modo conversación).
type Avisador interface {
	Aviso(texto string)
}

// Resultado es lo que deja un turno del agente: el texto final, el razonamiento
// acumulado y las métricas de tokens del turno.
type Resultado struct {
	Texto         string
	Razonamiento  string
	TokensEntrada uint64
	TokensSalida  uint64
	Aviso         string
}

// Ejecutor corre el ciclo conversacional de un agente sobre el modelo. Es el
// único bucle: `plan` y `build` lo comparten.
type Ejecutor struct {
	Runner     Generador
	Despachar  *Despachador
	MaxPasadas int
	// Testigo toma el turno de inferencia POR PETICIÓN al modelo, no por
	// ejecución completa: una sesión esperando una aprobación no retiene el
	// modelo de las demás (SPEC-OLLAMA-PERFIL). Puede ser nil.
	Testigo func(ctx context.Context, fn func(context.Context) error) error
	// PuedeHerramientas informa si el modelo en uso declara la capacidad de
	// pedir herramientas. nil —o una función que devuelve true— deja el turno
	// como está; false degrada el agente a modo conversación.
	PuedeHerramientas func(modelo string) bool
}

// Ejecutar responde una petición: arma los mensajes con el historial de la
// conversación y el contexto del turno, llama al modelo y, mientras pida
// herramientas, las ejecuta y reinyecta sus resultados hasta agotar las rondas
// o quedarse sin pedidos.
// `numCtx` es la ventana de contexto a pedir al modelo en cada pasada (0 = la
// del servidor). Llega por llamada, no como campo del `Ejecutor`: las sesiones
// corren en paralelo y pueden tener modelos con ventanas distintas.
func (e *Ejecutor) Ejecutar(ctx context.Context, a Agente, modelo, contexto string, historial []ollama.Mensaje, imagenes []string, numCtx int, sink Sink) (Resultado, error) {
	max := e.MaxPasadas
	if max <= 0 {
		max = pasadasPorDefecto
	}

	// Degradación honesta: si el modelo no declara capacidad de herramientas, no
	// se le presentan. No se le impide usarlo; el agente conversa.
	var aviso string
	var herramientas []ollama.Herramienta
	if e.PuedeHerramientas != nil && !e.PuedeHerramientas(modelo) {
		aviso = "el modelo «" + modelo + "» no declara capacidad de herramientas; el agente va a conversar sin ellas"
		if av, ok := sink.(Avisador); ok {
			av.Aviso(aviso)
		}
	} else if e.Despachar != nil {
		herramientas = e.Despachar.Definiciones(a)
	}

	mensajes := make([]ollama.Mensaje, 0, len(historial)+1)
	mensajes = append(mensajes, historial...)
	mensajes = append(mensajes, ollama.Mensaje{Role: "user", Content: contexto, Images: imagenes})

	var texto, razon strings.Builder
	var tokensIn, tokensOut uint64
	for pasada := 0; pasada < max; pasada++ {
		texto.Reset()
		razon.Reset()
		var pedidos []ollama.ToolCall

		err := e.conTestigo(ctx, func(c context.Context) error {
			ch, gErr := e.Runner.Generar(c, a, modelo, mensajes, herramientas, numCtx)
			if gErr != nil {
				return gErr
			}
			for ev := range ch {
				switch ev.Tipo {
				case ollama.EventoToken:
					texto.WriteString(ev.Texto)
					if sink != nil {
						sink.Token(ev.Texto, false)
					}
				case ollama.EventoRazonamiento:
					razon.WriteString(ev.Texto)
					if sink != nil {
						sink.Token(ev.Texto, true)
					}
				case ollama.EventoDone:
					// Las peticiones de herramienta llegan con la señal de fin;
					// el texto ya se ha emitido token a token.
					if ev.Done != nil {
						tokensIn += ev.Done.TokensEntr
						tokensOut += ev.Done.TokensSal
						pedidos = append(pedidos, ev.Done.ToolCalls...)
					}
				case ollama.EventoError:
					return ev.Error
				}
			}
			return nil
		})
		if err != nil {
			if sink != nil {
				sink.Token("[sin respuesta del modelo]", false)
			}
			return Resultado{}, err
		}

		if len(pedidos) == 0 {
			return Resultado{
				Texto:         texto.String(),
				Razonamiento:  razon.String(),
				TokensEntrada: tokensIn,
				TokensSalida:  tokensOut,
				Aviso:         aviso,
			}, nil
		}

		// El mensaje del asistente con sus peticiones, y después un mensaje de
		// herramienta por cada resultado, en el orden en que se pidieron.
		mensajes = append(mensajes, ollama.Mensaje{
			Role:      "assistant",
			Content:   texto.String(),
			ToolCalls: pedidos,
		})
		for _, llamada := range pedidos {
			res, dErr := e.Despachar.Despachar(ctx, a, SolicitudHerramienta{
				Nombre:     llamada.Nombre(),
				Argumentos: llamada.ArgumentosJSON(),
			})
			salida := res.Salida
			if dErr != nil {
				salida = dErr.Error()
			} else if res.Error != "" {
				salida = res.Error
			}
			if strings.TrimSpace(salida) == "" {
				salida = "(sin salida)"
			}
			mensajes = append(mensajes, ollama.Mensaje{
				Role:     "tool",
				Content:  salida,
				ToolName: llamada.Nombre(),
			})
		}
	}
	// Se agotaron las rondas: el turno termina con lo que haya conseguido.
	return Resultado{
		Texto:         texto.String(),
		Razonamiento:  razon.String(),
		TokensEntrada: tokensIn,
		TokensSalida:  tokensOut,
		Aviso:         aviso,
	}, nil
}

// conTestigo ejecuta fn con el turno de inferencia tomado y soltado alrededor
// de la petición: entre pasada y pasada el modelo queda libre.
func (e *Ejecutor) conTestigo(ctx context.Context, fn func(context.Context) error) error {
	if e.Testigo == nil {
		return fn(ctx)
	}
	return e.Testigo(ctx, fn)
}

// Definiciones expone las definiciones de herramientas del agente: es lo que el
// cableado necesita para saber si un agente tiene alguna ruta hacia `tools`.
func (e *Ejecutor) Definiciones(a Agente) []ollama.Herramienta {
	if e.Despachar == nil {
		return nil
	}
	return e.Despachar.Definiciones(a)
}
