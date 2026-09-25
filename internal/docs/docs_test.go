package docs

import (
	"errors"
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
	if got := len(docs); got != 4 {
		t.Errorf("documentos cargados = %d, queremos 4", got)
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
	if want := []string{"a.md", "una/c.md"}; !reflect.DeepEqual(g.Revés("b.md"), want) {
		t.Errorf("entrantes de b.md = %v, queremos %v", g.Revés("b.md"), want)
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
