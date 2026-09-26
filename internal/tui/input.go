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
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// Entrada es la línea de texto de la interfaz principal. Una sola fila: lo que
// se escribe va hacia la sesión activa cuando se confirma.
type Entrada struct {
	campo textinput.Model
	Ancho int
}

// NuevaEntrada crea la línea enfocada, con su placeholder y de una sola fila.
func NuevaEntrada() Entrada {
	campo := textinput.New()
	campo.Placeholder = "Escribe tu petición…"
	campo.Focus()
	campo.Width = 80
	return Entrada{campo: campo}
}

// Foco activa la línea y devuelve el comando del cursor, que es como bubbles
// hace parpadear el caret.
func (e *Entrada) Foco() tea.Cmd {
	e.campo.Focus()
	return textinput.Blink
}

// Desenfocar la apaga, por ejemplo mientras otra zona (el selector) se lleva
// el teclado.
func (e *Entrada) Desenfocar() { e.campo.Blur() }

// Texto devuelve lo escrito hasta ahora, tal cual.
func (e *Entrada) Texto() string { return e.campo.Value() }

// Limpiar vacía la línea. Va aparte para que quien envía decida cuándo
// borrarla: un fallo al abrir la sesión no puede perder lo escrito.
func (e *Entrada) Limpiar() { e.campo.Reset() }

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

// View pinta la línea: el placeholder cuando está vacía, el texto y su cursor
// cuando no.
func (e *Entrada) View() string { return e.campo.View() }
