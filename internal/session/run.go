// run.go — T-B013-05: arrancar el flujo de una sesión delegando en `flow`.
//
// Fuente de verdad: BACKEND.md (la cadena `session → flow`), DOMAIN.md §3 y
// BUSINESS_RULES.md §Sesiones ("Cambiar de sesión no detiene nada", "una sesión
// en segundo plano sigue trabajando"). `session` no ejecuta etapas ni elementos:
// pide el flujo y espera el resultado. Su trabajo es el ciclo de vida y el
// estado, no el trabajo en sí.
//
// El mensaje del usuario se guarda ANTES de arrancar: si el modelo o el flujo
// fallan, lo que el usuario pidió sigue en el historial y la sesión queda en
// error, no en un limbo sin rastro. El cierre del turno (mensaje del agente +
// razonamiento + estado) es una sola transacción en `store` (DATA_FLOW.md).
package session

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"localcli/internal/flow"
	"localcli/internal/store"
	"localcli/internal/task"
)

// Motor es lo que `session` necesita de `flow`: arrancar un flujo y consumir una
// cola. Lo implementa `flow.Motor`.
type Motor interface {
	EjecutarFlujo(ctx context.Context, f flow.Flujo, objetivo string) (flow.EstadoFlujo, error)
	ConsumirCola(ctx context.Context, cola flow.Cola, f flow.Flujo) error
}

// El motor real cumple la interfaz que session necesita.
var _ Motor = (*flow.Motor)(nil)

// FlujoPorDefecto es el ciclo de trabajo de la acción `crear`, que es el que usa
// una sesión cuando no se le dice otro. El flujo concreto de una petición lo
// decide quien la interpreta (SPEC-CICLO-TRABAJO); session no lo adivina.
func FlujoPorDefecto() flow.Flujo { return flow.FlujoTrabajo(task.AccionCrear) }

// ObjetivoDeMensaje construye el objetivo que se le pasa a `flow` a partir del
// mensaje del usuario. Es el texto tal cual: interpretarlo es de `flow` y del
// agente, no de `session`.
func ObjetivoDeMensaje(texto string) string { return strings.TrimSpace(texto) }

// Enviar guarda el mensaje del usuario y arranca el flujo en segundo plano.
// Devuelve en cuanto el trabajo quedó arrancado: la respuesta llega por eventos.
func (g *Gestor) Enviar(ctx context.Context, sesionID, texto string) error {
	if strings.TrimSpace(texto) == "" {
		return fmt.Errorf("session: mensaje vacío")
	}
	ses, err := g.sesion(sesionID)
	if err != nil {
		return err
	}
	if ses.Estado == EstadoTerminada || ses.Estado == EstadoError {
		// Una sesión terminada o en error vuelve a trabajar pasando antes por
		// inactiva (ENUMS.md §3): se hace explícito, no se salta la regla.
		if err := g.cambiarEstado(ses, EstadoInactiva); err != nil {
			return err
		}
	}
	if err := g.Alcance.Almacen.EscribirMensaje(&store.Message{
		SessionID: sesionID,
		Role:      "user",
		Content:   texto,
	}); err != nil {
		return err
	}
	if err := g.cambiarEstado(ses, EstadoTrabajando); err != nil {
		return err
	}
	g.lanzar(ctx, sesionID, &trabajo{}, func(c context.Context) error {
		return g.ejecutarFlujo(c, sesionID, ObjetivoDeMensaje(texto), g.Flujo)
	})
	return nil
}

// ejecutarFlujo corre el flujo de una sesión y cierra su turno. Es lo que corre
// en la goroutine de segundo plano.
//
// El estado final no se inventa, se deduce del desenlace: cancelado → inactiva;
// fallo → error; el flujo pausado por un permiso → esperando permiso (que es el
// estado que `store` ya puso al registrar la aprobación); cualquier otro
// desenlace → terminada.
func (g *Gestor) ejecutarFlujo(ctx context.Context, sesionID, objetivo string, f flow.Flujo) error {
	if f.Nombre == "" {
		f = FlujoPorDefecto()
	}
	estadoFlujo, err := g.Motor.EjecutarFlujo(ctx, f, objetivo)
	return g.cerrarTurno(sesionID, estadoFlujo, err)
}

// cerrarTurno guarda el desenlace del turno y avisa del estado resultante.
func (g *Gestor) cerrarTurno(sesionID string, estadoFlujo flow.EstadoFlujo, err error) error {
	estado, motivo := EstadoTerminada, ""
	switch {
	case errors.Is(err, context.Canceled):
		estado = EstadoInactiva
	case err != nil:
		estado = EstadoError
		motivo = err.Error()
	case estadoFlujo == flow.EstadoPausadoPermiso:
		estado = EstadoEsperandoPermiso
		motivo = "el flujo espera una aprobación"
	}

	actual, oErr := g.Alcance.Almacen.Obtener(sesionID)
	if oErr != nil {
		return oErr
	}
	// El estado se escribe dentro de la misma transacción que el mensaje
	// (DATA_FLOW.md). Si la transición no es legal desde el estado actual —el
	// caso claro: se pidió aprobación y la sesión ya está esperando permiso—,
	// el turno se cierra igual y el estado se deja como está.
	destino := ""
	if store.ValidarTransicionSesion(actual.Status, estado) {
		destino = estado
	}
	if _, cErr := g.Alcance.Almacen.CerrarTurno(sesionID, resumenDe(estado, motivo), -1, -1, "", destino); cErr != nil {
		return cErr
	}
	// Se avisa del estado en el que queda la sesión: el nuevo si la transición
	// se aplicó, el que ya tenía si no (caso típico: esperaba permiso).
	nuevo := actual.Status
	if destino != "" {
		nuevo = destino
	}
	g.avisar(sesionID, nuevo, motivo)
	return nil
}

// resumenDe redacta lo que se guarda como mensaje del agente cuando el motor no
// aportó texto propio. No inventa resultados: dice el desenlace del turno.
func resumenDe(estado, motivo string) string {
	switch estado {
	case EstadoTerminada:
		return "El turno terminó."
	case EstadoInactiva:
		return "El turno se canceló."
	case EstadoEsperandoPermiso:
		return "El turno espera una aprobación."
	default:
		return "El turno falló: " + motivo
	}
}
