package session

import (
	"strconv"

	tea "github.com/charmbracelet/bubbletea"

	"localcli/internal/store"
	"localcli/internal/tui"
)

// DeshacerDesde revierte la conversación de una sesión desde el mensaje indicado
// hasta el final: es lo que hace el globo de un mensaje del usuario. Como puede
// sacar varios turnos de golpe, olvida la pila de rehacer de la sesión: rehacer a
// medias intercalaría turnos que ya no encajan detrás de lo que queda. Devuelve
// cuántos mensajes se sacaron; 0 significa que no había nada que revertir.
func (o *Orquestador) DeshacerDesde(sesionID, mensajeID string) (int, error) {
	_, n, err := store.DeshacerDesde(o.db, o.projectID, sesionID, mensajeID)
	if err != nil || n == 0 {
		return n, err
	}
	o.olvidarDeshechos(sesionID)
	return n, nil
}

// Bifurcar abre una sesión nueva con el historial de la sesión indicada hasta el
// mensaje dado, incluido: un punto de partida independiente que nace con el
// nombre `FORK:<original>`. Devuelve la sesión nueva para que la vista la adopte.
func (o *Orquestador) Bifurcar(sesionID, mensajeID string) (*tui.Sesion, error) {
	nueva, err := store.BifurcarSesion(o.db, o.projectID, sesionID, mensajeID)
	if err != nil {
		return nil, err
	}
	vista := DeStore(nueva).AVista()
	return &vista, nil
}

// SincronizarContexto calcula y emite el estado de tokens del contexto efectivo
// de una sesión (resumen + tramo reciente) para sincronizar el panel al retomarla.
func (o *Orquestador) SincronizarContexto(sesionID string) tea.Cmd {
	return func() tea.Msg {
		filas, err := store.ListarMensajes(o.db, o.projectID, sesionID)
		if err != nil {
			return nil
		}
		conv := conversacionDe(filas)
		if len(conv) == 0 {
			o.emitir(tui.Evento{Sesion: sesionID, Tipo: tui.EventoTokensTurno, Datos: map[string]string{
				"contexto":       "0",
				"limite":         strconv.Itoa(o.ventanaActual()),
				"limite_harness": "false",
			}})
			return nil
		}
		sum, hay := o.resumenPersistido(sesionID)
		cubierto := indiceCubierto(conv, sum.HastaID)
		if hay && cubierto == 0 {
			_ = store.LimpiarResumenHistorial(o.db, o.projectID, sesionID)
			hay = false
		}
		var total int
		if hay {
			total = estimarTokens(sum.Texto) + tokensDe(conv[cubierto:])
		} else {
			total = tokensDe(conv)
		}
		if total < 0 {
			total = 0
		}
		o.emitir(tui.Evento{Sesion: sesionID, Tipo: tui.EventoTokensTurno, Datos: map[string]string{
			"contexto":       strconv.Itoa(total),
			"limite":         strconv.Itoa(o.ventanaActual()),
			"limite_harness": "false",
		}})
		return nil
	}
}
