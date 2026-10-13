package render

import (
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

const (
	sgrReset = "\x1b[0m"
	// realceOn y realceOff solo encienden y apagan el video inverso, para no
	// perder los colores que la línea ya traía.
	realceOn  = "\x1b[7m"
	realceOff = "\x1b[27m"
)

// SinANSI quita los códigos de escape de un texto ya pintado.
func SinANSI(texto string) string { return codigosANSIRE.ReplaceAllString(texto, "") }

// Ancho cuenta las columnas de pantalla de un texto ya pintado.
func Ancho(texto string) int {
	columnas := 0
	for _, r := range SinANSI(texto) {
		columnas += runewidth.RuneWidth(r)
	}
	return columnas
}

// RecortarColumnas deja el prefijo que cabe en `ancho` columnas y conserva los códigos ANSI.
func RecortarColumnas(texto string, ancho int) string {
	if ancho <= 0 {
		return ""
	}
	var b strings.Builder
	columnas, i := 0, 0
	for i < len(texto) {
		if loc := codigosANSIRE.FindStringIndex(texto[i:]); loc != nil && loc[0] == 0 {
			b.WriteString(texto[i : i+loc[1]])
			i += loc[1]
			continue
		}
		r, tam := utf8.DecodeRuneInString(texto[i:])
		peso := runewidth.RuneWidth(r)
		if columnas+peso > ancho {
			break
		}
		b.WriteRune(r)
		columnas += peso
		i += tam
	}
	if i < len(texto) {
		b.WriteString(sgrReset)
	}
	return b.String()
}

// EnvolverConCursor reparte el texto en filas de `ancho` columnas y devuelve dónde cae el cursor.
func EnvolverConCursor(texto string, pos, ancho int) (lineas []string, fila, columna int) {
	runas := []rune(texto)
	pos = acotar(pos, 0, len(runas))
	if ancho <= 0 {
		return []string{texto}, 0, pos
	}
	var actual []rune
	for i := 0; i < len(runas); i++ {
		if len(actual) == ancho {
			lineas = append(lineas, string(actual))
			actual = nil
		}
		if i == pos {
			fila, columna = len(lineas), len(actual)
		}
		actual = append(actual, runas[i])
	}
	if pos == len(runas) {
		fila, columna = len(lineas), len(actual)
	}
	return append(lineas, string(actual)), fila, columna
}

// Centrar coloca un bloque en el centro de un marco; sin geometría lo devuelve tal cual.
func Centrar(bloque string, ancho, alto int) string {
	marco, _, _ := CentrarConOrigen(bloque, ancho, alto)
	return marco
}

// CentrarConOrigen coloca el bloque centrado y devuelve la fila y la columna de su
// esquina superior izquierda dentro del marco. Las zonas que el bloque dibuja
// (la caja de entrada de la bienvenida) necesitan situarse en coordenadas de pantalla.
func CentrarConOrigen(bloque string, ancho, alto int) (marco string, fila, columna int) {
	if ancho <= 0 || alto <= 0 {
		return bloque, 0, 0
	}
	cuadrado := cuadrar(bloque)
	lineas := strings.Split(cuadrado, "\n")
	fila = max((alto-len(lineas))/2, 0)
	columna = max((ancho-Ancho(lineas[0]))/2, 0)
	return lipgloss.Place(ancho, alto, lipgloss.Center, lipgloss.Center, cuadrado), fila, columna
}

// cuadrar rellena cada línea hasta el ancho del bloque: el centrado corre cada línea
// por su cuenta, y sin cuadrar antes una línea más corta se desalinea del resto.
func cuadrar(bloque string) string {
	lineas := strings.Split(bloque, "\n")
	ancho := 0
	for _, linea := range lineas {
		ancho = max(ancho, Ancho(linea))
	}
	for i, linea := range lineas {
		lineas[i] = linea + strings.Repeat(" ", max(ancho-Ancho(linea), 0))
	}
	return strings.Join(lineas, "\n")
}

// Chip es un tramo de una línea de estado que no se parte por la mitad al
// envolver: su texto ya pintado y, si procede, una clave para localizarlo.
type Chip struct {
	Texto string
	Clave string
}

// Punto es una celda local: la fila y la columna dentro de un bloque de filas.
type Punto struct {
	Fila    int
	Columna int
}

// EnvolverChips reparte los chips en filas que quepan en `ancho`, unidos por
// `separador` (que nunca queda colgando al final de una fila). Devuelve las
// filas y, por clave, dónde empieza cada chip con clave. Un chip más ancho que
// `ancho` se recorta para no desbordar: así el estado nunca pierde el ancho de
// la caja por estrecha que sea.
func EnvolverChips(chips []Chip, separador string, ancho int) (filas []string, posiciones map[string]Punto) {
	if ancho <= 0 {
		ancho = 1
	}
	posiciones = map[string]Punto{}
	sepAncho := Ancho(separador)
	var fila strings.Builder
	filaAncho := 0
	for _, chip := range chips {
		texto := chip.Texto
		if Ancho(texto) > ancho {
			texto = RecortarColumnas(texto, ancho)
		}
		anchoChip := Ancho(texto)
		columna := filaAncho
		if filaAncho > 0 {
			if filaAncho+sepAncho+anchoChip > ancho {
				filas = append(filas, fila.String())
				fila.Reset()
				filaAncho = 0
				columna = 0
			} else {
				fila.WriteString(separador)
				columna += sepAncho
			}
		}
		if chip.Clave != "" {
			posiciones[chip.Clave] = Punto{Fila: len(filas), Columna: columna}
		}
		fila.WriteString(texto)
		filaAncho = columna + anchoChip
	}
	filas = append(filas, fila.String())
	return filas, posiciones
}

// TramoColumnas devuelve el tramo [desde, hasta) de un texto, medido en columnas de pantalla.
func TramoColumnas(texto string, desde, hasta int) string {
	if hasta <= desde {
		return ""
	}
	var b strings.Builder
	columnas := 0
	for _, r := range SinANSI(texto) {
		peso := runewidth.RuneWidth(r)
		if columnas >= desde && columnas+peso <= hasta {
			b.WriteRune(r)
		}
		columnas += peso
	}
	return b.String()
}

// ResaltarTramo pone en video inverso el tramo [desde, hasta) de una línea ya pintada.
func ResaltarTramo(linea string, desde, hasta int) string {
	if desde >= hasta {
		return linea
	}
	var b strings.Builder
	columnas := 0
	realzando := false
	for i := 0; i < len(linea); {
		if loc := codigosANSIRE.FindStringIndex(linea[i:]); loc != nil && loc[0] == 0 {
			secuencia := linea[i : i+loc[1]]
			b.WriteString(secuencia)
			if (secuencia == sgrReset || secuencia == "\x1b[m") && realzando {
				b.WriteString(realceOn)
			}
			i += loc[1]
			continue
		}
		r, tam := utf8.DecodeRuneInString(linea[i:])
		dentro := columnas >= desde && columnas < hasta
		if dentro && !realzando {
			b.WriteString(realceOn)
			realzando = true
		}
		if !dentro && realzando {
			b.WriteString(realceOff)
			realzando = false
		}
		b.WriteRune(r)
		columnas += runewidth.RuneWidth(r)
		i += tam
	}
	if realzando {
		b.WriteString(realceOff)
	}
	return b.String()
}

// AlinearDerecha pega un texto al borde derecho de un marco de `ancho` columnas.
func AlinearDerecha(texto string, ancho int) string {
	relleno := ancho - Ancho(texto)
	if relleno < 0 {
		return texto
	}
	return strings.Repeat(" ", relleno) + texto
}

// Superponer coloca `encima` sobre `base` como un overlay real: cada línea de
// `encima` reemplaza las columnas [col, col+Ancho(linea)) de la línea de `base`
// en la fila correspondiente a partir de `fila`. No inserta líneas ni desplaza el
// contenido de debajo; los códigos ANSI del tramo anterior y posterior se
// conservan.
func Superponer(base, encima string, fila, col int) string {
	if encima == "" {
		return base
	}
	baseLineas := strings.Split(base, "\n")
	encimaLineas := strings.Split(encima, "\n")
	for i, linea := range encimaLineas {
		f := fila + i
		if f < 0 || f >= len(baseLineas) {
			continue
		}
		baseLineas[f] = reemplazaTramo(baseLineas[f], linea, col, Ancho(linea))
	}
	return strings.Join(baseLineas, "\n")
}

// reemplazaTramo sustituye las columnas [col, col+ancho) de `base` por `nuevo`,
// midiendo por columnas de pantalla y conservando los códigos ANSI del tramo
// anterior y posterior. Tras el overlay restaura el último SGR activo del tramo
// de `base` para no perder los colores que venían después.
func reemplazaTramo(base, nuevo string, col, ancho int) string {
	if ancho <= 0 {
		return base
	}
	if col < 0 {
		col = 0
	}
	fin := col + ancho
	var b strings.Builder
	columnas := 0
	ultimaSGR := ""
	puesto := false
	for i := 0; i < len(base); {
		if loc := codigosANSIRE.FindStringIndex(base[i:]); loc != nil && loc[0] == 0 {
			secuencia := base[i : i+loc[1]]
			if columnas < col || columnas >= fin {
				b.WriteString(secuencia)
			}
			if secuencia == sgrReset || secuencia == "\x1b[m" {
				ultimaSGR = ""
			} else {
				ultimaSGR = secuencia
			}
			i += loc[1]
			continue
		}
		r, tam := utf8.DecodeRuneInString(base[i:])
		peso := runewidth.RuneWidth(r)
		if columnas >= col && !puesto {
			b.WriteString(nuevo)
			if ultimaSGR != "" {
				b.WriteString(ultimaSGR)
			}
			puesto = true
		}
		if columnas < col || columnas >= fin {
			b.WriteRune(r)
		}
		columnas += peso
		i += tam
	}
	if !puesto {
		for columnas < col {
			b.WriteRune(' ')
			columnas++
		}
		b.WriteString(nuevo)
	}
	return b.String()
}

// EnvolverTexto reparte un texto en filas de `ancho` columnas, respetando sus
// saltos de línea. Es una función pura que sirve para avisos y errores largos.
func EnvolverTexto(texto string, ancho int) []string {
	if ancho <= 0 {
		return []string{texto}
	}
	var lineas []string
	for _, parrafo := range strings.Split(texto, "\n") {
		envueltas, _, _ := EnvolverConCursor(parrafo, 0, ancho)
		lineas = append(lineas, envueltas...)
	}
	return lineas
}

func acotar(valor, minimo, maximo int) int {
	if valor < minimo {
		return minimo
	}
	if valor > maximo {
		return maximo
	}
	return valor
}
