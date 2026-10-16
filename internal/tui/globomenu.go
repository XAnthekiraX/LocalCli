package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"localcli/internal/tui/components/chat"
	"localcli/internal/tui/render"
)

// AccionGlobo es una de las opciones del menú de un globo del usuario.
type AccionGlobo int

const (
	// GloboRevertir saca de la conversación lo que va desde ese mensaje del
	// usuario hasta el final.
	GloboRevertir AccionGlobo = iota
	// GloboCopiar lleva el texto del globo al portapapeles.
	GloboCopiar
	// GloboBifurcar abre una sesión nueva con el historial hasta ese mensaje.
	GloboBifurcar
)

// opcionGlobo es una fila del menú.
type opcionGlobo struct {
	accion AccionGlobo
	texto  string
}

var opcionesGlobo = []opcionGlobo{
	{accion: GloboRevertir, texto: "Revertir aquí"},
	{accion: GloboCopiar, texto: "Copiar"},
	{accion: GloboBifurcar, texto: "Bifurcar"},
}

// anchoMenuGlobo es el ancho fijo del menú: cabe la opción más larga con su aire.
const anchoMenuGlobo = 18

// MenuGlobo es el menú emergente de un globo del usuario: revertir, copiar o
// bifurcar. Se abre con un clic simple sobre un globo del usuario, se ancla a su
// línea del hilo —para recolocarse aunque el hilo se desplace— y se recorre con
// ↑/↓ y Enter, o se cierra con Esc. No decide nada: solo dice qué opción se
// eligió y sobre qué mensaje.
type MenuGlobo struct {
	visible bool
	sel     int
	// mensajeID es el mensaje del globo sobre el que se abrió.
	mensajeID string
	// contenido es la línea del hilo del globo, para recolocar el menú.
	contenido int
}

// NuevoMenuGlobo deja un menú cerrado.
func NuevoMenuGlobo() MenuGlobo { return MenuGlobo{} }

// clicGlobo recuerda el globo del usuario pulsado mientras el botón sigue abajo,
// para distinguir un clic simple —que abre el menú— de un arrastre, que
// selecciona texto. Cualquier movimiento cancela el candidato.
type clicGlobo struct {
	activo    bool
	mensajeID string
	contenido int
	inicio    punto
}

// Abrir despliega el menú sobre un mensaje y su línea del hilo.
func (m *MenuGlobo) Abrir(mensajeID string, contenido int) {
	m.visible = true
	m.sel = 0
	m.mensajeID = mensajeID
	m.contenido = contenido
}

// Cerrar retira el menú.
func (m *MenuGlobo) Cerrar() {
	m.visible = false
	m.mensajeID = ""
}

// Visible dice si el menú está desplegado.
func (m MenuGlobo) Visible() bool { return m.visible }

// MensajeID devuelve el mensaje sobre el que se abrió el menú.
func (m MenuGlobo) MensajeID() string { return m.mensajeID }

// Contenido devuelve la línea del hilo del globo del menú.
func (m MenuGlobo) Contenido() int { return m.contenido }

// Subir y Bajar mueven el resaltado sin salirse de la lista.
func (m *MenuGlobo) Subir() {
	if m.sel > 0 {
		m.sel--
	}
}

func (m *MenuGlobo) Bajar() {
	if m.sel < len(opcionesGlobo)-1 {
		m.sel++
	}
}

// Elegida devuelve la opción resaltada.
func (m MenuGlobo) Elegida() (AccionGlobo, bool) {
	return m.EnFila(m.sel)
}

// EnFila devuelve la opción de una fila del menú, para resolver un clic sobre él.
func (m MenuGlobo) EnFila(fila int) (AccionGlobo, bool) {
	if fila < 0 || fila >= len(opcionesGlobo) {
		return 0, false
	}
	return opcionesGlobo[fila].accion, true
}

// prepararClicGlobo decide si el clic en `p` cae sobre el globo de un mensaje del
// usuario con identidad conocida y, si es así, lo recuerda como candidato. Un
// arrastre posterior lo cancela; un clic sin movimiento lo abre.
func (a *App) prepararClicGlobo(p punto) {
	a.clicGlobo = clicGlobo{}
	if a.vista != VistaPrincipal || a.zonaChat.Alto <= 0 || !a.zonaChat.contiene(p) {
		return
	}
	linea, ok := a.chat.ContenidoBajoVentana(p.Fila - a.zonaChat.Fila)
	if !ok {
		return
	}
	seg, ok := a.chat.SegmentoEnLinea(linea)
	if !ok || seg.Hablante != chat.Usuario || seg.ID == "" {
		return
	}
	a.clicGlobo = clicGlobo{activo: true, mensajeID: seg.ID, contenido: linea, inicio: p}
}

// rectGlobo resuelve la geometría del menú del globo: anclado a la línea de su
// globo dentro de la zona del chat y subido por encima si no cabe hacia abajo.
func (a *App) rectGlobo() (fila, col, ancho, alto int) {
	vista := a.globo.View()
	if vista == "" {
		return 0, 0, 0, 0
	}
	ancho = anchoMenuGlobo
	alto = altoDe(vista)
	col = a.zonaChat.Columna + 2
	filaVentana, ok := a.chat.VentanaDeContenido(a.globo.Contenido())
	if !ok {
		filaVentana = 0
	}
	fila = a.zonaChat.Fila + filaVentana
	if fila+alto > a.zonaChat.Fila+a.zonaChat.Alto {
		fila -= alto
	}
	if fila < a.zonaChat.Fila {
		fila = a.zonaChat.Fila
	}
	if col+ancho > a.zonaChat.Columna+a.zonaChat.Ancho {
		col = a.zonaChat.Columna
	}
	return
}

// clicEnMenuGlobo resuelve un clic sobre el menú desplegado: devuelve la opción
// de la fila pulsada, si el clic cayó dentro.
func (a *App) clicEnMenuGlobo(p punto) (AccionGlobo, bool) {
	fila, col, ancho, alto := a.rectGlobo()
	if alto <= 0 || p.Columna < col || p.Columna >= col+ancho || p.Fila < fila || p.Fila >= fila+alto {
		return 0, false
	}
	return a.globo.EnFila(p.Fila - fila)
}

// aplicarMenuGlobo resuelve la opción resaltada del menú y lo cierra.
func (a *App) aplicarMenuGlobo() tea.Cmd {
	accion, ok := a.globo.Elegida()
	if !ok {
		a.globo.Cerrar()
		return nil
	}
	return a.ejecutarMenuGlobo(accion)
}

// ejecutarMenuGlobo lleva a cabo la opción elegida sobre el mensaje del menú.
func (a *App) ejecutarMenuGlobo(accion AccionGlobo) tea.Cmd {
	id := a.globo.MensajeID()
	texto := ""
	if seg, ok := a.chat.SegmentoEnLinea(a.globo.Contenido()); ok {
		texto = seg.Texto
	}
	a.globo.Cerrar()
	if a.puerto == nil || a.sesion == nil || id == "" {
		return nil
	}
	switch accion {
	case GloboRevertir:
		return turnoTocadoCmd(a.puerto, a.sesion.ID,
			"no se pudo revertir", "turno revertido", "no hay nada que revertir",
			func(p Puerto, sesionID string) (int, error) { return p.DeshacerDesde(sesionID, id) })
	case GloboCopiar:
		if texto == "" {
			return nil
		}
		return tea.Batch(copiarCmd(texto), a.avisarCopiado())
	case GloboBifurcar:
		return bifurcarCmd(a.puerto, a.sesion.ID, id)
	}
	return nil
}

// bifurcarCmd abre una sesión nueva con el historial hasta el mensaje indicado y
// trae la sesión creada para que el app la adopte.
func bifurcarCmd(p Puerto, sesionID, mensajeID string) tea.Cmd {
	return func() tea.Msg {
		if p == nil {
			return bifurcadaMsg{}
		}
		sesion, err := p.Bifurcar(sesionID, mensajeID)
		return bifurcadaMsg{Sesion: sesion, Err: err}
	}
}

// teclaMenuGlobo resuelve una pulsación con el menú del globo desplegado.
func (a *App) teclaMenuGlobo(m tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.Type {
	case tea.KeyEsc:
		a.globo.Cerrar()
	case tea.KeyUp:
		a.globo.Subir()
	case tea.KeyDown:
		a.globo.Bajar()
	case tea.KeyEnter:
		return a, a.aplicarMenuGlobo()
	}
	return a, nil
}

// View pinta el menú: cada fila es una superficie del color de la paleta y la
// resaltada va sobre el fondo de selección, como la paleta de comandos.
func (m MenuGlobo) View() string {
	if !m.visible || len(opcionesGlobo) == 0 {
		return ""
	}
	lineas := make([]string, 0, len(opcionesGlobo))
	for i, op := range opcionesGlobo {
		texto := recortar(" "+op.texto, anchoMenuGlobo)
		if i == m.sel {
			lineas = append(lineas,
				render.PintarFondo(render.EstiloPaletaResaltada.Render(texto), anchoMenuGlobo, render.FondoPaletaResaltada))
			continue
		}
		lineas = append(lineas,
			render.PintarFondo(render.EstiloPaleta.Render(texto), anchoMenuGlobo, render.FondoPaleta))
	}
	return strings.Join(lineas, "\n")
}
