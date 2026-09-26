package tools

// permission.go — T-B007-05: comprobar que el agente puede pedir la
// herramienta.
//
// Fuente de verdad: ai/docs/backend/02-interfaces/TOOLS.md §2 y §7,
// ai/docs/backend/03-security/SECURITY.md §2 y ai/docs/backend/DECISIONS.md
// ("Permiso comprobado en tools, aplicado en fileops y exec").
//
// La garantía central no es que este módulo "bloquee" la escritura para
// `plan`: es que `plan` **no tiene** ninguna herramienta de escritura en su
// catálogo. El campo `herramientas` del JSON del agente es la fuente de
// verdad, así que la comprobación recibe esa lista tal cual (el catálogo del
// agente) y solo verifica dos cosas: que la herramienta exista en el catálogo
// cerrado y que el agente la tenga. Todo lo demás —la aprobación concreta, la
// frontera de rutas, la lista blanca— lo aplican `fileops` y `exec`.

// Identidad de los dos agentes base (specs/SPEC-AGENTE-BASE). El nombre es
// para trazabilidad; la lista de herramientas es la que decide de verdad.
const (
	AgentePlan  = "plan"
	AgenteBuild = "build"
)

// HerramientasDePlan devuelve el catálogo del agente base `plan`: solo las
// herramientas de lectura. Que esta lista no contenga ninguna de escritura es
// la garantía; no hay ningún filtro posterior que la pueda debilitar.
func HerramientasDePlan() []string {
	var out []string
	for _, h := range catalogo {
		if h.Modo == Lee {
			out = append(out, h.Nombre)
		}
	}
	return out
}

// HerramientasDeBuild devuelve el catálogo completo: `build` es el único
// agente que crea, modifica y borra.
func HerramientasDeBuild() []string { return NombresCatalogo() }

// ComprobarPermiso valida la petición contra el catálogo del agente.
//
//   - Una herramienta fuera del catálogo cerrado es E_TOOL_UNKNOWN, sin
//     importar el agente.
//   - Una herramienta que el agente no declara es E_TOOL_NOT_ALLOWED. Es el
//     caso de `plan` pidiendo escritura: no la tiene, así que se rechaza.
//
// `permitidas` es el campo `herramientas` del JSON del agente, ya validado
// contra el catálogo por `agent`. Un agente de solo conversación tiene la
// lista vacía y no puede pedir nada.
func ComprobarPermiso(permitidas []string, nombre string) error {
	h, ok := Buscar(nombre)
	if !ok {
		return nuevoError(CodigoHerramientaDesconocida,
			"la herramienta "+nombre+" no está en el catálogo cerrado")
	}
	if !contiene(permitidas, nombre) {
		return nuevoError(CodigoHerramientaNoPermitida,
			"el agente no tiene la herramienta "+nombre+" ("+h.Modo.String()+
				"); usa uno que la tenga en su catálogo")
	}
	return nil
}

// contiene informa si la lista incluye el nombre exacto.
func contiene(lista []string, nombre string) bool {
	for _, n := range lista {
		if n == nombre {
			return true
		}
	}
	return false
}
