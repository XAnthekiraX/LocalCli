package session

// cola.go — T-B018-14: eventos de cola de alcance de proyecto.
// T-B019-03/04: el bucle que consume la cola, un elemento por iteración, y el
// cierre con aviso cuando se vacía o se detiene.
//
// Fuente de verdad: [[backend/04-infrastructure/EVENTS]] §1/§3 (`queue` es quien
// «cambia», pero el bus de eventos vive en la sesión, así que el orquestador es
// quien publica `cola_actualizada` y `elemento_bloqueado`; son eventos de
// proyecto y no llevan `sesion`), [[specs/SPEC-COLA-TAREAS]] §Flujo principal
// (un elemento por iteración, con su contexto, su aprobación, su verificación y
// su marcado; al vaciarse la cola se detiene y avisa) y
// [[backend/01-domain/BUSINESS_RULES]] §Cola.

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"localcli/internal/flow"
	"localcli/internal/queue"
	"localcli/internal/task"
	"localcli/internal/tui"
)

// ComandoEjecutar es el comando del flujo oficial que arranca la cola. Con él
// el usuario manda ejecutar sin escribir la lista entera (SPEC-COLA-TAREAS:
// «La cola solo avanza cuando el usuario la arranca con `/ejecutar` o confirma
// la propuesta»).
const ComandoEjecutar = "/ejecutar"

// ConCola registra la cola derivada de una capa y publica su estado inicial.
func (o *Orquestador) ConCola(c *queue.Cola) {
	if c == nil {
		return
	}
	o.muCola.Lock()
	if o.colas == nil {
		o.colas = map[task.Capa]*queue.Cola{}
	}
	o.colas[c.Capa()] = c
	o.muCola.Unlock()
	o.emitirCola(c)
}

// Cola devuelve la cola derivada de una capa, si la hay.
func (o *Orquestador) Cola(capa task.Capa) (*queue.Cola, bool) {
	o.muCola.Lock()
	defer o.muCola.Unlock()
	c, ok := o.colas[capa]
	return c, ok
}

// emitirCola publica `cola_actualizada` y, por cada bloqueo, `elemento_bloqueado`.
func (o *Orquestador) emitirCola(c *queue.Cola) {
	v := c.Vista()
	o.emitir(tui.Evento{Tipo: tui.EventoColaActualizada, Datos: map[string]string{
		"capa":        string(v.Capa),
		"pendientes":  strconv.Itoa(v.Pendientes),
		"completados": strconv.Itoa(v.Completados),
		"bloqueados":  strconv.Itoa(len(v.Bloqueados)),
		"elegible":    v.Elegible,
	}})
	for _, b := range v.Bloqueados {
		o.emitir(tui.Evento{Tipo: tui.EventoElementoBloqueado, Datos: map[string]string{
			"capa":     string(v.Capa),
			"elemento": b.ID,
			"falta":    strings.Join(b.Falta, ", "),
			"motivo":   b.Motivo,
		}})
	}
}

// arrancarLaCola es la entrada de `/ejecutar`: elige la capa cuya cola se
// arranca y la recorre. Sin cola derivada no hay nada que arrancar, y eso se
// avisa en vez de fallar en silencio.
func (o *Orquestador) arrancarLaCola(ctx context.Context, sesionID string, f flow.Flujo) error {
	capa, ok := o.capaConCola()
	if !ok {
		o.avisar(sesionID, "no hay ninguna cola derivada que arrancar",
			"revisa ai/tasks/ de la capa y vuelve a lanzar /ejecutar")
		return nil
	}
	return o.arrancarCola(ctx, sesionID, f, capa)
}

// capaConCola devuelve la capa cuya cola arranca `/ejecutar`: la única derivada
// si solo hay una; si hay varias, la primera en el orden de construcción que
// fija la documentación (backend, frontend, database).
func (o *Orquestador) capaConCola() (task.Capa, bool) {
	o.muCola.Lock()
	defer o.muCola.Unlock()
	for _, capa := range capasEnOrden {
		if _, ok := o.colas[capa]; ok {
			return capa, true
		}
	}
	return "", false
}

// arrancarCola consume la cola de una capa: **un elemento por iteración**. Por
// cada elemento pide su contexto, corre las etapas del flujo —que son las que
// proponen el cambio, esperan la aprobación y lo verifican— y solo entonces lo
// marca completado. La cola avanza solo cuando la arranca el usuario: esto se
// llama desde `/ejecutar` o desde la confirmación de una propuesta, nunca desde
// un texto por su cuenta.
func (o *Orquestador) arrancarCola(ctx context.Context, sesionID string, f flow.Flujo, capa task.Capa) error {
	c, ok := o.Cola(capa)
	if !ok {
		o.avisar(sesionID, "no hay cola derivada para la capa "+string(capa),
			"revisa ai/tasks/"+string(capa)+"/")
		return fmt.Errorf("session: no hay cola para la capa %q", capa)
	}
	// Consumir exige el permiso del orquestador: la cola no se lanza por su
	// cuenta (queue/consumo.go).
	cons, err := c.Consumir(queue.PermitirAlOrquestador())
	if err != nil {
		return err
	}

	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		e, err := cons.Siguiente(ctx)
		if err != nil {
			return err
		}
		if e == nil {
			return o.cerrarCola(sesionID, capa, cons)
		}

		o.emitirCola(c)
		// Las etapas del flujo son el trabajo de un elemento: contexto,
		// propuesta, aprobación y verificación. La entrega de la última etapa
		// es la respuesta del turno.
		if err := o.turnoDeEtapas(ctx, sesionID, f, queue.Objetivo(*e)); err != nil {
			// Un elemento que no terminó bien no se marca completado: se
			// retoma donde estaba la próxima vez.
			return err
		}
		if err := cons.Marcar(e.ID, task.EstadoCompletada); err != nil {
			return err
		}
		o.emitirCola(c)
	}
}

// cerrarCola avisa de por qué la cola se detuvo y termina. Los dos finales no se
// confunden: vaciarse es haber terminado; detenerse es que queda trabajo que no
// puede arrancar (SPEC-COLA-TAREAS §Flujos alternativos).
func (o *Orquestador) cerrarCola(sesionID string, capa task.Capa, cons *queue.Consumidor) error {
	v := cons.Vista()
	if v.Pendientes == 0 {
		o.avisar(sesionID, "la cola de "+string(capa)+" se vació",
			"no queda trabajo pendiente en la capa")
		return nil
	}
	motivo := "queda trabajo que no puede arrancar todavía"
	if b := cons.Bloqueados(); len(b) > 0 {
		motivo = b[0].ID + ": " + b[0].Motivo
	}
	o.avisar(sesionID, "la cola de "+string(capa)+" se detuvo: "+motivo,
		"resuelve lo que falta y vuelve a lanzar /ejecutar")
	return nil
}
