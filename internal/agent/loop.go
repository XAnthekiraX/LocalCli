package agent

// loop.go — el ciclo conversacional único del agente.
//
// Fuente de verdad: ai/docs/specs/SPEC-TOOLS.md §7 (el modelo pide, `tools`
// comprueba y enruta, el resultado vuelve al modelo) y DECISIONS.md.
//
// Un agente NO responde «por un lado» y usa herramientas «por otro»: las
// herramientas forman parte del mismo ciclo de inferencia. El modelo responde
// texto; si ese texto pide una o varias herramientas, se ejecutan y sus
// resultados se reinyectan como un mensaje más, y se vuelve a preguntar al
// modelo. `plan` y `build` corren ESTE mismo bucle: solo cambian su prompt y
// sus permisos.
//
// El bucle vive aquí, no en el arranque: el agente es la unidad que conoce su
// prompt, su catálogo y su ciclo. La emisión de tokens a la TUI llega por un
// `Sink`, así que `agent` no conoce `tui` ni `session`.

import (
	"context"
	"encoding/json"
	"strings"

	"localcli/internal/ollama"
	"localcli/internal/tools"
)

// pasadasPorDefecto acota cuántas veces puede pedir herramientas un turno
// antes de cerrarse: el modelo responde, pide, se ejecutan y vuelve a
// responder. Un turno no puede quedar abierto indefinidamente.
const pasadasPorDefecto = 3

// Sink recibe los fragmentos de texto y razonamiento a medida que el modelo
// los produce. Quien lo implemente decide qué hacer con ellos (la TUI los
// pinta). Puede ser nil: el bucle trabaja igual sin consumidor.
type Sink interface {
	Token(texto string, esRazon bool)
}

// Resultado es lo que deja un turno del agente: el texto final y el
// razonamiento acumulado.
type Resultado struct {
	Texto        string
	Razonamiento string
}

// Ejecutor corre el ciclo conversacional de un agente sobre el modelo. Es el
// único bucle: `plan` y `build` lo comparten.
type Ejecutor struct {
	Runner     Generador
	Despachar  *Despachador
	MaxPasadas int // por defecto pasadasPorDefecto
}

// Ejecutar responde una petición: arma los mensajes con el contexto, llama al
// modelo y, mientras el modelo pida herramientas, las despacha a `tools` y
// reinyecta sus resultados hasta agotar las pasadas o quedarse sin pedidos.
func (e *Ejecutor) Ejecutar(ctx context.Context, a Agente, modelo, contexto string, sink Sink) (Resultado, error) {
	max := e.MaxPasadas
	if max <= 0 {
		max = pasadasPorDefecto
	}

	mensajes := []ollama.Mensaje{{Role: "user", Content: contexto}}
	var texto, razon strings.Builder
	for pasada := 0; pasada < max; pasada++ {
		ch, err := e.Runner.Generar(ctx, a, modelo, mensajes)
		if err != nil {
			if sink != nil {
				sink.Token("[sin respuesta del modelo]", false)
			}
			return Resultado{}, err
		}

		texto.Reset()
		razon.Reset()
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
			case ollama.EventoError:
				return Resultado{}, ev.Error
			}
		}

		pedidos := Solicitudes(texto.String())
		if len(pedidos) == 0 {
			return Resultado{Texto: texto.String(), Razonamiento: razon.String()}, nil
		}

		mensajes = append(mensajes, ollama.Mensaje{Role: "assistant", Content: texto.String()})
		var informe strings.Builder
		for _, sol := range pedidos {
			resultado, err := e.Despachar.Despachar(ctx, a, sol)
			if err != nil {
				// Un rechazo de `tools` no corta el turno: se le cuenta al
				// modelo como resultado y él decide qué hacer.
				resultado = map[string]string{"error": err.Error()}
			}
			datos, mErr := json.Marshal(resultado)
			if mErr != nil {
				datos = []byte(`{"error":"respuesta no serializable"}`)
			}
			informe.WriteString(sol.Nombre + " → " + string(datos) + "\n")
		}
		mensajes = append(mensajes, ollama.Mensaje{Role: "user", Content: "Resultados de herramientas:\n" + informe.String()})
	}
	return Resultado{Texto: texto.String(), Razonamiento: razon.String()}, nil
}

// Solicitudes extrae del texto del modelo los bloques
// ```herramienta ...``` con argumentos JSON. Es el contrato mínimo de llamada
// a herramientas para un modelo local sin function-calling nativo: lo que el
// modelo no declara en ese formato no se ejecuta, y `tools` valida el nombre y
// los argumentos contra el catálogo cerrado antes de enrutar (VALIDATION.md).
func Solicitudes(texto string) []SolicitudHerramienta {
	var out []SolicitudHerramienta
	resto := texto
	for {
		i := strings.Index(resto, "```herramienta")
		if i < 0 {
			break
		}
		resto = resto[i+len("```herramienta"):]
		j := strings.Index(resto, "\n```")
		if j < 0 {
			break
		}
		cuerpo := strings.TrimSpace(resto[:j])
		resto = resto[j+4:]
		nombre, args, ok := strings.Cut(cuerpo, "\n")
		nombre = strings.TrimSpace(nombre)
		if !ok || !tools.Existe(nombre) {
			continue
		}
		argJSON := strings.TrimSpace(args)
		if argJSON == "" {
			argJSON = "{}"
		}
		if !json.Valid([]byte(argJSON)) {
			continue
		}
		out = append(out, SolicitudHerramienta{Nombre: nombre, Argumentos: json.RawMessage(argJSON)})
	}
	return out
}
