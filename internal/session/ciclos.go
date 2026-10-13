package session

// ciclos.go — T-B019-05/06/07: lo que deja cada ciclo oficial al cerrar.
//
// Fuente de verdad: [[backend/01-domain/BUSINESS_RULES]] §Detección de trabajo
// ordenado («El TODO se crea en el flujo que corresponde») y §Cola,
// [[specs/SPEC-COLA-TAREAS]] §Flujos alternativos («Aparece un elemento nuevo:
// la cola se re-deriva y el nuevo entra en su posición») y
// [[specs/SPEC-RESOLVER]] («no crea tareas ni implementa: entregar el PLAN es el
// final del ciclo»).
//
// Los tres ciclos oficiales dejan cosas distintas, y aquí está la diferencia:
//
//   - `/planificar` deja sus **fases en la lista de pasos**, que es el TODO del
//     que se deriva la cola. La etapa que las registra usa `crear_todo`, que es
//     de permiso `read`: la lista de pasos no toca archivos, así que el flujo de
//     planificación sigue siendo lanzable por `plan`.
//   - `/crear`, `/actualizar` y `/eliminar` dejan la tarea escrita en su
//     archivo; aquí solo se **re-deriva** la cola para que entre en su posición.
//   - `/resolver` no deja nada en la cola: su entrega es el PLAN.
//
// Nada de esto ejecuta trabajo: la fase ya corrió; esto comprueba o re-deriva lo
// que quedó.

import (
	"strconv"

	"localcli/internal/flow"
	"localcli/internal/store"
	"localcli/internal/task"
)

// Comandos de los ciclos oficiales, tal como los declaran sus flujos.
const (
	ComandoPlanificar = "/planificar"
	ComandoResolver   = "/resolver"
)

// comandosDeTrabajo son los flujos que meten su tarea en la cola al cerrar.
var comandosDeTrabajo = []string{"/crear", "/actualizar", "/eliminar"}

// capasEnOrden es el orden de construcción que fija la documentación. Es el
// orden en el que se elige la capa a arrancar y en el que se re-derivan las
// colas.
var capasEnOrden = []task.Capa{task.CapaBackend, task.CapaFrontend, task.CapaDatabase}

// cerrarCiclo hace lo que cada ciclo deja al terminar.
func (o *Orquestador) cerrarCiclo(sesionID string, f flow.Flujo) {
	switch {
	case f.Comando == ComandoPlanificar:
		o.comprobarFasesDeLaPlanificacion(sesionID)
	case esComandoDeTrabajo(f.Comando):
		o.rederivarColas(sesionID)
	}
}

// esComandoDeTrabajo dice si un comando es uno de los ciclos de trabajo.
func esComandoDeTrabajo(comando string) bool {
	for _, c := range comandosDeTrabajo {
		if c == comando {
			return true
		}
	}
	return false
}

// comprobarFasesDeLaPlanificacion avisa de lo que dejó la planificación. Su TODO
// son sus fases en la lista de pasos: si cerró sin ninguna, la cola no tiene de
// dónde derivarse y eso se avisa en vez de darse por bueno.
func (o *Orquestador) comprobarFasesDeLaPlanificacion(sesionID string) {
	fases, err := store.ListarTodos(o.db, o.projectID, sesionID)
	if err != nil {
		o.avisar(sesionID, "no se pudo leer la lista de pasos: "+err.Error(), "")
		return
	}
	if len(fases) == 0 {
		o.avisar(sesionID, "la planificación terminó sin dejar fases en la lista de pasos",
			"vuelve a lanzar /planificar o registra las fases a mano")
		return
	}
	o.avisar(sesionID, "la planificación dejó "+strconv.Itoa(len(fases))+" fases en la lista de pasos",
		"lánzalas con /ejecutar cuando quieras")
}

// rederivarColas vuelve a leer los archivos de tarea de cada capa derivada y
// publica la cola nueva: la tarea que se acaba de aprobar entra en su posición
// sin reiniciar nada. Una capa cuyo TODO quedó inválido avisa y se queda como
// estaba; no se inventa una cola a medias.
func (o *Orquestador) rederivarColas(sesionID string) {
	for _, capa := range capasEnOrden {
		c, ok := o.Cola(capa)
		if !ok {
			continue
		}
		if err := c.Rederrivar(); err != nil {
			o.avisar(sesionID, "la cola de "+string(capa)+" no se pudo re-derivar: "+err.Error(),
				"revisa ai/tasks/"+string(capa)+"/")
			continue
		}
		o.emitirCola(c)
	}
}
