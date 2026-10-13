package render

import "github.com/charmbracelet/lipgloss"

const (
	Nombre  = "LocalCli"
	Version = "v0.1 alpha"
)

var (
	FondoPantalla = lipgloss.Color("#17191F")
	FondoCaja     = lipgloss.Color("#1E2030")
	// FondoPanel es el fondo del sidebar de datos: el mismo tono que la caja de
	// entrada, porque las dos son superficies de datos.
	FondoPanel   = lipgloss.Color("#1E2030")
	ColorUsuario = lipgloss.Color("#606CD5")
	ColorAgente  = lipgloss.Color("#4CEE75")
	ColorSutil   = lipgloss.Color("#828BB8")
	ColorBlanco  = lipgloss.Color("#FFFFFF")
	// FondoPaleta es el fondo de la paleta de flujos y FondoPaletaResaltada el de
	// su fila resaltada; el texto va oscuro para leerse sobre el lavanda.
	FondoPaleta          = lipgloss.Color("#B1BAF9")
	FondoPaletaResaltada = ColorUsuario
	ColorTextoPaleta     = lipgloss.Color("#17191F")
)

var (
	EstiloSutil   = lipgloss.NewStyle().Foreground(ColorSutil)
	EstiloBlanco  = lipgloss.NewStyle().Foreground(ColorBlanco)
	EstiloUsuario = lipgloss.NewStyle().Foreground(ColorUsuario)
	EstiloAgente  = lipgloss.NewStyle().Foreground(ColorAgente).Bold(true)
	// EstiloTitulo encabeza el sidebar con el nombre de la conversación.
	EstiloTitulo = lipgloss.NewStyle().Bold(true)
	// EstiloSeccion titula los bloques del sidebar (CONTEXTO, TODO, LISTA DE
	// TAREAS, ESTADO).
	EstiloSeccion = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14"))
	// EstiloEtiqueta marca la etiqueta de una fila del panel y la firma del
	// harness en el pie.
	EstiloEtiqueta = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14"))
	// EstiloAviso destaca lo que hay que mirar de un vistazo: el aviso de que el
	// contexto se acerca a su límite (SPEC-PANEL-CONTEXTO).
	EstiloAviso = lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Bold(true)
	// EstiloCapaz y EstiloIncapaz marcan las capacidades del modelo en el pie de
	// la caja de entrada: en verde lo que tiene, en rojo lo que no; lo que aún no
	// se sabe va en el tono sutil.
	EstiloCapaz   = lipgloss.NewStyle().Foreground(ColorAgente)
	EstiloIncapaz = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5F5F"))
	// EstiloAdjunto distingue del texto escrito los tokens de archivo, carpeta o
	// pegado que muestra la caja de entrada ([nombre.ext], [CARPETA N elementos],
	// [PEGADO N líneas]).
	EstiloAdjunto = lipgloss.NewStyle().Foreground(lipgloss.Color("#C792EA")).Bold(true)
	// EstiloRazonamiento pinta el texto crudo del razonamiento revelado con
	// `Ctrl+R`: atenuado y en cursiva, para que nunca se confunda con la respuesta
	// (SPEC-INTERFAZ §Razonamiento del modelo).
	EstiloRazonamiento = lipgloss.NewStyle().Foreground(ColorSutil).Italic(true)
	// EstiloPaleta pinta el texto de la paleta de flujos y EstiloPaletaResaltada su
	// fila resaltada.
	EstiloPaleta          = lipgloss.NewStyle().Foreground(ColorTextoPaleta)
	EstiloPaletaResaltada = lipgloss.NewStyle().Foreground(ColorBlanco).Bold(true)
)
