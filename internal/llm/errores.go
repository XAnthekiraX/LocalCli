// errores.go — T-B036-03: errores tipados del contrato con el modelo.
//
// Fuente de verdad: ai/docs/backend/05-quality/ERRORS.md. Los códigos `E_` son
// los mismos para cualquier proveedor: que no responda es
// `E_PROVEEDOR_NO_DISPONIBLE`, y que el modelo no quepa es `E_MODEL_TOO_BIG`,
// que es un AVISO (los helpers de perfil lo devuelven junto al resultado, nunca
// abortan la carga).
package llm

import (
	"errors"
	"net"
	"strconv"
	"strings"
)

// Códigos internos documentados en ERRORS.md. Se comparan con errors.Is /
// errors.As sobre ErrorProveedor.Codigo.
const (
	CodigoProveedorNoDisponible = "E_PROVEEDOR_NO_DISPONIBLE"
	CodigoModeloNoCabe          = "E_MODEL_TOO_BIG"
)

// Centinelas comparables con errors.Is.
var (
	ErrProveedorNoDisponible = errors.New(CodigoProveedorNoDisponible)
	ErrModeloNoCabe          = errors.New(CodigoModeloNoCabe)
)

// ErrorProveedor es el error tipado de la frontera. El Código es para el motor
// y los tests; el Mensaje es para la persona (dice qué pasó y, cuando aplica,
// qué hacer), según ERRORS.md §1.
type ErrorProveedor struct {
	Codigo   string
	Mensaje  string
	Detalle  string // causa técnica cruda, para logs/tests
	sentinel error  // centinela para errors.Is
}

func (e *ErrorProveedor) Error() string {
	if e.Detalle != "" {
		return e.Codigo + ": " + e.Mensaje + " (" + e.Detalle + ")"
	}
	return e.Codigo + ": " + e.Mensaje
}

// Unwrap expone el centinela para que errors.Is(err, ErrProveedorNoDisponible)
// funcione sobre cualquier envoltorio intermedio.
func (e *ErrorProveedor) Unwrap() error { return e.sentinel }

// MensajeLevantar explica cómo actuar: el proveedor no responde y hay que
// levantarlo. Cada adaptador nombra su runtime y su instrucción de arranque.
func MensajeLevantar(nombre, baseURL, instruccion string) string {
	mensaje := "no se puede generar: " + nombre + " no responde en " + baseURL
	if strings.TrimSpace(instruccion) != "" {
		mensaje += "; levántalo con `" + instruccion + "`"
	}
	return mensaje
}

// NuevoErrorNoDisponible construye el error tipado de indisponibilidad sin
// clasificar: lo usan los adaptadores al construir sus fallos de red.
func NuevoErrorNoDisponible(mensaje, detalle string) *ErrorProveedor {
	return &ErrorProveedor{
		Codigo:   CodigoProveedorNoDisponible,
		Mensaje:  mensaje,
		Detalle:  detalle,
		sentinel: ErrProveedorNoDisponible,
	}
}

// ClasificarFalloRed mapea cualquier fallo de conexión contra el servidor local
// al error documentado. Devuelve (errTipada, true) si el fallo es de conexión;
// (nil, false) si el llamante debe tratarlo de otra forma (p. ej. status HTTP).
func ClasificarFalloRed(err error, mensaje string) (*ErrorProveedor, bool) {
	if err == nil {
		return nil, false
	}
	var ne net.Error
	esRed := errors.As(err, &ne) ||
		errors.Is(err, ErrProveedorNoDisponible) ||
		strings.Contains(err.Error(), "connection refused") ||
		strings.Contains(err.Error(), "connect: network is unreachable")
	if !esRed {
		return nil, false
	}
	return NuevoErrorNoDisponible(mensaje, err.Error()), true
}

// NuevoModeloNoCabe construye el aviso tipado por modelo que no cabe. Como es
// un aviso (no un fallo), el flujo de carga continúa.
func NuevoModeloNoCabe(modelo string, requiereMB, vramMB int64) *ErrorProveedor {
	return &ErrorProveedor{
		Codigo: CodigoModeloNoCabe,
		Mensaje: "el modelo " + modelo + " no cabe en la VRAM (" +
			strconv.FormatInt(requiereMB, 10) + " MB estimados frente a " +
			strconv.FormatInt(vramMB, 10) + " MB); se cargará en RAM, más lento",
		sentinel: ErrModeloNoCabe,
	}
}
