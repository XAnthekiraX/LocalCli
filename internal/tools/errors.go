// errors.go — errores tipados del módulo tools.
//
// Fuente de verdad: ai/docs/backend/05-quality/ERRORS.md §1 ("Un error tiene
// un código interno y un mensaje para la persona") y §3 (los códigos que
// puede producir este módulo: E_TOOL_UNKNOWN, E_TOOL_NOT_ALLOWED, E_BAD_ARGS).
//
// `tools` es la frontera donde el modelo pide algo: aquí es donde se rechaza
// una herramienta inventada, una que el agente no tiene, o unos argumentos que
// no encajan con el contrato. Ninguno de esos rechazos es una avería: es el
// caso normal de una petición que no procede (ERRORS.md §4).
package tools

import "errors"

// Códigos internos documentados en ERRORS.md §3. Se comparan con errors.Is
// sobre los centinelas de abajo, sin leer el mensaje.
const (
	// CodigoHerramientaDesconocida: el modelo pidió una herramienta fuera del
	// catálogo cerrado (TOOLS.md §1).
	CodigoHerramientaDesconocida = "E_TOOL_UNKNOWN"

	// CodigoHerramientaNoPermitida: el agente pidió una herramienta que no
	// tiene (`plan` pidiendo escritura), o una de internet con la salida
	// desactivada (SECURITY.md §2, CONFIGURATION.md §2).
	CodigoHerramientaNoPermitida = "E_TOOL_NOT_ALLOWED"

	// CodigoArgumentosInvalidos: los argumentos no encajan con el contrato de
	// la herramienta (VALIDATION.md §1).
	CodigoArgumentosInvalidos = "E_BAD_ARGS"
)

// Centinelas comparables con errors.Is.
var (
	ErrHerramientaDesconocida = errors.New(CodigoHerramientaDesconocida)
	ErrHerramientaNoPermitida = errors.New(CodigoHerramientaNoPermitida)
	ErrArgumentosInvalidos    = errors.New(CodigoArgumentosInvalidos)
)

// ErrorHerramienta es el error tipado del módulo. El Código es para el motor y
// los tests; el Mensaje es para la persona y dice qué pasó y qué hacer.
type ErrorHerramienta struct {
	Codigo  string
	Mensaje string
	// sentinel permite que errors.Is distinga el código sin leer el texto.
	sentinel error
}

func (e *ErrorHerramienta) Error() string { return e.Codigo + ": " + e.Mensaje }

// Unwrap expone el centinela para errors.Is sobre cualquier envoltorio.
func (e *ErrorHerramienta) Unwrap() error { return e.sentinel }

// nuevoError construye un error tipado asociado a su centinela.
func nuevoError(codigo, mensaje string) *ErrorHerramienta {
	var sentinel error
	switch codigo {
	case CodigoHerramientaDesconocida:
		sentinel = ErrHerramientaDesconocida
	case CodigoHerramientaNoPermitida:
		sentinel = ErrHerramientaNoPermitida
	case CodigoArgumentosInvalidos:
		sentinel = ErrArgumentosInvalidos
	}
	return &ErrorHerramienta{Codigo: codigo, Mensaje: mensaje, sentinel: sentinel}
}
