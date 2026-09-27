package agent

// catalog.go — T-B006-03 y T-B006-08: derivar el catálogo efectivo de un
// agente a partir de sus permisos, y validar esos permisos.
//
// Fuente de verdad: ai/docs/specs/SPEC-AGENTE-BASE.md §Dónde se definen y
// ai/docs/backend/DECISIONS.md ("`permisos` es la fuente; el catálogo efectivo
// se deriva contra el catálogo cerrado de `tools`").
//
// El catálogo cerrado es una sola fuente de verdad y vive en `tools`: aquí no
// se duplica la lista de las trece, se pregunta por acción. Un agente sin
// permisos es válido: es un agente de solo conversación.

import (
	"fmt"

	"localcli/internal/tools"
)

// errAgente construye un error localizado de carga de agente. No usa ningún
// código `E_` porque ERRORS.md no define uno para un JSON de agente inválido:
// fingir un código ajeno sería peor que un mensaje claro.
func errAgente(msg string) error { return fmt.Errorf("agente: %s", msg) }

// HerramientasDe deriva el catálogo efectivo de un agente a partir de sus
// permisos. Default deny: una acción que no aparece como `permitir` no está.
// La acción de cada herramienta la decide `tools` (categoría + modo); aquí no
// se repite esa tabla.
//
// Rechaza una acción o un efecto desconocidos, y dos permisos contradictorios
// sobre la misma acción (`permitir` + `denegar`): un contrato que se
// contradice no se carga.
func HerramientasDe(permisos []Permiso) ([]string, error) {
	visto := map[tools.Accion]string{}
	for _, p := range permisos {
		accion := tools.Accion(p.Accion)
		if !accion.Valida() {
			return nil, errAgente("acción de permiso desconocida: " + p.Accion)
		}
		switch p.Efecto {
		case EfectoPermitir, EfectoDenegar:
		default:
			return nil, errAgente("efecto de permiso desconocido: " + p.Efecto)
		}
		if previo, ok := visto[accion]; ok && previo != p.Efecto {
			return nil, errAgente("permisos contradictorios para la acción " + p.Accion)
		}
		visto[accion] = p.Efecto
	}

	out := make([]string, 0, len(tools.NombresCatalogo()))
	for _, h := range tools.Herramientas() {
		if visto[h.Accion()] == EfectoPermitir {
			out = append(out, h.Nombre)
		}
	}
	return out, nil
}

// ValidarPermisos comprueba que los permisos de un agente son válidos: acciones
// y efectos conocidos y sin contradicciones.
func ValidarPermisos(a Agente) error {
	_, err := HerramientasDe(a.Permisos)
	return err
}

// Declara informa si el agente tiene la herramienta en su catálogo efectivo.
// Un agente de solo conversación no declara ninguna, así que esto es false.
func (a Agente) Declara(nombre string) bool {
	for _, n := range a.Herramientas {
		if n == nombre {
			return true
		}
	}
	return false
}

// TieneEscritura informa si el agente tiene al menos una herramienta de
// escritura. Para `plan` tiene que ser false: es la garantía de que no puede
// escribir, sostenida en sus permisos y no en una comprobación olvidable.
func (a Agente) TieneEscritura() bool {
	for _, nombre := range a.Herramientas {
		if h, ok := tools.Buscar(nombre); ok && h.SoloBuild() {
			return true
		}
	}
	return false
}
