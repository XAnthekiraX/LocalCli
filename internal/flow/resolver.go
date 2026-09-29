package flow

// resolver.go — el ciclo oficial de resolver problemas.
//
// Fuente de verdad: [[specs/SPEC-RESOLVER]]. Es un ciclo aparte: no amplía el
// alcance del proyecto ni implementa nada. El resolver investiga, diagnostica y
// entrega un PLAN de solución; NO aplica cambios. Su flujo es un recorrido de
// nueve pasos que empieza recibiendo la tarea, sigue por entender el problema y
// buscar contexto, investiga y diagnostica, identifica los archivos afectados,
// diseña la solución, arma el plan de ejecución y termina entregando el PLAN.
//
// Esta es la definición OFICIAL de respaldo (funciona sin configuración). Un
// proyecto puede personalizarla con `.localcli/flows/resolver.json`, que la
// sobreescribe por comando; una prueba de integración los mantiene en sincronía.

import "localcli/internal/tools"

// ReglasResolver son las reglas de comportamiento del ciclo de resolver. Viajan
// con el contexto de cada etapa (ver engine.go): son las que obligan a resolver
// las seis preguntas y a cerrar con la salida estándar, sin implementar.
var ReglasResolver = []string{
	"AGENTS.md (en la raíz del proyecto, no en ai/docs) es la puerta de entrada: identifica las capas y el flujo aplicable antes de nada.",
	"ai/docs/PROJECT.md es el índice: úsalo para orientarte.",
	"No cargues todo ai/docs: pide solo los documentos relevantes siguiendo las dependencias declaradas en el frontmatter.",
	"No afirmes nada sin evidencia: cita el archivo y la línea.",
	"El resolver NO implementa: investiga, diagnostica y entrega un plan de solución.",
	"Antes de terminar debes poder responder estas seis preguntas: qué está pasando, qué debería pasar, dónde está el problema, por qué ocurre, qué archivos están involucrados y qué hay que hacer para solucionarlo.",
	"Cada paso devuelve solo lo suyo, en pocas líneas; el último compone la salida estándar con lo que devolvieron los pasos anteriores.",
	"Cierra siempre con la salida estándar: Diagnóstico; Archivos involucrados (Modificar, Consultar, No modificar); Solución; Plan; Resultado esperado.",
}

// FlujoResolver devuelve el ciclo oficial de resolución de problemas: investiga,
// diagnostica y entrega un plan, sin implementar.
func FlujoResolver() Flujo {
	return Flujo{
		Nombre:      "resolver",
		Comando:     "/resolver",
		Descripcion: "resolver un problema",
		Peticion:    "diagnosticar un problema y entregar un plan de solución",
		Pregunta:    "un diagnóstico del problema y un plan de solución",
		Reglas:      ReglasResolver,
		Etapas: []Etapa{
			// 1. Recibir la tarea.
			{ID: "recibir_tarea", Nombre: "Recibir la tarea", Agente: tools.AgentePlan,
				Pregunta:    "¿cuál es la tarea a resolver?",
				Instruccion: "Toma la petición del usuario como la tarea a resolver."},
			// 2. Entender el problema: qué se pidió y qué se espera.
			{ID: "entender_problema", Nombre: "Entender el problema", Agente: tools.AgentePlan,
				Pregunta:    "¿qué se pidió y qué comportamiento se espera?",
				Instruccion: "Aclara qué se pidió y qué comportamiento se espera."},
			// 3. Buscar contexto: documentación, código, tests y configuración.
			{ID: "buscar_contexto", Nombre: "Buscar contexto", Agente: tools.AgentePlan,
				Pregunta:    "¿qué documentación y archivos son relevantes?",
				Instruccion: "Reúne la documentación, la estructura del código, los archivos relacionados, los tests y la configuración relevantes; solo lo necesario."},
			// 4. Investigar el problema: seguir el flujo real y verificar hipótesis.
			{ID: "investigar", Nombre: "Investigar el problema", Agente: tools.AgentePlan,
				Pregunta:    "¿dónde ocurre el problema y qué lo confirma?",
				Instruccion: "Sigue el flujo real, localiza dónde ocurre, verifica hipótesis y ejecuta pruebas si hace falta."},
			// 5. Diagnosticar: causa, comportamiento actual y esperado.
			{ID: "diagnosticar", Nombre: "Diagnosticar", Agente: tools.AgentePlan,
				Pregunta:    "¿cuál es la causa y qué comportamiento se espera?",
				Instruccion: "Determina la causa, el comportamiento actual y el comportamiento esperado."},
			// 6. Identificar los archivos afectados.
			{ID: "archivos_afectados", Nombre: "Identificar los archivos afectados", Agente: tools.AgentePlan,
				Pregunta:    "¿qué archivos se modifican, se consultan o no se tocan?",
				Instruccion: "Clasifica los archivos en MODIFICAR, CONSULTAR y NO TOCAR, con el motivo de cada uno."},
			// 7. Diseñar la solución.
			{ID: "diseno", Nombre: "Diseñar la solución", Agente: tools.AgentePlan,
				Pregunta:    "¿qué hay que cambiar y cómo se relacionan los cambios?",
				Instruccion: "Define qué cambiar, dónde cambiarlo y cómo se relacionan los cambios."},
			// 8. Crear el plan de ejecución.
			{ID: "plan_ejecucion", Nombre: "Crear el plan de ejecución", Agente: tools.AgentePlan,
				Pregunta:    "¿cuáles son los pasos concretos, en orden, con su validación?",
				Instruccion: "Escribe los pasos concretos, en orden, con su validación."},
			// 9. Entregar el PLAN. No se implementa. Es la entrega: corre sin herramientas.
			{ID: "entregar_plan", Nombre: "Entregar el PLAN", Agente: tools.AgentePlan,
				Pregunta:    "¿cuál es el PLAN de solución con la salida estándar?",
				Instruccion: "Entrega el PLAN con la salida estándar, componiéndola con lo que devolvieron los pasos anteriores. No implementes nada.",
				Entrega:     true},
		},
	}
}
