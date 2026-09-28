// adjuntos.go — T-F026: detectar imágenes en el texto del turno.
//
// Fuente de verdad: [[specs/SPEC-INTERFAZ]] (la entrada compone y envía) y
// [[specs/SPEC-OLLAMA-PERFIL]] (los modelos multimodales reciben imágenes). La
// vista no decide qué entiende el modelo: solo detecta rutas de imagen en lo que
// el usuario escribió, las lee y las entrega codificadas en base64 —la forma que
// espera `/api/chat`—. Quien sabe de modelos es `ollama`, detrás del puerto.
//
// Es best-effort: un archivo que no está se ignora (evita falsos positivos en
// prosa como «renombra logo.png»), y un archivo ilegible se avisa sin cortar el
// envío. Las imágenes son efímeras: acompañan a este turno y no se persisten.
package tui

import (
	"encoding/base64"
	"os"
	"path/filepath"
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

// Anotar sustituye en el texto pegado cada ruta de imagen por su token y guarda
// la correspondencia. Devuelve el texto a insertar en la línea. Dos archivos
// con el mismo nombre no se confunden: el segundo conserva su ruta tal cual.
func (a *adjuntos) Anotar(texto string) string {
	for _, ruta := range RutasImagen(texto) {
		token := "[" + filepath.Base(ruta) + "]"
		if previa, ok := a.rutas[token]; ok && previa != ruta {
			continue
		}
		if a.rutas == nil {
			a.rutas = map[string]string{}
		}
		a.rutas[token] = ruta
		texto = strings.ReplaceAll(texto, ruta, token)
	}
	return texto
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
