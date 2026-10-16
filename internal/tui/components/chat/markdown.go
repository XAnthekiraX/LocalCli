package chat

import (
	"strings"
	"unicode"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"

	"localcli/internal/tui/render"
)

// textoBotonCopiar es el rótulo del botón de la cabecera de un bloque de código.
// Su ancho en columnas es el de su texto: solo ASCII, para que coincidan.
const textoBotonCopiar = "copiar"

// etiquetaCodigo es la etiqueta genérica de un bloque sin lenguaje declarado.
const etiquetaCodigo = "código"

// BotonCopia es el botón de copiar de la cabecera de un bloque de código. Linea y
// Col van en coordenadas del contenido del hilo: la línea ya con el prefijo del
// bloque, y la columna medida desde el borde de la zona del chat. Codigo es el
// cuerpo exacto del bloque, sin fences ni cabecera (SPEC-VISTA-CODIGO §Reglas).
type BotonCopia struct {
	Linea  int
	Col    int
	Ancho  int
	Codigo string
}

// renderRespuesta interpreta el subconjunto de Markdown de una respuesta y
// devuelve sus líneas ya pintadas —cada una de a lo sumo `ancho` columnas— y los
// botones de copia de sus bloques de código, con la línea local al contenido
// (0-based). Es una función pura: no toca el Model ni el texto guardado, que se
// interpreta solo al pintar (SPEC-VISTA-CODIGO §Reglas de negocio).
func renderRespuesta(texto string, ancho int) ([]string, []BotonCopia) {
	if ancho < 1 {
		ancho = 1
	}
	var lineas []string
	var botones []BotonCopia
	crudas := strings.Split(texto, "\n")
	for i := 0; i < len(crudas); i++ {
		if lenguaje, ok := abrirFence(crudas[i]); ok {
			// Un bloque de código: la cabecera y su cuerpo, hasta el cierre o el
			// final (un fence sin cerrar es un bloque en curso, SPEC-VISTA-CODIGO
			// §Flujos alternativos).
			i++
			var cuerpo []string
			for ; i < len(crudas); i++ {
				if esFence(crudas[i]) {
					break
				}
				cuerpo = append(cuerpo, crudas[i])
			}
			codigo := strings.Join(cuerpo, "\n")
			cabecera, colBoton := pintarCabeceraCodigo(lenguaje, ancho)
			lineas = append(lineas, cabecera)
			botones = append(botones, BotonCopia{
				Linea:  len(lineas) - 1,
				Col:    colBoton,
				Ancho:  min(runewidth.StringWidth(textoBotonCopiar), ancho),
				Codigo: codigo,
			})
			lineas = append(lineas, pintarCuerpoCodigo(cuerpo, ancho)...)
			continue
		}
		lineas = append(lineas, pintarLineaMD(crudas[i], ancho)...)
	}
	return lineas, botones
}

// abrirFence reconoce la línea que abre un bloque de código y devuelve el
// lenguaje declarado (vacío si no hay) y si la línea abre un bloque. Un fence es
// una línea cuyo texto recortado empieza por tres backticks; lo que sigue al
// primer espacio es el lenguaje.
func abrirFence(linea string) (string, bool) {
	recortada := strings.TrimSpace(linea)
	if !strings.HasPrefix(recortada, "```") {
		return "", false
	}
	info := strings.TrimSpace(strings.TrimPrefix(recortada, "```"))
	if info == "" {
		return "", true
	}
	return strings.Fields(info)[0], true
}

// esFence dice si la línea cierra un bloque de código.
func esFence(linea string) bool {
	return strings.HasPrefix(strings.TrimSpace(linea), "```")
}

// pintarCabeceraCodigo compone la cabecera de un bloque: la etiqueta del lenguaje
// a la izquierda y el botón de copiar a la derecha, sobre una misma superficie de
// `ancho` columnas. Devuelve la línea pintada y la columna donde empieza el botón
// (0-based dentro de la línea de contenido). Es una función pura.
func pintarCabeceraCodigo(lenguaje string, ancho int) (string, int) {
	anchoBoton := min(runewidth.StringWidth(textoBotonCopiar), ancho)
	if lenguaje == "" {
		lenguaje = etiquetaCodigo
	}
	// La etiqueta cede ante el botón: una etiqueta larga se recorta para que el
	// botón quede entero y en el borde derecho.
	disponible := ancho - anchoBoton
	if disponible < 0 {
		disponible = 0
	}
	etiqueta := render.RecortarColumnas(lenguaje, disponible)
	relleno := ancho - render.Ancho(etiqueta) - anchoBoton
	if relleno < 0 {
		relleno = 0
	}
	boton := render.EstiloBotonCopiar.Render(textoBotonCopiar[:anchoBoton])
	cabecera := render.EstiloCabeceraCodigo.Render(etiqueta) +
		render.EstiloCodigo.Render(strings.Repeat(" ", relleno)) +
		boton
	return cabecera, ancho - anchoBoton
}

// pintarCuerpoCodigo pinta el cuerpo de un bloque: cada línea literal, envuelta
// si excede el ancho y rellenada hasta él para que la superficie quede recta. El
// Markdown del cuerpo no se interpreta. Un cuerpo vacío deja una fila de
// superficie para que el bloque se vea. Es una función pura.
func pintarCuerpoCodigo(cuerpo []string, ancho int) []string {
	if len(cuerpo) == 0 {
		return []string{pintarFilaCodigo("", ancho)}
	}
	var lineas []string
	for _, cruda := range cuerpo {
		for _, envuelta := range envolverColumnas(expandirTabs(cruda), ancho) {
			lineas = append(lineas, pintarFilaCodigo(envuelta, ancho))
		}
	}
	return lineas
}

// pintarFilaCodigo pinta una línea de código sobre la superficie del bloque,
// rellenada con su fondo hasta `ancho` columnas. Es una función pura.
func pintarFilaCodigo(texto string, ancho int) string {
	relleno := ancho - render.Ancho(texto)
	if relleno < 0 {
		relleno = 0
	}
	return render.EstiloCodigo.Render(texto + strings.Repeat(" ", relleno))
}

// pintarLineaMD pinta una línea del Markdown de nivel de bloque: título, regla,
// cita, viñeta o párrafo con su Markdown inline. Devuelve una o más filas, ya
// envueltas al ancho. Es una función pura.
func pintarLineaMD(linea string, ancho int) []string {
	if esRegla(linea) {
		return []string{render.EstiloRegla.Render(strings.Repeat("─", ancho))}
	}
	if texto, ok := titulo(linea); ok {
		return pintarEnvuelto(texto, ancho, render.EstiloTituloMD)
	}
	if texto, ok := cita(linea); ok {
		return pintarConPrefijo(texto, ancho, "│ ")
	}
	if texto, ok := vineta(linea); ok {
		return pintarConPrefijo(texto, ancho, "• ")
	}
	if numero, texto, ok := listaOrdenada(linea); ok {
		return pintarConPrefijo(texto, ancho, numero+". ")
	}
	return pintarEnvueltoInline(linea, ancho)
}

// esRegla reconoce una regla horizontal: una línea que, recortada, son tres o más
// guiones. Es una función pura.
func esRegla(linea string) bool {
	recortada := strings.TrimSpace(linea)
	if len(recortada) < 3 {
		return false
	}
	return strings.Trim(recortada, "-") == ""
}

// titulo reconoce un título de nivel 1 a 3 (`#`, `##`, `###` seguidos de espacio)
// y devuelve su texto sin las almohadillas. Es una función pura.
func titulo(linea string) (string, bool) {
	nivel := 0
	for nivel < len(linea) && nivel < 3 && linea[nivel] == '#' {
		nivel++
	}
	if nivel == 0 || nivel >= len(linea) || linea[nivel] != ' ' {
		return "", false
	}
	return strings.TrimSpace(linea[nivel+1:]), true
}

// cita reconoce una cita (`> ` al inicio) y devuelve su texto. Es una función pura.
func cita(linea string) (string, bool) {
	recortada := strings.TrimLeft(linea, " ")
	if !strings.HasPrefix(recortada, ">") {
		return "", false
	}
	return strings.TrimPrefix(strings.TrimPrefix(recortada, ">"), " "), true
}

// vineta reconoce un elemento de lista sin orden (`- ` o `* `) y devuelve su
// texto. Es una función pura.
func vineta(linea string) (string, bool) {
	recortada := strings.TrimLeft(linea, " ")
	if strings.HasPrefix(recortada, "- ") || strings.HasPrefix(recortada, "* ") {
		return recortada[2:], true
	}
	return "", false
}

// listaOrdenada reconoce un elemento de lista ordenada (`1. `) y devuelve su
// número —que se conserva— y su texto. Es una función pura.
func listaOrdenada(linea string) (string, string, bool) {
	recortada := strings.TrimLeft(linea, " ")
	i := 0
	for i < len(recortada) && recortada[i] >= '0' && recortada[i] <= '9' {
		i++
	}
	if i == 0 || i+1 >= len(recortada) || recortada[i] != '.' || recortada[i+1] != ' ' {
		return "", "", false
	}
	return recortada[:i], recortada[i+2:], true
}

// pintarConPrefijo pinta una línea con un prefijo (viñeta, número o cita): el
// texto se envuelve al ancho que deja el prefijo y las filas de continuación se
// alinean bajo el texto. El prefijo va atenuado y el contenido interpreta su
// Markdown inline. Es una función pura.
func pintarConPrefijo(texto string, ancho int, prefijo string) []string {
	anchoPrefijo := render.Ancho(prefijo)
	anchoTexto := ancho - anchoPrefijo
	if anchoTexto < 1 {
		anchoTexto = 1
	}
	filas := envolverColumnas(texto, anchoTexto)
	salida := make([]string, 0, len(filas))
	for i, fila := range filas {
		marca := strings.Repeat(" ", anchoPrefijo)
		if i == 0 {
			marca = render.EstiloVineta.Render(prefijo)
		}
		salida = append(salida, marca+pintarInline(fila))
	}
	return salida
}

// pintarEnvuelto envuelve el texto al ancho dado y pinta cada fila con un estilo
// de bloque (un título, por ejemplo), sin interpretar su Markdown inline. Es una
// función pura.
func pintarEnvuelto(texto string, ancho int, estilo lipgloss.Style) []string {
	filas := envolverColumnas(texto, ancho)
	salida := make([]string, 0, len(filas))
	for _, fila := range filas {
		salida = append(salida, estilo.Render(fila))
	}
	return salida
}

// pintarEnvueltoInline envuelve el texto al ancho dado e interpreta el Markdown
// inline de cada fila. Es una función pura.
func pintarEnvueltoInline(texto string, ancho int) []string {
	filas := envolverColumnas(texto, ancho)
	salida := make([]string, 0, len(filas))
	for _, fila := range filas {
		salida = append(salida, pintarInline(fila))
	}
	return salida
}

// pintarInline interpreta el Markdown de una línea: negrita (`**…**`), código
// inline (“ `…` “) y cursiva (`*…*` o `_…_`). Un marcador sin pareja queda
// literal: nunca se pierde texto (SPEC-VISTA-CODIGO §Reglas de negocio). La
// cursiva con guion bajo no parte una palabra, para no tocar identificadores como
// `snake_case`. Es una función pura.
func pintarInline(texto string) string {
	runas := []rune(texto)
	var b strings.Builder
	var plano strings.Builder
	vaciar := func() {
		if plano.Len() > 0 {
			b.WriteString(render.EstiloBlanco.Render(plano.String()))
			plano.Reset()
		}
	}
	for i := 0; i < len(runas); {
		if i+1 < len(runas) && runas[i] == '*' && runas[i+1] == '*' {
			if j := indexarDoble(runas, i+2); j >= 0 {
				vaciar()
				b.WriteString(render.EstiloNegrita.Render(string(runas[i+2 : j])))
				i = j + 2
				continue
			}
		}
		if runas[i] == '`' {
			if j := indexarRuna(runas, '`', i+1); j >= 0 && j > i {
				vaciar()
				b.WriteString(render.EstiloCodigoInline.Render(string(runas[i+1 : j])))
				i = j + 1
				continue
			}
		}
		if runas[i] == '*' || runas[i] == '_' {
			if j, ok := cierreCursiva(runas, i); ok {
				vaciar()
				b.WriteString(render.EstiloCursiva.Render(string(runas[i+1 : j])))
				i = j + 1
				continue
			}
		}
		plano.WriteRune(runas[i])
		i++
	}
	vaciar()
	return b.String()
}

// indexarDoble busca el cierre de una negrita (`**`) desde `desde` y devuelve su
// índice, o -1 si no lo hay. Es una función pura.
func indexarDoble(runas []rune, desde int) int {
	for i := desde; i+1 < len(runas); i++ {
		if runas[i] == '*' && runas[i+1] == '*' {
			return i
		}
	}
	return -1
}

// indexarRuna busca una runa en una línea desde `desde` y devuelve su índice, o
// -1 si no está. Es una función pura.
func indexarRuna(runas []rune, objetivo rune, desde int) int {
	for i := desde; i < len(runas); i++ {
		if runas[i] == objetivo {
			return i
		}
	}
	return -1
}

// cierreCursiva busca el cierre de una cursiva abierta en `i` y devuelve su
// índice y si el marcador está bien formado. Para el guion bajo exige que los
// límites no estén pegados a una letra o un dígito, de modo que `snake_case` no
// se convierta en cursiva. Es una función pura.
func cierreCursiva(runas []rune, i int) (int, bool) {
	marca := runas[i]
	if marca == '_' {
		if i > 0 && esAlfanumerico(runas[i-1]) {
			return 0, false
		}
	}
	j := indexarRuna(runas, marca, i+1)
	if j < 0 || j == i+1 {
		return 0, false
	}
	if marca == '_' && j+1 < len(runas) && esAlfanumerico(runas[j+1]) {
		return 0, false
	}
	return j, true
}

// esAlfanumerico dice si una runa es letra o dígito. Es una función pura.
func esAlfanumerico(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}

// envolverColumnas parte un texto en filas de a lo sumo `ancho` columnas de
// pantalla, sin partir por palabras (el código y los identificadores no se
// recortan). Respeta los anchos de las runas anchas. Es una función pura.
func envolverColumnas(texto string, ancho int) []string {
	if ancho < 1 {
		ancho = 1
	}
	var lineas []string
	var actual strings.Builder
	columnas := 0
	for _, r := range texto {
		peso := runewidth.RuneWidth(r)
		if columnas+peso > ancho && columnas > 0 {
			lineas = append(lineas, actual.String())
			actual.Reset()
			columnas = 0
		}
		actual.WriteRune(r)
		columnas += peso
	}
	lineas = append(lineas, actual.String())
	return lineas
}

// expandirTabs cambia los tabuladores por cuatro espacios, para que su ancho
// cuente en la superficie del bloque. Es una función pura.
func expandirTabs(texto string) string {
	return strings.ReplaceAll(texto, "\t", "    ")
}
