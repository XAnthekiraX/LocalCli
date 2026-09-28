package tools

// truncado.go — T-B024-04: recorte universal de la salida por tokens.
//
// Fuente de verdad: ai/docs/backend/02-interfaces/TOOLS.md §8 ("Toda salida se
// recorta antes de volver al modelo… se mide en tokens, contra el presupuesto
// de contexto, con un estimador propio de `tools`") y ai/docs/backend/DECISIONS.md
// ("El truncado de la salida de herramientas vive en `tools`, con estimador
// propio").
//
// `tools` NO reutiliza el estimador de `internal/context`: `context` ya depende
// de `ollama`, `store` y `docs`, y que `tools` lo importara convertiría un
// módulo de capa baja en uno que arrastra capa alta. El estimador de aquí es
// pequeño a propósito.

import (
	"strconv"
	"strings"
	"unicode/utf8"
)

// LimiteTokensSalida es el presupuesto por defecto de la salida de una
// herramienta. Cuenta como el del contexto: una lectura enorme o el contenido
// de una página desbordan igual que un comando sin fin.
const LimiteTokensSalida = 2000

// Estimador propio: la proporción habitual en texto y código es de unos cuatro
// caracteres por token. No pretende ser exacto, solo comparable con el
// presupuesto de contexto y lo bastante barato para llamarlo en cada salida.
func EstimarTokens(s string) int {
	if s == "" {
		return 0
	}
	return len(s)/4 + 1
}

// Recortar corta la salida para que quepa en el límite de tokens y avisa de por
// dónde se cortó. Devuelve la cadena resultante y si hubo recorte. Un límite no
// positivo significa "sin recorte": no se corta nada.
//
// El recorte no es silencioso: la marca dice que se cortó y cuánto queda fuera,
// de modo que el modelo no actúe como si tuviera el resultado entero.
func Recortar(salida string, limite int) (string, bool) {
	if limite <= 0 || EstimarTokens(salida) <= limite {
		return salida, false
	}
	marca := marcaDe(salida, limite)
	// Se reserva sitio para la marca y para el salto que la precede, de modo
	// que el resultado —marca incluida— quepa de verdad en el límite.
	disponible := (limite-1)*4 - len(marca) - 1
	if disponible < 0 {
		disponible = 0
	}
	if disponible >= len(salida) {
		return salida, false
	}
	// No cortar en mitad de un rune: se retrocede hasta el inicio del carácter.
	corte := disponible
	for corte > 0 && !utf8.RuneStart(salida[corte]) {
		corte--
	}
	return strings.TrimRight(salida[:corte], "\n") + "\n" + marca, true
}

// marcaDe compone el aviso de recorte, anunciando por dónde se cortó.
func marcaDe(salida string, limite int) string {
	return "[salida recortada: se muestran los primeros " + strconv.Itoa(limite) +
		" tokens de " + strconv.Itoa(EstimarTokens(salida)) + "]"
}
