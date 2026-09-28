// herramientas_test.go — T-B024-01 y T-B024-06: la carpeta del proyecto del
// ejecutor y la aprobación única por `Contexto.Ask`.
package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"localcli/internal/store"
	"localcli/internal/tools"
)

// proyectoDePrueba crea un proyecto mínimo (con ai/docs/PROJECT.md, que es lo
// que hace que `store.Open` reconozca la carpeta) y devuelve su carpeta y el
// repositorio de cambios sobre su base.
func proyectoDePrueba(t *testing.T) (string, *store.Cambios) {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "ai", "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ai", "docs", "PROJECT.md"), []byte("# prueba\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	conexion, err := store.Open(dir)
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { _ = conexion.Close() })
	return dir, store.NuevosCambios(conexion)
}

// TestHerramientaDeEscrituraPideAprobacionPorAsk — T-B024-06: una herramienta
// de escritura no monta su propia notificación: pide por `Contexto.Ask`. Sin
// aprobación no toca el disco; con ella, escribe.
func TestHerramientaDeEscrituraPideAprobacionPorAsk(t *testing.T) {
	proyecto, cambios := proyectoDePrueba(t)
	ad := &Adaptador{}
	h := herramientaEscritura(proyecto, cambios, ad, "crear_archivo")

	// Declinada: no hay archivo.
	pedidas := 0
	c := tools.Contexto{Ask: func(ctx context.Context, s tools.Solicitud) (tools.Decision, error) {
		pedidas++
		return tools.Decision{Aprobada: false}, nil
	}}
	res, err := h(context.Background(), &tools.PeticionCrearArchivo{Ruta: "a.txt", Contenido: "hola"}, c)
	if err != nil {
		t.Fatalf("no es un error duro: %v", err)
	}
	if pedidas != 1 || res.Error == "" {
		t.Fatalf("pedidas=%d res=%+v, quiero una aprobación y un resultado con el motivo", pedidas, res)
	}
	if _, statErr := os.Stat(filepath.Join(proyecto, "a.txt")); statErr == nil {
		t.Fatal("no puede haber archivo sin aprobación")
	}

	// Aprobada: el archivo existe.
	c.Ask = func(ctx context.Context, s tools.Solicitud) (tools.Decision, error) {
		return tools.Decision{Aprobada: true, Explicita: true}, nil
	}
	if _, err := h(context.Background(), &tools.PeticionCrearArchivo{Ruta: "a.txt", Contenido: "hola"}, c); err != nil {
		t.Fatalf("aprobada: %v", err)
	}
	if datos, statErr := os.ReadFile(filepath.Join(proyecto, "a.txt")); statErr != nil || string(datos) != "hola" {
		t.Fatalf("la escritura aprobada no se aplicó: %q, %v", datos, statErr)
	}
}

// TestEjecutorUsaLaCarpetaDelProyecto — T-B024-01: el ejecutor corre dentro de
// la carpeta del proyecto; la carpeta del payload es una subcarpeta suya, no
// la raíz del comando.
func TestEjecutorUsaLaCarpetaDelProyecto(t *testing.T) {
	proyecto := t.TempDir()
	sub := filepath.Join(proyecto, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	// A propósito sin formatear: `gofmt -l` solo lista los archivos que
	// cambiaría.
	if err := os.WriteFile(filepath.Join(sub, "marca.go"), []byte("package sub\n\nfunc  F( )  {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	h := herramientaEjecutarComando(proyecto)
	// `gofmt -l .` está en la lista blanca: no pide aprobación y lista los
	// archivos de la carpeta de trabajo. Si el ejecutor corriera en otra
	// carpeta, no encontraría `marca.go`.
	res, err := h(context.Background(), &tools.PeticionEjecutarComando{Comando: "gofmt -l .", Carpeta: "sub"}, tools.Contexto{})
	if err != nil {
		t.Fatalf("Ejecutar: %v", err)
	}
	if !strings.Contains(res.Salida, "marca.go") {
		t.Errorf("la terminal debe correr en la carpeta del proyecto (sub): %q", res.Salida)
	}
}
