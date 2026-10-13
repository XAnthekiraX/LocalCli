package modals

import (
	"fmt"

	"localcli/internal/tui/render"
)

// Fila es una entrada del listado de atajos: el grupo al que pertenece, la tecla
// —o varias, ya unidas— y lo que hace. El modal no conoce el mapa de teclas: el
// `app` compone las filas del mapa vigente y aquí solo se agrupan y se alinean.
type Fila struct {
	Grupo  string
	Tecla  string
	Accion string
}

// Atajos es el modal de atajos (`Ctrl+P`): la tabla de acciones con su tecla,
// agrupada por categorías. Es de solo lectura —no aplica nada— y la tabla sale
// del mapa vigente, así que si el usuario reasignó algo, lo que se ve es lo que
// hay.
type Atajos struct {
	Modal
}

// Abrir compone la tabla con las filas dadas, en su orden: cada grupo encabeza su
// sección y la columna de teclas va alineada al ancho de la más larga. El
// resaltado queda fuera —-1— porque aquí no hay nada que elegir: `Enter` no
// aplica nada (SPEC-INTERFAZ §Modales, fila Atajos).
func (am *Atajos) Abrir(filas []Fila) {
	anchoTeclas := len("(deshabilitada)")
	for _, f := range filas {
		if w := len([]rune(f.Tecla)); w > anchoTeclas {
			anchoTeclas = w
		}
	}
	var lineas []string
	grupo := ""
	for _, f := range filas {
		if f.Grupo != grupo {
			if len(lineas) > 0 {
				lineas = append(lineas, "")
			}
			grupo = f.Grupo
			lineas = append(lineas, render.EstiloEtiqueta.Render(grupo))
		}
		lineas = append(lineas, fmt.Sprintf("%-*s  %s", anchoTeclas, f.Tecla, f.Accion))
	}
	am.Pie = PieAtajos
	am.Modal.Abrir("ATAJOS", lineas, "")
	am.Indice = -1
}

// Mover no hace nada: en una tabla de solo lectura no hay fila resaltada que
// recorrer. Sin este método, el `↑`/`↓` que comparten los modales encendería un
// resaltado que en este modal no significa nada.
func (am *Atajos) Mover(int) {}
