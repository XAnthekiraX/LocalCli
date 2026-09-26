// errors.go — errores tipados del módulo fileops.
//
// Fuente de verdad: ai/docs/backend/05-quality/ERRORS.md §3 y §5 (los códigos
// que produce una operación de archivo): E_PATH_OUTSIDE, E_PATH_EXISTS,
// E_NEEDS_APPROVAL, E_APPROVAL_DECLINED, E_NEEDS_CONFIRM, E_BAD_ARGS.
package fileops

import "errors"

// Códigos internos documentados en ERRORS.md §3.
const (
	CodigoRutaFuera            = "E_PATH_OUTSIDE"
	CodigoRutaExiste           = "E_PATH_EXISTS"
	CodigoNecesitaAprobacion   = "E_NEEDS_APPROVAL"
	CodigoAprobacionDeclinada  = "E_APPROVAL_DECLINED"
	CodigoNecesitaConfirmacion = "E_NEEDS_CONFIRM"
	CodigoArgumentosInvalidos  = "E_BAD_ARGS"
)

// Centinelas comparables con errors.Is.
var (
	ErrRutaFuera            = errors.New(CodigoRutaFuera)
	ErrRutaExiste           = errors.New(CodigoRutaExiste)
	ErrNecesitaAprobacion   = errors.New(CodigoNecesitaAprobacion)
	ErrAprobacionDeclinada  = errors.New(CodigoAprobacionDeclinada)
	ErrNecesitaConfirmacion = errors.New(CodigoNecesitaConfirmacion)
	ErrArgumentosInvalidos  = errors.New(CodigoArgumentosInvalidos)
)

// ErrorFileops es el error tipado del módulo. El Código es para el motor y los
// tests; el Mensaje es para la persona.
type ErrorFileops struct {
	Codigo   string
	Mensaje  string
	sentinel error
}

func (e *ErrorFileops) Error() string { return e.Codigo + ": " + e.Mensaje }

// Unwrap expone el centinela para errors.Is.
func (e *ErrorFileops) Unwrap() error { return e.sentinel }

func nuevoError(codigo, mensaje string) *ErrorFileops {
	var sentinel error
	switch codigo {
	case CodigoRutaFuera:
		sentinel = ErrRutaFuera
	case CodigoRutaExiste:
		sentinel = ErrRutaExiste
	case CodigoNecesitaAprobacion:
		sentinel = ErrNecesitaAprobacion
	case CodigoAprobacionDeclinada:
		sentinel = ErrAprobacionDeclinada
	case CodigoNecesitaConfirmacion:
		sentinel = ErrNecesitaConfirmacion
	case CodigoArgumentosInvalidos:
		sentinel = ErrArgumentosInvalidos
	}
	return &ErrorFileops{Codigo: codigo, Mensaje: mensaje, sentinel: sentinel}
}
