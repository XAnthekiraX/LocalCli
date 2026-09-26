package context

// tokens.go — T-B011-03: estimar los tokens de cada documento candidato.
//
// Fuente de verdad: [[specs/SPEC-NODO-CONTEXTO]] ("El contexto entregado nunca
// supera el límite de contexto del modelo") y VALIDATION.md §4 ("Tokens: se
// contabilizan por mensaje para poder ajustar el contexto").
//
// No hay tokenizador por modelo y no hace falta uno exacto para recortar: la
// estimación es estable y monótona (más texto, más tokens), y el recorte deja
// margen. Cuatro caracteres por token es la aproximación habitual para texto
// mixto y español.

// CaracteresPorToken es la razón usada para estimar.
const CaracteresPorToken = 4

// EstimarTokens devuelve una estimación estable y no negativa.
func EstimarTokens(texto string) int {
	if texto == "" {
		return 0
	}
	return (len(texto) + CaracteresPorToken - 1) / CaracteresPorToken
}
