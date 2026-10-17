package chat

import (
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"localcli/internal/tui/render"
)

// franja es el ancho de la banda vertical del color del hablante: una columna.
const franja = 1

// Hablante es quién produjo un segmento de texto del hilo.
type Hablante string

const (
	// Usuario es lo que escribió quien usa la aplicación.
	Usuario Hablante = "usuario"
	// Agente es lo que respondió el modelo.
	Agente Hablante = "agente"
)

// Segmento es un bloque de texto del hilo con el razonamiento de su turno. Un
// mismo turno puede dejar varios —el texto anterior a una herramienta y el
// posterior no se funden— y cada uno conserva el razonamiento que lo acompañó;
// el turno en vivo crece dentro del segmento abierto.
type Segmento struct {
	// ID es el identificador del mensaje en la base, cuando el segmento viene del
	// historial guardado. Vacío en un segmento del turno en vivo, que todavía no
	// tiene fila propia. Es lo que permite revertir o bifurcar sobre un globo.
	ID       string
	Hablante Hablante
	Texto    string
	// Razonamiento es el texto crudo del modelo para este segmento. No se vuelca
	// por defecto: se revela con `Ctrl+R`.
	Razonamiento string
	// SinRazonamiento dice que el turno se cerró sin que el modelo entregara
	// razonamiento: revelarlo entonces lo avisa en vez de dejar la pantalla muda.
	SinRazonamiento bool
	// Duracion es lo que tardó en llegar esta respuesta, del envío al cierre.
	// Cero = no se midió, y entonces no se pinta nada ni se deja hueco.
	Duracion time.Duration
}

// TipoLinea clasifica lo que va suelto en el hilo: ni texto del usuario ni del
// agente, sino el trabajo del sistema.
type TipoLinea string

const (
	// LineaHerramienta es la línea de una llamada a herramienta: nace al invocar
	// con su verbo y su tema, y se completa al llegar el resultado.
	LineaHerramienta TipoLinea = "herramienta"
	// LineaSubProceso es la línea `[Sub Proceso] <nombre>` de una etapa de flujo.
	LineaSubProceso TipoLinea = "subproceso"
	// LineaAviso es un aviso o una notificación del sistema.
	LineaAviso TipoLinea = "aviso"
)

// Linea es una línea del sistema: una herramienta, una etapa de un flujo o un
// aviso. Se pinta suelta, sin bloque de fondo ni color de hablante. `Bloque` es
// el diagrama renderizado de una herramienta `diagram`, ya en líneas Unicode de
// ancho calculado: va justo debajo de la línea, como bloque adicional, no
// sustitutivo (INTERFACES §1.1). Vacío para el resto de tipos.
type Linea struct {
	Tipo   TipoLinea
	Texto  string
	Bloque string
}

// Pieza es una pieza del hilo en su sitio cronológico: o un segmento de texto o
// una línea del sistema.
type Pieza interface{ esPieza() }

func (*Segmento) esPieza() {}
func (*Linea) esPieza()    {}

// Model es el historial de la sesión activa más el turno que se está generando.
type Model struct {
	piezas []Pieza

	// El turno en vivo se guarda por fragmentos, tal como llegan del streaming, y
	// solo pasa al historial cuando se cierra. Cada vez que algo se interpone en
	// el hilo —una línea de sistema— su texto se cierra como un segmento, para
	// que quede arriba de esa línea y lo que venga después abra uno nuevo.
	respuesta    []string
	razonamiento []string
	inicio       time.Time
	// ultimoAgente es el índice+1 del último segmento del agente cerrado en el
	// turno: sirve para colgarle la duración del turno cuando este termina justo
	// después de una herramienta y no queda texto al que colgarla.
	ultimoAgente int
	// herramienta es el nombre de la herramienta que corre, para el indicador en
	// vivo («Usando herramienta: X»).
	herramienta string

	// pendientes son los índices de las líneas de herramienta abiertas, en orden
	// de invocación: al llegar el resultado se cierra la primera (FIFO), así la
	// misma línea pasa de «LEER [ruta]» a «✓ LEER [ruta] · 70 líneas · 0.4 s».
	pendientes []int

	// MostrarRazonamiento revela el texto crudo del razonamiento (`Ctrl+R`).
	MostrarRazonamiento bool

	// offset y maxOffset son la ventana del historial: la primera línea visible y
	// el máximo para el ancho de la última pintura. `scrolleado` se enciende al
	// subir: apagado, la vista sigue el final y baja sola con cada respuesta
	// nueva. `altoUltimo` recuerda el alto para avanzar una página entera.
	offset     int
	maxOffset  int
	scrolleado bool
	altoUltimo int

	// hiloLineas son las líneas del hilo de la última pintura: las coordenadas de
	// contenido de la selección del chat. `arrastrando` suspende el seguimiento
	// del final mientras dura el arrastre.
	hiloLineas []string
	// lineasPieza ata cada línea del hilo con la pieza cerrada que la pintó, por
	// índice en `piezas`; -1 en las líneas de separación y en el turno en vivo,
	// que no son globos. Lo usa el menú del globo para saber sobre qué segmento
	// cayó un clic. Tiene la misma longitud que hiloLineas.
	lineasPieza []int
	arrastrando bool

	// botones son los botones de copiar de los bloques de código de la última
	// pintura, en coordenadas del hilo (línea de contenido y columnas de la zona
	// del chat). Se recalculan en cada Hilo, porque el Markdown se interpreta al
	// pintar: un botón vale para la línea que está bajo el ratón ahora.
	botones []BotonCopia
}

// Nuevo deja un hilo vacío.
func Nuevo() Model { return Model{} }

// Piezas devuelve las piezas cerradas del hilo, en su orden.
func (m Model) Piezas() []Pieza { return m.piezas }

// Vacio dice si no hay nada que pintar: ni historial ni turno en vivo.
func (m Model) Vacio() bool {
	return len(m.piezas) == 0 && len(m.respuesta) == 0 && len(m.razonamiento) == 0
}

// AñadirSegmento cuelga un segmento de texto del final del hilo.
func (m *Model) AñadirSegmento(s Segmento) { m.piezas = append(m.piezas, &s) }

// FijarIDUsuarioPendiente cierra el id de la base sobre el primer segmento del
// usuario todavía sin id: la burbuja viva que la vista pintó al enviar nace sin
// id porque el mensaje aún no existía en la base, y el menú del globo lo exige
// para revertir o bifurcar. El emparejamiento es FIFO —los ids llegan en el
// orden en que se persistió y las burbujas se pintaron en el orden de envío—.
// Si no queda ninguna burbuja del usuario sin id —p. ej. el hilo se recargó ya
// desde el historial, que trae los suyos—, no hace nada.
func (m *Model) FijarIDUsuarioPendiente(id string) {
	if id == "" {
		return
	}
	for _, pieza := range m.piezas {
		seg, ok := pieza.(*Segmento)
		if !ok || seg.Hablante != Usuario || seg.ID != "" {
			continue
		}
		seg.ID = id
		return
	}
}

// AñadirLinea cuelga una línea del sistema del final del hilo. Antes cierra el
// texto en curso: la línea se interpone en su sitio cronológico, no al final del
// turno (INTERFACES §1.1).
func (m *Model) AñadirLinea(l Linea) {
	m.cerrarSegmento()
	m.piezas = append(m.piezas, &l)
}

// Vaciar deja el hilo como recién abierto, al cambiar de sesión: solo se pinta la
// sesión activa. La revelación del razonamiento se conserva, como su `Ctrl+R`.
func (m *Model) Vaciar() { *m = Model{MostrarRazonamiento: m.MostrarRazonamiento} }

// ReemplazarHistorial cambia las piezas cerradas del hilo por las de una sesión
// retomada —sus turnos y sus líneas del sistema, ya intercalados—, conservando el
// turno en vivo —su respuesta parcial y su contador— y la revelación del
// razonamiento: cargar el historial no debe cortar lo que se está generando ni el
// `Ctrl+R`.
func (m *Model) ReemplazarHistorial(piezas []Pieza) {
	m.piezas = piezas
}

// cerrarSegmento pasa lo generado hasta ahora al historial como un segmento del
// agente, con su razonamiento. Lo llaman las líneas del sistema y la entrada del
// usuario antes de interponerse: el texto anterior queda arriba y lo que el
// modelo escriba después abre un segmento nuevo (SPEC-INTERFAZ §1).
func (m *Model) cerrarSegmento() {
	if len(m.respuesta) == 0 && len(m.razonamiento) == 0 {
		return
	}
	m.AñadirSegmento(Segmento{
		Hablante:        Agente,
		Texto:           strings.Join(m.respuesta, ""),
		Razonamiento:    strings.Join(m.razonamiento, ""),
		SinRazonamiento: len(m.razonamiento) == 0,
	})
	m.ultimoAgente = len(m.piezas)
	m.respuesta, m.razonamiento = nil, nil
}

// bloque pinta un bloque del chat a partir de las líneas de su contenido ya
// compuestas: ancho completo sobre el fondo del chat (#1E2030, el mismo de la
// caja de entrada), una banda vertical de una columna del color del hablante en
// su borde —azul `#606CD5` el usuario a la derecha, verde `#4CEE75` el agente a
// la izquierda—, el contenido dentro y una celda de aire arriba, abajo y a cada
// lado. Sin icono, sin bordes ni glifos de adorno: la superficie es el fondo, no
// un carácter (SPEC-INTERFAZ §1, DOMAIN §2). Sin contenido no deja bloque.
func bloque(contenido []string, hablante Hablante, ancho int) string {
	if len(contenido) == 0 {
		return ""
	}
	if ancho < franja+3 {
		ancho = franja + 3
	}
	// La banda ocupa una columna del borde y el contenido dispone del resto menos
	// la celda de aire de cada lado.
	interno := ancho - franja - 2
	tira := render.PintarFondo("", franja, colorFranja(hablante))
	vacia := render.PintarFondo("", interno+2, render.FondoCaja)
	fila := func(centro string) string {
		if hablante == Usuario {
			return centro + tira
		}
		return tira + centro
	}
	// Aire arriba y abajo: una fila del fondo del chat antes y después del texto.
	salida := make([]string, 0, len(contenido)+2)
	salida = append(salida, fila(vacia))
	for _, linea := range contenido {
		texto := " " + render.RecortarColumnas(linea, interno) + " "
		salida = append(salida, fila(render.PintarFondo(texto, interno+2, render.FondoCaja)))
	}
	salida = append(salida, fila(vacia))
	return strings.Join(salida, "\n")
}

// colorFranja es el color de la banda del hablante: azul la del usuario y verde
// la del agente.
func colorFranja(hablante Hablante) lipgloss.Color {
	if hablante == Usuario {
		return render.ColorUsuario
	}
	return render.ColorAgente
}

// envolver reparte el texto en filas que caben en `ancho` columnas, respetando
// sus saltos de línea: el texto de un segmento no se recorta. Es una función pura.
func envolver(texto string, ancho int) []string {
	if ancho < 1 {
		ancho = 1
	}
	var lineas []string
	for _, parrafo := range strings.Split(texto, "\n") {
		envueltas, _, _ := render.EnvolverConCursor(parrafo, 0, ancho)
		lineas = append(lineas, envueltas...)
	}
	return lineas
}
