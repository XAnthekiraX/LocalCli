package agent

// run.go — T-B036-08: construir la llamada al modelo con el prompt del agente,
// sus mensajes y el canal nativo de herramientas, sobre la frontera neutra.
//
// Fuente de verdad: ai/docs/backend/02-interfaces/INTERFACES-GENERAL.md §5 y
// ai/docs/backend/02-interfaces/TOOLS.md §11 (el cable: cómo llegan las
// herramientas al modelo).
//
// El prompt del agente viaja como mensaje de sistema, antes de los turnos de la
// conversación. El catálogo de herramientas NO viaja aquí: va por el campo de
// herramientas de la petición neutra, que cada adaptador serializa en su
// formato. Por eso `PromptDeSistema` se queda con el prompt y nada más.

import (
	"context"

	"localcli/internal/llm"
)

// RolSistema es el rol del mensaje que lleva el prompt del agente.
const RolSistema = "system"

// ConstruirPeticion arma la llamada al modelo: el mensaje de sistema del
// agente primero, y después los mensajes de la sesión. `herramientas` son las
// definiciones que el modelo puede pedir; un agente de solo conversación viaja
// sin ninguna. `numCtx` es la ventana de contexto a pedir (0 = la del
// servidor). `pensar` decide si el modelo razona: nil NO manda nada.
func ConstruirPeticion(a Agente, modelo string, mensajes []llm.Mensaje, herramientas []llm.Herramienta, numCtx int, pensar *bool) llm.Peticion {
	msgs := make([]llm.Mensaje, 0, len(mensajes)+1)
	msgs = append(msgs, llm.Mensaje{Role: RolSistema, Content: PromptDeSistema(a)})
	msgs = append(msgs, mensajes...)
	return llm.Peticion{
		Modelo:       modelo,
		Mensajes:     msgs,
		Herramientas: herramientas,
		NumCtx:       numCtx,
		Pensar:       pensar,
	}
}

// PromptDeSistema devuelve el mensaje de sistema del agente: su prompt. El
// catálogo no se inyecta —viaja por el canal de herramientas—.
func PromptDeSistema(a Agente) string { return a.Prompt }

// Generador envía la petición de un agente al modelo y devuelve su stream de
// eventos. Lo implementa Runner (producción); existe como interfaz para poder
// probar el bucle con un doble sin modelo.
type Generador interface {
	Generar(ctx context.Context, a Agente, modelo string, mensajes []llm.Mensaje, herramientas []llm.Herramienta, numCtx int, pensar *bool) (<-chan llm.Evento, error)
}

// Runner envía la petición de un agente al modelo. Es la frontera de `agent`
// con `llm.Proveedor`: no sabe si detrás hay Ollama o llama.cpp.
type Runner struct {
	Proveedor llm.Proveedor
}

// Generar construye la petición del agente y la lanza en streaming.
func (r Runner) Generar(ctx context.Context, a Agente, modelo string, mensajes []llm.Mensaje, herramientas []llm.Herramienta, numCtx int, pensar *bool) (<-chan llm.Evento, error) {
	return r.Proveedor.Chat(ctx, ConstruirPeticion(a, modelo, mensajes, herramientas, numCtx, pensar))
}
