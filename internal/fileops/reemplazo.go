package fileops

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"localcli/internal/tools"
)

// Este archivo implementa el núcleo de editar_archivo: una sustitución
// tolerante a los fallos típicos de los modelos al copiar texto — finales de
// línea CRLF, BOM UTF-8, comillas y guiones tipográficos, NBSP y espacios
// finales o de sangría — con coincidencia única obligatoria (nunca reemplaza
// en bloque como un ReplaceAll silencioso).

// bomUTF8 marca el BOM que algunos editores dejan al inicio del archivo.
const bomUTF8 = "\ufeff"

// aplicarReemplazos aplica los cambios en orden sobre `antes` y devuelve el
// resultado. Cada `buscar` debe coincidir exactamente una vez; si no se
// encuentra o es ambiguo, devuelve un error accionable para el modelo.
func aplicarReemplazos(antes string, cambios []tools.Cambio) (string, error) {
	if len(cambios) == 0 {
		return "", fmt.Errorf("%s: `reemplazos` no puede estar vacío", tools.CodBadArgs)
	}
	conBOM := strings.HasPrefix(antes, bomUTF8)
	texto := strings.TrimPrefix(antes, bomUTF8)
	crlf := detectarCRLF(texto)

	for i, cambio := range cambios {
		n := i + 1
		if cambio.Buscar == "" {
			return "", fmt.Errorf("%s: el fragmento %d tiene `buscar` vacío", tools.CodBadArgs, n)
		}
		if cambio.Buscar == cambio.Reemplazo {
			return "", fmt.Errorf("%s: el fragmento %d es idéntico a su reemplazo", tools.CodBadArgs, n)
		}
		rango, ocurrencias, err := unicaOcurrencia(texto, cambio.Buscar)
		if err != nil {
			return "", fmt.Errorf("fragmento %d: %w", n, err)
		}
		if ocurrencias > 1 {
			linea := contarLinea(texto, rango.inicio)
			return "", fmt.Errorf("el fragmento %d %s aparece %d veces (primera en la línea %d); "+
				"añade más contexto —líneas de alrededor— para que sea único",
				n, citar(cambio.Buscar), ocurrencias, linea)
		}
		reemplazo := reemplazoConEOL(cambio.Reemplazo, crlf)
		texto = texto[:rango.inicio] + reemplazo + texto[rango.fin:]
	}
	if conBOM {
		texto = bomUTF8 + texto
	}
	return texto, nil
}

// rango son offsets de byte [inicio, fin) dentro del texto original.
type rango struct{ inicio, fin int }

// vista es el texto con CRLF y Unicode tipográfico normalizados, con los mapas
// que traducen cada posición a los offsets del texto original.
type vista struct {
	norm      string
	canon     string
	mapiCanon []int // byte canon → inicio del rune origen
	mapfCanon []int // byte canon → fin exclusivo del rune origen
	mapiNorm  []int // byte norm → inicio original
	mapfNorm  []int // byte norm → fin exclusivo original
	lineas    []lineaOffset
}

type lineaOffset struct {
	inicio, fin int // fin excluye el \n
}

// nuevaVista construye la vista normalizada de `texto`.
func nuevaVista(texto string) vista {
	norm, mapiN, mapfN := normalizarConMapa(texto)
	canon, mapiC, mapfC := unicodeConMapa(norm)
	return vista{
		norm: norm, canon: canon, mapiCanon: mapiC, mapfCanon: mapfC,
		mapiNorm: mapiN, mapfNorm: mapfN,
		lineas: lineasDe(norm),
	}
}

// unicaOcurrencia localiza el fragmento con la primera estrategia que encuentra
// alguna coincidencia y devuelve su rango en el texto original y el total.
func unicaOcurrencia(texto, patron string) (rango, int, error) {
	v := nuevaVista(texto)
	patronN := normalizarLineas(patron)

	// 1) Subcadena exacta: admite ediciones a media línea.
	if oc := ocurrenciasDe(patron, texto); len(oc) > 0 {
		return rango{oc[0], oc[0] + len(patron)}, len(oc), nil
	}
	// 2) Subcadena con CRLF y Unicode normalizados.
	if oc := ocurrenciasDe(normalizarUnicode(patronN), v.canon); len(oc) > 0 {
		inicio, fin := v.traducir(oc[0], oc[0]+len(normalizarUnicode(patronN)))
		return rango{inicio, fin}, len(oc), nil
	}
	// 3) Bloques de línea con tolerancia a espacios finales, sangría y Unicode.
	if r, n := coincidenciaPorLineas(v, patronN); n > 0 {
		return r, n, nil
	}
	return rango{}, 0, fmt.Errorf("no se encontró %s en el archivo; cópialo exactamente con leer_archivo, "+
		"respetando sangría, espacios y finales de línea", citar(patron))
}

// traducir mapea un rango [ini, fin) de la vista canon al texto original.
func (v vista) traducir(ini, fin int) (int, int) {
	normIni := v.mapiCanon[ini]
	normFin := v.mapfCanon[fin-1] // fin exclusivo en norm
	return v.traducirNorm(normIni, normFin)
}

// traducirNorm mapea un rango [ini, fin) de la vista norm al texto original.
func (v vista) traducirNorm(ini, fin int) (int, int) {
	if fin <= ini {
		return v.mapiNorm[ini], v.mapiNorm[ini]
	}
	return v.mapiNorm[ini], v.mapfNorm[fin-1]
}

// ocurrenciasDe devuelve los offsets de todas las apariciones de `patron` en
// `texto` (patron no vacío).
func ocurrenciasDe(patron, texto string) []int {
	var out []int
	desde := 0
	for {
		i := strings.Index(texto[desde:], patron)
		if i < 0 {
			return out
		}
		pos := desde + i
		out = append(out, pos)
		desde = pos + len(patron)
	}
}

// normalizarConMapa convierte CRLF y CR sueltos en \n con un mapa de cada byte
// normalizado a su inicio y fin exclusivo en el original.
func normalizarConMapa(texto string) (string, []int, []int) {
	var b strings.Builder
	b.Grow(len(texto))
	mapi := make([]int, 0, len(texto)+1)
	mapf := make([]int, 0, len(texto)+1)
	for i := 0; i < len(texto); {
		if texto[i] == '\r' && i+1 < len(texto) && texto[i+1] == '\n' {
			mapi = append(mapi, i)
			mapf = append(mapf, i+2)
			b.WriteByte('\n')
			i += 2
			continue
		}
		mapi = append(mapi, i)
		mapf = append(mapf, i+1)
		b.WriteByte(texto[i])
		i++
	}
	mapi = append(mapi, len(texto))
	mapf = append(mapf, len(texto))
	return b.String(), mapi, mapf
}

// sustitucionesUnicode mapea los caracteres típicos de salida de modelos y
// navegadores a su forma ASCII (mismo criterio que el apply_patch de opencode).
var sustitucionesUnicode = map[rune]string{
	'‘': "'", '’': "'", '‚': "'", '‛': "'",
	'“': `"`, '”': `"`, '„': `"`, '‟': `"`,
	'‐': "-", '‑': "-", '‒': "-", '–': "-", '—': "-", '―': "-",
	'…': "...",
	'\u00a0': " ", ' ': " ", ' ': " ", '　': " ", // espacios sin separación
}

// unicodeConMapa sustituye los caracteres tipográficos por su forma ASCII con
// un mapa de cada byte de salida a su rune origen (un rune puede producir
// varios bytes, p. ej. la elipsis a tres puntos).
func unicodeConMapa(texto string) (string, []int, []int) {
	var b strings.Builder
	b.Grow(len(texto))
	mapi := make([]int, 0, len(texto)+1)
	mapf := make([]int, 0, len(texto)+1)
	for i, r := range texto {
		salida, ok := sustitucionesUnicode[r]
		if !ok {
			salida = string(r)
		}
		finRune := i + utf8.RuneLen(r)
		for j := 0; j < len(salida); j++ {
			mapi = append(mapi, i)
			mapf = append(mapf, finRune)
		}
		b.WriteString(salida)
	}
	mapi = append(mapi, len(texto))
	mapf = append(mapf, len(texto))
	return b.String(), mapi, mapf
}

// normalizarUnicode aplica las mismas sustituciones sobre un texto entero.
func normalizarUnicode(texto string) string {
	var b strings.Builder
	b.Grow(len(texto))
	for _, r := range texto {
		if s, ok := sustitucionesUnicode[r]; ok {
			b.WriteString(s)
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// normalizarLineas convierte CRLF y CR a \n.
func normalizarLineas(texto string) string {
	return strings.ReplaceAll(strings.ReplaceAll(texto, "\r\n", "\n"), "\r", "\n")
}

func lineasDe(texto string) []lineaOffset {
	var out []lineaOffset
	inicio := 0
	for i := 0; i < len(texto); i++ {
		if texto[i] == '\n' {
			out = append(out, lineaOffset{inicio, i})
			inicio = i + 1
		}
	}
	out = append(out, lineaOffset{inicio, len(texto)})
	return out
}

// coincidenciaPorLineas busca bloques de líneas completas del patrón con
// tolerancia creciente: igualdad exacta, sin espacios finales, recortada y,
// por último, con Unicode normalizado.
func coincidenciaPorLineas(v vista, patronN string) (rango, int) {
	conFinal := strings.HasSuffix(patronN, "\n")
	patronL := strings.Split(patronN, "\n")
	if conFinal && len(patronL) > 0 {
		patronL = patronL[:len(patronL)-1]
	}
	if len(patronL) == 0 || (len(patronL) == 1 && patronL[0] == "") {
		return rango{}, 0
	}
	normTexto := v.norm
	lineasTexto := v.lineas
	total := len(lineasTexto)
	if len(patronL) > total {
		return rango{}, 0
	}
	for _, nivel := range []int{0, 1, 2, 3} {
		var oc []rango
		for i := 0; i+len(patronL) <= total; i++ {
			if bloqueCoincide(lineasTexto, normTexto, i, patronL, nivel) {
				inicio := lineasTexto[i].inicio
				ultimo := lineasTexto[i+len(patronL)-1]
				fin := ultimo.fin
				if conFinal && fin < len(normTexto) {
					fin++ // incluye el \n final
				}
				oc = append(oc, rango{inicio, fin})
			}
		}
		if len(oc) == 0 {
			continue
		}
		inicio, fin := v.traducirNorm(oc[0].inicio, oc[0].fin)
		return rango{inicio, fin}, len(oc)
	}
	return rango{}, 0
}

func bloqueCoincide(lineasTexto []lineaOffset, texto string, desde int, patronL []string, nivel int) bool {
	for j, p := range patronL {
		linea := texto[lineasTexto[desde+j].inicio:lineasTexto[desde+j].fin]
		if !lineasCoinciden(linea, p, nivel) {
			return false
		}
	}
	return true
}

func lineasCoinciden(a, b string, nivel int) bool {
	switch nivel {
	case 0:
		return a == b
	case 1:
		return strings.TrimRight(a, " \t") == strings.TrimRight(b, " \t")
	case 2:
		return strings.TrimSpace(a) == strings.TrimSpace(b)
	default:
		return normalizarUnicode(strings.TrimSpace(a)) == normalizarUnicode(strings.TrimSpace(b))
	}
}

// detectarCRLF decide el estilo de fin de línea del archivo por su primera
// ocurrencia, como hace opencode.
func detectarCRLF(texto string) bool {
	i := strings.IndexByte(texto, '\n')
	return i > 0 && texto[i-1] == '\r'
}

func reemplazoConEOL(reemplazo string, crlf bool) string {
	r := normalizarLineas(reemplazo)
	if crlf {
		r = strings.ReplaceAll(r, "\n", "\r\n")
	}
	return r
}

func contarLinea(texto string, offset int) int {
	if offset > len(texto) {
		offset = len(texto)
	}
	return 1 + strings.Count(texto[:offset], "\n")
}

// cita un fragmento para el mensaje de error, acotado por runes y escapado.
func citar(s string) string {
	const tope = 60
	if len(s) > tope {
		corte := tope
		for corte > 0 && !utf8.RuneStart(s[corte]) {
			corte--
		}
		s = s[:corte] + "…"
	}
	return strconv.Quote(s)
}
