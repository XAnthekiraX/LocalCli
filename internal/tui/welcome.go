// welcome.go — T-F003: la pantalla de bienvenida.
//
// Fuente de verdad: frontend/01-domain/DOMAIN.md §3 ("Es la primera vista al
// ejecutar localcli. Solo hay logotipo, nombre con versión y una línea de
// entrada; sin paneles, selector ni aprobaciones", "La única salida desde ella
// es Ctrl+C", "Se pinta sin esperar a Ollama ni a la base"), SPEC-INTERFAZ
// §Pantalla de bienvenida y §Arte canónico del logotipo ("se pinta tal cual",
// "no se reescala ni se centra dinámicamente") e INTERFACES §2 ("Crear o
// retomar sesión: al enviar desde la bienvenida… y luego va el mensaje").
//
// Es un modelo con el estado mínimo —lo escrito— y su propio teclado: en la
// bienvenida no existen los atajos de la interfaz principal, y eso se cumple
// aquí y no con un `if` repartido por app.go.
package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// Bienvenida es el modelo de la primera pantalla: lo escrito, que será la
// primera petición, y el foco de su única línea de entrada.
type Bienvenida struct {
	Texto string
	Foco  bool
}

// NuevaBienvenida deja la pantalla lista con la entrada enfocada: es la única
// línea que existe aquí, así que el foco es el estado por defecto.
func NuevaBienvenida() Bienvenida { return Bienvenida{Foco: true} }

// Escribir añade lo tecleado a la primera petición.
func (b *Bienvenida) Escribir(s string) {
	if b.Foco {
		b.Texto += s
	}
}

// Borrar quita el último carácter escrito.
func (b *Bienvenida) Borrar() {
	if r := []rune(b.Texto); len(r) > 0 {
		b.Texto = string(r[:len(r)-1])
	}
}

// teclaBienvenida resuelve una pulsación de la bienvenida. Solo hay tres cosas
// que hacer aquí: escribir, enviar y salir (SPEC-INTERFAZ: "Desde la bienvenida
// no se lanza nada"). Cualquier otra tecla no hace nada: no hay panel, selector
// ni aprobaciones a los que abrir.
func (a *App) teclaBienvenida(m tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.String() {
	case "ctrl+c", "ctrl+q":
		// DOMAIN §3: la única salida desde la bienvenida. Ctrl+Q se acepta
		// además de Ctrl+C para que el atajo de salida del mapa funcione aquí
		// también.
		return a, tea.Quit
	case "enter":
		return a, a.enviarDesdeBienvenida()
	}
	switch m.Type {
	case tea.KeyRunes:
		a.Bienvenida.Escribir(string(m.Runes))
	case tea.KeySpace:
		a.Bienvenida.Escribir(" ")
	case tea.KeyBackspace:
		a.Bienvenida.Borrar()
	}
	return a, nil
}

// enviarDesdeBienvenida manda lo escrito como primera petición: la sesión
// activa se retoma si existe o se crea si no (INTERFACES §2) y después va el
// mensaje. Son las dos operaciones de este envío, cada una exactamente una vez;
// la vista cambia a la principal sin repetir la petición ni pedir
// confirmación (DOMAIN §3).
func (a *App) enviarDesdeBienvenida() tea.Cmd {
	texto := strings.TrimSpace(a.Bienvenida.Texto)
	if texto == "" {
		return nil
	}
	if a.Panel.SesionID == "" {
		ses, err := a.Puerto.ResolverActiva()
		if err != nil {
			a.Chat.AñadirSistema("no se pudo abrir la sesión: " + err.Error())
			return nil
		}
		a.activar(ses)
	}
	a.Bienvenida.Texto = ""
	a.Chat.AñadirUsuario(texto)
	a.Vista = VistaPrincipal
	return a.enviarCmd(a.Panel.SesionID, texto)
}

// viewBienvenida pinta el logotipo tal cual —del dorado embebido, sin
// reescalar ni centrar— y alrededor, fuera del arte, el nombre con versión y la
// línea de entrada (SPEC-INTERFAZ §Arte canónico del logotipo).
func (a *App) viewBienvenida() string {
	var b strings.Builder
	b.WriteString(strings.TrimRight(LogoCanonico, "\n"))
	b.WriteString("\n\n")
	b.WriteString(estiloMarca.Render(Nombre + " · " + Version))
	b.WriteString("\n\n")
	b.WriteString("En qué te ayudo hoy: " + a.Bienvenida.Texto)
	if a.Bienvenida.Foco {
		b.WriteString("▌")
	}
	b.WriteString("\n")
	return b.String()
}
