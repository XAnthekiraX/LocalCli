// input.go — T-F004 (y T-F035): la línea de entrada de la interfaz principal.
//
// Fuente de verdad: frontend/01-domain/DOMAIN.md §1 (`input` "compone y envía
// la petición hacia la sesión activa"; "No valida reglas de negocio"),
// SPEC-INTERFAZ §Zonas 2 ("Una sola línea para escribir", que crece en varias
// al desbordar el ancho sin cortar el texto: T-F035) e INTERFACES §5 ("Si la
// sesión activa está generando, la entrada sigue operativa: escribir no bloquea
// ni cancela nada").
//
// Es solo composición y envío: quien confirma el envío es la vista raíz, que
// conoce la sesión activa. El componente se apoya en el `textarea` de bubbles,
// que envuelve por palabras y crece en alto: cuando el texto supera el ancho de
// la línea salta de renglón en vez de recortarse, y al pasar del tope se
// desplaza dentro de la ventana.
package tui

import (
	"strings"
	"unicode"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	rw "github.com/mattn/go-runewidth"
	"github.com/rivo/uniseg"
)

// anchoEntradaPorDefecto es el ancho que toma la línea antes de conocer el
// tamaño real de la terminal (o si nunca llega una `WindowSizeMsg`).
const anchoEntradaPorDefecto = 80

// altoMáximoEntrada es el tope de filas que la entrada ocupa en pantalla: al
// alcanzarlo, el texto se desplaza dentro de la ventana en vez de seguir
// comiendo el chat.
const altoMáximoEntrada = 6

// Entrada es la línea de texto de la interfaz principal. Envuelve en varias
// líneas lo que se escribe cuando no cabe, pero sigue siendo un solo párrafo:
// lo que se compone va hacia la sesión activa cuando se confirma. La precede el
// indicador del agente activo (T-F015-01).
type Entrada struct {
	campo textarea.Model
	Ancho int
	// Agente es el que está activo, `AgentePlan` o `AgenteBuild`: solo lo pinta,
	// no decide nada. Lo escribe la acción `agent_cycle`.
	Agente string
	// adjuntos recuerda las imágenes pegadas o arrastradas en la línea: se
	// muestran como [nombre.ext] y se expanden a su ruta al enviar.
	adjuntos adjuntos
}

// NuevaEntrada crea la línea enfocada, con su placeholder, sin numeración ni
// prompt propio (el indicador del agente ya compone el `> `, T-F015-01).
func NuevaEntrada() Entrada {
	campo := textarea.New()
	campo.Placeholder = "Escribe tu petición…"
	campo.Prompt = ""
	campo.ShowLineNumbers = false
	campo.CharLimit = 0
	campo.MaxHeight = altoMáximoEntrada
	campo.EndOfBufferCharacter = ' '
	// Estilos planos: el `textarea` trae por defecto una banda de fondo sobre la
	// línea del cursor; aquí la entrada es una sola pieza sin resaltado de línea,
	// como el input de una fila que era.
	plano := estiloCampoPlano()
	campo.FocusedStyle = plano
	campo.BlurredStyle = plano
	campo.Focus()
	e := Entrada{campo: campo, Agente: AgentePlan, Ancho: anchoEntradaPorDefecto}
	e.AjustarTeclasPropias()
	e.fijarAnchoCampo()
	return e
}

// estiloCampoPlano devuelve los estilos del campo sin color ni fondo: el texto
// se pinta tal cual y el cursor queda como bloque al parpadear.
func estiloCampoPlano() textarea.Style {
	base := lipgloss.NewStyle()
	return textarea.Style{
		Base:             base,
		CursorLine:       base,
		CursorLineNumber: base,
		EndOfBuffer:      base,
		LineNumber:       base,
		Placeholder:      base,
		Prompt:           base,
		Text:             base,
	}
}

// FijarAgente deja el agente activo que se pinta en el indicador. Vale
// cualquier nombre de la lista de agentes disponibles (p. ej. uno propio); solo
// un valor vacío cae en `plan`, para que el indicador nunca salga sin nombre
// (SPEC-INTERFAZ §Zonas 2). El ancho del campo se reajusta porque el indicador
// cambia de largo (`[plan]` no mide lo mismo que `[build]`).
func (e *Entrada) FijarAgente(agente string) {
	if strings.TrimSpace(agente) == "" {
		agente = AgentePlan
	}
	e.Agente = agente
	e.fijarAnchoCampo()
}

// anchoIndicador mide lo que ocupa el indicador del agente activo (`[plan] > `)
// delante del campo.
func anchoIndicador(agente string) int {
	return lipgloss.Width("[" + agente + "] > ")
}

// fijarAnchoCampo reparte el ancho disponible entre el indicador y el campo: el
// texto solo dispone de lo que queda tras el indicador, para que la línea no
// desborde la terminal. Sin ancho conocido no toca nada.
func (e *Entrada) fijarAnchoCampo() {
	if e.Ancho <= 0 {
		return
	}
	campoW := e.Ancho - anchoIndicador(e.Agente)
	if campoW < 1 {
		campoW = 1
	}
	e.campo.SetWidth(campoW)
	e.ajustarAlto()
}

// ajustarAlto deja el campo con las filas que ocupa el texto ya envuelto, sin
// pasar del tope: el textarea, por sí solo, no crece ni decrece.
func (e *Entrada) ajustarAlto() {
	filas := filasEnvueltas(e.campo.Value(), e.campo.Width())
	if filas < 1 {
		filas = 1
	}
	if filas > altoMáximoEntrada {
		filas = altoMáximoEntrada
	}
	e.campo.SetHeight(filas)
}

// filasEnvueltas cuenta las filas que ocupa un texto al envolverlo por palabras
// con el ancho dado. Replica el soft-wrap del `textarea` de bubbles (que no lo
// expone) para poder ajustar su alto: si la cuenta se quedara corta, el campo
// recortaría la primera línea, que es justo lo que se quiere evitar.
func filasEnvueltas(texto string, width int) int {
	if width < 1 {
		return 1
	}
	var (
		filas  = [][]rune{{}}
		word   = []rune{}
		row    int
		spaces int
	)
	for _, r := range texto {
		if unicode.IsSpace(r) {
			spaces++
		} else {
			word = append(word, r)
		}
		if spaces > 0 {
			if uniseg.StringWidth(string(filas[row]))+uniseg.StringWidth(string(word))+spaces > width {
				row++
				filas = append(filas, []rune{})
				filas[row] = append(filas[row], word...)
				filas[row] = append(filas[row], espacios(spaces)...)
				spaces, word = 0, nil
			} else {
				filas[row] = append(filas[row], word...)
				filas[row] = append(filas[row], espacios(spaces)...)
				spaces, word = 0, nil
			}
			continue
		}
		lastCharLen := rw.RuneWidth(word[len(word)-1])
		if uniseg.StringWidth(string(word))+lastCharLen > width {
			if len(filas[row]) > 0 {
				row++
				filas = append(filas, []rune{})
			}
			filas[row] = append(filas[row], word...)
			word = nil
		}
	}
	if uniseg.StringWidth(string(filas[row]))+uniseg.StringWidth(string(word))+spaces >= width {
		filas = append(filas, []rune{})
		filas[row+1] = append(filas[row+1], word...)
		filas[row+1] = append(filas[row+1], espacios(spaces+1)...)
	} else {
		filas[row] = append(filas[row], word...)
		filas[row] = append(filas[row], espacios(spaces+1)...)
	}
	return len(filas)
}

// espacios devuelve n espacios como runas.
func espacios(n int) []rune { return []rune(strings.Repeat(" ", n)) }

// Foco activa la línea y devuelve el comando del cursor, que es como bubbles
// hace parpadear el caret.
func (e *Entrada) Foco() tea.Cmd { return e.campo.Focus() }

// Desenfocar la apaga, por ejemplo mientras un modal se lleva el teclado.
func (e *Entrada) Desenfocar() { e.campo.Blur() }

// Texto devuelve lo escrito, con los tokens de imagen expandidos a su ruta
// real: es el texto que se envía y con el que trabaja el resto del harness.
func (e *Entrada) Texto() string { return e.adjuntos.Expandir(e.campo.Value()) }

// AnotarPegado convierte en tokens las rutas de imagen de un texto pegado o
// arrastrado y devuelve lo que hay que insertar en la línea.
func (e *Entrada) AnotarPegado(texto string) string { return e.adjuntos.Anotar(texto) }

// Limpiar vacía la línea. Va aparte para que quien envía decida cuándo
// borrarla: un fallo al abrir la sesión no puede perder lo escrito.
func (e *Entrada) Limpiar() {
	e.campo.Reset()
	e.adjuntos.Olvidar()
	e.ajustarAlto()
}

// FijarTexto reemplaza el contenido de la línea y deja el cursor al final. Lo
// usa la paleta de comandos para autocompletar el comando resaltado; al
// reemplazar todo, se olvidan los adjuntos previos.
func (e *Entrada) FijarTexto(texto string) {
	e.campo.SetValue(texto)
	e.campo.CursorEnd()
	e.adjuntos.Olvidar()
	e.ajustarAlto()
}

// FijarAncho adapta la línea al ancho del layout recibido. Un ancho no
// conocido (0 o menos) no toca nada: mejor la medida anterior que una línea
// invisible.
func (e *Entrada) FijarAncho(ancho int) {
	if ancho <= 0 {
		return
	}
	e.Ancho = ancho
	e.fijarAnchoCampo()
}

// Update reenvía el mensaje al campo y reajusta el alto: el texto puede haber
// ganado o perdido una fila. Las teclas de escritura llegan aquí siempre,
// también mientras la sesión está generando: escribir no bloquea ni cancela
// nada (INTERFACES §5), y el texto se conserva hasta que se confirma.
func (e *Entrada) Update(msg tea.Msg) (Entrada, tea.Cmd) {
	campo, cmd := e.campo.Update(msg)
	e.campo = campo
	e.ajustarAlto()
	return *e, cmd
}

// AjustarTeclasPropias deja al campo con la edición de línea completa: flechas
// (con ctrl+b/ctrl+f y alt+←/→), inicio/fin de línea envuelta (home/end y
// ctrl+e) y borrado por palabra y por delante/atrás del caret. El KeyResolver
// sigue mandando: las teclas que el mapa reclame se resuelven antes de llegar
// aquí, así que no hay colisión (por ejemplo, ctrl+a y ctrl+f son acciones de
// la app y nunca alcanzan el editor). Se anulan dos teclas del campo: `enter`
// (envía, no inserta un salto) y `ctrl+v` (el pegado llega como texto pegado, no
// como lectura asíncrona del portapapeles).
//
// Se aplica al construir la línea y se reaplica en cada cambio de tamaño: un
// campo reconstruido no debe recuperar sus atajos viejos.
func (e *Entrada) AjustarTeclasPropias() {
	km := e.campo.KeyMap
	anulada := key.NewBinding(key.WithKeys("ctrl+@"), key.WithHelp("ctrl+@", ""))
	km.InsertNewline = anulada
	km.Paste = anulada
	e.campo.KeyMap = km
}

// View pinta la línea con su indicador de agente a la izquierda y, detrás, el
// campo: el placeholder cuando está vacía, el texto y su cursor cuando no
// (SPEC-INTERFAZ §Zonas 2, "Indicador de agente a la izquierda del input",
// p. ej. `[plan] > █`). Las filas de continuación se sangran al ancho del
// indicador para que el texto quede alineado bajo la primera. Los tokens de
// imagen pegada se resaltan.
func (e *Entrada) View() string {
	indicador := IndicadorAgente(e.Agente)
	sangria := strings.Repeat(" ", lipgloss.Width(indicador))
	lineas := strings.Split(e.campo.View(), "\n")
	for i, l := range lineas {
		if i == 0 {
			lineas[i] = indicador + l
		} else {
			lineas[i] = sangria + l
		}
	}
	return e.adjuntos.Resaltar(strings.Join(lineas, "\n"))
}

// IndicadorAgente compone el indicador del agente activo: `[plan] > ` o
// `[build] > `. Vive aquí porque las dos vistas lo pintan —la principal a
// través de `Entrada.View` y la bienvenida en su línea de entrada— y tiene que
// ser la misma palabra en las dos (SPEC-INTERFAZ §Zonas 2).
func IndicadorAgente(agente string) string {
	return estiloIndicador.Render("[" + agente + "] > ")
}
