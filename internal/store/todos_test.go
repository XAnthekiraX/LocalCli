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

	if _, err := CrearTodo(db, projectID, sesion, Todo{Contenido: "primero", Estado: "pendiente", Prioridad: "media"}, nil); err != nil {
		t.Fatal(err)
	}
	lista, err := CrearTodo(db, projectID, sesion, Todo{Contenido: "segundo", Estado: "pendiente", Prioridad: "alta"}, nil)
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

	if _, err := CrearTodo(db, projectID, sesion, Todo{Contenido: "viejo", Estado: "pendiente", Prioridad: "media"}, nil); err != nil {
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

// Un grupo se inserta entero, seguido y con las subtareas apuntando a la
// posición de su tarea principal.
func TestCrearTodoInsertaElGrupoSeguidoYConPadre(t *testing.T) {
	db, projectID, sesion := sesionDePrueba(t)

	lista, err := CrearTodo(db, projectID, sesion,
		Todo{Contenido: "Ciclo de vida del software", Estado: "pendiente", Prioridad: "media"},
		[]Todo{
			{Contenido: "Análisis", Estado: "pendiente", Prioridad: "media"},
			{Contenido: "Diseño", Estado: "pendiente", Prioridad: "media"},
		})
	if err != nil {
		t.Fatal(err)
	}
	if len(lista) != 3 {
		t.Fatalf("el grupo debería dejar tres filas: %+v", lista)
	}
	if lista[0].Posicion != 0 || lista[0].PosicionPadre != nil || lista[0].Contenido != "Ciclo de vida del software" {
		t.Fatalf("la tarea principal no quedó como raíz: %+v", lista[0])
	}
	for i, esperado := range []string{"Análisis", "Diseño"} {
		fila := lista[i+1]
		if fila.Posicion != i+1 || fila.Contenido != esperado {
			t.Fatalf("subtarea %d fuera de sitio: %+v", i, fila)
		}
		if fila.PosicionPadre == nil || *fila.PosicionPadre != 0 {
			t.Fatalf("la subtarea %q no apunta a su principal: %+v", esperado, fila)
		}
	}
}

// Un padre huérfano no se escribe: la referencia se impone en la capa de datos.
func TestReemplazarTodosRechazaUnPadreHuerfano(t *testing.T) {
	db, projectID, sesion := sesionDePrueba(t)

	huerfano := 7
	_, err := ReemplazarTodos(db, projectID, sesion, []Todo{
		{Contenido: "suelto", Estado: "pendiente", Prioridad: "media"},
		{Contenido: "colgado", Estado: "pendiente", Prioridad: "media", PosicionPadre: &huerfano},
	})
	if err == nil {
		t.Fatal("un `parent_position` huérfano debería dar error al escribir")
	}
	lista, err := ListarTodos(db, projectID, sesion)
	if err != nil {
		t.Fatal(err)
	}
	if len(lista) != 0 {
		t.Fatalf("la lista anterior debería quedar intacta: %+v", lista)
	}
}

// La jerarquía es de un nivel: un padre que ya es subtarea se rechaza.
func TestReemplazarTodosRechazaSegundoNivel(t *testing.T) {
	db, projectID, sesion := sesionDePrueba(t)

	raiz := 0
	subtarea := 1
	_, err := ReemplazarTodos(db, projectID, sesion, []Todo{
		{Contenido: "principal", Estado: "pendiente", Prioridad: "media"},
		{Contenido: "subtarea", Estado: "pendiente", Prioridad: "media", PosicionPadre: &raiz},
		{Contenido: "nieta", Estado: "pendiente", Prioridad: "media", PosicionPadre: &subtarea},
	})
	if err == nil {
		t.Fatal("un segundo nivel debería rechazarse")
	}
}

// `ReemplazarTodos` escribe `parent_position` y la lectura lo devuelve.
func TestReemplazarTodosEscribeYLeeElPadre(t *testing.T) {
	db, projectID, sesion := sesionDePrueba(t)

	raiz := 0
	lista, err := ReemplazarTodos(db, projectID, sesion, []Todo{
		{Contenido: "principal", Estado: "pendiente", Prioridad: "media"},
		{Contenido: "subtarea", Estado: "en_progreso", Prioridad: "alta", PosicionPadre: &raiz},
	})
	if err != nil {
		t.Fatal(err)
	}
	if lista[1].PosicionPadre == nil || *lista[1].PosicionPadre != 0 {
		t.Fatalf("la lectura perdió el padre: %+v", lista[1])
	}
}
