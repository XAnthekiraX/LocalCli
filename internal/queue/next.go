// next.go — T-B012-05: entregar el siguiente elemento ejecutable a `flow`.
//
// Fuente de verdad: SPEC-COLA-TAREAS §Flujo principal paso 4 ("`build` toma el
// siguiente elemento disponible cuyas dependencias ya estén cumplidas") y §Reglas
// ("Se ejecuta un elemento por iteración, nunca varios a la vez").
//
// `Siguiente` re-deriva la cola en cada llamada. Es lo que hace cierta la regla
// "la cola refleja siempre el estado real del TODO": si el archivo cambió —otra
// sesión lo tocó, el usuario añadió una fase— la decisión se toma sobre lo que
// dice el archivo ahora, no sobre una copia vieja. Elegir el elemento es una
// decisión de lectura; quién marca en_progreso y completada es el motor.
package queue

import (
	"context"
	"strings"

	"localcli/internal/flow"
	"localcli/internal/task"
)

// Siguiente devuelve el siguiente elemento elegible en orden de ejecución, o
// nil si nada puede arrancar (todo completado, o lo que queda está bloqueado).
// Un elemento devuelto puede estar pendiente o ya en progreso: retomar un
// elemento empezado no repite lo completado.
func (c *Consumidor) Siguiente(ctx context.Context) (*flow.ElementoCola, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := c.cola.Rederrivar(); err != nil {
		return nil, err
	}
	elegibles, err := task.Elegibles(c.cola.elementos)
	if err != nil {
		return nil, err
	}
	if len(elegibles) == 0 {
		return nil, nil
	}
	e := elegibles[0]
	return &flow.ElementoCola{ID: e.ID, Objetivo: Objetivo(e)}, nil
}

// Objetivo redacta con qué objetivo se pide el contexto de un elemento.
//
// No hay campo de texto libre en el frontmatter del elemento (DECISIONS.md [28]:
// siete campos, y el contexto se referencia por rutas), así que el objetivo se
// compone con lo que el propio TODO declara: el ID del elemento y los documentos
// que referencia. Inventar una descripción desde `queue` sería inventar el
// trabajo, que es justo lo que este módulo no hace.
func Objetivo(e task.Elemento) string {
	if len(e.Documentos) == 0 {
		return "ejecutar " + e.ID
	}
	return "ejecutar " + e.ID + " (documentado en " + strings.Join(e.Documentos, ", ") + ")"
}

// Vista devuelve el estado de la cola tal como está ahora en el TODO.
func (c *Consumidor) Vista() Vista { return c.cola.Vista() }

// Bloqueados devuelve lo que no puede arrancar y qué le falta, para que el
// consumidor pueda avisar cuando la cola se detiene.
func (c *Consumidor) Bloqueados() []Bloqueo { return Bloqueados(c.cola.Elementos()) }

// Detenida informa si la cola ya no puede avanzar: no hay elegible pero queda
// trabajo. Es el caso en que toca avisar de qué falta (SPEC-COLA-TAREAS: "Al
// vaciarse, la cola se detiene sola y avisa").
func (c *Consumidor) Detenida() bool {
	v := c.cola.Vista()
	return v.Elegible == "" && v.Pendientes > 0
}
