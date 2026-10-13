package fileops

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"localcli/internal/tools"
)

func TestBuscarArchivosPorNombre(t *testing.T) {
	raiz := t.TempDir()
	if err := os.WriteFile(filepath.Join(raiz, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(raiz, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(raiz, "sub", "main_test.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Una carpeta de control de versiones no se recorre.
	if err := os.MkdirAll(filepath.Join(raiz, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(raiz, ".git", "main.go"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	o := &Operaciones{Raiz: raiz}
	res, err := o.BuscarArchivosHandler()(context.Background(), tools.PeticionBuscarArchivos{Patron: "main"}, tools.Contexto{})
	if err != nil || res.Error != "" {
		t.Fatalf("no debería fallar: %+v %v", res, err)
	}
	var rutas []string
	if err := json.Unmarshal([]byte(res.Salida), &rutas); err != nil {
		t.Fatalf("la salida no es una lista de rutas: %v (%s)", err, res.Salida)
	}
	if strings.Join(rutas, "|") != "main.go|sub/main_test.go" {
		t.Fatalf("rutas = %v", rutas)
	}
	if res.Meta["medida"] != "2 coincidencias" {
		t.Fatalf("medida = %v", res.Meta["medida"])
	}
}

func TestBuscarEnArchivosPorContenido(t *testing.T) {
	raiz := t.TempDir()
	if err := os.WriteFile(filepath.Join(raiz, "a.txt"), []byte("uno\ndos\nAQUÍ va\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(raiz, "b.txt"), []byte("aquí también\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	o := &Operaciones{Raiz: raiz}
	res, err := o.BuscarEnArchivosHandler()(context.Background(), tools.PeticionBuscarEnArchivos{Patron: "aquí"}, tools.Contexto{})
	if err != nil || res.Error != "" {
		t.Fatalf("no debería fallar: %+v %v", res, err)
	}
	var coincidencias []Coincidencia
	if err := json.Unmarshal([]byte(res.Salida), &coincidencias); err != nil {
		t.Fatalf("la salida no es una lista de coincidencias: %v (%s)", err, res.Salida)
	}
	if len(coincidencias) != 2 {
		t.Fatalf("coincidencias = %+v", coincidencias)
	}
	if coincidencias[0].Archivo != "a.txt" || coincidencias[0].Linea != 3 {
		t.Fatalf("la primera coincidencia = %+v", coincidencias[0])
	}
	if res.Meta["medida"] != "2 coincidencias" {
		t.Fatalf("medida = %v", res.Meta["medida"])
	}
}

func TestBuscarEnArchivosAcotaPorRuta(t *testing.T) {
	raiz := t.TempDir()
	if err := os.MkdirAll(filepath.Join(raiz, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(raiz, "raiz.txt"), []byte("aguja\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(raiz, "sub", "hijo.txt"), []byte("aguja\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	o := &Operaciones{Raiz: raiz}
	res, err := o.BuscarEnArchivosHandler()(context.Background(), tools.PeticionBuscarEnArchivos{Patron: "aguja", Ruta: "sub"}, tools.Contexto{})
	if err != nil || res.Error != "" {
		t.Fatalf("no debería fallar: %+v %v", res, err)
	}
	var coincidencias []Coincidencia
	_ = json.Unmarshal([]byte(res.Salida), &coincidencias)
	if len(coincidencias) != 1 || coincidencias[0].Archivo != "sub/hijo.txt" {
		t.Fatalf("la búsqueda debería acotarse a sub/: %+v", coincidencias)
	}
}

func TestBuscarEnArchivosFueraExigeMotivo(t *testing.T) {
	raiz := t.TempDir()
	o := &Operaciones{Raiz: raiz}
	res, err := o.BuscarEnArchivosHandler()(context.Background(), tools.PeticionBuscarEnArchivos{Patron: "x", Ruta: "../fuera"}, tools.Contexto{})
	if err != nil {
		t.Fatalf("no debería ser un fallo de harness: %v", err)
	}
	if !strings.HasPrefix(res.Error, tools.CodPathOutside) {
		t.Fatalf("una ruta exterior sin motivo debería dar E_PATH_OUTSIDE: %q", res.Error)
	}
}

func TestBuscarRecorreCarpetasOcultasDeContenido(t *testing.T) {
	raiz := t.TempDir()
	if err := os.MkdirAll(filepath.Join(raiz, ".agents", "commands"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(raiz, ".agents", "commands", "actualizar.md"), []byte("# Comando: actualizar\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Las dependencias y el control de versiones siguen fuera.
	if err := os.MkdirAll(filepath.Join(raiz, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(raiz, ".git", "actualizar.md"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	o := &Operaciones{Raiz: raiz}
	res, err := o.BuscarArchivosHandler()(context.Background(), tools.PeticionBuscarArchivos{Patron: "actualizar"}, tools.Contexto{})
	if err != nil || res.Error != "" {
		t.Fatalf("no debería fallar: %+v %v", res, err)
	}
	var rutas []string
	if err := json.Unmarshal([]byte(res.Salida), &rutas); err != nil {
		t.Fatalf("la salida no es una lista de rutas: %v (%s)", err, res.Salida)
	}
	if strings.Join(rutas, "|") != ".agents/commands/actualizar.md" {
		t.Fatalf("debería encontrar el archivo oculto y evitar .git: %v", rutas)
	}

	res, err = o.BuscarEnArchivosHandler()(context.Background(), tools.PeticionBuscarEnArchivos{Patron: "actualizar"}, tools.Contexto{})
	if err != nil || res.Error != "" {
		t.Fatalf("no debería fallar: %+v %v", res, err)
	}
	var coincidencias []Coincidencia
	if err := json.Unmarshal([]byte(res.Salida), &coincidencias); err != nil {
		t.Fatalf("la salida no es una lista de coincidencias: %v (%s)", err, res.Salida)
	}
	if len(coincidencias) != 1 || coincidencias[0].Archivo != ".agents/commands/actualizar.md" {
		t.Fatalf("la búsqueda de contenido debería ver .agents: %+v", coincidencias)
	}
}

func TestBuscarPatronVacioEsCorregible(t *testing.T) {
	o := &Operaciones{Raiz: t.TempDir()}
	res, _ := o.BuscarArchivosHandler()(context.Background(), tools.PeticionBuscarArchivos{Patron: "  "}, tools.Contexto{})
	if !strings.HasPrefix(res.Error, tools.CodBadArgs) {
		t.Fatalf("un patrón vacío debería dar E_BAD_ARGS: %q", res.Error)
	}
}
