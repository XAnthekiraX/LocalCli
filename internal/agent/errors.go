// errors.go — errores tipados del módulo agent.
//
// Fuente de verdad: ai/docs/backend/05-quality/ERRORS.md §1 («Un error tiene un
// código interno y un mensaje para la persona») y §3 (E_NO_RESPONSE).
//
// La pasada de redacción de un turno agotado puede quedarse sin texto: el
// modelo vuelve a pedir herramientas aunque no se le ofrezcan, o no dice nada.
// Ese turno NO se cierra como si hubiera respondido —antes se guardaba con el
// marcador `(respuesta vacía)`—: se devuelve este error, que la sesión deja a la
// vista en vez de inventar una respuesta (SPEC-AGENTE-BASE §El ciclo de un turno).
package agent

import "errors"

// Códigos internos documentados en ERRORS.md §3. Se comparan con errors.Is
// sobre los centinelas de abajo, sin leer el mensaje.
const (
	// CodigoSinRespuesta: el turno agotó sus rondas, se le pidió una redacción
	// (y un reintento, si volvió a pedir herramientas) y el modelo no entregó
	// texto. No se guarda como respuesta.
	CodigoSinRespuesta = "E_NO_RESPONSE"
)

// Centinelas comparables con errors.Is.
var (
	ErrSinRespuesta = errors.New(CodigoSinRespuesta)
)

// ErrorSinRespuesta es el error tipado del módulo. El Código es para el motor y
// los tests; el Mensaje es para la persona y dice qué pasó.
type ErrorSinRespuesta struct {
	Codigo  string
	Mensaje string
}

func (e *ErrorSinRespuesta) Error() string { return e.Codigo + ": " + e.Mensaje }

// Unwrap expone el centinela para errors.Is sobre cualquier envoltorio.
func (e *ErrorSinRespuesta) Unwrap() error { return ErrSinRespuesta }

// nuevoErrorSinRespuesta construye el error del turno que se quedó mudo. El
// motivo nombra las dos cosas que el usuario necesita saber: que se agotaron las
// rondas y que el modelo no entregó nada.
func nuevoErrorSinRespuesta() *ErrorSinRespuesta {
	return &ErrorSinRespuesta{
		Codigo:  CodigoSinRespuesta,
		Mensaje: "se agotaron las rondas de herramientas y el modelo no entregó ninguna respuesta",
	}
}
