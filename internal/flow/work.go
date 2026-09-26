package flow

// work.go — T-B010-05: las etapas del ciclo de trabajo.
//
// Fuente de verdad: [[specs/SPEC-CICLO-TRABAJO]] §El esqueleto, paso a paso
// (los 7 pasos del flujo único con sus tres entradas crear/actualizar/eliminar)
// y §Ejecutar (la ejecución va por la cola, un elemento por iteración).
//
// El relevo es el mecanismo: `plan` analiza y propone, tú apruebas, y `build`
// escribe la documentación, la tarea y la implementación. Por eso las primeras
// etapas corren con `plan` y las de escritura con `build`, y todas las que
// cambian algo pasan por aprobación.
//
// La entrada `eliminar` exige, además, confirmación explícita antes de seguir
// (spec §Entrada 3): en este flujo la representa la aprobación de la etapa de
// documentación, que en una eliminación se plantea como confirmación del
// alcance y de los datos que se pierden.

import (
	"localcli/internal/task"
	"localcli/internal/tools"
)

// FlujoTrabajo devuelve el ciclo oficial de trabajo para una acción.
func FlujoTrabajo(accion task.Accion) Flujo {
	return Flujo{
		Nombre: "trabajo/" + string(accion),
		Etapas: []Etapa{
			// 1-2. Impacto: qué funcionalidades, entidades y datos se ven afectados.
			{ID: "impacto", Nombre: "Analizar el impacto", Agente: tools.AgentePlan},
			// 3. `plan` presenta el plan y espera confirmación.
			{ID: "plan", Nombre: "Presentar el plan", Agente: tools.AgentePlan, Aprobacion: true},
			// 4. Documentación por archivo, con `build` escribiendo tras aprobar.
			{ID: "documentacion", Nombre: "Actualizar la documentación", Agente: tools.AgentePlan, Aprobacion: true},
			// 5. La tarea (o el elemento del TODO) con su acción y su contexto.
			{ID: "tarea", Nombre: "Crear o actualizar la tarea", Agente: tools.AgenteBuild, Aprobacion: true},
			// 7. La ejecución del elemento, un elemento por iteración.
			{ID: "ejecutar", Nombre: "Ejecutar el elemento", Agente: tools.AgenteBuild, Aprobacion: true},
		},
	}
}
