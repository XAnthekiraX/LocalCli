package models

import (
	"os"
	"path/filepath"
	"testing"
)

// raizCatalogo crea una raíz tipo ~/.config/localcli con su models/ y devuelve
// el gestor y el directorio de modelos.
func raizCatalogo(t *testing.T) (*Gestor, string) {
	t.Helper()
	raiz := t.TempDir()
	dirModels := filepath.Join(raiz, "models")
	if err := os.MkdirAll(dirModels, 0o755); err != nil {
		t.Fatal(err)
	}
	return NuevoGestor(filepath.Join(raiz, "models.json")), dirModels
}

func TestRoundTripConservaCampos(t *testing.T) {
	g, dirModels := raizCatalogo(t)
	e := Entrada{
		ID:      "qwen3-14b",
		Ruta:    filepath.Join(dirModels, "qwen3-14b", "qwen3-14b-q4_k_m.gguf"),
		Repo:    "QuantFactory/Qwen3-14B-GGUF",
		Archivo: "qwen3-14b-q4_k_m.gguf",
		Ventana: 4096,
		Tamano:  9123456789,
	}
	if err := g.Alta(e); err != nil {
		t.Fatalf("alta: %v", err)
	}

	c, err := g.Cargar()
	if err != nil {
		t.Fatalf("cargar: %v", err)
	}
	if len(c.Modelos) != 1 {
		t.Fatalf("modelos = %d, quiero 1", len(c.Modelos))
	}
	if c.Modelos[0] != e {
		t.Fatalf("round-trip perdió campos: %+v", c.Modelos[0])
	}
}

func TestAltaTrasDescargaRegistraRutaCanonica(t *testing.T) {
	g, dirModels := raizCatalogo(t)
	v := Variante{Repo: "org/repo", Archivo: "m.gguf", Tamano: 2048}

	if err := g.AltaDescarga("Qwen3.8", v, g.RutaCanonica("Qwen3.8", v.Archivo), 8192, ""); err != nil {
		t.Fatalf("alta: %v", err)
	}

	c, err := g.Cargar()
	if err != nil {
		t.Fatal(err)
	}
	got, ok := c.Buscar("Qwen3.8")
	if !ok {
		t.Fatal("el modelo no quedó registrado")
	}
	quiero := filepath.Join(dirModels, "Qwen3.8", "m.gguf")
	if got.Ruta != quiero {
		t.Fatalf("ruta = %q, quiero %q", got.Ruta, quiero)
	}
	if got.Repo != v.Repo || got.Archivo != v.Archivo || got.Tamano != v.Tamano || got.Ventana != 8192 {
		t.Fatalf("metadatos mal registrados: %+v", got)
	}
}

func TestAltaReemplazaMismoID(t *testing.T) {
	g, _ := raizCatalogo(t)
	_ = g.Alta(Entrada{ID: "m", Repo: "viejo"})
	_ = g.Alta(Entrada{ID: "m", Repo: "nuevo"})

	c, _ := g.Cargar()
	if len(c.Modelos) != 1 || c.Modelos[0].Repo != "nuevo" {
		t.Fatalf("no reemplazó: %+v", c.Modelos)
	}
}

func TestAltaRechazaNombreQueEsRuta(t *testing.T) {
	g, _ := raizCatalogo(t)
	if err := g.Alta(Entrada{ID: "carpeta/modelo", Ruta: "x.gguf"}); err == nil {
		t.Fatal("un id con separador debería rechazarse")
	}
	if err := g.Alta(Entrada{ID: ""}); err == nil {
		t.Fatal("un id vacío debería rechazarse")
	}
}

func TestSeleccionPersiste(t *testing.T) {
	g, _ := raizCatalogo(t)
	_ = g.Alta(Entrada{ID: "a"})
	_ = g.Alta(Entrada{ID: "b"})

	if err := g.Seleccionar("b"); err != nil {
		t.Fatalf("seleccionar: %v", err)
	}
	if c, _ := g.Cargar(); c.Seleccionado != "b" {
		t.Fatalf("seleccionado = %q, quiero b", c.Seleccionado)
	}
	// Una segunda lectura (como al reiniciar) lo conserva.
	c, _ := g.Cargar()
	activo, ok := c.Activo()
	if !ok || activo.ID != "b" {
		t.Fatalf("activo = %+v, ok = %v", activo, ok)
	}

	if err := g.Seleccionar("no-existe"); err == nil {
		t.Fatal("seleccionar un modelo ausente debería fallar")
	}
}

func TestUsableCompruebaElGGUF(t *testing.T) {
	_, dirModels := raizCatalogo(t)
	ruta := filepath.Join(dirModels, "m", "m.gguf")
	if err := os.MkdirAll(filepath.Dir(ruta), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ruta, []byte("gguf"), 0o644); err != nil {
		t.Fatal(err)
	}
	e := Entrada{ID: "m", Ruta: ruta}
	if !e.Usable() {
		t.Fatal("el modelo con archivo presente debería ser usable")
	}
	if err := os.Remove(ruta); err != nil {
		t.Fatal(err)
	}
	if e.Usable() {
		t.Fatal("un modelo con archivo borrado no debería ser usable")
	}
}

func TestCatalogoAusenteEsVacio(t *testing.T) {
	g := NuevoGestor(filepath.Join(t.TempDir(), "no-existe", "models.json"))
	c, err := g.Cargar()
	if err != nil {
		t.Fatalf("ausente no debería dar error: %v", err)
	}
	if len(c.Modelos) != 0 || c.Seleccionado != "" {
		t.Fatalf("catálogo ausente debería ser vacío: %+v", c)
	}
}

func TestCatalogoMalFormadoAvisaYNoSePisa(t *testing.T) {
	g, _ := raizCatalogo(t)
	roto := []byte("{ esto no es json")
	if err := os.WriteFile(g.ruta, roto, 0o644); err != nil {
		t.Fatal(err)
	}

	c, err := g.Cargar()
	if err == nil {
		t.Fatal("un catálogo mal formado debería avisar")
	}
	if len(c.Modelos) != 0 {
		t.Fatalf("mal formado debería leerse como vacío: %+v", c)
	}

	if err := g.Alta(Entrada{ID: "m"}); err == nil {
		t.Fatal("el alta no debería pisar un catálogo mal formado")
	}
	datos, err := os.ReadFile(g.ruta)
	if err != nil {
		t.Fatal(err)
	}
	if string(datos) != string(roto) {
		t.Fatal("el catálogo mal formado se sobrescribió")
	}
}

func TestExpandirRuta(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if got := ExpandirRuta("~/models/x.gguf"); got != filepath.Join(home, "models", "x.gguf") {
		t.Fatalf("expandir = %q", got)
	}
	if got := ExpandirRuta("/abs/x.gguf"); got != "/abs/x.gguf" {
		t.Fatalf("una ruta absoluta no se toca: %q", got)
	}
}
