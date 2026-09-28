package flow

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCatalogoPorDefectoReconoceLosSeis(t *testing.T) {
	c := CatalogoPorDefecto()
	for comando, nombre := range map[string]string{
		"/planificar": "planificacion",
		"/crear":      "trabajo/crear",
		"/actualizar": "trabajo/actualizar",
		"/eliminar":   "trabajo/eliminar",
		"/resolver":   "resolver",
	} {
		cmd, ok := c.De(comando)
		if !ok {
			t.Errorf("%s debe reconocerse", comando)
			continue
		}
		if cmd.Consumir {
			t.Errorf("%s no consume la cola", comando)
		}
		if cmd.Flujo.Nombre != nombre {
			t.Errorf("%s: flujo = %q, quiero %q", comando, cmd.Flujo.Nombre, nombre)
		}
	}
	if cmd, ok := c.De("/ejecutar"); !ok || !cmd.Consumir {
		t.Errorf("/ejecutar debe reconocerse y consumir la cola: %+v, %v", cmd, ok)
	}
	if _, ok := c.De("/otro"); ok {
		t.Error("un comando fuera del catálogo no debe reconocerse")
	}
}

func TestCargarFlujosAnadeYReescribe(t *testing.T) {
	raiz := t.TempDir()
	dir := filepath.Join(raiz, "ai", "flows")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Un flujo propio nuevo y una personalización de uno oficial.
	propio := `{"comando":"/demo","nombre":"demo","etapas":[{"id":"p","nombre":"P","agente":"plan"}]}`
	override := `{"comando":"/crear","nombre":"crear-custom","etapas":[{"id":"p","nombre":"P","agente":"plan"}]}`
	if err := os.WriteFile(filepath.Join(dir, "demo.json"), []byte(propio), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "crear.json"), []byte(override), 0o644); err != nil {
		t.Fatal(err)
	}

	c, err := CargarFlujos(raiz)
	if err != nil {
		t.Fatalf("CargarFlujos: %v", err)
	}
	if _, ok := c.De("/demo"); !ok {
		t.Error("el flujo propio /demo debe estar en el catálogo")
	}
	if f, _ := c.PorNombre("crear-custom"); f.Nombre == "" {
		t.Error("la personalización de /crear debe reemplazar al oficial")
	}
	// Los oficiales no tocados siguen ahí.
	if _, ok := c.De("/resolver"); !ok {
		t.Error("/resolver debe seguir en el catálogo")
	}
}

func TestCargarFlujosJSONRotoDevuelveError(t *testing.T) {
	raiz := t.TempDir()
	dir := filepath.Join(raiz, "ai", "flows")
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
