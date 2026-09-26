// sync.go — T-B012-06: devolver el estado ejecutado al archivo del TODO.
//
// Fuente de verdad: SPEC-COLA-TAREAS §Reglas ("La cola refleja siempre el estado
// real del TODO"), BUSINESS_RULES.md §Cola ("Un elemento solo se marca
// completado cuando se terminó, con su verificación hecha") y DECISIONS.md
// ("Cola derivada de los archivos de tarea, no almacenada").
//
// El cambio lo aplica `task` (T-B004-04), que escribe solo la celda Estado de la
// fila. `queue` no abre el archivo ni sabe de tablas: pide el cambio y vuelve a
// leer la verdad del archivo cuando haga falta. Si el estado que se pide no es
// escribible en el formato actual del TODO, se dice y no se toca nada.
package queue

import (
	"fmt"

	"localcli/internal/task"
)

// Marcar cambia el estado del elemento en su archivo de TODO. Es la interfaz
// que consume el motor (`flow.Cola`) y por eso lleva exactamente esta firma.
//
// Un ID que no está en la cola se rechaza: sin esa comprobación, un ID mal
// pasado escribiría en el TODO de una tarea que esta ejecución no gobierna.
func (c *Consumidor) Marcar(id string, estado task.Estado) error {
	if _, ok := c.cola.PorID(id); !ok {
		return errSoloMotor("el elemento " + id + " no pertenece al TODO de esta ejecución")
	}
	if !task.EsEstado(string(estado)) {
		return fmt.Errorf("queue: estado inválido %q", estado)
	}
	if estado == task.EstadoBloqueada {
		// Marcarlo en el archivo exigiría escribir `bloqueada_por`, que la
		// tabla del TODO todavía no declara; un bloqueo sin motivo invalida el
		// TODO entero. El bloqueo se informa en la vista (block.go).
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
