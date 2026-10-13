package llama

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// binarioFalso crea un `llama-server` de mentira que apunta sus argumentos en el
// archivo de `ARGS_FILE` y espera a que lo maten.
func binarioFalso(t *testing.T) (ruta, argsFile string) {
	t.Helper()
	dir := t.TempDir()
	ruta = filepath.Join(dir, "llama-server")
	argsFile = filepath.Join(dir, "args.txt")
	guion := "#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$ARGS_FILE\"\nexec sleep 300\n"
	if err := os.WriteFile(ruta, []byte(guion), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ARGS_FILE", argsFile)
	return ruta, argsFile
}

// modeloFalso crea un GGUF de mentira.
func modeloFalso(t *testing.T) string {
	t.Helper()
	ruta := filepath.Join(t.TempDir(), "m.gguf")
	if err := os.WriteFile(ruta, []byte("gguf"), 0o644); err != nil {
		t.Fatal(err)
	}
	return ruta
}

func leerArgs(t *testing.T, ruta string) []string {
	t.Helper()
	espera := time.Now().Add(3 * time.Second)
	for time.Now().Before(espera) {
		datos, err := os.ReadFile(ruta)
		if err == nil && len(datos) > 0 {
			return strings.Split(strings.TrimSpace(string(datos)), "\n")
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("el binario falso no escribió sus argumentos en %s", ruta)
	return nil
}

func TestHostPuerto(t *testing.T) {
	casos := []struct {
		url    string
		host   string
		puerto int
	}{
		{"http://localhost:8080", "localhost", 8080},
		{"http://127.0.0.1:11434", "127.0.0.1", 11434},
		{"localhost:9000", "localhost", 9000},
		{"http://example.com", "example.com", 80},
		{"https://example.com", "example.com", 443},
		{"http://0.0.0.0:8080/v1", "0.0.0.0", 8080},
	}
	for _, c := range casos {
		host, puerto, err := HostPuerto(c.url)
		if err != nil {
			t.Errorf("HostPuerto(%q): %v", c.url, err)
			continue
		}
		if host != c.host || puerto != c.puerto {
			t.Errorf("HostPuerto(%q) = %s:%d, quiero %s:%d", c.url, host, puerto, c.host, c.puerto)
		}
	}
	if _, _, err := HostPuerto(""); err == nil {
		t.Error("una dirección vacía debería fallar")
	}
	if _, _, err := HostPuerto("http://h:99999"); err == nil {
		t.Error("un puerto fuera de rango debería fallar")
	}
}

func TestLocalizarBinario(t *testing.T) {
	bin, _ := binarioFalso(t)
	if got, err := LocalizarBinario(bin); err != nil || got != bin {
		t.Fatalf("ruta explícita: got %q, err %v", got, err)
	}
	if _, err := LocalizarBinario(filepath.Join(t.TempDir(), "no-existe")); err == nil {
		t.Fatal("una ruta inexistente debería fallar")
	}

	dir := t.TempDir()
	enPath := filepath.Join(dir, "llama-server")
	if err := os.WriteFile(enPath, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	got, err := LocalizarBinario("")
	if err != nil || got != enPath {
		t.Fatalf("descubrimiento en PATH: got %q, err %v", got, err)
	}
	t.Setenv("PATH", t.TempDir())
	if _, err := LocalizarBinario(""); err == nil {
		t.Fatal("sin llama-server en PATH debería avisar")
	}
}

func TestArrancaConAliasYPuertoYContexto(t *testing.T) {
	bin, argsFile := binarioFalso(t)
	modelo := modeloFalso(t)

	s, err := Nuevo(bin, "http://127.0.0.1:18080", 8192)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Arrancar(context.Background(), Modelo{Nombre: "qwen3-14b", Ruta: modelo}); err != nil {
		t.Fatalf("arrancar: %v", err)
	}
	defer s.Detener()

	args := strings.Join(leerArgs(t, argsFile), " ")
	for _, quiero := range []string{"--model " + modelo, "--alias qwen3-14b", "--port 18080", "-c 8192"} {
		if !strings.Contains(args, quiero) {
			t.Errorf("faltan argumentos %q en %q", quiero, args)
		}
	}
	if m, ok := s.Activo(); !ok || m.Nombre != "qwen3-14b" {
		t.Fatalf("activo = %+v, ok = %v", m, ok)
	}
}

func TestArrancaConProyectorMultimodal(t *testing.T) {
	bin, argsFile := binarioFalso(t)
	modelo := modeloFalso(t)
	mmproj := filepath.Join(t.TempDir(), "mmproj-f16.gguf")
	if err := os.WriteFile(mmproj, []byte("gguf"), 0o644); err != nil {
		t.Fatal(err)
	}

	s, err := Nuevo(bin, "http://127.0.0.1:18086", 4096)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Arrancar(context.Background(), Modelo{Nombre: "vision", Ruta: modelo, MMProj: mmproj}); err != nil {
		t.Fatalf("arrancar: %v", err)
	}
	defer s.Detener()

	args := strings.Join(leerArgs(t, argsFile), " ")
	if !strings.Contains(args, "--mmproj "+mmproj) {
		t.Fatalf("faltan los argumentos del proyector en %q", args)
	}
}

func TestDetenerApagaElProceso(t *testing.T) {
	bin, argsFile := binarioFalso(t)
	s, err := Nuevo(bin, "http://127.0.0.1:18081", 4096)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Arrancar(context.Background(), Modelo{Nombre: "m", Ruta: modeloFalso(t)}); err != nil {
		t.Fatal(err)
	}
	leerArgs(t, argsFile) // confirma que arrancó

	if err := s.Detener(); err != nil {
		t.Fatalf("detener: %v", err)
	}
	if _, ok := s.Activo(); ok {
		t.Fatal("tras detener no debería haber servidor activo")
	}
	if err := s.Detener(); err != nil {
		t.Fatalf("detener dos veces no debería fallar: %v", err)
	}
}

func TestReiniciarCambiaDeModelo(t *testing.T) {
	bin, argsFile := binarioFalso(t)
	s, err := Nuevo(bin, "http://127.0.0.1:18082", 4096)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Arrancar(context.Background(), Modelo{Nombre: "viejo", Ruta: modeloFalso(t)}); err != nil {
		t.Fatal(err)
	}
	leerArgs(t, argsFile)

	if err := os.Remove(argsFile); err != nil {
		t.Fatal(err)
	}
	nuevo := modeloFalso(t)
	s.espera = 300 * time.Millisecond
	// El binario falso no sirve HTTP, así que el reinicio acaba avisando que no
	// respondió; lo que importa aquí es que arrancó con el modelo nuevo.
	_, _ = s.Reiniciar(context.Background(), Modelo{Nombre: "nuevo", Ruta: nuevo})
	defer s.Detener()

	args := strings.Join(leerArgs(t, argsFile), " ")
	if !strings.Contains(args, "--alias nuevo") || !strings.Contains(args, "--model "+nuevo) {
		t.Fatalf("no arrancó con el modelo nuevo: %q", args)
	}
}

func TestDisponibleYSirveModelo(t *testing.T) {
	servidor := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/models" {
			_, _ = w.Write([]byte(`{"data":[{"id":"qwen3-14b"}]}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer servidor.Close()

	s, err := Nuevo("", servidor.URL, 4096)
	if err != nil {
		t.Fatal(err)
	}
	if !s.Disponible(context.Background()) {
		t.Fatal("el servidor de prueba debería responder")
	}
	if !s.sirveModelo(context.Background(), "qwen3-14b") {
		t.Fatal("debería reconocer el modelo servido")
	}
	if s.sirveModelo(context.Background(), "otro") {
		t.Fatal("no debería reconocer un modelo ausente")
	}
}

func TestAsegurarReutilizaServidorCompatible(t *testing.T) {
	servidor := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/models" {
			_, _ = w.Write([]byte(`{"data":[{"id":"qwen3-14b"}]}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer servidor.Close()

	bin, argsFile := binarioFalso(t)
	s, err := Nuevo(bin, servidor.URL, 4096)
	if err != nil {
		t.Fatal(err)
	}
	avisos, err := s.Asegurar(context.Background(), Modelo{Nombre: "qwen3-14b", Ruta: modeloFalso(t)})
	if err != nil {
		t.Fatalf("asegurar: %v", err)
	}
	if len(avisos) != 0 {
		t.Fatalf("no esperaba avisos: %v", avisos)
	}
	if _, err := os.Stat(argsFile); !os.IsNotExist(err) {
		t.Fatal("no debería arrancar un segundo servidor")
	}
}

func TestAsegurarDetectaPuertoOcupado(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	puerto := ln.Addr().(*net.TCPAddr).Port

	bin, _ := binarioFalso(t)
	s, err := Nuevo(bin, fmt.Sprintf("http://127.0.0.1:%d", puerto), 4096)
	if err != nil {
		t.Fatal(err)
	}
	avisos, err := s.Asegurar(context.Background(), Modelo{Nombre: "m", Ruta: modeloFalso(t)})
	if err != nil {
		t.Fatalf("asegurar no debería dar error: %v", err)
	}
	if len(avisos) == 0 || !strings.Contains(strings.Join(avisos, " "), "ocupado") {
		t.Fatalf("esperaba un aviso de puerto ocupado: %v", avisos)
	}
}

func TestAsegurarAvisaSinBinarioSinModeloYModeloAusente(t *testing.T) {
	bin, _ := binarioFalso(t)

	sinBinario, err := Nuevo(filepath.Join(t.TempDir(), "no-existe"), "http://127.0.0.1:18083", 4096)
	if err != nil {
		t.Fatal(err)
	}
	if avisos, err := sinBinario.Asegurar(context.Background(), Modelo{Nombre: "m", Ruta: modeloFalso(t)}); err != nil || len(avisos) == 0 {
		t.Fatalf("sin binario: avisos=%v err=%v", avisos, err)
	}

	sinModelo, err := Nuevo(bin, "http://127.0.0.1:18084", 4096)
	if err != nil {
		t.Fatal(err)
	}
	if avisos, err := sinModelo.Asegurar(context.Background(), Modelo{}); err != nil || len(avisos) == 0 {
		t.Fatalf("sin modelo: avisos=%v err=%v", avisos, err)
	}

	modeloAusente, err := Nuevo(bin, "http://127.0.0.1:18085", 4096)
	if err != nil {
		t.Fatal(err)
	}
	avisos, err := modeloAusente.Asegurar(context.Background(), Modelo{Nombre: "m", Ruta: filepath.Join(t.TempDir(), "no.gguf")})
	if err != nil || len(avisos) == 0 || !strings.Contains(strings.Join(avisos, " "), "disco") {
		t.Fatalf("modelo ausente: avisos=%v err=%v", avisos, err)
	}
}

func TestCabeEnMemoriaSinModelo(t *testing.T) {
	if cabe, conocido := CabeEnMemoria(0); !cabe || conocido {
		t.Fatalf("sin tamaño no se afirma nada: cabe=%v conocido=%v", cabe, conocido)
	}
	if total, ok := MemoriaTotal(); ok && total > 0 {
		if cabe, conocido := CabeEnMemoria(total * 4); !conocido || cabe {
			t.Fatalf("un modelo mayor que la RAM no debería caber: cabe=%v conocido=%v", cabe, conocido)
		}
	}
}
