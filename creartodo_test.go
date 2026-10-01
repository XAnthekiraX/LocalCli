// creartodo_test.go — T-B035: `crear_todo` añade y `actualizar_todo`
// reemplaza, pero las dos devuelven el MISMO checklist de la lista resultante.
package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"localcli/internal/session"
	"localcli/internal/store"
	"localcli/internal/tools"
)

// TestAmbasDevuelvenElMismoChecklist — construir la misma lista por las dos vías
// produce la misma respuesta: una herramienta de añadir no inventa su propio
// formato, y el modelo no tiene que distinguir de dónde salió la lista.
func TestAmbasDevuelvenElMismoChecklist(t *testing.T) {
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

	ad := &Adaptador{bus: session.NuevoBus()}
	todos := store.NuevosTodos(conexion)
	reemplazar := herramientaActualizarTodo(ad, todos)
	crear := herramientaCrearTodo(ad, todos)

	sesA, err := store.CrearSesion(conexion, "a", "")
	if err != nil {
		t.Fatalf("CrearSesion a: %v", err)
	}
	resA, err := reemplazar(context.Background(), &tools.PeticionActualizarTodo{Elementos: []tools.ElementoTodo{
		{Contenido: "primero", Estado: "pendiente"},
		{Contenido: "segundo", Estado: "en_progreso"},
	}}, tools.Contexto{SesionID: sesA.ID})
	if err != nil {
		t.Fatalf("actualizar_todo: %v", err)
	}

	sesB, err := store.CrearSesion(conexion, "b", "")
	if err != nil {
		t.Fatalf("CrearSesion b: %v", err)
	}
	if _, err := crear(context.Background(), &tools.PeticionCrearTodo{Contenido: "primero", Estado: "pendiente"}, tools.Contexto{SesionID: sesB.ID}); err != nil {
		t.Fatalf("crear_todo primero: %v", err)
	}
	resB, err := crear(context.Background(), &tools.PeticionCrearTodo{Contenido: "segundo", Estado: "en_progreso"}, tools.Contexto{SesionID: sesB.ID})
	if err != nil {
		t.Fatalf("crear_todo segundo: %v", err)
	}

	if resA.Salida != resB.Salida {
		t.Errorf("checklist de actualizar_todo = %q, de crear_todo = %q", resA.Salida, resB.Salida)
	}
	if resB.Salida == "" {
		t.Error("el checklist no puede venir vacío")
	}

	// Y la lista de `crear_todo` sigue teniendo los dos pasos, no solo el último.
	items, err := todos.Leer(sesB.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("crear_todo dos veces debe dejar dos pasos: %+v", items)
	}
}
