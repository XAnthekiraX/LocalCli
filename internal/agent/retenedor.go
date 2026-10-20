package agent

import "strings"

// maxRetenido acota lo que se guarda a la espera de saber si es una llamada:
// superarlo vuelca el texto y deja de retener, para no acumular una respuesta
// larga en memoria.
const maxRetenido = 32 * 1024

// umbralNombre es cuánto texto se tolera sin ver la clave `name` antes de dar
// por hecho que el JSON no es una llamada: así una respuesta JSON normal no se
// queda retenida hasta el final de la ronda.
const umbralNombre = 256

// iniciosDeLlamada son los prefijos con los que empieza una llamada escrita como
// texto. Mientras lo escrito sea un prefijo de alguno de ellos, o empiece por
// uno, se retiene.
var iniciosDeLlamada = []string{"{", "```", etiquetaLlamadaAbre}

// retenedor envuelve el callback del stream y retiene el texto que todavía
// podría ser una llamada de herramienta. Así el JSON no se pinta como respuesta
// cuando acaba ejecutándose, sin dejar de transmitir la prosa en vivo: la
// retención solo empieza si la respuesta abre con un JSON, una cerca o la
// etiqueta de la plantilla.
type retenedor struct {
	alDelta    func(Delta) error
	deltas     []Delta
	acumulado  strings.Builder
	reteniendo bool
	descartado bool
}

// nuevoRetenedor devuelve un retenedor para una ronda. Sin callback no hay nada
// que retener.
func nuevoRetenedor(alDelta func(Delta) error) *retenedor {
	return &retenedor{alDelta: alDelta, reteniendo: alDelta != nil}
}

// Fragmento es el callback que se le pasa al motor. El razonamiento no se toca;
// el texto se retiene solo mientras podría ser una llamada.
func (r *retenedor) Fragmento(d Delta) error {
	if r.alDelta == nil {
		return nil
	}
	if d.Razonamiento {
		return r.alDelta(d)
	}
	if r.descartado {
		return nil
	}
	if !r.reteniendo {
		return r.alDelta(d)
	}
	r.deltas = append(r.deltas, d)
	r.acumulado.WriteString(d.Texto)
	texto := r.acumulado.String()
	if !podriaSerLlamada(texto) || r.acumulado.Len() > maxRetenido {
		return r.Volcar()
	}
	if r.acumulado.Len() > umbralNombre && !strings.Contains(texto, `"name"`) && !strings.Contains(texto, `"function"`) {
		return r.Volcar()
	}
	return nil
}

// podriaSerLlamada indica si el texto, por ahora, puede acabar siendo una
// llamada. El espacio inicial no decide: se mira el primer carácter visible.
func podriaSerLlamada(texto string) bool {
	t := strings.TrimLeft(texto, " \t\r\n")
	if t == "" {
		return true
	}
	for _, inicio := range iniciosDeLlamada {
		if strings.HasPrefix(inicio, t) || strings.HasPrefix(t, inicio) {
			return true
		}
	}
	return false
}

// Volcar entrega lo retenido y deja de retener. Es lo que se hace al terminar
// una ronda que no era una llamada y al cortar el turno por un error.
func (r *retenedor) Volcar() error {
	if !r.reteniendo {
		return nil
	}
	r.reteniendo = false
	deltas := r.deltas
	r.deltas = nil
	r.acumulado.Reset()
	for _, d := range deltas {
		if err := r.alDelta(d); err != nil {
			return err
		}
	}
	return nil
}

// Descartar tira lo retenido y silencia el resto de la ronda: era una llamada y
// su JSON no debe aparecer en el hilo como si fuera la respuesta del agente.
func (r *retenedor) Descartar() {
	r.reteniendo = false
	r.descartado = true
	r.deltas = nil
	r.acumulado.Reset()
}
