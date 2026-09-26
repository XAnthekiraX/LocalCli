// styles.go — T-F002: los estilos de la pantalla y el render de bloques.
//
// Fuente de verdad: FRONTEND.md §2 (styles.go: "estilos Lip Gloss y render de
// bloques"), frontend/01-domain/DOMAIN.md §2 ("El razonamiento se muestra en
// vivo, arriba de la respuesta, distinguible visualmente y ocultable sin
// detener la generación") y SPEC-INTERFAZ §Razonamiento del modelo ("Se
// distingue visualmente de la respuesta para que nunca se confunden", "El
// razonamiento nunca se mezcla visualmente con la respuesta final").
//
// Todo lo que pinta con color o forma vive aquí: si los estilos se reparten por
// los archivos, un cambio de color se convierte en una búsqueda por todo el
// paquete. Las funciones de render son puras —reciben texto y devuelven
// texto—, sin estado y sin tocar el bucle de Bubble Tea.
package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// estilos de la vista, en un solo sitio para que la pantalla no tenga colores
// sueltos por los archivos.
var (
	estiloUsuario      = lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true)
	estiloAgente       = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	estiloSistema      = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	estiloRazonamiento = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Italic(true)
	estiloTitulo       = lipgloss.NewStyle().Bold(true)
	estiloEtiqueta     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14"))
	estiloAviso        = lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Bold(true)
	estiloMarca        = lipgloss.NewStyle().Bold(true)
)

// recortar deja el texto en una línea por párrafo y sin exceder el ancho, para
// que el razonamiento no rompa la disposición del panel. Es una función pura:
// no toca la pantalla, solo devuelve texto ya medido.
func recortar(texto string, ancho int) string {
	if ancho <= 0 {
		return texto
	}
	var out []string
	for _, linea := range strings.Split(texto, "\n") {
		for len([]rune(linea)) > ancho {
			r := []rune(linea)
			out = append(out, string(r[:ancho]))
			linea = string(r[ancho:])
		}
		out = append(out, linea)
	}
	return strings.Join(out, "\n")
}

// renderReasoning pinta un bloque de razonamiento. Devuelve "" si está oculto o
// si no hay nada que decir: una cabecera vacía solo estorba.
//
// Ocultar es solo pintura: esta función no toca la generación ni el texto
// acumulado, de modo que volver a mostrarlo lo recupera entero
// (frontend/01-domain/DOMAIN.md §2). Distingue visualmente de la respuesta por
// llevar el estilo propio del razonamiento, atenuado y en cursiva.
func renderReasoning(texto string, visible bool, noDisponible bool, ancho int) string {
	if !visible {
		return ""
	}
	if strings.TrimSpace(texto) == "" {
		if noDisponible {
			return estiloRazonamiento.Render("(el modelo no entregó razonamiento)")
		}
		return ""
	}
	return estiloRazonamiento.Render(recortar(texto, ancho))
}

// renderIntercambio pinta un intercambio del chat: el razonamiento, si hay y se
// muestra, arriba de la respuesta, y separado de ella por una línea en blanco
// para que nunca se mezclen visualmente (SPEC-INTERFAZ §Razonamiento del
// modelo). Con respuesta vacía no pinta hueco: la respuesta que todavía no ha
// empezado no deja sitio reservado.
func renderIntercambio(razonamiento, respuesta string, mostrar bool, noDisponible bool, ancho int) string {
	var partes []string
	if r := renderReasoning(razonamiento, mostrar, noDisponible, ancho); r != "" {
		partes = append(partes, r)
	}
	if respuesta != "" {
		partes = append(partes, estiloAgente.Render(recortar(respuesta, ancho)))
	}
	if len(partes) == 0 {
		return ""
	}
	return strings.Join(partes, "\n\n")
}
