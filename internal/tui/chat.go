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
package tui

import "strings"

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
}

// Chat es el historial de la sesión activa.
type Chat struct {
	mensajes []Mensaje
	enCurso  strings.Builder
	hayCurso bool
}

// AñadirUsuario añade lo que escribió el usuario.
func (c *Chat) AñadirUsuario(texto string) {
	c.mensajes = append(c.mensajes, Mensaje{Rol: RolUsuario, Texto: texto})
}

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

// CerrarTurno pasa la respuesta en curso al historial. Con `razonamiento` vacío
// se marca que el modelo no lo entregó, para que la vista lo pueda decir.
func (c *Chat) CerrarTurno(razonamiento string) {
	if !c.hayCurso && !c.tieneUltimoAgente() {
		return
	}
	m := Mensaje{Rol: RolAgente, Texto: c.enCurso.String(), Razonamiento: razonamiento}
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
}

// Render pinta solo el historial cerrado. El mensaje en curso y el razonamiento
// los pinta la vista, porque van en otro orden (razonamiento arriba, respuesta
// debajo) y con otro estilo.
func (c *Chat) Render(ancho int) string {
	var b strings.Builder
	for _, m := range c.mensajes {
		b.WriteString(prefijoDe(m.Rol))
		b.WriteString(m.Texto)
		b.WriteString("\n")
		if m.Rol == RolAgente && m.Razonamiento != "" {
			b.WriteString(estiloRazonamiento.Render(recortar(m.Razonamiento, ancho)))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
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

// recortar deja el texto en una línea por párrafo y sin exceder el ancho, para
// que el razonamiento no rompa la disposición del panel.
func recortar(texto string, ancho int) string {
	if ancho <= 0 {
		return texto
	}
	var out []string
	for _, linea := range strings.Split(texto, "\n") {
		for len([]rune(linea)) > ancho {
			r := []rune(linea)
			out = append(out, string(r[:ancho]))
			linea = string(r[ancho:])
		}
		out = append(out, linea)
	}
	return strings.Join(out, "\n")
}
