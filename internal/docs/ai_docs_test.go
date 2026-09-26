package docs

import (
	"sort"
	"strings"
	"testing"
)

const raizDocs = "../../ai/docs"

// carga la documentación real del proyecto. Los tests de este archivo vigilan
// el estado de ai/docs/, no una copia: si un documento nuevo nace sin
// frontmatter, este test falla.
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

	// El valor de una clave ausente en un map[string]int es 0, asi que los
	// estados empiezan en 1: con iota, un nodo no visitado pareceria en curso.
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
			if _, ok := docs[v]; !ok {
				continue
			}
			if visitar(v, append(pila, ruta)) {
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

	// Se compara contra el mismo grafo tomando todas las menciones del cuerpo
	// como dependencia, que es lo que ocurriria si `depende_de` no separara
	// "hay que leer esto" de "esto se menciona aqui".
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
					if _, ok := docs[v]; ok {
						cerrar(v)
					}
				}
			}
			for _, v := range vecinos(ruta) {
				cerrar(v)
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

	// El peor caso real es de 8 documentos (SPEC-CICLO-TRABAJO) y contando
	// solo el cuerpo llegaria a 20. El limite deja margen para que una
	// dependencia legitima no rompa el test, y sigue muy por debajo de lo que
	// daria el cuerpo entero, que es el fallo que hay que cazar.
	const maximoRazonable = 12
	if peorDecl > maximoRazonable {
		t.Errorf("el documento %s arrastra %d documentos en depende_de; el limite es %d",
			desde, peorDecl, maximoRazonable)
	}
	// Si la distincion no hace nada, ambas cifras coinciden y el test pasa por
	// el motivo equivocado.
	if peorDecl >= peorMenc {
		t.Errorf("depende_de (%d) no es mas estrecho que el cuerpo (%d): la distincion no esta haciendo nada",
			peorDecl, peorMenc)
	}
}

// enlacesDeSeccion devuelve los [[destino]] que aparecen bajo el titulo dado,
// hasta el siguiente titulo del mismo o superio nivel.
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
			// Solo cierra la seccion un titulo del mismo nivel o superior;
			// uno de menor nivel es un subapartado y se sigue leyendo.
			if len(linea)-len(strings.TrimLeft(linea, "#")) <= nivel {
				dentro = false
			}
			continue
		}
		out = append(out, ExtraerEnlaces(linea)...)
	}
	return out
}

// Las specs traen una seccion "## Dependencias funcionales" que declara, en el
// propio cuerpo, de que dependen. Es la fuente autoritativa: si el frontmatter
// pone esas dependencias en `relacionado`, el grafo afirma que la spec no
// necesita lo que ella misma dice necesitar, y el fallo es invisible porque
// los dos lados son enlaces validos.
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
					"pero no esta en `depende_de`", ruta, v)
			}
		}
	}
	if revisadas != 17 {
		t.Errorf("revisadas %d secciones 'Dependencias funcionales', se esperaban 17", revisadas)
	}
}
