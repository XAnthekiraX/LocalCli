// Package compact define el formato de los resúmenes de compactación del
// historial de conversación. Es puro: no habla con el modelo ni con la base de
// datos; solo produce los prompts que el gestor de contexto envía al motor.
// Ver [[specs/SPEC-HISTORIAL-CONVERSACION]].
package compact

import "strings"

// Formatos de resumen que entiende el gestor de contexto
// ([[specs/SPEC-HISTORIAL-CONVERSACION]] §Formato y modelo del resumen).
const (
	// FormatoOpenCode es el resumen estructurado estilo OpenCode: secciones
	// Objective, Important Details, Work State, Next Move y Relevant Files.
	FormatoOpenCode = "opencode"
	// FormatoLegacy es el resumen libre en pocas líneas.
	FormatoLegacy = "legacy"
	// FormatoCustom usa la plantilla del usuario tal cual.
	FormatoCustom = "custom"
)

// Opciones describe cómo se pide el resumen: qué formato y, para el formato
// "custom", con qué plantilla.
type Opciones struct {
	Formato   string
	Plantilla string
}

// normalizado devuelve el formato efectivo: solo "legacy" y "custom" se
// distinguen; cualquier otro valor (o vacío) cae en "opencode".
func (o Opciones) normalizado() string {
	switch strings.ToLower(strings.TrimSpace(o.Formato)) {
	case FormatoLegacy:
		return FormatoLegacy
	case FormatoCustom:
		return FormatoCustom
	default:
		return FormatoOpenCode
	}
}

// PromptInicial es el prompt para resumir un tramo sin resumen previo.
func (o Opciones) PromptInicial() string {
	switch o.normalizado() {
	case FormatoLegacy:
		return promptLegacy
	case FormatoCustom:
		if p := strings.TrimSpace(o.Plantilla); p != "" {
			return p
		}
		return promptLegacy
	default:
		return promptOpenCode
	}
}

// PromptIncremental es el prompt para actualizar un resumen previo con los
// mensajes nuevos, sin perder lo ya resumido.
func (o Opciones) PromptIncremental(anterior string) string {
	anterior = strings.TrimSpace(anterior)
	if anterior == "" {
		return o.PromptInicial()
	}
	switch o.normalizado() {
	case FormatoLegacy:
		return promptLegacyIncremental + anterior
	case FormatoCustom:
		base := strings.TrimSpace(o.Plantilla)
		if base == "" {
			base = promptLegacy
		}
		return base + "\n\n" + separadorResumenPrevio + anterior
	default:
		return promptOpenCodeIncremental + anterior
	}
}

// separadorResumenPrevio introduce el resumen anterior dentro del prompt.
const separadorResumenPrevio = "### Resumen previo\n"

const promptLegacy = "Resume la conversación siguiente en pocas líneas, conservando " +
	"los datos y las decisiones importantes y sin añadir nada que no aparezca en ella. " +
	"Devuelve solo el resumen."

const promptLegacyIncremental = promptLegacy + " Ya tienes un resumen de la parte " +
	"anterior de la conversación: intégralo con la información nueva sin perder nada. " +
	"Devuelve solo el resumen actualizado.\n\n"

const promptOpenCode = "Eres un asistente que compacta el contexto de una conversación de " +
	"desarrollo de software. Resume la conversación siguiente de modo que otro agente " +
	"pueda continuar el trabajo con solo leer el resumen. No inventes nada que no " +
	"aparezca en ella. Usa exactamente estas secciones en Markdown:\n\n" +
	"## Objective\n- ...\n\n" +
	"## Important Details\n- ...\n\n" +
	"## Work State\n### Completed\n- ...\n### Active\n- ...\n### Blocked\n- ...\n\n" +
	"## Next Move\n1. ...\n\n" +
	"## Relevant Files\n- `ruta`: para qué sirve\n\n" +
	"Devuelve solo el resumen."

const promptOpenCodeIncremental = "Eres un asistente que compacta el contexto de una " +
	"conversación de desarrollo de software. Tienes un resumen previo y su continuación. " +
	"Actualiza el resumen integrando la información nueva sin perder la anterior ni " +
	"inventar nada. Conserva el mismo formato Markdown con las secciones Objective, " +
	"Important Details, Work State, Next Move y Relevant Files. Devuelve solo el resumen " +
	"actualizado.\n\n" + separadorResumenPrevio
