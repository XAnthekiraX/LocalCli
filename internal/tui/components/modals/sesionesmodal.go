package modals

// EstadoTrabajando es el estado con el que una sesión no se borra sin
// confirmación. Es el literal que pinta la base ([[database/01-schema/ENUMS]]
// §Estado de sesión); el modal no inventa estados.
const EstadoTrabajando = "trabajando"

// Sesion es una sesión del proyecto tal como la pinta el modal: su
// identificador, su título y su estado.
type Sesion struct {
	ID     string
	Nombre string
	Estado string
}

// Sesiones es el modal de sesiones (`Ctrl+X l`): las sesiones del proyecto con su
// nombre y su estado, incluida la activa, y la elegida por el usuario.
type Sesiones struct {
	Modal
	// Lista es la lista real, en el mismo orden que `Lineas`.
	Lista []Sesion
	// Activa es la sesión que se está viendo: se marca en su fila y es donde
	// arranca el resaltado.
	Activa string
	// confirmando es el identificador de la sesión cuya eliminación espera
	// confirmación. Vacío = no hay nada pendiente de confirmar.
	confirmando string
}

// Abrir deja el modal pidiendo la lista. Cada apertura es una lectura nueva:
// lo que se ve es lo que hay ahora, no la lista de la última vez.
func (sm *Sesiones) Abrir(activa string) {
	sm.Lista = nil
	sm.Activa = activa
	sm.confirmando = ""
	sm.Pie = PieSesiones
	sm.Modal.Abrir("SESIONES", nil, AvisoCargando)
}

// Fijar rellena el modal con la lista que llegó. Sin sesiones —o con una lectura
// fallida— queda el aviso «sin sesiones», que se ve y se cierra con `esc` como
// el resto. Si el modal ya no está abierto, la lista se descarta: llegó tarde.
func (sm *Sesiones) Fijar(lista []Sesion, err error) {
	if !sm.Abierto {
		return
	}
	if err != nil || len(lista) == 0 {
		sm.Lista = nil
		sm.Modal.Abrir("SESIONES", nil, AvisoSinSesiones)
		return
	}
	sm.Lista = lista
	sm.repintar(sm.IndiceDe(sm.Activa))
}

// SesionElegida devuelve la sesión resaltada, lista para que el `app` la abra.
// Sin lista —«cargando…» o «sin sesiones»— no hay nada que elegir. No se llama
// `Elegida` porque ese nombre ya es el de la mecánica común, que devuelve el
// índice resaltado.
func (sm *Sesiones) SesionElegida() (Sesion, bool) {
	indice, ok := sm.Modal.Elegida()
	if !ok {
		return Sesion{}, false
	}
	return sm.Lista[indice], true
}

// PedirEliminar resuelve `Ctrl+D` sobre la sesión resaltada: si está trabajando
// y aún no se había pedido confirmación, la pide y no entrega nada; si ya venía
// confirmada —o la sesión no trabaja— entrega la sesión para eliminarla. El
// modal no borra nada: expone la eliminación a quien lo abrió (SPEC-INTERFAZ
// §Cambiar de sesión).
func (sm *Sesiones) PedirEliminar() (Sesion, bool) {
	sesion, ok := sm.SesionElegida()
	if !ok {
		return Sesion{}, false
	}
	if sesion.Estado == EstadoTrabajando && sm.confirmando != sesion.ID {
		sm.confirmando = sesion.ID
		sm.repintar(sm.Indice)
		return Sesion{}, false
	}
	// La pregunta ya está contestada: se retira, de modo que un borrado que falle
	// no deje el siguiente `ctrl+d` armado sin volver a preguntar.
	sm.confirmando = ""
	sm.repintar(sm.Indice)
	return sesion, true
}

// Confirmando dice si hay una eliminación esperando confirmación: mientras la
// haya, `esc` la cancela en vez de cerrar el modal.
func (sm *Sesiones) Confirmando() bool { return sm.confirmando != "" }

// CancelarConfirmacion descarta la eliminación pendiente y deja la lista como
// estaba: la confirmación no cambia nada, solo pregunta.
func (sm *Sesiones) CancelarConfirmacion() {
	if sm.confirmando == "" {
		return
	}
	sm.confirmando = ""
	sm.repintar(sm.Indice)
}

// ActualizarEstado refresca el estado de una sesión de la lista abierta. El
// modal es momentáneo, pero mientras está abierto pinta estados vivos: una
// sesión de segundo plano cambia delante del usuario. El resaltado no se mueve
// —el usuario puede estar recorriéndola— y una sesión que no esté en la lista no
// toca nada.
func (sm *Sesiones) ActualizarEstado(sesionID, estado string) {
	for i := range sm.Lista {
		if sm.Lista[i].ID == sesionID {
			sm.Lista[i].Estado = estado
			sm.repintar(sm.Indice)
			return
		}
	}
}

// ActualizarNombre refresca el título de una sesión de la lista abierta: el
// título lo genera el modelo con la primera petición y el identificador no
// cambia.
func (sm *Sesiones) ActualizarNombre(sesionID, nombre string) {
	for i := range sm.Lista {
		if sm.Lista[i].ID == sesionID {
			sm.Lista[i].Nombre = nombre
			sm.repintar(sm.Indice)
			return
		}
	}
}

// IndiceDe es la posición de una sesión en la lista, o 0 si no está: al abrir,
// el resaltado arranca en la sesión activa, que es la que se está viendo.
func (sm *Sesiones) IndiceDe(sesionID string) int {
	for i, ses := range sm.Lista {
		if ses.ID == sesionID {
			return i
		}
	}
	return 0
}

// repintar reconstruye las filas y deja el resaltado donde se le dice. Con la
// lista vacía pinta el aviso y no hay nada que resaltar. La pregunta de
// confirmación, si la hay, va antes del pie.
func (sm *Sesiones) repintar(indice int) {
	if !sm.Abierto {
		return
	}
	if len(sm.Lista) == 0 {
		sm.Modal.Abrir("SESIONES", nil, AvisoSinSesiones)
		return
	}
	lineas := make([]string, 0, len(sm.Lista)+1)
	for _, ses := range sm.Lista {
		lineas = append(lineas, lineaDeSesion(ses, ses.ID == sm.Activa))
	}
	if sm.confirmando != "" {
		lineas = append(lineas, "", "¿seguro que deseas eliminar? (ctrl+d confirma · esc cancela)")
	}
	sm.Modal.Abrir("SESIONES", lineas, "")
	sm.Indice = indice
}

// lineaDeSesion compone la fila: nombre y estado tal cual llegan —los estados
// que se pintan son los de la base—, y la marca «←» en la sesión que se está
// viendo.
func lineaDeSesion(ses Sesion, activa bool) string {
	nombre := ses.Nombre
	if nombre == "" {
		nombre = ses.ID
	}
	linea := nombre + "  [" + ses.Estado + "]"
	if activa {
		linea += " ←"
	}
	return linea
}
