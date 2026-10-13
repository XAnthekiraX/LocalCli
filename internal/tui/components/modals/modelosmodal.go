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
}

// Abrir deja el modal pidiendo la lista. Cada apertura es una lectura nueva: no
// se guarda la lista entre aperturas para que lo que se vea sea lo que hay ahora.
func (mm *Modelos) Abrir(actual string) {
	mm.Lista = nil
	mm.Actual = actual
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
		mm.Modal.Abrir("MODELOS", nil, AvisoSinModelos)
		return
	}
	mm.Lista = lista
	lineas := make([]string, 0, len(lista))
	for _, m := range lista {
		lineas = append(lineas, lineaDeModelo(m))
	}
	mm.Modal.Abrir("MODELOS", lineas, "")
	mm.Indice = mm.IndiceDe(mm.Actual)
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
