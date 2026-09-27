// input.go — T-F004: la línea de entrada de la interfaz principal.
//
// Fuente de verdad: frontend/01-domain/DOMAIN.md §1 (`input` "compone y envía
// la petición hacia la sesión activa"; "No valida reglas de negocio"),
// SPEC-INTERFAZ §Zonas 2 ("Una sola línea para escribir", "Escribe hacia la
// sesión activa") e INTERFACES §5 ("Si la sesión activa está generando, la
// entrada sigue operativa: escribir no bloquea ni cancela nada").
//
// Es solo composición y envío: quien confirma el envío es la vista raíz, que
// conoce la sesión activa. El componente es el campo de bubbles de una fila;
// aquí no hay historial ni segunda línea.
package tui

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// Entrada es la línea de texto de la interfaz principal. Una sola fila: lo que
// se escribe va hacia la sesión activa cuando se confirma. La precede el
// indicador del agente activo (T-F015-01).
type Entrada struct {
	campo textinput.Model
	Ancho int
	// Agente es el que está activo, `AgentePlan` o `AgenteBuild`: solo lo pinta,
	// no decide nada. Lo escribe la acción `agent_cycle`.
	Agente string
}

// NuevaEntrada crea la línea enfocada, con su placeholder y de una sola fila.
func NuevaEntrada() Entrada {
	campo := textinput.New()
	campo.Placeholder = "Escribe tu petición…"
	// El indicador del agente ya compone el `> ` (T-F015-01), así que el
	// prompt propio del campo se apaga: sin esto la línea saldría
	// `[plan] > > …`, con la flecha duplicada (SPEC-INTERFAZ §Zonas 2: el
	// render canónico es `[plan] > █`).
	campo.Prompt = ""
	campo.Focus()
	campo.Width = 80
	e := Entrada{campo: campo, Agente: AgentePlan}
	e.AjustarTeclasPropias()
	return e
}

// FijarAgente deja el agente activo que se pinta en el indicador. Un valor
// inesperado se trata como `plan`: el indicador nunca sale vacío ni inventa un
// tercer agente (SPEC-INTERFAZ §Zonas 2).
func (e *Entrada) FijarAgente(agente string) {
	if agente != AgenteBuild {
		agente = AgentePlan
	}
	e.Agente = agente
}

// Foco activa la línea y devuelve el comando del cursor, que es como bubbles
// hace parpadear el caret.
func (e *Entrada) Foco() tea.Cmd {
	e.campo.Focus()
	return textinput.Blink
}

// Desenfocar la apaga, por ejemplo mientras un modal se lleva el teclado.
func (e *Entrada) Desenfocar() { e.campo.Blur() }

// Texto devuelve lo escrito hasta ahora, tal cual.
func (e *Entrada) Texto() string { return e.campo.Value() }

// Limpiar vacía la línea. Va aparte para que quien envía decida cuándo
// borrarla: un fallo al abrir la sesión no puede perder lo escrito.
func (e *Entrada) Limpiar() { e.campo.Reset() }

// FijarTexto reemplaza el contenido de la línea y deja el cursor al final. Lo
// usa la paleta de comandos para autocompletar el comando resaltado.
func (e *Entrada) FijarTexto(texto string) {
	e.campo.SetValue(texto)
	e.campo.CursorEnd()
}

// FijarAncho adapta la línea al ancho del layout recibido. Un ancho no
// conocido (0 o menos) no toca nada: mejor la medida anterior que una línea
// invisible.
func (e *Entrada) FijarAncho(ancho int) {
	if ancho <= 0 {
		return
	}
	e.Ancho = ancho
	e.campo.Width = ancho
}

// Update reenvía el mensaje al campo. Las teclas de escritura llegan aquí
// siempre, también mientras la sesión está generando: escribir no bloquea ni
// cancela nada (INTERFACES §5), y el texto se conserva hasta que se confirma.
func (e *Entrada) Update(msg tea.Msg) (Entrada, tea.Cmd) {
	campo, cmd := e.campo.Update(msg)
	e.campo = campo
	return *e, cmd
}

// AjustarTeclasPropias deja al campo textinput con la edición de línea completa:
// flechas (con ctrl+b/ctrl+f y ctrl+←/→), home/end (con ctrl+a/ctrl+e) y borrado
// por palabra y por delante/atrás del caret. El KeyResolver sigue mandando: las
// teclas que el mapa reclame se resuelven antes de llegar aquí, así que no hay
// colisión (por ejemplo, ctrl+a y ctrl+f son acciones de la app y nunca alcanzan
// el editor). Solo se anulan las teclas que no queremos del campo: pegar y las
// sugerencias de autocompletado, que aquí no se usan.
//
// Se aplica al construir la línea y se reaplica en cada cambio de tamaño: un
// textinput reconstruido no debe recuperar sus atajos viejos.
func (e *Entrada) AjustarTeclasPropias() {
	km := e.campo.KeyMap
	uno := key.NewBinding(key.WithKeys("ctrl+@"), key.WithHelp("ctrl+@", ""))
	km.Paste = uno
	km.AcceptSuggestion = uno
	km.NextSuggestion = uno
	km.PrevSuggestion = uno
	e.campo.KeyMap = km
}

// View pinta la línea con su indicador de agente a la izquierda y, detrás, el
// campo: el placeholder cuando está vacía, el texto y su cursor cuando no
// (SPEC-INTERFAZ §Zonas 2, "Indicador de agente a la izquierda del input",
// p. ej. `[plan] > █`).
func (e *Entrada) View() string { return IndicadorAgente(e.Agente) + e.campo.View() }

// IndicadorAgente compone el indicador del agente activo: `[plan] > ` o
// `[build] > `. Vive aquí porque las dos vistas lo pintan —la principal a
// través de `Entrada.View` y la bienvenida en su línea de entrada— y tiene que
// ser la misma palabra en las dos (SPEC-INTERFAZ §Zonas 2).
func IndicadorAgente(agente string) string {
	return estiloIndicador.Render("[" + agente + "] > ")
}
