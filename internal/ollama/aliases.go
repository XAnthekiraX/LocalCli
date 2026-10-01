// aliases.go — T-B036-04: re-exportación de los tipos neutros de `llm`.
//
// Los tipos del contrato con el modelo viven en `llm` (T-B036-01); este
// adaptador los re-exporta para no duplicar su definición y mantener la
// superficie que habla. No son tipos propios: son los mismos de la frontera.
package ollama

import "localcli/internal/llm"

type (
	Mensaje        = llm.Mensaje
	Herramienta    = llm.Herramienta
	Definicion     = llm.Definicion
	ToolCall       = llm.ToolCall
	Evento         = llm.Evento
	TipoEvento     = llm.TipoEvento
	RespuestaFinal = llm.RespuestaFinal
	Modelo         = llm.Modelo

	ColaInferencia  = llm.ColaInferencia
	OpcionesEncolar = llm.OpcionesEncolar

	// ErrorOllama es el nombre histórico del error tipado de la frontera.
	ErrorOllama = llm.ErrorProveedor
)

const (
	EventoToken        = llm.EventoToken
	EventoRazonamiento = llm.EventoRazonamiento
	EventoDone         = llm.EventoDone
	EventoError        = llm.EventoError

	// Códigos con su nombre histórico. El valor es el neutro del harness.
	CodigoOllamaNoDisponible = llm.CodigoProveedorNoDisponible
	CodigoModeloNoCabe       = llm.CodigoModeloNoCabe

	TopeVentanaPorDefecto = llm.TopeVentanaPorDefecto
)

var (
	ErrOllamaNoDisponible = llm.ErrProveedorNoDisponible
	ErrModeloNoCabe       = llm.ErrModeloNoCabe
)

// NewColaInferencia crea la cola FIFO neutra.
func NewColaInferencia() *ColaInferencia { return llm.NewColaInferencia() }

// VentanaDeModelo delega en la regla neutra de la ventana.
func VentanaDeModelo(m Modelo, tope int) int { return llm.VentanaDeModelo(m, tope) }

// PuedeUsarHerramientas delega en la normalización neutra de capacidades.
func PuedeUsarHerramientas(c []string) bool { return llm.PuedeUsarHerramientas(c) }

// PuedeVer delega en la normalización neutra de capacidades.
func PuedeVer(c []string) bool { return llm.PuedeVer(c) }

// PuedePensar delega en la normalización neutra de capacidades.
func PuedePensar(c []string) bool { return llm.PuedePensar(c) }
