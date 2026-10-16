package session

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"localcli/internal/agent"
	"localcli/internal/llm"
)

// topeImagen es el tamaño máximo de una imagen adjunta. Por encima no se lee.
const topeImagen = 10 << 20

// extensionesImagen son las extensiones que se consideran candidatas antes de
// verificar el contenido del archivo.
var extensionesImagen = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".webp": true, ".bmp": true,
}

// imagenesDelTurno resuelve las imágenes que el usuario escribió en el mensaje:
// comprueba que el modelo las vea, pide permiso si salen de la carpeta del
// proyecto y las lee. Son efímeras: acompañan a este turno y no se persisten.
func (o *Orquestador) imagenesDelTurno(ctx context.Context, sesionID, texto string) []agent.Imagen {
	rutas := rutasImagen(texto)
	if len(rutas) == 0 {
		return nil
	}
	switch o.capacidadesActuales().Vision {
	case llm.NoSoportada:
		o.avisar(sesionID, "el modelo activo no declara visión; no se adjuntan imágenes",
			"cambia a un modelo multimodal si quieres que las vea")
		return nil
	case llm.Desconocida:
		o.avisar(sesionID, "no se pudo confirmar que el modelo vea imágenes; se adjuntan igual", "")
	}

	var imagenes []agent.Imagen
	for _, ruta := range rutas {
		abs := filepath.Clean(expandirHOME(ruta))
		if !filepath.IsAbs(abs) {
			abs = filepath.Join(o.cwd, abs)
		}
		imagen, ok := leerImagen(abs)
		if !ok {
			o.avisar(sesionID, "no se pudo adjuntar la imagen "+ruta, "comprueba que sea una imagen legible")
			continue
		}
		imagenes = append(imagenes, imagen)
	}
	return imagenes
}

// adjuntarAlUltimoUsuario copia las imágenes al último mensaje del usuario del
// historial: es a ese turno al que pertenecen. También quita las rutas de
// imagen del texto, ya que van adjuntas como datos y no deben confundir al
// modelo.
func adjuntarAlUltimoUsuario(historial []agent.Mensaje, imagenes []agent.Imagen, textoOriginal string) []agent.Mensaje {
	for i := len(historial) - 1; i >= 0; i-- {
		if historial[i].Rol == agent.RolUsuario {
			historial[i].Imagenes = imagenes
			historial[i].Contenido = quitarRutasImagen(historial[i].Contenido, textoOriginal)
			return historial
		}
	}
	return historial
}

// quitarRutasImagen elimina del texto las rutas de imagen que existan como
// archivo, para que no confundan al modelo (las imágenes ya van adjuntas).
func quitarRutasImagen(texto, textoOriginal string) string {
	rutas := rutasImagen(textoOriginal)
	if len(rutas) == 0 {
		return texto
	}
	for _, ruta := range rutas {
		texto = strings.ReplaceAll(texto, ruta, "")
	}
	return strings.TrimSpace(texto)
}

// rutasImagen devuelve, sin duplicados y en orden, las rutas del texto que
// existen como archivo y tienen extensión de imagen. La comprobación del tipo
// real se hace al leer.
func rutasImagen(texto string) []string {
	// Regex que encuentra rutas de archivo con extensiones de imagen.
	// Acepta:
	// - Rutas entre comillas: "ruta/archivo.png" o 'ruta/archivo.png'
	// - Rutas sin comillas que empiezan con /, ~/, ./, ../ y terminan en extensión de imagen
	// Permite espacios en la ruta (se valida existencia real con os.Stat).
	// Usa clases Unicode para permitir letras acentuadas en nombres de archivo.
	re := regexp.MustCompile(`(?:"[^"]*\.(?:png|jpg|jpeg|gif|webp|bmp)"|'[^']*\.(?:png|jpg|jpeg|gif|webp|bmp)'|(?:^|\s)(?:/|~|\./|\.\./)[/\p{L}\p{N}._\-\s]*?\.(?:png|jpg|jpeg|gif|webp|bmp))\b`)

	visto := map[string]bool{}
	var out []string
	matches := re.FindAllString(texto, -1)
	for _, match := range matches {
		// Quitar espacio inicial si lo hay (del grupo (^|\s))
		ruta := strings.TrimSpace(match)
		// Quitar comillas envolventes si las hay
		ruta = strings.Trim(ruta, "\"'")
		if ruta == "" || visto[ruta] {
			continue
		}
		info, err := os.Stat(expandirHOME(ruta))
		if err != nil || !info.Mode().IsRegular() {
			continue
		}
		visto[ruta] = true
		out = append(out, ruta)
	}
	return out
}

// leerImagen lee un archivo de imagen, dentro del tope y con tipo reconocido por
// su contenido.
func leerImagen(abs string) (agent.Imagen, bool) {
	info, err := os.Stat(abs)
	if err != nil || !info.Mode().IsRegular() || info.Size() > topeImagen || info.Size() == 0 {
		return agent.Imagen{}, false
	}
	datos, err := os.ReadFile(abs)
	if err != nil {
		return agent.Imagen{}, false
	}
	mime, ok := mimeDeImagen(datos)
	if !ok {
		return agent.Imagen{}, false
	}
	return agent.Imagen{Ruta: abs, MIME: mime, Datos: datos}, true
}

// mimeDeImagen reconoce el tipo por el contenido, no por la extensión.
func mimeDeImagen(datos []byte) (string, bool) {
	if len(datos) == 0 {
		return "", false
	}
	muestra := datos
	if len(muestra) > 512 {
		muestra = muestra[:512]
	}
	switch http.DetectContentType(muestra) {
	case "image/png", "image/jpeg", "image/gif", "image/webp", "image/bmp":
		return http.DetectContentType(muestra), true
	}
	return "", false
}

// expandirHOME resuelve un `~` inicial al directorio del usuario.
func expandirHOME(ruta string) string {
	if ruta != "~" && !strings.HasPrefix(ruta, "~/") {
		return ruta
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ruta
	}
	return filepath.Join(home, strings.TrimPrefix(strings.TrimPrefix(ruta, "~"), "/"))
}
