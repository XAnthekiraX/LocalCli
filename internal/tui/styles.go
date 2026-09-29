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

	// estiloSutil pinta el texto de apoyo con un gris azulado propio: el
	// placeholder de la entrada y el modelo/proveedor de la línea de estado.
	estiloSutil = lipgloss.NewStyle().Foreground(lipgloss.Color("#828BB8"))

	// estiloBlanco destaca lo que se lee de un vistazo: el texto de los mensajes
	// del chat, las teclas de la barra de pistas y el modelo en uso en la línea
	// de estado de la entrada.
	estiloBlanco = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF"))

	// estiloCapaz y estiloIncapaz marcan las capacidades del modelo en el pie del
	// input: en verde lo que tiene, en rojo lo que no.
	estiloCapaz   = lipgloss.NewStyle().Foreground(lipgloss.Color("#4CEE75"))
	estiloIncapaz = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5F5F"))

	// Estilos del rediseño de la vista principal (sidebar, caja de entrada y
	// iconos de las burbujas).
	//
	// estiloSeccion titula los bloques del sidebar (CONTEXTO, TODO, LISTA DE
	// TAREAS).
	estiloSeccion = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14"))
)

// Paleta de fondos de las superficies de la TUI. Se declaran aparte de los
// estilos porque no son «estilos de texto» sino el color del área de cada zona:
// la capa común (marco.go) los usa para que cada celda del layout tenga fondo y
// el área visual sea continua. Son colores hex; el perfil de color vigente los
// degrada solo si la terminal no soporta color verdadero.
var (
	fondoApp   = lipgloss.Color("#17191F")
	fondoPanel = lipgloss.Color("#1E2030")
	fondoCaja  = lipgloss.Color("#1E2030")
	// fondoChat es el fondo del área del chat: cada fila de mensaje es una
	// superficie de ancho completo sobre este color, con el color del rol detrás
	// del texto y sin icono (ver `bloqueChat`).
	fondoChat = lipgloss.Color("#1E2030")
	// Los mensajes del chat son filas de ancho completo con una franja del color
	// de cada rol: azul la del usuario (en su borde derecho) y verde la del
	// agente (en el izquierdo); sin icono ni glifos de adorno (ver `bloqueChat`).
	franjaUsuario = lipgloss.Color("#606CD5")
	franjaAgente  = lipgloss.Color("#4CEE75")
	// fondoBienvenida es el fondo de la caja de entrada de la bienvenida: la caja
	// entera —cuerpo y bandas— comparte un solo color.
	fondoBienvenida = lipgloss.Color("#222436")
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

// truncarPorLaIzquierda deja el final del texto, que es la parte que lo
// identifica, y pone «…» delante de lo que no cabe. Es el espejo de `truncar` y
// existe para las rutas: `/home/user/Documentos/LocalCli` recortado por la
// derecha deja `/home/user/Documentos/Lo…`, que no dice nada, mientras que por
// la izquierda queda `[…/Documentos/LocalCli]`, que sí. Mide por columnas de
// pantalla, igual que su espejo, y es una función pura.
func truncarPorLaIzquierda(texto string, ancho int) string {
	if ancho <= 0 {
		return ""
	}
	if runewidth.StringWidth(texto) <= ancho {
		return texto
	}
	if ancho == 1 {
		return "…"
	}
	// Se recorre desde el final acumulando el ancho hasta que ya no quepa la
	// runa siguiente; desde `i` hacia abajo es el sufijo que se queda.
	r := []rune(texto)
	i, w := len(r), 0
	for i > 0 {
		rw := runewidth.RuneWidth(r[i-1])
		if w+rw > ancho-1 {
			break
		}
		w += rw
		i--
	}
	return "…" + string(r[i:])
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
		partes = append(partes, estiloBlanco.Render(recortar(respuesta, ancho)))
	}
	if len(partes) == 0 {
		return ""
	}
	return strings.Join(partes, "\n\n")
}

// bloqueChat pinta un mensaje del chat como una fila de ancho completo sobre el
// fondo del chat (#1E2030): una franja de dos columnas con el color del rol en
// su lado —el usuario a la derecha, el agente a la izquierda— y el texto dentro,
// con aire de una celda arriba, abajo y a cada lado. No lleva icono. Un contenido
// vacío no deja bloque.
func bloqueChat(rol Rol, contenido string, ancho int) string {
	if strings.TrimSpace(contenido) == "" {
		return ""
	}
	if ancho < 4 {
		ancho = 4
	}
	franjaColor := franjaAgente
	if rol == RolUsuario {
		franjaColor = franjaUsuario
	}
	// La franja de color del rol (una columna); el resto de la fila es fondo del
	// chat. El texto dispone del ancho menos la franja y el relleno de una columna
	// a cada lado.
	const franja = 1
	interno := ancho - franja - 2
	if interno < 1 {
		interno = 1
	}
	tira := pintarFondo("", franja, franjaColor)
	vacia := pintarFondo("", interno+2, fondoChat)
	// La franja va siempre en el lado del rol.
	fila := func(centro string) string {
		if rol == RolUsuario {
			return centro + tira
		}
		return tira + centro
	}
	// Aire arriba y abajo: una fila en blanco antes y después del texto.
	out := []string{fila(vacia)}
	for _, l := range strings.Split(contenido, "\n") {
		linea := " " + recortarColumnas(l, interno) + " "
		out = append(out, fila(pintarFondo(linea, interno+2, fondoChat)))
	}
	out = append(out, fila(vacia))
	return strings.Join(out, "\n")
}

// lineasALaDerecha coloca un bloque pegado al borde derecho de un ancho dado,
// rellenando con espacios a su izquierda. Sin ancho conocido no mueve nada.
func lineasALaDerecha(bloque string, ancho int) string {
	if ancho <= 0 {
		return bloque
	}
	return lipgloss.PlaceHorizontal(ancho, lipgloss.Right, bloque)
}

// cajaConBorde envuelve las líneas dadas en la caja de la línea de entrada: una
// superficie de ancho total EXACTO `ancho`, con fondo propio, una columna de
// relleno a cada lado y una banda de separación arriba y abajo. No usa glifos de
// borde (nada de ╭╮╰╯│): la caja se lee por su fondo, y así la selección con el
// ratón cubre el área completa y la copia no arrastra caracteres de adorno. Cada
// línea se recorta (contando ANSI) al ancho interno. Es pura.
func cajaConBorde(ancho int, lineas []string) string {
	if ancho < 3 {
		return strings.Join(lineas, "\n")
	}
	interno := ancho - 2
	// Las filas de arriba y abajo de la caja son fondo de pantalla, no una banda
	// de otro color: separan el componente del resto con el mismo fondo.
	banda := pintarFondo("", ancho, fondoApp)
	// Aire dentro de la caja: una fila en blanco arriba y otra abajo del texto.
	vacia := pintarFondo("", ancho, fondoCaja)
	var b strings.Builder
	b.WriteString(banda)
	b.WriteString("\n" + vacia)
	for _, l := range lineas {
		b.WriteString("\n" + pintarFondo(" "+recortarColumnas(l, interno), ancho, fondoCaja))
	}
	b.WriteString("\n" + vacia)
	b.WriteString("\n" + banda)
	return b.String()
}

// cajaEntradaBienvenida envuelve el contenido de la caja de entrada de la
// bienvenida. Es la misma forma que `cajaConBorde` —relleno de una columna a
// cada lado y una banda de separación arriba y abajo, sin glifos—, pero con un
// solo color de fondo (#222436) para la caja entera: cuerpo y bandas. Cada línea
// se recorta (contando ANSI) al ancho interno. Es pura.
func cajaEntradaBienvenida(ancho int, lineas []string) string {
	if ancho < 3 {
		return strings.Join(lineas, "\n")
	}
	interno := ancho - 2
	banda := pintarFondo("", ancho, fondoBienvenida)
	var b strings.Builder
	b.WriteString(banda)
	for _, l := range lineas {
		b.WriteString("\n" + pintarFondo(" "+recortarColumnas(l, interno), ancho, fondoBienvenida))
	}
	b.WriteString("\n" + banda)
	return b.String()
}
