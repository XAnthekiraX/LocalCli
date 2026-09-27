// chat.go — T-B014-02: la vista del chat.
//
// Fuente de verdad: SPEC-INTERFAZ §Zonas 1 ("Muestra el historial de la sesión
// activa. Cada intercambio muestra el razonamiento del modelo y su respuesta") y
// §Reglas ("El razonamiento nunca se mezcla visualmente con la respuesta final").
//
// El chat guarda el historial y el mensaje que se está escribiendo, y nada más:
// no sabe de sesiones, ni de flujos, ni de herramientas. Lo que se ve aquí llegó
// por eventos o por la acción de enviar.
//
// El mensaje en curso se guarda aparte del historial porque se llena token a
// token: mezclarlo con los mensajes cerrados obligaría a reescribir el último en
// cada token y a distinguir "ya terminado" de "a medias" en cada lectura.
//
// El chat también lleva el tiempo: marca cuándo se envió el turno en curso y
// cuánto tardó en cerrarse. Es un dato de presentación —cuánto tarda el modelo
// en responder—, no una regla de negocio, así que vive aquí y no en el motor.
package tui

import (
	"strings"
	"time"
)

// Rol es quién produjo un mensaje del chat.
type Rol string

const (
	RolUsuario Rol = "usuario"
	RolAgente  Rol = "agente"
	RolSistema Rol = "sistema"
)

// Mensaje es una línea del chat ya cerrada.
type Mensaje struct {
	Rol          Rol
	Texto        string
	Razonamiento string
	SinRazonami  bool // el modelo no entregó razonamiento (SPEC-PANEL-CONTEXTO)
	// Duracion es lo que tardó el modelo en entregar esta respuesta, del envío
	// al cierre del turno. Cero = no se midió (p. ej. un historial recargado).
	Duracion time.Duration
}

// Chat es el historial de la sesión activa.
type Chat struct {
	mensajes   []Mensaje
	enCurso    strings.Builder
	hayCurso   bool
	propuestas []Propuesta
	// inicio marca cuándo se envió el turno en curso; cero = no hay turno vivo.
	inicio time.Time
}

// AñadirUsuario añade lo que escribió el usuario y arranca el contador del
// turno: el tiempo que se mide es el que percibe quien espera la respuesta.
func (c *Chat) AñadirUsuario(texto string) {
	c.inicio = time.Now()
	c.mensajes = append(c.mensajes, Mensaje{Rol: RolUsuario, Texto: texto})
}

// HayTurno informa si hay una respuesta en camino: desde que se envió el
// mensaje hasta que su turno se cierra. Es lo que hace latir el contador en vivo.
func (c *Chat) HayTurno() bool { return !c.inicio.IsZero() }

// Transcurrido devuelve lo que lleva el turno en curso (0 si no hay ninguno).
func (c *Chat) Transcurrido() time.Duration {
	if c.inicio.IsZero() {
		return 0
	}
	return time.Since(c.inicio)
}

// CancelarTurno deja de contar el turno sin cerrar mensaje: la petición no llegó
// a arrancar (el envío falló), así que no hay respuesta que fechar.
func (c *Chat) CancelarTurno() { c.inicio = time.Time{} }

// AñadirSistema añade una línea del sistema (un aviso, el desenlace de un
// turno). Va como mensaje para que quede en el hilo, no en una barra aparte.
func (c *Chat) AñadirSistema(texto string) {
	c.mensajes = append(c.mensajes, Mensaje{Rol: RolSistema, Texto: texto})
}

// AñadirAgente añade una respuesta ya completa.
func (c *Chat) AñadirAgente(texto string) {
	c.mensajes = append(c.mensajes, Mensaje{Rol: RolAgente, Texto: texto})
}

// Token añade un fragmento a la respuesta en curso. Es lo que llega del
// streaming: mientras se genera, esto es lo único que crece.
func (c *Chat) Token(texto string) {
	c.hayCurso = true
	c.enCurso.WriteString(texto)
}

// EnCurso devuelve la respuesta que se está generando ("" si no hay ninguna).
func (c *Chat) EnCurso() string { return c.enCurso.String() }

// CerrarTurno pasa la respuesta en curso al historial, con el tiempo que tardó
// el turno. Con `razonamiento` vacío se marca que el modelo no lo entregó, para
// que la vista lo pueda decir. El contador se detiene siempre, aunque no haya
// nada que cerrar (turno terminado sin emitir un token).
func (c *Chat) CerrarTurno(razonamiento string) {
	duracion := c.Transcurrido()
	c.inicio = time.Time{}
	if !c.hayCurso && !c.tieneUltimoAgente() {
		return
	}
	m := Mensaje{Rol: RolAgente, Texto: c.enCurso.String(), Razonamiento: razonamiento, Duracion: duracion}
	m.SinRazonami = razonamiento == ""
	c.mensajes = append(c.mensajes, m)
	c.enCurso.Reset()
	c.hayCurso = false
}

// tieneUltimoAgente dice si el último mensaje del historial es del agente (caso
// de un turno que terminó sin emitir ni un token).
func (c *Chat) tieneUltimoAgente() bool {
	return len(c.mensajes) > 0 && c.mensajes[len(c.mensajes)-1].Rol == RolAgente
}

// Mensajes devuelve el historial cerrado.
func (c *Chat) Mensajes() []Mensaje {
	out := make([]Mensaje, len(c.mensajes))
	copy(out, c.mensajes)
	return out
}

// Vaciar deja el chat como recién abierto (al cambiar de sesión).
func (c *Chat) Vaciar() {
	c.mensajes = nil
	c.enCurso.Reset()
	c.hayCurso = false
	c.propuestas = nil
	c.inicio = time.Time{}
}

// MensajeHistorial es un turno cargado de la sesión activa, ya con su
// razonamiento cerrado. Llega como dato a través del puerto, igual que el resto
// de lecturas (INTERFACES §3): la vista no consulta la base.
type MensajeHistorial struct {
	Rol          string // "user" | "agent"
	Texto        string
	Razonamiento string
}

// Cargar reemplaza el historial mostrado por el de la sesión activa (T-F005-05).
// Solo cambia lo que se pinta: no toca nada de las ejecuciones en segundo
// plano, que viven en `session`.
func (c *Chat) Cargar(ms []MensajeHistorial) {
	c.Vaciar()
	for _, m := range ms {
		r := RolAgente
		if m.Rol == "user" {
			r = RolUsuario
		}
		c.mensajes = append(c.mensajes, Mensaje{Rol: r, Texto: m.Texto, Razonamiento: m.Razonamiento})
	}
}

// Render pinta solo el historial cerrado. Cada intercambio de agente se pinta
// con renderIntercambio (styles.go): razonamiento arriba de la respuesta y
// separado de ella, igual que en vivo (T-F005-03); el razonamiento nunca se
// mezcla visualmente con la respuesta final (SPEC-INTERFAZ §Reglas).
func (c *Chat) Render(ancho int) string {
	var b strings.Builder
	for _, m := range c.mensajes {
		b.WriteString(prefijoDe(m.Rol))
		if m.Rol == RolAgente {
			b.WriteString(renderIntercambio(m.Razonamiento, m.Texto, true, false, ancho))
			// El tiempo de la respuesta se cuelga al final, atenuado, para no
			// confundirse con lo que dijo el modelo. Sin medición no se pinta.
			if s := sufijoDuracion(m.Duracion); s != "" {
				b.WriteString(" " + s)
			}
		} else {
			b.WriteString(m.Texto)
		}
		b.WriteString("\n\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

// Propuesta es una propuesta pendiente de aprobación que se muestra dentro del
// chat de la sesión activa (T-F005-06). Vive aparte del historial: el historial
// es lo que ya pasó, y una propuesta que se retira al resolverse no lo es.
type Propuesta struct {
	ID          string
	Sesion      string
	Descripcion string
}

// AñadirPropuesta deja la propuesta a la vista hasta que se resuelva.
func (c *Chat) AñadirPropuesta(p Propuesta) { c.propuestas = append(c.propuestas, p) }

// RetirarPropuesta quita la propuesta resuelta. No es error retirar una que no
// está: la resolución puede llegar por dos caminos (panel y chat).
func (c *Chat) RetirarPropuesta(id string) {
	for i, p := range c.propuestas {
		if p.ID == id {
			c.propuestas = append(c.propuestas[:i], c.propuestas[i+1:]...)
			return
		}
	}
}

// RenderPropuestas pinta las líneas de propuestas pendientes, cada una marcada
// como tal (SPEC-INTERFAZ §Zonas 1: "Muestra las propuestas pendientes de
// aprobación"). Sin propuestas no pinta nada.
func (c *Chat) RenderPropuestas() string {
	if len(c.propuestas) == 0 {
		return ""
	}
	var b strings.Builder
	for _, p := range c.propuestas {
		b.WriteString(estiloAviso.Render("propuesta pendiente: "+p.Descripcion) + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

// recortarAlto conserva el final del texto cuando no cabe en el alto dado: lo
// que interesa leer es lo último que dijo el modelo, no el principio de la
// conversación (T-F005-07). Es una función pura, como recortar.
func recortarAlto(texto string, alto int) string {
	if alto <= 0 {
		return texto
	}
	lineas := strings.Split(texto, "\n")
	if len(lineas) <= alto {
		return texto
	}
	return strings.Join(lineas[len(lineas)-alto:], "\n")
}

// prefijoDe distingue visualmente quién habla. La spec pide que el razonamiento
// y la respuesta nunca se confundan; el prefijo es la parte más barata de esa
// separación.
func prefijoDe(r Rol) string {
	switch r {
	case RolUsuario:
		return estiloUsuario.Render("› ")
	case RolAgente:
		return estiloAgente.Render("· ")
	default:
		return estiloSistema.Render("· ")
	}
}
