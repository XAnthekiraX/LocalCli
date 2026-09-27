// titulo.go — el nombre de una sesión y su título generado por el modelo.
//
// Fuente de verdad: [[specs/SPEC-SESIONES]] (una sesión nueva se crea con la
// primera petición desde la bienvenida; su nombre es un título breve generado a
// partir de esa petición; el id es la identidad permanente y el nombre un
// atributo mutable) y SPEC-INTERFAZ §Pantalla de bienvenida.
//
// `session` no habla con Ollama: el título lo produce quien implemente
// `Titulador` (el arranque, con su cliente). Aquí viven el nombre provisional,
// el prompt y la limpieza de la respuesta, que sí son decisiones del módulo y
// se pueden probar sin modelo.

package session

import (
	"context"
	"strings"
)

// NombreProvisional es el nombre con el que nace toda sesión antes de tener su
// primera petición. Un título generado lo sustituye; una sesión que el usuario
// nunca usó se queda con él.
const NombreProvisional = "Nueva sesión"

// LongitudMaximaTitulo acota el título generado para que no invada el panel ni
// el modal con una frase entera.
const LongitudMaximaTitulo = 60

// Titulador produce un título breve a partir de la primera petición de una
// sesión. Lo implementa el arranque con el modelo local; puede fallar y el
// llamador conserva el nombre provisional (nunca bloquea el turno).
type Titulador interface {
	Titulo(ctx context.Context, texto string) (string, error)
}

// EsProvisional dice si un nombre sigue siendo el de fábrica. Es el segundo
// guardián del título: solo se genera si la sesión todavía se llama así, de modo
// que un nombre cambiado a mano nunca se sobrescribe.
func EsProvisional(nombre string) bool {
	return strings.TrimSpace(nombre) == NombreProvisional
}

// PromptTitulo redacta la consulta que produce el título. Es el único texto que
// sale hacia el modelo desde aquí: la instrucción y la primera petición.
func PromptTitulo(texto string) string {
	return "A partir de esta petición, escribe un título breve (de 3 a 6 palabras) que la represente. " +
		"Responde SOLO con el título, en una línea, sin comillas, sin viñetas y sin punto final.\n\n" +
		"Petición: " + strings.TrimSpace(texto)
}

// LimpiarTitulo normaliza la respuesta del modelo a un título usable: una sola
// línea, sin comillas ni viñetas, con espacios colapsados y acotado. Devuelve ""
// cuando no queda nada; el llamador lo trata como fallo y conserva el
// provisional.
func LimpiarTitulo(s string) string {
	linea := s
	if i := strings.IndexAny(linea, "\r\n"); i >= 0 {
		linea = linea[:i]
	}
	linea = strings.TrimSpace(linea)
	linea = strings.Trim(linea, "`\"'*_#>-–— \t")
	linea = strings.Join(strings.Fields(linea), " ")
	linea = strings.TrimRight(linea, ".。")
	linea = strings.TrimSpace(linea)
	r := []rune(linea)
	if len(r) > LongitudMaximaTitulo {
		linea = strings.TrimSpace(string(r[:LongitudMaximaTitulo]))
	}
	return linea
}
