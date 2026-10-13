package chat

import (
	"strings"
	"time"

	"localcli/internal/tui/render"
)

// framesActividad son los glifos del indicador en vivo, en orden: se recorren en
// ciclo mientras el turno está en marcha.
var framesActividad = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// AñadirUsuario añade lo que escribió el usuario y arranca el contador del turno:
// lo que se mide es lo que percibe quien espera la respuesta, del envío al cierre.
func (m *Model) AñadirUsuario(texto string) {
	m.inicio = time.Now()
	m.AñadirEntrada(texto)
}

// AñadirEntrada añade una línea del usuario sin arrancar el contador del turno:
// lo escrito que no va al modelo —un comando de flujo— no tiene respuesta que
// medir. Una línea del usuario cierra el segmento del turno anterior.
func (m *Model) AñadirEntrada(texto string) {
	m.cerrarSegmento()
	m.ultimoAgente = 0
	m.AñadirSegmento(Segmento{Hablante: Usuario, Texto: texto})
}

// HayTurno informa si hay una respuesta en camino: desde que se envió el mensaje
// hasta que su turno se cierra. Es lo que hace girar el indicador en vivo.
func (m Model) HayTurno() bool { return !m.inicio.IsZero() }

// Transcurrido devuelve lo que lleva el turno en curso (0 si no hay ninguno).
func (m Model) Transcurrido() time.Duration {
	if m.inicio.IsZero() {
		return 0
	}
	return time.Since(m.inicio)
}

// CancelarTurno corta el turno en curso conservando lo que el modelo ya entregó:
// cierra lo parcial como un segmento con su tiempo, igual que un turno que termina
// solo, para no borrar de la pantalla lo que ya se había generado.
func (m *Model) CancelarTurno() {
	m.CerrarTurno()
}

// ReanudarTurno vuelve a arrancar el contador del turno en curso. Lo usa la vista
// al retomar una sesión que ya estaba trabajando: su turno sigue vivo en segundo
// plano, pero el inicio real no se persiste, así que el tiempo se cuenta desde el
// cambio (INTERFACES §1). Sin esto, el indicador quedaría mudo aunque la sesión
// trabajara.
func (m *Model) ReanudarTurno() { m.inicio = time.Now() }

// Fragmento añade un fragmento de la respuesta en curso. Es lo que llega del
// streaming: mientras el modelo genera, esto es lo único que crece.
func (m *Model) Fragmento(texto string) { m.respuesta = append(m.respuesta, texto) }

// FragmentoRazonamiento añade un fragmento del razonamiento del turno en curso.
// No se vuelca por defecto, pero se acumula igual: revelarlo con `Ctrl+R` lo
// recupera entero y no detiene la generación.
func (m *Model) FragmentoRazonamiento(texto string) {
	m.razonamiento = append(m.razonamiento, texto)
}

// CerrarTurno pasa lo generado al historial con el tiempo que tardó el turno y
// detiene el contador. Un turno que terminó justo después de una herramienta, sin
// texto nuevo, no deja segmento vacío: su tiempo se cuelga del último segmento
// del agente para no perderlo.
func (m *Model) CerrarTurno() {
	duracion := m.Transcurrido()
	m.inicio = time.Time{}
	if len(m.respuesta) > 0 {
		m.AñadirSegmento(Segmento{
			Hablante:        Agente,
			Texto:           strings.Join(m.respuesta, ""),
			Razonamiento:    strings.Join(m.razonamiento, ""),
			SinRazonamiento: len(m.razonamiento) == 0,
			Duracion:        duracion,
		})
		m.respuesta, m.razonamiento = nil, nil
		m.ultimoAgente = 0
		return
	}
	if m.ultimoAgente > 0 && m.ultimoAgente <= len(m.piezas) {
		if segmento, ok := m.piezas[m.ultimoAgente-1].(*Segmento); ok {
			segmento.Duracion = duracion
		}
	}
	m.razonamiento = nil
	m.ultimoAgente = 0
}

// Actividad pinta el indicador en vivo con el tiempo transcurrido: `[⠋ Pensando]`
// si todavía no hay respuesta, `[⠋ Usando herramienta: X]` si corre una
// herramienta y `[⠋ Generando]` si ya llega respuesta. Se ve siempre que el turno
// está en marcha, esté o no revelado el razonamiento, y desaparece al cerrarse.
func (m Model) Actividad(frame int) string {
	if !m.HayTurno() {
		return ""
	}
	etiqueta := "Pensando"
	switch {
	case m.herramienta != "":
		etiqueta = "Usando herramienta: " + m.herramienta
	case len(m.respuesta) > 0:
		etiqueta = "Generando"
	}
	marca := render.EstiloSutil.Render("[" + glifoActividad(frame) + " " + etiqueta + "]")
	return marca + " " + render.EstiloSutil.Render(formatearDuracion(m.Transcurrido()))
}

// glifoActividad devuelve el glifo del frame pedido, en ciclo. Un frame negativo
// o desbordado se envuelve; sin frames cae en «*» para no pintar vacío. Es una
// función pura.
func glifoActividad(frame int) string {
	if len(framesActividad) == 0 {
		return "*"
	}
	n := frame % len(framesActividad)
	if n < 0 {
		n += len(framesActividad)
	}
	return framesActividad[n]
}
