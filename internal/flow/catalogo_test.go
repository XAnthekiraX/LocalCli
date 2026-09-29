package flow

import (
	"os"
	"path/filepath"
	"testing"
)

// El catálogo por defecto no registra flujos: los flujos son los archivos de
// `.localcli/flows/`. `/ejecutar` no es un flujo —consume la cola— y se
// reconoce igual.
func TestCatalogoPorDefectoEstaVacio(t *testing.T) {
	c := CatalogoPorDefecto()
	if len(c.Flujos()) != 0 {
		t.Errorf("el catálogo por defecto no registra flujos: %v", c.Flujos())
	}
	if cmd, ok := c.De("/ejecutar"); !ok || !cmd.Consumir {
		t.Errorf("/ejecutar debe reconocerse y consumir la cola: %+v, %v", cmd, ok)
	}
	if _, ok := c.De("/resolver"); ok {
		t.Error("un comando sin JSON en la carpeta no debe reconocerse")
	}
}

// CargarFlujos lee `.localcli/flows/*.json`: los flujos del proyecto son los
// archivos que hay, y los que no tienen archivo no existen.
func TestCargarFlujosLeeLaCarpetaDelProyecto(t *testing.T) {
	raiz := t.TempDir()
	dir := filepath.Join(raiz, ".localcli", "flows")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	demo := `{"comando":"/demo","nombre":"demo","pregunta":"algo","etapas":[{"id":"p","nombre":"P","agente":"plan","pregunta":"¿p?","entrega":true}]}`
	otro := `{"comando":"/otro","nombre":"otro","pregunta":"algo","etapas":[{"id":"p","nombre":"P","agente":"plan","pregunta":"¿p?","entrega":true}]}`
	if err := os.WriteFile(filepath.Join(dir, "demo.json"), []byte(demo), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "otro.json"), []byte(otro), 0o644); err != nil {
		t.Fatal(err)
	}

	c, err := CargarFlujos(raiz)
	if err != nil {
		t.Fatalf("CargarFlujos: %v", err)
	}
	if _, ok := c.De("/demo"); !ok {
		t.Error("/demo debe estar en el catálogo")
	}
	if _, ok := c.De("/otro"); !ok {
		t.Error("/otro debe estar en el catálogo")
	}
	if _, ok := c.De("/resolver"); ok {
		t.Error("sin archivo, /resolver no debe existir en el catálogo")
	}
}

// Un comando repetido en dos archivos: gana el último en orden alfabético.
func TestCargarFlujosUnComandoRepetidoSeReemplaza(t *testing.T) {
	raiz := t.TempDir()
	dir := filepath.Join(raiz, ".localcli", "flows")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	primero := `{"comando":"/demo","nombre":"demo-uno","pregunta":"algo","etapas":[{"id":"p","nombre":"P","agente":"plan","pregunta":"¿p?","entrega":true}]}`
	segundo := `{"comando":"/demo","nombre":"demo-dos","pregunta":"algo","etapas":[{"id":"p","nombre":"P","agente":"plan","pregunta":"¿p?","entrega":true}]}`
	if err := os.WriteFile(filepath.Join(dir, "a.json"), []byte(primero), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.json"), []byte(segundo), 0o644); err != nil {
		t.Fatal(err)
	}

	c, err := CargarFlujos(raiz)
	if err != nil {
		t.Fatalf("CargarFlujos: %v", err)
	}
	if f, _ := c.PorNombre("demo-dos"); f.Nombre == "" {
		t.Errorf("el último archivo debe ganar por comando: %+v", c.Flujos())
	}
}

func TestCargarFlujosJSONRotoDevuelveError(t *testing.T) {
	raiz := t.TempDir()
	dir := filepath.Join(raiz, ".localcli", "flows")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "roto.json"), []byte("{no"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := CargarFlujos(raiz); err == nil {
		t.Fatal("un JSON roto debe devolver error para que el arranque avise")
	}
}

func TestComandoObjetivoUsaLaPeticionDelFlujo(t *testing.T) {
	f := Flujo{Nombre: "resolver", Comando: "/resolver", Peticion: "arreglar lo que está roto"}
	cmd := Comando{Nombre: "/resolver", Flujo: f}
	// Con texto detrás, gana el texto.
	if got := cmd.Objetivo("/resolver el login falla"); got != "el login falla" {
		t.Errorf("objetivo = %q", got)
	}
	// Sin texto, la petición del flujo.
	if got := cmd.Objetivo("/resolver"); got != "arreglar lo que está roto" {
		t.Errorf("objetivo = %q", got)
	}
	// Sin petición declarada, el nombre del comando: nunca queda vacío.
	solo := Comando{Nombre: "/planificar"}
	if got := solo.Objetivo("/planificar"); got != "/planificar" {
		t.Errorf("objetivo = %q", got)
	}
}
