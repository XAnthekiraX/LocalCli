package docs

// errors.go — T-B003-06: errores de parseo localizados, sin abortar la carga.
//
// ERRORES.md fija el catálogo con código E_ que conoce el resto del motor;
// para documentos existen E_DOC_NOT_FOUND (pedir un documento inexistente) y
// E_STAGE_FAILED (la etapa de carga falla). El parseo roto de UN archivo no
// puede tumbar la carga entera del grafo: se reporta por archivo —ErrDocParse
// lleva la ruta— y el documento problemático simplemente no entra al grafo.
// CargarDocs acumula esos errores en su segundo valor de retorno.

import (
	"errors"
	"fmt"
)

// ErrDocParse es el error localizado de parseo de un documento: contiene la
// ruta relativa y el motivo (frontmatter inválido, lectura fallida). El
// mensaje incluye el código E_DOC_PARSE para trazas y logs, siguiendo la
// convención de códigos de ERRORES.md §3.
type ErrDocParse struct {
	Ruta  string // ruta relativa a la raíz de docs
	Causa error  // ErrFrenteInvalido u otro fallo subyacente
}

func (e ErrDocParse) Error() string {
	return fmt.Sprintf("%s: E_DOC_PARSE: %v", e.Ruta, e.Causa)
}

func (e ErrDocParse) Unwrap() error { return e.Causa }

// errEtapa es un fallo que detiene la etapa completa (no localizado a un
// archivo). Lleva E_STAGE_FAILED para el motor y el motivo concreto en el
// mensaje, por el mismo motivo que ErrDocParse.
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

// ErrDocNotFound: el destino pedido no es un documento cargado. Es el código
// documentado en ERRORES.md (E_DOC_NOT_FOUND); se produce al consultar el
// grafo por una ruta o wiki-link desconocido.
var ErrDocNotFound = errors.New("E_DOC_NOT_FOUND")

// ErrStageFailed es la vista a nivel de etapa de un fallo de carga.
// ERRORES.md §3 lo fija así: "los errores de una etapa se convierten en
// E_STAGE_FAILED para el motor, que detiene el flujo y deja que el usuario
// decida". El mensaje de ErrDocParse conserva el código específico
// E_DOC_PARSE, que es el diagnóstico; este centinela es lo que el motor
// comprueba para saber que la etapa no puede continuar. Los dos se cumplen a
// la vez: E_DOC_PARSE explica qué pasó, E_STAGE_FAILED avisa de que hay que
// parar.
var ErrStageFailed = errors.New("E_STAGE_FAILED")

// Is hace que errors.Is(err, ErrStageFailed) sea true para todo fallo
// localizado de carga. Sin esto el motor tendría que inspeccionar la cadena
// del mensaje para reconocer E_STAGE_FAILED, que es justo lo que ERRORES.md
// prohíbe al resto del motor.
func (e ErrDocParse) Is(target error) bool { return target == ErrStageFailed }
