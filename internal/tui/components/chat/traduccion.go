package chat

// Traducción entre las líneas del hilo (coordenadas de contenido) y las filas de
// la ventana que pinta `Vista` (coordenadas de pantalla). La selección del chat
// se rastrea en contenido, así que necesita ir en los dos sentidos.

// LineasContenido devuelve las líneas del hilo de la última pintura.
func (m Model) LineasContenido() []string { return m.hiloLineas }

// TotalContenido es cuántas líneas tiene el hilo.
func (m Model) TotalContenido() int { return len(m.hiloLineas) }

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
