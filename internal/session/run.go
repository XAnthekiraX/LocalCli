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

// Motor es lo que `session` necesita de `flow`: responder como chat, arrancar
// un flujo explícito y consumir una cola. Lo implementa `flow.Motor`.
type Motor interface {
	Conversar(ctx context.Context, agente, objetivo string, historial []flow.Mensaje) (flow.Resultado, error)
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

// ArrancarFlujo guarda el objetivo y arranca un flujo explícito en segundo
// plano. Es lo que corre un comando (`/planificar`, `/crear`, `/actualizar`,
// `/eliminar`, `/resolver`); sin comando, `session` responde como chat
// (`Conversar`) y no arranca etapas (SPEC-MOTOR-FLUJOS §Reglas).
func (g *Gestor) ArrancarFlujo(ctx context.Context, sesionID string, f flow.Flujo, objetivo string) error {
	if strings.TrimSpace(objetivo) == "" {
		return fmt.Errorf("session: el flujo necesita un objetivo")
	}
	ses, err := g.sesion(sesionID)
	if err != nil {
		return err
	}
	if err := g.preparar(ses); err != nil {
		return err
	}
	if err := g.Alcance.Almacen.EscribirMensaje(&store.Message{
		SessionID: sesionID,
		Role:      "user",
		Content:   objetivo,
	}); err != nil {
		return err
	}
	if err := g.marcarTrabajando(ses); err != nil {
		return err
	}
	if f.Nombre == "" {
		f = g.flujoOFectivo()
	}
	g.lanzar(ctx, sesionID, &trabajo{}, func(c context.Context) error {
		return g.ejecutarFlujo(c, sesionID, objetivo, f)
	})
	return nil
}

// Enviar arranca el flujo configurado con el mensaje del usuario. Es el arranque
// explícito de un flujo; el camino por defecto de la vista es `Conversar`.
func (g *Gestor) Enviar(ctx context.Context, sesionID, texto string) error {
	return g.ArrancarFlujo(ctx, sesionID, g.flujoOFectivo(), ObjetivoDeMensaje(texto))
}

// Conversar responde el mensaje como chat con el agente activo: es el camino
// por defecto de la vista principal (SPEC-INTERFAZ §Reglas). El agente corre con
// su JSON y sus herramientas, y su respuesta queda en el historial (la escribe
// el ejecutor del agente); `session` solo cierra el turno y deja el estado.
//
// El historial se lee ANTES de escribir este mensaje: así el turno en curso no
// aparece duplicado (viaja como contexto, no como turno de la conversación) y
// `len(previos) == 0` marca la primera petición de la sesión. Con esa primera
// petición —y mientras el nombre siga siendo el provisional— se pide al modelo
// un título para la sesión, sin bloquear la respuesta ([[specs/SPEC-SESIONES]]).
func (g *Gestor) Conversar(ctx context.Context, sesionID, agente, texto string) error {
	if strings.TrimSpace(texto) == "" {
		return fmt.Errorf("session: mensaje vacío")
	}
	ses, err := g.sesion(sesionID)
	if err != nil {
		return err
	}
	if err := g.preparar(ses); err != nil {
		return err
	}
	previos, err := g.historialPara(ctx, sesionID)
	if err != nil {
		return err
	}
	// Se pide título mientras el nombre siga siendo el provisional: con la
	// primera petición se genera; si el modelo falló, la siguiente lo reintenta
	// (el nombre solo deja de ser provisional cuando un título se aplica). Una
	// sesión ya titulada no vuelve a pedirlo.
	titular := EsProvisional(ses.Nombre)
	if err := g.Alcance.Almacen.EscribirMensaje(&store.Message{
		SessionID: sesionID,
		Role:      "user",
		Content:   texto,
	}); err != nil {
		return err
	}
	// La detección de trabajo ordenado solo propone: se deja constancia y se
	// responde como chat (SPEC-COLA-TAREAS). Nunca arranca la cola.
	if aviso, ok := flow.SugerenciaTrabajo(texto); ok {
		_ = g.Alcance.Almacen.EscribirMensaje(&store.Message{
			SessionID: sesionID,
			Role:      "system",
			Content:   aviso,
		})
	}
	if err := g.marcarTrabajando(ses); err != nil {
		return err
	}
	g.lanzar(ctx, sesionID, &trabajo{}, func(c context.Context) error {
		_, cErr := g.Motor.Conversar(c, agente, ObjetivoDeMensaje(texto), previos)
		if cErr := g.cerrarChat(sesionID, cErr); cErr != nil {
			return cErr
		}
		if titular {
			g.generarTitulo(c, sesionID, texto)
		}
		return nil
	})
	return nil
}

// generarTitulo pide al modelo un título para la sesión a partir de su primera
// petición y lo persiste. Es best-effort: un fallo deja el nombre provisional y
// se reintenta con la siguiente petición (mientras siga siendo provisional), sin
// bloquear nunca la respuesta. El `id` no cambia: renombrar solo toca el nombre.
func (g *Gestor) generarTitulo(ctx context.Context, sesionID, texto string) {
	if g.Titulador == nil {
		return
	}
	titulo, err := g.Titulador.Titulo(ctx, texto)
	if err != nil {
		return
	}
	titulo = LimpiarTitulo(titulo)
	if titulo == "" {
		return
	}
	if err := g.Alcance.Almacen.Renombrar(sesionID, titulo); err != nil {
		return
	}
	if g.Bus == nil {
		return
	}
	g.Bus.Emitir(Evento{Nombre: EventoTituloSesion, Datos: map[string]string{
		"sesion": sesionID,
		"nombre": titulo,
	}})
}

// preparar deja la sesión lista para un turno nuevo. Una sesión terminada o en
// error vuelve pasando por inactiva (ENUMS.md §3), no se salta la regla. Una que
// quedó en `trabajando` sin trabajo vivo —el proceso se cerró a media
// generación— también: su estado es obsoleto y `trabajando → trabajando` es una
// transición ilegal, así que una petición nueva fallaría sin este paso.
func (g *Gestor) preparar(ses *Sesion) error {
	switch {
	case ses.Estado == EstadoTerminada || ses.Estado == EstadoError:
		return g.cambiarEstado(ses, EstadoInactiva)
	case ses.Estado == EstadoTrabajando && !g.EnCurso(ses.ID):
		return g.cambiarEstado(ses, EstadoInactiva)
	}
	return nil
}

// marcarTrabajando pasa la sesión a trabajando. Si ya lo está —turno anterior
// vivo que este va a reemplazar— no repite la transición, que sería ilegal.
func (g *Gestor) marcarTrabajando(ses *Sesion) error {
	if ses.Estado == EstadoTrabajando {
		return nil
	}
	return g.cambiarEstado(ses, EstadoTrabajando)
}

// Anotar deja constancia de un comando en el historial sin arrancar nada. Lo
// usa `/ejecutar`, que consume la cola que ya existe en lugar de correr etapas.
func (g *Gestor) Anotar(sesionID, texto string) error {
	if strings.TrimSpace(texto) == "" {
		return fmt.Errorf("session: mensaje vacío")
	}
	return g.Alcance.Almacen.EscribirMensaje(&store.Message{
		SessionID: sesionID,
		Role:      "user",
		Content:   texto,
	})
}

// cerrarChat cierra un turno de chat. Con desenlace correcto, el ejecutor del
// agente ya guardó su respuesta, así que aquí solo se deja el estado: escribir
// otro mensaje duplicaría el turno. Con error o cancelación se cierra con el
// resumen para que quede rastro (ERRORS.md: un error no borra trabajo hecho).
func (g *Gestor) cerrarChat(sesionID string, err error) error {
	if err != nil {
		return g.cerrarTurno(sesionID, flow.EstadoTerminado, err)
	}
	actual, oErr := g.Alcance.Almacen.Obtener(sesionID)
	if oErr != nil {
		return oErr
	}
	if store.ValidarTransicionSesion(actual.Status, EstadoTerminada) {
		if cErr := g.Alcance.Almacen.CambiarEstado(sesionID, EstadoTerminada); cErr != nil {
			return cErr
		}
		// Un turno de chat termina igual que un flujo, pero su aviso dice
		// «terminó de responder»: aquí no corrió ninguna etapa (NotificacionDeChat).
		g.avisarComo(sesionID, EstadoTerminada, NotificacionDeChat)
		return nil
	}
	g.avisar(sesionID, actual.Status, "")
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
