// errors.go — errores tipados del módulo flow.
//
// Fuente de verdad: ai/docs/backend/05-quality/ERRORS.md §3 y §5 (los códigos
// del motor de etapas y de la cola): E_STAGE_FAILED, E_FLOW_CANCELLED,
// E_ELEMENTO_BLOQUEADO.
package flow

import "errors"

// Códigos internos documentados en ERRORS.md §3.
const (
	CodigoEtapaFallida      = "E_STAGE_FAILED"
	CodigoFlujoCancelado    = "E_FLOW_CANCELLED"
	CodigoElementoBloqueado = "E_ELEMENTO_BLOQUEADO"
)

// Centinelas comparables con errors.Is.
var (
	ErrEtapaFallida      = errors.New(CodigoEtapaFallida)
	ErrFlujoCancelado    = errors.New(CodigoFlujoCancelado)
	ErrElementoBloqueado = errors.New(CodigoElementoBloqueado)
)

// ErrorFlow es el error tipado del módulo.
type ErrorFlow struct {
	Codigo   string
	Mensaje  string
	sentinel error
}

func (e *ErrorFlow) Error() string { return e.Codigo + ": " + e.Mensaje }

// Unwrap expone el centinela para errors.Is.
func (e *ErrorFlow) Unwrap() error { return e.sentinel }

func nuevoError(codigo, mensaje string) *ErrorFlow {
	var sentinel error
	switch codigo {
	case CodigoEtapaFallida:
		sentinel = ErrEtapaFallida
	case CodigoFlujoCancelado:
		sentinel = ErrFlujoCancelado
	case CodigoElementoBloqueado:
		sentinel = ErrElementoBloqueado
	}
	return &ErrorFlow{Codigo: codigo, Mensaje: mensaje, sentinel: sentinel}
}
