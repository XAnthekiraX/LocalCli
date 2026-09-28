package tui

// herramientas_test.go — T-B024-18 / T-F037: la línea de herramienta se pinta
// desde los eventos, en una sola línea que se completa al terminar, con el verbo
// y el tema (la ruta que se busca), sin agente y sin volcar la salida cruda.

import (
	"strings"
	"testing"

	"localcli/internal/session"
)

// TestLosEventosDeHerramientaSePintan — la invocación abre la línea con el verbo
// y el tema; el resultado la completa con la marca y la medida, sin volcar la
// salida ni el agente.
func TestLosEventosDeHerramientaSePintan(t *testing.T) {
	h := nuevoArnes(t)
	h.evento(EventoHerramientaInvocada, map[string]string{
		"herramienta": "leer_archivo",
		"agente":      "plan",
		"verbo":       "LEER",
		"tema":        "internal/tui/chat.go",
	})
	h.veSiContiene("LEER [internal/tui/chat.go]")
	if strings.Contains(h.ve(), "(agente plan)") {
		t.Errorf("la línea ya no nombra al agente:\n%s", h.ve())
	}
	if strings.Contains(h.ve(), "[→ herramienta:") {
		t.Errorf("la línea ya no lleva el prefijo «[→ herramienta:»:\n%s", h.ve())
	}
	// Mientras corre, el indicador en vivo lo dice.
	h.veSiContiene("Usando herramienta: leer_archivo")

	h.evento(EventoHerramientaResultado, map[string]string{
		"herramienta": "leer_archivo",
		"ok":          "true",
		"truncado":    "true",
		"medida":      "70 líneas",
	})
	h.veSiContiene("✓ LEER [internal/tui/chat.go] · 70 líneas · recortado")
	// Es UNA sola línea: se completa, no se añade otra.
	if n := strings.Count(h.ve(), "LEER [internal/tui/chat.go]"); n != 1 {
		t.Errorf("la línea de herramienta debe ser una sola, hay %d:\n%s", n, h.ve())
	}
	// Al terminar, el indicador deja de nombrar la herramienta.
	if strings.Contains(h.ve(), "Usando herramienta") {
		t.Error("terminada la herramienta, el indicador ya no la nombra")
	}
}

// TestUnResultadoConErrorSePinta — cuando el fallo llega al evento, la línea lo
// dice; la salida cruda nunca se pinta.
func TestUnResultadoConErrorSePinta(t *testing.T) {
	h := nuevoArnes(t)
	h.evento(EventoHerramientaInvocada, map[string]string{
		"herramienta": "crear_archivo",
		"verbo":       "CREAR",
		"tema":        "nuevo.txt",
	})
	h.evento(EventoHerramientaResultado, map[string]string{
		"herramienta": "crear_archivo",
		"ok":          "false",
		"error":       "el archivo ya existe",
	})
	h.veSiContiene("✗ CREAR [nuevo.txt] · el archivo ya existe")
	if strings.Contains(h.ve(), "contenido:") {
		t.Error("la línea de herramienta no puede volcar la salida cruda")
	}
}

// TestElChatRespetaElOrdenTextoHerramientaTexto — el texto que precede a una
// herramienta queda arriba de su línea y el que viene después abre un globo
// nuevo: el hilo sigue el orden de ejecución, no el orden en que llegaron los
// mensajes al historial.
func TestElChatRespetaElOrdenTextoHerramientaTexto(t *testing.T) {
	h := nuevoArnes(t)
	h.evento(EventoToken, map[string]string{"texto": "voy a leerlo"})
	h.evento(EventoHerramientaInvocada, map[string]string{
		"herramienta": "leer_archivo",
		"verbo":       "LEER",
		"tema":        "internal/tui/chat.go",
	})
	h.evento(EventoHerramientaResultado, map[string]string{
		"herramienta": "leer_archivo",
		"ok":          "true",
		"medida":      "70 líneas",
	})
	h.evento(EventoToken, map[string]string{"texto": "ya lo leí"})
	// Cerrar el turno vuelca el texto posterior como globo propio, tras la línea.
	h.evento(session.EventoEstadoSesion, map[string]string{"sesion": "s1", "estado": session.EstadoTerminada})

	v := h.ve()
	ip, il, is := strings.Index(v, "voy a leerlo"), strings.Index(v, "LEER ["), strings.Index(v, "ya lo leí")
	if ip < 0 || il < 0 || is < 0 {
		t.Fatalf("faltan piezas en la vista:\n%s", v)
	}
	if !(ip < il && il < is) {
		t.Errorf("orden texto→herramienta→texto (texto %d, herramienta %d, texto %d):\n%s", ip, il, is, v)
	}
}

// TestLosTokensDelTurnoAlimentanElPanel — el consumo que reporta el turno se
// refleja en el panel de contexto.
func TestLosTokensDelTurnoAlimentanElPanel(t *testing.T) {
	h := nuevoArnes(t)
	h.evento(EventoTokensTurno, map[string]string{"entrada": "120", "salida": "45"})
	if h.app.Panel.Tokens != 45 {
		t.Errorf("tokens del panel = %d, quiero 45", h.app.Panel.Tokens)
	}
	if h.app.Panel.TokensEstimados {
		t.Error("un conteo reportado por el modelo no es una estimación")
	}
}
