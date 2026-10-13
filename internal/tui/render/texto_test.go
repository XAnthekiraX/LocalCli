package render

import (
	"strings"
	"testing"
)

// SinANSI y Ancho miden lo que se ve, no lo que se escribe: los códigos de color
// no cuentan como columnas.
func TestAnchoIgnoraLosCodigosYCuentaColumnas(t *testing.T) {
	casos := []struct {
		texto  string
		quiero int
	}{
		{"hola", 4},
		{"\x1b[1mhola\x1b[0m", 4},
		{"", 0},
		{"日本", 4},
		{"a\x1b[38;5;12mb\x1b[0m", 2},
	}
	for _, c := range casos {
		if ancho := Ancho(c.texto); ancho != c.quiero {
			t.Errorf("Ancho(%q) = %d, quiero %d", c.texto, ancho, c.quiero)
		}
		if sin := SinANSI(c.texto); strings.Contains(sin, "\x1b") {
			t.Errorf("SinANSI(%q) dejó códigos: %q", c.texto, sin)
		}
	}
}

// RecortarColumnas deja el prefijo que cabe y no parte un carácter ancho por la
// mitad.
func TestRecortarColumnasNoParteCaracteresAnchos(t *testing.T) {
	if got := SinANSI(RecortarColumnas("hola mundo", 4)); got != "hola" {
		t.Errorf("recorte a 4 columnas = %q, quiero %q", got, "hola")
	}
	if got := SinANSI(RecortarColumnas("日本", 3)); got != "日" {
		t.Errorf("recorte a 3 columnas = %q, quiero %q", got, "日")
	}
	if got := RecortarColumnas("hola", 0); got != "" {
		t.Errorf("recorte a 0 columnas = %q, quiero vacío", got)
	}
	// Un texto que no cabe cierra los códigos que traía abiertos.
	if got := RecortarColumnas("\x1b[31mhola\x1b[0m", 2); !strings.HasSuffix(got, sgrReset) {
		t.Errorf("el recorte no cerró los códigos: %q", got)
	}
}

// EnvolverConCursor reparte el texto en filas del ancho dado y dice dónde quedó
// el cursor.
func TestEnvolverConCursorReparteYDiceDondeEstaElCursor(t *testing.T) {
	lineas, fila, columna := EnvolverConCursor("abcdef", 4, 3)
	if len(lineas) != 2 || lineas[0] != "abc" || lineas[1] != "def" {
		t.Fatalf("reparto = %q, quiero [abc def]", lineas)
	}
	if fila != 1 || columna != 1 {
		t.Fatalf("el cursor quedó en (%d,%d), quiero (1,1)", fila, columna)
	}
	// Al final del texto el cursor cae tras la última runa.
	if _, fila, columna = EnvolverConCursor("abcdef", 6, 3); fila != 1 || columna != 3 {
		t.Fatalf("el cursor al final quedó en (%d,%d), quiero (1,3)", fila, columna)
	}
	// Un texto corto es una sola fila.
	if lineas, fila, _ = EnvolverConCursor("ab", 1, 10); len(lineas) != 1 || fila != 0 {
		t.Fatalf("texto corto = %q fila %d, quiero una fila", lineas, fila)
	}
}

// TramoColumnas devuelve el tramo pedido, medido en columnas de pantalla.
func TestTramoColumnasDevuelveElTramoPedido(t *testing.T) {
	if got := TramoColumnas("hola mundo", 2, 6); got != "la m" {
		t.Errorf("tramo [2,6) = %q, quiero %q", got, "la m")
	}
	if got := TramoColumnas("hola", 3, 1); got != "" {
		t.Errorf("un tramo vacío debería ser vacío: %q", got)
	}
}

// ResaltarTramo pone en video inverso solo el tramo pedido y no pierde el texto.
func TestResaltarTramoMarcaSoloElTramo(t *testing.T) {
	pintado := ResaltarTramo("hola", 1, 3)
	if SinANSI(pintado) != "hola" {
		t.Fatalf("el realce cambió el texto: %q", pintado)
	}
	if !strings.Contains(pintado, realceOn) || !strings.Contains(pintado, realceOff) {
		t.Fatalf("el realce no encendió y apagó el video inverso: %q", pintado)
	}
	if pintado := ResaltarTramo("hola", 2, 2); pintado != "hola" {
		t.Fatalf("un tramo vacío no debería tocar la línea: %q", pintado)
	}
}

// AlinearDerecha pega el texto al borde derecho del ancho dado.
func TestAlinearDerechaPegaAlBorde(t *testing.T) {
	if got := AlinearDerecha("hola", 10); Ancho(got) != 10 || !strings.HasSuffix(got, "hola") {
		t.Errorf("alineado = %q", got)
	}
	// Un texto más ancho que el marco se devuelve tal cual: nunca se recorta.
	if got := AlinearDerecha("holaholahola", 4); got != "holaholahola" {
		t.Errorf("alineado demasiado ancho = %q", got)
	}
}

// CentrarConOrigen deja el bloque en el centro y dice dónde cayó su esquina.
func TestCentrarConOrigenColocaElBloqueYSuOrigen(t *testing.T) {
	marco, fila, columna := CentrarConOrigen("ab\ncd", 10, 5)
	lineas := strings.Split(marco, "\n")
	if len(lineas) != 5 {
		t.Fatalf("el marco tiene %d filas, quiero 5", len(lineas))
	}
	for i, linea := range lineas {
		if Ancho(linea) != 10 {
			t.Fatalf("la fila %d mide %d columnas, quiero 10", i, Ancho(linea))
		}
	}
	if fila != 1 || columna != 4 {
		t.Fatalf("el origen del bloque fue (%d,%d), quiero (1,4)", fila, columna)
	}
	if got := SinANSI(lineas[fila]); !strings.HasPrefix(got, "    ab") {
		t.Fatalf("el bloque no quedó centrado: %q", got)
	}
	// Sin geometría no hay dónde centrar y el bloque se devuelve tal cual.
	if marco, _, _ := CentrarConOrigen("ab", 0, 0); marco != "ab" {
		t.Errorf("sin geometría el bloque cambió: %q", marco)
	}
}

// Superponer reemplaza por columnas de pantalla sin insertar líneas: conserva el
// número de filas, respeta el ancho total y deja intacto el tramo posterior
// aunque la base lleve códigos ANSI (el caso que rompía el overlay).
func TestSuperponerReemplazaColumnasSinEmpujar(t *testing.T) {
	reset := "\x1b[0m"
	base := ""
	for i := 0; i < 3; i++ {
		base += "\x1b[41m" + strings.Repeat("x", 20) + reset + "\n"
	}
	base = strings.TrimRight(base, "\n")

	encima := "\x1b[42m" + strings.Repeat("#", 6) + reset

	out := Superponer(base, encima, 1, 4)
	lineas := strings.Split(out, "\n")
	if len(lineas) != 3 {
		t.Fatalf("Superponer cambió el número de filas: %d, quiero 3", len(lineas))
	}
	for i, l := range lineas {
		if Ancho(l) != 20 {
			t.Fatalf("la fila %d mide %d columnas, quiero 20", i, Ancho(l))
		}
	}
	// La fila 0 no se toca; la fila 1 reemplaza las columnas 4..10; la 2 tampoco.
	if got := SinANSI(lineas[0]); got != strings.Repeat("x", 20) {
		t.Errorf("la fila 0 cambió: %q", got)
	}
	if got := SinANSI(lineas[1]); got != "xxxx"+strings.Repeat("#", 6)+strings.Repeat("x", 10) {
		t.Errorf("la fila 1 no superpuso en sitio: %q", got)
	}
	if got := SinANSI(lineas[2]); got != strings.Repeat("x", 20) {
		t.Errorf("la fila 2 cambió: %q", got)
	}
}

// Con una base más corta que el alto pedido, Superponer no inventa filas ni
// desborda: ignora las líneas que caen fuera.
func TestSuperponerIgnoraFilasFueraDelMarco(t *testing.T) {
	base := "aaaa\nbbbb"
	encima := "##"
	out := Superponer(base, encima, 5, 0)
	if out != base {
		t.Errorf("Superponer fuera de rango alteró la base: %q", out)
	}
}
