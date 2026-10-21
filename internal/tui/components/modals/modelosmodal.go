package modals

// Modelo es un modelo local tal como lo pinta el modal: su nombre, el motor que
// lo sirve y si declara capacidad de herramientas. El dato viene resuelto del
// puerto; el modal solo lo pinta.
type Modelo struct {
	Nombre string
	// Motor rotula la fila (`motor / modelo`), porque el mismo nombre no
	// significa lo mismo en runtimes distintos.
	Motor string
	// SinHerramientas marca los modelos que no declaran usar herramientas: se
	// sabe antes de elegirlos, que es cuando importa.
	SinHerramientas bool
}

// Modelos es el modal de modelos (`Ctrl+X m`): la lista del motor de la sesión,
// cargada al abrir, y el modelo en uso, en el que arranca el resaltado.
type Modelos struct {
	Modal
	// Lista es la lista real, en el mismo orden que `Lineas`: la elegida se
	// traduce con el índice resaltado.
	Lista []Modelo
	// Actual es el nombre del modelo en uso: el resaltado arranca en él para no
	// obligar a recorrer la lista desde el principio.
	Actual string
	// confirmando es el nombre del modelo cuya eliminación espera la respuesta
	// `s`/`n`. Vacío = no hay nada pendiente de confirmar.
	confirmando string
}

// Abrir deja el modal pidiendo la lista. Cada apertura es una lectura nueva: no
// se guarda la lista entre aperturas para que lo que se vea sea lo que hay ahora.
func (mm *Modelos) Abrir(actual string) {
	mm.Lista = nil
	mm.Actual = actual
	mm.confirmando = ""
	mm.Pie = PieModal
	mm.Modal.Abrir("MODELOS", nil, AvisoCargando)
}

// Fijar rellena el modal con la lista que llegó del puerto. Una lectura fallida
// o vacía no bloquea ni propaga: queda el aviso «sin modelos» y el modal se
// cierra con `esc`. Si el modal ya no está abierto, la lista se descarta: llegó
// tarde.
func (mm *Modelos) Fijar(lista []Modelo, err error) {
	if !mm.Abierto {
		return
	}
	if err != nil || len(lista) == 0 {
		mm.Lista = nil
		mm.confirmando = ""
		mm.Modal.Abrir("MODELOS", nil, AvisoSinModelos)
		return
	}
	mm.Lista = lista
	mm.repintar(mm.IndiceDe(mm.Actual))
}

// PedirEliminar resuelve `Ctrl+D` sobre el modelo resaltado: deja la pregunta
// «¿seguro que deseas eliminar `<modelo>`? (s/n)» y no entrega nada. La respuesta
// (`s` o `n`) la resuelve quien tiene el teclado, no el modal: el modal no borra
// ni decide si un modelo se puede borrar (SPEC-INTERFAZ §Modales).
func (mm *Modelos) PedirEliminar() {
	modelo, ok := mm.Elegido()
	if !ok {
		return
	}
	mm.confirmando = modelo.Nombre
	mm.repintar(mm.Indice)
}

// Confirmando dice si hay una eliminación esperando respuesta: mientras la haya,
// `s` concede el borrado, `n` y `esc` lo cancelan, y el resaltado no se mueve.
func (mm *Modelos) Confirmando() bool { return mm.confirmando != "" }

// CancelarConfirmacion descarta la eliminación pendiente y deja la lista como
// estaba: la pregunta no cambia nada.
func (mm *Modelos) CancelarConfirmacion() {
	if mm.confirmando == "" {
		return
	}
	mm.confirmando = ""
	mm.repintar(mm.Indice)
}

// Avisar deja un mensaje sobre la lista sin perderla: es el error del puerto
// —por ejemplo, el modelo en uso— y se lee sin cerrar el modal.
func (mm *Modelos) Avisar(texto string) {
	if !mm.Abierto {
		return
	}
	mm.Aviso = texto
}

// Colocar deja el resaltado en el índice pedido, recortado a la lista: tras
// eliminar, el hueco lo ocupa el siguiente —o el último, si era el final—.
func (mm *Modelos) Colocar(indice int) {
	if len(mm.Lista) == 0 {
		return
	}
	if indice < 0 {
		indice = 0
	}
	if indice >= len(mm.Lista) {
		indice = len(mm.Lista) - 1
	}
	mm.Indice = indice
}

// Mover mueve el resaltado, salvo mientras hay una eliminación esperando
// respuesta: la pregunta nombra un modelo concreto y `s` borra ese, no el que
// quede debajo si el usuario sigue recorriendo la lista.
func (mm *Modelos) Mover(delta int) {
	if mm.confirmando != "" {
		return
	}
	mm.Modal.Mover(delta)
}

// repintar reconstruye las filas y deja el resaltado donde se le dice. La
// pregunta de confirmación, si la hay, va en una línea propia antes del pie.
func (mm *Modelos) repintar(indice int) {
	if !mm.Abierto {
		return
	}
	if len(mm.Lista) == 0 {
		mm.confirmando = ""
		mm.Modal.Abrir("MODELOS", nil, AvisoSinModelos)
		return
	}
	lineas := make([]string, 0, len(mm.Lista)+2)
	for _, m := range mm.Lista {
		lineas = append(lineas, lineaDeModelo(m))
	}
	if mm.confirmando != "" {
		lineas = append(lineas, "", "¿seguro que deseas eliminar `"+mm.confirmando+"`? (s/n)")
	}
	mm.Modal.Abrir("MODELOS", lineas, "")
	mm.Indice = indice
}

// Elegido devuelve el modelo resaltado. Sin lista —«cargando…» o «sin
// modelos»— no hay nada que elegir.
func (mm *Modelos) Elegido() (Modelo, bool) {
	i, ok := mm.Elegida()
	if !ok {
		return Modelo{}, false
	}
	return mm.Lista[i], true
}

// IndiceDe es la posición de un modelo en la lista, o 0 si no está: al abrir, el
// resaltado arranca en el modelo en uso.
func (mm *Modelos) IndiceDe(nombre string) int {
	for i, m := range mm.Lista {
		if m.Nombre == nombre {
			return i
		}
	}
	return 0
}

// lineaDeModelo compone la fila del modelo: rotulada con su motor —para no
// confundir el mismo nombre entre motores— y con la marca de lo que no declara.
func lineaDeModelo(m Modelo) string {
	fila := m.Nombre
	if m.Motor != "" {
		fila = m.Motor + " / " + m.Nombre
	}
	if m.SinHerramientas {
		fila += "  (sin herramientas)"
	}
	return fila
}
