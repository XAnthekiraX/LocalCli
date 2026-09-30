package flow

// pruebas_flujos_test.go — flujos de prueba del motor. Los flujos del proyecto
// viven en `.localcli/flows/*.json` (son dato, no código): aquí solo se arman los
// mínimos que cada prueba del motor necesita, sin depender de ningún flujo
// oficial.

import "localcli/internal/tools"

// flujoDeDosEtapas es el flujo mínimo: una fase y su entrega.
func flujoDeDosEtapas() Flujo {
	return Flujo{
		Nombre:   "dos",
		Comando:  "/dos",
		Pregunta: "el resultado",
		Etapas: []Etapa{
			{ID: "fase", Nombre: "Fase única", Agente: tools.AgentePlan,
				Pregunta: "¿fase?", Instruccion: "haz la fase"},
			{ID: "entrega", Nombre: "Entregar", Agente: tools.AgentePlan,
				Pregunta: "¿entrega?", Instruccion: "No implementes nada.", Entrega: true},
		},
	}
}

// flujoConAprobacion mezcla `plan` y `build` con aprobaciones: sirve para el
// relevo plan → aprobación → build y para los eventos de pausa/reanudación.
func flujoConAprobacion() Flujo {
	return Flujo{
		Nombre:   "trabajo",
		Comando:  "/trabajo",
		Pregunta: "el resultado del trabajo",
		Reglas:   []string{"regla uno", "regla dos"},
		Etapas: []Etapa{
			{ID: "analisis", Nombre: "Analizar el impacto", Agente: tools.AgentePlan,
				Pregunta: "¿qué impacto tiene?"},
			{ID: "plan", Nombre: "Presentar el plan", Agente: tools.AgentePlan, Aprobacion: true,
				Pregunta: "¿cuál es el plan?"},
			{ID: "ejecucion", Nombre: "Ejecutar el elemento", Agente: tools.AgenteBuild, Aprobacion: true,
				Pregunta: "¿qué se ejecutó?"},
			{ID: "entrega", Nombre: "Entregar", Agente: tools.AgentePlan,
				Pregunta: "¿qué se hizo?", Instruccion: "No implementes nada.", Entrega: true},
		},
	}
}
