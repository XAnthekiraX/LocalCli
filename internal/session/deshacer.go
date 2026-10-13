package session

import "localcli/internal/store"

// Deshacer saca de la conversación el último turno de una sesión y lo guarda para
// poder rehacerlo. Devuelve cuántos mensajes se sacaron; 0 significa que no había
// nada que deshacer. No toca los archivos.
func (o *Orquestador) Deshacer(sesionID string) (int, error) {
	turno, n, err := store.DeshacerTurno(o.db, o.projectID, sesionID)
	if err != nil || n == 0 {
		return n, err
	}
	o.apilarDeshecho(sesionID, turno)
	return n, nil
}

// Rehacer vuelve a escribir el último turno deshecho de la sesión. Si la escritura
// falla, el turno se conserva para reintentar. Devuelve cuántos mensajes
// volvieron; 0 significa que no había nada que rehacer.
func (o *Orquestador) Rehacer(sesionID string) (int, error) {
	turno, ok := o.desapilarDeshecho(sesionID)
	if !ok {
		return 0, nil
	}
	if err := store.RehacerTurno(o.db, turno); err != nil {
		o.apilarDeshecho(sesionID, turno)
		return 0, err
	}
	return len(turno.Mensajes), nil
}

// apilarDeshecho guarda un turno deshecho al final de la pila de su sesión: el
// último deshecho es el primero en rehacerse.
func (o *Orquestador) apilarDeshecho(sesionID string, t store.Turno) {
	o.muDeshechos.Lock()
	defer o.muDeshechos.Unlock()
	if o.deshechos == nil {
		o.deshechos = map[string][]store.Turno{}
	}
	o.deshechos[sesionID] = append(o.deshechos[sesionID], t)
}

// desapilarDeshecho saca el último turno deshecho de una sesión.
func (o *Orquestador) desapilarDeshecho(sesionID string) (store.Turno, bool) {
	o.muDeshechos.Lock()
	defer o.muDeshechos.Unlock()
	pila := o.deshechos[sesionID]
	if len(pila) == 0 {
		return store.Turno{}, false
	}
	turno := pila[len(pila)-1]
	o.deshechos[sesionID] = pila[:len(pila)-1]
	return turno, true
}

// olvidarDeshechos tira la pila de rehacer de una sesión. Lo hace un turno nuevo:
// lo deshecho ya no encaja detrás de él y volver a escribirlo lo dejaría
// intercalado en medio del hilo.
func (o *Orquestador) olvidarDeshechos(sesionID string) {
	o.muDeshechos.Lock()
	defer o.muDeshechos.Unlock()
	delete(o.deshechos, sesionID)
}
