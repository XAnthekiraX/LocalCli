package render

import (
	"strings"
	"testing"
)

// PintarFondo rellena cada fila hasta el ancho pedido y la deja con el fondo: es
// lo que convierte la pantalla en una superficie rectangular continua.
func TestPintarFondoRellenaHastaElAncho(t *testing.T) {
	pintado := PintarFondo("ab", 4, FondoPantalla)
	if got := SinANSI(pintado); got != "ab  " {
		t.Errorf("el relleno = %q, quiero %q", got, "ab  ")
	}
	if !strings.Contains(pintado, "\x1b[") || Ancho(pintado) != 4 {
		t.Errorf("la fila no quedó pintada a 4 columnas: %q", pintado)
	}
	// Un ancho nulo no toca el bloque: sin geometría no hay nada que rellenar.
	if got := PintarFondo("ab", 0, FondoPantalla); got != "ab" {
		t.Errorf("con ancho 0 el bloque cambió: %q", got)
	}
}

// MarcoCompleto normaliza el marco a Ancho×Alto: exactamente esas celdas, todas
// con fondo, y rellena lo que falte.
func TestMarcoCompletoNormalizaAltoYAncho(t *testing.T) {
	marco := MarcoCompleto("a\nb", 3, 4)
	lineas := strings.Split(marco, "\n")
	if len(lineas) != 4 {
		t.Fatalf("el marco tiene %d filas, quiero 4", len(lineas))
	}
	for i, linea := range lineas {
		if ancho := Ancho(linea); ancho != 3 {
			t.Fatalf("la fila %d mide %d columnas, quiero 3", i, ancho)
		}
	}
	if !strings.Contains(lineas[0], "a") || !strings.Contains(lineas[1], "b") {
		t.Fatalf("el contenido se perdió: %q", lineas)
	}
}

// Un bloque más alto que la ventana se recorta por arriba: lo último pintado es
// lo que queda a la vista, como en un historial que se pegó al final.
func TestMarcoCompletoRecortaPorArriba(t *testing.T) {
	marco := MarcoCompleto("uno\ndos\ntres", 5, 2)
	sin := SinANSI(marco)
	if strings.Contains(sin, "uno") {
		t.Fatalf("la primera fila debería caerse: %q", sin)
	}
	if !strings.Contains(sin, "dos") || !strings.Contains(sin, "tres") {
		t.Fatalf("se perdió el final del bloque: %q", sin)
	}
	if len(strings.Split(marco, "\n")) != 2 {
		t.Fatalf("el marco no se recortó a 2 filas")
	}
}
