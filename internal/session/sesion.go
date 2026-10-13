package session

import (
	"fmt"

	"localcli/internal/store"
	"localcli/internal/tui"
)

// Estados válidos de una sesión ([[database/01-schema/ENUMS]] §2).
const (
	EstadoInactiva         = tui.EstadoInactiva
	EstadoTrabajando       = tui.EstadoTrabajando
	EstadoEsperandoPermiso = tui.EstadoEsperandoPermiso
	EstadoTerminada        = tui.EstadoTerminada
	EstadoError            = tui.EstadoError
)

// Sesion es la identidad, el nombre y el estado de una sesión.
type Sesion struct {
	ID     string
	Nombre string
	Estado string
}

// DeStore adapta una fila de store a la vista del módulo.
func DeStore(s *store.Sesion) *Sesion {
	if s == nil {
		return nil
	}
	return &Sesion{ID: s.ID, Nombre: s.Name, Estado: s.Status}
}

// AVista traduce la sesión al dato que consume la TUI.
func (s *Sesion) AVista() tui.Sesion {
	return tui.Sesion{ID: s.ID, Nombre: s.Nombre, Estado: s.Estado}
}

// EsEstadoValido dice si el texto es un estado del catálogo cerrado.
func EsEstadoValido(e string) bool {
	switch e {
	case EstadoInactiva, EstadoTrabajando, EstadoEsperandoPermiso, EstadoTerminada, EstadoError:
		return true
	}
	return false
}

// EsEstadoFinal dice si la sesión ya no vuelve a trabajando sin pasar por inactiva.
func EsEstadoFinal(e string) bool {
	return e == EstadoTerminada || e == EstadoError
}

// TransicionValida comprueba una transición contra ENUMS §3.
func TransicionValida(desde, hacia string) bool {
	if !EsEstadoValido(desde) || !EsEstadoValido(hacia) || desde == hacia {
		return false
	}
	if hacia == EstadoError {
		return true
	}
	switch desde {
	case EstadoInactiva:
		return hacia == EstadoTrabajando
	case EstadoTrabajando:
		return hacia == EstadoInactiva || hacia == EstadoEsperandoPermiso || hacia == EstadoTerminada
	case EstadoEsperandoPermiso:
		return hacia == EstadoTrabajando
	case EstadoTerminada, EstadoError:
		return hacia == EstadoInactiva
	}
	return false
}

func errorTransicion(desde, hacia string) error {
	return fmt.Errorf("session: transición no válida %q → %q", desde, hacia)
}
