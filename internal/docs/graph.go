package docs

// graph.go — T-B003-04: grafo dirigido en memoria, reconstruido al arrancar.
//
// DECISIONS.md fija la decisión: "Grafo de documentación en memoria,
// reconstruido al arrancar" — no hay tabla que lo guarde (DOMAIN.md). Los
// nodos son los documentos cargados; las aristas salen de los wiki-links del
// cuerpo resueltos contra el mapa de rutas. Un enlace roto simplemente no
// genera arista: se reporta aparte (links.EnlacesRotos).

import "sort"

// Grafo es el grafo dirigido de dependencias entre documentos.
type Grafo struct {
	docs   map[string]*Doc
	sal    map[string][]string // ruta -> destinos directos (out)
	entran map[string][]string // ruta -> orígenes directos (in)
	rotos  []string            // "origen -> destino" sin resolver, ordenado
}

// ConstruirGrafo arma el grafo a partir de los documentos ya cargados con
// sus enlaces extraídos. Determinista: adyacencias ordenadas por ruta.
func ConstruirGrafo(docs map[string]*Doc) *Grafo {
	g := &Grafo{docs: docs, sal: map[string][]string{}, entran: map[string][]string{}}
	for _, ruta := range Rutas(docs) {
		var vecinos []string
		visto := map[string]bool{}
		for _, dest := range docs[ruta].Links {
			res, ok := ResolverEnlace(docs, dest)
			if !ok || res == ruta || visto[res] {
				continue
			}
			visto[res] = true
			vecinos = append(vecinos, res)
		}
		sort.Strings(vecinos)
		g.sal[ruta] = vecinos
		for _, v := range vecinos {
			g.entran[v] = append(g.entran[v], ruta)
		}
	}
	for v := range g.entran {
		sort.Strings(g.entran[v])
	}
	g.rotos = EnlacesRotos(docs)
	return g
}

// Docs devuelve el mapa de documentos sobre los que se construyó el grafo.
func (g *Grafo) Docs() map[string]*Doc { return g.docs }

// VecinosDevuelve las dependencias directas (aristas salientes) de un nodo.
func (g *Grafo) Vecinos(ruta string) []string { return g.sal[ruta] }

// Revés devuelve las dependencias directas entrantes (quién enlaza a este).
func (g *Grafo) Revés(ruta string) []string { return g.entran[ruta] }

// Rotos lista los enlaces que no resolvieron, como "origen -> destino".
func (g *Grafo) Rotos() []string { return g.rotos }

// Transitivas devuelve el cierre transitivo de las aristas salientes desde
// ruta (excluyendo el propio nodo), ordenado y sin ciclos infinitos.
func (g *Grafo) Transitivas(ruta string) []string {
	visitado := map[string]bool{ruta: true}
	pila := append([]string{}, g.sal[ruta]...)
	var out []string
	for len(pila) > 0 {
		n := pila[len(pila)-1]
		pila = pila[:len(pila)-1]
		if visitado[n] {
			continue
		}
		visitado[n] = true
		out = append(out, n)
		pila = append(pila, g.sal[n]...)
	}
	sort.Strings(out)
	return out
}
