package tools

// permission.go — comprobar que el agente puede pedir la herramienta.
//
// Fuente de verdad: ai/docs/backend/02-interfaces/TOOLS.md §2 y §8,
// ai/docs/backend/03-security/SECURITY.md §2 y ai/docs/backend/DECISIONS.md
// ("Permiso comprobado en tools, aplicado en fileops y exec").
//
// La garantía central no es que este módulo "bloquee" la escritura para `plan`:
// es que `plan` **no tiene** ninguna herramienta de escritura en su catálogo
// efectivo, porque sus permisos no conceden `write` ni `edit`. El catálogo
// efectivo se DERIVA de los permisos contra el catálogo cerrado
// (`agent.HerramientasDe`), y la comprobación de la capa universal vuelve a
// mirar el permiso: la herramienta declara el suyo y el agente concede o no ese
// permiso.

// Identidad de los dos agentes base (specs/SPEC-AGENTE-BASE).
const (
	AgentePlan  = "plan"
	AgenteBuild = "build"
)

// HerramientasDePlan devuelve el catálogo del agente base `plan`: solo
// herramientas de lectura, nunca de escritura. Se conserva como valor por
// defecto y para los tests; el catálogo real de cada agente sale de sus
// `permissions` (agent.HerramientasDe).
func HerramientasDePlan() []string {
	return NombresDePermiso(PermisoRead)
}

// HerramientasDeBuild devuelve el catálogo completo.
func HerramientasDeBuild() []string { return NombresCatalogo() }

// PermisosDePlan y PermisosDeBuild son los permisos de los agentes base, para
// las pruebas y el respaldo.
func PermisosDePlan() []Permiso {
	return []Permiso{PermisoRead}
}

func PermisosDeBuild() []Permiso {
	return []Permiso{PermisoRead, PermisoWrite, PermisoEdit}
}

// ComprobarPermiso valida la petición contra los permisos del agente.
//
//   - Una herramienta fuera del catálogo cerrado es E_TOOL_UNKNOWN, sin importar
//     el agente.
//   - Un permiso que el agente no concede es E_TOOL_NOT_ALLOWED. Es el caso de
//     `plan` pidiendo escritura: no la tiene, así que se rechaza.
func ComprobarPermiso(permisos []Permiso, nombre string) error {
	h, ok := Buscar(nombre)
	if !ok {
		return nuevoError(CodigoHerramientaDesconocida,
			"la herramienta "+nombre+" no está en el catálogo cerrado")
	}
	if !h.Permiso.Permitida(permisos) {
		return nuevoError(CodigoHerramientaNoPermitida,
			"el agente no tiene el permiso `"+h.Permiso.String()+"`, que es el de "+nombre)
	}
	return nil
}
