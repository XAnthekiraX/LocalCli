// errors.go — T-B005-08: errores tipados del contrato con Ollama.
//
// Fuente de verdad: ai/docs/backend/05-quality/ERRORS.md [19-20, 34-35].
// Reglas que implementan aquí:
//   - "Ollama no está corriendo" → aviso de que no se puede generar; el
//     harness sigue vivo. Es decir: connection refused u otros fallos de
//     red NO matan la UI ni entran en pánico: se mapean al error tipado
//     E_OLLAMA_UNAVAILABLE y se propagan como valor (un error es un valor,
//     no una excepción).
//   - "El modelo elegido no cabe en la VRAM" → aviso de que irá a RAM, más
//     lento; se carga igual, sin fallar en silencio. Por eso E_MODEL_TOO_BIG
//     es un AVISO: los helpers de perfil lo devuelven junto con el resultado,
//     nunca abortan la carga.
package ollama

import (
	"errors"
	"net"
	"strings"
)

// Códigos internos documentados en ERRORS.md §3. Se comparan con
// errors.Is / ErrorsAs sobre ErrorOllama.Codigo.
const (
	CodigoOllamaNoDisponible = "E_OLLAMA_UNAVAILABLE"
	CodigoModeloNoCabe       = "E_MODEL_TOO_BIG"
)

// ErrorOllama es el error tipado del módulo. El Código es para el motor y
// los tests; el Mensaje es para la persona (dice qué pasó y, cuando aplica,
// qué hacer), según ERRORS.md §1.
type ErrorOllama struct {
	Codigo   string
	Mensaje  string
	Detalle  string // causa técnica cruda, para logs/tests
	sentinel error  // centinela para errors.Is
}

func (e *ErrorOllama) Error() string {
	if e.Detalle != "" {
		return e.Codigo + ": " + e.Mensaje + " (" + e.Detalle + ")"
	}
	return e.Codigo + ": " + e.Mensaje
}

// Unwrap expone el centinela para que errors.Is(err, ErrOllamaNoDisponible)
// funcione sobre cualquier envoltorio intermedio.
func (e *ErrorOllama) Unwrap() error { return e.sentinel }

// Centinelas comparables con errors.Is.
var (
	ErrOllamaNoDisponible = errors.New(CodigoOllamaNoDisponible)
	ErrModeloNoCabe       = errors.New(CodigoModeloNoCabe)
)

// mensajeLevantarOllama explica cómo actuar, tal pide SPEC-OLLAMA-PERFIL
// ("si Ollama no está disponible, avisa con una instrucción clara").
const mensajeLevantarOllama = "no se puede generar: Ollama no responde en la dirección configurada; levántalo con `ollama serve`"

// clasificarFalloRed mapea cualquier fallo de conexión contra el servidor
// local al error documentado. Nunca devuelve nil: si no es reconocible como
// problema de red, lo deja pasar envuelto pero con código de indisponibilidad
// solo cuando hay señales de red (refused/reset/timeouts/DNS).
//
// Devuelve (errTipada, true) si el fallo es de conexión; (nil, false) si el
// llamante debe tratarlo como otro tipo de error (p. ej. status HTTP 500).
func clasificarFalloRed(err error) (*ErrorOllama, bool) {
	if err == nil {
		return nil, false
	}
	var ne net.Error
	esRed := errors.As(err, &ne) ||
		errors.Is(err, ErrOllamaNoDisponible) ||
		strings.Contains(err.Error(), "connection refused") ||
		strings.Contains(err.Error(), "connect: network is unreachable")
	if !esRed {
		return nil, false
	}
	return &ErrorOllama{
		Codigo:   CodigoOllamaNoDisponible,
		Mensaje:  mensajeLevantarOllama,
		Detalle:  err.Error(),
		sentinel: ErrOllamaNoDisponible,
	}, true
}

// nuevoModeloNoCabe construye el aviso tipado por modelo que no cabe. Como
// es un aviso (no un fallo), el flujo de carga continúa; ver profile.go.
func nuevoModeloNoCabe(modelo string, requiereMB, vramMB int64) *ErrorOllama {
	return &ErrorOllama{
		Codigo: CodigoModeloNoCabe,
		Mensaje: "el modelo " + modelo + " no cabe en la VRAM (" +
			itob(requiereMB) + " MB estimados frente a " + itob(vramMB) +
			" MB); se cargará en RAM, más lento",
		sentinel: ErrModeloNoCabe,
	}
}

// itob evita arrastrar strconv/fmt para dos números en un mensaje.
func itob(n int64) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
