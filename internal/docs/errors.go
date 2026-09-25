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

// NuevoErrParse construye el error localizado.
func NuevoErrParse(ruta string, causa error) error {
	return ErrDocParse{Ruta: ruta, Causa: causa}
}

// ErrDocNotFound: el destino pedido no es un documento cargado. Es el código
// documentado en ERRORES.md (E_DOC_NOT_FOUND); se produce al consultar el
// grafo por una ruta o wiki-link desconocido.
var ErrDocNotFound = errors.New("E_DOC_NOT_FOUND")
