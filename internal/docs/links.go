package docs

// links.go — T-B003-03: extraer los wiki-links [[...]] del cuerpo.
//
// Los documentos del proyecto se refieren entre sí con la sintaxis
// [[ruta]] o [[ruta|texto]]; la ruta es relativa a ai/docs sin extensión
// .md (ej.: [[backend/DECISIONS]], [[database/02-rules/DATA_FLOW]]). Cada
// aparición es una arista de dependencia hacia ese documento.
//
// Se ignoran los enlaces dentro de bloques de código ``` para no contar
// ejemplos, y el destino puede resolverse contra el mapa de documentos con
// ResolverEnlace: un destino que no corresponde a ningún documento cargado
// queda como enlace roto (los tests los listan por separado).

import (
	"sort"
	"strings"
)

// ExtraerEnlaces devuelve los destinos [[...]] del cuerpo, en orden de
// aparición y sin duplicados.
func ExtraerEnlaces(cuerpo string) []string {
	var (
		out   []string
		visto = map[string]bool{}
		fuera bool // dentro de bloque de código
	)
	for _, linea := range strings.Split(cuerpo, "\n") {
		if strings.HasPrefix(strings.TrimSpace(linea), "```") {
			fuera = !fuera
			continue
		}
		if fuera {
			continue
		}
		for len(linea) > 0 {
			i := strings.Index(linea, "[[")
			if i < 0 {
				break
			}
			linea = linea[i+2:]
			j := strings.Index(linea, "]]")
			if j < 0 {
				break // enlace sin cerrar: se ignora
			}
			dest := linea[:j]
			linea = linea[j+2:]
			if k := strings.IndexAny(dest, "|#"); k >= 0 {
				dest = dest[:k] // [[ruta|texto]] o [[ruta#sección]]
			}
			dest = strings.TrimSpace(dest)
			if dest != "" && !visto[dest] {
				visto[dest] = true
				out = append(out, dest)
			}
		}
	}
	return out
}

// ResolverEnlace convierte un destino [[x]] en la ruta relativa de un
// documento cargado. Acepta x con o sin ".md". Devuelve ok=false si ningún
// documento coincide: eso es un enlace roto.
func ResolverEnlace(docs map[string]*Doc, destino string) (string, bool) {
	if destino == "" {
		return "", false
	}
	candidatos := []string{destino + ".md", destino}
	for _, c := range candidatos {
		if _, ok := docs[c]; ok {
			return c, true
		}
	}
	return "", false
}

// EnlacesRotos devuelve, ordenadas, las apariciones "origen -> destino" de
// wiki-links que no resuelven a ningún documento cargado.
func EnlacesRotos(docs map[string]*Doc) []string {
	var rotos []string
	for _, ruta := range Rutas(docs) {
		for _, dest := range docs[ruta].Links {
			if _, ok := ResolverEnlace(docs, dest); !ok {
				rotos = append(rotos, ruta+" -> "+dest)
			}
		}
	}
	sort.Strings(rotos)
	return rotos
}
