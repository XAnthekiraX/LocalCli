package docs

// consulta.go — T-B020-06: API de consulta del grafo.
//
// Lo que necesitan las etapas del flujo (context/agent): documentos por
// etiqueta, dependencias directas y transitivas, y la carga puntual de un
// documento por su ruta o destino de wiki-link — con el código E_DOC_NOT_FOUND
// cuando no existe (ERRORS.md: "El modelo pidió un documento que no existe").

import (
	"fmt"
	"sort"
	"strings"
)

// PorEtiqueta devuelve las rutas de los documentos cuya lista de etiquetas
// del frontmatter contiene tag (comparación sin distinción de mayúsculas),
// ordenadas.
func (g *Grafo) PorEtiqueta(tag string) []string {
	tag = strings.ToLower(strings.TrimSpace(tag))
	var out []string
	for _, ruta := range Rutas(g.docs) {
		for _, t := range g.docs[ruta].Front.Tags {
			if strings.ToLower(t) == tag {
				out = append(out, ruta)
				break
			}
		}
	}
	sort.Strings(out)
	return out
}

// Directas devuelve las dependencias directas (aristas salientes) de un
// documento ya resuelto. Error E_DOC_NOT_FOUND si la ruta no existe.
func (g *Grafo) Directas(rutaOrDestino string) ([]string, error) {
	res, ok := g.resolver(rutaOrDestino)
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrDocNotFound, rutaOrDestino)
	}
	return g.sal[res], nil
}

// Cierre devuelve el cierre transitivo de dependencias desde un documento.
func (g *Grafo) Cierre(rutaOrDestino string) ([]string, error) {
	res, ok := g.resolver(rutaOrDestino)
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrDocNotFound, rutaOrDestino)
	}
	return g.Transitivas(res), nil
}

// Documento devuelve el documento cargado por ruta ("backend/DECISIONS.md")
// o por destino de wiki-link ("backend/DECISIONS").
func (g *Grafo) Documento(rutaOrDestino string) (*Doc, error) {
	res, ok := g.resolver(rutaOrDestino)
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrDocNotFound, rutaOrDestino)
	}
	return g.docs[res], nil
}

func (g *Grafo) resolver(x string) (string, bool) {
	if _, ok := g.docs[x]; ok {
		return x, true
	}
	return ResolverEnlace(g.docs, x)
}
