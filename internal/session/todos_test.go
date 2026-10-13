package session

import (
	"context"
	"strings"
	"testing"

	"localcli/internal/llm/openai"
	"localcli/internal/store"
	"localcli/internal/tools"
	"localcli/internal/tui"
)

func orquestadorDeTodo(t *testing.T) (*Orquestador, string) {
	t.Helper()
	db, limpiar, _ := store.CrearDBTemporal()
	t.Cleanup(func() { _ = limpiar() })
	cwd := t.TempDir()
	projectID, _ := store.ResolucionProyecto(db, cwd)
	o := Nuevo(db, projectID, cwd, agentesDePrueba(t), openai.NuevoCliente("http://127.0.0.1:1"), "m", 0, nil, nil)
	sesion, err := o.Crear()
	if err != nil {
		t.Fatal(err)
	}
	return o, sesion.ID
}

func TestCrearTodoEmiteLaListaYLaDevuelve(t *testing.T) {
	o, sesion := orquestadorDeTodo(t)
	canal, cerrar := o.Suscribir()
	defer cerrar()

	res, err := o.crearTodo(context.Background(), tools.PeticionCrearTodo{Contenido: "escribir", Estado: "pendiente"}, tools.Contexto{SesionID: sesion})
	if err != nil || res.Error != "" {
		t.Fatalf("crear_todo: %+v %v", res, err)
	}
	if res.Salida != "[ ] escribir" {
		t.Fatalf("checklist = %q", res.Salida)
	}

	encontrado := false
	for _, e := range drenar(canal) {
		if e.Tipo == tui.EventoTodoActualizada {
			encontrado = true
			if !strings.Contains(e.Datos["elementos"], "escribir") {
				t.Fatalf("el evento no lleva la lista: %s", e.Datos["elementos"])
			}
		}
	}
	if !encontrado {
		t.Fatal("debería emitirse `todo_actualizada`")
	}
}

func TestCrearTodoRechazaSegundoEnProgreso(t *testing.T) {
	o, sesion := orquestadorDeTodo(t)

	if _, err := o.crearTodo(context.Background(), tools.PeticionCrearTodo{Contenido: "uno", Estado: "en_progreso"}, tools.Contexto{SesionID: sesion}); err != nil {
		t.Fatal(err)
	}
	res, err := o.crearTodo(context.Background(), tools.PeticionCrearTodo{Contenido: "dos", Estado: "en_progreso"}, tools.Contexto{SesionID: sesion})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(res.Error, tools.CodBadArgs) {
		t.Fatalf("un segundo `en_progreso` debería rechazarse: %q", res.Error)
	}
	if !strings.Contains(res.Error, "actualizar_todo") {
		t.Fatalf("el mensaje debería nombrar `actualizar_todo`: %q", res.Error)
	}
}

func TestActualizarTodoDevuelveElChecklistConGlifos(t *testing.T) {
	o, sesion := orquestadorDeTodo(t)

	res, err := o.actualizarTodo(context.Background(), tools.PeticionActualizarTodo{Elementos: []tools.ElementoTodo{
		{Contenido: "hecho", Estado: "completada"},
		{Contenido: "en curso", Estado: "en_progreso"},
		{Contenido: "pendiente", Estado: "pendiente"},
	}}, tools.Contexto{SesionID: sesion})
	if err != nil || res.Error != "" {
		t.Fatalf("actualizar_todo: %+v %v", res, err)
	}
	quiero := "[✓] hecho\n[•] en curso\n[ ] pendiente"
	if res.Salida != quiero {
		t.Fatalf("checklist = %q, quiero %q", res.Salida, quiero)
	}
}
