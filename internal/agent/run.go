package agent

// run.go — construir la llamada a ollama con el prompt del agente, sus
// mensajes y el canal nativo de herramientas.
//
// Fuente de verdad: ai/docs/backend/02-interfaces/INTERFACES-GENERAL.md §5
// ("`agent` construye la llamada de un agente y la envía") y
// ai/docs/backend/02-interfaces/TOOLS.md §11 (el cable: cómo llegan las
// herramientas al modelo).
//
// El prompt del agente es su identidad y viaja como mensaje de sistema, antes
// de los turnos de la conversación. El catálogo de herramientas NO viaja aquí:
// va por el campo `tools` de `/api/chat`, que es el canal nativo de Ollama. Por
// eso `PromptDeSistema` se queda con el prompt y nada más (DECISIONS.md: "El
// prompt del agente no incluye el catálogo ni el formato de llamada").

import (
	"context"

	"localcli/internal/ollama"
)

// RolSistema es el rol del mensaje que lleva el prompt del agente.
const RolSistema = "system"

// ConstruirPeticion arma la llamada al modelo: el mensaje de sistema del
// agente primero, y después los mensajes de la sesión. `herramientas` son las
// definiciones que el modelo puede pedir; un agente de solo conversación viaja
// sin ninguna. `numCtx` es la ventana de contexto a pedir (0 = la del
// servidor): sin ella, Ollama usa un valor pequeño que corta los turnos con
// herramientas.
//
// `pensar` decide si el modelo razona antes de responder: nil NO manda el campo
// —Ollama decide, que es lo que quiere un modelo sin razonamiento— y un puntero
// manda ese valor. En Ollama el razonamiento viene encendido por defecto en los
// modelos que lo declaran, así que apagarlo es una decisión explícita del
// usuario (SPEC-OLLAMA-PERFIL).
func ConstruirPeticion(a Agente, modelo string, mensajes []ollama.Mensaje, herramientas []ollama.Herramienta, numCtx int, pensar *bool) ollama.GenerarRequest {
	msgs := make([]ollama.Mensaje, 0, len(mensajes)+1)
	msgs = append(msgs, ollama.Mensaje{Role: RolSistema, Content: PromptDeSistema(a)})
	msgs = append(msgs, mensajes...)
	req := ollama.GenerarRequest{
		Model:    modelo,
		Messages: msgs,
		Tools:    herramientas,
		NumCtx:   numCtx,
	}
	// Se asigna el valor desnudo: un puntero nil dentro de `any` NO queda vacío al
	// serializar (`omitempty` mira la interfaz, no lo apuntado) y mandaría
	// `"think": null`.
	if pensar != nil {
		req.Think = *pensar
	}
	return req
}

// PromptDeSistema devuelve el mensaje de sistema del agente: su prompt. El
// catálogo no se inyecta —viaja por el canal de herramientas—, así que aquí no
// hay nada más que añadir.
func PromptDeSistema(a Agente) string { return a.Prompt }

// Generador envía la petición de un agente al modelo y devuelve su stream de
// eventos. Lo implementa Runner (producción); existe como interfaz para poder
// probar el bucle con un doble sin Ollama.
type Generador interface {
	Generar(ctx context.Context, a Agente, modelo string, mensajes []ollama.Mensaje, herramientas []ollama.Herramienta, numCtx int, pensar *bool) (<-chan ollama.Evento, error)
}

// Runner envía la petición de un agente al modelo. Es la frontera de `agent`
// con `ollama`.
type Runner struct {
	Cliente *ollama.Client
}

// Generar construye la petición del agente y la lanza en streaming.
func (r Runner) Generar(ctx context.Context, a Agente, modelo string, mensajes []ollama.Mensaje, herramientas []ollama.Herramienta, numCtx int, pensar *bool) (<-chan ollama.Evento, error) {
	return r.Cliente.Chat(ctx, ConstruirPeticion(a, modelo, mensajes, herramientas, numCtx, pensar))
}
