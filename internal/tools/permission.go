package tools

// permission.go — comprobar que el agente puede pedir la herramienta.
//
// Fuente de verdad: ai/docs/backend/02-interfaces/TOOLS.md §2 y §8,
// ai/docs/backend/03-security/SECURITY.md §2 y ai/docs/backend/DECISIONS.md
// ("Permiso comprobado en tools, aplicado en fileops y exec").
//
// La garantía central no es que este módulo "bloquee" la escritura para `plan`:
// es que `plan` **no tiene** ninguna herramienta de escritura en su catálogo
// efectivo, porque sus permisos no conceden `editar`. El catálogo efectivo se
// DERIVA de los permisos contra el catálogo cerrado (`agent.HerramientasDe`), y
// la comprobación de la capa universal vuelve a mirar la acción: la herramienta
// pertenece a una acción y el agente concede o no esa acción.

// Identidad de los dos agentes base (specs/SPEC-AGENTE-BASE).
const (
	AgentePlan  = "plan"
	AgenteBuild = "build"
)

// HerramientasDePlan devuelve el catálogo del agente base `plan`: leer,
// ejecutar, internet y tareas, nunca editar. Se conserva como valor por defecto
// y para los tests; el catálogo real de cada agente sale de sus `permisos`
// (agent.HerramientasDe).
func HerramientasDePlan() []string {
	var out []string
	out = append(out, NombresDeAccion(AccionLeer)...)
	out = append(out, NombresDeAccion(AccionEjecutar)...)
	out = append(out, NombresDeAccion(AccionInternet)...)
	out = append(out, NombresDeAccion(AccionTareas)...)
	return out
}

// HerramientasDeBuild devuelve el catálogo completo.
func HerramientasDeBuild() []string { return NombresCatalogo() }

// AccionesDePlan y AccionesDeBuild son los permisos por acción de los agentes
// base, para las pruebas y el respaldo.
func AccionesDePlan() []Accion {
	return []Accion{AccionLeer, AccionEjecutar, AccionInternet, AccionTareas}
}

func AccionesDeBuild() []Accion {
	return []Accion{AccionLeer, AccionEditar, AccionEjecutar, AccionInternet, AccionTareas}
}

// ComprobarPermiso valida la petición contra las acciones del agente.
//
//   - Una herramienta fuera del catálogo cerrado es E_TOOL_UNKNOWN, sin importar
//     el agente.
//   - Una acción que el agente no concede es E_TOOL_NOT_ALLOWED. Es el caso de
//     `plan` pidiendo escritura: no la tiene, así que se rechaza.
func ComprobarPermiso(permisos []Accion, nombre string) error {
	h, ok := Buscar(nombre)
	if !ok {
		return nuevoError(CodigoHerramientaDesconocida,
			"la herramienta "+nombre+" no está en el catálogo cerrado")
	}
	if !h.Accion().Permitida(permisos) {
		return nuevoError(CodigoHerramientaNoPermitida,
			"el agente no tiene la acción `"+h.Accion().String()+"`, que es la de "+nombre)
	}
	return nil
}
