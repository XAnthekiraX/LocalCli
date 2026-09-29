// styles.go — T-F002: los estilos de la pantalla y el render de bloques.
//
// Fuente de verdad: FRONTEND.md §2 (styles.go: "estilos Lip Gloss y render de
// bloques"), frontend/01-domain/DOMAIN.md §2 ("El razonamiento se muestra en
// vivo, arriba de la respuesta, distinguible visualmente y ocultable sin
// detener la generación") y SPEC-INTERFAZ §Razonamiento del modelo ("Se
// distingue visualmente de la respuesta para que nunca se confunden", "El
// razonamiento nunca se mezcla visualmente con la respuesta final").
//
// Todo lo que pinta con color o forma vive aquí: si los estilos se reparten por
// los archivos, un cambio de color se convierte en una búsqueda por todo el
// paquete. Las funciones de render son puras —reciben texto y devuelven
// texto—, sin estado y sin tocar el bucle de Bubble Tea.
package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

// estilos de la vista, en un solo sitio para que la pantalla no tenga colores
// sueltos por los archivos.
var (
	estiloUsuario      = lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true)
	estiloAgente       = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	estiloIndicador    = lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true)
	estiloSistema      = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	estiloRazonamiento = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Italic(true)
	estiloTitulo       = lipgloss.NewStyle().Bold(true)
	estiloEtiqueta     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14"))
	estiloAviso        = lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Bold(true)
	estiloMarca        = lipgloss.NewStyle().Bold(true)
	// estiloAdjunto pinta el token de una imagen pegada o arrastrada en la línea
	// de entrada ([foto.png]), para distinguirla del texto escrito a mano.
	estiloAdjunto = lipgloss.NewStyle().Foreground(lipgloss.Color("13")).Bold(true)
	// estiloActividad pinta el indicador en vivo ([⠋ Pensando], [⠋ Usando
	// herramienta: X]): lo que reemplaza al vuelco crudo del razonamiento.
	estiloActividad = lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true)

	// estiloCopiado pinta el aviso transitorio [Copiado] que aparece arriba a la
	// derecha tras copiar una selección con el ratón (selection.go).
	estiloCopiado = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)

	// estiloGloboUsuario y estiloGloboAgente dibujan los globos del chat (T-F036):
	// borde redondeado con el color de cada interlocutor —azul para lo que escribes,
	// verde para lo que responde el agente/terminal— y un relleno de una columna a
	// cada lado. Ver `burbuja`.
	estiloGloboUsuario = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("12")).
				Padding(0, 1)
	estiloGloboAgente = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("10")).
				Padding(0, 1)

	// Estilos del rediseño de la vista principal (sidebar, caja de entrada y
	// iconos de las burbujas).
	//
	// estiloDivisor pinta la línea vertical entre la columna principal y el
	// sidebar y la horizontal entre el chat y la caja de entrada.
	estiloDivisor = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	// estiloSeccion titula los bloques del sidebar (CONTEXTO, TODO, LISTA DE
	// TAREAS).
	estiloSeccion = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14"))
	// estiloCajaBorde pinta el borde de la caja que envuelve la línea de
	// escritura.
	estiloCajaBorde = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	// estiloIconoAgente y estiloIconoUsuario colorean la cajita que precede
	// (agente) o sigue (usuario) a cada globo del chat.
	estiloIconoAgente  = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	estiloIconoUsuario = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))
)

// Códigos del realce de la selección con el ratón (selection.go). No se usa
// lipgloss.Render porque hay que envolver un TRAMO dentro de una línea ya
// pintada, no un string entero: `seleccionOn` enciende el video inverso (7) al
// entrar en el tramo y `seleccionOff` lo apaga (27) —solo el inverso, para no
// perder los colores que la línea ya traía—.
const (
	seleccionOn  = "\x1b[7m"
	seleccionOff = "\x1b[27m"
	// sgrReset cierra cualquier atributo abierto. Se usa antes del aviso
	// [Copiado] para que no herede el color de lo que tuviera debajo.
	sgrReset = "\x1b[0m"
)

// framesActividad son los glifos del indicador en vivo, en orden. Se recorren
// uno por latido: el movimiento es lo que dice «sigue trabajando» aunque no
// haya llegado texto. Es un solo sitio editable si se quiere otro giro.
var framesActividad = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// glifoActividad devuelve el glifo del frame pedido, en ciclo. Un frame
// negativo o desbordado se envuelve; sin frames, cae en «*» para no pintar
// vacío. Es una función pura, para probarla sin pantalla.
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

// renderActividad pinta el indicador en vivo: el glifo del frame entre
// corchetes junto a la etiqueta ([⠋ Pensando]). Es una función pura: recibe el
// frame y devuelve texto, sin tocar el bucle de Bubble Tea.
func renderActividad(frame int, etiqueta string) string {
	return "[" + glifoActividad(frame) + " " + etiqueta + "]"
}

// formatearTokens acorta el conteo de tokens en la unidad que se lee de un
// vistazo: entero por debajo del millar, miles con un decimal hasta diez mil y
// miles enteros a partir de ahí (54000 → «54k»). Es una función pura.
func formatearTokens(n int) string {
	switch {
	case n <= 0:
		return "0"
	case n < 1000:
		return fmt.Sprintf("%d", n)
	case n < 10000:
		return fmt.Sprintf("%.1fk", float64(n)/1000)
	default:
		return fmt.Sprintf("%.0fk", float64(n)/1000)
	}
}

// centrar coloca un bloque en el centro de la ventana, en horizontal y en
// vertical. Es la misma colocación para la bienvenida y para los tres modales
// ("Tres modales centrados", SPEC-INTERFAZ §Modales): sin geometría conocida no
// hay dónde centrar, así que el bloque se devuelve tal cual, y nunca se recorta
// (SPEC-INTERFAZ §Arte canónico).
func centrar(bloque string, ancho, alto int) string {
	if ancho <= 0 || alto <= 0 {
		return bloque + "\n"
	}
	return lipgloss.Place(ancho, alto, lipgloss.Center, lipgloss.Center, bloque)
}

// corteAncho devuelve el índice de byte donde termina el prefijo de `s` que
// cabe en `ancho` columnas de pantalla (los caracteres anchos, como un emoji o
// un ideograma, ocupan dos). Es la base de todo el recorte por columnas.
func corteAncho(s string, ancho int) int {
	w := 0
	for i, r := range s {
		rw := runewidth.RuneWidth(r)
		if w+rw > ancho {
			return i
		}
		w += rw
	}
	return len(s)
}

// recortar envuelve el texto en líneas de a lo sumo `ancho` columnas de
// pantalla, partiendo por columnas (no por runas): un emoji no desborda la
// línea. Es una función pura y la usan los globos del chat.
func recortar(texto string, ancho int) string {
	if ancho <= 0 {
		return texto
	}
	var out []string
	for _, linea := range strings.Split(texto, "\n") {
		for runewidth.StringWidth(linea) > ancho {
			corte := corteAncho(linea, ancho)
			if corte == 0 {
				// Un solo carácter más ancho que la columna: se deja tal cual
				// para no entrar en bucle.
				break
			}
			out = append(out, linea[:corte])
			linea = linea[corte:]
		}
		out = append(out, linea)
	}
	return strings.Join(out, "\n")
}

// truncar deja el texto en una sola línea de a lo sumo `ancho` columnas,
// recortando con «…» lo que sobre. Es para las filas del sidebar y el pie de la
// caja, donde una línea no puede partirse; el chat y los mensajes usan
// `recortar`, que envuelve. Es una función pura.
func truncar(texto string, ancho int) string {
	if ancho <= 0 {
		return ""
	}
	if runewidth.StringWidth(texto) <= ancho {
		return texto
	}
	if ancho == 1 {
		return "…"
	}
	return texto[:corteAncho(texto, ancho-1)] + "…"
}

// envolverConCursor reparte el texto en líneas de a lo sumo `ancho` columnas
// (recorte por runas, igual que `recortar`) y devuelve además en qué línea y en
// qué columna —en runas— cae la posición `pos` del cursor. Es lo que permite
// que la línea de entrada de la bienvenida salte de renglón sin cortar el texto
// y que el caret siga pintándose donde toca (T-F035). Sin ancho conocido no se
// parte nada: una sola línea con el cursor donde esté.
func envolverConCursor(texto string, pos, ancho int) (lineas []string, fila, col int) {
	r := []rune(texto)
	if pos < 0 {
		pos = 0
	}
	if pos > len(r) {
		pos = len(r)
	}
	if ancho <= 0 {
		return []string{texto}, 0, pos
	}
	var actual []rune
	for i := 0; i < len(r); i++ {
		if len(actual) == ancho {
			lineas = append(lineas, string(actual))
			actual = nil
		}
		if i == pos {
			fila, col = len(lineas), len(actual)
		}
		actual = append(actual, r[i])
	}
	if pos == len(r) {
		fila, col = len(lineas), len(actual)
	}
	return append(lineas, string(actual)), fila, col
}

// renderReasoning pinta el texto crudo del razonamiento. Devuelve "" si no está
// revelado o si no hay nada que decir: una cabecera vacía solo estorba.
//
// Por defecto el razonamiento no se vuelca: en su lugar la vista pinta el
// indicador en vivo ([⠋ Pensando]). Revelarlo con `Ctrl+R` es solo pintura: no
// toca la generación ni el texto acumulado, de modo que volver a mostrarlo lo
// recupera entero (frontend/01-domain/DOMAIN.md §2). Cuando se revela, lleva el
// estilo propio del razonamiento, atenuado y en cursiva, para distinguirse de
// la respuesta.
func renderReasoning(texto string, revelado bool, noDisponible bool, ancho int) string {
	if !revelado {
		return ""
	}
	if strings.TrimSpace(texto) == "" {
		if noDisponible {
			return estiloRazonamiento.Render("(el modelo no entregó razonamiento)")
		}
		return ""
	}
	return estiloRazonamiento.Render(recortar(texto, ancho))
}

// formatearDuracion compone el tiempo de una respuesta en la unidad que se lee
// de un vistazo: milisegundos por debajo del segundo, décimas de segundo por
// debajo del minuto, y minutos y segundos a partir de ahí. Es una función pura,
// para probarla sin pantalla (frontend/05-quality/TESTING.md §1).
func formatearDuracion(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	switch {
	case d < time.Second:
		return fmt.Sprintf("%d ms", d.Milliseconds())
	case d < time.Minute:
		return fmt.Sprintf("%.1f s", d.Seconds())
	default:
		total := int(d.Seconds())
		return fmt.Sprintf("%d m %d s", total/60, total%60)
	}
}

// sufijoDuracion pinta el tiempo que tardó una respuesta, atenuado para no
// confundirse con lo que dijo el modelo. Sin duración medida no pinta nada: un
// historial recargado no la trae (la base no guarda el tiempo).
func sufijoDuracion(d time.Duration) string {
	if d <= 0 {
		return ""
	}
	return estiloSistema.Render("(" + formatearDuracion(d) + ")")
}

// renderIntercambio pinta un intercambio del chat: el razonamiento, solo si se
// revela (`Ctrl+R`), arriba de la respuesta y separado de ella por una línea en
// blanco para que nunca se mezclen visualmente (SPEC-INTERFAZ §Razonamiento del
// modelo). Por defecto el razonamiento no se vuelca: la vista lo sustituye por
// el indicador en vivo mientras el turno corre. Con respuesta vacía no pinta
// hueco: la respuesta que todavía no ha empezado no deja sitio reservado.
func renderIntercambio(razonamiento, respuesta string, revelar bool, noDisponible bool, ancho int) string {
	var partes []string
	if r := renderReasoning(razonamiento, revelar, noDisponible, ancho); r != "" {
		partes = append(partes, r)
	}
	if respuesta != "" {
		partes = append(partes, estiloAgente.Render(recortar(respuesta, ancho)))
	}
	if len(partes) == 0 {
		return ""
	}
	return strings.Join(partes, "\n\n")
}

// anchoGlobo es el ancho disponible para el CONTENIDO del globo: el ancho total
// menos el borde (dos columnas) y el relleno horizontal (dos más). Quien compone
// el texto de dentro envuelve a esta medida.
func anchoGlobo(ancho int) int {
	n := ancho - 4
	if n < 1 {
		n = 1
	}
	return n
}

// burbuja envuelve en el globo de un rol un contenido que YA viene envuelto y
// pintado (razonamiento, respuesta). No vuelve a medir el texto: solo pone el
// borde y el color del interlocutor, que lipgloss aplica sobre el ancho visible
// aunque el contenido traiga estilos. Un contenido vacío no deja globo.
func burbuja(rol Rol, contenido string) string {
	if strings.TrimSpace(contenido) == "" {
		return ""
	}
	if rol == RolUsuario {
		return estiloGloboUsuario.Render(contenido)
	}
	return estiloGloboAgente.Render(contenido)
}

// glifoIcono es el avatar que precede (agente) o sigue (usuario) a cada globo
// del chat.
const glifoIcono = "▣"

// anchoIcono es lo que ocupa el icono más su separación del globo: las columnas
// que hay que descontar del ancho del chat al medir una burbuja.
const anchoIcono = 2

// margenChat son las columnas que los globos y las líneas del chat dejan libres
// a cada lado, para no quedar pegados al borde de la columna.
const margenChat = 1

// iconoDeRol dibuja el avatar de un rol: el glifo con el color de quien habla.
// Es una función pura.
func iconoDeRol(rol Rol) string {
	if rol == RolUsuario {
		return estiloIconoUsuario.Render(glifoIcono)
	}
	return estiloIconoAgente.Render(glifoIcono)
}

// lineasALaDerecha coloca un bloque pegado al borde derecho de un ancho dado,
// rellenando con espacios a su izquierda. Es lo que alinea el globo del usuario
// (con su icono detrás) contra el margen derecho del chat. Sin ancho conocido
// no mueve nada.
func lineasALaDerecha(bloque string, ancho int) string {
	if ancho <= 0 {
		return bloque
	}
	return lipgloss.PlaceHorizontal(ancho, lipgloss.Right, bloque)
}

// cajaConBorde envuelve las líneas dadas en una caja de borde redondeado con una
// columna de relleno a cada lado, de ancho total EXACTO `ancho`: cada línea se
// recorta (contando ANSI) al ancho interno. Se dibuja a mano —en vez de con el
// borde de Lip Gloss— para que el ancho no dependa de cómo reparte el estilo el
// relleno y el borde, y las columnas del layout queden rectas. Es pura.
func cajaConBorde(ancho int, lineas []string) string {
	if ancho < 5 {
		return strings.Join(lineas, "\n")
	}
	interno := ancho - 4
	barra := estiloCajaBorde.Render("│")
	var b strings.Builder
	b.WriteString(estiloCajaBorde.Render("╭" + strings.Repeat("─", ancho-2) + "╮"))
	for _, l := range lineas {
		b.WriteString("\n" + barra + " " + recortarColumnas(l, interno) + " " + barra)
	}
	b.WriteString("\n" + estiloCajaBorde.Render("╰"+strings.Repeat("─", ancho-2)+"╯"))
	return b.String()
}
