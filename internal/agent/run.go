package agent

// run.go — T-B006-05: construir la llamada a ollama con el prompt del agente
// y sus mensajes.
//
// Fuente de verdad: ai/docs/backend/02-interfaces/INTERFACES-GENERAL.md §5
// ("`agent` construye la llamada de un agente y la envía") y
// ai/docs/backend/BACKEND.md §4 ("el agente construye la llamada con su prompt
// y la envía").
//
// El prompt del agente es su identidad y viaja como mensaje de sistema, antes
// de los turnos de la conversación: define cómo se comporta el modelo en toda
// la llamada. `agent` no habla HTTP por su cuenta: se lo pide a `ollama`, que
// es el único que habla con el modelo.

import (
	"context"

	"localcli/internal/ollama"
)

// RolSistema es el rol del mensaje que lleva el prompt del agente.
const RolSistema = "system"

// ConstruirPeticion arma la llamada al modelo: el prompt del agente primero,
// como mensaje de sistema, y después los mensajes de la sesión. No reordena ni
// recorta nada; el contexto que llega es el que entrega el nodo de contexto.
func ConstruirPeticion(a Agente, modelo string, mensajes []ollama.Mensaje) ollama.GenerarRequest {
	msgs := make([]ollama.Mensaje, 0, len(mensajes)+1)
	msgs = append(msgs, ollama.Mensaje{Role: RolSistema, Content: a.Prompt})
	msgs = append(msgs, mensajes...)
	return ollama.GenerarRequest{
		Model:    modelo,
		Messages: msgs,
	}
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
