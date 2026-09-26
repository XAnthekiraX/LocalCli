package docs

// graph.go — T-B003-04: grafo dirigido en memoria, reconstruido al arrancar.
//
// DECISIONS.md fija la decisión: "Grafo de documentación en memoria,
// reconstruido al arrancar" — no hay tabla que lo guarde (DOMAIN.md). Los
// nodos son los documentos cargados.
//
// De dónde salen las aristas. La fuente PRIMARIA es el frontmatter: DOMAIN.md
// ("las dependencias de la documentación viven en el frontmatter de los
// archivos, no en filas"), DATA_FLOW.md ("cada archivo declara en su
// frontmatter de qué depende") y BACKEND.md ("el grafo se reconstruye leyendo
// el frontmatter") lo dicen sin excepción. Antes de arreglar esto, las aristas
// salían solo de los wiki-links del CUERPO, que es otra cosa: ahí un enlace es
// una mención en prosa, no una dependencia declarada, y el grafo acababa lleno de aristas que solo dependían de por dónde
// caía la mención. Con el
// frontmatter vacío en los 49 documentos reales el resultado era un grafo sin
// ninguna arista.
//
// Los wiki-links del cuerpo se siguen leyendo como fuente COMPLEMENTARIA: un
// enlace del cuerpo a un documento sí es una dependencia real, y perderlas
// vaciaría el grafo de los archivos que aún no declaren frontmatter. La unión
// de ambas fuentes es lo que se indexa; se pueden consultar por separado con
// VecinosFrontmatter y VecinosCuerpo para saber de dónde salió cada arista.

import "sort"

// Grafo es el grafo dirigido de dependencias entre documentos.
type Grafo struct {
	docs   map[string]*Doc
	sal    map[string][]string // ruta -> destinos directos (out)
	entran map[string][]string // ruta -> orígenes directos (in)
	rotos  []string            // "origen -> destino" sin resolver, ordenado
}

// frenteDependencias son TODAS las claves con las que un documento declara una
// relación documental: las de dependencia fuerte y la de relación. DOMAIN.md
// las llama "etiquetas y enlaces que declaran dependencias": el valor es una
// lista de destinos [[...]], no una etiqueta.
var frenteDependencias = []string{
	"depende_de", "dependencias", "requiere", "requiere_de",
	"usa", "extiende", "relacionado",
}

// Claves de dependencia FUERTE: lo que hay que leer para que el documento sea
// correcto. Las aristas del grafo las usan, pero también el resto.
const (
	ClaveDependeDe = "depende_de"
	ClaveRelaciona = "relacionado"
)

// frenteDependenciasFuertes son las claves que expresan dependencia real. La
// separación importa: `depende_de` es lo que hay que leer; `relacionado` es a
// quién toca el documento sin que este necesite de él. Aplanarlos en una sola
// lista haría que el grafo afirmara que BACKEND.md depende de FRONTEND.md, que
// es justo al revés de lo que dice la documentación.
var frenteDependenciasFuertes = []string{
	"depende_de", "dependencias", "requiere", "requiere_de", "usa", "extiende",
}

// dependenciasDeClaves lee de un frontmatter los destinos de un conjunto de
// claves, tanto si vienen como lista en bloque como inline, y los devuelve en
// orden de aparición y sin duplicados. Las claves de etiquetas
// (tags/keywords) no se consultan nunca: etiquetan al documento y no lo hacen
// depender de nada.
func dependenciasDeClaves(d *Doc, claves []string) []string {
	if d == nil || !d.HasFront {
		return nil
	}
	var (
		out   []string
		visto = map[string]bool{}
	)
	anotar := func(v string) {
		for _, dest := range ExtraerEnlaces(v) {
			if !visto[dest] {
				visto[dest] = true
				out = append(out, dest)
			}
		}
	}
	for _, k := range claves {
		if v := d.Front.Claves[k]; v != "" {
			anotar(v)
		}
		for _, it := range d.Front.Listas[k] {
			anotar(it)
		}
	}
	return out
}

// DependenciasFrontmatter devuelve TODOS los destinos declarados en el
// frontmatter, dependencia fuerte y relación por igual. Es la lectura plana,
// la que necesita quien solo quiere saber qué declara el documento.
func DependenciasFrontmatter(d *Doc) []string {
	return dependenciasDeClaves(d, frenteDependencias)
}

// DependenciasFuertes devuelve solo lo declarado en las claves de dependencia
// real, sin lo que el documento solo relaciona. Un agente que cambia
// BACKEND.md necesita estas dos; FRONTEND.md no es una de ellas.
func DependenciasFuertes(d *Doc) []string {
	return dependenciasDeClaves(d, frenteDependenciasFuertes)
}

// ConstruirGrafo arma el grafo a partir de los documentos ya cargados,
// tomando las aristas del frontmatter y de los wiki-links del cuerpo.
// Determinista: adyacencias ordenadas por ruta.
func ConstruirGrafo(docs map[string]*Doc) *Grafo {
	g := &Grafo{docs: docs, sal: map[string][]string{}, entran: map[string][]string{}}
	for _, ruta := range Rutas(docs) {
		doc := docs[ruta]
		visto := map[string]bool{}
		var vecinos []string
		agregar := func(destino string) {
			res, ok := ResolverEnlace(docs, destino)
			if !ok || res == ruta || visto[res] {
				return
			}
			visto[res] = true
			vecinos = append(vecinos, res)
		}
		for _, dest := range DependenciasFrontmatter(doc) {
			agregar(dest)
		}
		for _, dest := range doc.Links {
			agregar(dest)
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

// Vecinos devuelve las dependencias directas (aristas salientes) de un nodo:
// la unión de lo declarado en el frontmatter y de los enlaces del cuerpo.
func (g *Grafo) Vecinos(ruta string) []string { return g.sal[ruta] }

// VecinosFrontmatter devuelve solo lo declarado en el frontmatter, sin
// resolver, aplanando dependencia y relación. Permite distinguir una
// declaración de una mención en prosa.
func (g *Grafo) VecinosFrontmatter(ruta string) []string {
	return DependenciasFrontmatter(g.docs[ruta])
}

// VecinosDeclarados devuelve solo las dependencias FUERTES declaradas en el
// frontmatter: lo que hay que leer para que el documento sea correcto. Es la
// lista que consume el nodo de contexto cuando no puede permitirse el cierre
// completo.
func (g *Grafo) VecinosDeclarados(ruta string) []string {
	return DependenciasFuertes(g.docs[ruta])
}

// VecinosRelacionados devuelve lo que el documento declara como relación sin
// dependencia. Sirve para saber a qué otros documentos toca un cambio.
func (g *Grafo) VecinosRelacionados(ruta string) []string {
	if d := g.docs[ruta]; d != nil {
		return dependenciasDeClaves(d, []string{ClaveRelaciona})
	}
	return nil
}

// VecinosCuerpo devuelve solo los destinos de los wiki-links del cuerpo, sin
// resolver.
func (g *Grafo) VecinosCuerpo(ruta string) []string {
	if d := g.docs[ruta]; d != nil {
		return d.Links
	}
	return nil
}

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

// DependenciasRelacionadas devuelve lo que el documento declara como relación
// sin dependencia. Simétrica de VecinosRelacionados para quien ya tiene el Doc.
func DependenciasRelacionadas(d *Doc) []string {
	return dependenciasDeClaves(d, []string{ClaveRelaciona})
}
