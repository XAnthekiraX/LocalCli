package docs

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

// T-B020-04: la carga encuentra todos los .md y reporta el roto sin abortar.
func TestCargarDocsTodosLosMarkdown(t *testing.T) {
	docs, errs, err := CargarDocs("testdata")
	if err != nil {
		t.Fatal(err)
	}
	queridas := []string{"a.md", "b.md", "frente.md", "sin_front.md", "una/c.md"}
	for _, q := range queridas {
		if _, ok := docs[q]; !ok {
			t.Errorf("falta %s en la carga; rutas: %v", q, Rutas(docs))
		}
	}
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
	if got := len(docs); got != 5 {
		t.Errorf("documentos cargados = %d, queremos 5", got)
	}
}

// T-B020-02: con y sin frontmatter parsea correcto.
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

// T-B020-03: enlaces válidos y rotos se listan por separado.
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

// T-B020-05: el frontmatter crea aristas y una etiqueta no.
func TestGrafoAristasDelFrontmatter(t *testing.T) {
	docs, _, _ := CargarDocs("testdata")
	g := ConstruirGrafo(docs)

	if want := []string{"sin_front"}; !reflect.DeepEqual(g.VecinosDeclarados("frente.md"), want) {
		t.Errorf("vecinos declarados de frente.md = %v, queremos %v", g.VecinosDeclarados("frente.md"), want)
	}
	if want := []string{"sin_front.md"}; !reflect.DeepEqual(g.Vecinos("frente.md"), want) {
		t.Errorf("aristas de frente.md = %v, queremos %v (arista del frontmatter resuelta)", g.Vecinos("frente.md"), want)
	}
	// Una etiqueta compartida (comun) no crea aristas: b solo enlaza a a.
	if want := []string{"a.md"}; !reflect.DeepEqual(g.Vecinos("b.md"), want) {
		t.Errorf("aristas de b.md = %v, queremos %v", g.Vecinos("b.md"), want)
	}
	if want := []string{"a.md", "b.md", "una/c.md"}; !reflect.DeepEqual(g.PorEtiqueta("comun"), want) {
		t.Errorf("PorEtiqueta(comun) = %v, queremos %v", g.PorEtiqueta("comun"), want)
	}
}

// T-B020-06: consultas por documento y cierre; E_DOC_NOT_FOUND si no existe.
func TestConsultas(t *testing.T) {
	docs, _, _ := CargarDocs("testdata")
	g := ConstruirGrafo(docs)

	if _, err := g.Documento("b"); err != nil {
		t.Errorf("Documento(b): %v", err)
	}
	if _, err := g.Documento("nope"); !errors.Is(err, ErrDocNotFound) {
		t.Errorf("Documento(nope) = %v, esperaba E_DOC_NOT_FOUND", err)
	}
	directas, err := g.Directas("a.md")
	if err != nil {
		t.Fatalf("Directas: %v", err)
	}
	if want := []string{"b.md", "una/c.md"}; !reflect.DeepEqual(directas, want) {
		t.Errorf("directas de a.md = %v, queremos %v", directas, want)
	}
}

// T-B020-08: el grafo implementa context.Fuente.
func TestFuenteContexto(t *testing.T) {
	docs, _, _ := CargarDocs("testdata")
	g := ConstruirGrafo(docs)

	var encontrado bool
	for _, d := range g.Documentos() {
		if d.Ruta == "a.md" {
			encontrado = true
			if want := []string{"b"}; !reflect.DeepEqual(d.Dependencias, want) {
				t.Errorf("dependencias de a.md = %v, queremos %v", d.Dependencias, want)
			}
		}
	}
	if !encontrado {
		t.Error("a.md no aparece en el catálogo de la fuente")
	}
	cuerpo, err := g.Leer("a")
	if err != nil {
		t.Fatalf("Leer(a): %v", err)
	}
	if !strings.Contains(cuerpo, "Cuerpo de A") {
		t.Errorf("el cuerpo leído no contiene A: %q", cuerpo)
	}
	if _, err := g.Leer("nope"); !errors.Is(err, ErrDocNotFound) {
		t.Errorf("Leer(nope) = %v, esperaba E_DOC_NOT_FOUND", err)
	}
}

// T-B020-07: E_DOC_PARSE explica y E_STAGE_FAILED avisa, sin leer el mensaje.
func TestErroresLocalizados(t *testing.T) {
	_, errs, err := CargarDocs("testdata")
	if err != nil {
		t.Fatal(err)
	}
	if len(errs) == 0 {
		t.Fatal("esperaba el error localizado de malo.md")
	}
	if !errors.Is(errs[0], ErrStageFailed) {
		t.Errorf("un frontmatter roto debe reconocerse como E_STAGE_FAILED: %v", errs[0])
	}
}

// T-B020-04: una raíz que no es carpeta de documentación detiene la carga.
func TestCargarDocsRaizNoEsCarpeta(t *testing.T) {
	if _, _, err := CargarDocs("no-existe-esta-carpeta"); !errors.Is(err, ErrStageFailed) {
		t.Errorf("CargarDocs(inexistente) = %v, esperaba E_STAGE_FAILED", err)
	}
}
