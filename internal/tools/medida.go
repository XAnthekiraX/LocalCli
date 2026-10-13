package tools

import "strings"

// Medir cuenta las unidades con contenido de la salida y las describe. La medida
// va al evento y al panel, nunca al modelo.
func Medir(unidad Unidad, salida string) string {
	if unidad == "" {
		return ""
	}
	n := contarConContenido(salida)
	switch unidad {
	case UnidadLinea:
		return plural(n, "línea", "líneas")
	case UnidadEntrada:
		return plural(n, "entrada", "entradas")
	case UnidadCoincidencia:
		return plural(n, "coincidencia", "coincidencias")
	case UnidadResultado:
		return plural(n, "resultado", "resultados")
	case UnidadPaso:
		return plural(n, "paso", "pasos")
	case UnidadEtapa:
		return plural(n, "etapa", "etapas")
	}
	return ""
}

func contarConContenido(salida string) int {
	n := 0
	for _, linea := range strings.Split(salida, "\n") {
		if strings.TrimSpace(linea) != "" {
			n++
		}
	}
	return n
}

func plural(n int, singular, pluralForma string) string {
	if n == 1 {
		return "1 " + singular
	}
	return itoa(n) + " " + pluralForma
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digitos []byte
	for n > 0 {
		digitos = append([]byte{byte('0' + n%10)}, digitos...)
		n /= 10
	}
	return string(digitos)
}
