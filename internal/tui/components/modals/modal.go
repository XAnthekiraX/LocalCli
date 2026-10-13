package modals

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"localcli/internal/tui/render"
)

// Pies de los modales: la línea de teclas que se anuncia abajo. El de atajos no
// aplica nada —es de solo lectura— y lo dice.
const (
	PieModal    = "(↑/↓ mover · enter aplicar · esc cerrar)"
	PieSesiones = "(↑/↓ mover · enter abrir · ctrl+d eliminar · esc cerrar)"
	PieAtajos   = "(solo lectura · esc cerrar)"
)

// Lo que se ve cuando la lista que se pidió al puerto aún no ha llegado o no
// llegó: el modal se abre al instante con «cargando…» y, si la lectura falla o
// no hay nada, queda el aviso. Se cierra con `esc` sin bloquear nada.
const (
	AvisoCargando    = "cargando…"
	AvisoSinModelos  = "sin modelos"
	AvisoSinSesiones = "sin sesiones"
)

// Modal es la mecánica común de los tres modales (DOMAIN §1 `modals`): una lista
// con un elemento resaltado, un aviso, un pie de teclas y un descarte. No sabe de
// modelos ni de sesiones: solo de «hay una lista abierta y la estoy
// recorriendo». Que solo haya uno abierto a la vez lo garantiza quien los abre
// —el `app`—, no este tipo.
type Modal struct {
	// Titulo es la cabecera del modal («MODELOS», «SESIONES», «ATAJOS»).
	Titulo string
	// Abierto dice si el modal se pinta. Es la única fuente de «hay modal».
	Abierto bool
	// Lineas es el texto de cada elemento, ya compuesto para pintar.
	Lineas []string
	// Indice es el elemento resaltado. Con la lista vacía vale 0 y no resalta
	// nada; un modal de solo lectura lo deja en -1, porque no hay nada que
	// aplicar.
	Indice int
	// Aviso sustituye a la lista cuando no hay nada que listar («cargando…»,
	// «sin modelos»). No bloquea: se pinta y se cierra con `esc`.
	Aviso string
	// Pie es la línea de teclas. Vacío usa PieModal.
	Pie string
}

// Abrir muestra el modal con la lista dada. Un modal sin lista sigue siendo un
// modal: se abre con su aviso y se cierra con `esc`.
func (m *Modal) Abrir(titulo string, lineas []string, aviso string) {
	m.Titulo = titulo
	m.Lineas = lineas
	m.Aviso = aviso
	m.Indice = 0
	m.Abierto = true
}

// Cerrar lo esconde sin devolver nada: descartar no cambia nada.
func (m *Modal) Cerrar() { m.Abierto = false }

// Mover cambia el resaltado sin salirse de la lista: dar la vuelta por los
// extremos es lo que se espera de un recorrido con `↑`/`↓`. Con la lista vacía
// no hay a dónde moverse y no hace nada.
func (m *Modal) Mover(delta int) {
	if len(m.Lineas) == 0 {
		return
	}
	m.Indice = (m.Indice + delta + len(m.Lineas)) % len(m.Lineas)
}

// Elegida devuelve el índice resaltado y si hay algo que elegir: sin lista —o
// con un modal de solo lectura, que deja el índice fuera— no hay nada.
func (m *Modal) Elegida() (int, bool) {
	if len(m.Lineas) == 0 || m.Indice < 0 || m.Indice >= len(m.Lineas) {
		return 0, false
	}
	return m.Indice, true
}

// PieTexto devuelve el pie del modal: el que puso su tipo o el de fábrica.
func (m *Modal) PieTexto() string {
	if m.Pie != "" {
		return m.Pie
	}
	return PieModal
}

// Render pinta el modal centrado en la ventana: cabecera, lista con el resaltado
// y el pie de teclas. Con aviso, el aviso se pinta antes de la lista —y solo, si
// no hay lista— porque es lo que explica que no haya nada.
//
// Las filas de la lista se rellenan a un ancho común: el bloque se centra entero,
// así que sin rellenar la tabla saldría escalonada.
func (m *Modal) Render(ancho, alto int) string {
	if !m.Abierto {
		return ""
	}
	filas := make([]string, 0, len(m.Lineas))
	for i, linea := range m.Lineas {
		if i == m.Indice {
			filas = append(filas, render.EstiloUsuario.Render("› "+linea))
			continue
		}
		filas = append(filas, "  "+linea)
	}
	anchoFilas := 0
	for _, fila := range filas {
		if w := lipgloss.Width(fila); w > anchoFilas {
			anchoFilas = w
		}
	}

	var b strings.Builder
	b.WriteString(render.EstiloTitulo.Render(m.Titulo) + "\n\n")
	if m.Aviso != "" {
		b.WriteString(render.EstiloAviso.Render(m.Aviso) + "\n")
	}
	for _, fila := range filas {
		b.WriteString(fila + strings.Repeat(" ", anchoFilas-lipgloss.Width(fila)) + "\n")
	}
	b.WriteString(render.EstiloSutil.Render(m.PieTexto()))
	return centrar(ajustar(b.String(), alto), ancho, alto)
}

// ajustar recorta el bloque a las filas de la ventana. Una lista larga —la tabla
// de atajos lo es— no cabe en una terminal baja, y quien normaliza el marco de
// la aplicación se queda con el final del bloque: sin este recorte, un modal alto
// perdería su cabecera. Lo que se cae es el final de la lista, y se dice cuántas
// filas quedaron fuera.
func ajustar(bloque string, alto int) string {
	if alto <= 0 {
		return bloque
	}
	lineas := strings.Split(bloque, "\n")
	if len(lineas) <= alto {
		return bloque
	}
	// Sin sitio para el aviso ni para el pie, manda la ventana: se corta y ya.
	if alto < 3 {
		return strings.Join(lineas[:alto], "\n")
	}
	cabecera := alto - 2
	fuera := len(lineas) - cabecera - 1
	recorte := append([]string{}, lineas[:cabecera]...)
	recorte = append(recorte, render.EstiloSutil.Render(fmt.Sprintf("… %d líneas más", fuera)))
	recorte = append(recorte, lineas[len(lineas)-1])
	return strings.Join(recorte, "\n")
}

// centrar coloca el bloque en el centro de la ventana, en horizontal y en
// vertical. Sin geometría conocida no hay dónde centrar —antes de la primera
// medida de la terminal— y el bloque se devuelve tal cual: nunca se recorta.
func centrar(bloque string, ancho, alto int) string {
	if ancho <= 0 || alto <= 0 {
		return bloque
	}
	return lipgloss.Place(ancho, alto, lipgloss.Center, lipgloss.Center, bloque)
}
