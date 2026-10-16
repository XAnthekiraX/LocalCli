package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"localcli/internal/tui/render"
)

// caracteresDelArte son los únicos caracteres que el arte canónico puede usar:
// bloques y esquinas en UTF-8 (SPEC-INTERFAZ §Arte canónico del logotipo). El
// espacio solo aparece dentro del dibujo, nunca al final de la fila.
const caracteresDelArte = "█╗╝║╔═╚ "

// El logotipo es arte fijo: seis filas, sin espacios finales, y el del binario es
// byte a byte el de la dorada `testdata/logo.txt`, que es su definición canónica.
// El arte vigente mide más ancho que las 53 columnas que anota SPEC-INTERFAZ
// §Arte canónico —se rediseñó sin actualizar el documento—, así que aquí se
// comprueba lo que el arte declara de sí mismo: seis filas, el juego de
// caracteres del canon y ningún espacio final.
func TestElLogotipoEsElArteCanonico(t *testing.T) {
	bruto, err := os.ReadFile(filepath.Join("testdata", "logo.txt"))
	if err != nil {
		t.Fatalf("no se pudo leer la dorada del logotipo: %v", err)
	}
	if arteCanonico != string(bruto) {
		t.Fatalf("el arte del binario no es el de testdata/logo.txt:\n%q\n%q", arteCanonico, string(bruto))
	}

	lineas := strings.Split(string(bruto), "\n")
	if ultima := len(lineas) - 1; ultima >= 0 && lineas[ultima] == "" {
		lineas = lineas[:ultima]
	}
	if len(lineas) != 6 {
		t.Fatalf("el logotipo tiene %d filas, quiero 6", len(lineas))
	}
	for i, linea := range lineas {
		if strings.TrimRight(linea, " ") != linea {
			t.Errorf("la fila %d lleva espacio final: %q", i, linea)
		}
		if strings.TrimSpace(linea) == "" {
			t.Errorf("la fila %d está vacía", i)
		}
		for _, r := range linea {
			if !strings.ContainsRune(caracteresDelArte, r) {
				t.Errorf("la fila %d usa un carácter fuera del arte canónico: %q", i, r)
			}
		}
	}
}

// El logotipo compacto es el arte que la bienvenida pinta en terminales
// estrechas: seis filas, el mismo juego de bloques y las mismas reglas que el
// completo, pero más estrecho (deletrea `L.CLI`), embebido byte a byte desde
// `testdata/logo-compacto.txt`.
func TestElLogotipoCompactoEsSuArteCanonico(t *testing.T) {
	bruto, err := os.ReadFile(filepath.Join("testdata", "logo-compacto.txt"))
	if err != nil {
		t.Fatalf("no se pudo leer el arte compacto: %v", err)
	}
	if arteCompacto != string(bruto) {
		t.Fatalf("el arte compacto del binario no es el de testdata/logo-compacto.txt:\n%q\n%q", arteCompacto, string(bruto))
	}
	lineas := strings.Split(string(bruto), "\n")
	if ultima := len(lineas) - 1; ultima >= 0 && lineas[ultima] == "" {
		lineas = lineas[:ultima]
	}
	if len(lineas) != 6 {
		t.Fatalf("el logotipo compacto tiene %d filas, quiero 6", len(lineas))
	}
	for i, linea := range lineas {
		if strings.TrimRight(linea, " ") != linea {
			t.Errorf("la fila %d lleva espacio final: %q", i, linea)
		}
		if strings.TrimSpace(linea) == "" {
			t.Errorf("la fila %d está vacía", i)
		}
		if ancho := render.Ancho(linea); ancho > 40 {
			t.Errorf("la fila %d mide %d columnas y no cabe en una terminal estrecha", i, ancho)
		}
		for _, r := range linea {
			if !strings.ContainsRune(caracteresDelArte, r) {
				t.Errorf("la fila %d usa un carácter fuera del arte canónico: %q", i, r)
			}
		}
	}
}
