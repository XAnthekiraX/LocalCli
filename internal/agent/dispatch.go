package agent

// dispatch.go — T-B006-06: despachar las peticiones de herramienta del modelo
// hacia `tools`.
//
// Fuente de verdad: ai/docs/backend/02-interfaces/TOOLS.md §7 (pasos 1-3: la
// petición llega con nombre y argumentos, se comprueba, se enruta) y
// ai/docs/backend/02-interfaces/INTERFACES-GENERAL.md §5 ("`agent` construye
// la llamada de un agente y despacha sus herramientas").
//
// `agent` no ejecuta la herramienta ni decide permisos: convierte la petición
// del modelo en la `Peticion` de `tools` y deja que `tools` compruebe el
// permiso, valide y enrute. El modelo no puede concederse permisos: lo único
// que aporta es el nombre y los argumentos, y el catálogo del agente lo pone
// el motor desde el JSON.

import (
	"context"
	"encoding/json"

	"localcli/internal/tools"
)

// SolicitudHerramienta es lo que el modelo pide: el nombre de una herramienta
// y sus argumentos. El nombre tiene que estar en el catálogo cerrado; los
// argumentos los valida `tools` contra el contrato de esa herramienta.
type SolicitudHerramienta struct {
	Nombre     string          `json:"nombre"`
	Argumentos json.RawMessage `json:"argumentos"`
}

// Despachador lleva las peticiones de herramienta de un agente a `tools`.
type Despachador struct {
	registro *tools.Registro
}

// NuevoDespachador envuelve el registro de herramientas de `tools`.
func NuevoDespachador(registro *tools.Registro) *Despachador {
	return &Despachador{registro: registro}
}

// Despachar convierte la solicitud en la petición de `tools` y la enruta. El
// catálogo del agente (`a.Herramientas`) viaja tal cual: es lo que sostiene la
// garantía de que `plan` no escribe.
//
// Los argumentos se decodifican contra el contrato de la herramienta; si el
// nombre no está en el catálogo o los argumentos no encajan, `tools` devuelve
// E_TOOL_UNKNOWN o E_BAD_ARGS y la herramienta no se ejecuta.
func (d *Despachador) Despachar(ctx context.Context, a Agente, s SolicitudHerramienta) (any, error) {
	argumentos, err := tools.Decodificar(s.Nombre, s.Argumentos)
	if err != nil {
		return nil, err
	}
	return d.registro.Enrutar(ctx, tools.Peticion{
		Agente:      a.Nombre,
		Permitidas:  a.Herramientas,
		Herramienta: s.Nombre,
		Argumentos:  argumentos,
	})
}
