package store

import (
	"database/sql"
	"testing"
)

func sesionDePrueba(t *testing.T) (*sql.DB, string, string) {
	t.Helper()
	db, limpiar, err := CrearDBTemporal()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = limpiar() })
	projectID, err := ResolucionProyecto(db, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	s, err := CrearSesion(db, projectID, "prueba", nil)
	if err != nil {
		t.Fatal(err)
	}
	return db, projectID, s.ID
}

func TestCrearTodoAñadeSinPerderLosPasos(t *testing.T) {
	db, projectID, sesion := sesionDePrueba(t)

	if _, err := CrearTodo(db, projectID, sesion, "primero", "pendiente", "media"); err != nil {
		t.Fatal(err)
	}
	lista, err := CrearTodo(db, projectID, sesion, "segundo", "pendiente", "alta")
	if err != nil {
		t.Fatal(err)
	}
	if len(lista) != 2 || lista[0].Contenido != "primero" || lista[1].Contenido != "segundo" {
		t.Fatalf("añadir un paso no debería perder los anteriores: %+v", lista)
	}
	if lista[1].Posicion != 1 || lista[1].Prioridad != "alta" {
		t.Fatalf("el paso nuevo va al final: %+v", lista[1])
	}
}

func TestReemplazarTodosSustituyeLaListaEntera(t *testing.T) {
	db, projectID, sesion := sesionDePrueba(t)

	if _, err := CrearTodo(db, projectID, sesion, "viejo", "pendiente", "media"); err != nil {
		t.Fatal(err)
	}
	lista, err := ReemplazarTodos(db, projectID, sesion, []Todo{
		{Contenido: "uno", Estado: "en_progreso", Prioridad: "media"},
		{Contenido: "dos", Estado: "pendiente", Prioridad: "baja"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(lista) != 2 || lista[0].Contenido != "uno" || lista[1].Contenido != "dos" {
		t.Fatalf("la lista no se reemplazó: %+v", lista)
	}

	vacia, err := ReemplazarTodos(db, projectID, sesion, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(vacia) != 0 {
		t.Fatalf("una lista vacía debería dejar la lista en blanco: %+v", vacia)
	}
}
