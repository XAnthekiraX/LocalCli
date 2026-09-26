package docs

import (
	"errors"
	"path/filepath"
	"reflect"
	"testing"
)

// T-B003-01: la carga encuentra todos los .md, incluidos los de subcarpetas.
func TestCargarDocsTodosLosMarkdown(t *testing.T) {
	docs, errs, err := CargarDocs("testdata")
	if err != nil {
		t.Fatal(err)
	}
	queridas := []string{"a.md", "b.md", "malo.md", "sin_front.md", "una/c.md"}
	for _, q := range queridas {
		if _, ok := docs[q]; !ok && q != "malo.md" {
			t.Errorf("falta %s en la carga; rutas: %v", q, Rutas(docs))
		}
	}
	// malo.md tiene frontmatter inválido: NO entra, pero hay un error localizado.
	if _, ok := docs["malo.md"]; ok {
		t.Error("malo.md debía descartarse por frontmatter inválido")
	}
	if len(errs) != 1 {
		t.Fatalf("errores localizados = %d (%v), queremos 1", len(errs), errs)
	}
	var ep ErrDocParse
	if !errors.As(errs[0], &ep) || ep.Ruta != "malo.md" {
		t.Errorf("error localizado inesperado: %#v", errs[0])
	}
	// 5 documentos válidos: a, b, frente, sin_front y una/c (malo.md se
	// descarta por su frontmatter roto y se reporta aparte).
	if got := len(docs); got != 5 {
		t.Errorf("documentos cargados = %d, queremos 5", got)
	}
}

// T-B003-02: con y sin frontmatter parsea correcto.
func TestFrontmatterConYSin(t *testing.T) {
	docs, _, err := CargarDocs("testdata")
	if err != nil {
		t.Fatal(err)
	}
	a := docs["a.md"]
	if !a.HasFront || a.Front.Claves["title"] != "Documento A" {
		t.Errorf("frontmatter de a.md: %+v", a.Front)
	}
	if want := []string{"alfa", "comun"}; !reflect.DeepEqual(a.Front.Tags, want) {
		t.Errorf("tags de a.md = %v, queremos %v", a.Front.Tags, want)
	}
	b := docs["b.md"]
	if want := []string{"beta", "comun"}; !reflect.DeepEqual(b.Front.Tags, want) {
		t.Errorf("etiquetas en bloque de b.md = %v, queremos %v", b.Front.Tags, want)
	}
	sf := docs["sin_front.md"]
	if sf.HasFront || len(sf.Front.Tags) != 0 {
		t.Errorf("sin_front no debe tener frontmatter: %+v", sf)
	}
	if sf.Body == "" {
		t.Error("el cuerpo de sin_front.md no puede estar vacío")
	}
}

// T-B003-03: enlaces válidos y rotos se listan por separado; los de bloque
// de código no cuentan.
func TestEnlacesValidosYRotos(t *testing.T) {
	docs, _, _ := CargarDocs("testdata")
	a := docs["a.md"]
	if want := []string{"b", "una/c", "no-existe"}; !reflect.DeepEqual(a.Links, want) {
		t.Errorf("enlaces de a.md = %v, queremos %v", a.Links, want)
	}
	g := ConstruirGrafo(docs)
	rotos := g.Rotos()
	q := []string{"a.md -> no-existe", "b.md -> fantasma"}
	if !reflect.DeepEqual(rotos, q) {
		t.Errorf("enlaces rotos = %v, queremos %v", rotos, q)
	}
}

// T-B003-04: vecinos de un nodo fixture coinciden con lo declarado.
func TestGrafoVecinos(t *testing.T) {
	docs, _, _ := CargarDocs("testdata")
	g := ConstruirGrafo(docs)
	if want := []string{"b.md", "una/c.md"}; !reflect.DeepEqual(g.Vecinos("a.md"), want) {
		t.Errorf("vecinos de a.md = %v, queremos %v", g.Vecinos("a.md"), want)
	}
	// b.md lo enlazan a.md por el cuerpo y frente.md por el frontmatter: las
	// dos fuentes cuentan, cada una por su lado.
	if want := []string{"a.md", "frente.md", "una/c.md"}; !reflect.DeepEqual(g.Revés("b.md"), want) {
		t.Errorf("entrantes de b.md = %v, queremos %v", g.Revés("b.md"), want)
	}
}

// El frontmatter es la fuente de aristas. DOMAIN.md, DATA_FLOW.md y
// BACKEND.md coinciden: "las dependencias viven en el frontmatter de los
// archivos, no en filas". Antes las aristas salían solo de los wiki-links del
// cuerpo, así que un documento cuyas dependencias declara arriba —y no
// menciona abajo— era una isla sin ninguna arista.
func TestGrafoTomaLasAristasDelFrontmatter(t *testing.T) {
	docs, _, _ := CargarDocs("testdata")
	g := ConstruirGrafo(docs)

	want := []string{"b.md", "una/c.md"}
	if got := g.Vecinos("frente.md"); !reflect.DeepEqual(got, want) {
		t.Errorf("vecinos de frente.md = %v, queremos %v (declarados en el frontmatter)", got, want)
	}
	// Y el camino inverso también existe: los documentos declarados saben quién
	// los necesita.
	if got := g.Revés("una/c.md"); !reflect.DeepEqual(got, []string{"a.md", "frente.md"}) {
		t.Errorf("revés de una/c.md = %v, queremos [a.md frente.md]", got)
	}
	// Se puede consultar cada fuente por separado, para saber si una arista es
	// dependencia explícita o una mención en prosa.
	if got := g.VecinosFrontmatter("frente.md"); !reflect.DeepEqual(got, []string{"b", "una/c"}) {
		t.Errorf("frontmatter de frente.md = %v, queremos [b una/c]", got)
	}
	if len(g.VecinosCuerpo("frente.md")) != 0 {
		t.Errorf("el cuerpo de frente.md no menciona nada: %v", g.VecinosCuerpo("frente.md"))
	}
	// El cierre transitivo tiene que pasar por las aristas de frontmatter.
	cierre, err := g.Cierre("frente.md")
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"b.md", "una/c.md"}; !reflect.DeepEqual(cierre, want) {
		t.Errorf("cierre de frente.md = %v, queremos %v", cierre, want)
	}
}

// Una etiqueta no es una dependencia. Los tags existen para clasificar; si
// contaran como aristas, "tags: [comun]" ataría documentos que solo comparten
// una palabra.
func TestLasEtiquetasNoSonAristas(t *testing.T) {
	docs, _, _ := CargarDocs("testdata")
	g := ConstruirGrafo(docs)
	if len(g.VecinosFrontmatter("b.md")) != 0 {
		t.Errorf("b.md solo tiene etiquetas, no dependencias: %v", g.VecinosFrontmatter("b.md"))
	}
	// Y sin embargo sigue consultable por etiqueta, que es para lo que sirve.
	if _, err := g.Cierre("b.md"); err != nil {
		t.Fatal(err)
	}
}

// Las dependencias del frontmatter admiten las dos formas de YAML: lista en
// bloque e inline. Solo con inline, un documento escrito con la lista customary
// se quedaba sin aristas.
func TestDependenciasFrontmatterEnInlineYEnBloque(t *testing.T) {
	casos := []struct {
		nombre string
		frente string
	}{
		{"inline", "depende_de: [\"[[b]]\"]"},
		{"bloque", "depende_de:\n  - \"[[b]]\""},
		{"requiere", "requiere: \"[[b]]\""},
		{"sin_dependencias", "tags: [b]"},
	}
	for _, c := range casos {
		raw := "---\ntitle: t\n" + c.frente + "\n---\n\ncuerpo\n"
		m, _, hay, err := ParseFrontmatter([]byte(raw))
		if err != nil {
			t.Errorf("%s: %v", c.nombre, err)
			continue
		}
		if !hay {
			t.Errorf("%s: no detectó el frontmatter", c.nombre)
			continue
		}
		d := &Doc{Path: "x.md", Front: m, HasFront: hay}
		got := DependenciasFrontmatter(d)
		quiere := 0
		if c.nombre != "sin_dependencias" {
			quiere = 1
		}
		if len(got) != quiere {
			t.Errorf("%s: DependenciasFrontmatter = %v, queremos %d entradas", c.nombre, got, quiere)
		}
	}
}

// Un documento sin frontmatter no declara dependencias, pero sus enlaces de
// cuerpo siguen contando: si no, el grafo de los 49 archivos reales —ninguno
// con frontmatter— quedaría vacío.
func TestGrafoSigueUsandoLosEnlacesDelCuerpo(t *testing.T) {
	docs, _, _ := CargarDocs("testdata")
	g := ConstruirGrafo(docs)
	if want := []string{"b.md", "una/c.md"}; !reflect.DeepEqual(g.Vecinos("a.md"), want) {
		t.Errorf("a.md sigue teniendo sus vecinos por el cuerpo: %v", want)
	}
	if len(g.VecinosFrontmatter("a.md")) != 0 {
		t.Errorf("a.md no declara frontmatter: %v", g.VecinosFrontmatter("a.md"))
	}
}

// El parser tenía que aguantar el YAML que se escribe a mano. Un "# nota" al
// final de la línea se guardaba DENTRO del valor, y las comillas de un valor
// se guardaban como parte del valor.
func TestFrontmatterQuitaComentariosYComillas(t *testing.T) {
	casos := []struct {
		frente string
		quiere string
	}{
		{"titulo: Mi titulo # nota al final", "Mi titulo"},
		{"titulo: \"Mi titulo\" # nota", "Mi titulo"},
		{"titulo: 'otro titulo'", "otro titulo"},
		{"titulo: Sin comillas", "Sin comillas"},
		// En YAML un # solo abre comentario si va precedido de espacio: dentro
		// de una palabra forma parte del valor.
		{"titulo: con#dentro no es comentario", "con#dentro no es comentario"},
		{"titulo: con #dentro", "con"},
	}
	for _, c := range casos {
		raw := "---\n" + c.frente + "\n---\n\ncuerpo\n"
		m, _, hay, err := ParseFrontmatter([]byte(raw))
		if err != nil || !hay {
			t.Errorf("%q: hay=%v err=%v", c.frente, hay, err)
			continue
		}
		if got := m.Claves["titulo"]; got != c.quiere {
			t.Errorf("%q → %q, queremos %q", c.frente, got, c.quiere)
		}
	}
}

// Una comilla que se abre y no se cierra es un frontmatter roto, no un valor.
// Aceptarla a medias metía el resto de la línea en el contenido del documento y
// el archivo pasaba por bueno.
func TestFrontmatterRechazaComillaSinCerrar(t *testing.T) {
	raw := "---\ntitulo: \"sin cerrar\nautor: alguien\n---\n\ncuerpo\n"
	_, _, hay, err := ParseFrontmatter([]byte(raw))
	if !hay {
		t.Fatal("debería detectar el bloque de frontmatter")
	}
	if err == nil {
		t.Fatal("una comilla sin cerrar debe ser error de frontmatter")
	}
	if !errors.Is(err, ErrFrenteInvalido) {
		t.Errorf("err = %v, queremos ErrFrenteInvalido", err)
	}
}

// La lista de dependencias con las dos formas de sintaxis, y con comentarios y
// comillas: es justo lo que se escribe al declarar de qué depende un documento.
func TestListaDeDependenciasToleraComentariosYComillas(t *testing.T) {
	raw := "---\n" +
		"depende_de:\n" +
		"  - \"[[backend/DECISIONS]]\" # la decisión de fondo\n" +
		"  - '[[database/DATABASE]]'\n" +
		"tags: [a, b] # no es una dependencia\n" +
		"---\n\ncuerpo\n"
	m, _, _, err := ParseFrontmatter([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	quiere := []string{"[[backend/DECISIONS]]", "[[database/DATABASE]]"}
	got := m.Listas["depende_de"]
	if len(got) != len(quiere) {
		t.Fatalf("depende_de = %v, queremos %v", got, quiere)
	}
	for i := range quiere {
		if got[i] != quiere[i] {
			t.Errorf("depende_de[%d] = %q, queremos %q", i, got[i], quiere[i])
		}
	}
}

// ERRORES.md §3: los errores de una etapa se convierten en E_STAGE_FAILED para
// el motor. El mensaje conserva E_DOC_PARSE (el diagnóstico) y el error debe
// reconocerse con errors.Is, no raspando la cadena.
func TestFalloDeCargaEsReconocibleComoEStageFailed(t *testing.T) {
	err := NuevoErrParse("malo.md", ErrFrenteInvalido)
	if !errors.Is(err, ErrStageFailed) {
		t.Error("un fallo de parseo de un documento debe ser un E_STAGE_FAILED para el motor")
	}
	if !contains(err.Error(), "E_DOC_PARSE") {
		t.Errorf("el mensaje debe conservar el diagnóstico E_DOC_PARSE: %v", err)
	}
	if !contains(err.Error(), "malo.md") {
		t.Errorf("el mensaje debe decir qué archivo falló: %v", err)
	}
	// Y el fallo de la etapa completa también.
	raiz := filepath.Join(t.TempDir(), "no-existe")
	_, _, err = CargarDocs(raiz)
	if err == nil {
		t.Fatal("una raíz inexistente debe fallar")
	}
	if !errors.Is(err, ErrStageFailed) {
		t.Errorf("err = %v, queremos un E_STAGE_FAILED", err)
	}
	if !contains(err.Error(), "no-existe") {
		t.Errorf("el mensaje debe decir qué raíz falló: %v", err)
	}
}

// Dependencia fuerte y relación no son lo mismo. Aplanarlas en una sola lista
// hacía que el grafo afirmara que BACKEND.md depende de FRONTEND.md, cuando la
// dependencia real va al revés: el frontend consume el contrato de eventos del
// backend.
func TestDependenciaFuerteYRelacionSeConsultanPorSeparado(t *testing.T) {
	raw := "---\n" +
		"depende_de:\n  - \"[[PROJECT]]\"\n  - \"[[backend/DECISIONS]]\"\n" +
		"relacionado:\n  - \"[[frontend/FRONTEND]]\"\n  - \"[[database/DATABASE]]\"\n" +
		"---\n\ncuerpo\n"
	m, _, hay, err := ParseFrontmatter([]byte(raw))
	if err != nil || !hay {
		t.Fatalf("hay=%v err=%v", hay, err)
	}
	docs := map[string]*Doc{
		"x.md": {Path: "x.md", Front: m, HasFront: hay, Links: []string{"mencion"}},
	}
	g := ConstruirGrafo(docs)

	fuertes := g.VecinosDeclarados("x.md")
	quiereFuertes := []string{"PROJECT", "backend/DECISIONS"}
	if !reflect.DeepEqual(fuertes, quiereFuertes) {
		t.Errorf("vecinos declarados = %v, queremos %v", fuertes, quiereFuertes)
	}
	rel := g.VecinosRelacionados("x.md")
	quiereRel := []string{"frontend/FRONTEND", "database/DATABASE"}
	if !reflect.DeepEqual(rel, quiereRel) {
		t.Errorf("vecinos relacionados = %v, queremos %v", rel, quiereRel)
	}
	// La lectura plana sigue siendo la unión, y por eso no puede usarse para
	// decidir qué hay que leer.
	plano := g.VecinosFrontmatter("x.md")
	if len(plano) != 4 {
		t.Errorf("la lectura plana debe dar las 4, dio %d: %v", len(plano), plano)
	}
	for _, d := range quiereRel {
		for _, f := range fuertes {
			if f == d {
				t.Errorf("%q es relación y no puede aparecer como dependencia fuerte", d)
			}
		}
	}
	// Un documento sin la clave de relación no inventa una lista.
	vacio := g.VecinosRelacionados("no-existe")
	if len(vacio) != 0 {
		t.Errorf("un documento inexistente no tiene relaciones: %v", vacio)
	}
}

// T-B003-05: cierre transitivo y consultas por etiqueta.
func TestCierreTransitivoYPorEtiqueta(t *testing.T) {
	docs, _, _ := CargarDocs("testdata")
	g := ConstruirGrafo(docs)
	cierre, err := g.Cierre("a.md")
	if err != nil {
		t.Fatal(err)
	}
	// a -> b, a -> c, c -> b: el cierre de a es {b, c} (sin bucles infinitos).
	if want := []string{"b.md", "una/c.md"}; !reflect.DeepEqual(cierre, want) {
		t.Errorf("cierre de a.md = %v, queremos %v", cierre, want)
	}
	if want := []string{"a.md", "b.md"}; !reflect.DeepEqual(g.PorEtiqueta("comun"), want) {
		t.Errorf("por etiqueta 'comun' = %v, queremos %v", g.PorEtiqueta("comun"), want)
	}
	if _, err := g.Documento("no-existe"); err == nil {
		t.Error("documento inexistente debe dar E_DOC_NOT_FOUND")
	} else if !contains(err.Error(), "E_DOC_NOT_FOUND") {
		t.Errorf("err = %v", err)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}

// Documentar un ejemplo de wiki link no puede convertirlo en dependencia. Ni en
// bloque de código ni en código en línea: ninguno de los dos se resuelve como
// enlace, ni al pintarse ni en Obsidian.
func TestUnEjemploDeWikiLinkNoEsUnaDependencia(t *testing.T) {
	casos := []struct {
		nombre string
		cuerpo string
		quiere []string
	}{
		{"bloque de código", "antes\n```\n[[no/EXISTE]]\n```\ndespues", nil},
		{"codigo en linea", "antes `[[no/EXISTE]]` despues", nil},
		{"varios tramos", "a `x` [[backend/DECISIONS]] b `[[no/EXISTE]]` c", []string{"backend/DECISIONS"}},
		{"enlace real sobrevive", "[[backend/DECISIONS]] `no [[esto]]`", []string{"backend/DECISIONS"}},
		{"acentos impares no rompen", "a ` sin cerrar [[backend/DECISIONS]]", nil},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			got := ExtraerEnlaces(c.cuerpo)
			if !reflect.DeepEqual(got, c.quiere) {
				t.Errorf("ExtraerEnlaces(%q) = %v, queremos %v", c.cuerpo, got, c.quiere)
			}
		})
	}
}
