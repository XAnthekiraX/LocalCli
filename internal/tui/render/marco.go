package render

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

// codigosANSIRE reconoce una secuencia SGR de color, negrita o reset.
var codigosANSIRE = regexp.MustCompile("\x1b\\[[0-9;]*m")

// marcaFondo es un carácter que no se pinta: sirve para extraer la secuencia que enciende un fondo.
const marcaFondo = "\uE000"

func codigoFondo(fondo lipgloss.Color) string {
	pintado := lipgloss.NewStyle().Background(fondo).Render(marcaFondo)
	i := strings.Index(pintado, marcaFondo)
	if i < 0 {
		return ""
	}
	return pintado[:i]
}

// PintarFondo deja `fondo` en todas las celdas del bloque y rellena cada fila hasta `ancho`.
func PintarFondo(bloque string, ancho int, fondo lipgloss.Color) string {
	if ancho <= 0 {
		return bloque
	}
	codigo := codigoFondo(fondo)
	lineas := strings.Split(bloque, "\n")
	for i, linea := range lineas {
		lineas[i] = pintarLinea(linea, ancho, codigo)
	}
	return strings.Join(lineas, "\n")
}

func pintarLinea(linea string, ancho int, codigo string) string {
	var b strings.Builder
	b.WriteString(codigo)
	columnas := 0
	for i := 0; i < len(linea); {
		if loc := codigosANSIRE.FindStringIndex(linea[i:]); loc != nil && loc[0] == 0 {
			secuencia := linea[i : i+loc[1]]
			b.WriteString(secuencia)
			if secuencia == sgrReset || secuencia == "\x1b[m" {
				b.WriteString(codigo)
			}
			i += loc[1]
			continue
		}
		r, tam := utf8.DecodeRuneInString(linea[i:])
		b.WriteRune(r)
		columnas += runewidth.RuneWidth(r)
		i += tam
	}
	b.WriteString(strings.Repeat(" ", max(ancho-columnas, 0)))
	b.WriteString(sgrReset)
	return b.String()
}

// MarcoCompleto normaliza la vista a `ancho`×`alto` celdas con el fondo de la pantalla.
func MarcoCompleto(bloque string, ancho, alto int) string {
	if ancho <= 0 || alto <= 0 {
		return bloque
	}
	lineas := strings.Split(bloque, "\n")
	if len(lineas) > alto {
		lineas = lineas[len(lineas)-alto:]
	}
	filas := make([]string, 0, alto)
	for _, linea := range lineas {
		filas = append(filas, PintarFondo(RecortarColumnas(linea, ancho), ancho, FondoPantalla))
	}
	relleno := PintarFondo("", ancho, FondoPantalla)
	for len(filas) < alto {
		filas = append(filas, relleno)
	}
	return strings.Join(filas, "\n")
}
