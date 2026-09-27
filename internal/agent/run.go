package agent

// run.go — T-B006-05: construir la llamada a ollama con el prompt del agente
// y sus mensajes.
//
// Fuente de verdad: ai/docs/backend/02-interfaces/INTERFACES-GENERAL.md §5
// ("`agent` construye la llamada de un agente y la envía"), BACKEND.md §4 y
// ai/docs/specs/SPEC-TOOLS.md (el agente responde con las herramientas de su
// catálogo).
//
// El prompt del agente es su identidad y viaja como mensaje de sistema, antes
// de los turnos de la conversación. Como el modelo local no tiene
// function-calling nativo, el mensaje de sistema se COMPLETA con el catálogo
// disponible del agente y el formato exacto de llamada: así sabe qué puede
// pedir y cómo pedirlo sin depender de prosa del JSON. `agent` no habla HTTP
// por su cuenta: se lo pide a `ollama`.

import (
	"context"
	"strings"

	"localcli/internal/ollama"
	"localcli/internal/tools"
)

// RolSistema es el rol del mensaje que lleva el prompt del agente.
const RolSistema = "system"

// ConstruirPeticion arma la llamada al modelo: el mensaje de sistema del
// agente (prompt + catálogo y formato) primero, y después los mensajes de la
// sesión. No reordena ni recorta nada; el contexto que llega es el que entrega
// el nodo de contexto.
func ConstruirPeticion(a Agente, modelo string, mensajes []ollama.Mensaje) ollama.GenerarRequest {
	msgs := make([]ollama.Mensaje, 0, len(mensajes)+1)
	msgs = append(msgs, ollama.Mensaje{Role: RolSistema, Content: PromptDeSistema(a)})
	msgs = append(msgs, mensajes...)
	return ollama.GenerarRequest{
		Model:    modelo,
		Messages: msgs,
	}
}

// PromptDeSistema compone el mensaje de sistema del agente: su prompt y, si
// tiene herramientas, el catálogo disponible y el formato exacto para pedirlas.
// Un agente de solo conversación recibe únicamente su prompt.
func PromptDeSistema(a Agente) string {
	if len(a.Herramientas) == 0 {
		return a.Prompt
	}
	var b strings.Builder
	b.WriteString(a.Prompt)
	b.WriteString("\n\n## Herramientas disponibles\n\n")
	b.WriteString(tools.CatalogoTexto(a.Herramientas))
	b.WriteString("\n## Cómo pedir una herramienta\n\n")
	b.WriteString("Cuando necesites una herramienta no la describas en prosa: escribe un bloque con el nombre exacto y los argumentos en JSON, así:\n\n")
	b.WriteString("```herramienta\n")
	b.WriteString("<nombre de la herramienta>\n")
	b.WriteString(`{"campo": "valor"}`)
	b.WriteString("\n```\n")
	b.WriteString("\nPuedes pedir varias herramientas seguidas, un bloque por cada una. Recibirás sus resultados y continuarás.\n")
	return b.String()
}

// Generador envía la petición de un agente al modelo y devuelve su stream de
// eventos. Lo implementa Runner (producción); existe como interfaz para poder
// probar el bucle con un doble sin Ollama.
type Generador interface {
	Generar(ctx context.Context, a Agente, modelo string, mensajes []ollama.Mensaje) (<-chan ollama.Evento, error)
}

// Runner envía la petición de un agente al modelo. Es la frontera de `agent`
// con `ollama`.
type Runner struct {
	Cliente *ollama.Client
}

// Generar construye la petición del agente y la lanza en streaming. Devuelve
// el canal de eventos de `ollama` tal cual: los tokens y el razonamiento los
// consume la TUI.
func (r Runner) Generar(ctx context.Context, a Agente, modelo string, mensajes []ollama.Mensaje) (<-chan ollama.Evento, error) {
	return r.Cliente.Chat(ctx, ConstruirPeticion(a, modelo, mensajes))
}
