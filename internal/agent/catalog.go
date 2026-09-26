package agent

// catalog.go — T-B006-03 y T-B006-08: validar el campo `herramientas` contra
// el catálogo cerrado de `tools`, y tratar un agente sin herramientas como
// agente de solo conversación.
//
// Fuente de verdad: ai/docs/backend/DECISIONS.md ("La carga valida contra el
// catálogo cerrado y rechaza una herramienta inexistente; `herramientas` vacío
// produce un agente de solo conversación") y
// ai/docs/specs/SPEC-AGENTE-BASE.md §Dónde se definen.
//
// El catálogo es una sola fuente de verdad y vive en `tools`: aquí no se
// duplica la lista de las trece, se pregunta. Una herramienta que no esté allí
// no existe, y un agente que la declare no se carga.

import (
	"fmt"

	"localcli/internal/tools"
)

// errAgente construye un error localizado de carga de agente. No usa ningún
// código `E_` porque ERRORS.md no define uno para un JSON de agente inválido:
// fingir un código ajeno sería peor que un mensaje claro.
func errAgente(msg string) error { return fmt.Errorf("agente: %s", msg) }

// ValidarHerramientas comprueba que todas las herramientas que declara un
// agente existen en el catálogo cerrado. Un agente sin herramientas es válido:
// es un agente de solo conversación.
func ValidarHerramientas(a Agente) error {
	for _, nombre := range a.Herramientas {
		if !tools.Existe(nombre) {
			return errAgente("el agente " + a.Nombre + " declara la herramienta `" +
				nombre + "`, que no está en el catálogo cerrado")
		}
	}
	return nil
}

// Declara informa si el agente tiene la herramienta en su catálogo. Un agente
// de solo conversación no declara ninguna, así que esto siempre es false.
func (a Agente) Declara(nombre string) bool {
	for _, n := range a.Herramientas {
		if n == nombre {
			return true
		}
	}
	return false
}

// TieneEscritura informa si el agente declara al menos una herramienta de
// escritura. Para `plan` tiene que ser false: es la garantía de que no puede
// escribir, sostenida en sus datos y no en una comprobación olvidable.
func (a Agente) TieneEscritura() bool {
	for _, nombre := range a.Herramientas {
		if h, ok := tools.Buscar(nombre); ok && h.SoloBuild() {
			return true
		}
	}
	return false
}
