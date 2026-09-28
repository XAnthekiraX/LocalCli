package tui

// Tests de la selección con el ratón y su copia (selection.go) y del scroll con
// la rueda.

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestTextoSeleccionadoExtraeElRangoSinCódigos(t *testing.T) {
	// Los códigos de color no cuentan como columnas.
	vista := "\x1b[31mhola\x1b[0m mundo\nsegunda línea"
	if got := textoSeleccionado(vista, posicion{X: 0, Y: 0}, posicion{X: 5, Y: 0}); got != "hola" {
		t.Errorf("selección de una línea: %q", got)
	}
	// Varias líneas y extremos en cualquier orden (se ordenan solos).
	got := textoSeleccionado(vista, posicion{X: 3, Y: 1}, posicion{X: 0, Y: 0})
	if got != "hola mundo\nseg" {
		t.Errorf("selección multilínea con extremos invertidos: %q", got)
	}
	// Un clic sin arrastrar no copia nada.
	if got := textoSeleccionado(vista, posicion{X: 2, Y: 0}, posicion{X: 2, Y: 0}); got != "" {
		t.Errorf("un clic sin arrastre no selecciona: %q", got)
	}
	// Un marco vacío no revienta.
	if got := textoSeleccionado("", posicion{X: 0, Y: 0}, posicion{X: 1, Y: 1}); got != "" {
		t.Errorf("vista vacía: %q", got)
	}
}

func TestElArrastreDelRatónCopiaLaSelección(t *testing.T) {
	var copiado []string
	original := copiarFunc
	copiarFunc = func(s string) { copiado = append(copiado, s) }
	defer func() { copiarFunc = original }()

	a := Nuevo(&puertoStub{})
	a.Vista = VistaPrincipal
	a.Panel.SesionID = "s1"
	pulsa(t, a, tea.WindowSizeMsg{Width: 60, Height: 12})
	a.ultimaVista = "hola mundo\nsegunda línea"

	pulsa(t, a, tea.MouseMsg{Button: tea.MouseButtonLeft, Action: tea.MouseActionPress, X: 0, Y: 0})
	pulsa(t, a, tea.MouseMsg{Button: tea.MouseButtonLeft, Action: tea.MouseActionMotion, X: 5, Y: 0})
	pulsa(t, a, tea.MouseMsg{Button: tea.MouseButtonLeft, Action: tea.MouseActionRelease, X: 5, Y: 0})
	if len(copiado) != 1 || copiado[0] != "hola" {
		t.Fatalf("al soltar se copia el texto seleccionado: %v", copiado)
	}
	if a.ratonSelec {
		t.Error("al soltar termina la selección")
	}

	// Un clic sin arrastrar no copia.
	pulsa(t, a, tea.MouseMsg{Button: tea.MouseButtonLeft, Action: tea.MouseActionPress, X: 2, Y: 0})
	pulsa(t, a, tea.MouseMsg{Button: tea.MouseButtonLeft, Action: tea.MouseActionRelease, X: 2, Y: 0})
	if len(copiado) != 1 {
		t.Error("un clic sin arrastre no copia")
	}
}

func TestLaRuedaDesplazaElHistorial(t *testing.T) {
	a := Nuevo(&puertoStub{})
	a.Vista = VistaPrincipal
	a.Panel.SesionID = "s1"
	pulsa(t, a, tea.WindowSizeMsg{Width: 60, Height: 12})
	for i := 1; i <= 40; i++ {
		a.Chat.AñadirSistema(fmt.Sprintf("mensaje %d", i))
	}
	_ = a.View()
	if a.Chat.OcultasAbajo() != 0 {
		t.Fatalf("al pie no queda nada abajo, quedan %d", a.Chat.OcultasAbajo())
	}

	// La rueda hacia arriba sube por el historial.
	pulsa(t, a, tea.MouseMsg{Button: tea.MouseButtonWheelUp, Action: tea.MouseActionPress})
	_ = a.View()
	if a.Chat.OcultasAbajo() == 0 {
		t.Error("la rueda hacia arriba sube por el historial")
	}

	// La rueda hacia abajo vuelve al final.
	for i := 0; i < 40; i++ {
		pulsa(t, a, tea.MouseMsg{Button: tea.MouseButtonWheelDown, Action: tea.MouseActionPress})
	}
	if a.Chat.OcultasAbajo() != 0 || !strings.Contains(sinEstilo(a.View()), "mensaje 40") {
		t.Error("la rueda hacia abajo vuelve al final")
	}
}

// filaDe localiza en el marco la primera fila que contiene `texto` y devuelve su
// índice (base 0) y la columna visible donde empieza el texto.
func filaDe(t *testing.T, vista, texto string) (y, x int) {
	t.Helper()
	for i, l := range strings.Split(sinEstilo(vista), "\n") {
		if j := strings.Index(l, texto); j >= 0 {
			return i, len([]rune(l[:j]))
		}
	}
	t.Fatalf("no encontré %q en el marco:\n%s", texto, sinEstilo(vista))
	return 0, 0
}

func TestElResaltadoEnvuelveSoloLaSelección(t *testing.T) {
	// Sin códigos: se envuelve exactamente el tramo y el texto visible no cambia.
	got := resaltarSeleccion("hola mundo", posicion{X: 0, Y: 0}, posicion{X: 4, Y: 0})
	if !strings.Contains(got, seleccionOn+"hola"+seleccionOff) {
		t.Errorf("el tramo seleccionado va en video inverso: %q", got)
	}
	if sinANSI(got) != "hola mundo" {
		t.Errorf("el realce no altera el texto visible: %q", sinANSI(got))
	}
	// Con códigos de color dentro del tramo: se conservan y el inverso se
	// reafirma tras el reset intermedio para no apagarse.
	conColor := "\x1b[31mhola\x1b[0m mundo"
	got = resaltarSeleccion(conColor, posicion{X: 0, Y: 0}, posicion{X: 4, Y: 0})
	if sinANSI(got) != "hola mundo" || !strings.Contains(got, "\x1b[7mhola") {
		t.Errorf("con color, el tramo sigue en inverso sin cambiar el texto: %q", got)
	}
	// Un clic sin arrastre (una sola celda) no resalta nada.
	if got := resaltarSeleccion("hola", posicion{X: 2, Y: 0}, posicion{X: 2, Y: 0}); got != "hola" {
		t.Errorf("un clic sin arrastre no resalta: %q", got)
	}
	// Un marco vacío no revienta.
	if got := resaltarSeleccion("", posicion{X: 0, Y: 0}, posicion{X: 1, Y: 1}); got != "" {
		t.Errorf("vista vacía: %q", got)
	}
}

func TestElArrastrePintaLaSelección(t *testing.T) {
	a := Nuevo(&puertoStub{})
	a.Vista = VistaPrincipal
	a.Panel.SesionID = "s1"
	pulsa(t, a, tea.WindowSizeMsg{Width: 60, Height: 16})
	a.Chat.AñadirUsuario("hola mundo")
	y, x := filaDe(t, a.View(), "hola mundo")

	// Un clic sin arrastrar (pulsar y soltar en la misma celda) no pinta nada.
	// Se busca el tramo concreto: el cursor del input también usa video inverso,
	// así que no basta con que aparezca el código de encendido.
	pulsa(t, a, tea.MouseMsg{Button: tea.MouseButtonLeft, Action: tea.MouseActionPress, X: x, Y: y})
	pulsa(t, a, tea.MouseMsg{Button: tea.MouseButtonLeft, Action: tea.MouseActionRelease, X: x, Y: y})
	if strings.Contains(a.View(), seleccionOn+"hola") {
		t.Error("un clic sin arrastre no resalta")
	}

	// Arrastrar sí: el tramo se ve en video inverso, sin alterar el texto.
	pulsa(t, a, tea.MouseMsg{Button: tea.MouseButtonLeft, Action: tea.MouseActionPress, X: x, Y: y})
	pulsa(t, a, tea.MouseMsg{Button: tea.MouseButtonLeft, Action: tea.MouseActionMotion, X: x + 4, Y: y})
	pintado := a.View()
	if !strings.Contains(pintado, seleccionOn+"hola"+seleccionOff) {
		t.Errorf("la selección debe pintarse resaltada:\n%q", pintado)
	}
	if sinEstilo(pintado) != sinEstilo(a.ultimaVista) {
		t.Error("resaltar no puede alterar el texto de la pantalla")
	}
}

func TestLaRuedaNoCancelaLaSelecciónEnCurso(t *testing.T) {
	var copiado []string
	original := copiarFunc
	copiarFunc = func(s string) { copiado = append(copiado, s) }
	defer func() { copiarFunc = original }()

	a := Nuevo(&puertoStub{})
	a.Vista = VistaPrincipal
	a.Panel.SesionID = "s1"
	pulsa(t, a, tea.WindowSizeMsg{Width: 60, Height: 20})
	for i := 1; i <= 40; i++ {
		a.Chat.AñadirSistema(fmt.Sprintf("mensaje %d", i))
	}
	y, x := filaDe(t, a.View(), "mensaje")

	pulsa(t, a, tea.MouseMsg{Button: tea.MouseButtonLeft, Action: tea.MouseActionPress, X: x, Y: y})
	pulsa(t, a, tea.MouseMsg{Button: tea.MouseButtonLeft, Action: tea.MouseActionMotion, X: x + 5, Y: y})
	iniAntes := a.ratonIni

	// La rueda desplaza el historial sin cortar el arrastre: la selección sigue
	// viva y su ancla se reancla al texto que tenía debajo.
	pulsa(t, a, tea.MouseMsg{Button: tea.MouseButtonWheelUp, Action: tea.MouseActionPress})
	if !a.ratonSelec {
		t.Fatal("la rueda no puede cancelar la selección en curso")
	}
	if a.ratonIni == iniAntes {
		t.Errorf("el ancla debe reanclarse al texto al desplazar: %v", a.ratonIni)
	}
	_ = a.View()

	// Y al soltar sigue copiando lo seleccionado.
	pulsa(t, a, tea.MouseMsg{Button: tea.MouseButtonLeft, Action: tea.MouseActionRelease, X: x + 5, Y: y})
	if len(copiado) != 1 || strings.TrimSpace(copiado[0]) == "" {
		t.Fatalf("soltar tras el scroll copia la selección: %v", copiado)
	}
}

func TestLaSelecciónSeAnclaAlScroll(t *testing.T) {
	a := Nuevo(&puertoStub{})
	a.Vista = VistaPrincipal
	a.Panel.SesionID = "s1"
	pulsa(t, a, tea.WindowSizeMsg{Width: 60, Height: 30})
	for i := 1; i <= 30; i++ {
		a.Chat.AñadirSistema(fmt.Sprintf("mensaje %02d", i))
	}
	_ = a.View()
	// A media historia: queda contenido por encima y por debajo para poder
	// desplazar sin topar con los extremos.
	a.Chat.Subir(1000)
	a.Chat.Bajar(20)
	_ = a.View()

	y, _ := filaDe(t, a.View(), "mensaje")
	lineas := strings.Split(sinANSI(a.ultimaVista), "\n")
	ancla := lineas[y]

	pulsa(t, a, tea.MouseMsg{Button: tea.MouseButtonLeft, Action: tea.MouseActionPress, X: 0, Y: y})
	pulsa(t, a, tea.MouseMsg{Button: tea.MouseButtonLeft, Action: tea.MouseActionMotion, X: 0, Y: y + 1})

	// La rueda arriba mueve el contenido hacia abajo; el ancla debe seguir sobre
	// la MISMA línea (anclada al texto, no a la pantalla), no desaparecer.
	pulsa(t, a, tea.MouseMsg{Button: tea.MouseButtonWheelUp, Action: tea.MouseActionPress})
	if !a.ratonSelec {
		t.Fatal("la rueda no cancela la selección")
	}
	despues := strings.Split(sinANSI(a.View()), "\n")
	if a.ratonIni.Y < 0 || a.ratonIni.Y >= len(despues) || despues[a.ratonIni.Y] != ancla {
		t.Fatalf("el ancla debe seguir en la misma línea tras el scroll:\nantes: %q (fila %d)\nahora: fila %d",
			ancla, y, a.ratonIni.Y)
	}
}

func TestLaSelecciónDesapareceAlSoltar(t *testing.T) {
	var copiado []string
	original := copiarFunc
	copiarFunc = func(s string) { copiado = append(copiado, s) }
	defer func() { copiarFunc = original }()

	a := Nuevo(&puertoStub{})
	a.Vista = VistaPrincipal
	a.Panel.SesionID = "s1"
	pulsa(t, a, tea.WindowSizeMsg{Width: 60, Height: 16})
	a.Chat.AñadirUsuario("hola mundo")
	y, x := filaDe(t, a.View(), "hola mundo")

	pulsa(t, a, tea.MouseMsg{Button: tea.MouseButtonLeft, Action: tea.MouseActionPress, X: x, Y: y})
	pulsa(t, a, tea.MouseMsg{Button: tea.MouseButtonLeft, Action: tea.MouseActionMotion, X: x + 4, Y: y})
	if !strings.Contains(a.View(), seleccionOn+"hola") {
		t.Fatal("durante el arrastre la selección se ve resaltada")
	}

	// Al soltar: se copia y el realce desaparece (la selección ya no vive). No se
	// ejecuta el temporizador del aviso: se comprueba aparte.
	pulsa(t, a, tea.MouseMsg{Button: tea.MouseButtonLeft, Action: tea.MouseActionRelease, X: x + 4, Y: y})
	if len(copiado) != 1 || copiado[0] != "hola" {
		t.Fatalf("al soltar se copia lo seleccionado: %v", copiado)
	}
	if a.ratonIni != (posicion{}) || a.ratonFin != (posicion{}) {
		t.Errorf("al soltar la selección se descarta: %v, %v", a.ratonIni, a.ratonFin)
	}
	if strings.Contains(a.View(), seleccionOn+"hola") {
		t.Error("al soltar el realce desaparece")
	}
}

func TestElAvisoDeCopiadoSeVeArribaALaDerechaYSeApagaSolo(t *testing.T) {
	original := copiarFunc
	copiarFunc = func(string) {}
	defer func() { copiarFunc = original }()

	a := Nuevo(&puertoStub{})
	a.Vista = VistaPrincipal
	a.Panel.SesionID = "s1"
	pulsa(t, a, tea.WindowSizeMsg{Width: 60, Height: 16})
	a.Chat.AñadirUsuario("hola mundo")
	y, x := filaDe(t, a.View(), "hola mundo")

	// Antes de copiar no hay aviso.
	if strings.Contains(sinEstilo(a.View()), "[Copiado]") {
		t.Fatal("sin copiar no debe verse el aviso")
	}

	pulsa(t, a, tea.MouseMsg{Button: tea.MouseButtonLeft, Action: tea.MouseActionPress, X: x, Y: y})
	pulsa(t, a, tea.MouseMsg{Button: tea.MouseButtonLeft, Action: tea.MouseActionMotion, X: x + 4, Y: y})
	cmd := pulsa(t, a, tea.MouseMsg{Button: tea.MouseButtonLeft, Action: tea.MouseActionRelease, X: x + 4, Y: y})
	if cmd == nil {
		t.Fatal("copiar debe programar el apagado del aviso")
	}

	lineas := strings.Split(sinEstilo(a.View()), "\n")
	// Va en la primera línea (arriba) y pegado al borde derecho.
	if !strings.HasSuffix(lineas[0], "[Copiado]") {
		t.Fatalf("el aviso va arriba a la derecha:\n%s", lineas[0])
	}
	if ancho := len([]rune(lineas[0])); ancho != 60 {
		t.Errorf("el aviso queda pegado al borde derecho (ancho %d, quiero 60)", ancho)
	}

	// Vencido su tiempo, se apaga solo.
	pulsa(t, a, copiadoExpiradoMsg{gen: a.copiadoGen})
	if a.copiado || strings.Contains(sinEstilo(a.View()), "[Copiado]") {
		t.Error("el aviso debe apagarse al vencer su tiempo")
	}
	// Un temporizador viejo no apaga el aviso de una copia nueva.
	a.copiado = true
	a.copiadoGen++
	pulsa(t, a, copiadoExpiradoMsg{gen: a.copiadoGen - 1})
	if !a.copiado {
		t.Error("un temporizador de una copia anterior no debe apagar el aviso vigente")
	}
}
