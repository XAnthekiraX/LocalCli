package models

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"sort"
	"strings"
)

// URLHuggingFace es la base de la API pública de Hugging Face.
const URLHuggingFace = "https://huggingface.co"

// HF es el cliente contra Hugging Face. `baseURL` permite apuntar a un mirror o
// a un servidor de prueba.
type HF struct {
	baseURL string
	http    *http.Client
}

// NuevoHF devuelve un cliente contra la API pública de Hugging Face.
func NuevoHF() *HF {
	return &HF{baseURL: URLHuggingFace, http: &http.Client{}}
}

// Variante es una variante GGUF de un repositorio, lista para descargar.
type Variante struct {
	Repo    string
	Archivo string
	// URL es la dirección de descarga cuando la variante viene de una URL
	// directa en vez de un repositorio; con ella, `URLDescarga` la respeta.
	URL    string
	Tamano int64  // 0 si el repositorio no lo publica
	SHA256 string // sha256 del LFS, en minúsculas; vacío si no lo publica
}

// entradaArbol es una entrada del árbol de archivos de un repositorio.
type entradaArbol struct {
	Tipo   string `json:"type"`
	Ruta   string `json:"path"`
	Tamano int64  `json:"size"`
	LFS    *struct {
		OID    string `json:"oid"`
		Tamano int64  `json:"size"`
	} `json:"lfs"`
}

// ListarGGUF lista las variantes `.gguf` de un repositorio, ordenadas por nombre.
func (h *HF) ListarGGUF(ctx context.Context, repo string) ([]Variante, error) {
	if strings.TrimSpace(repo) == "" {
		return nil, errorf("falta el repositorio")
	}
	destino := strings.TrimSuffix(h.baseURL, "/") + "/api/models/" + repo + "/tree/main?recursive=true"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, destino, nil)
	if err != nil {
		return nil, errorf("no se pudo preparar la consulta: %w", err)
	}

	resp, err := h.http.Do(req)
	if err != nil {
		return nil, errorf("no se pudo consultar %s: %w", repo, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errorf("Hugging Face respondió %s para %s: %s", resp.Status, repo, trozoDeRespuesta(resp.Body))
	}

	var entradas []entradaArbol
	if err := json.NewDecoder(resp.Body).Decode(&entradas); err != nil {
		return nil, errorf("el árbol de %s no se pudo leer: %w", repo, err)
	}

	variantes := make([]Variante, 0, len(entradas))
	for _, e := range entradas {
		if e.Tipo != "file" || !esGGUF(e.Ruta) {
			continue
		}
		v := Variante{Repo: repo, Archivo: e.Ruta, Tamano: e.Tamano}
		if e.LFS != nil {
			if v.Tamano == 0 {
				v.Tamano = e.LFS.Tamano
			}
			v.SHA256 = strings.ToLower(strings.TrimSpace(e.LFS.OID))
		}
		variantes = append(variantes, v)
	}
	sort.Slice(variantes, func(i, j int) bool { return variantes[i].Archivo < variantes[j].Archivo })
	return variantes, nil
}

// Seleccionar elige la variante con ese nombre exacto de archivo.
func (h *HF) Seleccionar(ctx context.Context, repo, archivo string) (Variante, error) {
	variantes, err := h.ListarGGUF(ctx, repo)
	if err != nil {
		return Variante{}, err
	}
	for _, v := range variantes {
		if v.Archivo == archivo {
			return v, nil
		}
	}
	return Variante{}, errorf("el repositorio %s no tiene la variante %q", repo, archivo)
}

// EsProyector indica si la variante es un proyector multimodal (`mmproj`), no un
// modelo de lenguaje: se guarda junto al GGUF del modelo para darle visión.
func (v Variante) EsProyector() bool {
	return strings.Contains(strings.ToLower(filepath.Base(v.Archivo)), "mmproj")
}

// URLDescarga resuelve la URL de descarga de una variante. Si la variante ya
// trae su propia URL —una descarga directa—, la devuelve tal cual.
func (h *HF) URLDescarga(v Variante) string {
	if v.URL != "" {
		return v.URL
	}
	return strings.TrimSuffix(h.baseURL, "/") + "/" + v.Repo + "/resolve/main/" + url.PathEscape(v.Archivo)
}

// TamanoURL consulta por HEAD el tamaño declarado de una URL. Un servidor que no
// lo publique, que no admita HEAD o una respuesta que no sea 200 deja el tamaño
// en 0: la descarga sigue, solo que sin barra de progreso.
func (h *HF) TamanoURL(ctx context.Context, raw string) int64 {
	if h == nil || h.http == nil || strings.TrimSpace(raw) == "" {
		return 0
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, raw, nil)
	if err != nil {
		return 0
	}
	resp, err := h.http.Do(req)
	if err != nil {
		return 0
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0
	}
	return resp.ContentLength
}

func esGGUF(ruta string) bool {
	return strings.HasSuffix(strings.ToLower(ruta), ".gguf")
}

func errorf(format string, args ...any) error {
	return fmt.Errorf("modelos: "+format, args...)
}

// trozoDeRespuesta lee un fragmento corto del cuerpo para el mensaje de un error.
func trozoDeRespuesta(r io.Reader) string {
	const tope = 200
	datos, _ := io.ReadAll(io.LimitReader(r, tope))
	return strings.TrimSpace(string(datos))
}
