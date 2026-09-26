// errors.go — errores tipados del módulo exec.
//
// Fuente de verdad: ai/docs/backend/05-quality/ERRORS.md §3 y §5 (E_CMD_NOT_WHITELISTED,
// E_CMD_TIMEOUT, E_CMD_OUTPUT_TRUNCATED, E_NO_LANDLOCK, E_BAD_ARGS).
package exec

import "errors"

// Códigos internos documentados en ERRORS.md §3.
const (
	CodigoComandoNoEnBlanco   = "E_CMD_NOT_WHITELISTED"
	CodigoComandoAgotado      = "E_CMD_TIMEOUT"
	CodigoSalidaTruncada      = "E_CMD_OUTPUT_TRUNCATED"
	CodigoSinLandlock         = "E_NO_LANDLOCK"
	CodigoAprobacionDeclinada = "E_APPROVAL_DECLINED"
	CodigoArgumentosInvalidos = "E_BAD_ARGS"
)

// Centinelas comparables con errors.Is.
var (
	ErrComandoNoEnBlanco   = errors.New(CodigoComandoNoEnBlanco)
	ErrComandoAgotado      = errors.New(CodigoComandoAgotado)
	ErrSalidaTruncada      = errors.New(CodigoSalidaTruncada)
	ErrSinLandlock         = errors.New(CodigoSinLandlock)
	ErrAprobacionDeclinada = errors.New(CodigoAprobacionDeclinada)
	ErrArgumentosInvalidos = errors.New(CodigoArgumentosInvalidos)
)

// ErrorExec es el error tipado del módulo.
type ErrorExec struct {
	Codigo   string
	Mensaje  string
	sentinel error
}

func (e *ErrorExec) Error() string { return e.Codigo + ": " + e.Mensaje }

// Unwrap expone el centinela para errors.Is.
func (e *ErrorExec) Unwrap() error { return e.sentinel }

func nuevoError(codigo, mensaje string) *ErrorExec {
	var sentinel error
	switch codigo {
	case CodigoComandoNoEnBlanco:
		sentinel = ErrComandoNoEnBlanco
	case CodigoComandoAgotado:
		sentinel = ErrComandoAgotado
	case CodigoSalidaTruncada:
		sentinel = ErrSalidaTruncada
	case CodigoSinLandlock:
		sentinel = ErrSinLandlock
	case CodigoAprobacionDeclinada:
		sentinel = ErrAprobacionDeclinada
	case CodigoArgumentosInvalidos:
		sentinel = ErrArgumentosInvalidos
	}
	return &ErrorExec{Codigo: codigo, Mensaje: mensaje, sentinel: sentinel}
}
