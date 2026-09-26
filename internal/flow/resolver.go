package flow

// resolver.go — T-B010-06: las etapas del ciclo resolver.
//
// Fuente de verdad: [[specs/SPEC-RESOLVER]] §Flujo principal (los 8 pasos) y
// §Reglas de negocio ("nunca se propone una solución sin evidencia"; "la
// solución la aplica `build` después del relevo"; "después de aplicar, se
// verifica").
//
// Es un ciclo aparte: no amplía el alcance del proyecto ni crea tareas. Por eso
// su flujo no incluye las etapas de documentación y de tarea del ciclo de
// trabajo; si el arreglo necesitara una funcionalidad nueva, se deriva al ciclo
// de trabajo (spec §Qué lo distingue).

import "localcli/internal/tools"

// FlujoResolver devuelve el ciclo oficial de arreglo de problemas.
func FlujoResolver() Flujo {
	return Flujo{
		Nombre: "resolver",
		Etapas: []Etapa{
			// 2. `build` captura evidencia: logs, mensajes de error, fallos.
			{ID: "evidencia", Nombre: "Capturar evidencia", Agente: tools.AgenteBuild},
			// 3. Localizar el código relacionado.
			{ID: "localizar", Nombre: "Localizar el código", Agente: tools.AgentePlan},
			// 4. Identificar la causa raíz, con evidencia.
			{ID: "causa_raiz", Nombre: "Identificar la causa raíz", Agente: tools.AgentePlan},
			// 5-6. Presentar la propuesta y esperar aprobación.
			{ID: "propuesta", Nombre: "Proponer la solución", Agente: tools.AgentePlan, Aprobacion: true},
			// 7. `build` aplica la solución aprobada.
			{ID: "aplicar", Nombre: "Aplicar la solución", Agente: tools.AgenteBuild, Aprobacion: true},
			// 8. Verificar que el problema quedó resuelto.
			{ID: "verificar", Nombre: "Verificar la solución", Agente: tools.AgenteBuild},
		},
	}
}
