package flow

// plan.go — T-B010-04: las etapas del ciclo de planificación.
//
// Fuente de verdad: [[specs/SPEC-CICLO-PLANIFICACION]] §Flujo principal (la
// lista de pasos). La spec deja "las etapas internas de la decisión técnica y
// del diseño por definir"; aquí se toman sus pasos como etapas del flujo, que
// es la lectura mínima y no inventa pasos: cada etapa corresponde a un paso
// numerado de la spec.
//
// Todas las etapas corren con `plan`: el ciclo de planificación es el agente
// que pregunta, investiga y propone. Las escrituras las provoca `build` tras
// la aprobación del archivo correspondiente, no una etapa de este flujo; por
// eso el flujo de planificación no escribe por sí mismo.

import "localcli/internal/tools"

// FlujoPlanificacion devuelve el ciclo oficial de planificación.
func FlujoPlanificacion() Flujo {
	return Flujo{
		Nombre: "planificacion",
		Etapas: []Etapa{
			{ID: "tipo_proyecto", Nombre: "Tipo de proyecto", Agente: tools.AgentePlan},
			{ID: "vision", Nombre: "Visión", Agente: tools.AgentePlan, Aprobacion: true},
			{ID: "requisitos", Nombre: "Requisitos", Agente: tools.AgentePlan, Aprobacion: true},
			{ID: "decision_tecnica", Nombre: "Decisión técnica", Agente: tools.AgentePlan, Aprobacion: true},
			{ID: "documentacion_capas", Nombre: "Documentación por capas", Agente: tools.AgentePlan, Aprobacion: true},
			{ID: "estructura_trabajo", Nombre: "Estructura de trabajo", Agente: tools.AgentePlan, Aprobacion: true},
			{ID: "validacion", Nombre: "Validación", Agente: tools.AgentePlan},
			{ID: "entrega", Nombre: "Entrega", Agente: tools.AgentePlan},
		},
	}
}
