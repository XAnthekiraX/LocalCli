package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"localcli/internal/tui/components/chat"
	"localcli/internal/tui/render"
)

// arrastrar pulsa y mueve el ratón con el botón izquierdo; soltar queda aparte.
func arrastrar(app *App, desde, hasta punto) {
	app.raton(tea.MouseMsg{X: desde.Columna, Y: desde.Fila, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	app.raton(tea.MouseMsg{X: hasta.Columna, Y: hasta.Fila, Action: tea.MouseActionMotion})
}

// ejecutar corre un comando como lo haría el bucle: un lote ejecuta sus comandos.
func ejecutar(cmd tea.Cmd) {
	if batch, ok := cmd().(tea.BatchMsg); ok {
		for _, sub := range batch {
			sub()
		}
	}
}

// La caja de entrada de la bienvenida expone su rectángulo en cada repintado.
func TestLaCajaDeEntradaExponeSusLimites(t *testing.T) {
	app := Nueva(nil)
	app.ancho, app.alto = 100, 40
	app.marco()

	limites := app.zonaEntrada
	if limites.Alto <= 0 || limites.Ancho <= 0 {
		t.Fatalf("la caja no expuso sus límites: %+v", limites)
	}
	if limites.Fila <= 6 {
		t.Fatalf("la caja debería quedar bajo el logotipo: fila %d", limites.Fila)
	}
	if limites.Columna+limites.Ancho > app.ancho || limites.Fila+limites.Alto > app.alto {
		t.Fatalf("la caja se sale del marco: %+v", limites)
	}

	// El rectángulo coincide con la caja pintada: el placeholder queda dentro, con
	// su celda de aire a la izquierda, y las pistas y la versión quedan fuera.
	lineas := strings.Split(render.SinANSI(app.marco()), "\n")
	filaTexto := -1
	for i, linea := range lineas {
		if strings.Contains(linea, "Escribe para iniciar la conversacion") {
			filaTexto = i
		}
	}
	if len(lineas) != app.alto {
		t.Fatalf("el marco tiene %d filas, quiero %d", len(lineas), app.alto)
	}
	if filaTexto != limites.Fila+1 {
		t.Fatalf("la línea de entrada está en la fila %d, el rectángulo empieza en %d: %+v", filaTexto, limites.Fila, limites)
	}
	if columna := strings.Index(lineas[filaTexto], "Escribe"); columna != limites.Columna+2 {
		t.Fatalf("la caja arranca en %d, su texto en %d, quiero %d", limites.Columna, columna, limites.Columna+2)
	}
	for _, ajeno := range []string{"TAB agentes", render.Version} {
		for fila := limites.Fila; fila < limites.Fila+limites.Alto; fila++ {
			if strings.Contains(lineas[fila], ajeno) {
				t.Fatalf("%q quedó dentro del rectángulo de la caja", ajeno)
			}
		}
	}
}

// Un arrastre que empieza en la caja de entrada no sale de ella.
func TestArrastreEnLaCajaNoAlcanzaElLogotipo(t *testing.T) {
	app := Nueva(nil)
	app.ancho, app.alto = 100, 40
	app.marco()

	zona := app.zonaEntrada
	// Tira desde dentro de la caja hasta la esquina del logotipo.
	arrastrar(app, punto{Fila: zona.Fila + 1, Columna: zona.Columna + 2}, punto{Fila: 0, Columna: 0})

	tramos := app.sel.resolver(app.lineasMarco())
	if len(tramos) == 0 {
		t.Fatal("la selección quedó vacía")
	}
	for _, tr := range tramos {
		if tr.Fila < zona.Fila || tr.Fila >= zona.Fila+zona.Alto {
			t.Fatalf("el realce salió de la caja por filas: %+v", tr)
		}
		if tr.Desde < zona.Columna || tr.Hasta > zona.Columna+zona.Ancho {
			t.Fatalf("el realce salió de la caja por columnas: %+v", tr)
		}
	}

	texto := app.sel.texto()
	if texto == "" {
		t.Fatal("no se eligió nada dentro de la caja")
	}
	for _, ajeno := range []string{"█", "TAB agentes", render.Version} {
		if strings.Contains(texto, ajeno) {
			t.Fatalf("la selección alcanzó %q: %q", ajeno, texto)
		}
	}
}

// Un repintado no cancela ni mueve el realce: sigue sobre el mismo texto.
func TestRepintadoNoMueveElRealceAnclado(t *testing.T) {
	antes := []string{
		"    alfa",
		"    hola mundo elegido",
		"    omega",
	}
	var s seleccion
	s.iniciar(punto{Fila: 1, Columna: 4}, antes, rect{}, false)
	s.extender(punto{Fila: 1, Columna: 13}, antes)

	if texto := s.texto(); texto != "hola mundo" {
		t.Fatalf("texto anclado = %q, quiero %q", texto, "hola mundo")
	}

	// El repintado mete una línea arriba: el texto elegido baja a la fila 2.
	despues := []string{
		"    linea nueva",
		"    alfa",
		"    hola mundo elegido",
		"    omega",
	}
	tramos := s.resolver(despues)
	if len(tramos) != 1 {
		t.Fatalf("tramos del realce = %d, quiero 1", len(tramos))
	}
	if tramos[0] != (tramo{Fila: 2, Desde: 4, Hasta: 14}) {
		t.Fatalf("el realce no siguió al texto elegido: %+v", tramos[0])
	}
}

// appEnPrincipal deja la vista principal con el sidebar abierto y algo de
// historial en el chat: es donde viven las zonas del chat y del sidebar.
func appEnPrincipal(ancho, alto int) *App {
	app := Nueva(nil)
	app.vista = VistaPrincipal
	app.ancho, app.alto = ancho, alto
	app.chat.AñadirUsuario("hola")
	app.chat.Fragmento("respuesta del agente")
	app.chat.CerrarTurno()
	app.marco()
	return app
}

// appConHiloLargo deja la vista principal con un historial que desborda la ventana.
func appConHiloLargo(ancho, alto int) *App {
	app := Nueva(nil)
	app.vista = VistaPrincipal
	app.ancho, app.alto = ancho, alto
	app.chat.AñadirUsuario("pregunta inicial")
	for i := 0; i < 60; i++ {
		app.chat.AñadirSegmento(chat.Segmento{Hablante: chat.Agente, Texto: "linea de relleno"})
	}
	app.marco()
	return app
}

// Un arrastre del chat contra el borde superior desplaza el historial y la
// selección abarca líneas que estaban fuera del viewport; al soltar se copia el
// tramo completo y vuelve el seguimiento.
func TestElArrastreEnElChatSeProlongaFueraDelViewport(t *testing.T) {
	app := appConHiloLargo(100, 16)
	if app.zonaChat.Alto < 4 {
		t.Fatalf("el chat es demasiado bajo para la prueba: %+v", app.zonaChat)
	}

	visibleArriba, ok := app.chat.ContenidoBajoVentana(1)
	if !ok {
		t.Fatal("el chat no expuso contenido")
	}

	app.raton(tea.MouseMsg{X: 2, Y: app.zonaChat.Alto - 2, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	for i := 0; i < 5; i++ {
		app.raton(tea.MouseMsg{X: 2, Y: 0, Action: tea.MouseActionMotion})
	}

	if app.sel.lineaFin >= visibleArriba {
		t.Fatalf("la selección no rebasó el viewport: fin %d, arriba visible %d", app.sel.lineaFin, visibleArriba)
	}

	original := copiarFunc
	defer func() { copiarFunc = original }()
	var copiado string
	copiarFunc = func(texto string) { copiado = texto }

	_, cmd := app.raton(tea.MouseMsg{Action: tea.MouseActionRelease, Button: tea.MouseButtonLeft})
	ejecutar(cmd)
	if copiado == "" {
		t.Fatal("no se copió el tramo")
	}
	if app.sel.arrastrando {
		t.Fatal("el arrastre no terminó al soltar")
	}
}

// Un arrastre que empieza en el chat no alcanza el sidebar: el realce se queda en
// la columna del chat y no cruza la banda de separación.
func TestArrastreDelChatNoCruzaAlSidebar(t *testing.T) {
	app := appEnPrincipal(100, 24)
	anchoChat := app.anchoColumna()
	if app.zonaChat.Alto <= 0 {
		t.Fatalf("el chat no expuso su zona: %+v", app.zonaChat)
	}
	if app.zonaPanel.Alto <= 0 {
		t.Fatalf("el sidebar no expuso su zona: %+v", app.zonaPanel)
	}

	// Tira desde el chat hasta la última columna del sidebar.
	arrastrar(app, punto{Fila: 1, Columna: 2}, punto{Fila: 1, Columna: app.ancho - 1})

	tramos := app.tramosSeleccion()
	if len(tramos) == 0 {
		t.Fatal("el arrastre del chat no resaltó nada")
	}
	tope := false
	for _, tr := range tramos {
		if tr.Desde < 0 || tr.Hasta > anchoChat {
			t.Fatalf("el realce salió del chat hacia el sidebar: %+v (el chat tiene %d columnas)", tr, anchoChat)
		}
		tope = tope || tr.Hasta == anchoChat
	}
	if !tope {
		t.Fatalf("el realce no llegó al borde del chat, que es donde debe cortarse: %+v", tramos)
	}
	if texto := app.sel.textoContenido(app.chat.LineasContenido()); strings.Contains(texto, "CONTEXTO") {
		t.Fatalf("la selección del chat alcanzó el sidebar: %q", texto)
	}
}

// Y al revés: un arrastre que empieza en el sidebar no cruza al chat.
func TestArrastreDelSidebarNoCruzaAlChat(t *testing.T) {
	app := appEnPrincipal(100, 24)
	panel := app.zonaPanel
	if panel.Alto <= 0 {
		t.Fatalf("el sidebar no expuso su zona: %+v", panel)
	}

	// Tira desde el sidebar hasta la primera columna del chat.
	arrastrar(app, punto{Fila: 1, Columna: panel.Columna + 3}, punto{Fila: 1, Columna: 0})

	tramos := app.sel.resolver(app.lineasMarco())
	if len(tramos) == 0 {
		t.Fatal("el arrastre del sidebar no resaltó nada")
	}
	tope := false
	for _, tr := range tramos {
		if tr.Desde < panel.Columna || tr.Hasta > panel.Columna+panel.Ancho {
			t.Fatalf("el realce salió del sidebar hacia el chat: %+v (el sidebar es %+v)", tr, panel)
		}
		tope = tope || tr.Desde == panel.Columna
	}
	if !tope {
		t.Fatalf("el realce no llegó al borde del sidebar, que es donde debe cortarse: %+v", tramos)
	}
}

// Al soltar, se copia el texto anclado y el realce se apaga.
func TestSoltarCopiaElTextoElegido(t *testing.T) {
	app := Nueva(nil)
	app.ancho, app.alto = 100, 40
	app.marco()

	zona := app.zonaEntrada
	arrastrar(app, punto{Fila: zona.Fila + 1, Columna: zona.Columna + 2}, punto{Fila: zona.Fila + 1, Columna: zona.Columna + 6})

	if pintado := app.conSeleccion(app.marco()); !strings.Contains(pintado, "\x1b[7m") {
		t.Fatal("el arrastre no resaltó nada")
	}

	original := copiarFunc
	defer func() { copiarFunc = original }()
	var copiado string
	copiarFunc = func(texto string) { copiado = texto }

	_, cmd := app.raton(tea.MouseMsg{Action: tea.MouseActionRelease, Button: tea.MouseButtonLeft})
	if cmd == nil {
		t.Fatal("soltar no copió nada")
	}
	ejecutar(cmd)
	if copiado == "" {
		t.Fatal("el portapapeles se quedó vacío")
	}
	if app.sel.arrastrando {
		t.Fatal("el realce no se apagó al soltar")
	}
	if pintado := app.conSeleccion(app.marco()); strings.Contains(pintado, "\x1b[7m") {
		t.Fatal("el realce sigue pintado tras soltar")
	}
}
