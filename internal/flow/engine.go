package flow

// engine.go — T-B010-07 y T-B010-08: el motor.
//
// Fuente de verdad: ai/docs/backend/04-infrastructure/EVENTS.md §1 y §3 (los
// eventos del motor y sus payloads) y
// ai/docs/backend/01-domain/BUSINESS_RULES.md §Motor de etapas y §Cola ("se
// ejecuta un elemento del TODO por iteración, nunca varios a la vez").
//
// El motor encadena etapas, decide seguir/parar/esperar tras cada una y emite
// el evento que corresponde. No habla con el modelo ni con las herramientas por
// su cuenta: pide contexto y ejecución a través de sus dependencias (`context`
// y `agent`), que se conectan como interfaces para poder probar el motor con
// stubs.

import (
	"context"
	"fmt"

	"localcli/internal/task"
)

// Nombres de los eventos que emite el motor (EVENTS.md §1). No se inventa
// ninguno: lo que no está aquí, no se emite.
const (
	EventoEtapaIniciada  = "etapa_iniciada"
	EventoEtapaTerminada = "etapa_terminada"
	EventoEtapaFallida   = "etapa_fallida"
	EventoFlujoPausado   = "flujo_pausado"
	EventoFlujoReanudado = "flujo_reanudado"
	EventoFlujoCancelado = "flujo_cancelado"
)

// Evento es una notificación del motor. El payload lleva lo mínimo para pintar
// o decidir (EVENTS.md §3).
type Evento struct {
	Nombre string
	Datos  map[string]string
}

// Emisor publica eventos. Un consumidor lento no bloquea al productor: quien lo
// implemente decide cómo (un canal con cola).
type Emisor interface {
	Emitir(Evento)
}

// EmisorFunc adapta una función a Emisor.
type EmisorFunc func(Evento)

func (f EmisorFunc) Emitir(e Evento) { f(e) }

// Resultado es lo que produce una etapa.
type Resultado struct {
	Texto       string
	Herramienta string
}

// Contexto entrega a una etapa solo el contexto que necesita. Lo implementa
// `context` (T-B011); el motor no lo arma por su cuenta.
type Contexto interface {
	ContextoPara(ctx context.Context, objetivo string) (string, error)
}

// Agente ejecuta una etapa con el agente indicado (`plan` o `build`) sobre el
// contexto que recibe. Lo implementa `agent` (T-B006). No ejecuta herramientas
// aquí: eso es cosa de `agent` → `tools`.
type Agente interface {
	Ejecutar(ctx context.Context, agente, contexto string) (Resultado, error)
}

// Aprobador pide la decisión del usuario para una etapa que la requiere.
type Aprobador interface {
	Aprobar(ctx context.Context, descripcion string) (bool, error)
}

// Motor encadena las etapas de un flujo.
type Motor struct {
	Contexto Contexto
	Agente   Agente
	// Aprobador es necesario solo si el flujo tiene etapas con Aprobacion.
	Aprobador Aprobador
	// Eventos es opcional: sin emisor, el motor trabaja igual.
	Eventos Emisor
}

func (m *Motor) emitir(nombre string, datos map[string]string) {
	if m.Eventos == nil {
		return
	}
	m.Eventos.Emitir(Evento{Nombre: nombre, Datos: datos})
}

// EjecutarFlujo corre las etapas en orden. Devuelve el estado final y, si algo
// falló o se canceló, el error correspondiente (E_STAGE_FAILED o
// E_FLOW_CANCELLED). Un flujo con una etapa que falla NO encadena la siguiente
// (BUSINESS_RULES.md §Invariantes).
func (m *Motor) EjecutarFlujo(ctx context.Context, f Flujo, objetivo string) (EstadoFlujo, error) {
	if err := f.Validar(); err != nil {
		return EstadoConError, err
	}
	if m.Contexto == nil || m.Agente == nil {
		return EstadoConError, fmt.Errorf("flow: el motor no tiene contexto y agente conectados")
	}

	estado := EstadoPendiente
	var err error
	if estado, err = EstadoTrasDecision(estado, Seguir); err != nil {
		return EstadoConError, err
	}

	for _, etapa := range f.Etapas {
		if cerr := ctx.Err(); cerr != nil {
			m.emitir(EventoFlujoCancelado, map[string]string{"flujo": f.Nombre, "etapa": etapa.ID})
			return EstadoDetenido, nuevoError(CodigoFlujoCancelado,
				"el flujo "+f.Nombre+" se canceló en la etapa "+etapa.ID)
		}

		m.emitir(EventoEtapaIniciada, map[string]string{"flujo": f.Nombre, "etapa": etapa.ID})

		contexto, cErr := m.Contexto.ContextoPara(ctx, objetivo)
		if cErr != nil {
			m.emitir(EventoEtapaFallida, map[string]string{"flujo": f.Nombre, "etapa": etapa.ID})
			return EstadoConError, fmt.Errorf("%w: %s: %v", ErrEtapaFallida, etapa.ID, cErr)
		}
		res, aErr := m.Agente.Ejecutar(ctx, etapa.Agente, contexto)
		if aErr != nil {
			m.emitir(EventoEtapaFallida, map[string]string{"flujo": f.Nombre, "etapa": etapa.ID})
			return EstadoConError, fmt.Errorf("%w: %s: %v", ErrEtapaFallida, etapa.ID, aErr)
		}

		if etapa.Aprobacion {
			// El flujo se pausa y se retoma en la MISMA etapa.
			estado, err = EstadoTrasDecision(estado, EsperarAprobacion)
			if err != nil {
				return EstadoConError, err
			}
			m.emitir(EventoFlujoPausado, map[string]string{"flujo": f.Nombre, "etapa": etapa.ID})
			if m.Aprobador == nil {
				return EstadoPausadoPermiso, nuevoError(CodigoFlujoCancelado,
					"la etapa "+etapa.ID+" necesita aprobación y no hay aprobador conectado")
			}
			ok, pErr := m.Aprobador.Aprobar(ctx, "aprobar la etapa "+etapa.Nombre)
			if pErr != nil {
				return EstadoConError, pErr
			}
			if !ok {
				m.emitir(EventoFlujoCancelado, map[string]string{"flujo": f.Nombre, "etapa": etapa.ID})
				return EstadoDetenido, nuevoError(CodigoFlujoCancelado,
					"el usuario declinó la etapa "+etapa.ID)
			}
			estado, err = EstadoTrasDecision(estado, Seguir)
			if err != nil {
				return EstadoConError, err
			}
			m.emitir(EventoFlujoReanudado, map[string]string{"flujo": f.Nombre, "etapa": etapa.ID})
		}

		m.emitir(EventoEtapaTerminada, map[string]string{
			"flujo":   f.Nombre,
			"etapa":   etapa.ID,
			"resumen": res.Texto,
		})
	}

	estado = EstadoTerminado
	return estado, nil
}

// ElementoCola es un elemento del TODO listo para ejecutarse.
type ElementoCola struct {
	ID       string
	Objetivo string
}

// Cola es lo que el motor necesita para consumir el TODO. Lo implementa `queue`
// (T-B012), que deriva la cola de los archivos de tarea.
type Cola interface {
	// Siguiente devuelve el siguiente elemento elegible, o nil si la cola está
	// vacía.
	Siguiente(ctx context.Context) (*ElementoCola, error)
	// Marcar cambia el estado del elemento en su archivo de tarea.
	Marcar(id string, estado task.Estado) error
}

// ConsumirCola ejecuta la cola entera, un elemento por iteración. Cada elemento
// se marca en_progreso, se ejecuta como un flujo y se marca completada; nunca
// hay dos en_progreso a la vez. Al vaciarse, se detiene.
func (m *Motor) ConsumirCola(ctx context.Context, cola Cola, f Flujo) error {
	if cola == nil {
		return fmt.Errorf("flow: no hay cola que consumir")
	}
	enProgreso := ""
	for {
		if err := ctx.Err(); err != nil {
			return nuevoError(CodigoFlujoCancelado, "la cola se canceló")
		}
		elem, err := cola.Siguiente(ctx)
		if err != nil {
			return err
		}
		if elem == nil {
			return nil // cola vacía: se detiene sola
		}
		// Invariante de un elemento por iteración: no se arranca otro mientras
		// el anterior siga en progreso.
		if enProgreso != "" {
			return fmt.Errorf("flow: ya hay un elemento en progreso (%s); se ejecuta uno por iteración", enProgreso)
		}
		if err := cola.Marcar(elem.ID, task.EstadoEnProgreso); err != nil {
			return err
		}
		enProgreso = elem.ID

		if _, err := m.EjecutarFlujo(ctx, f, elem.Objetivo); err != nil {
			// El elemento queda bloqueado con lo que falta; la cola no sigue
			// con él, pero tampoco se marca completado.
			_ = cola.Marcar(elem.ID, task.EstadoBloqueada)
			return err
		}
		if err := cola.Marcar(elem.ID, task.EstadoCompletada); err != nil {
			return err
		}
		enProgreso = ""
	}
}
