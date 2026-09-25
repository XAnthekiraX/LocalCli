// reasoning.go — T-B005-03: extracción del razonamiento como evento separado.
//
// Fuente de verdad: INTEGRATIONS.md §5 ("el harness espera respuestas en
// streaming, con el razonamiento distinguible del texto final") y
// DECISIONS.md [25] (la pantalla recibe los tokens por eventos; la fila de
// reasoning solo existe para auditoría/recuperación, persistida cada 200 ms).
//
// Dos orígenes posibles del razonamiento, según modelo/versión de Ollama:
//  1. Campo `thinking` en la línea NDJSON → ya viene separado; pasa tal cual
//     (separarRazonamiento lo limpia por si el proveedor mete espacios).
//  2. Modelos que imprimen bloques  dentro del propio
//     texto → SeparadorEnTexto los extrae del flujo de tokens.
package ollama

import "strings"

// separarRazonamiento normaliza el campo thinking. Un stream puede traer
// cadenas vacías o de relleno; devolver "" suprime el evento.
func separarRazonamiento(thinking string) string {
	if thinking == "" {
		return ""
	}
	// No se recorta el interior: los espacios son parte del token. Solo se
	// ignora un thinking compuesto enteramente de whitespace.
	if strings.TrimSpace(thinking) == "" {
		return ""
	}
	return thinking
}

// SeparadorEnTexto es un extractor con estado para razonamiento incrustado en
// el texto final. Se usa cuando el modelo no trae campo thinking pero sí
// emite  ...  inline.
//
// Uso: alimentar cada token con Push(token); consume los fragmentos que
// pueden emitirse YA (fuera de bloque → EventoToken; dentro de bloque →
// EventoRazonamiento). El diseño retiene hasta 19 bytes (longitud de
// "") porque un token puede partir el marcador por la mitad.
type SeparadorEnTexto struct {
	buf     strings.Builder
	dentro  bool // estamos dentro de  ?
	cerrado bool // vimos  ; nada más razonamiento
}

const (
	apertura = ""
	cierre   = ""
)

// Fragmento es una pieza ya clasificada producida por el separador.
type Fragmento struct {
	EsRazonamiento bool
	Texto          string
}

// Push añade un token y devuelve los fragmentos listos para emitir. Puede
// devolver varios (p. ej. texto antes de un apertura + arranque de bloque).
func (s *SeparadorEnTexto) Push(token string) []Fragmento {
	s.buf.WriteString(token)
	var salidas []Fragmento
	for {
		texto := s.buf.String()
		if !s.dentro {
			i := strings.Index(texto, apertura)
			if i < 0 {
				// ¿Podría estar el marcador a medias al final? Retener el
				// sufijo más largo que sea prefijo de "".
				ret := prefijoParcialFinal(texto, apertura)
				emitir := len(texto) - ret
				if emitir > 0 {
					salidas = append(salidas, Fragmento{false, texto[:emitir]})
					s.buf.Reset()
					s.buf.WriteString(texto[emitir:])
				}
				return salidas
			}
			if i > 0 {
				salidas = append(salidas, Fragmento{false, texto[:i]})
			}
			s.buf.Reset()
			s.buf.WriteString(texto[i+len(apertura):])
			s.dentro = true
			continue
		}
		// dentro del bloque
		j := strings.Index(texto, cierre)
		if j < 0 {
			ret := prefijoParcialFinal(texto, cierre)
			emitir := len(texto) - ret
			if emitir > 0 {
				salidas = append(salidas, Fragmento{true, texto[:emitir]})
				s.buf.Reset()
				s.buf.WriteString(texto[emitir:])
			}
			return salidas
		}
		if j > 0 {
			salidas = append(salidas, Fragmento{true, texto[:j]})
		}
		residual := texto[j+len(cierre):]
		s.buf.Reset()
		s.buf.WriteString(residual)
		s.dentro = false
		s.cerrado = true
		if residual == "" {
			return salidas // nada que reevaluar; evita bucle con buffer vacío
		}
		// Bucle solo si queda residuo procesable (tras el cierre).
	}
}

// Cerrar devuelve lo retenido pendiente al terminar el stream. Si el bloque
// nunca se cerró, el contenido residual cuenta como razonamiento (mejor
// etiquetado conservador que perderlo).
func (s *SeparadorEnTexto) Cerrar() []Fragmento {
	res := s.buf.String()
	s.buf.Reset()
	if res == "" {
		return nil
	}
	return []Fragmento{{EsRazonamiento: s.dentro, Texto: res}}
}

// prefijoParcialFinal devuelve cuántos bytes finales de texto coinciden con
// un prefijo PROPIO de marcador (para retener "<thin" entre tokens).
func prefijoParcialFinal(texto, marcador string) int {
	max := len(marcador) - 1
	if max > len(texto) {
		max = len(texto)
	}
	for n := max; n > 0; n-- {
		if strings.HasSuffix(texto, marcador[:n]) {
			return n
		}
	}
	return 0
}
