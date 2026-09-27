// sessionsmodal.go — T-F014: el modal de sesiones (`Ctrl+X l`).
//
// Fuente de verdad: SPEC-INTERFAZ §Modales ("Tres modales centrados comparten el
// mismo comportamiento: uno abierto a la vez, sus teclas capturan el teclado
// (↑/↓ navegan, Enter aplica y cierra), Esc cierra sin cambios y Ctrl+C sigue
// saliendo") y §Cambiar de sesión ("Ctrl+X l abre el modal de sesiones con las
// sesiones del proyecto: nombre y estado de cada una… al elegir una con Enter,
// el chat cambia a esa sesión. Las sesiones en segundo plano se ven desde
// cualquier otra en el modal, con su estado. El modal aparece y desaparece: no
// ocupa espacio permanente"), más DOMAIN §1 (`modals`: sessionsmodal, "al
// aplicar abre esa sesión") y §2 ("Cambiar de sesión no interrumpe ninguna
// ejecución: solo cambia lo que se pinta").
//
// Comparte con los otros dos modales la mecánica de `Modal` (modelsmodal.go) y
// no decide nada: entrega la sesión elegida al `app`, que la abre. La ejecución
// en segundo plano vive en `session`, no aquí.
package tui

import (
	"localcli/internal/session"
)

// SessionsModal es el modal de sesiones: las sesiones del proyecto con su nombre
// y su estado, incluida la activa, y la que el usuario elige.
type SessionsModal struct {
	Modal
	// Sesiones es la lista real, en el mismo orden que Modal.Lineas: la elegida
	// se traduce con el índice resaltado.
	Sesiones []session.Sesion
	// SesionID es la sesión activa, la que se marca en su fila.
	SesionID string
}

// AvisoSinSesiones es lo que se ve cuando el proyecto todavía no tiene sesiones
// o la lectura no pudo hacerse. Se muestra y se cierra con Esc: nada se bloquea
// (SPEC-INTERFAZ §Modales, con la misma Mecánica que el aviso «sin modelos»).
const AvisoSinSesiones = "sin sesiones"

// AbrirSesiones muestra el modal vacío y pidiendo la lista. Cada apertura es una
// lectura nueva a `store` (T-F014-04): lo que se ve es lo que hay ahora, no una
// lista guardada de la última vez.
func (sm *SessionsModal) AbrirSesiones(activa string) {
	sm.Sesiones = nil
	sm.SesionID = activa
	// El pie recuerda la tecla de borrado: es la única acción propia de este
	// modal además de navegar y aplicar (SPEC-KEYBINDS §Acción
	// `session_delete`, ctrl+d en el ámbito del modal).
	sm.Modal.Pie = "(↑/↓ mover · enter abrir · ctrl+d eliminar · esc cerrar)"
	sm.Modal.Abrir("SESIONES", nil, AvisoCargando)
}

// FijarSesiones rellena el modal con la lista que llegó. Sin sesiones —o con una
// lectura fallida— el aviso «sin sesiones» sustituye a la lista: se ve y se
// cierra, igual que en el modal de modelos. Si el modal ya no está abierto, la
// lista se descarta: llegó tarde.
func (sm *SessionsModal) FijarSesiones(sesiones []session.Sesion) {
	if !sm.Abierto {
		return
	}
	sm.Sesiones = sesiones
	sm.repintar(sm.IndiceDe(sm.SesionID))
}

// ActualizarEstado refresca el estado de una sesión en la lista abierta. El modal
// es momentáneo, pero mientras está abierto pinta estados vivos: una sesión de
// segundo plano cambia delante del usuario (T-F007-05, SPEC-INTERFAZ §Cambiar de
// sesión: "con su estado"). El resaltado no se mueve: el usuario puede estar
// recorriendo la lista. Si la sesión no está en ella, no toca nada.
func (sm *SessionsModal) ActualizarEstado(sesionID, estado string) {
	for i := range sm.Sesiones {
		if sm.Sesiones[i].ID == sesionID {
			sm.Sesiones[i].Estado = estado
			sm.repintar(sm.Indice)
			return
		}
	}
}

// ActualizarNombre refresca el título de una sesión en la lista abierta. El
// título lo genera el modelo con la primera petición (`titulo_sesion`) y el id
// de la sesión no cambia. Si la sesión no está en la lista, no toca nada.
func (sm *SessionsModal) ActualizarNombre(sesionID, nombre string) {
	for i := range sm.Sesiones {
		if sm.Sesiones[i].ID == sesionID {
			sm.Sesiones[i].Nombre = nombre
			sm.repintar(sm.Indice)
			return
		}
	}
}

// IndiceDe devuelve la posición de una sesión en la lista, o 0 si no está: al
// abrir, el resaltado arranca en la sesión activa, que es la que se está
// viendo (SPEC-INTERFAZ §Cambiar de sesión).
func (sm *SessionsModal) IndiceDe(sesionID string) int {
	for i, ses := range sm.Sesiones {
		if ses.ID == sesionID {
			return i
		}
	}
	return 0
}

// repintar reconstruye las filas y deja el resaltado donde se le dice. Con la
// lista vacía pinta el aviso y no hay nada que resaltar.
func (sm *SessionsModal) repintar(indice int) {
	if !sm.Abierto {
		return
	}
	if len(sm.Sesiones) == 0 {
		sm.Modal.Abrir(sm.Titulo, nil, AvisoSinSesiones)
		return
	}
	lineas := make([]string, 0, len(sm.Sesiones))
	for _, ses := range sm.Sesiones {
		lineas = append(lineas, lineaDeSesion(ses, ses.ID == sm.SesionID))
	}
	sm.Modal.Abrir(sm.Titulo, lineas, "")
	sm.Indice = indice
}

// SesionElegida devuelve la sesión resaltada, lista para abrirla. Sin lista —
// «cargando…» o «sin sesiones»— no hay nada que elegir.
func (sm *SessionsModal) SesionElegida() (session.Sesion, bool) {
	i, ok := sm.Elegida()
	if !ok {
		return session.Sesion{}, false
	}
	return sm.Sesiones[i], true
}

// lineaDeSesion compone la fila: nombre, capa si la tiene y el estado tal cual
// los enums de store (DOMAIN §2: "los estados que se pintan son los de ENUMS;
// la TUI no inventa estados"). La activa se marca con «←» para que se sepa de
// cuál se está viendo.
func lineaDeSesion(ses session.Sesion, activa bool) string {
	nombre := ses.Nombre
	if nombre == "" {
		nombre = ses.ID
	}
	if ses.Capa != "" {
		nombre += " · " + ses.Capa
	}
	linea := nombre + "  [" + ses.Estado + "]"
	if activa {
		linea += " ←"
	}
	return linea
}
