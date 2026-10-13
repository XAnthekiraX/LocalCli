package tui

import (
	"strings"
	"testing"

	"localcli/internal/tui/components/chat"
	"localcli/internal/tui/render"
)

// TestLaActividadEsUnaFilaFija: el indicador en vivo va fuera del chat, así que
// el scroll del historial no lo desplaza.
func TestLaActividadEsUnaFilaFija(t *testing.T) {
	app := appDorada(80, 20, VistaPrincipal)
	app.chat.Vaciar()
	app.chat.AñadirUsuario("hola")
	for i := 0; i < 40; i++ {
		app.chat.AñadirSegmento(chat.Segmento{Hablante: chat.Agente, Texto: "linea de relleno"})
	}

	fila := filaCon(render.SinANSI(app.View()), "Pensando")
	if fila < 0 {
		t.Fatalf("la actividad debería verse en la vista:\n%s", render.SinANSI(app.View()))
	}
	app.chat.Subir(10)
	if otra := filaCon(render.SinANSI(app.View()), "Pensando"); otra != fila {
		t.Fatalf("la actividad se desplazó con el scroll: fila %d → %d", fila, otra)
	}
}

// filaCon devuelve el índice de la primera fila que contiene `sub`, o -1.
func filaCon(texto, sub string) int {
	for i, linea := range strings.Split(texto, "\n") {
		if strings.Contains(linea, sub) {
			return i
		}
	}
	return -1
}

// La fila fija de actividad queda fuera de la zona de selección del chat.
func TestLaActividadQuedaFueraDeLaZonaDelChat(t *testing.T) {
	app := appDorada(80, 20, VistaPrincipal)
	app.chat.Vaciar()
	app.chat.AñadirUsuario("hola")
	app.chat.Fragmento("respuesta en curso")
	app.marco()

	fila := filaCon(render.SinANSI(app.View()), "Generando")
	if fila < 0 {
		t.Fatalf("no se ve la línea de actividad:\n%s", render.SinANSI(app.View()))
	}
	if app.zonaChat.contiene(punto{Fila: fila, Columna: 0}) {
		t.Fatalf("la fila de actividad (%d) cayó dentro de la zona del chat (%+v)", fila, app.zonaChat)
	}
}
