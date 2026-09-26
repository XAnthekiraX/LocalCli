package tui

// Tests de T-F002: los estilos y el render de bloques. Una prueba, una regla,
// igual que en el resto del paquete; los nombres enuncian la regla que
// comprueban (frontend/05-quality/TESTING.md §3).
//
//	T-F002-02 → TestClipLineRecortaAlAncho
//	T-F002-03 → TestRenderRazonamientoDistinguibleYOcultable
//	T-F002-04 → TestRenderIntercambioRazonamientoAntesQueRespuesta
//
// Son funciones puras: nada de Bubble Tea, ni base, ni red. Lo que compara el
// ojo de la persona es el texto sin códigos de color, pero que la vista sea
// distinguible exige que el render LLEVE estilo: se comprueba contra el texto
// con códigos.

import (
	"os"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// TestMain fija el perfil de color para todo el paquete: sin terminal, lipgloss
// no emite códigos y las pruebas que comprueban que un bloque SE DISTINGUE por
// su estilo no tendrían nada que ver. Con el perfil forzado, el render es
// determinista en cualquier entorno; las pruebas que miran texto plano lo
// obtienen con sinEstilo, igual que hace el ojo.
func TestMain(m *testing.M) {
	lipgloss.SetColorProfile(termenv.ANSI)
	os.Exit(m.Run())
}

// --- T-F002-02: recortar ----------------------------------------------------

func TestClipLineRecortaAlAncho(t *testing.T) {
	linea := "el razonamiento del modelo puede ser larguísimo y no cabe"
	got := recortar(linea, 20)
	// Partir no quita caracteres: sin los saltos vuelve el texto entero.
	if strings.ReplaceAll(got, "\n", "") != linea {
		t.Fatalf("recortar parte la línea pero no la acorta: %q", got)
	}
	for _, l := range strings.Split(got, "\n") {
		if n := len([]rune(l)); n > 20 {
			t.Errorf("ninguna línea puede pasar del ancho, hay una de %d: %q", n, l)
		}
	}

	// Cada línea se mide por separado: no se rompe el texto entero seguido.
	varias := "una línea corta\nesta otra sí que se pasa del ancho permitido"
	got = recortar(varias, 10)
	for _, l := range strings.Split(got, "\n") {
		if n := len([]rune(l)); n > 10 {
			t.Errorf("ninguna línea puede pasar del ancho, hay una de %d: %q", n, l)
		}
	}

	// Recortar es por runas, no por bytes: partir a la mitad de "á" dejaría
	// un carácter roto en pantalla. Si al juntar los trozos vuelve el texto
	// original, ningún carácter se ha dañado.
	got = recortar("áéíóúñç", 3)
	if strings.ReplaceAll(got, "\n", "") != "áéíóúñç" {
		t.Errorf("el recorte cuenta caracteres, no bytes: %q", got)
	}
	for _, l := range strings.Split(got, "\n") {
		if n := len([]rune(l)); n > 3 {
			t.Errorf("ninguna línea puede pasar del ancho, hay una de %d: %q", n, l)
		}
	}

	// Sin ancho conocido no se toca nada: mejor texto largo que texto roto.
	if got := recortar("lo que sea", 0); got != "lo que sea" {
		t.Errorf("ancho 0 significa no recortar: %q", got)
	}
}

// --- T-F002-03: el bloque de razonamiento -----------------------------------

func TestRenderRazonamientoDistinguibleYOcultable(t *testing.T) {
	texto := "estoy pensando la respuesta"

	// Visible: el texto se ve igual para el ojo...
	got := renderReasoning(texto, true, false, 100)
	if !strings.Contains(sinEstilo(got), texto) {
		t.Fatalf("el razonamiento visible debe contener el texto: %q", got)
	}
	// ...pero con el estilo del razonamiento, no pelado como la respuesta.
	if got == texto {
		t.Error("el razonamiento debe llevar su estilo para distinguirse de la respuesta")
	}
	// El modelo parte el razonamiento en varias líneas: se mantiene así, sin
	// unirlas ni editarlas (SPEC-PANEL-CONTEXTO: tal como lo devuelve).
	// lipgloss estiliza línea a línea, así que cada una lleva su estilo: la
	// distinción con la respuesta vale para el bloque entero.
	varias := "por un lado\npor otro"
	got = renderReasoning(varias, true, false, 100)
	plano := sinEstilo(got)
	if !strings.Contains(plano, "por un lado") || !strings.Contains(plano, "por otro") {
		t.Errorf("el razonamiento se muestra tal como llegó: %q", got)
	}
	if strings.Count(got, "\x1b[3;90m") != 2 {
		t.Errorf("cada línea del bloque lleva el estilo del razonamiento: %q", got)
	}

	// Ocultable: oculto no se pinta, con texto o con aviso de no disponible.
	if got := renderReasoning(texto, false, false, 100); got != "" {
		t.Errorf("oculto no debe pintarse: %q", got)
	}
	if got := renderReasoning("", false, true, 100); got != "" {
		t.Errorf("oculto no pinta ni el aviso de no disponible: %q", got)
	}
	if got := renderReasoning(texto, false, true, 100); got != "" {
		t.Errorf("oculto no pinta nada aunque haya texto: %q", got)
	}

	// Sin razonamiento y visible, la spec manda decirlo en lugar de callarse.
	got = renderReasoning("", true, true, 100)
	if !strings.Contains(sinEstilo(got), "(el modelo no entregó razonamiento)") {
		t.Errorf("sin razonamiento del modelo se lo indica: %q", got)
	}
	// Y sin nada que decir, no se pinta una cabecera vacía.
	if got := renderReasoning("", true, false, 100); got != "" {
		t.Errorf("un bloque vacío no pinta nada: %q", got)
	}
}

// --- T-F002-04: el intercambio ----------------------------------------------

func TestRenderIntercambioRazonamientoAntesQueRespuesta(t *testing.T) {
	razon := "estoy pensando"
	respuesta := "la respuesta es 4"

	got := renderIntercambio(razon, respuesta, true, false, 100)
	plano := sinEstilo(got)
	if !strings.Contains(plano, razon) || !strings.Contains(plano, respuesta) {
		t.Fatalf("deben verse el razonamiento y la respuesta: %q", plano)
	}
	if strings.Index(plano, razon) >= strings.Index(plano, respuesta) {
		t.Error("el razonamiento va arriba de la respuesta")
	}
	if !strings.Contains(plano, razon+"\n\n"+respuesta) {
		t.Error("razonamiento y respuesta van separados por una línea en blanco: no se mezclan visualmente")
	}

	// Turno sin razonamiento: solo la respuesta, con su estilo.
	got = renderIntercambio("", respuesta, true, false, 100)
	if plano := sinEstilo(got); plano != respuesta {
		t.Errorf("sin razonamiento solo queda la respuesta: %q", plano)
	}
	if got == respuesta {
		t.Error("la respuesta lleva el estilo del agente, no sale pelada")
	}

	// El turno empieza: ni razonamiento ni respuesta, nada que pintar.
	if got := renderIntercambio("", "", true, false, 100); got != "" {
		t.Errorf("un intercambio vacío no pinta nada: %q", got)
	}

	// Ocultar el razonamiento no borra la respuesta: solo deja de verse el
	// bloque (DOMAIN §2: ocultable sin detener la generación).
	oculto := renderIntercambio(razon, respuesta, false, false, 100)
	if plano := sinEstilo(oculto); plano != respuesta {
		t.Errorf("ocultar el razonamiento deja la respuesta intacta: %q", plano)
	}
	// Y la función es pura: repetir la llamada da el mismo texto.
	primera := renderIntercambio(razon, respuesta, true, false, 100)
	segunda := renderIntercambio(razon, respuesta, true, false, 100)
	if primera != segunda {
		t.Error("renderIntercambio no puede depender de estado: la misma entrada pinta lo mismo")
	}
}

// Los estilos viven en styles.go y ninguno puede quedarse sin definir por
// accidente: un estilo vacío pintaría sin formato y la vista perdería la
// distinción que la spec pide.
func TestLosEstilosDelBloqueEstanDefinidos(t *testing.T) {
	for nombre, estilo := range map[string]lipgloss.Style{
		"razonamiento": estiloRazonamiento,
		"respuesta":    estiloAgente,
	} {
		if estilo.Render("x") == "" {
			t.Errorf("el estilo %s debe estar definido", nombre)
		}
	}
}
