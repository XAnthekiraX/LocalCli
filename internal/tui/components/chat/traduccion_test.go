package chat

import "testing"

// hiloLargo deja un historial que desborda la ventana, para probar el scroll.
func hiloLargo(n int) *Model {
	m := &Model{}
	m.AñadirUsuario("pregunta inicial")
	for i := 0; i < n; i++ {
		m.AñadirSegmento(Segmento{Hablante: Agente, Texto: "linea de relleno"})
	}
	return m
}

// La traducción contenido↔ventana es de ida y vuelta y acota fuera de rango.
func TestTraduccionContenidoVentana(t *testing.T) {
	m := hiloLargo(40)
	m.Vista(60, 12)
	if m.TotalContenido() <= 12 {
		t.Fatalf("el hilo debería desbordar la ventana: %d líneas", m.TotalContenido())
	}
	// La fila 1 es la primera de contenido; la 0 es el aviso de arriba.
	c, ok := m.ContenidoBajoVentana(1)
	if !ok || c != m.offset {
		t.Fatalf("la fila 1 = contenido %d (%v), quiero %d", c, ok, m.offset)
	}
	if v, ok := m.VentanaDeContenido(c); !ok || v != 1 {
		t.Fatalf("ida y vuelta: contenido %d → ventana %d (%v)", c, v, ok)
	}
	// Una fila de aviso cae en la línea visible más próxima.
	if c, _ := m.ContenidoBajoVentana(0); c != m.offset {
		t.Fatalf("la fila 0 debería caer en la primera visible: %d", c)
	}
	// Fuera del viewport no se realza.
	if _, ok := m.VentanaDeContenido(m.offset - 1); ok {
		t.Fatal("una línea por encima del viewport no debería mapear a una fila")
	}
	if _, ok := m.VentanaDeContenido(m.TotalContenido()); ok {
		t.Fatal("una línea inexistente no debería mapear a una fila")
	}
}

// Sin desbordar, la fila de ventana es la propia línea del hilo.
func TestTraduccionSinDesbordar(t *testing.T) {
	m := &Model{}
	m.AñadirSegmento(Segmento{Hablante: Usuario, Texto: "hola"})
	m.Vista(60, 20)
	if c, ok := m.ContenidoBajoVentana(1); !ok || c != 1 {
		t.Fatalf("fila 1 → contenido %d (%v)", c, ok)
	}
	if v, ok := m.VentanaDeContenido(1); !ok || v != 1 {
		t.Fatalf("contenido 1 → ventana %d (%v)", v, ok)
	}
}

// El arrastre suspende el seguimiento: un token nuevo no mueve la vista.
func TestSuspenderSeguimientoCongelaLaVista(t *testing.T) {
	m := hiloLargo(40)
	m.Vista(60, 12)
	antes := m.offset

	seguia := m.SuspenderSeguimiento()
	m.Fragmento(" texto nuevo que llega en vivo")
	m.Vista(60, 12)
	if m.offset != antes {
		t.Fatalf("la vista saltó al llegar contenido: %d → %d", antes, m.offset)
	}

	m.ReanudarSeguimiento(seguia)
	m.Vista(60, 12)
	if m.offset != m.maxOffset {
		t.Fatalf("al soltar debería retomar el final: %d ≠ %d", m.offset, m.maxOffset)
	}
}
