package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"localcli/internal/tui/components/chat"
	"localcli/internal/tui/render"
)

// appConBloqueDeCodigo deja la vista principal con una respuesta del agente que
// trae un bloque de código, pintada y lista para localizar su botón.
func appConBloqueDeCodigo(ancho, alto int) *App {
	app := Nueva(nil)
	app.vista = VistaPrincipal
	app.ancho, app.alto = ancho, alto
	app.chat.AñadirSegmento(chat.Segmento{
		Hablante: chat.Agente,
		Texto:    "mira esto:\n```go\nx := 1\n```\nfin",
	})
	app.marco()
	return app
}

// rotuloCopiar es el rótulo del botón tal como se pinta en la cabecera del bloque.
const rotuloCopiar = "copiar"

// localizarBoton busca en el marco la celda donde se pinta el rótulo del botón de
// copiar y la devuelve como punto de pantalla.
func localizarBoton(t *testing.T, app *App) punto {
	t.Helper()
	for fila, linea := range app.lineasMarco() {
		if col := strings.Index(linea, rotuloCopiar); col >= 0 {
			return punto{Fila: fila, Columna: col + 1}
		}
	}
	t.Fatalf("el botón de copiar no se pintó en el marco:\n%s", strings.Join(app.lineasMarco(), "\n"))
	return punto{}
}

// Un clic sobre el botón de copiar de un bloque copia su cuerpo exacto —sin
// fences ni cabecera— y no inicia una selección.
func TestClicEnElBotonDeCopiarCopiaElBloque(t *testing.T) {
	app := appConBloqueDeCodigo(100, 40)
	p := localizarBoton(t, app)

	original := copiarFunc
	defer func() { copiarFunc = original }()
	var copiado string
	copiarFunc = func(texto string) { copiado = texto }

	_, cmd := app.raton(tea.MouseMsg{X: p.Columna, Y: p.Fila, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	if cmd == nil {
		t.Fatal("el clic sobre el botón no copió nada")
	}
	ejecutar(cmd)

	if copiado != "x := 1" {
		t.Fatalf("se copió %q, quiero %q", copiado, "x := 1")
	}
	if app.sel.arrastrando {
		t.Fatal("el clic sobre el botón inició una selección")
	}
}

// Un clic que no cae sobre el botón no copia: sigue el curso normal de la
// selección.
func TestClicFueraDelBotonNoCopia(t *testing.T) {
	app := appConBloqueDeCodigo(100, 40)
	p := localizarBoton(t, app)

	original := copiarFunc
	defer func() { copiarFunc = original }()
	var copiado string
	copiarFunc = func(texto string) { copiado = texto }

	// Una fila por debajo del botón (el cuerpo del bloque) no es el botón.
	otro := punto{Fila: p.Fila + 1, Columna: p.Columna}
	_, cmd := app.raton(tea.MouseMsg{X: otro.Columna, Y: otro.Fila, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	if cmd != nil {
		ejecutar(cmd)
	}
	if copiado != "" {
		t.Fatalf("un clic fuera del botón copió %q", copiado)
	}
}

// La geometría del botón se traduce a coordenadas de ventana: la línea de
// contenido del botón, una vez pintada, expone una celda donde el chat lo
// localiza.
func TestElBotonSeLocalizaEnCoordenadasDeVentana(t *testing.T) {
	app := appConBloqueDeCodigo(100, 40)
	p := localizarBoton(t, app)

	boton, ok := app.chat.BotonEnVentana(p.Fila-app.zonaChat.Fila, p.Columna-app.zonaChat.Columna)
	if !ok {
		t.Fatalf("el chat no localizó el botón en (%d, %d)", p.Fila, p.Columna)
	}
	if boton.Codigo != "x := 1" {
		t.Fatalf("el botón localizado copia %q", boton.Codigo)
	}
	// La celda clicada cae dentro de las columnas del botón.
	if p.Columna-app.zonaChat.Columna < boton.Col || p.Columna-app.zonaChat.Columna >= boton.Col+boton.Ancho {
		t.Fatalf("la celda (%d, %d) queda fuera del botón %+v", p.Fila, p.Columna, boton)
	}
	// Y el marco coincide: en la fila del botón el rótulo está donde el botón dice.
	linea := render.SinANSI(app.lineasMarco()[p.Fila])
	if !strings.Contains(linea, rotuloCopiar) {
		t.Fatalf("la fila del botón no lleva su rótulo: %q", linea)
	}
}
