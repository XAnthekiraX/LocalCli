package session

import (
	"context"
	"encoding/json"
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

// El caso aprobado: una petición con fases se registra como una tarea principal
// con sus subtareas, y el checklist las anida bajo el título.
func TestCrearTodoConTituloYSubtareasAnidaElChecklist(t *testing.T) {
	o, sesion := orquestadorDeTodo(t)

	fases := []string{
		"Análisis de requisitos", "Diseño", "Implementación", "Pruebas",
		"Despliegue", "Mantenimiento", "Retirada",
	}
	peticion := tools.PeticionCrearTodo{Titulo: "Ciclo de vida del software"}
	for _, fase := range fases {
		peticion.Subtareas = append(peticion.Subtareas, tools.Subtarea{Contenido: fase, Estado: "pendiente"})
	}

	res, err := o.crearTodo(context.Background(), peticion, tools.Contexto{SesionID: sesion})
	if err != nil || res.Error != "" {
		t.Fatalf("crear_todo con subtareas: %+v %v", res, err)
	}

	var quiero strings.Builder
	quiero.WriteString("[ ] Ciclo de vida del software")
	for _, fase := range fases {
		quiero.WriteString("\n  [ ] ")
		quiero.WriteString(fase)
	}
	if res.Salida != quiero.String() {
		t.Fatalf("checklist = %q, quiero %q", res.Salida, quiero.String())
	}

	lista, err := store.ListarTodos(o.db, o.projectID, sesion)
	if err != nil {
		t.Fatal(err)
	}
	if len(lista) != 8 {
		t.Fatalf("filas = %d, quiero la principal más 7 subtareas", len(lista))
	}
	for i, paso := range lista[1:] {
		if paso.PosicionPadre == nil || *paso.PosicionPadre != 0 {
			t.Fatalf("la subtarea %d no apunta a la principal: %+v", i, paso)
		}
	}
}

// Los dos caminos de `crear_todo` se excluyen: mezclarlos, o no usar ninguno,
// devuelve un error corregible y no escribe nada.
func TestCrearTodoExcluyeLasDosFormas(t *testing.T) {
	o, sesion := orquestadorDeTodo(t)

	casos := []tools.PeticionCrearTodo{
		{Contenido: "suelto", Estado: "pendiente", Titulo: "principal", Subtareas: []tools.Subtarea{{Contenido: "fase", Estado: "pendiente"}}},
		{Contenido: "suelto", Estado: "pendiente", Subtareas: []tools.Subtarea{{Contenido: "fase", Estado: "pendiente"}}},
		{Titulo: "principal"},
		{},
	}
	for _, peticion := range casos {
		res, err := o.crearTodo(context.Background(), peticion, tools.Contexto{SesionID: sesion})
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasPrefix(res.Error, tools.CodBadArgs) {
			t.Fatalf("la llamada %+v debería ser E_BAD_ARGS: %q", peticion, res.Error)
		}
	}

	lista, err := store.ListarTodos(o.db, o.projectID, sesion)
	if err != nil {
		t.Fatal(err)
	}
	if len(lista) != 0 {
		t.Fatalf("una llamada inválida no debería escribir: %+v", lista)
	}
}

// El `en_progreso` único se cuenta sobre toda la lista, subtareas incluidas.
func TestCrearTodoCuentaEnProgresoEnLasSubtareas(t *testing.T) {
	o, sesion := orquestadorDeTodo(t)

	res, err := o.crearTodo(context.Background(), tools.PeticionCrearTodo{
		Titulo: "principal",
		Subtareas: []tools.Subtarea{
			{Contenido: "una", Estado: "en_progreso"},
			{Contenido: "otra", Estado: "en_progreso"},
		},
	}, tools.Contexto{SesionID: sesion})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(res.Error, tools.CodBadArgs) {
		t.Fatalf("dos subtareas en curso deberían rechazarse: %q", res.Error)
	}
}

// `actualizar_todo` admite subtareas por elemento y emite el `padre` de cada
// paso en el evento que repinta el panel.
func TestActualizarTodoConSubtareasEmiteElPadre(t *testing.T) {
	o, sesion := orquestadorDeTodo(t)
	canal, cerrar := o.Suscribir()
	defer cerrar()

	res, err := o.actualizarTodo(context.Background(), tools.PeticionActualizarTodo{Elementos: []tools.ElementoTodo{
		{Contenido: "principal", Estado: "pendiente", Subtareas: []tools.Subtarea{
			{Contenido: "fase", Estado: "pendiente"},
		}},
		{Contenido: "suelto", Estado: "pendiente"},
	}}, tools.Contexto{SesionID: sesion})
	if err != nil || res.Error != "" {
		t.Fatalf("actualizar_todo con subtareas: %+v %v", res, err)
	}
	if res.Salida != "[ ] principal\n  [ ] fase\n[ ] suelto" {
		t.Fatalf("checklist = %q", res.Salida)
	}

	var elementos []struct {
		Contenido string `json:"contenido"`
		Padre     *int   `json:"padre"`
	}
	encontrado := false
	for _, e := range drenar(canal) {
		if e.Tipo == tui.EventoTodoActualizada {
			encontrado = true
			if err := json.Unmarshal([]byte(e.Datos["elementos"]), &elementos); err != nil {
				t.Fatalf("el evento no lleva una lista legible: %v", err)
			}
		}
	}
	if !encontrado || len(elementos) != 3 {
		t.Fatalf("el evento debería llevar los tres pasos: %+v", elementos)
	}
	if elementos[0].Padre != nil {
		t.Fatalf("la tarea principal debería ser raíz: %+v", elementos[0])
	}
	if elementos[1].Padre == nil || *elementos[1].Padre != 0 {
		t.Fatalf("la subtarea debería apuntar a la principal: %+v", elementos[1])
	}
	if elementos[2].Padre != nil {
		t.Fatalf("el paso suelto debería ser raíz: %+v", elementos[2])
	}
}
