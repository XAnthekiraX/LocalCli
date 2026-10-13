package docs

// fuente.go — T-B020-08: el grafo como fuente de contexto.
//
// El nodo de contexto consume el grafo por la interfaz `context.Fuente`
// ([[backend/02-interfaces/INTERFACES-GENERAL]] §5: `docs` provee el grafo a
// `context`). El grafo ya tiene todo lo que esa interfaz pide, así que la
// implementa él mismo: el catálogo de documentos con sus dependencias
// declaradas y la lectura de uno por su ruta.

import (
	contexto "localcli/internal/context"
)

// Documentos devuelve el catálogo que consume el nodo de contexto: cada
// documento con las dependencias FUERTES que declara (lo que hay que leer para
// entenderlo), no las simples menciones del cuerpo.
func (g *Grafo) Documentos() []contexto.Documento {
	out := make([]contexto.Documento, 0, len(g.docs))
	for _, ruta := range Rutas(g.docs) {
		out = append(out, contexto.Documento{
			Ruta:         ruta,
			Dependencias: DependenciasFuertes(g.docs[ruta]),
		})
	}
	return out
}

// Leer devuelve el cuerpo de un documento por su ruta o su destino de
// wiki-link. Error E_DOC_NOT_FOUND si no existe.
func (g *Grafo) Leer(ruta string) (string, error) {
	doc, err := g.Documento(ruta)
	if err != nil {
		return "", err
	}
	return doc.Body, nil
}
