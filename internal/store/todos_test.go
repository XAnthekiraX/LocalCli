package store

import "testing"

// TestReemplazarYLeerTodos — la lista se reescribe entera: reemplazar sustituye,
// no acumula, y una lista vacía la deja en blanco. El orden es la `position`.
func TestReemplazarYLeerTodos(t *testing.T) {
	db := abrirBaseTemporal(t)
	ses, err := CrearSesion(db, "s", "")
	if err != nil {
		t.Fatal(err)
	}

	if err := ReemplazarTodos(db, ses.ID, []Todo{
		{Contenido: "primero", Estado: "en_progreso", Prioridad: "alta"},
		{Contenido: "segundo", Estado: "pendiente"},
	}); err != nil {
		t.Fatal(err)
	}
	got, err := LeerTodos(db, ses.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("leídos %d pasos, quiero 2", len(got))
	}
	if got[0].Contenido != "primero" || got[0].Estado != "en_progreso" || got[0].Prioridad != "alta" {
		t.Errorf("primer paso = %+v", got[0])
	}
	if got[1].Prioridad != prioridadPorDefecto {
		t.Errorf("la prioridad por defecto debe ser %q: %q", prioridadPorDefecto, got[1].Prioridad)
	}

	// Segunda escritura: sustituye, no acumula.
	if err := ReemplazarTodos(db, ses.ID, []Todo{{Contenido: "solo", Estado: "pendiente"}}); err != nil {
		t.Fatal(err)
	}
	got, _ = LeerTodos(db, ses.ID)
	if len(got) != 1 || got[0].Contenido != "solo" {
		t.Fatalf("tras reemplazar = %+v", got)
	}

	// Lista vacía: la deja en blanco.
	if err := ReemplazarTodos(db, ses.ID, nil); err != nil {
		t.Fatal(err)
	}
	if got, _ = LeerTodos(db, ses.ID); len(got) != 0 {
		t.Errorf("la lista vacía debe dejar cero pasos: %+v", got)
	}
}

// TestAgregarTodoNoPierdeLosAnteriores — `crear_todo` inserta una fila al final
// sin borrar nada: los pasos que ya estaban siguen igual y el nuevo es el
// último. Es lo que lo distingue de `actualizar_todo`, que reemplaza.
func TestAgregarTodoNoPierdeLosAnteriores(t *testing.T) {
	db := abrirBaseTemporal(t)
	ses, err := CrearSesion(db, "s", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := ReemplazarTodos(db, ses.ID, []Todo{
		{Contenido: "primero", Estado: "completada"},
		{Contenido: "segundo", Estado: "en_progreso"},
		{Contenido: "tercero", Estado: "pendiente"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := AgregarTodo(db, ses.ID, Todo{Contenido: "cuarto", Estado: "pendiente", Prioridad: "alta"}); err != nil {
		t.Fatal(err)
	}
	got, err := LeerTodos(db, ses.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 4 {
		t.Fatalf("tras agregar hay %d pasos, quiero 4: %+v", len(got), got)
	}
	quiere := []string{"primero", "segundo", "tercero", "cuarto"}
	for i, c := range quiere {
		if got[i].Contenido != c {
			t.Errorf("paso %d = %q, quiero %q (los anteriores no se tocan)", i, got[i].Contenido, c)
		}
	}
	if got[0].Estado != "completada" || got[1].Estado != "en_progreso" {
		t.Errorf("agregar no debe cambiar los estados anteriores: %+v", got)
	}
	if got[3].Prioridad != "alta" {
		t.Errorf("la prioridad del nuevo paso se guarda: %+v", got[3])
	}
}

// TestTodosCaenEnCascadaConLaSesion — borrar la sesión se lleva su lista.
func TestTodosCaenEnCascadaConLaSesion(t *testing.T) {
	db := abrirBaseTemporal(t)
	ses, err := CrearSesion(db, "s", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := ReemplazarTodos(db, ses.ID, []Todo{{Contenido: "x", Estado: "pendiente"}}); err != nil {
		t.Fatal(err)
	}
	if err := BorrarSesion(db, ses.ID); err != nil {
		t.Fatal(err)
	}
	got, err := LeerTodos(db, ses.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("la lista debe caer con la sesión: %+v", got)
	}
}

// TestTodoConEstadoInvalidoSeRechaza — el CHECK de la base es la última red: un
// estado fuera del vocabulario no llega a guardarse (la transacción revierte).
func TestTodoConEstadoInvalidoSeRechaza(t *testing.T) {
	db := abrirBaseTemporal(t)
	ses, err := CrearSesion(db, "s", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := ReemplazarTodos(db, ses.ID, []Todo{{Contenido: "x", Estado: "hecho"}}); err == nil {
		t.Fatal("un estado fuera del vocabulario debe rechazarse")
	}
	// La transacción revirtió: no quedó nada.
	if got, _ := LeerTodos(db, ses.ID); len(got) != 0 {
		t.Errorf("un reemplazo fallido no deja filas: %+v", got)
	}
}
