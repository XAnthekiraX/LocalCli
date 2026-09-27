// keysmodal.go — T-F014: el modal de atajos (`Ctrl+P`).
//
// Fuente de verdad: SPEC-INTERFAZ §Modales ("Atajos | Ctrl+P | Tabla de los
// atajos existentes: acción y tecla(s) de cada uno, incluidas las secuencias con
// líder | No aplica nada: es solo lectura; Esc lo cierra") y §Reglas ("No hay
// paleta de comandos ejecutables: Ctrl+P abre la lista de atajos"), SPEC-KEYBINDS
// §Acción ("La ayuda clásica (?) queda sustituida por el modal de atajos
// (command_palette)") y DOMAIN §1 (`modals`: keysmodal, "lista de atajos
// existentes, solo lectura, Ctrl+P").
//
// Comparte con los otros dos modales la mecánica de `Modal` (modelsmodal.go) y
// se queda en solo lectura: no aplica nada, no navega y no reasigna —reasignar es
// cosa del mapa de teclas, no de un modal (DOMAIN §1 `keys`: "Un atajo no cambia
// ninguna regla de permiso")—.
package tui

import (
	"fmt"
	"strings"
)

// KeysModal es el modal de atajos: la tabla de acciones con su tecla, tal como
// está el keymap ahora mismo. Si el usuario reasignó algo, lo que se ve es lo que
// hay: la tabla se genera del mapa vigente cada vez que se abre (INTERFACES §4:
// "el cambio se guarda sin reiniciar la aplicación").
type KeysModal struct {
	Modal
}

// PieAtajos es el pie de este modal: no hay nada que aplicar, así que solo se
// anuncia cómo se sale.
const PieAtajos = "(solo lectura · esc cerrar)"

// ordenDeCategorias fija el orden de los grupos del listado: de lo general a lo
// particular, para que la tabla se lea de un vistazo.
func ordenDeCategorias() []string {
	return []string{"General", "Chat", "Vista", "Modales", "Aprobaciones", "Entrada"}
}

// categoriaDeAtajo reparte cada acción en su grupo. Un grupo nuevo se añade aquí
// y en ordenDeCategorias.
func categoriaDeAtajo(a Accion) string {
	switch a {
	case AccionSalir, AccionAyuda, AccionModalModelos, AccionSelector:
		return "General"
	case AccionChatSubir, AccionChatBajar, AccionChatPaginaArriba, AccionChatPaginaAbajo:
		return "Chat"
	case AccionPanel, AccionRazonamiento, AccionAprobaciones, AccionCancelar, AccionCiclarAgente, AccionSesionNueva, AccionPausar:
		return "Vista"
	case AccionCerrarSelector, AccionEliminarSesion, AccionSubir, AccionBajar:
		return "Modales"
	case AccionAprobar, AccionDeclinar:
		return "Aprobaciones"
	case AccionEnviar:
		return "Entrada"
	}
	return "Otras"
}

// AbrirAtajos muestra la tabla de acciones y teclas del mapa dado, agrupada por
// categorías y con las teclas alineadas en una columna. Incluye las secuencias
// con líder («<leader>m», «<leader>l») y las acciones deshabilitadas, que se
// listan marcadas (SPEC-KEYBINDS §Binding: "Cero significa deshabilitada").
func (km *KeysModal) AbrirAtajos(entradas []Atajo) {
	porCategoria := map[string][]Atajo{}
	for _, a := range entradas {
		c := categoriaDeAtajo(a.Accion)
		porCategoria[c] = append(porCategoria[c], a)
	}
	// Ancho común de la columna de teclas: la más larga de todo el mapa.
	anchoTeclas := len("(deshabilitada)")
	for _, a := range entradas {
		if w := len([]rune(teclasDeAtajo(a))); w > anchoTeclas {
			anchoTeclas = w
		}
	}

	var lineas []string
	for _, grupo := range ordenDeCategorias() {
		ats := porCategoria[grupo]
		if len(ats) == 0 {
			continue
		}
		if len(lineas) > 0 {
			lineas = append(lineas, "")
		}
		lineas = append(lineas, estiloEtiqueta.Render(grupo))
		for _, a := range ats {
			lineas = append(lineas, filaDeAtajo(a, anchoTeclas))
		}
	}
	km.Modal.Abrir("ATAJOS", lineas, "")
	km.Pie = PieAtajos
	// En una tabla de solo lectura no hay nada que resaltar: el modal entero se
	// lee y Enter no aplica nada, así que el resaltado queda sin fila
	// (SPEC-INTERFAZ §Modales, fila Atajos: "No aplica nada").
	km.Indice = -1
}

// filaDeAtajo compone la fila «tecla(s) + descripción» con la columna de teclas
// alineada al ancho dado.
func filaDeAtajo(a Atajo, anchoTeclas int) string {
	return fmt.Sprintf("%-*s  %s", anchoTeclas, teclasDeAtajo(a), a.Descripcion)
}

// teclasDeAtajo junta los literales de una acción; sin ninguno, la acción está
// deshabilitada.
func teclasDeAtajo(a Atajo) string {
	literales := make([]string, 0, len(a.Secuencias))
	for _, sec := range a.Secuencias {
		literales = append(literales, sec.Describir())
	}
	if len(literales) == 0 {
		return "(deshabilitada)"
	}
	return strings.Join(literales, ", ")
}
