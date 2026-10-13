package queue

// marcar.go — T-B018-12: devolver el estado ejecutado al archivo del TODO.
//
// Fuente de verdad: SPEC-COLA-TAREAS §Reglas ("La cola refleja siempre el estado
// real del TODO"), BUSINESS_RULES.md §Cola ("Un elemento solo se marca
// completado cuando se terminó, con su verificación hecha") y DECISIONS.md
// ("Cola derivada de los archivos de tarea, no almacenada").
//
// El cambio lo aplica `task` (T-B018-06), que escribe solo la celda Estado de
// la fila. `queue` no abre el archivo ni sabe de tablas: pide el cambio y
// vuelve a leer la verdad del archivo cuando haga falta.

import (
	"fmt"

	"localcli/internal/task"
)

// Marcar cambia el estado del elemento en su archivo de TODO. Un ID que no está
// en la cola se rechaza: sin esa comprobación, un ID mal pasado escribiría en
// el TODO de una tarea que esta ejecución no gobierna.
func (c *Consumidor) Marcar(id string, estado task.Estado) error {
	if _, ok := c.cola.PorID(id); !ok {
		return errSoloOrquestador("el elemento " + id + " no pertenece al TODO de esta ejecución")
	}
	if !task.EsEstado(string(estado)) {
		return fmt.Errorf("queue: estado inválido %q", estado)
	}
	if estado == task.EstadoBloqueada {
		// Marcarlo en el archivo exigiría escribir `bloqueada_por`, que la
		// tabla del TODO todavía no declara; un bloqueo sin motivo invalida el
		// TODO entero. El bloqueo se informa en la vista (bloqueo.go).
		return errBloqueoNoEscribible(id)
	}
	if _, err := task.CambiarEstadoElemento(c.cola.raiz, c.cola.capa, id, estado); err != nil {
		return fmt.Errorf("queue: escribir el estado de %s: %w", id, err)
	}
	c.cola.fijarEstado(id, estado)
	return nil
}

// fijarEstado actualiza la copia en memoria para que la vista no mienta entre
// dos lecturas del archivo.
func (c *Cola) fijarEstado(id string, estado task.Estado) {
	for i := range c.elementos {
		if c.elementos[i].ID == id {
			c.elementos[i].Estado = estado
			return
		}
	}
}

// errBloqueoNoEscribible explica por qué un bloqueo no se marca en el archivo.
//
// El frontmatter del elemento admite `bloqueada_por`, pero las tablas reales de
// `MAIN-TASKS.md` y `NNN-task-*.md` no tienen esa columna (task/parseo.go solo
// la lee "si existe"). Escribir `estado = bloqueada` sin `bloqueada_por` dejaría
// el TODO inválido —ValidarReglaBloqueo exige razón— y la carga entera de la
// capa fallaría. Hasta que el formato del TODO tenga esa columna, el bloqueo se
// informa en la vista y no se escribe.
func errBloqueoNoEscribible(id string) error {
	return fmt.Errorf("queue: %s no puede marcarse bloqueada en el archivo: "+
		"la tabla del TODO no declara la columna bloqueada_por y un bloqueo sin "+
		"motivo invalida el TODO; el bloqueo se informa en la vista", id)
}
