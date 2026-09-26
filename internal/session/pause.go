// pause.go — T-B013-06: pausar, reanudar y cancelar el trabajo en curso.
//
// Fuente de verdad: SPEC-COLA-TAREAS §Flujos alternativos ("El usuario cancela
// un elemento en curso: se detiene y lo que estaba en curso no se aplica", "El
// usuario pausa la cola: se detiene después del elemento actual") y §Reglas ("Se
// ejecuta un elemento por iteración").
//
// Pausar se implementa donde el motor ya mira: en la cola. El motor pide el
// siguiente elemento entre iteraciones, así que una cola que responde "no hay
// siguiente" mientras está pausada detiene el trabajo justo después del elemento
// en curso — sin matar nada a mitad y sin que `flow` tenga que saber que existe
// la pausa. Cancelar es otra cosa: corta el contexto, y el motor ya trata el
// corte como cancelación, no como fallo.
package session

import (
	"context"
	"fmt"

	"localcli/internal/flow"
	"localcli/internal/task"
)

// colaPausable envuelve una cola y deja de entregar elementos mientras la
// sesión esté pausada. Delega todo lo demás en la cola de verdad.
type colaPausable struct {
	cola    flow.Cola
	pausada func() bool
}

// Siguiente devuelve el siguiente elemento, o nil si la sesión está pausada:
// así el motor termina el elemento en curso y se detiene.
func (c *colaPausable) Siguiente(ctx context.Context) (*flow.ElementoCola, error) {
	if c.pausada != nil && c.pausada() {
		return nil, nil
	}
	return c.cola.Siguiente(ctx)
}

// Marcar delega: marcar el elemento en curso no se pausa.
func (c *colaPausable) Marcar(id string, estado task.Estado) error { return c.cola.Marcar(id, estado) }

// ConsumirCola ejecuta una cola delegando en el motor, un elemento por
// iteración. Devuelve en cuanto el trabajo quedó arrancado: el desenlace llega
// por eventos.
func (g *Gestor) ConsumirCola(ctx context.Context, sesionID string, cola flow.Cola) error {
	if cola == nil {
		return fmt.Errorf("session: no hay cola que consumir")
	}
	ses, err := g.sesion(sesionID)
	if err != nil {
		return err
	}
	if err := g.cambiarEstado(ses, EstadoTrabajando); err != nil {
		return err
	}
	t := &trabajo{}
	envuelta := &colaPausable{cola: cola, pausada: g.lectorDePausa(t)}
	t.cola = envuelta
	g.lanzar(ctx, sesionID, t, func(c context.Context) error {
		err := g.Motor.ConsumirCola(c, envuelta, g.flujoOFectivo())
		var estadoFlujo flow.EstadoFlujo
		if err == nil {
			estadoFlujo = flow.EstadoTerminado
		} else {
			estadoFlujo = flow.EstadoConError
		}
		return g.cerrarTurno(sesionID, estadoFlujo, err)
	})
	return nil
}

// flujoOFectivo devuelve el flujo configurado o el de por defecto.
func (g *Gestor) flujoOFectivo() flow.Flujo {
	if g.Flujo.Nombre == "" {
		return FlujoPorDefecto()
	}
	return g.Flujo
}

// lectorDePausa devuelve la lectura segura del flag de pausa de un trabajo.
func (g *Gestor) lectorDePausa(t *trabajo) func() bool {
	return func() bool {
		g.mu.Lock()
		defer g.mu.Unlock()
		return t.pausada
	}
}

// trabajoDe devuelve el trabajo en curso de una sesión, o nil.
func (g *Gestor) trabajoDe(sesionID string) *trabajo {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.enCurso[sesionID]
}

// Pausar detiene la sesión después del elemento en curso. No hay elemento en
// curso ⇒ no hay nada que pausar, y se avisa en vez de fingir que sí.
func (g *Gestor) Pausar(sesionID string) error {
	t := g.trabajoDe(sesionID)
	if t == nil {
		return fmt.Errorf("session: la sesión %s no tiene trabajo en curso que pausar", sesionID)
	}
	g.mu.Lock()
	t.pausada = true
	g.mu.Unlock()
	return nil
}

// Reanudar vuelve a dejar pasar elementos a la cola.
func (g *Gestor) Reanudar(sesionID string) error {
	t := g.trabajoDe(sesionID)
	if t == nil {
		return fmt.Errorf("session: la sesión %s no tiene trabajo en curso que reanudar", sesionID)
	}
	g.mu.Lock()
	t.pausada = false
	g.mu.Unlock()
	return nil
}

// Pausada informa si el trabajo de la sesión está pausado.
func (g *Gestor) Pausada(sesionID string) bool {
	t := g.trabajoDe(sesionID)
	if t == nil {
		return false
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	return t.pausada
}

// Cancelar corta el trabajo en curso. Lo que ya se aplicó permanece (ERRORS.md:
// "un error no borra trabajo ya hecho"); lo que no llegó a aplicarse, no se
// aplica.
func (g *Gestor) Cancelar(sesionID string) {
	t := g.trabajoDe(sesionID)
	if t != nil && t.cancel != nil {
		t.cancel()
	}
}

// Rederrivar re-deriva la cola en curso del TODO, para que un elemento nuevo
// entre en su posición sin reiniciar la ejecución (SPEC-COLA-TAREAS). Si no hay
// cola viva, no hay nada que re-derivar y se dice.
func (g *Gestor) Rederrivar(sesionID string) error {
	t := g.trabajoDe(sesionID)
	if t == nil || t.cola == nil {
		return fmt.Errorf("session: la sesión %s no tiene cola en curso que re-derivar", sesionID)
	}
	if r, ok := t.cola.cola.(interface{ Rederrivar() error }); ok {
		return r.Rederrivar()
	}
	return fmt.Errorf("session: la cola en curso no se puede re-derivar")
}
