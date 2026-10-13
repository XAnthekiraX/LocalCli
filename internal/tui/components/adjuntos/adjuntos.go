// Package adjuntos guarda la correspondencia token→valor de los archivos,
// carpetas y textos pegados o arrastrados en una línea de entrada. La línea
// muestra el token —por ejemplo `[foto.png]`— y al enviar se expande a su valor
// real: la ruta o el texto entero. Es efímero: vive solo mientras la línea tiene
// el token. Lo comparten la caja de entrada principal y la de bienvenida.
package adjuntos

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"localcli/internal/tui/render"
)

// Adjuntos guarda la correspondencia token→valor de los archivos, carpetas y
// textos pegados o arrastrados en la línea.
type Adjuntos struct {
	valores map[string]string
}

// Anotar sustituye en el texto pegado cada ruta —archivo o carpeta— por su token
// y guarda la correspondencia. Un pegado de una sola línea tokeniza toda ruta
// existente que aparezca; uno de varias líneas se resume como `[PEGADO N líneas]`,
// salvo que todas sus líneas sean rutas, en cuyo caso se anota un token por
// elemento. Devuelve el texto a insertar en la línea.
func (a *Adjuntos) Anotar(texto string) string {
	// Un salto de línea final no convierte un pegado de una línea en pegado de dos.
	limpio := strings.TrimRight(texto, "\r\n")
	if strings.Contains(limpio, "\n") {
		return a.anotarMultilinea(limpio)
	}
	for _, ruta := range rutasExistentes(limpio) {
		limpio = strings.ReplaceAll(limpio, ruta, a.tokenDe(ruta))
	}
	return limpio
}

// anotarMultilinea resume un pegado de varias líneas. Si TODAS las líneas son
// rutas existentes (arrastrar varios archivos o carpetas a la vez), deja un token
// por elemento; si no, guarda el texto entero bajo `[PEGADO N líneas]`.
func (a *Adjuntos) anotarMultilinea(texto string) string {
	lineas := strings.Split(texto, "\n")
	tokens := make([]string, 0, len(lineas))
	for _, l := range lineas {
		ruta := limpiarToken(strings.TrimSpace(l))
		if ruta == "" || !esRuta(ruta) {
			return a.tokenPegado(texto, len(lineas))
		}
		tokens = append(tokens, a.tokenDe(ruta))
	}
	if len(tokens) == 0 {
		return a.tokenPegado(texto, len(lineas))
	}
	return strings.Join(tokens, " ")
}

// tokenDe registra la ruta bajo su token y lo devuelve: `[base.ext]` para un
// archivo y `[CARPETA N elementos]` para una carpeta. Dos archivos con el mismo
// nombre no se confunden: el segundo conserva su ruta tal cual.
func (a *Adjuntos) tokenDe(ruta string) string {
	if esCarpeta(ruta) {
		return a.tokenCarpeta(ruta)
	}
	token := "[" + filepath.Base(ruta) + "]"
	if previo, ok := a.valores[token]; ok && previo != ruta {
		return ruta
	}
	return a.registrar(token, ruta)
}

// tokenCarpeta registra la carpeta bajo `[CARPETA N elementos]`, con N entradas
// de primer nivel. Si la carpeta no se puede leer, conserva la ruta tal cual.
func (a *Adjuntos) tokenCarpeta(ruta string) string {
	entradas, err := os.ReadDir(ruta)
	if err != nil {
		return ruta
	}
	unidad := "elementos"
	if len(entradas) == 1 {
		unidad = "elemento"
	}
	return a.registrar("[CARPETA "+strconv.Itoa(len(entradas))+" "+unidad+"]", ruta)
}

// tokenPegado registra el texto completo bajo `[PEGADO N líneas]` y lo devuelve.
func (a *Adjuntos) tokenPegado(texto string, n int) string {
	return a.registrar("[PEGADO "+strconv.Itoa(n)+" líneas]", texto)
}

// registrar guarda un valor bajo el token base y lo devuelve. Si base ya apuntara
// a otro valor, le añade un sufijo para que dos tokens distintos no se pisen.
func (a *Adjuntos) registrar(base, valor string) string {
	if a.valores == nil {
		a.valores = map[string]string{}
	}
	token := base
	for i := 2; ; i++ {
		previo, ok := a.valores[token]
		if !ok || previo == valor {
			break
		}
		token = base[:len(base)-1] + " (" + strconv.Itoa(i) + ")]"
	}
	a.valores[token] = valor
	return token
}

// Expandir devuelve el texto con cada token sustituido por su valor real.
func (a *Adjuntos) Expandir(texto string) string {
	for token, valor := range a.valores {
		texto = strings.ReplaceAll(texto, token, valor)
	}
	return texto
}

// Estilizar pinta una línea para la caja: cada token con EstiloAdjunto y el resto
// con el estilo base del texto. Reconoce el token más largo en cada posición, para
// que dos tokens que compartan prefijo no se pisen. Si el cursor de la línea parte
// un token, ese deja de ser subcadena y no se resalta: se pinta tal cual.
func (a *Adjuntos) Estilizar(linea string) string {
	if len(a.valores) == 0 {
		return render.EstiloBlanco.Render(linea)
	}
	tokens := make([]string, 0, len(a.valores))
	for token := range a.valores {
		tokens = append(tokens, token)
	}
	sort.Slice(tokens, func(i, j int) bool { return len(tokens[i]) > len(tokens[j]) })

	var salida, base strings.Builder
	for i := 0; i < len(linea); {
		coincidido := ""
		for _, token := range tokens {
			if strings.HasPrefix(linea[i:], token) {
				coincidido = token
				break
			}
		}
		if coincidido != "" {
			if base.Len() > 0 {
				salida.WriteString(render.EstiloBlanco.Render(base.String()))
				base.Reset()
			}
			salida.WriteString(render.EstiloAdjunto.Render(coincidido))
			i += len(coincidido)
			continue
		}
		r, ancho := utf8.DecodeRuneInString(linea[i:])
		base.WriteRune(r)
		i += ancho
	}
	if base.Len() > 0 {
		salida.WriteString(render.EstiloBlanco.Render(base.String()))
	}
	return salida.String()
}

// Olvidar descarta las correspondencias al vaciar la línea.
func (a *Adjuntos) Olvidar() { a.valores = nil }

// rutasExistentes devuelve, en orden y sin duplicados, las rutas —archivos o
// carpetas— que aparecen como tokens en el texto y existen en disco. Un token
// cuenta como ruta si, tras limpiar comillas y signos de puntuación, existe.
// Las rutas relativas se resuelven contra el directorio de trabajo del proceso.
func rutasExistentes(texto string) []string {
	visto := map[string]bool{}
	var out []string
	for _, token := range strings.Fields(texto) {
		ruta := limpiarToken(token)
		if ruta == "" || visto[ruta] || !esRuta(ruta) {
			continue
		}
		visto[ruta] = true
		out = append(out, ruta)
	}
	return out
}

// esRuta dice si la ruta existe en disco, sea archivo o carpeta.
func esRuta(ruta string) bool {
	_, err := os.Stat(ruta)
	return err == nil
}

// esCarpeta dice si la ruta existe y es una carpeta.
func esCarpeta(ruta string) bool {
	info, err := os.Stat(ruta)
	return err == nil && info.IsDir()
}

// limpiarToken quita la puntuación con la que el usuario suele envolver una ruta
// al escribirla (comillas, backticks, paréntesis, comas…).
func limpiarToken(t string) string {
	return strings.Trim(t, "`\"'()[]{}<>,;:")
}
