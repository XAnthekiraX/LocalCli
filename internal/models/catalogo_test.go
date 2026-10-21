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

// escribirModelo deja un GGUF y, si se pide, un proyector, dentro de la carpeta
// del modelo.
func escribirModelo(t *testing.T, dirModels, id string, extras ...string) string {
	t.Helper()
	carpeta := filepath.Join(dirModels, id)
	if err := os.MkdirAll(carpeta, 0o755); err != nil {
		t.Fatal(err)
	}
	ruta := filepath.Join(carpeta, "modelo.gguf")
	if err := os.WriteFile(ruta, []byte("gguf"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, nombre := range extras {
		if err := os.WriteFile(filepath.Join(carpeta, nombre), []byte("extra"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return ruta
}

func TestEliminarBorraLaCarpetaCompletaYLaEntrada(t *testing.T) {
	g, dirModels := raizCatalogo(t)
	ruta := escribirModelo(t, dirModels, "m", "mmproj-f16.gguf", "info.json")
	if err := g.Alta(Entrada{ID: "m", Ruta: ruta, MMProj: "mmproj-f16.gguf"}); err != nil {
		t.Fatal(err)
	}
	if err := g.Alta(Entrada{ID: "otro", Ruta: filepath.Join(dirModels, "otro", "modelo.gguf")}); err != nil {
		t.Fatal(err)
	}

	if err := g.Eliminar("m"); err != nil {
		t.Fatalf("eliminar: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dirModels, "m")); !os.IsNotExist(err) {
		t.Fatalf("la carpeta del modelo sigue ahí: %v", err)
	}
	c, err := g.Cargar()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := c.Buscar("m"); ok {
		t.Fatal("la entrada del modelo sigue en el catálogo")
	}
	if _, ok := c.Buscar("otro"); !ok {
		t.Fatal("eliminar un modelo borró la entrada de otro")
	}
}

func TestEliminarLimpiaLaEntradaHuerfana(t *testing.T) {
	g, _ := raizCatalogo(t)
	if err := g.Alta(Entrada{ID: "m", Ruta: "/no/existe/modelo.gguf"}); err != nil {
		t.Fatal(err)
	}

	if err := g.Eliminar("m"); err != nil {
		t.Fatalf("una carpeta ya ausente no debería impedir limpiar la entrada: %v", err)
	}
	if c, _ := g.Cargar(); len(c.Modelos) != 0 {
		t.Fatalf("la entrada huérfana sigue: %+v", c.Modelos)
	}
}

func TestEliminarRechazaIDInexistenteOInvalido(t *testing.T) {
	g, dirModels := raizCatalogo(t)
	ruta := escribirModelo(t, dirModels, "m")
	if err := g.Alta(Entrada{ID: "m", Ruta: ruta}); err != nil {
		t.Fatal(err)
	}

	for _, id := range []string{"no-existe", "", "..", "../fuera", "carpeta/modelo"} {
		if err := g.Eliminar(id); err == nil {
			t.Fatalf("eliminar %q debería fallar", id)
		}
	}
	if _, err := os.Stat(filepath.Join(dirModels, "m")); err != nil {
		t.Fatalf("un id inválido borró la carpeta del modelo: %v", err)
	}
	if c, _ := g.Cargar(); len(c.Modelos) != 1 {
		t.Fatalf("un id inválido tocó el catálogo: %+v", c.Modelos)
	}
}

func TestEliminarNoSigueLaRutaDelCatalogo(t *testing.T) {
	g, dirModels := raizCatalogo(t)
	fuera := t.TempDir()
	ajeno := filepath.Join(fuera, "ajeno.gguf")
	if err := os.WriteFile(ajeno, []byte("no se toca"), 0o644); err != nil {
		t.Fatal(err)
	}
	// La carpeta real del modelo existe; su `ruta` en el catálogo apunta fuera.
	escribirModelo(t, dirModels, "m")
	if err := g.Alta(Entrada{ID: "m", Ruta: ajeno}); err != nil {
		t.Fatal(err)
	}

	if err := g.Eliminar("m"); err != nil {
		t.Fatalf("eliminar: %v", err)
	}
	if _, err := os.Stat(ajeno); err != nil {
		t.Fatalf("el borrado siguió la ruta del catálogo y tocó un archivo ajeno: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dirModels, "m")); !os.IsNotExist(err) {
		t.Fatalf("la carpeta del modelo no se borró: %v", err)
	}
}

func TestEliminarNoPisaCatalogoMalFormado(t *testing.T) {
	g, dirModels := raizCatalogo(t)
	escribirModelo(t, dirModels, "m")
	roto := []byte("{ esto no es json")
	if err := os.WriteFile(g.ruta, roto, 0o644); err != nil {
		t.Fatal(err)
	}

	if err := g.Eliminar("m"); err == nil {
		t.Fatal("un catálogo mal formado debería avisar")
	}
	datos, err := os.ReadFile(g.ruta)
	if err != nil {
		t.Fatal(err)
	}
	if string(datos) != string(roto) {
		t.Fatal("el catálogo mal formado se sobrescribió")
	}
	if _, err := os.Stat(filepath.Join(dirModels, "m")); err != nil {
		t.Fatalf("no se debía borrar la carpeta con el catálogo mal formado: %v", err)
	}
}

func TestEliminarNoEscribeSiLaCarpetaNoSePuedeBorrar(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("como root los permisos no impiden borrar")
	}
	g, dirModels := raizCatalogo(t)
	ruta := escribirModelo(t, dirModels, "m")
	if err := g.Alta(Entrada{ID: "m", Ruta: ruta}); err != nil {
		t.Fatal(err)
	}
	carpeta := filepath.Join(dirModels, "m")
	t.Cleanup(func() { _ = os.Chmod(carpeta, 0o755) })
	if err := os.Chmod(carpeta, 0o555); err != nil {
		t.Fatal(err)
	}

	if err := g.Eliminar("m"); err == nil {
		t.Fatal("una carpeta que no se puede borrar debería dar error")
	}
	if c, _ := g.Cargar(); len(c.Modelos) != 1 {
		t.Fatalf("el catálogo se tocó aunque la carpeta no se borró: %+v", c.Modelos)
	}
}
