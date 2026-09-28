package flow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// demoJSON es una definición válida mínima, como la escribiría un proyecto.
const demoJSON = `{
  "comando": "/demo",
  "nombre": "demo",
  "descripcion": "un flujo de prueba",
  "peticion": "hacer la demo",
  "reglas": ["una regla", "  "],
  "etapas": [
    {"id": "p1", "nombre": "Paso 1", "agente": "plan", "aprobacion": false, "instruccion": "haz algo"},
    {"id": "p2", "nombre": "Paso 2", "agente": "build", "aprobacion": true}
  ]
}`

func TestDecodificarFlujoValido(t *testing.T) {
	f, err := DecodificarFlujo([]byte(demoJSON))
	if err != nil {
		t.Fatalf("DecodificarFlujo: %v", err)
	}
	if f.Comando != "/demo" || f.Nombre != "demo" || f.Peticion != "hacer la demo" {
		t.Fatalf("cabecera mal decodificada: %+v", f)
	}
	// Las reglas se limpian: el elemento en blanco se descarta.
	if len(f.Reglas) != 1 || f.Reglas[0] != "una regla" {
		t.Errorf("reglas = %v, quiero [una regla]", f.Reglas)
	}
	if len(f.Etapas) != 2 {
		t.Fatalf("etapas = %d, quiero 2", len(f.Etapas))
	}
	if f.Etapas[0].Agente != "plan" || f.Etapas[1].Agente != "build" || !f.Etapas[1].Aprobacion {
		t.Errorf("etapas mal decodificadas: %+v", f.Etapas)
	}
	if err := f.Validar(); err != nil {
		t.Errorf("el flujo decodificado debe validar: %v", err)
	}
}

func TestDecodificarFlujoRechazaErrores(t *testing.T) {
	casos := map[string]string{
		"campo desconocido": `{"comando":"/x","nombre":"x","etapas":[{"id":"p","agente":"plan"}],"extra":1}`,
		"sin comando":       `{"nombre":"x","etapas":[{"id":"p","agente":"plan"}]}`,
		"sin nombre":        `{"comando":"/x","etapas":[{"id":"p","agente":"plan"}]}`,
		"sin etapas":        `{"comando":"/x","nombre":"x","etapas":[]}`,
		"comando sin barra": `{"comando":"x","nombre":"x","etapas":[{"id":"p","agente":"plan"}]}`,
		"sin id":            `{"comando":"/x","nombre":"x","etapas":[{"agente":"plan"}]}`,
		"etapa sin agente":  `{"comando":"/x","nombre":"x","etapas":[{"id":"p"}]}`,
		"contenido de más":  demoJSON + `{}`,
	}
	for nombre, datos := range casos {
		if _, err := DecodificarFlujo([]byte(datos)); err == nil {
			t.Errorf("%s: se esperaba error", nombre)
		}
	}
}

func TestCargarFlujosCarpetaInexistente(t *testing.T) {
	flujos, err := CargarFlujosCarpeta(filepath.Join(t.TempDir(), "no-existe"))
	if err != nil {
		t.Fatalf("una carpeta inexistente no es error: %v", err)
	}
	if len(flujos) != 0 {
		t.Errorf("flujos = %v, quiero vacío", flujos)
	}
}

func TestCargarFlujosCarpetaLeeEnOrden(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "b.json"), []byte(strings.Replace(demoJSON, "/demo", "/b", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a.json"), []byte(strings.Replace(demoJSON, "/demo", "/a", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	// Un archivo que no es .json se ignora.
	if err := os.WriteFile(filepath.Join(dir, "notas.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	flujos, err := CargarFlujosCarpeta(dir)
	if err != nil {
		t.Fatalf("CargarFlujosCarpeta: %v", err)
	}
	if len(flujos) != 2 || flujos[0].Comando != "/a" || flujos[1].Comando != "/b" {
		t.Fatalf("flujos = %+v, quiero [/a /b] en orden", flujos)
	}
}

func TestCargarFlujosCarpetaJSONRoto(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "roto.json"), []byte("{no es json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := CargarFlujosCarpeta(dir); err == nil {
		t.Fatal("un JSON roto debe detener la carga")
	}
}
