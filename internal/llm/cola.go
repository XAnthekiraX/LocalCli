// cola.go — T-B036-03 (por motor desde T-B037-05): serialización de la
// inferencia.
//
// Fuente de verdad: DECISIONS.md («Una cola de inferencia por motor, no una
// global» y «El testigo FIFO se toma por petición al modelo, no por ejecución
// completa»), INTEGRATIONS.md §Concurrencia («por orden de llegada por motor,
// con una cola propia para cada uno») y BACKEND.md §Concurrencia («Dos
// motores distintos son dos servidores distintos y no se esperan entre sí»).
//
// Diseño:
//   - Un testigo por canal de capacidad 1 = semáforo FIFO. Quien lo recibe
//     genera; al terminar lo devuelve. El runtime entrega los receptores de un
//     canal en orden de llegada, lo que da el orden FIFO real entre sesiones
//     sin necesidad de mutex de cola.
//   - Punto único de estado: Esperando()/Ocupada() y los notificadores solo
//     cambian aquí; la TUI consulta este sitio, no deduce.
package llm

import (
	"context"
	"sync"
)

// ColasInferencia es el conjunto de colas, UNA POR MOTOR (DECISIONS: «Una cola
// de inferencia por motor, no una global»). Dos motores son dos servidores
// distintos: uno genera mientras el otro atiende, y las sesiones del mismo se
// serializan por orden de llegada. El cero NO es utilizable: crear con
// NewColasInferencia.
type ColasInferencia struct {
	mu    sync.Mutex
	colas map[string]*ColaInferencia
	onEsp func(esperando bool) // notificador agregado, compartido por todas
}

// NewColasInferencia crea el conjunto vacío, sin notificador.
func NewColasInferencia() *ColasInferencia {
	return &ColasInferencia{colas: map[string]*ColaInferencia{}}
}

// Para devuelve la cola del motor `id`, creándola a la primera petición. Cada
// motor tiene la suya: las colas de dos motores distintos no se esperan entre
// sí, y las del mismo respetan el orden de llegada.
func (cs *ColasInferencia) Para(idMotor string) *ColaInferencia {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	c, ok := cs.colas[idMotor]
	if !ok {
		c = NewColaInferencia()
		if cs.onEsp != nil {
			c.SetNotificadorGlobal(cs.onEsp)
		}
		cs.colas[idMotor] = c
	}
	return c
}

// SetNotificadorGlobal fija el callback al que TODO cambio de espera se
// reporta, en cualquier motor. Se aplica también a las colas creadas después:
// la capa superior se suscribe una sola vez y no tiene que volver a llamar.
func (cs *ColasInferencia) SetNotificadorGlobal(fn func(esperando bool)) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	cs.onEsp = fn
	for _, c := range cs.colas {
		c.SetNotificadorGlobal(fn)
	}
}

// ColaInferencia serializa los accesos al modelo entre sesiones. El cero-valor
// NO es utilizable: crear con NewColaInferencia.
type ColaInferencia struct {
	paso chan struct{} // capacidad 1: contiene el testigo cuando está libre

	mu      sync.Mutex
	ocupado bool
	esperan int
	onEsp   func(esperando bool) // notificador global del estado de espera
}

// NewColaInferencia crea la cola con el testigo disponible.
func NewColaInferencia() *ColaInferencia {
	c := &ColaInferencia{paso: make(chan struct{}, 1)}
	c.paso <- struct{}{}
	return c
}

// OpcionesEncolar acompaña cada petición de generación.
type OpcionesEncolar struct {
	// IdSesion permite correlacionar esperas (informativo; la cola no decide).
	IdSesion string
	// AlEsperar, si no es nil, se llama con true al entrar en espera y con
	// false al conseguir el testigo (o morir intentándolo).
	AlEsperar func(esperando bool)
}

// Encolar adquiere el testigo FIFO y ejecuta fn con él. Solo una llamada está
// dentro de fn a la vez; el orden de entrada se respeta. Si ctx se cancela
// antes de adquirir, devuelve ctx.Err() sin ejecutar fn y sin perder el testigo.
func (c *ColaInferencia) Encolar(ctx context.Context, op OpcionesEncolar, fn func(context.Context) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	// Intento rápido: si el testigo está en el buffer, comprarlo ya. Si no,
	// esta petición pasa a esperar (notificación desde el punto único).
	var testigo struct{}
	compradoDirecto := false
	select {
	case testigo = <-c.paso:
		compradoDirecto = true
	default:
		c.notificar(op, true)
	}

	if !compradoDirecto {
		select {
		case testigo = <-c.paso:
			c.notificar(op, false)
		case <-ctx.Done():
			c.notificar(op, false)
			return ctx.Err()
		}
	}

	c.marcarOcupado(true)
	defer func() {
		c.marcarOcupado(false)
		c.paso <- testigo // liberar el testigo
	}()
	return fn(ctx)
}

// Esperando informa si hay alguna sesión esperando al modelo AHORA. Punto único
// consultable por la TUI.
func (c *ColaInferencia) Esperando() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.esperan > 0
}

// Ocupada informa si una generación está en curso.
func (c *ColaInferencia) Ocupada() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ocupado
}

// SetNotificadorGlobal fija el callback al que TODO cambio de espera agregada
// se reporta (la capa superior se suscribe una sola vez).
func (c *ColaInferencia) SetNotificadorGlobal(fn func(esperando bool)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onEsp = fn
}

func (c *ColaInferencia) marcarOcupado(v bool) {
	c.mu.Lock()
	c.ocupado = v
	c.mu.Unlock()
}

func (c *ColaInferencia) notificar(op OpcionesEncolar, entrando bool) {
	c.mu.Lock()
	if entrando {
		c.esperan++
	} else if c.esperan > 0 {
		c.esperan--
	}
	total := c.esperan
	global := c.onEsp
	c.mu.Unlock()
	if op.AlEsperar != nil {
		op.AlEsperar(entrando)
	}
	if global != nil {
		global(total > 0)
	}
}
