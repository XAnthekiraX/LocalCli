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
		Nombre:      "planificacion",
		Comando:     "/planificar",
		Descripcion: "planificar el proyecto desde cero",
		Peticion:    "planificar el proyecto desde cero",
		Pregunta:    "un plan de proyecto desde cero",
		Etapas: []Etapa{
			{ID: "tipo_proyecto", Nombre: "Tipo de proyecto", Agente: tools.AgentePlan,
				Pregunta: "¿qué tipo de proyecto es?"},
			{ID: "vision", Nombre: "Visión", Agente: tools.AgentePlan, Aprobacion: true,
				Pregunta: "¿cuál es la visión del proyecto?"},
			{ID: "requisitos", Nombre: "Requisitos", Agente: tools.AgentePlan, Aprobacion: true,
				Pregunta: "¿qué requisitos tiene el proyecto?"},
			{ID: "decision_tecnica", Nombre: "Decisión técnica", Agente: tools.AgentePlan, Aprobacion: true,
				Pregunta: "¿qué decisiones técnicas se toman?"},
			{ID: "documentacion_capas", Nombre: "Documentación por capas", Agente: tools.AgentePlan, Aprobacion: true,
				Pregunta: "¿qué documentación por capas hace falta?"},
			{ID: "estructura_trabajo", Nombre: "Estructura de trabajo", Agente: tools.AgentePlan, Aprobacion: true,
				Pregunta: "¿cómo se estructura el trabajo?"},
			{ID: "validacion", Nombre: "Validación", Agente: tools.AgentePlan,
				Pregunta: "¿qué hay que validar?"},
			{ID: "entrega", Nombre: "Entrega", Agente: tools.AgentePlan,
				Pregunta: "¿cuál es el plan de proyecto entregado?",
				Entrega:  true},
		},
	}
}
