package docs

import (
	"sort"
	"strings"
	"testing"
)

const raizDocs = "../../ai/docs"

// cargarReal carga la documentación real del proyecto. Los tests de este
// archivo vigilan el estado de ai/docs/, no una copia: si un documento nuevo
// nace sin frontmatter, este test falla.
func cargarReal(t *testing.T) (map[string]*Doc, *Grafo) {
	t.Helper()
	docs, errs, err := CargarDocs(raizDocs)
	if err != nil {
		t.Fatalf("CargarDocs: %v", err)
	}
	if len(errs) > 0 {
		for _, e := range errs {
			t.Errorf("documento no carga: %v", e)
		}
		t.Fatalf("%d documentos no cargan", len(errs))
	}
	return docs, ConstruirGrafo(docs)
}

// Un documento sin frontmatter no declara de qué depende, y el nodo de
// contexto se queda sin saber qué leer. Todos los de ai/docs/ lo llevan.
func TestTodoDocumentoDeclaraSusDependencias(t *testing.T) {
	docs, _ := cargarReal(t)

	var sinFrente []string
	for _, ruta := range Rutas(docs) {
		d := docs[ruta]
		if !d.HasFront {
			sinFrente = append(sinFrente, ruta)
			continue
		}
		if d.Front.Claves["title"] == "" {
			t.Errorf("%s: frontmatter sin title", ruta)
		}
		if d.Front.Claves["tags"] == "" && len(d.Front.Listas["tags"]) == 0 {
			t.Errorf("%s: frontmatter sin tags", ruta)
		}
	}
	sort.Strings(sinFrente)
	if len(sinFrente) > 0 {
		t.Errorf("documentos sin frontmatter: %v", sinFrente)
	}
}

// Una dependencia que no existe no es una dependencia: es un error que
// aparecería como documento ausente en el momento de cargarlo.
func TestNingunaDependenciaDeclaradaEstaRota(t *testing.T) {
	_, g := cargarReal(t)
	rotos := g.Rotos()
	sort.Strings(rotos)
	if len(rotos) > 0 {
		t.Errorf("enlaces rotos: %v", rotos)
	}
}

// Una dependencia fuerte que se referencia a sí misma por un ciclo rompe la
// ordenación topológica que usa el nodo de contexto para cargar en orden.
func TestDependenciasFuertesNoFormanCiclos(t *testing.T) {
	docs, g := cargarReal(t)

	// El valor de una clave ausente en un map[string]int es 0, así que los
	// estados empiezan en 1: con iota, un nodo no visitado parecería en curso.
	const (
		enPila = iota + 1
		hecho
	)
	estado := map[string]int{}

	var visitar func(string, []string) bool
	visitar = func(ruta string, pila []string) bool {
		switch estado[ruta] {
		case hecho:
			return false
		case enPila:
			t.Errorf("ciclo: %v -> %s", pila, ruta)
			return true
		}
		estado[ruta] = enPila
		for _, v := range g.VecinosDeclarados(ruta) {
			res, ok := ResolverEnlace(docs, v)
			if !ok {
				continue
			}
			if visitar(res, append(pila, ruta)) {
				return true
			}
		}
		estado[ruta] = hecho
		return false
	}

	for _, ruta := range Rutas(docs) {
		if visitar(ruta, nil) {
			return
		}
	}
}

// Esta es la razón de separar `depende_de` de `relacionado`, y el test que la
// protege. `depende_de` significa "hay que leer esto para entender el
// documento"; el cuerpo entero de cada documento cita de veinte a treinta
// archivos, y si todo eso contara como dependencia, cargar un documento
// arrastraría medio proyecto. El fallo sería silencioso —el grafo seguiría
// siendo válido— y por eso se fija con un número en vez de con una intuición.
func TestElCierreDeDependenciasNoArrastraElProyecto(t *testing.T) {
	docs, g := cargarReal(t)

	cierre := func(vecinos func(string) []string) map[string]int {
		out := map[string]int{}
		for _, ruta := range Rutas(docs) {
			vistos := map[string]bool{ruta: true}
			var cerrar func(string)
			cerrar = func(x string) {
				if vistos[x] {
					return
				}
				vistos[x] = true
				for _, v := range vecinos(x) {
					if res, ok := ResolverEnlace(docs, v); ok {
						cerrar(res)
					}
				}
			}
			for _, v := range vecinos(ruta) {
				if res, ok := ResolverEnlace(docs, v); ok {
					cerrar(res)
				}
			}
			out[ruta] = len(vistos) - 1
		}
		return out
	}

	declarado := cierre(g.VecinosDeclarados)
	conMenciones := cierre(g.VecinosCuerpo)

	var (
		peorDecl int
		peorMenc int
		desde    string
	)
	for ruta, n := range declarado {
		if n > peorDecl {
			peorDecl, desde = n, ruta
		}
		if conMenciones[ruta] > peorMenc {
			peorMenc = conMenciones[ruta]
		}
	}

	t.Logf("peor cierre con depende_de: %d documentos (desde %s)", peorDecl, desde)
	t.Logf("peor cierre si el cuerpo contara: %d documentos", peorMenc)

	// El peor cierre real medido es 19 documentos
	// (frontend/02-interfaces/INTERFACES, que suma los de frontend/FRONTEND,
	// backend/04-infrastructure/EVENTS y specs/SPEC-INTERFAZ, los tres
	// navegación legítima de sus capas), frente a los 51 que daría el cuerpo.
	// El límite deja margen por encima de lo medido. Antes era 12 porque el
	// test resolvía mal los enlaces —buscaba el destino sin ".md" en un mapa
	// con claves ".md"— y no seguía ninguna arista: medía el abanico directo
	// (8 documentos), no el cierre. Lo que se sigue exigiendo es lo que
	// distingue `depende_de` de una mención: ser mucho más estrecho que el
	// cuerpo, y por eso se mantiene la comparación de abajo.
	const maximoRazonable = 25
	if peorDecl > maximoRazonable {
		t.Errorf("el documento %s arrastra %d documentos en depende_de; el límite es %d",
			desde, peorDecl, maximoRazonable)
	}
	if peorDecl >= peorMenc {
		t.Errorf("depende_de (%d) no es más estrecho que el cuerpo (%d): la distinción no está haciendo nada",
			peorDecl, peorMenc)
	}
}

// enlacesDeSeccion devuelve los [[destino]] que aparecen bajo el título dado,
// hasta el siguiente título del mismo o superior nivel.
func enlacesDeSeccion(body, titulo string) []string {
	var (
		dentro bool
		nivel  = 2
		out    []string
	)
	for _, linea := range strings.Split(body, "\n") {
		if !dentro {
			if linea == "## "+titulo {
				dentro = true
			}
			continue
		}
		if strings.HasPrefix(linea, "#") {
			if len(linea)-len(strings.TrimLeft(linea, "#")) <= nivel {
				dentro = false
			}
			continue
		}
		out = append(out, ExtraerEnlaces(linea)...)
	}
	return out
}

// Las specs traen una sección "## Dependencias funcionales" que declara, en el
// propio cuerpo, de qué dependen. Es la fuente autoritativa: si el frontmatter
// pone esas dependencias en `relacionado`, el grafo afirma que la spec no
// necesita lo que ella misma dice necesitar.
func TestElFrontmatterCoincideConLasDependenciasFuncionales(t *testing.T) {
	docs, _ := cargarReal(t)

	var revisadas int
	for _, ruta := range Rutas(docs) {
		d := docs[ruta]
		funcionales := enlacesDeSeccion(d.Body, "Dependencias funcionales")
		if len(funcionales) == 0 {
			continue
		}
		revisadas++

		declaradas := map[string]bool{}
		for _, v := range DependenciasFuertes(d) {
			declaradas[v] = true
		}
		relacionadas := map[string]bool{}
		for _, v := range DependenciasRelacionadas(d) {
			relacionadas[v] = true
		}

		for _, v := range funcionales {
			if relacionadas[v] {
				t.Errorf("%s: el cuerpo declara %q como dependencia funcional, "+
					"pero el frontmatter la pone en `relacionado`", ruta, v)
				continue
			}
			if !declaradas[v] {
				t.Errorf("%s: el cuerpo declara %q como dependencia funcional, "+
					"pero no está en `depende_de`", ruta, v)
			}
		}
	}
	if revisadas != 20 {
		t.Errorf("revisadas %d secciones 'Dependencias funcionales', se esperaban 20", revisadas)
	}
}
