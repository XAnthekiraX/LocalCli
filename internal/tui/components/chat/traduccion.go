package chat

// Traducción entre las líneas del hilo (coordenadas de contenido) y las filas de
// la ventana que pinta `Vista` (coordenadas de pantalla). La selección del chat
// se rastrea en contenido, así que necesita ir en los dos sentidos.

// LineasContenido devuelve las líneas del hilo de la última pintura.
func (m Model) LineasContenido() []string { return m.hiloLineas }

// TotalContenido es cuántas líneas tiene el hilo.
func (m Model) TotalContenido() int { return len(m.hiloLineas) }

// SegmentoEnLinea devuelve el segmento cerrado que pinta la línea de contenido
// dada, si esa línea cae dentro del globo de un segmento. Las líneas de
// separación, las sueltas y el turno en vivo no pertenecen a ningún globo: ahí
// devuelve ok=false. Es lo que permite abrir el menú de un globo al hacer clic.
func (m *Model) SegmentoEnLinea(contenido int) (*Segmento, bool) {
	if contenido < 0 || contenido >= len(m.lineasPieza) {
		return nil, false
	}
	idx := m.lineasPieza[contenido]
	if idx < 0 || idx >= len(m.piezas) {
		return nil, false
	}
	seg, ok := m.piezas[idx].(*Segmento)
	return seg, ok
}

// SuspenderSeguimiento deja la vista anclada donde está mientras dura el arrastre
// y devuelve si seguía el final, para restaurarlo al soltar.
func (m *Model) SuspenderSeguimiento() bool {
	m.arrastrando = true
	return m.scrolleado
}

// ReanudarSeguimiento vuelve a dejar el seguimiento como estaba antes del arrastre.
func (m *Model) ReanudarSeguimiento(antes bool) {
	m.arrastrando = false
	m.scrolleado = antes
}

// ContenidoBajoVentana traduce una fila de la ventana a la línea del hilo que hay
// ahí, para anclar y extender el arrastre. Una fila de aviso cae en la línea
// visible más próxima.
func (m Model) ContenidoBajoVentana(ventana int) (int, bool) {
	n := m.TotalContenido()
	if n == 0 {
		return 0, false
	}
	if m.altoUltimo <= 0 || n <= m.altoUltimo {
		return acotar(ventana, 0, n-1), true
	}
	cuerpo := max(m.altoUltimo-2, 1)
	switch {
	case ventana < 1:
		return m.offset, true
	case ventana > cuerpo:
		return min(m.offset+cuerpo-1, n-1), true
	default:
		return min(m.offset+ventana-1, n-1), true
	}
}

// VentanaDeContenido traduce una línea del hilo a la fila de la ventana donde se
// pinta, si está a la vista. Fuera del viewport devuelve ok=false: solo se realza
// lo visible.
func (m Model) VentanaDeContenido(contenido int) (int, bool) {
	n := m.TotalContenido()
	if contenido < 0 || contenido >= n || m.altoUltimo <= 0 {
		return 0, false
	}
	if n <= m.altoUltimo {
		return contenido, true
	}
	cuerpo := max(m.altoUltimo-2, 1)
	fila := contenido - m.offset + 1
	if fila < 1 || fila > cuerpo {
		return 0, false
	}
	return fila, true
}

// BotonEnVentana busca el botón de copiar que cae en una celda de la ventana: la
// fila traducida a línea del hilo y la columna, ya medida desde el borde de la
// zona del chat. Devuelve el botón y si hay uno en esa celda; es lo que dispara la
// copia al hacer clic (SPEC-VISTA-CODIGO §Reglas de negocio).
func (m Model) BotonEnVentana(ventana, columna int) (BotonCopia, bool) {
	contenido, ok := m.ContenidoBajoVentana(ventana)
	if !ok {
		return BotonCopia{}, false
	}
	for _, b := range m.botones {
		if b.Linea == contenido && columna >= b.Col && columna < b.Col+b.Ancho {
			return b, true
		}
	}
	return BotonCopia{}, false
}
