// model.go — T-B013-01: el tipo Sesión y los estados de su ciclo de vida.
//
// Fuente de verdad: SPEC-SESIONES §Reglas ("Cada sesión expone su estado:
// inactiva, trabajando, esperando permiso, terminada o con error") y
// BUSINESS_RULES.md §Sesiones. Los estados y sus transiciones legales ya están
// fijados en la capa de datos (database/01-schema/ENUMS.md §2 y §3) y `store` es
// quien los conoce: aquí no se redefinen ni se duplican, se reexportan para que
// el resto del módulo no escriba literales.
//
// La transición ilegal se rechaza dos veces: en el código (aquí, antes de
// escribir) y en `store` (dentro de la transacción que la aplica). No es
// redundancia: la primera da un error claro sin tocar la base, la segunda cierra
// la ventana de concurrencia entre leer el estado y escribirlo.
package session

import (
	"fmt"

	"localcli/internal/store"
)

// Estados del ciclo de vida, reexportados desde `store` (ENUMS.md §2).
const (
	EstadoInactiva         = store.StatusInactiva
	EstadoTrabajando       = store.StatusTrabajando
	EstadoEsperandoPermiso = store.StatusEsperandoPermiso
	EstadoTerminada        = store.StatusTerminada
	EstadoError            = store.StatusError
)

// Sesion es la vista de una sesión que ve el resto del motor: su identidad, a
// qué capa apunta y en qué estado está.
type Sesion struct {
	ID          string
	Nombre      string
	Capa        string // "" = sesión general
	Estado      string
	Creada      string
	Actualizada string
}

// DeStore adapta una fila de `store` a la vista del módulo.
func DeStore(s *store.Session) *Sesion {
	if s == nil {
		return nil
	}
	return &Sesion{
		ID:          s.ID,
		Nombre:      s.Name,
		Capa:        s.Layer,
		Estado:      s.Status,
		Creada:      s.CreatedAt,
		Actualizada: s.UpdatedAt,
	}
}

// EsEstadoValido dice si el texto pertenece al catálogo cerrado de estados.
func EsEstadoValido(e string) bool {
	switch e {
	case EstadoInactiva, EstadoTrabajando, EstadoEsperandoPermiso, EstadoTerminada, EstadoError:
		return true
	}
	return false
}

// PuedePasarA comprueba la transición contra ENUMS.md §3 antes de intentarla.
func (s *Sesion) PuedePasarA(nuevo string) error {
	if !EsEstadoValido(nuevo) {
		return fmt.Errorf("session: %q no es un estado de sesión (ENUMS.md §2)", nuevo)
	}
	if !store.ValidarTransicionSesion(s.Estado, nuevo) {
		return fmt.Errorf("session: la sesión %s no puede pasar de %q a %q", s.ID, s.Estado, nuevo)
	}
	return nil
}

// EnCurso dice si la sesión tiene trabajo vivo: está trabajando o esperando una
// decisión. Cerrar una sesión en este estado exige decir qué hacer con el flujo.
func (s *Sesion) EnCurso() bool {
	return s.Estado == EstadoTrabajando || s.Estado == EstadoEsperandoPermiso
}
