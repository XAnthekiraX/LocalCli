package session

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"localcli/internal/llama"
	"localcli/internal/llm/openai"
	"localcli/internal/models"
	"localcli/internal/store"
)

// catalogoConModelos deja un catálogo con esas carpetas, cada una con su GGUF.
func catalogoConModelos(t *testing.T, ids ...string) *models.Gestor {
	t.Helper()
	dir := t.TempDir()
	dirModels := filepath.Join(dir, "models")
	for _, id := range ids {
		carpeta := filepath.Join(dirModels, id)
		if err := os.MkdirAll(carpeta, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(carpeta, id+".gguf"), []byte("gguf"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	g := models.NuevoGestor(filepath.Join(dir, "models.json"))
	for _, id := range ids {
		if err := g.Alta(models.Entrada{ID: id, Ruta: g.RutaCanonica(id, id+".gguf")}); err != nil {
			t.Fatal(err)
		}
	}
	return g
}

// orquestadorDeModelos monta un orquestador con ese catálogo y ese modelo activo.
func orquestadorDeModelos(t *testing.T, modelo string, g *models.Gestor, runtime *llama.Servidor) *Orquestador {
	t.Helper()
	db, limpiar, _ := store.CrearDBTemporal()
	t.Cleanup(func() { _ = limpiar() })
	cwd := t.TempDir()
	projectID, err := store.ResolucionProyecto(db, cwd)
	if err != nil {
		t.Fatal(err)
	}
	return Nuevo(db, projectID, cwd, agentesDePrueba(t), openai.NuevoCliente("http://127.0.0.1:1"), modelo, 0, runtime, g)
}

// modeloEnMarcha arranca un llama-server de mentira que sirve ese modelo hasta
// que se detenga: es la tercera fuente del modelo activo.
func modeloEnMarcha(t *testing.T, id, ruta string) *llama.Servidor {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "llama-server")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\nexec sleep 300\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	s, err := llama.Nuevo(bin, "http://127.0.0.1:19997", 4096)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Arrancar(context.Background(), llama.Modelo{Nombre: id, Ruta: ruta}); err != nil {
		t.Fatalf("arrancar el servidor de prueba: %v", err)
	}
	t.Cleanup(func() { _ = s.Detener() })
	return s
}

// exigirModeloIntacto comprueba que la carpeta y la entrada siguen ahí.
func exigirModeloIntacto(t *testing.T, g *models.Gestor, id string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(g.DirectorioModels(), id)); err != nil {
		t.Fatalf("la carpeta de %q se tocó: %v", id, err)
	}
	c, err := g.Cargar()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := c.Buscar(id); !ok {
		t.Fatalf("la entrada de %q se tocó", id)
	}
}

func TestEliminarModeloBorraElNoActivo(t *testing.T) {
	g := catalogoConModelos(t, "uno", "dos")
	o := orquestadorDeModelos(t, "uno", g, nil)

	if err := o.EliminarModelo("dos"); err != nil {
		t.Fatalf("eliminar el modelo no activo: %v", err)
	}
	if _, err := os.Stat(filepath.Join(g.DirectorioModels(), "dos")); !os.IsNotExist(err) {
		t.Fatalf("la carpeta del modelo eliminado sigue ahí: %v", err)
	}
	c, err := g.Cargar()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := c.Buscar("dos"); ok {
		t.Fatal("la entrada del modelo eliminado sigue en el catálogo")
	}
	exigirModeloIntacto(t, g, "uno")
}

func TestEliminarModeloRechazaElDeLaSesion(t *testing.T) {
	g := catalogoConModelos(t, "uno", "dos")
	o := orquestadorDeModelos(t, "uno", g, nil)

	err := o.EliminarModelo("uno")
	if err == nil || !strings.Contains(err.Error(), "en uso") {
		t.Fatalf("el modelo de la sesión debería rechazarse con un aviso, err = %v", err)
	}
	exigirModeloIntacto(t, g, "uno")
}

func TestEliminarModeloRechazaElSeleccionado(t *testing.T) {
	g := catalogoConModelos(t, "uno", "dos")
	if err := g.Seleccionar("dos"); err != nil {
		t.Fatal(err)
	}
	o := orquestadorDeModelos(t, "", g, nil)

	err := o.EliminarModelo("dos")
	if err == nil || !strings.Contains(err.Error(), "en uso") {
		t.Fatalf("el modelo seleccionado debería rechazarse con un aviso, err = %v", err)
	}
	exigirModeloIntacto(t, g, "dos")
}

func TestEliminarModeloRechazaElQueSirveLlamaServer(t *testing.T) {
	g := catalogoConModelos(t, "uno", "dos")
	c, err := g.Cargar()
	if err != nil {
		t.Fatal(err)
	}
	entrada, ok := c.Buscar("dos")
	if !ok {
		t.Fatal("falta el modelo dos en el catálogo")
	}
	o := orquestadorDeModelos(t, "", g, modeloEnMarcha(t, "dos", entrada.RutaReal()))

	err = o.EliminarModelo("dos")
	if err == nil || !strings.Contains(err.Error(), "en uso") {
		t.Fatalf("el modelo servido por llama-server debería rechazarse, err = %v", err)
	}
	exigirModeloIntacto(t, g, "dos")
}
