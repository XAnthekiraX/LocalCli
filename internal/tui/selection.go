// selection.go — selección de texto con el ratón y copia al portapapeles.
//
// La aplicación captura el ratón (main.go, `tea.WithMouseCellMotion`), así que
// puede saber qué se arrastra: al soltar, el texto seleccionado del último marco
// pintado se copia al portapapeles. La copia intenta primero las herramientas del
// sistema (wl-copy, xclip, xsel, pbcopy) y, si no hay ninguna, usa OSC 52, que
// pide a la terminal que lo copie.
//
// Nota: al capturar el ratón, la selección nativa de la terminal deja de
// funcionar salvo que se mantenga Shift (comportamiento habitual de cualquier
// TUI). La rueda del ratón también se atiende aquí para desplazar el chat.
package tui

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattn/go-runewidth"
)

// posicion es una celda de la pantalla: columna (X) y fila (Y), en base 0.
type posicion struct{ X, Y int }

// raton atiende los eventos del ratón: la rueda desplaza el chat y el arrastre
// con el botón izquierdo selecciona texto, que al soltar se copia.
func (a *App) raton(m tea.MouseMsg) (tea.Model, tea.Cmd) {
	switch {
	case m.Button == tea.MouseButtonWheelUp:
		if !a.modalAbierto() {
			a.desplazarChat(-3)
		}
		return a, nil
	case m.Button == tea.MouseButtonWheelDown:
		if !a.modalAbierto() {
			a.desplazarChat(3)
		}
		return a, nil
	case m.Action == tea.MouseActionPress && m.Button == tea.MouseButtonLeft:
		a.ratonSelec = true
		a.ratonIni = posicion{X: m.X, Y: m.Y}
		a.ratonFin = a.ratonIni
		a.iniEnChat = a.enChat(m.X, m.Y)
		a.finEnChat = a.iniEnChat
		return a, nil
	case m.Action == tea.MouseActionMotion && a.ratonSelec:
		a.ratonFin = posicion{X: m.X, Y: m.Y}
		a.finEnChat = a.enChat(m.X, m.Y)
		return a, nil
	case m.Action == tea.MouseActionRelease && a.ratonSelec:
		a.ratonSelec = false
		fin := posicion{X: m.X, Y: m.Y}
		a.finEnChat = a.enChat(fin.X, fin.Y)
		// Un clic (pulsar y soltar sin arrastrar) sobre «aprobar» o «declinar»
		// de una fila de aprobaciones resuelve ESA aprobación. Un arrastre sigue
		// siendo una selección de texto.
		if a.ratonIni == fin {
			if ap, aprobar, ok := a.decisionEnCelda(fin); ok {
				a.limpiarSeleccion()
				return a, a.decidirAprobacion(ap, aprobar)
			}
		}
		a.ratonFin = fin
		ini, finSel := a.limitarSeleccion(a.ratonIni, a.ratonFin)
		texto := textoSeleccionado(a.ultimaVista, ini, finSel)
		// Al soltar termina la selección: el realce desaparece (solo vivía en el
		// arrastre). Si había algo, se copia y se avisa.
		a.limpiarSeleccion()
		if strings.TrimSpace(texto) == "" {
			return a, nil
		}
		// La copia corre en el hilo del bucle (síncrona): así no se entremezcla
		// con el pintado de la siguiente pantalla.
		copiarFunc(texto)
		return a, a.avisarCopiado()
	}
	return a, nil
}

// limitarSeleccion recorta la selección a la zona donde empezó: la columna del
// chat o la del sidebar. Un arrastre no debe mezclar las dos —seleccionar el
// chat y llevarse el sidebar de paso, o al revés—, así que el extremo que se
// salga de su columna se pega a su borde. La separación (dos columnas) no es de
// ninguna de las dos.
func (a *App) limitarSeleccion(ini, fin posicion) (posicion, posicion) {
	if !a.Panel.Abierto || a.Ancho <= 0 {
		return ini, fin
	}
	col := a.anchoColumna()
	esChat := ini.X < col
	limitar := func(p posicion) posicion {
		if esChat {
			if p.X >= col {
				p.X = col - 1
			}
			if p.X < 0 {
				p.X = 0
			}
			return p
		}
		inicio := col + 2
		if p.X < inicio {
			p.X = inicio
		}
		if p.X >= a.Ancho {
			p.X = a.Ancho - 1
		}
		return p
	}
	return limitar(ini), limitar(fin)
}

// limpiarSeleccion descarta la selección vigente (extremos a cero). Se llama
// cuando una tecla abre una interacción nueva —abrir un modal, cambiar de
// vista—: el realce no debe quedar pegado sobre lo que venga después. La rueda
// no la usa: desplazar nunca cancela la selección (desplazarChat).
func (a *App) limpiarSeleccion() {
	a.ratonIni, a.ratonFin = posicion{}, posicion{}
	a.iniEnChat, a.finEnChat = false, false
}

// enChat dice si una celda de pantalla cae sobre la ventana visible del
// historial (chatFilaIni..chatFilaFin, fijados al pintar en app.go) Y en la
// columna principal: el sidebar comparte filas con el chat, así que una celda
// suya no cuenta como del chat a la hora de reanclar. Fuera de la vista
// principal la banda vale 0 y ningún extremo cuenta como del chat.
func (a *App) enChat(x, y int) bool {
	if a.Panel.Abierto && x >= a.anchoColumna() {
		return false
	}
	return y >= a.chatFilaIni && y < a.chatFilaFin
}

// cabecera vale 1 cuando el historial va precedido de la línea «↑ N líneas
// arriba» —que ocupa una fila antes de la ventana— y 0 cuando no.
func cabecera(ocultasArriba int) int {
	if ocultasArriba > 0 {
		return 1
	}
	return 0
}

// desplazarChat mueve el historial n líneas (n<0 sube, n>0 baja) y reancla la
// selección al texto en vez de dejarla clavada a la pantalla: al cambiar el
// offset, las mismas coordenadas señalarían otro texto. Cada extremo que vive
// en el chat se corrige por el desplazamiento real en filas de pantalla —que es
// `cabecera(oa) - oa + contenido`—, de modo que la rueda puede usarse mientras
// se selecciona sin que la selección se cancele ni quede desalineada. Durante
// el arrastre se reancla solo el ancla (el puntero sigue en su celda del ratón,
// para poder seguir seleccionando); con la selección ya soltada se reanclan los
// dos extremos.
func (a *App) desplazarChat(n int) {
	if n == 0 {
		return
	}
	oaAntes := a.Chat.OcultasArriba()
	if n < 0 {
		a.Chat.Subir(-n)
	} else {
		a.Chat.Bajar(n)
	}
	oaDespues := a.Chat.OcultasArriba()
	delta := cabecera(oaDespues) - cabecera(oaAntes) - (oaDespues - oaAntes)
	if delta == 0 {
		return
	}
	if a.iniEnChat {
		a.ratonIni.Y += delta
	}
	if !a.ratonSelec && a.finEnChat {
		a.ratonFin.Y += delta
	}
}

// decisionEnCelda mira si una celda del último marco pintado cae sobre la
// palabra «aprobar» o «declinar» de una fila del panel de aprobaciones, y
// devuelve a qué aprobación pertenece y qué decisión toca. Se apoya en el texto
// pintado —el panel no expone su geometría—: localiza la fila por el formato
// documentado (`FilaDe`) y cuenta columnas en runas desde el marcador de
// selección de dos columnas. Con el panel cerrado no hay filas que pulsar: la
// línea de aviso no es interactiva.
func (a *App) decisionEnCelda(pos posicion) (Aprobacion, bool, bool) {
	if !a.Aprobs.Abierto || a.ultimaVista == "" {
		return Aprobacion{}, false, false
	}
	lineas := strings.Split(a.ultimaVista, "\n")
	if pos.Y < 0 || pos.Y >= len(lineas) {
		return Aprobacion{}, false, false
	}
	cuerpo := []rune(sinANSI(lineas[pos.Y]))
	// Con el sidebar abierto, la fila trae también su texto a la derecha: se
	// recorta a la columna principal para comparar con la fila de la aprobación.
	if col := a.anchoColumna(); col > 0 && len(cuerpo) > col {
		cuerpo = cuerpo[:col]
	}
	if len(cuerpo) < 2 {
		return Aprobacion{}, false, false
	}
	// Sin el marcador («› » o «  ») y sin el relleno con que la columna principal
	// completa la fila hasta el divisor.
	texto := strings.TrimRight(string(cuerpo[2:]), " ")
	x := pos.X - 2
	for _, it := range a.Aprobs.Items {
		if it.Obsoleta || FilaDe(it) != texto {
			continue
		}
		prefijo := len([]rune(it.Sesion + " | " + it.Descripcion + " | "))
		if x >= prefijo && x < prefijo+len([]rune("aprobar")) {
			return it, true, true
		}
		colDeclinar := prefijo + len([]rune("aprobar | "))
		if x >= colDeclinar && x < colDeclinar+len([]rune("declinar")) {
			return it, false, true
		}
		return Aprobacion{}, false, false
	}
	return Aprobacion{}, false, false
}

// codigosANSIRE reconoce los códigos de color SGR; se quitan para contar
// columnas sobre el texto visible.
var codigosANSIRE = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func sinANSI(s string) string { return codigosANSIRE.ReplaceAllString(s, "") }

// indiceColumna devuelve el índice de runa donde empieza la columna `col` de una
// línea ya sin códigos ANSI, contando el ancho de pantalla (un emoji ocupa dos
// columnas). Es lo que traduce la X del ratón a una posición dentro de la línea.
func indiceColumna(r []rune, col int) int {
	w := 0
	for i, ch := range r {
		if w >= col {
			return i
		}
		w += runewidth.RuneWidth(ch)
	}
	return len(r)
}

// textoSeleccionado extrae de un marco pintado el texto que va de una celda a
// otra. Los extremos se ordenan; una selección de una sola celda (un clic sin
// arrastrar) no produce texto. Las columnas se cuentan sobre el texto visible,
// sin códigos de color y por ancho de pantalla.
func textoSeleccionado(vista string, ini, fin posicion) string {
	if vista == "" {
		return ""
	}
	if fin.Y < ini.Y || (fin.Y == ini.Y && fin.X < ini.X) {
		ini, fin = fin, ini
	}
	lineas := strings.Split(vista, "\n")
	if ini.Y < 0 {
		ini.Y = 0
	}
	if fin.Y >= len(lineas) {
		fin.Y = len(lineas) - 1
	}
	if ini.Y >= len(lineas) || fin.Y < ini.Y {
		return ""
	}

	var out []string
	for y := ini.Y; y <= fin.Y; y++ {
		r := []rune(sinANSI(lineas[y]))
		desde, hasta := 0, len(r)
		if y == ini.Y {
			desde = indiceColumna(r, ini.X)
		}
		if y == fin.Y {
			hasta = indiceColumna(r, fin.X)
		}
		if desde < 0 {
			desde = 0
		}
		if hasta > len(r) {
			hasta = len(r)
		}
		if desde > hasta {
			desde = hasta
		}
		out = append(out, strings.TrimRight(string(r[desde:hasta]), " "))
	}
	// El salto final sobra: se quitan las líneas vacías del final.
	for len(out) > 0 && strings.TrimSpace(out[len(out)-1]) == "" {
		out = out[:len(out)-1]
	}
	return strings.Join(out, "\n")
}

// resaltarSeleccion envuelve en video inverso el tramo que va de una celda a
// otra del marco pintado: es el realce que dice qué se está seleccionando. No
// altera el texto visible (los códigos de color se conservan; solo se añade el
// inverso), así que el texto que se copia no cambia. Los extremos se ordenan
// igual que en textoSeleccionado y las columnas se cuentan en runas sobre el
// texto sin códigos. Un extremo fuera de pantalla —el ancla puede salirse del
// marco al hacer scroll— cuenta como el borde de la línea.
func resaltarSeleccion(vista string, ini, fin posicion) string {
	if vista == "" {
		return vista
	}
	if fin.Y < ini.Y || (fin.Y == ini.Y && fin.X < ini.X) {
		ini, fin = fin, ini
	}
	lineas := strings.Split(vista, "\n")
	for y := ini.Y; y <= fin.Y; y++ {
		if y < 0 || y >= len(lineas) {
			continue
		}
		n := runewidth.StringWidth(sinANSI(lineas[y]))
		x0, x1 := 0, n
		if y == ini.Y {
			x0 = ini.X
		}
		if y == fin.Y {
			x1 = fin.X
		}
		if x0 < 0 {
			x0 = 0
		}
		if x1 > n {
			x1 = n
		}
		if x0 >= x1 {
			continue
		}
		lineas[y] = resaltarTramo(lineas[y], x0, x1)
	}
	return strings.Join(lineas, "\n")
}

// resaltarTramo aplica el video inverso a las columnas [x0,x1) de una línea ya
// pintada, que puede traer códigos ANSI. Cuenta columnas por runas visibles
// (los escapes no ocupan columna) y reafirma el inverso tras cada secuencia de
// escape dentro del tramo, para que un `\x1b[0m` intermedio no lo apague.
func resaltarTramo(linea string, x0, x1 int) string {
	var b strings.Builder
	col := 0
	dentro := false
	for i := 0; i < len(linea); {
		if loc := codigosANSIRE.FindStringIndex(linea[i:]); loc != nil && loc[0] == 0 {
			b.WriteString(linea[i : i+loc[1]])
			if dentro {
				b.WriteString(seleccionOn)
			}
			i += loc[1]
			continue
		}
		r, size := utf8.DecodeRuneInString(linea[i:])
		w := runewidth.RuneWidth(r)
		nuevo := col >= x0 && col < x1
		if nuevo != dentro {
			if nuevo {
				b.WriteString(seleccionOn)
			} else {
				b.WriteString(seleccionOff)
			}
			dentro = nuevo
		}
		b.WriteRune(r)
		col += w
		i += size
	}
	if dentro {
		b.WriteString(seleccionOff)
	}
	return b.String()
}

// superponerDerecha pega `texto` al borde derecho de la primera línea del marco,
// pisando lo que hubiera en esas columnas. Es lo que sitúa el aviso transitorio
// [Copiado] arriba a la derecha. No cambia el resto del marco ni ocupa una línea
// nueva (así la disposición no salta). Sin ancho conocido no pinta nada.
func superponerDerecha(vista, texto string, ancho int) string {
	if vista == "" || ancho <= 0 {
		return vista
	}
	anchoTexto := len([]rune(sinANSI(texto)))
	if anchoTexto >= ancho {
		lineas := strings.Split(vista, "\n")
		lineas[0] = texto
		return strings.Join(lineas, "\n")
	}
	lineas := strings.Split(vista, "\n")
	// El relleno hasta el borde con el fondo de la app: el aviso no debe dejar
	// un hueco sin pintar en la primera fila. `pintarFondo` cierra con un reset,
	// así que el aviso no hereda el color de lo que tuviera debajo.
	lineas[0] = pintarFondo(recortarColumnas(lineas[0], ancho-anchoTexto), ancho-anchoTexto, fondoApp) + texto
	return strings.Join(lineas, "\n")
}

// recortarColumnas deja la línea con exactamente n columnas visibles de ancho de
// pantalla: copia lo que quepa (sin cortar códigos de color ni partir un carácter
// ancho) y rellena con espacios lo que falte. Cierra cualquier atributo abierto
// antes del relleno.
func recortarColumnas(linea string, n int) string {
	if n <= 0 {
		return ""
	}
	var b strings.Builder
	col := 0
	for i := 0; i < len(linea) && col < n; {
		if loc := codigosANSIRE.FindStringIndex(linea[i:]); loc != nil && loc[0] == 0 {
			b.WriteString(linea[i : i+loc[1]])
			i += loc[1]
			continue
		}
		r, size := utf8.DecodeRuneInString(linea[i:])
		w := runewidth.RuneWidth(r)
		if col+w > n {
			// El carácter ancho no cabe entero: mejor dejarlo fuera que partir
			// la columna.
			break
		}
		b.WriteRune(r)
		col += w
		i += size
	}
	if col > 0 {
		b.WriteString(sgrReset)
	}
	for ; col < n; col++ {
		b.WriteByte(' ')
	}
	return b.String()
}

// duracionCopiado es lo que el aviso [Copiado] permanece en pantalla antes de
// apagarse solo.
const duracionCopiado = 1500 * time.Millisecond

// copiadoExpiradoMsg apaga el aviso [Copiado] cuando vence su tiempo. `gen`
// distingue la copia que lo encendió: un aviso nuevo no lo apaga el temporizador
// de uno viejo.
type copiadoExpiradoMsg struct{ gen uint64 }

// avisarCopiado enciende el aviso [Copiado] arriba a la derecha y programa su
// apagado. Una copia nueva reemplaza la generación vigente, así que su
// temporizador no apaga antes de tiempo el aviso recién encendido.
func (a *App) avisarCopiado() tea.Cmd {
	a.copiado = true
	a.copiadoGen++
	gen := a.copiadoGen
	return tea.Tick(duracionCopiado, func(time.Time) tea.Msg { return copiadoExpiradoMsg{gen: gen} })
}

// copiarFunc es la copia efectiva; variable para poder sustituirla en las
// pruebas y no tocar el portapapeles real.
var copiarFunc = copiarAlPortapapeles

// copiarAlPortapapeles intenta las herramientas del sistema y, si ninguna está
// disponible, recurre a OSC 52.
func copiarAlPortapapeles(texto string) {
	for _, c := range [][]string{
		{"wl-copy"},
		{"xclip", "-selection", "clipboard"},
		{"xsel", "--clipboard", "--input"},
		{"pbcopy"},
	} {
		if _, err := exec.LookPath(c[0]); err != nil {
			continue
		}
		cmd := exec.Command(c[0], c[1:]...)
		cmd.Stdin = strings.NewReader(texto)
		if err := cmd.Run(); err == nil {
			return
		}
	}
	// OSC 52: la terminal copia lo que va en base64. Funciona también por SSH.
	_, _ = fmt.Fprint(os.Stdout, "\x1b]52;c;"+base64.StdEncoding.EncodeToString([]byte(texto))+"\x07")
}
