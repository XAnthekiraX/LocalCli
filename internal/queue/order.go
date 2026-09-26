// order.go — T-B012-03: orden por dependencias con desempate por prefijo NNN.
//
// Fuente de verdad: SPEC-COLA-TAREAS ("Los elementos se toman en el orden que
// marcan sus dependencias", "El orden del TODO es el orden de ejecución, y se
// respeta") y DECISIONS.md [28] ("orden topológico por `depende_de`, con el
// prefijo NNN del nombre como desempate").
//
// El algoritmo vive en `task` (T-B004-06), que ya lo implementa de forma estable
// —Kahn con desempate por NNN y por posición en el archivo— y ya resuelve los
// rangos `T-B002..T-B014` expandiéndolos por aristas reales. Duplicarlo aquí
// sería tener dos órdenes que pueden divergir; lo que aporta `queue` es
// traducir el fallo: un TODO con ciclo no se puede ordenar, y ese es el único
// motivo por el que `Ordenar` devuelve error.
package queue

import (
	"fmt"

	"localcli/internal/task"
)

// Ordenar devuelve los elementos en orden de ejecución. Error si el TODO tiene
// un ciclo de dependencias o una dependencia que no existe.
func Ordenar(elems []task.Elemento) ([]task.Elemento, error) {
	orden, err := task.OrdenarTopologico(elems)
	if err != nil {
		return nil, fmt.Errorf("queue: no se puede ordenar el TODO: %w", err)
	}
	return orden, nil
}
