package tools

// TopeSalidaBytes es el límite de la capa universal sobre la salida de una
// herramienta. No es configurable en esta versión.
const TopeSalidaBytes = 32 * 1024

const marcaRecorte = "\n… [salida recortada: se superó el límite de la herramienta]"

// Recortar acota la salida al tope, medido en bytes, y avisa por dónde se cortó.
// Una salida que ya quepa se devuelve tal cual.
func Recortar(salida string) (string, bool) {
	if len(salida) <= TopeSalidaBytes {
		return salida, false
	}
	corte := TopeSalidaBytes - len(marcaRecorte)
	if corte < 0 {
		corte = 0
	}
	return cortarEnFrontera(salida, corte) + marcaRecorte, true
}

// cortarEnFrontera corta sin partir un carácter multibyte.
func cortarEnFrontera(s string, corte int) string {
	if corte >= len(s) {
		return s
	}
	for corte > 0 && !utf8Inicio(s[corte]) {
		corte--
	}
	return s[:corte]
}

func utf8Inicio(b byte) bool { return b&0xC0 != 0x80 }

// EstimarTokens es un estimador propio y pequeño de tools (≈4 bytes por token);
// sirve para medir, no para recortar.
func EstimarTokens(texto string) int {
	if texto == "" {
		return 0
	}
	n := (len(texto) + 3) / 4
	if n < 1 {
		n = 1
	}
	return n
}
