package chat

import (
	"strings"
	"testing"

	"localcli/internal/tui/render"
)

// Un bloque de código se pinta con cabecera y superficie, sin los fences, y su
// Markdown interno queda literal. El botón apunta a la cabecera y copia el cuerpo.
func TestBloqueDeCodigoPintaCabeceraYBoton(t *testing.T) {
	texto := "antes\n```go\nx := *no*\n```\ndespués"
	lineas, botones := renderRespuesta(texto, 40)
	plano := render.SinANSI(strings.Join(lineas, "\n"))

	if strings.Contains(plano, "```") {
		t.Fatalf("los fences no deben verse:\n%s", plano)
	}
	if !strings.Contains(plano, "go") {
		t.Fatalf("falta la etiqueta del lenguaje:\n%s", plano)
	}
	if !strings.Contains(plano, textoBotonCopiar) {
		t.Fatalf("falta el botón de copiar:\n%s", plano)
	}
	if !strings.Contains(plano, "x := *no*") {
		t.Fatalf("el cuerpo del código debe quedar literal:\n%s", plano)
	}
	if len(botones) != 1 {
		t.Fatalf("quiero un botón, tengo %d", len(botones))
	}
	b := botones[0]
	if b.Codigo != "x := *no*" {
		t.Fatalf("el botón copia %q", b.Codigo)
	}
	if b.Linea != 1 {
		t.Fatalf("el botón va en la línea 1 (la cabecera), va en %d", b.Linea)
	}
	if fila := render.SinANSI(lineas[b.Linea]); fila[b.Col:b.Col+b.Ancho] != textoBotonCopiar {
		t.Fatalf("la columna del botón no cae sobre el rótulo: %q", fila)
	}
}

// Un fence sin cerrar es un bloque en curso: pinta cabecera y cuerpo hasta el final.
func TestFenceSinCerrarEsBloqueEnCurso(t *testing.T) {
	lineas, botones := renderRespuesta("```py\nprint(1)", 40)
	plano := render.SinANSI(strings.Join(lineas, "\n"))
	if !strings.Contains(plano, "print(1)") {
		t.Fatalf("el cuerpo del bloque en curso no se pinta:\n%s", plano)
	}
	if len(botones) != 1 || botones[0].Codigo != "print(1)" {
		t.Fatalf("el bloque en curso debe exponer su botón: %+v", botones)
	}
}

// El subconjunto de Markdown inline se interpreta: los marcadores desaparecen y
// el texto se conserva.
func TestMarkdownInline(t *testing.T) {
	casos := []struct{ entrada, quiero string }{
		{"hola **mundo**", "hola mundo"},
		{"hola *mundo*", "hola mundo"},
		{"hola _mundo_", "hola mundo"},
		{"usa `codigo` aquí", "usa codigo aquí"},
	}
	for _, c := range casos {
		lineas := pintarLineaMD(c.entrada, 80)
		plano := render.SinANSI(strings.Join(lineas, "\n"))
		if plano != c.quiero {
			t.Fatalf("%q → %q, quiero %q", c.entrada, plano, c.quiero)
		}
	}
}

// Un marcador sin pareja queda literal: nunca se pierde texto.
func TestMarcadorSinParejaQuedaLiteral(t *testing.T) {
	lineas := pintarLineaMD("2 * 3 = 6", 80)
	if plano := render.SinANSI(strings.Join(lineas, "\n")); plano != "2 * 3 = 6" {
		t.Fatalf("el asterisco suelto se perdió: %q", plano)
	}
}

// El guion bajo no parte una palabra, para no tocar identificadores.
func TestGuionBajoNoRompeSnakeCase(t *testing.T) {
	lineas := pintarLineaMD("el campo mi_variable_local sigue", 80)
	if plano := render.SinANSI(strings.Join(lineas, "\n")); plano != "el campo mi_variable_local sigue" {
		t.Fatalf("snake_case se interpretó como cursiva: %q", plano)
	}
}

// Título, regla, cita y viñetas se pintan sin sus marcadores.
func TestBloquesDeMarkdown(t *testing.T) {
	casos := []struct{ entrada, quiero string }{
		{"## Título", "Título"},
		{"---", strings.Repeat("─", 10)},
		{"> una cita", "│ una cita"},
		{"- un punto", "• un punto"},
		{"1. primero", "1. primero"},
	}
	for _, c := range casos {
		lineas := pintarLineaMD(c.entrada, 10)
		plano := render.SinANSI(strings.Join(lineas, "\n"))
		if plano != c.quiero {
			t.Fatalf("%q → %q, quiero %q", c.entrada, plano, c.quiero)
		}
	}
}

// Todo se envuelve al ancho: ninguna fila excede las columnas pedidas.
func TestNingunaFilaExcedeElAncho(t *testing.T) {
	texto := "palabra " + strings.Repeat("larguisima ", 6) + "\n```go\n" + strings.Repeat("abcdefg", 12) + "\n```"
	lineas, _ := renderRespuesta(texto, 20)
	for i, linea := range lineas {
		if ancho := render.Ancho(render.SinANSI(linea)); ancho > 20 {
			t.Fatalf("la línea %d mide %d columnas", i, ancho)
		}
	}
}

// Un texto sin Markdown no gana ni pierde nada frente al pintado llano.
func TestTextoPlanoIntacto(t *testing.T) {
	lineas := pintarLineaMD("una frase normal sin marcadores", 80)
	if plano := render.SinANSI(strings.Join(lineas, "\n")); plano != "una frase normal sin marcadores" {
		t.Fatalf("el texto plano cambió: %q", plano)
	}
}
