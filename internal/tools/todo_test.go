package tools

import (
	"errors"
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
