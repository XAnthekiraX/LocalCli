// notify.go — T-B013-07: el estado de la sesión y sus notificaciones.
//
// Fuente de verdad: BUSINESS_RULES.md §Notificaciones ("Cuando una sesión pasa a
// esperando permiso, se manda una notificación… Cuando pasa a terminada, se
// manda una notificación… se manda aunque no estés viendo esa sesión") y
// EVENTS.md §1 y §3 (`estado_sesion` con sesión y estado nuevo; `notificacion`
// con sesión, motivo y una línea de qué hacer a continuación, sin el contenido).
//
// Dos detalles que no son adorno:
//
//   - El estado se emite SIEMPRE que cambia, y la notificación solo en los
//     estados que el usuario necesita saber (esperando permiso, terminada, con
//     error). Emitir los dos separa "pintar la sesión" de "avisar".
//   - La notificación es idempotente por estado: el estado se puede leer dos
//     veces (la aprobación la escribe `fileops` en su propia transacción, y
//     `session` la observa después), y nadie quiere dos avisos del mismo hecho.
package session

import "fmt"

// Notificacion dice, para un estado, si merece aviso y con qué texto: el motivo
// y qué hacer a continuación (EVENTS.md §3).
func Notificacion(estado string) (motivo, siguiente string, ok bool) {
	switch estado {
	case EstadoEsperandoPermiso:
		return "la sesión espera una aprobación", "revisa el panel de aprobaciones y decide", true
	case EstadoTerminada:
		return "la sesión terminó el trabajo", "revisa el resultado en su historial", true
	case EstadoError:
		return "la sesión quedó con error", "revisa el error y decide si reintentar", true
	}
	return "", "", false
}

// cambiarEstado aplica una transición ya validada y avisa del estado nuevo.
func (g *Gestor) cambiarEstado(ses *Sesion, nuevo string) error {
	if err := ses.PuedePasarA(nuevo); err != nil {
		return err
	}
	if err := g.Alcance.Almacen.CambiarEstado(ses.ID, nuevo); err != nil {
		return err
	}
	g.avisar(ses.ID, nuevo, "")
	return nil
}

// avisar emite `estado_sesion` y, si toca, `notificacion`. No bloquea y no
// espera respuesta: quien lo reciba decide qué hacer (EVENTS.md §4).
func (g *Gestor) avisar(sesionID, estado, detalle string) {
	if g.Bus == nil {
		return
	}
	g.Bus.Emitir(Evento{
		Nombre: EventoEstadoSesion,
		Datos:  map[string]string{"sesion": sesionID, "estado": estado},
	})
	motivo, siguiente, ok := Notificacion(estado)
	if !ok {
		return
	}
	if detalle != "" {
		motivo += ": " + detalle
	}
	if !g.marcarNotificado(sesionID, estado) {
		return
	}
	g.Bus.Emitir(Evento{
		Nombre: EventoNotificacion,
		Datos: map[string]string{
			"sesion":    sesionID,
			"motivo":    motivo,
			"siguiente": siguiente,
		},
	})
}

// marcarNotificado devuelve true solo la primera vez que se avisa de ese estado
// para esa sesión.
func (g *Gestor) marcarNotificado(sesionID, estado string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.notificado[sesionID] == estado {
		return false
	}
	g.notificado[sesionID] = estado
	return true
}

// AvisarEsperaDePermiso mira las aprobaciones pendientes del proyecto y avisa de
// las sesiones que esperan decisión.
//
// Hace falta porque quien pone la sesión en `esperando_permiso` es `fileops`, en
// la transacción que crea la aprobación (DATA_FLOW.md), sin pasar por este
// módulo. `session` no inventa un temporizador: expone la comprobación y quien
// tenga el reloj (el bucle de la TUI) la llama. Como el aviso es idempotente por
// estado, llamarla de más no duplica nada.
func (g *Gestor) AvisarEsperaDePermiso() (int, error) {
	pendientes, err := g.Alcance.Almacen.PendientesDeAprobacion()
	if err != nil {
		return 0, err
	}
	vistos := map[string]bool{}
	avisadas := 0
	for _, p := range pendientes {
		if vistos[p.SessionID] {
			continue
		}
		vistos[p.SessionID] = true
		g.avisar(p.SessionID, EstadoEsperandoPermiso, p.Description)
		avisadas++
	}
	return avisadas, nil
}

// AvisarEstado relee el estado de la sesión y emite lo que corresponda. Sirve
// para el arranque y para el caso en que el estado cambió fuera de este módulo.
func (g *Gestor) AvisarEstado(sesionID string) error {
	ses, err := g.sesion(sesionID)
	if err != nil {
		return err
	}
	if ses.Estado == "" {
		return fmt.Errorf("session: la sesión %s no tiene estado", sesionID)
	}
	g.avisar(sesionID, ses.Estado, "")
	return nil
}
