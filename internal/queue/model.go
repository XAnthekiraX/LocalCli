// model.go — T-B012-01: la cola como proyección en memoria del TODO.
//
// Fuente de verdad: SPEC-COLA-TAREAS ("La cola es el TODO de esta ejecución",
// "no hay una representación paralela"), DECISIONS.md ("Cola derivada de los
// archivos de tarea, no almacenada: la cola se reconstruye al arrancar y
// refleja el estado real") y DOMAIN.md §3 ("queue no tiene tabla propia… lee
// con task y guarda su vista en memoria, no en la base").
//
// De ahí la forma del tipo: la cola NO guarda estado propio que pueda divergir
// del archivo. Guarda a lo sumo una copia de los elementos leídos, y cualquier
// decisión (qué sigue, qué está bloqueado) se recalcula del TODO releído. La
// identidad de la ejecución es el TODO del que se deriva: raíz del proyecto +
// capa, que es el archivo `ai/tasks/<capa>/` que `task` lee. Dos colas de dos
// TODO distintos no se ven entre sí; no hay cola global.
package queue

import (
	"localcli/internal/task"
)

// Cola es la cola de una ejecución: los elementos de su TODO, en orden de
// ejecución. Es una proyección, no un almacén: no escribe nada por su cuenta.
type Cola struct {
	raiz      string
	capa      task.Capa
	elementos []task.Elemento
}

// Raiz devuelve la raíz del proyecto de la que se deriva la cola.
func (c *Cola) Raiz() string { return c.raiz }

// Capa devuelve la capa cuyo TODO alimenta esta cola.
func (c *Cola) Capa() task.Capa { return c.capa }

// Elementos devuelve los elementos en orden de ejecución. Es una copia: quien
// lo reciba no puede alterar la cola por accidente.
func (c *Cola) Elementos() []task.Elemento {
	out := make([]task.Elemento, len(c.elementos))
	copy(out, c.elementos)
	return out
}

// PorID busca un elemento de esta cola. La segunda devolución distingue "no
// está en la cola" de "está pendiente": sin esa distinción no se puede rechazar
// una tarea suelta, que no existe en la primera versión (DECISIONS.md).
func (c *Cola) PorID(id string) (task.Elemento, bool) {
	for _, e := range c.elementos {
		if e.ID == id {
			return e, true
		}
	}
	return task.Elemento{}, false
}

// Vista es el estado de la cola que se puede mostrar desde cualquier sesión
// (SPEC-COLA-TAREAS: "El estado de la cola es visible desde cualquier sesión").
type Vista struct {
	Capa        task.Capa
	Orden       []task.Elemento
	Bloqueados  []Bloqueo
	Pendientes  int
	Completados int
	// Elegible es el ID del elemento que arrancaría ahora, o "" si nada puede
	// arrancar todavía.
	Elegible string
}

// Vacía informa si no queda trabajo por hacer.
func (v Vista) Vacía() bool { return v.Pendientes == 0 && v.Elegible == "" }
