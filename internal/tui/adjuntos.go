// adjuntos.go — T-F026 / T-F038: tokens de archivo y de pegado en la entrada.
//
// Fuente de verdad: [[specs/SPEC-INTERFAZ]] (la entrada compone y envía) y
// [[specs/SPEC-OLLAMA-PERFIL]] (los modelos multimodales reciben imágenes). La
// vista no decide qué entiende el modelo: detecta en lo que el usuario pega o
// arrastra las rutas de archivo —las muestra como `[nombre.ext]` y las expande a
// su ruta real al enviar— y resume un pegado de varias líneas como
// `[PEGADO N líneas]`. Para las imágenes, además, lee el archivo y lo entrega
// codificado en base64 —la forma que espera `/api/chat`—; quien sabe de modelos
// es `ollama`, detrás del puerto.
//
// Es best-effort: un archivo que no está se ignora (evita falsos positivos en
// prosa como «renombra logo.png»), y un archivo ilegible se avisa sin cortar el
// envío. Los adjuntos son efímeros: acompañan a este turno y no se persisten.
package tui

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// extensionesImagen son las que se reconocen al buscar rutas en el texto. No se
// inspecciona el contenido: la extensión y la existencia bastan para adjuntar.
var extensionesImagen = map[string]bool{
	".png":  true,
	".jpg":  true,
	".jpeg": true,
	".gif":  true,
	".webp": true,
	".bmp":  true,
}

// RutasImagen devuelve, en orden y sin duplicados, las rutas de imagen que
// aparecen como tokens en el texto y existen en disco. Un token cuenta como
// imagen si, tras limpiar comillas y signos de puntuación, tiene extensión de
// imagen y el archivo existe. Rutas relativas se resuelven contra el directorio
// de trabajo del proceso (la raíz del proyecto).
func RutasImagen(texto string) []string {
	visto := map[string]bool{}
	var out []string
	for _, token := range strings.Fields(texto) {
		ruta := limpiarToken(token)
		if ruta == "" || visto[ruta] || !esRutaDeImagen(ruta) {
			continue
		}
		if _, err := os.Stat(ruta); err != nil {
			continue // no existe: no es una imagen real, se ignora en silencio
		}
		visto[ruta] = true
		out = append(out, ruta)
	}
	return out
}

// RutasExistentes devuelve, en orden y sin duplicados, las rutas —archivos o
// carpetas— que aparecen como tokens en el texto y existen en disco. Un token
// cuenta como ruta si, tras limpiar comillas y signos de puntuación, existe.
// Rutas relativas se resuelven contra el directorio de trabajo del proceso (la
// raíz del proyecto).
func RutasExistentes(texto string) []string {
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

// AdjuntosDe detecta rutas de imagen en el texto del turno, las lee y devuelve
// su contenido en base64 (sin prefijo `data:`), que es lo que espera `/api/chat`.
// Devuelve además los avisos de archivos que existen pero no se pudieron leer.
func AdjuntosDe(texto string) (imagenes []string, avisos []string) {
	for _, ruta := range RutasImagen(texto) {
		datos, err := os.ReadFile(ruta)
		if err != nil {
			avisos = append(avisos, "no se pudo leer la imagen "+ruta+": "+err.Error())
			continue
		}
		imagenes = append(imagenes, base64.StdEncoding.EncodeToString(datos))
	}
	return imagenes, avisos
}

// adjuntos guarda la correspondencia token→ruta de las imágenes pegadas o
// arrastradas en una línea de entrada. La línea muestra el token `[foto.png]`
// (resaltado); al enviar se expande a su ruta real, que es la que detecta y
// adjunta la imagen. Es efímero: vive solo mientras la línea tiene el token.
type adjuntos struct {
	rutas map[string]string
}

// Anotar sustituye en el texto pegado cada ruta —archivo o carpeta— por su token
// y guarda la correspondencia. Un pegado de una sola línea tokeniza toda ruta
// existente que aparezca —igual que las imágenes—; uno de varias líneas se
// resume como `[PEGADO N líneas]`, salvo que todas sus líneas sean rutas, en
// cuyo caso se anota un token por elemento. Devuelve el texto a insertar en la
// línea.
func (a *adjuntos) Anotar(texto string) string {
	// Un salto de línea final no convierte un pegado de una línea en pegado de dos.
	limpio := strings.TrimRight(texto, "\r\n")
	if strings.Contains(limpio, "\n") {
		return a.anotarMultilinea(limpio)
	}
	for _, ruta := range RutasExistentes(limpio) {
		limpio = strings.ReplaceAll(limpio, ruta, a.tokenDe(ruta))
	}
	return limpio
}

// anotarMultilinea resume un pegado de varias líneas. Si TODAS las líneas son
// rutas existentes (arrastrar varios archivos o carpetas a la vez), deja un
// token por elemento; si no, guarda el texto entero bajo `[PEGADO N líneas]`.
func (a *adjuntos) anotarMultilinea(texto string) string {
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
func (a *adjuntos) tokenDe(ruta string) string {
	if esCarpeta(ruta) {
		return a.tokenCarpeta(ruta)
	}
	token := "[" + filepath.Base(ruta) + "]"
	if previa, ok := a.rutas[token]; ok && previa != ruta {
		return ruta
	}
	if a.rutas == nil {
		a.rutas = map[string]string{}
	}
	a.rutas[token] = ruta
	return token
}

// tokenCarpeta registra la carpeta bajo `[CARPETA N elementos]`, con N entradas
// de primer nivel. Si la carpeta no se puede leer, conserva la ruta tal cual.
func (a *adjuntos) tokenCarpeta(ruta string) string {
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
func (a *adjuntos) tokenPegado(texto string, n int) string {
	return a.registrar("[PEGADO "+strconv.Itoa(n)+" líneas]", texto)
}

// registrar guarda un valor bajo el token base y lo devuelve. Si base ya
// apuntara a otro valor, le añade un sufijo para que dos tokens distintos no se
// pisen.
func (a *adjuntos) registrar(base, valor string) string {
	if a.rutas == nil {
		a.rutas = map[string]string{}
	}
	token := base
	for i := 2; ; i++ {
		previo, ok := a.rutas[token]
		if !ok || previo == valor {
			break
		}
		token = base[:len(base)-1] + " (" + strconv.Itoa(i) + ")]"
	}
	a.rutas[token] = valor
	return token
}

// Expandir devuelve el texto con cada token sustituido por su ruta real.
func (a *adjuntos) Expandir(texto string) string {
	for token, ruta := range a.rutas {
		texto = strings.ReplaceAll(texto, token, ruta)
	}
	return texto
}

// Resaltar pinta los tokens presentes en un texto ya renderizado. Si el cursor
// de la línea parte un token, ese no se resalta: se pinta tal cual.
func (a *adjuntos) Resaltar(texto string) string {
	for token := range a.rutas {
		if strings.Contains(texto, token) {
			texto = strings.ReplaceAll(texto, token, estiloAdjunto.Render(token))
		}
	}
	return texto
}

// Olvidar descarta las correspondencias al vaciar la línea.
func (a *adjuntos) Olvidar() { a.rutas = nil }

// limpiarToken quita la puntuación con la que el usuario suele envolver una ruta
// al escribirla (comillas, backticks, paréntesis, comas…).
func limpiarToken(t string) string {
	return strings.Trim(t, "`\"'()[]{}<>,;:")
}

// esRutaDeImagen dice si el token parece una imagen por su extensión.
func esRutaDeImagen(ruta string) bool {
	if ruta == "" {
		return false
	}
	return extensionesImagen[strings.ToLower(filepath.Ext(ruta))]
}
