package context

import (
	"strings"
	"unicode/utf8"
)

// presupuestoDefecto es el tope de tokens del bloque sin preferencia declarada.
const presupuestoDefecto = 4096

// RecortarPorSeccion entrega el contenido recortado por secciones (un encabezado
// `#` y lo que le sigue) hasta agotar el presupuesto. Devuelve el texto, si se
// omitió algo y los tokens estimados del tramo entregado.
func RecortarPorSeccion(contenido string, presupuesto int) (string, bool, int) {
	if presupuesto <= 0 {
		return "", true, 0
	}
	if EstimarTokens(contenido) <= presupuesto {
		return contenido, false, EstimarTokens(contenido)
	}

	var b strings.Builder
	usados := 0
	truncado := false
	for _, seccion := range secciones(contenido) {
		tokens := EstimarTokens(seccion)
		if usados+tokens > presupuesto {
			truncado = true
			break
		}
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		b.WriteString(seccion)
		usados += tokens
	}
	if usados == 0 {
		return "", true, 0
	}
	return b.String(), truncado, usados
}

// secciones parte el contenido por sus encabezados de nivel superior.
func secciones(contenido string) []string {
	var secciones []string
	var actual strings.Builder
	for _, linea := range strings.Split(contenido, "\n") {
		if strings.HasPrefix(linea, "#") && actual.Len() > 0 {
			secciones = append(secciones, strings.TrimRight(actual.String(), "\n"))
			actual.Reset()
		}
		actual.WriteString(linea)
		actual.WriteString("\n")
	}
	if actual.Len() > 0 {
		secciones = append(secciones, strings.TrimRight(actual.String(), "\n"))
	}
	return secciones
}

// EstimarTokens estima tokens por longitud (≈4 caracteres por token).
func EstimarTokens(texto string) int {
	if texto == "" {
		return 0
	}
	n := (utf8.RuneCountInString(texto) + 3) / 4
	if n < 1 {
		n = 1
	}
	return n
}
