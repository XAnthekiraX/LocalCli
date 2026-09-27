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

// AbrirAtajos muestra la tabla de acciones y teclas del mapa dado, incluidas las
// secuencias con líder («<leader>m», «<leader>l») y las acciones deshabilitadas,
// que se listan marcadas (SPEC-KEYBINDS §Binding: "Cero significa deshabilitada").
func (km *KeysModal) AbrirAtajos(entradas []Atajo) {
	lineas := make([]string, 0, len(entradas))
	for _, a := range entradas {
		lineas = append(lineas, lineaDeAtajo(a))
	}
	km.Modal.Abrir("ATAJOS", lineas, "")
	km.Pie = PieAtajos
	// En una tabla de solo lectura no hay nada que resaltar: el modal entero se
	// lee y Enter no aplica nada, así que el resaltado queda sin fila
	// (SPEC-INTERFAZ §Modales, fila Atajos: "No aplica nada").
	km.Indice = -1
}
