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

	"github.com/charmbracelet/lipgloss"
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
	// MostrarRazonamiento decide si el razonamiento cerrado del historial se
	// vuelca. Por defecto no: es la misma bandera que revela el texto en vivo
	// (`Ctrl+R`) y la sincroniza el `app`. El indicador en vivo no depende de
	// ella.
	MostrarRazonamiento bool
	// inicio marca cuándo se envió el turno en curso; cero = no hay turno vivo.
	inicio time.Time

	// offset y maxOffset son la ventana del historial: la primera línea visible
	// y el máximo para el ancho de la última pintura. scrolleado se enciende al
	// subir: mientras está apagado, la vista sigue el final y baja sola con cada
	// token nuevo; al leer hacia arriba se respeta la posición. altoUltimo
	// recuerda el alto para avanzar una página entera.
	offset     int
	maxOffset  int
	scrolleado bool
	altoUltimo int

	// pendientes son los índices de las líneas de herramienta abiertas, en
	// orden de invocación. Al llegar el resultado se cierra la primera (FIFO),
	// así la misma línea pasa de «LEER [ruta]» a «✓ LEER [ruta] · 70 líneas».
	pendientes []int

	// ultimoAgente es el índice+1 del último segmento del agente cerrado en el
	// turno (0 = ninguno). Sirve para colgar la duración del turno del último
	// segmento cuando la respuesta termina justo después de una herramienta y no
	// queda texto final al que colgarla.
	ultimoAgente int
}

// AñadirUsuario añade lo que escribió el usuario y arranca el contador del
// turno: el tiempo que se mide es el que percibe quien espera la respuesta.
func (c *Chat) AñadirUsuario(texto string) {
	c.inicio = time.Now()
	c.AñadirEntrada(texto)
}

// AñadirEntrada añade una línea del usuario SIN arrancar el contador del turno.
// Es lo que se usa cuando lo escrito no va al modelo —por ejemplo un comando de
// flujo—: no hay respuesta en camino que medir.
func (c *Chat) AñadirEntrada(texto string) {
	// Una línea del usuario cierra lo que hubiera del turno anterior: el
	// segmento de este turno todavía no existe.
	c.ultimoAgente = 0
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

// AnotarInvocacion abre la línea de una herramienta en una forma compacta: su
// verbo de pantalla y su objetivo —«LEER [ruta]»—. No cierra nada todavía: la
// línea se completa con CerrarHerramienta cuando llega el resultado. Mientras
// corre, el indicador en vivo de la vista dice «Usando herramienta: <nombre>».
func (c *Chat) AnotarInvocacion(verbo, tema string) {
	if verbo == "" {
		return
	}
	linea := verbo
	if tema != "" {
		linea += " [" + tema + "]"
	}
	c.mensajes = append(c.mensajes, Mensaje{Rol: RolSistema, Texto: linea})
	c.pendientes = append(c.pendientes, len(c.mensajes)-1)
}

// CerrarHerramienta cierra la línea abierta por AnotarInvocacion con su marca
// compacta: «✓» si terminó bien, «✗» si falló, más la medida del resultado
// («70 líneas») y, si se recortó, el aviso. La salida cruda no se pinta: va al
// modelo, no a la pantalla (EVENTS.md §3). Sin invocación pendiente —un
// resultado suelto— se añade una línea nueva con el nombre de la herramienta.
func (c *Chat) CerrarHerramienta(nombre string, ok, truncado bool, medida, err string) {
	marca := "✓ "
	if !ok {
		marca = "✗ "
	}
	idx := c.siguientePendiente()
	linea := nombre
	if idx >= 0 {
		linea = c.mensajes[idx].Texto
	}
	linea = marca + linea
	if medida != "" {
		linea += " · " + medida
	}
	if truncado {
		linea += " · recortado"
	}
	if !ok && err != "" {
		linea += " · " + recortarError(err)
	}
	if idx >= 0 {
		c.mensajes[idx].Texto = linea
		return
	}
	c.AñadirSistema(linea)
}

// siguientePendiente saca la línea abierta más antigua y la deja fuera de la
// cola. Devuelve -1 si no hay ninguna.
func (c *Chat) siguientePendiente() int {
	if len(c.pendientes) == 0 {
		return -1
	}
	idx := c.pendientes[0]
	c.pendientes = c.pendientes[1:]
	if idx < 0 || idx >= len(c.mensajes) {
		return -1
	}
	return idx
}

// recortarError abrevia el motivo de un fallo para la línea: una sola línea.
func recortarError(err string) string {
	err = strings.Join(strings.Fields(err), " ")
	r := []rune(err)
	if len(r) <= 60 {
		return err
	}
	return string(r[:60]) + "…"
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

// CerrarSegmento cierra el texto en curso como un intercambio del agente, sin dar
// el turno por terminado. Se usa cuando algo se interpone en el hilo —una línea de
// herramienta, un aviso—: lo dicho hasta ahí queda en el historial, arriba de esa
// línea, y lo que el modelo diga después abre un globo nuevo. Así el chat respeta
// el orden en que ocurrió y no funde en un solo globo el texto anterior y el
// posterior a la herramienta. Sin nada acumulado no añade nada (una invocación
// directa no deja globo vacío).
func (c *Chat) CerrarSegmento(razonamiento string) {
	if c.enCurso.Len() == 0 && strings.TrimSpace(razonamiento) == "" {
		return
	}
	m := Mensaje{Rol: RolAgente, Texto: c.enCurso.String(), Razonamiento: razonamiento}
	m.SinRazonami = razonamiento == ""
	c.mensajes = append(c.mensajes, m)
	c.ultimoAgente = len(c.mensajes) // índice+1
	c.enCurso.Reset()
	c.hayCurso = false
}

// CerrarTurno pasa el texto en curso al historial, con el tiempo que tardó el
// turno. Con `razonamiento` vacío se marca que el modelo no lo entregó, para que
// la vista lo pueda decir. El contador se detiene siempre, aunque no haya nada que
// cerrar (turno terminado sin emitir un token). Si el turno terminó justo después
// de un segmento ya cerrado —por ejemplo tras una herramienta—, la duración se
// cuelga de ese último segmento para no perderla.
func (c *Chat) CerrarTurno(razonamiento string) {
	duracion := c.Transcurrido()
	c.inicio = time.Time{}
	if c.hayCurso {
		m := Mensaje{Rol: RolAgente, Texto: c.enCurso.String(), Razonamiento: razonamiento, Duracion: duracion}
		m.SinRazonami = razonamiento == ""
		c.mensajes = append(c.mensajes, m)
		c.enCurso.Reset()
		c.hayCurso = false
		c.ultimoAgente = 0
		return
	}
	if c.ultimoAgente > 0 {
		c.mensajes[c.ultimoAgente-1].Duracion = duracion
		c.ultimoAgente = 0
	}
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
	c.offset = 0
	c.maxOffset = 0
	c.scrolleado = false
	c.altoUltimo = 0
	c.pendientes = nil
	c.ultimoAgente = 0
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

// Render pinta solo el historial cerrado. Lo escrito por el usuario y lo que
// responde el agente van cada uno en su globo de color (burbuja, styles.go):
// azul lo tuyo, verde lo del agente/terminal (T-F036). Cada intercambio de
// agente se compone con renderIntercambio: razonamiento arriba de la respuesta
// y separado de ella, igual que en vivo (T-F005-03); el razonamiento nunca se
// mezcla visualmente con la respuesta final (SPEC-INTERFAZ §Reglas). Las líneas
// del sistema (herramientas, avisos) no son un turno: se pintan sueltas.
func (c *Chat) Render(ancho int) string {
	// Los globos dejan un margen a cada lado para no quedar pegados al borde de
	// la columna (ni al divisor con el sidebar).
	interno := anchoGlobo(ancho - anchoIcono - 2*margenChat)
	var b strings.Builder
	for _, m := range c.mensajes {
		var bloque string
		switch m.Rol {
		case RolAgente:
			if contenido := renderIntercambio(m.Razonamiento, m.Texto, c.MostrarRazonamiento, false, interno); contenido != "" {
				globo := burbuja(RolAgente, contenido)
				// El tiempo de la respuesta se cuelga al final, atenuado, para no
				// confundirse con lo que dijo el modelo. Sin medición no se pinta.
				if s := sufijoDuracion(m.Duracion); s != "" {
					globo += " " + s
				}
				// El icono del agente precede a su globo; el margen lo separa del
				// borde izquierdo.
				bloque = strings.Repeat(" ", margenChat) + lipgloss.JoinHorizontal(lipgloss.Top, iconoDeRol(RolAgente), " ", globo)
			}
		case RolUsuario:
			// El globo del usuario se pega a la derecha (con su margen) y su icono
			// va detrás.
			globo := burbuja(RolUsuario, recortar(m.Texto, interno))
			bloque = lineasALaDerecha(lipgloss.JoinHorizontal(lipgloss.Top, globo, " ", iconoDeRol(RolUsuario)), ancho-margenChat)
		default:
			bloque = strings.Repeat(" ", margenChat) + estiloSistema.Render("· ") + m.Texto
		}
		if bloque == "" {
			continue
		}
		b.WriteString(bloque)
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

// Ventana devuelve las líneas visibles del historial para el alto dado y deja
// lista la posición de scroll. Si el historial cabe entero, se ve todo y se
// sigue el final. Si no cabe y el usuario no ha subido (`scrolleado` apagado),
// la ventana se pega al final para que lo último dicho siga a la vista; si
// subió, se respeta su posición, acotada a lo que hay. Anclar arriba con la
// bandera de seguimiento evita que la vista salte mientras el modelo genera un
// token detrás de otro.
func (c *Chat) Ventana(ancho, alto int) string {
	texto := c.Render(ancho)
	c.altoUltimo = alto
	if alto <= 0 || texto == "" {
		c.offset, c.maxOffset, c.scrolleado = 0, 0, false
		return texto
	}
	lineas := strings.Split(texto, "\n")
	if len(lineas) <= alto {
		c.offset, c.maxOffset, c.scrolleado = 0, 0, false
		return texto
	}
	c.maxOffset = len(lineas) - alto
	if c.scrolleado {
		if c.offset < 0 {
			c.offset = 0
		}
		if c.offset > c.maxOffset {
			c.offset = c.maxOffset
		}
	} else {
		c.offset = c.maxOffset
	}
	return strings.Join(lineas[c.offset:c.offset+alto], "\n")
}

// Subir asciende el historial n líneas y deja de seguir el final: lo que se está
// leyendo no debe moverse cuando llegue un token nuevo.
func (c *Chat) Subir(n int) {
	if n <= 0 {
		return
	}
	c.scrolleado = true
	c.offset -= n
	if c.offset < 0 {
		c.offset = 0
	}
}

// Bajar desciende n líneas. Al alcanzar el final vuelve a seguir el final, de
// modo que la vista retoma sola las respuestas nuevas.
func (c *Chat) Bajar(n int) {
	if n <= 0 {
		return
	}
	c.offset += n
	if c.offset >= c.maxOffset {
		c.offset = c.maxOffset
		c.scrolleado = false
	}
}

// SubirPagina y BajarPagina avanzan una pantalla menos una línea, como el
// paginado de un lector.
func (c *Chat) SubirPagina() { c.Subir(c.pagina()) }
func (c *Chat) BajarPagina() { c.Bajar(c.pagina()) }

func (c *Chat) pagina() int {
	if c.altoUltimo <= 1 {
		return 1
	}
	return c.altoUltimo - 1
}

// IrAlFinal vuelve a pegar la vista al final y a seguir las respuestas nuevas.
func (c *Chat) IrAlFinal() {
	c.scrolleado = false
	c.offset = c.maxOffset
}

// HayArriba y HayAbajo dicen si quedan líneas fuera de la ventana, para pintar
// el aviso de que hay más.
func (c *Chat) HayArriba() bool { return c.offset > 0 }
func (c *Chat) HayAbajo() bool  { return c.scrolleado && c.offset < c.maxOffset }

// OcultasArriba y OcultasAbajo cuentan las líneas fuera de la ventana por cada
// lado.
func (c *Chat) OcultasArriba() int { return c.offset }
func (c *Chat) OcultasAbajo() int  { return c.maxOffset - c.offset }
