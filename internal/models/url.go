package models

import (
	"net/url"
	"path"
	"strings"
)

// ParsearURL interpreta una URL de descarga directa de un GGUF. Reconoce las
// URLs de Hugging Face (`…/<org>/<repo>/resolve/<rev>/<archivo>`) para poder
// resolver tamaño y hash por su API; cualquier otra URL `http(s)` se acepta como
// descarga directa, con el tamaño que declare el servidor. El nombre del archivo
// se guarda sin subcarpetas: cada modelo vive en su propia carpeta.
func ParsearURL(raw string) (Variante, error) {
	recortado := strings.TrimSpace(raw)
	if recortado == "" {
		return Variante{}, errorf("falta la URL")
	}
	u, err := url.Parse(recortado)
	if err != nil {
		return Variante{}, errorf("la URL no es válida: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return Variante{}, errorf("la URL debe empezar por http:// o https://")
	}
	if u.Host == "" {
		return Variante{}, errorf("la URL no tiene host")
	}
	segmentos := strings.Split(strings.Trim(u.Path, "/"), "/")
	archivo := segmentos[len(segmentos)-1]
	if archivo == "" || archivo == "." {
		return Variante{}, errorf("la URL no apunta a un archivo")
	}
	if !esGGUF(archivo) {
		return Variante{}, errorf("la URL debe apuntar a un archivo .gguf")
	}
	v := Variante{Archivo: path.Base(archivo), URL: u.String()}
	if repo := repoDeHuggingFace(u.Host, segmentos); repo != "" {
		v.Repo = repo
	}
	return v, nil
}

// NombreDeURL deriva el nombre del modelo a partir de su URL: el nombre del
// archivo sin la extensión `.gguf`. Es tolerante a propósito —el modal lo llama
// mientras se teclea—: una URL a medias devuelve lo que se pueda leer o cadena
// vacía.
func NombreDeURL(raw string) string {
	recortado := strings.TrimSpace(raw)
	if recortado == "" {
		return ""
	}
	u, err := url.Parse(recortado)
	if err != nil {
		return ""
	}
	base := path.Base(strings.Trim(u.Path, "/"))
	if base == "." || base == "/" || base == "" {
		return ""
	}
	if len(base) >= 5 && strings.EqualFold(base[len(base)-5:], ".gguf") {
		base = base[:len(base)-5]
	}
	return base
}

// repoDeHuggingFace devuelve el id `org/repo` si la URL pertenece a Hugging Face
// y sigue el patrón de descarga de un archivo de un repositorio.
func repoDeHuggingFace(host string, segmentos []string) string {
	if host != "huggingface.co" && !strings.HasSuffix(host, ".huggingface.co") {
		return ""
	}
	for i, s := range segmentos {
		if s == "resolve" && i >= 2 {
			return segmentos[i-2] + "/" + segmentos[i-1]
		}
	}
	return ""
}
