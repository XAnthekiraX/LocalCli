// rebuild.go — T-B012-02: reconstruir y re-derivar la cola desde el TODO.
//
// Fuente de verdad: SPEC-COLA-TAREAS §Flujos alternativos ("Aparece un elemento
// nuevo: la cola se re-deriva y el nuevo entra en su posición") y §Reglas ("La
// cola refleja siempre el estado real del TODO") y DECISIONS.md ("la cola se
// reconstruye al arrancar y refleja el estado real").
//
// Reconstruir se llama una vez al arrancar la ejecución; Rederrivar se llama
// antes de cada decisión de consumo. Las dos leen los archivos con `task` y no
// escriben nada: la única verdad es el TODO.
package queue

import (
	"fmt"

	"localcli/internal/task"
)

// Reconstruir arma la cola leyendo el TODO de una capa: MAIN-TASKS.md y sus
// NNN-task-*.md (task.CargarElementos). Valida el conjunto (vocabulario,
// IDs únicos y dependencias resolubles) antes de ordenarlo: un TODO inválido no
// se consume, porque no se puede saber qué está pendiente de qué.
func Reconstruir(raiz string, capa task.Capa) (*Cola, error) {
	if raiz == "" {
		return nil, fmt.Errorf("queue: sin raíz de proyecto no hay TODO que leer")
	}
	elems, err := task.CargarElementos(raiz, capa)
	if err != nil {
		return nil, fmt.Errorf("queue: leer el TODO de %s: %w", capa, err)
	}
	if err := task.ValidarConjunto(elems); err != nil {
		return nil, fmt.Errorf("queue: el TODO de %s no es válido: %w", capa, err)
	}
	orden, err := Ordenar(elems)
	if err != nil {
		return nil, err
	}
	return &Cola{raiz: raiz, capa: capa, elementos: orden}, nil
}

// Rederrivar vuelve a leer el TODO y actualiza la copia en memoria. Es la
// operación que hace que un elemento nuevo aparezca en su posición y que la
// vista coincida con el archivo aunque otra sesión lo haya cambiado.
func (c *Cola) Rederrivar() error {
	nueva, err := Reconstruir(c.raiz, c.capa)
	if err != nil {
		return err
	}
	c.elementos = nueva.elementos
	return nil
}
