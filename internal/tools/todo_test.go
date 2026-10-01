package tools

import (
	"errors"
	"strings"
	"testing"
)

// TestActualizarTodoValidaElVocabulario — el estado y la prioridad de cada paso
// se comprueban contra su vocabulario; un valor fuera vuelve al modelo como
// E_BAD_ARGS corregible, no ejecuta nada.
func TestActualizarTodoValidaElVocabulario(t *testing.T) {
	casos := []string{
		`{"elementos":[{"contenido":"x","estado":"hecho"}]}`,
		`{"elementos":[{"contenido":"  ","estado":"pendiente"}]}`,
		`{"elementos":[{"contenido":"x","estado":"pendiente","prioridad":"urgente"}]}`,
	}
	for _, cuerpo := range casos {
		_, err := Decodificar("actualizar_todo", []byte(cuerpo))
		if !errors.Is(err, ErrArgumentosInvalidos) {
			t.Errorf("%s: err = %v, quiero E_BAD_ARGS", cuerpo, err)
		}
	}
}

// TestActualizarTodoAceptaListasValidas — una lista válida decodifica, y una
// vacía es válida: es como se deja la lista en blanco.
func TestActualizarTodoAceptaListasValidas(t *testing.T) {
	validos := []string{
		`{"elementos":[]}`,
		`{"elementos":[{"contenido":"leer el esquema","estado":"en_progreso"}]}`,
		`{"elementos":[{"contenido":"a","estado":"completada","prioridad":"baja"}]}`,
	}
	for _, cuerpo := range validos {
		if _, err := Decodificar("actualizar_todo", []byte(cuerpo)); err != nil {
			t.Errorf("%s: no decodifica: %v", cuerpo, err)
		}
	}
}

// TestCrearTodoValidaElVocabulario — el vocabulario de `crear_todo` es el mismo
// que el de la lista: contenido no vacío, estado y prioridad dentro de su
// vocabulario. Un valor fuera es E_BAD_ARGS corregible.
func TestCrearTodoValidaElVocabulario(t *testing.T) {
	casos := []string{
		`{"contenido":"  ","estado":"pendiente"}`,
		`{"contenido":"x","estado":"hecho"}`,
		`{"contenido":"x","estado":"pendiente","prioridad":"urgente"}`,
	}
	for _, cuerpo := range casos {
		if _, err := Decodificar("crear_todo", []byte(cuerpo)); !errors.Is(err, ErrArgumentosInvalidos) {
			t.Errorf("%s: err = %v, quiero E_BAD_ARGS", cuerpo, err)
		}
	}
	// Un paso válido decodifica.
	if _, err := Decodificar("crear_todo", []byte(`{"contenido":"leer el esquema","estado":"en_progreso","prioridad":"alta"}`)); err != nil {
		t.Errorf("un paso válido no decodifica: %v", err)
	}
}

// TestCrearTodoConSegundoEnProgresoSeRechaza — añadir un segundo paso en curso
// es E_BAD_ARGS corregible, y el error nombra `actualizar_todo` como la vía
// para cambiarlo. Con la lista sin un paso en curso, el primero es válido.
func TestCrearTodoConSegundoEnProgresoSeRechaza(t *testing.T) {
	existentes := []ElementoTodo{{Contenido: "en curso", Estado: "en_progreso"}}
	err := ValidarCrearTodo(&PeticionCrearTodo{Contenido: "otro", Estado: "en_progreso"}, existentes)
	if !errors.Is(err, ErrArgumentosInvalidos) {
		t.Fatalf("err = %v, quiero E_BAD_ARGS", err)
	}
	if !strings.Contains(err.Error(), "actualizar_todo") {
		t.Errorf("el error debe nombrar `actualizar_todo`: %v", err)
	}
	if err := ValidarCrearTodo(&PeticionCrearTodo{Contenido: "primero", Estado: "en_progreso"}, nil); err != nil {
		t.Errorf("el primer paso en curso es válido: %v", err)
	}
}

// TestElEsquemaDeCrearTodoNoTieneElementos — el esquema es un objeto plano con
// `contenido`, `estado` y `prioridad`, y NO tiene campo `elementos`.
func TestElEsquemaDeCrearTodoNoTieneElementos(t *testing.T) {
	h, ok := Buscar("crear_todo")
	if !ok {
		t.Fatal("falta crear_todo")
	}
	if h.Esquema == nil || h.Esquema.Type != "object" {
		t.Fatalf("esquema = %+v, quiero un objeto plano", h.Esquema)
	}
	if _, ok := h.Esquema.Properties["elementos"]; ok {
		t.Error("el esquema de crear_todo NO puede tener `elementos`")
	}
	for _, campo := range []string{"contenido", "estado", "prioridad"} {
		if _, ok := h.Esquema.Properties[campo]; !ok {
			t.Errorf("el esquema debe exponer %q", campo)
		}
	}
}

// TestCrearTodoEsReadYNoSeConfundeConActualizar — la nueva herramienta cae en
// `read` (no abre ninguna vía de escritura al proyecto), tiene verbo `TODO` y
// tema `contenido`, y su descripción no se confunde con la de `actualizar_todo`.
func TestCrearTodoEsReadYNoSeConfundeConActualizar(t *testing.T) {
	h, ok := Buscar("crear_todo")
	if !ok {
		t.Fatal("falta crear_todo")
	}
	if h.Permiso != PermisoRead {
		t.Errorf("crear_todo es `read`, no `%s`", h.Permiso)
	}
	if h.Verbo != "TODO" || h.Tema != "contenido" || h.Unidad != "paso" {
		t.Errorf("presentación = %q/%q/%q, quiero TODO/contenido/paso", h.Verbo, h.Tema, h.Unidad)
	}
	otra, _ := Buscar("actualizar_todo")
	if h.Descripcion == otra.Descripcion {
		t.Error("las dos descripciones no pueden ser iguales")
	}
	if !strings.Contains(h.Descripcion, "final") {
		t.Errorf("la descripción debe decir que añade al final: %q", h.Descripcion)
	}
}

// TestEsquemaActualizarTodoAnidaLosElementos — el esquema que ve el modelo
// describe la lista como un array de objetos con sus campos; no un `object`
// vacío.
func TestEsquemaActualizarTodoAnidaLosElementos(t *testing.T) {
	h, ok := Buscar("actualizar_todo")
	if !ok {
		t.Fatal("falta actualizar_todo")
	}
	prop, ok := h.Esquema.Properties["elementos"]
	if !ok {
		t.Fatal("el esquema debe exponer `elementos`")
	}
	if prop.Type != "array" || prop.Items == nil || prop.Items.Type != "object" {
		t.Fatalf("elementos debe ser un array de objetos: %+v", prop)
	}
	for _, campo := range []string{"contenido", "estado", "prioridad"} {
		if _, ok := prop.Items.Properties[campo]; !ok {
			t.Errorf("el elemento debe exponer %q", campo)
		}
	}
}
