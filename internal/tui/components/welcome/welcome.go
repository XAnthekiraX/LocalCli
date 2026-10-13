package welcome

import (
	"strings"

	"localcli/internal/tui/components/adjuntos"
)

// Model guarda lo escrito y dónde está el cursor.
type Model struct {
	Texto string
	Pos   int
	// arte es el logotipo completo; compacto, su versión reducida, con la que la
	// bienvenida cabe en una terminal estrecha (SPEC-INTERFAZ §Arte canónico).
	arte     string
	compacto string

	adjuntos adjuntos.Adjuntos
}

// Nuevo deja la bienvenida lista con el arte completo y su versión compacta: el
// estrecho se usa cuando la terminal no da para el completo.
func Nuevo(arte, compacto string) Model { return Model{arte: arte, compacto: compacto} }

// Peticion devuelve lo escrito con los tokens expandidos a su valor real, sin
// espacios sobrantes: es lo que se envía como primera petición.
func (m Model) Peticion() string { return strings.TrimSpace(m.adjuntos.Expandir(m.Texto)) }

// InsertarPegado anota el texto pegado —rutas a token, texto a `[PEGADO N líneas]`
// — y lo escribe en la línea.
func (m *Model) InsertarPegado(texto string) { m.Escribir(m.adjuntos.Anotar(texto)) }

// Escribir inserta texto en la posición del cursor y lo avanza.
func (m *Model) Escribir(texto string) {
	runas := []rune(m.Texto)
	m.acotar()
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

func (m *Model) Izquierda() {
	m.Pos--
	m.acotar()
}

func (m *Model) Derecha() {
	m.Pos++
	m.acotar()
}

func (m *Model) Inicio() { m.Pos = 0 }

func (m *Model) Fin() { m.Pos = len([]rune(m.Texto)) }

// Limpiar vacía la entrada, olvida los adjuntos y devuelve el cursor al principio.
func (m *Model) Limpiar() {
	m.Texto = ""
	m.Pos = 0
	m.adjuntos.Olvidar()
}

func (m *Model) acotar() {
	switch {
	case m.Pos < 0:
		m.Pos = 0
	case m.Pos > len([]rune(m.Texto)):
		m.Pos = len([]rune(m.Texto))
	}
}
