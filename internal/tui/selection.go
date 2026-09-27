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

	tea "github.com/charmbracelet/bubbletea"
)

// posicion es una celda de la pantalla: columna (X) y fila (Y), en base 0.
type posicion struct{ X, Y int }

// raton atiende los eventos del ratón: la rueda desplaza el chat y el arrastre
// con el botón izquierdo selecciona texto, que al soltar se copia.
func (a *App) raton(m tea.MouseMsg) (tea.Model, tea.Cmd) {
	switch {
	case m.Button == tea.MouseButtonWheelUp:
		if !a.modalAbierto() {
			a.Chat.Subir(3)
		}
		return a, nil
	case m.Button == tea.MouseButtonWheelDown:
		if !a.modalAbierto() {
			a.Chat.Bajar(3)
		}
		return a, nil
	case m.Action == tea.MouseActionPress && m.Button == tea.MouseButtonLeft:
		a.ratonSelec = true
		a.ratonIni = posicion{X: m.X, Y: m.Y}
		a.ratonFin = a.ratonIni
		return a, nil
	case m.Action == tea.MouseActionMotion && a.ratonSelec:
		a.ratonFin = posicion{X: m.X, Y: m.Y}
		return a, nil
	case m.Action == tea.MouseActionRelease && a.ratonSelec:
		a.ratonSelec = false
		fin := posicion{X: m.X, Y: m.Y}
		// Un clic (pulsar y soltar sin arrastrar) sobre «aprobar» o «declinar»
		// de una fila de aprobaciones resuelve ESA aprobación. Un arrastre sigue
		// siendo una selección de texto.
		if a.ratonIni == fin {
			if ap, aprobar, ok := a.decisionEnCelda(fin); ok {
				a.ratonIni, a.ratonFin = posicion{}, posicion{}
				return a, a.decidirAprobacion(ap, aprobar)
			}
		}
		a.ratonFin = fin
		if texto := textoSeleccionado(a.ultimaVista, a.ratonIni, a.ratonFin); strings.TrimSpace(texto) != "" {
			// La copia corre en el hilo del bucle (síncrona): así no se
			// entremezcla con el pintado de la siguiente pantalla.
			copiarFunc(texto)
		}
		return a, nil
	}
	return a, nil
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
	if len(cuerpo) < 2 {
		return Aprobacion{}, false, false
	}
	texto := string(cuerpo[2:]) // sin el marcador («› » o «  »)
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

// textoSeleccionado extrae de un marco pintado el texto que va de una celda a
// otra. Los extremos se ordenan; una selección de una sola celda (un clic sin
// arrastrar) no produce texto. Las columnas se cuentan sobre el texto visible,
// sin códigos de color.
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
			desde = ini.X
		}
		if y == fin.Y {
			hasta = fin.X
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
