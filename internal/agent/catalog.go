package agent

// catalog.go — T-B034-04: derivar el catálogo efectivo de un agente a partir de
// sus `permissions`, y validar esos permisos.
//
// Fuente de verdad: ai/docs/specs/SPEC-AGENTE-BASE.md §Los permisos y
// ai/docs/backend/01-domain/BUSINESS_RULES.md §Agentes (`permissions` es un mapa
// de permiso a efecto con tres permisos; el catálogo efectivo se deriva).
//
// El catálogo cerrado es una sola fuente de verdad y vive en `tools`: aquí no
// se duplica la lista de herramientas, se compara el `Permiso` de cada una
// contra los permisos del agente. Un agente sin `permissions` es válido: es un
// agente de solo conversación.

import (
	"fmt"
	"strings"

	"localcli/internal/tools"
)

// errAgente construye un error localizado de carga de agente. No usa ningún
// código `E_` porque ERRORS.md no define uno para un agente inválido: fingir un
// código ajeno sería peor que un mensaje claro.
func errAgente(msg string) error { return fmt.Errorf("agente: %s", msg) }

// HerramientasDe deriva el catálogo efectivo de un agente a partir de sus
// `permissions`. Default deny: un permiso que no aparece como `allow` no está.
// El permiso de cada herramienta lo declara `tools`: aquí no se repite esa
// tabla.
//
// Rechaza un permiso o un efecto fuera del vocabulario, y un `default` con un
// valor distinto de `deny`. Sin `permissions` —o vacío— el agente es de solo
// conversación y no es un error.
func HerramientasDe(permissions map[string]string) ([]string, error) {
	concedidos := map[tools.Permiso]bool{}
	for clave, bruto := range permissions {
		efecto := strings.TrimSpace(bruto)
		if clave == ClaveDefault {
			if efecto != EfectoDeny {
				return nil, errAgente("`default` solo admite `deny`: un `allow` concedería todo en silencio")
			}
			continue
		}
		p := tools.Permiso(clave)
		if !p.Valida() {
			return nil, errAgente("permiso desconocido: " + clave)
		}
		switch efecto {
		case EfectoAllow:
			concedidos[p] = true
		case EfectoDeny:
			// Un `deny` explícito es redundante con el default y se admite:
			// documenta que la denegación es deliberada.
		default:
			return nil, errAgente("efecto desconocido para `" + clave + "`: " + bruto)
		}
	}

	out := make([]string, 0, len(tools.NombresCatalogo()))
	for _, h := range tools.Herramientas() {
		if concedidos[h.Permiso] {
			out = append(out, h.Nombre)
		}
	}
	return out, nil
}

// ValidarPermisos comprueba que los permisos de un agente son válidos: permisos
// y efectos conocidos y sin un `default` que conceda todo.
func ValidarPermisos(a Agente) error {
	_, err := HerramientasDe(a.Permissions)
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

// Permisos devuelve los permisos que el agente concede (`allow`), en el orden
// del vocabulario y sin repetir ninguno. Es lo que alimenta el reparto de la
// capa universal y las definiciones que viajan al modelo: las herramientas del
// usuario entran por la misma vía.
func (a Agente) Permisos() []tools.Permiso {
	var out []tools.Permiso
	for _, p := range []tools.Permiso{tools.PermisoRead, tools.PermisoWrite, tools.PermisoEdit} {
		if strings.TrimSpace(a.Permissions[string(p)]) == EfectoAllow {
			out = append(out, p)
		}
	}
	return out
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
