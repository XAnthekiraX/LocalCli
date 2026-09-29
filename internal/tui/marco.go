// marco.go — la capa común de render: cada componente del layout (chat, panel,
// caja de entrada, modales, cabecera y pie) describe su contenido, sus
// dimensiones y su fondo, y aquí se convierte en una superficie rectangular
// continua: exactamente `Alto` filas y cada una con `Ancho` celdas, todas con
// fondo explícito.
//
// El motivo de que exista esta capa y no un pintado por componente es la
// selección con el ratón: la terminal selecciona celdas, y una celda que no se
// pinta no pertenece a la superficie (aunque visualmente parezca vacía). Al
// rellenar cada fila y reafirmar el fondo tras cada reset de ANSI, los espacios
// del padding y del fondo dejan de ser «nada» y pasan a ser celdas reales, de
// modo que el arrastre cubre el área completa y la copia trae solo el texto.
//
// No se introduce un búfer de celdas propio: el paquete entero trabaja con
// cadenas anotadas con ANSI y ya tiene los helpers para recorrerlas
// (`codigosANSIRE`, `recortarColumnas`, `runewidth`). Un `Cell`/`Frame`
// paralelo duplicaría esa lógica y rompería la selección y la copia ya
// existentes. El «marco» es, por tanto, la cadena normalizada.
package tui

import (
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

// marcaFondo es un carácter de uso privado que no aparece en la interfaz: sirve
// para extraer de un estilo la secuencia ANSI que enciende su fondo (ver
// codigoFondo).
const marcaFondo = "\uE000"

// codigoFondo devuelve la secuencia ANSI que enciende el fondo `c` con el perfil
// de color vigente, o vacía si el perfil no pinta color. Se obtiene renderizando
// una marca con el estilo y recortando todo lo anterior a la marca, que es
// exactamente la secuencia de encendido; así no depende de cómo se llame el
// método que expone cada versión de la librería.
func codigoFondo(c lipgloss.Color) string {
	conFondo := lipgloss.NewStyle().Background(c).Render(marcaFondo)
	i := strings.Index(conFondo, marcaFondo)
	if i < 0 {
		return ""
	}
	return conFondo[:i]
}

// pintarFondo convierte un bloque en una superficie con fondo: deja `fondo` en
// todas sus celdas y rellena cada línea hasta `ancho` columnas. Reafirma el fondo
// tras cada reset para que un `\x1b[0m` interno (el final de cualquier tramo con
// color) no deje el resto de la línea sin fondo. No recorta: una línea que ya
// pase de `ancho` se deja tal cual (p. ej. el logotipo canónico en una terminal
// diminuta), y quien necesite ancho exacto usa `recortarColumnas` antes.
func pintarFondo(bloque string, ancho int, fondo lipgloss.Color) string {
	if ancho <= 0 {
		return bloque
	}
	codigo := codigoFondo(fondo)
	lineas := strings.Split(bloque, "\n")
	for i, l := range lineas {
		lineas[i] = pintarLineaFondo(l, ancho, codigo)
	}
	return strings.Join(lineas, "\n")
}

// pintarLineaFondo pinta una sola línea: enciende el fondo, conserva los códigos
// ANSI que traiga (reafirmando el fondo tras los resets) y rellena con espacios
// hasta `ancho` columnas de pantalla. Los caracteres anchos cuentan doble.
func pintarLineaFondo(linea string, ancho int, codigo string) string {
	var b strings.Builder
	b.WriteString(codigo)
	col := 0
	for i := 0; i < len(linea); {
		if loc := codigosANSIRE.FindStringIndex(linea[i:]); loc != nil && loc[0] == 0 {
			sec := linea[i : i+loc[1]]
			b.WriteString(sec)
			// Un reset apaga el fondo del resto de la línea: se vuelve a
			// encender para que la superficie no tenga agujeros.
			if sec == sgrReset || sec == "\x1b[m" {
				b.WriteString(codigo)
			}
			i += loc[1]
			continue
		}
		r, size := utf8.DecodeRuneInString(linea[i:])
		b.WriteRune(r)
		col += runewidth.RuneWidth(r)
		i += size
	}
	for ; col < ancho; col++ {
		b.WriteByte(' ')
	}
	b.WriteString(sgrReset)
	return b.String()
}

// marcoCompleto normaliza el marco de la vista a exactamente `ancho`×`alto`
// celdas: rellena cada línea hasta `ancho` con el fondo base y ajusta el número
// de filas. Si sobran filas conserva las últimas —igual que el renderer de
// Bubble Tea, que descarta las de arriba cuando el marco no cabe—; si faltan,
// añade filas de fondo. Sin geometría conocida (antes de la primera
// `WindowSizeMsg`) devuelve el bloque tal cual: las vistas que se pintan sin
// terminal no cambian.
func marcoCompleto(bloque string, ancho, alto int) string {
	if ancho <= 0 || alto <= 0 {
		return bloque
	}
	lineas := strings.Split(bloque, "\n")
	if len(lineas) > alto {
		lineas = lineas[len(lineas)-alto:]
	}
	salida := make([]string, 0, alto)
	for _, l := range lineas {
		salida = append(salida, pintarFondo(l, ancho, fondoApp))
	}
	relleno := pintarFondo("", ancho, fondoApp)
	for len(salida) < alto {
		salida = append(salida, relleno)
	}
	return strings.Join(salida, "\n")
}
