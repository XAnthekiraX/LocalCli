// errors.go — errores tipados del módulo queue.
//
// Fuente de verdad: ai/docs/backend/05-quality/ERRORS.md §5 (operación "Cola"):
// el único código documentado para este módulo es `E_ELEMENTO_BLOQUEADO`, y con
// un matiz que manda en todo el diseño: "la cola no es un error, sigue con
// otra". Por eso un bloqueo NO se devuelve como fallo de `Siguiente`: se
// reporta en la vista de la cola, y el consumo continúa con el siguiente
// elemento que sí pueda arrancar.
//
// Lo que sí es un error duro es romper una invariante de arquitectura: lanzar
// la cola sin ser el motor (DECISIONS.md: "Solo el motor lanza colas; no hay
// modo de tarea suelta en la primera versión"). Eso no es un E_ documentado
// porque no es una situación del usuario, así que va como error normal de Go.
package queue

import (
	"errors"
	"fmt"
)

// CodigoElementoBloqueado es el código documentado para un elemento que no
// puede arrancar por dependencias (ERRORS.md §3 y §5).
const CodigoElementoBloqueado = "E_ELEMENTO_BLOQUEADO"

// ErrElementoBloqueado es el centinela comparable con errors.Is.
var ErrElementoBloqueado = errors.New(CodigoElementoBloqueado)

// ErrorCola es el error tipado del módulo.
type ErrorCola struct {
	Codigo  string
	Mensaje string
	// sentinel permite que errors.Is reconozca el código sin leer el texto.
	sentinel error
}

func (e *ErrorCola) Error() string { return e.Codigo + ": " + e.Mensaje }

// Unwrap expone el centinela para errors.Is.
func (e *ErrorCola) Unwrap() error { return e.sentinel }

// nuevoError construye un error tipado asociado a su centinela.
func nuevoError(codigo, mensaje string) *ErrorCola {
	var sentinel error
	if codigo == CodigoElementoBloqueado {
		sentinel = ErrElementoBloqueado
	}
	return &ErrorCola{Codigo: codigo, Mensaje: mensaje, sentinel: sentinel}
}

// errSoloMotor protege la invariante "solo el motor lanza colas".
func errSoloMotor(detalle string) error {
	return fmt.Errorf("queue: solo el motor consume la cola; %s "+
		"(no hay modo de tarea suelta, ver DECISIONS.md)", detalle)
}

// errBloqueoNoEscribible explica por qué un bloqueo no se marca en el archivo.
//
// El frontmatter del elemento admite `bloqueada_por`, pero las tablas reales de
// `MAIN-TASKS.md` y `NNN-task-*.md` no tienen esa columna (task/parse.go solo la
// lee "si existe"). Escribir `estado = bloqueada` sin `bloqueada_por` dejaría el
// TODO inválido —`ValidarReglaBloqueo` exige razón, y con razón— y la carga
// entera de la capa fallaría. Hasta que el formato del TODO tenga esa columna
// (decisión de `task`, no de `queue`), el bloqueo se informa en la vista y no se
// escribe.
func errBloqueoNoEscribible(id string) error {
	return fmt.Errorf("queue: %s no puede marcarse bloqueada en el archivo: "+
		"la tabla del TODO no declara la columna bloqueada_por y un bloqueo sin "+
		"motivo invalida el TODO; el bloqueo se informa en la vista", id)
}
