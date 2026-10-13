package llm

import "unicode/utf8"

// VentanaDeContexto resuelve la ventana efectiva del turno: la menor entre la
// que declara el modelo (`declarada`) y el tope del harness (`tope`).
//
// Si el modelo no la declara, manda el tope y el resultado queda **desconocido**
// (`conocida == false`): el límite es del harness, no del modelo. Una ventana
// conocida puede seguir limitada por el tope; quien la muestre lo distingue
// comparando con el tope.
func VentanaDeContexto(declarada, tope int) (tokens int, conocida bool) {
	if declarada <= 0 {
		return tope, false
	}
	if tope > 0 && tope < declarada {
		return tope, true
	}
	return declarada, true
}

// VentanaEfectiva resuelve la ventana del turno por precedencia: el override del
// usuario si está definido; si no, el contexto real del modelo; y si el modelo
// no lo declara, el valor por defecto. El contexto del modelo manda sobre el
// defecto porque es su límite real, no una elección del harness.
func VentanaEfectiva(override, contextoModelo, defecto int) int {
	if override > 0 {
		return override
	}
	if contextoModelo > 0 {
		return contextoModelo
	}
	return defecto
}

// EstimarTokens estima tokens por longitud (≈4 caracteres por token). Sirve para
// presupuestar, no para tokenizar de verdad.
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
