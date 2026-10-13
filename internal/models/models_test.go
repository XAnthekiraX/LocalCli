package models

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// contenidoPrueba devuelve n bytes con un patrón determinista.
func contenidoPrueba(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(i % 251)
	}
	return b
}

// servidorFalso simula el árbol y la descarga de Hugging Face, con `Range`.
type servidorFalso struct {
	*httptest.Server
	Repo          string
	Archivo       string
	Contenido     []byte
	Tamano        int64  // declarado en el árbol; 0 => len(Contenido)
	SHA           string // sha256 declarado; "" => sin hash
	ResolveStatus int    // != 0 => la descarga responde con ese estado

	mu     sync.Mutex
	hits   int
	rangos []string
}

func nuevoServidorFalso(t *testing.T, repo, archivo string, contenido []byte) *servidorFalso {
	t.Helper()
	s := &servidorFalso{Repo: repo, Archivo: archivo, Contenido: contenido}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/models/"+repo+"/tree/main", func(w http.ResponseWriter, _ *http.Request) {
		tam := s.Tamano
		if tam == 0 {
			tam = int64(len(s.Contenido))
		}
		entrada := map[string]any{"type": "file", "path": archivo, "size": tam}
		if s.SHA != "" {
			entrada["lfs"] = map[string]any{"oid": s.SHA, "size": tam}
		}
		_ = json.NewEncoder(w).Encode([]any{
			entrada,
			map[string]any{"type": "file", "path": "README.md", "size": 3},
		})
	})
	mux.HandleFunc("/"+repo+"/resolve/main/"+archivo, func(w http.ResponseWriter, r *http.Request) {
		s.contarHit()
		if s.ResolveStatus != 0 {
			w.WriteHeader(s.ResolveStatus)
			return
		}
		if rg := r.Header.Get("Range"); rg != "" {
			s.anotarRango(rg)
			desde := 0
			_, _ = fmt.Sscanf(rg, "bytes=%d-", &desde)
			if desde > 0 && desde < len(s.Contenido) {
				w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", desde, len(s.Contenido)-1, len(s.Contenido)))
				w.WriteHeader(http.StatusPartialContent)
				_, _ = w.Write(s.Contenido[desde:])
				return
			}
		}
		_, _ = w.Write(s.Contenido)
	})

	s.Server = httptest.NewServer(mux)
	t.Cleanup(s.Server.Close)
	return s
}

func (s *servidorFalso) contarHit() {
	s.mu.Lock()
	s.hits++
	s.mu.Unlock()
}

func (s *servidorFalso) anotarRango(rg string) {
	s.mu.Lock()
	s.rangos = append(s.rangos, rg)
	s.mu.Unlock()
}

func (s *servidorFalso) Hits() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.hits
}

func (s *servidorFalso) Rangos() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, len(s.rangos))
	copy(out, s.rangos)
	return out
}

func (s *servidorFalso) cliente() *HF {
	return &HF{baseURL: s.URL, http: s.Client()}
}

func sha256Hex(b []byte) string {
	suma := sha256.Sum256(b)
	return hex.EncodeToString(suma[:])
}

func TestListarGGUFYSeleccionar(t *testing.T) {
	contenido := contenidoPrueba(2048)
	srv := nuevoServidorFalso(t, "org/repo", "Qwen-8B-Q4_K_M.gguf", contenido)
	srv.SHA = sha256Hex(contenido)

	variantes, err := srv.cliente().ListarGGUF(context.Background(), "org/repo")
	if err != nil {
		t.Fatalf("listar: %v", err)
	}
	if len(variantes) != 1 {
		t.Fatalf("variantes = %d, quiero 1 (solo el .gguf): %+v", len(variantes), variantes)
	}
	v := variantes[0]
	if v.Archivo != "Qwen-8B-Q4_K_M.gguf" || v.Tamano != int64(len(contenido)) || v.SHA256 != srv.SHA {
		t.Fatalf("variante mal leída: %+v", v)
	}

	elegida, err := srv.cliente().Seleccionar(context.Background(), "org/repo", "Qwen-8B-Q4_K_M.gguf")
	if err != nil {
		t.Fatalf("seleccionar: %v", err)
	}
	if elegida.Archivo != v.Archivo {
		t.Fatalf("seleccionó %q", elegida.Archivo)
	}
	if _, err := srv.cliente().Seleccionar(context.Background(), "org/repo", "no-existe.gguf"); err == nil {
		t.Fatal("una variante ausente debería dar error")
	}
}

func TestListadoSinGGUF(t *testing.T) {
	srv := nuevoServidorFalso(t, "org/solo-texto", "notas.txt", contenidoPrueba(10))

	variantes, err := srv.cliente().ListarGGUF(context.Background(), "org/solo-texto")
	if err != nil {
		t.Fatalf("listar: %v", err)
	}
	if len(variantes) != 0 {
		t.Fatalf("esperaba lista vacía: %+v", variantes)
	}
}

func TestURLDescarga(t *testing.T) {
	h := &HF{baseURL: "https://huggingface.co", http: &http.Client{}}
	u := h.URLDescarga(Variante{Repo: "org/repo", Archivo: "m.gguf"})
	if u != "https://huggingface.co/org/repo/resolve/main/m.gguf" {
		t.Fatalf("URL de descarga = %q", u)
	}
}

func TestDescargaCompletaConProgreso(t *testing.T) {
	contenido := contenidoPrueba(600_000)
	srv := nuevoServidorFalso(t, "org/repo", "m.gguf", contenido)
	srv.SHA = sha256Hex(contenido)
	raiz := t.TempDir()

	var ultimo int64
	d := NuevoDescargador(srv.cliente(), raiz).ConProgreso(func(hecho, _ int64) { ultimo = hecho })

	ruta, err := d.Descargar(context.Background(), Variante{Repo: "org/repo", Archivo: "m.gguf", Tamano: int64(len(contenido)), SHA256: srv.SHA}, "Qwen3.8")
	if err != nil {
		t.Fatalf("descargar: %v", err)
	}
	if ruta != filepath.Join(raiz, "Qwen3.8", "m.gguf") {
		t.Fatalf("ruta final = %q", ruta)
	}
	datos, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatal(err)
	}
	if string(datos) != string(contenido) {
		t.Fatal("el contenido descargado no coincide")
	}
	if _, err := os.Stat(ruta + ".part"); !os.IsNotExist(err) {
		t.Fatal("el .part debería haberse renombrado")
	}
	if ultimo != int64(len(contenido)) {
		t.Fatalf("progreso final = %d, quiero %d", ultimo, len(contenido))
	}
}

func TestNoReDescargaSiExisteCompleto(t *testing.T) {
	contenido := contenidoPrueba(4096)
	srv := nuevoServidorFalso(t, "org/repo", "m.gguf", contenido)
	raiz := t.TempDir()

	destDir := filepath.Join(raiz, "Qwen3.8")
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(destDir, "m.gguf"), contenido, 0o644); err != nil {
		t.Fatal(err)
	}

	ruta, err := NuevoDescargador(srv.cliente(), raiz).
		Descargar(context.Background(), Variante{Repo: "org/repo", Archivo: "m.gguf", Tamano: int64(len(contenido))}, "Qwen3.8")
	if err != nil {
		t.Fatalf("descargar: %v", err)
	}
	if ruta == "" {
		t.Fatal("debería devolver la ruta del archivo presente")
	}
	if srv.Hits() != 0 {
		t.Fatalf("no debería pedirse la descarga: %d peticiones", srv.Hits())
	}
}

func TestReanudaDescargaInterrumpida(t *testing.T) {
	contenido := contenidoPrueba(600_000)
	srv := nuevoServidorFalso(t, "org/repo", "m.gguf", contenido)
	raiz := t.TempDir()

	destDir := filepath.Join(raiz, "Qwen3.8")
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		t.Fatal(err)
	}
	const corte = 100_000
	if err := os.WriteFile(filepath.Join(destDir, "m.gguf.part"), contenido[:corte], 0o644); err != nil {
		t.Fatal(err)
	}

	ruta, err := NuevoDescargador(srv.cliente(), raiz).
		Descargar(context.Background(), Variante{Repo: "org/repo", Archivo: "m.gguf", Tamano: int64(len(contenido))}, "Qwen3.8")
	if err != nil {
		t.Fatalf("descargar: %v", err)
	}
	if rangos := srv.Rangos(); len(rangos) != 1 || rangos[0] != fmt.Sprintf("bytes=%d-", corte) {
		t.Fatalf("rangos pedidos = %v", rangos)
	}
	datos, _ := os.ReadFile(ruta)
	if string(datos) != string(contenido) {
		t.Fatal("el archivo reanudado no coincide")
	}
}

func TestDescargaTruncadaSeDetecta(t *testing.T) {
	contenido := contenidoPrueba(4096)
	srv := nuevoServidorFalso(t, "org/repo", "m.gguf", contenido)
	srv.Tamano = int64(len(contenido)) + 100 // el árbol declara más de lo que sirve
	raiz := t.TempDir()

	ruta, err := NuevoDescargador(srv.cliente(), raiz).
		Descargar(context.Background(), Variante{Repo: "org/repo", Archivo: "m.gguf", Tamano: srv.Tamano}, "Qwen3.8")
	if err == nil {
		t.Fatal("una descarga incompleta debería fallar")
	}
	if ruta != "" {
		t.Fatalf("no debería devolver ruta: %q", ruta)
	}
	if _, err := os.Stat(filepath.Join(raiz, "Qwen3.8", "m.gguf")); !os.IsNotExist(err) {
		t.Fatal("no debería quedar archivo final")
	}
	if _, err := os.Stat(filepath.Join(raiz, "Qwen3.8", "m.gguf.part")); err != nil {
		t.Fatal("el .part debería quedar para reanudar")
	}
}

func TestHashIncorrectoDescartaElParcial(t *testing.T) {
	contenido := contenidoPrueba(4096)
	srv := nuevoServidorFalso(t, "org/repo", "m.gguf", contenido)
	raiz := t.TempDir()
	malo := strings.Repeat("0", 64)

	ruta, err := NuevoDescargador(srv.cliente(), raiz).
		Descargar(context.Background(), Variante{Repo: "org/repo", Archivo: "m.gguf", Tamano: int64(len(contenido)), SHA256: malo}, "Qwen3.8")
	if err == nil {
		t.Fatal("un hash que no cuadra debería fallar")
	}
	if ruta != "" {
		t.Fatalf("no debería devolver ruta: %q", ruta)
	}
	if _, err := os.Stat(filepath.Join(raiz, "Qwen3.8", "m.gguf.part")); !os.IsNotExist(err) {
		t.Fatal("el parcial corrupto debería borrarse")
	}
}

func TestFalloDeRedNoBorraModelosExistentes(t *testing.T) {
	raiz := t.TempDir()
	existente := filepath.Join(raiz, "existente", "m.gguf")
	if err := os.MkdirAll(filepath.Dir(existente), 0o755); err != nil {
		t.Fatal(err)
	}
	previo := []byte("modelo ya presente")
	if err := os.WriteFile(existente, previo, 0o644); err != nil {
		t.Fatal(err)
	}

	srv := nuevoServidorFalso(t, "org/repo", "nuevo.gguf", contenidoPrueba(2048))
	srv.ResolveStatus = http.StatusInternalServerError

	_, err := NuevoDescargador(srv.cliente(), raiz).
		Descargar(context.Background(), Variante{Repo: "org/repo", Archivo: "nuevo.gguf"}, "otro")
	if err == nil {
		t.Fatal("una descarga que falla debería dar error")
	}
	datos, err := os.ReadFile(existente)
	if err != nil {
		t.Fatalf("el modelo existente debería seguir: %v", err)
	}
	if string(datos) != string(previo) {
		t.Fatal("el modelo existente se alteró")
	}
}
