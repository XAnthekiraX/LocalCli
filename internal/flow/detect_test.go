package flow

import "testing"

// TestDetectaTrabajoOrdenado — las peticiones que implican una lista ordenada
// crean un TODO (SPEC-COLA-TAREAS).
func TestDetectaTrabajoOrdenado(t *testing.T) {
	ordenadas := []string{
		"documenta el backend capa por capa",
		"hazlo paso a paso",
		"primero crea el modelo, luego el repositorio",
		"ejecuta tarea 1, tarea 2, tarea 3",
		"1. crear la entidad\n2. crear el repositorio",
		"revisa esto uno por uno",
	}
	for _, p := range ordenadas {
		if !EsTrabajoOrdenado(p) {
			t.Errorf("%q debería detectarse como trabajo ordenado", p)
		}
	}
}

// TestNoDetectaConversacionNormal — una petición sin orden ni lista no crea
// TODO: se responde en el chat.
func TestNoDetectaConversacionNormal(t *testing.T) {
	normales := []string{
		"¿qué hace el módulo de tools?",
		"explica cómo funciona el contexto",
		"corrige el error del compilador",
		"",
		"hola",
	}
	for _, p := range normales {
		if EsTrabajoOrdenado(p) {
			t.Errorf("%q no debería detectarse como trabajo ordenado", p)
		}
	}
}
