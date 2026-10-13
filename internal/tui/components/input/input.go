package input

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"localcli/internal/tui/components/adjuntos"
	"localcli/internal/tui/render"
)

const (
	// placeholder es lo que se enseña mientras la línea está vacía.
	placeholder = "Escribe para iniciar la conversacion"
	// cursor es la marca de la posición de edición.
	cursor = "▌"
	// altoMaximo es el tope de filas que ocupa el texto: al pasarlo, la ventana
	// se desplaza en vez de seguir comiendo el chat.
	altoMaximo = 6
	// anchoMinimo evita que la caja se componga por debajo de su propio aire.
	anchoMinimo = 20
)

// Capacidad es el estado de una capacidad del modelo: soportada, no soportada o
// todavía desconocida. La caja tiene que poder pintar «no lo sé» como estado de
// primera clase, sin confundirlo con «no lo soporta».
type Capacidad int

const (
	Desconocida Capacidad = iota
	Soportada
	NoSoportada
)

// Capacidades agrupa lo que la caja enseña del modelo en uso.
type Capacidades struct {
	Herramientas Capacidad
	Vision       Capacidad
	Razonamiento Capacidad
}

// Rect es un rectángulo de celdas dentro de la caja, en coordenadas locales: la
// fila 0 es su banda superior. Quien compone la vista lo traslada a la pantalla.
type Rect struct {
	Fila    int
	Columna int
	Ancho   int
	Alto    int
}

func (r Rect) contiene(fila, columna int) bool {
	return r.Alto > 0 && r.Ancho > 0 &&
		fila >= r.Fila && fila < r.Fila+r.Alto &&
		columna >= r.Columna && columna < r.Columna+r.Ancho
}

// EnviadoMsg es la petición que la caja emite al confirmar con `Enter`. Lleva el
// texto ya expandido (sin tokens); quien la recibe conoce la sesión activa.
type EnviadoMsg struct{ Texto string }

// Model es el estado de la caja de entrada.
type Model struct {
	Texto string
	Pos   int

	Agente      string
	Modelo      string
	Motor       string
	Capacidades Capacidades

	// Pensar es el interruptor de razonamiento: llega apagado y solo se enseña
	// —y se puede pulsar— si el modelo declara la capacidad.
	Pensar bool

	// Tokens es el consumo del turno; sin consumo no se pinta el conteo.
	Tokens int

	adjuntos adjuntos.Adjuntos

	ancho      int
	filas      int
	rectCaja   Rect
	rectPensar Rect
}

// Nuevo deja la caja lista con su agente, modelo y capacidades.
func Nuevo(agente, modelo, motor string, capacidades Capacidades) Model {
	return Model{Agente: agente, Modelo: modelo, Motor: motor, Capacidades: capacidades}
}

// Init no arranca nada: la caja no tiene suscripciones propias.
func (m Model) Init() tea.Cmd { return nil }

// Update atiende la edición de la línea. `Enter` emite la petición (EnviadoMsg)
// sin insertar un salto; el resto mueve el cursor y edita el texto.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	tecla, esTecla := msg.(tea.KeyMsg)
	if !esTecla {
		return m, nil
	}
	switch tecla.Type {
	case tea.KeyEnter:
		if texto := m.Peticion(); texto != "" {
			return m, emitir(texto)
		}
	case tea.KeyRunes:
		// Un pegado llega como un único KeyRunes con Paste en alto (bracketed
		// paste); se anota para mostrar tokens en vez de la ruta cruda. Lo
		// tecleado letra a letra se escribe tal cual.
		if tecla.Paste {
			m.InsertarPegado(string(tecla.Runes))
		} else {
			m.Escribir(string(tecla.Runes))
		}
	case tea.KeySpace:
		m.Escribir(" ")
	case tea.KeyBackspace:
		m.Borrar()
	case tea.KeyDelete:
		m.BorrarAdelante()
	case tea.KeyLeft:
		m.Izquierda()
	case tea.KeyRight:
		m.Derecha()
	case tea.KeyHome, tea.KeyCtrlB:
		m.Inicio()
	case tea.KeyEnd, tea.KeyCtrlE:
		m.Fin()
	}
	return m, nil
}

func emitir(texto string) tea.Cmd {
	return func() tea.Msg { return EnviadoMsg{Texto: texto} }
}

// Valor devuelve lo escrito con los tokens expandidos a su valor real: es lo que
// se envía a la sesión, nunca el token.
func (m Model) Valor() string { return m.adjuntos.Expandir(m.Texto) }

// Peticion devuelve lo escrito listo para enviar, sin espacios sobrantes.
func (m Model) Peticion() string { return strings.TrimSpace(m.Valor()) }

// Escribir inserta texto en la posición del cursor y lo avanza.
func (m *Model) Escribir(texto string) {
	m.acotar()
	runas := []rune(m.Texto)
	insertadas := []rune(texto)
	nuevas := make([]rune, 0, len(runas)+len(insertadas))
	nuevas = append(nuevas, runas[:m.Pos]...)
	nuevas = append(nuevas, insertadas...)
	nuevas = append(nuevas, runas[m.Pos:]...)
	m.Texto = string(nuevas)
	m.Pos += len(insertadas)
}

// Borrar quita la runa anterior al cursor.
func (m *Model) Borrar() {
	runas := []rune(m.Texto)
	m.acotar()
	if m.Pos == 0 {
		return
	}
	restantes := append([]rune{}, runas[:m.Pos-1]...)
	m.Texto = string(append(restantes, runas[m.Pos:]...))
	m.Pos--
}

// BorrarAdelante quita la runa del cursor.
func (m *Model) BorrarAdelante() {
	runas := []rune(m.Texto)
	m.acotar()
	if m.Pos >= len(runas) {
		return
	}
	m.Texto = string(append(runas[:m.Pos], runas[m.Pos+1:]...))
}

// Izquierda mueve el cursor una runa atrás.
func (m *Model) Izquierda() {
	m.Pos--
	m.acotar()
}

// Derecha mueve el cursor una runa adelante.
func (m *Model) Derecha() {
	m.Pos++
	m.acotar()
}

// Inicio lleva el cursor al principio de la línea.
func (m *Model) Inicio() { m.Pos = 0 }

// Fin lleva el cursor al final de la línea.
func (m *Model) Fin() { m.Pos = len([]rune(m.Texto)) }

// Limpiar vacía la línea, olvida los adjuntos y manda el cursor al principio.
func (m *Model) Limpiar() {
	m.Texto = ""
	m.Pos = 0
	m.adjuntos.Olvidar()
}

// AlternarPensar cambia el interruptor de razonamiento; el clic del ratón lo usa.
// Solo alterna si el modelo declara la capacidad.
func (m *Model) AlternarPensar() {
	if m.Capacidades.Razonamiento == Soportada {
		m.Pensar = !m.Pensar
	}
}

// AnotarPegado convierte en tokens las rutas de un texto pegado o arrastrado y
// devuelve lo que hay que insertar en la línea.
func (m *Model) AnotarPegado(texto string) string { return m.adjuntos.Anotar(texto) }

// InsertarPegado anota el texto pegado y lo escribe en la línea.
func (m *Model) InsertarPegado(texto string) { m.Escribir(m.adjuntos.Anotar(texto)) }

// RectCaja devuelve el rectángulo local de la caja, sin el conteo de tokens.
func (m Model) RectCaja() Rect { return m.rectCaja }

// RectPensar devuelve el rectángulo local del interruptor de razonamiento; vacío
// si el modelo no declara la capacidad.
func (m Model) RectPensar() Rect { return m.rectPensar }

// SobrePensar dice si una celda local de la caja cae sobre el interruptor.
func (m Model) SobrePensar(fila, columna int) bool { return m.rectPensar.contiene(fila, columna) }

func (m *Model) acotar() {
	switch {
	case m.Pos < 0:
		m.Pos = 0
	case m.Pos > len([]rune(m.Texto)):
		m.Pos = len([]rune(m.Texto))
	}
}

// View pinta la caja completa: fondo propio con una celda de aire arriba, abajo
// y a cada lado, sin glifos de borde; el texto (o su placeholder) con el cursor,
// una fila en blanco y el pie con el agente, el modelo y sus capacidades. Si hay
// consumo, el conteo de tokens va justo debajo, sobre el fondo de la pantalla.
// Registra los rectángulos locales de la caja y del interruptor (ratón).
func (m *Model) View(ancho int) string {
	if ancho < anchoMinimo {
		ancho = anchoMinimo
	}
	m.ancho = ancho
	interior := ancho - 2
	if interior < 1 {
		interior = 1
	}

	filas, filaCursor := m.filasInteriores(interior)
	visibles, _ := ventanaFilas(filas, filaCursor, altoMaximo)
	m.filas = len(visibles)

	pie := m.pie(interior)

	banda := render.PintarFondo("", ancho, render.FondoCaja)
	var b strings.Builder
	b.WriteString(banda)
	for _, fila := range visibles {
		b.WriteString("\n" + render.PintarFondo(" "+render.RecortarColumnas(fila, interior), ancho, render.FondoCaja))
	}
	b.WriteString("\n" + render.PintarFondo("", ancho, render.FondoCaja))
	filaPie := 1 + len(visibles) + 1
	for _, linea := range pie.filas {
		b.WriteString("\n" + render.PintarFondo(" "+render.RecortarColumnas(linea, interior), ancho, render.FondoCaja))
	}
	b.WriteString("\n" + banda)

	m.rectCaja = Rect{Fila: 0, Columna: 0, Ancho: ancho, Alto: filaPie + len(pie.filas) + 1}
	m.rectPensar = Rect{}
	if pie.tienePensar {
		m.rectPensar = Rect{
			Fila:    filaPie + pie.pensar.Fila,
			Columna: 1 + pie.pensar.Columna,
			Ancho:   pie.pensar.Ancho,
			Alto:    1,
		}
	}

	if m.Tokens > 0 {
		b.WriteString("\n" + render.PintarFondo(" "+render.EstiloSutil.Render("tokens: "+formatearTokens(m.Tokens)), ancho, render.FondoPantalla))
	}
	return b.String()
}

// filasInteriores envuelve lo escrito y coloca el cursor. Devuelve las filas y
// en cuál de ellas está el cursor.
func (m Model) filasInteriores(ancho int) ([]string, int) {
	if m.Peticion() == "" {
		return []string{render.EstiloSutil.Render(`"` + placeholder + `"`)}, 0
	}
	envueltas, filaCursor, columna := render.EnvolverConCursor(m.Texto, m.Pos, ancho)
	lineas := make([]string, 0, len(envueltas))
	for i, linea := range envueltas {
		if i != filaCursor {
			lineas = append(lineas, m.adjuntos.Estilizar(linea))
			continue
		}
		runas := []rune(linea)
		lineas = append(lineas, m.adjuntos.Estilizar(string(runas[:columna]))+
			cursor+m.adjuntos.Estilizar(string(runas[columna:])))
	}
	return lineas, filaCursor
}

// ventanaFilas recorta la lista de filas al tope, desplazándola para que el
// cursor quede visible. Devuelve la ventana y su índice de inicio.
func ventanaFilas(filas []string, cursor, tope int) ([]string, int) {
	if tope <= 0 || len(filas) <= tope {
		return filas, 0
	}
	inicio := 0
	if cursor >= tope {
		inicio = cursor - tope + 1
	}
	if inicio+tope > len(filas) {
		inicio = len(filas) - tope
	}
	if inicio < 0 {
		inicio = 0
	}
	return filas[inicio : inicio+tope], inicio
}

// pieLayout es el pie de la caja ya envuelto a un ancho: sus filas y, si el
// modelo declara razonamiento, el rectángulo local del interruptor entre ellas.
type pieLayout struct {
	filas       []string
	pensar      Rect
	tienePensar bool
}

// pie compone la línea de estado —agente, modelo y capacidades— envolviéndola a
// `ancho`: en una caja estrecha el interruptor de razonamiento no se pierde, cae
// a la fila siguiente. Registra dónde quedó el interruptor para que el ratón
// sepa dónde pulsarlo.
func (m Model) pie(ancho int) pieLayout {
	sep := render.EstiloSutil.Render(" · ")
	chips := []render.Chip{
		{Texto: render.EstiloAgente.Render("[" + m.Agente + "]")},
		{Texto: render.EstiloBlanco.Render(rotuloModelo(m))},
		{Texto: chipTool(m.Capacidades.Herramientas)},
	}
	if m.Capacidades.Razonamiento == Soportada {
		chips = append(chips, render.Chip{Texto: chipPensar(m.Pensar), Clave: "pensar"})
	}
	chips = append(chips,
		render.Chip{Texto: chipVision(m.Capacidades.Vision)},
		render.Chip{Texto: chipTexto()},
	)
	filas, posiciones := render.EnvolverChips(chips, sep, ancho)
	pie := pieLayout{filas: filas}
	if punto, ok := posiciones["pensar"]; ok {
		pie.tienePensar = true
		pie.pensar = Rect{Fila: punto.Fila, Columna: punto.Columna, Ancho: render.Ancho(chipPensar(true)), Alto: 1}
	}
	return pie
}

func rotuloModelo(m Model) string {
	switch {
	case m.Modelo == "":
		return "—"
	case m.Motor == "":
		return m.Modelo
	default:
		return m.Modelo + " (" + m.Motor + ")"
	}
}

// chipTool marca el uso de herramientas: verde si las usa, rojo si no y atenuado
// mientras no se sabe.
func chipTool(c Capacidad) string {
	switch c {
	case Soportada:
		return render.EstiloCapaz.Render("tool [*]")
	case NoSoportada:
		return render.EstiloIncapaz.Render("tool [x]")
	default:
		return render.EstiloSutil.Render("tool [?]")
	}
}

// chipPensar es el interruptor de razonamiento: `pensar [x]` encendido,
// `pensar [ ]` apagado. Solo se compone si el modelo declara la capacidad.
func chipPensar(activo bool) string {
	marca := "[ ]"
	if activo {
		marca = "[x]"
	}
	return render.EstiloBlanco.Render("pensar " + marca)
}

// chipVision marca la visión: verde si la soporta, rojo si no y `[v?]` si aún no
// se sabe.
func chipVision(c Capacidad) string {
	switch c {
	case Soportada:
		return render.EstiloCapaz.Render("[v]")
	case NoSoportada:
		return render.EstiloIncapaz.Render("[v]")
	default:
		return render.EstiloSutil.Render("[v?]")
	}
}

// chipTexto marca que el modelo acepta texto: `[T]` siempre.
func chipTexto() string { return render.EstiloSutil.Render("[T]") }

// formatearTokens acorta el conteo en la unidad que se lee de un vistazo: entero
// por debajo del millar, miles con un decimal hasta diez mil y miles enteros a
// partir de ahí (54000 → «54k»). Es una función pura.
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
