package llama

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// LocalizarBinario resuelve la ruta de `llama-server`: usa la ruta explícita si
// se dio y, si no, lo busca en PATH. Devuelve un error accionable cuando no
// aparece, para avisar sin bloquear el arranque.
func LocalizarBinario(ruta string) (string, error) {
	if strings.TrimSpace(ruta) != "" {
		info, err := os.Stat(ruta)
		if err != nil {
			return "", fmt.Errorf("llama: no está el binario %s", ruta)
		}
		if info.IsDir() {
			return "", fmt.Errorf("llama: %s es un directorio, no el binario", ruta)
		}
		if info.Mode().Perm()&0o111 == 0 {
			return "", fmt.Errorf("llama: %s no es ejecutable", ruta)
		}
		return ruta, nil
	}
	encontrado, err := exec.LookPath("llama-server")
	if err != nil {
		return "", fmt.Errorf("llama: no se encontró llama-server en PATH; instálalo o indica su ruta")
	}
	return encontrado, nil
}

// HostPuerto extrae el host y el puerto de la dirección base de la API
// (`api` de localcli.json). Sin puerto explícito usa el del esquema.
func HostPuerto(baseURL string) (string, int, error) {
	host, puerto, ok := separarHostPuerto(baseURL)
	if !ok {
		return "", 0, fmt.Errorf("llama: dirección de API inválida: %q", baseURL)
	}
	return host, puerto, nil
}

// separarHostPuerto analiza `http://host:puerto` sin depender de net/url.
func separarHostPuerto(baseURL string) (string, int, bool) {
	s := strings.TrimSpace(baseURL)
	esquema := "http"
	if i := strings.Index(s, "://"); i >= 0 {
		esquema = strings.ToLower(s[:i])
		s = s[i+3:]
	}
	if corte := strings.IndexAny(s, "/?#"); corte >= 0 {
		s = s[:corte]
	}
	if s == "" {
		return "", 0, false
	}
	predeterminado := 80
	if esquema == "https" {
		predeterminado = 443
	}
	host := s
	puerto := predeterminado
	if i := strings.LastIndex(s, ":"); i >= 0 {
		host = s[:i]
		n, err := strconv.Atoi(s[i+1:])
		if err != nil || n <= 0 || n > 65535 {
			return "", 0, false
		}
		puerto = n
	}
	if host == "" {
		host = "127.0.0.1"
	}
	return host, puerto, true
}

// CabeEnMemoria indica si un modelo de `tamano` bytes cabe en la RAM de la
// máquina, y si se pudo averiguar. Sin dato fiable no se afirma nada.
func CabeEnMemoria(tamano int64) (cabe bool, conocido bool) {
	total, ok := MemoriaTotal()
	if !ok || tamano <= 0 {
		return true, false
	}
	return tamano <= total, true
}

// MemoriaTotal devuelve la RAM total en bytes y si se pudo leer.
func MemoriaTotal() (int64, bool) {
	datos, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, false
	}
	for _, linea := range strings.Split(string(datos), "\n") {
		if !strings.HasPrefix(linea, "MemTotal:") {
			continue
		}
		campos := strings.Fields(linea)
		if len(campos) < 2 {
			return 0, false
		}
		kb, err := strconv.ParseInt(campos[1], 10, 64)
		if err != nil {
			return 0, false
		}
		return kb * 1024, true
	}
	return 0, false
}

// humano compone un tamaño legible para los avisos.
func humano(bytes int64) string {
	const (
		gb = 1 << 30
		mb = 1 << 20
	)
	switch {
	case bytes >= gb:
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(gb))
	case bytes >= mb:
		return fmt.Sprintf("%.0f MB", float64(bytes)/float64(mb))
	default:
		return fmt.Sprintf("%d bytes", bytes)
	}
}
