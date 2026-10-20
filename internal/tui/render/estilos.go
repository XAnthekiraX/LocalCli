package render

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const (
	Nombre  = "LocalCli"
	Version = "v0.1 alpha"
)

// estilosDefecto es la paleta de fábrica: la única fuente de verdad de los
// valores por defecto. Sus claves son exactamente las que acepta el archivo del
// usuario `~/.config/localcli/styles.json` (clave:valor). Los tokens `fondo_*`
// son fondos y los `color_*` son colores de texto (foreground).
var estilosDefecto = map[string]string{
	"fondo_pantalla":         "#17191F",
	"fondo_caja":             "#1E2030",
	"fondo_panel":            "#1E2030",
	"fondo_paleta":           "#B1BAF9",
	"fondo_paleta_resaltada": "#606CD5",
	"fondo_codigo":           "#0E1017",
	"color_usuario":          "#606CD5",
	"color_agente":           "#4CEE75",
	"color_sutil":            "#828BB8",
	"color_blanco":           "#FFFFFF",
	"color_ruta":             "#FFD480",
	"color_texto_paleta":     "#17191F",
	"color_seccion":          "14",
	"color_aviso":            "11",
	"color_incapaz":          "#FF5F5F",
	"color_codigo_inline":    "#C792EA",
}

var (
	FondoPantalla = lipgloss.Color(estilosDefecto["fondo_pantalla"])
	FondoCaja     = lipgloss.Color(estilosDefecto["fondo_caja"])
	// FondoPanel es el fondo del sidebar de datos: el mismo tono que la caja de
	// entrada, porque las dos son superficies de datos.
	FondoPanel   = lipgloss.Color(estilosDefecto["fondo_panel"])
	ColorUsuario = lipgloss.Color(estilosDefecto["color_usuario"])
	ColorAgente  = lipgloss.Color(estilosDefecto["color_agente"])
	ColorSutil   = lipgloss.Color(estilosDefecto["color_sutil"])
	ColorBlanco  = lipgloss.Color(estilosDefecto["color_blanco"])
	// ColorRuta pinta las rutas de la interfaz —el objetivo de una herramienta, la
	// ruta del proyecto en el sidebar y los tokens de archivos y carpetas—: un
	// ámbar que las distingue del texto y se lee sobre cualquier superficie.
	ColorRuta = lipgloss.Color(estilosDefecto["color_ruta"])
	// FondoPaleta es el fondo de la paleta de flujos y FondoPaletaResaltada el de
	// su fila resaltada; el texto va oscuro para leerse sobre el lavanda.
	FondoPaleta          = lipgloss.Color(estilosDefecto["fondo_paleta"])
	FondoPaletaResaltada = lipgloss.Color(estilosDefecto["fondo_paleta_resaltada"])
	ColorTextoPaleta     = lipgloss.Color(estilosDefecto["color_texto_paleta"])
	// FondoCodigo es el fondo del bloque de código: una superficie propia, más
	// oscura que el globo, para separar el código del texto que lo rodea
	// (SPEC-VISTA-CODIGO). La superficie es el fondo, no un glifo de adorno.
	FondoCodigo = lipgloss.Color(estilosDefecto["fondo_codigo"])
	// ColorSeccion titula los bloques y las etiquetas del sidebar; ColorAviso
	// destaca lo que hay que mirar de un vistazo; ColorIncapaz marca una
	// capacidad ausente; ColorCodigoInline pinta el código dentro del texto.
	ColorSeccion      = lipgloss.Color(estilosDefecto["color_seccion"])
	ColorAviso        = lipgloss.Color(estilosDefecto["color_aviso"])
	ColorIncapaz      = lipgloss.Color(estilosDefecto["color_incapaz"])
	ColorCodigoInline = lipgloss.Color(estilosDefecto["color_codigo_inline"])
)

var (
	EstiloSutil           lipgloss.Style
	EstiloBlanco          lipgloss.Style
	EstiloUsuario         lipgloss.Style
	EstiloAgente          lipgloss.Style
	EstiloTitulo          lipgloss.Style
	EstiloSeccion         lipgloss.Style
	EstiloEtiqueta        lipgloss.Style
	EstiloAviso           lipgloss.Style
	EstiloCapaz           lipgloss.Style
	EstiloIncapaz         lipgloss.Style
	EstiloAdjunto         lipgloss.Style
	EstiloRuta            lipgloss.Style
	EstiloRazonamiento    lipgloss.Style
	EstiloPaleta          lipgloss.Style
	EstiloPaletaResaltada lipgloss.Style
	EstiloCodigo          lipgloss.Style
	EstiloCabeceraCodigo  lipgloss.Style
	EstiloBotonCopiar     lipgloss.Style
	EstiloCodigoInline    lipgloss.Style
	// EstiloDiferenciaMas y EstiloDiferenciaMenos pintan las líneas añadidas y
	// quitadas de un bloque de diferencia: verde para lo que entra, rojo para lo
	// que sale (SPEC-TOOLS §La diferencia a la vista).
	EstiloDiferenciaMas   lipgloss.Style
	EstiloDiferenciaMenos lipgloss.Style
	EstiloNegrita         lipgloss.Style
	EstiloCursiva         lipgloss.Style
	EstiloTituloMD        lipgloss.Style
	EstiloRegla           lipgloss.Style
	EstiloVineta          lipgloss.Style
)

func init() {
	reconstruirEstilos()
}

// reconstruirEstilos compone todos los estilos a partir de los colores y fondos
// vigentes. Se llama al arrancar y cada vez que se aplica una paleta del usuario,
// de modo que los estilos siempre reflejen los tokens actuales.
func reconstruirEstilos() {
	EstiloSutil = lipgloss.NewStyle().Foreground(ColorSutil)
	EstiloBlanco = lipgloss.NewStyle().Foreground(ColorBlanco)
	EstiloUsuario = lipgloss.NewStyle().Foreground(ColorUsuario)
	EstiloAgente = lipgloss.NewStyle().Foreground(ColorAgente).Bold(true)
	// EstiloTitulo encabeza el sidebar con el nombre de la conversación.
	EstiloTitulo = lipgloss.NewStyle().Bold(true)
	// EstiloSeccion titula los bloques del sidebar (CONTEXTO, TODO, LISTA DE
	// TAREAS, ESTADO).
	EstiloSeccion = lipgloss.NewStyle().Bold(true).Foreground(ColorSeccion)
	// EstiloEtiqueta marca la etiqueta de una fila del panel y la firma del
	// harness en el pie.
	EstiloEtiqueta = lipgloss.NewStyle().Bold(true).Foreground(ColorSeccion)
	// EstiloAviso destaca lo que hay que mirar de un vistazo: el aviso de que el
	// contexto se acerca a su límite (SPEC-PANEL-CONTEXTO).
	EstiloAviso = lipgloss.NewStyle().Foreground(ColorAviso).Bold(true)
	// EstiloCapaz y EstiloIncapaz marcan las capacidades del modelo en el pie de
	// la caja de entrada: en verde lo que tiene, en rojo lo que no; lo que aún no
	// se sabe va en el tono sutil.
	EstiloCapaz = lipgloss.NewStyle().Foreground(ColorAgente)
	EstiloIncapaz = lipgloss.NewStyle().Foreground(ColorIncapaz)
	// EstiloAdjunto distingue del texto escrito los tokens de archivo, carpeta o
	// pegado que muestra la caja de entrada ([nombre.ext], [CARPETA N elementos],
	// [PEGADO N líneas]).
	EstiloAdjunto = lipgloss.NewStyle().Foreground(ColorRuta).Bold(true)
	// EstiloRuta pinta una ruta —el objetivo de una herramienta del chat y la ruta
	// del proyecto en el pie del sidebar— en el color de ruta.
	EstiloRuta = lipgloss.NewStyle().Foreground(ColorRuta)
	// EstiloRazonamiento pinta el texto crudo del razonamiento revelado con
	// `Ctrl+R`: atenuado y en cursiva, para que nunca se confunda con la respuesta
	// (SPEC-INTERFAZ §Razonamiento del modelo).
	EstiloRazonamiento = lipgloss.NewStyle().Foreground(ColorSutil).Italic(true)
	// EstiloPaleta pinta el texto de la paleta de flujos y EstiloPaletaResaltada su
	// fila resaltada.
	EstiloPaleta = lipgloss.NewStyle().Foreground(ColorTextoPaleta)
	EstiloPaletaResaltada = lipgloss.NewStyle().Foreground(ColorBlanco).Bold(true)
	// Los estilos del Markdown de la respuesta (SPEC-VISTA-CODIGO). El bloque de
	// código se pinta sobre su propia superficie: código, cabecera con la etiqueta
	// del lenguaje y el botón de copiar. El código inline, la negrita, la cursiva,
	// los títulos, las viñetas y las reglas completan el subconjunto.
	EstiloCodigo = lipgloss.NewStyle().Foreground(ColorBlanco).Background(FondoCodigo)
	EstiloCabeceraCodigo = lipgloss.NewStyle().Foreground(ColorSutil).Background(FondoCodigo)
	EstiloBotonCopiar = lipgloss.NewStyle().Foreground(ColorUsuario).Bold(true).Background(FondoCodigo)
	EstiloCodigoInline = lipgloss.NewStyle().Foreground(ColorCodigoInline).Background(FondoCodigo)
	// La diferencia de un cambio se lee de un vistazo: lo añadido en verde y lo
	// quitado en rojo, sin negrita que compita con el texto del hilo.
	EstiloDiferenciaMas = lipgloss.NewStyle().Foreground(ColorAgente)
	EstiloDiferenciaMenos = lipgloss.NewStyle().Foreground(ColorIncapaz)
	EstiloNegrita = lipgloss.NewStyle().Foreground(ColorBlanco).Bold(true)
	EstiloCursiva = lipgloss.NewStyle().Foreground(ColorBlanco).Italic(true)
	EstiloTituloMD = lipgloss.NewStyle().Foreground(ColorBlanco).Bold(true)
	EstiloRegla = lipgloss.NewStyle().Foreground(ColorSutil)
	EstiloVineta = lipgloss.NewStyle().Foreground(ColorSutil)
}

// EstilosPorDefecto devuelve una copia de la paleta de fábrica, para sembrar el
// archivo del usuario o para volver a él cuando el suyo es inválido.
func EstilosPorDefecto() map[string]string {
	copia := make(map[string]string, len(estilosDefecto))
	for clave, valor := range estilosDefecto {
		copia[clave] = valor
	}
	return copia
}

// AplicarTokens aplica una paleta del usuario sobre la de fábrica: lo que la
// paleta no nombra conserva su valor por defecto. Una clave desconocida o un
// color inválido se rechazan enteros —no se adivina—, igual que en keys.json.
func AplicarTokens(t map[string]string) error {
	for clave, valor := range t {
		if _, ok := estilosDefecto[clave]; !ok {
			return fmt.Errorf("render: styles.json nombra un token desconocido: %s", clave)
		}
		if !colorValido(valor) {
			return fmt.Errorf("render: styles.json da un color inválido a %s: %q", clave, valor)
		}
	}

	combinada := EstilosPorDefecto()
	for clave, valor := range t {
		combinada[clave] = valor
	}

	FondoPantalla = lipgloss.Color(combinada["fondo_pantalla"])
	FondoCaja = lipgloss.Color(combinada["fondo_caja"])
	FondoPanel = lipgloss.Color(combinada["fondo_panel"])
	FondoPaleta = lipgloss.Color(combinada["fondo_paleta"])
	FondoPaletaResaltada = lipgloss.Color(combinada["fondo_paleta_resaltada"])
	FondoCodigo = lipgloss.Color(combinada["fondo_codigo"])
	ColorUsuario = lipgloss.Color(combinada["color_usuario"])
	ColorAgente = lipgloss.Color(combinada["color_agente"])
	ColorSutil = lipgloss.Color(combinada["color_sutil"])
	ColorBlanco = lipgloss.Color(combinada["color_blanco"])
	ColorRuta = lipgloss.Color(combinada["color_ruta"])
	ColorTextoPaleta = lipgloss.Color(combinada["color_texto_paleta"])
	ColorSeccion = lipgloss.Color(combinada["color_seccion"])
	ColorAviso = lipgloss.Color(combinada["color_aviso"])
	ColorIncapaz = lipgloss.Color(combinada["color_incapaz"])
	ColorCodigoInline = lipgloss.Color(combinada["color_codigo_inline"])

	reconstruirEstilos()
	return nil
}

// colorValido acepta las formas que entiende Lip Gloss: un hex `#RGB` o
// `#RRGGBB`, o un índice ANSI de 0 a 255.
func colorValido(valor string) bool {
	if valor == "" {
		return false
	}
	if strings.HasPrefix(valor, "#") {
		hex := valor[1:]
		if len(hex) != 3 && len(hex) != 6 {
			return false
		}
		for _, r := range hex {
			if !strings.ContainsRune("0123456789abcdefABCDEF", r) {
				return false
			}
		}
		return true
	}
	if n, err := strconv.Atoi(valor); err == nil {
		return n >= 0 && n <= 255
	}
	return false
}
