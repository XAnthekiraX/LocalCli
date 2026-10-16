package session

import (
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
