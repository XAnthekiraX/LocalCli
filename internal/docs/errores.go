package docs

// errores.go — T-B020-07: errores de parseo localizados, sin abortar la carga.
//
// ERRORES.md fija el catálogo con código E_ que conoce el resto del motor; para
// documentos existen E_DOC_NOT_FOUND (pedir un documento inexistente) y
// E_STAGE_FAILED (la etapa de carga falla). El parseo roto de UN archivo no
// puede tumbar la carga entera del grafo: se reporta por archivo —ErrDocParse
// lleva la ruta— y el documento problemático simplemente no entra al grafo.

import (
	"errors"
	"fmt"
)

// ErrDocParse es el error localizado de parseo de un documento: contiene la
// ruta relativa y el motivo (frontmatter inválido, lectura fallida). El
// mensaje incluye el código E_DOC_PARSE para trazas y logs.
type ErrDocParse struct {
	Ruta  string // ruta relativa a la raíz de docs
	Causa error  // ErrFrenteInvalido u otro fallo subyacente
}

func (e ErrDocParse) Error() string {
	return fmt.Sprintf("%s: E_DOC_PARSE: %v", e.Ruta, e.Causa)
}

func (e ErrDocParse) Unwrap() error { return e.Causa }

// errEtapa es un fallo que detiene la etapa completa (no localizado a un
// archivo). Lleva E_STAGE_FAILED para el motor y el motivo concreto.
type errEtapa struct{ msg string }

func (e errEtapa) Error() string { return "E_STAGE_FAILED: " + e.msg }

// Is permite errors.Is(err, ErrStageFailed) sin inspeccionar el mensaje.
func (e errEtapa) Is(target error) bool { return target == ErrStageFailed }

// NuevoErrStage construye un fallo de etapa con su motivo.
func NuevoErrStage(msg string) error { return errEtapa{msg: msg} }

// NuevoErrParse construye el error localizado.
func NuevoErrParse(ruta string, causa error) error {
	return ErrDocParse{Ruta: ruta, Causa: causa}
}

// ErrDocNotFound: el destino pedido no es un documento cargado.
var ErrDocNotFound = errors.New("E_DOC_NOT_FOUND")

// ErrStageFailed es la vista a nivel de etapa de un fallo de carga. Los dos se
// cumplen a la vez: E_DOC_PARSE explica qué pasó, E_STAGE_FAILED avisa de que
// hay que parar.
var ErrStageFailed = errors.New("E_STAGE_FAILED")

// Is hace que errors.Is(err, ErrStageFailed) sea true para todo fallo
// localizado de carga.
func (e ErrDocParse) Is(target error) bool { return target == ErrStageFailed }
