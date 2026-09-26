// sessions.go — T-B014-05: el selector momentáneo de sesiones.
//
// Fuente de verdad: SPEC-INTERFAZ §Cambiar de sesión ("No hay una lista de
// sesiones siempre visible… Un atajo abre un selector momentáneo con las
// sesiones del proyecto: nombre y estado de cada una. Al elegir una, el chat
// cambia a esa sesión. Las sesiones en segundo plano se ven desde cualquier otra
// en el selector, con su estado") y §Reglas ("Cambiar de sesión no detiene lo que
// está corriendo").
//
// El selector no detiene nada: solo cambia a quién se le pinta el chat. La
// ejecución en segundo plano vive en `session`, no aquí.
package tui

import (
	"strings"

	"localcli/internal/session"
)

// Selector es la lista momentánea de sesiones.
type Selector struct {
	Abierto  bool
	Sesiones []session.Sesion
	Indice   int
	SesionID string // la sesión activa, para marcarla
}

// Abrir carga las sesiones y lo muestra. Sin sesiones no se abre: un selector
// vacío no es una pantalla, es un callejón.
func (s *Selector) Abrir(sesiones []session.Sesion, activa string) {
	s.Sesiones = sesiones
	s.SesionID = activa
	s.Indice = 0
	for i, ses := range sesiones {
		if ses.ID == activa {
			s.Indice = i
			break
		}
	}
	s.Abierto = len(sesiones) > 0
}

// Cerrar lo esconde sin tocar nada más.
func (s *Selector) Cerrar() { s.Abierto = false }

// Mover cambia la selección, sin salirse de la lista.
func (s *Selector) Mover(delta int) {
	if len(s.Sesiones) == 0 {
		return
	}
	s.Indice = (s.Indice + delta + len(s.Sesiones)) % len(s.Sesiones)
}

// ActualizarEstado refresca el estado de una sesión en la lista abierta. El
// selector es momentáneo, pero mientras está abierto pinta estados vivos: una
// sesión de segundo plano cambia delante del usuario (T-F007-05). Si la sesión
// no está en la lista, no toca nada.
func (s *Selector) ActualizarEstado(sesionID, estado string) {
	for i := range s.Sesiones {
		if s.Sesiones[i].ID == sesionID {
			s.Sesiones[i].Estado = estado
			return
		}
	}
}

// Elegida devuelve la sesión seleccionada.
func (s *Selector) Elegida() (session.Sesion, bool) {
	if len(s.Sesiones) == 0 || s.Indice < 0 || s.Indice >= len(s.Sesiones) {
		return session.Sesion{}, false
	}
	return s.Sesiones[s.Indice], true
}

// Render pinta el selector: nombre y estado de cada sesión, con la activa
// marcada.
func (s *Selector) Render() string {
	if !s.Abierto {
		return ""
	}
	var b strings.Builder
	b.WriteString(estiloTitulo.Render("SESIONES") + "\n")
	for i, ses := range s.Sesiones {
		marca := "  "
		if i == s.Indice {
			marca = "› "
		}
		nombre := ses.Nombre
		if ses.Capa != "" {
			nombre += " · " + ses.Capa
		}
		linea := marca + nombre + "  [" + ses.Estado + "]"
		if ses.ID == s.SesionID {
			linea += " ←"
		}
		if i == s.Indice {
			linea = estiloUsuario.Render(linea)
		}
		b.WriteString(linea + "\n")
	}
	b.WriteString(estiloSistema.Render("(enter para elegir · esc para cerrar)"))
	return b.String()
}
