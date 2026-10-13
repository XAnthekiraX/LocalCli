package llamacpp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"localcli/internal/llm"
)

func TestLeeVentanaYVision(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/props" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`{"default_generation_settings":{"n_ctx":8192},"modalities":{"vision":true}}`))
	}))
	defer srv.Close()

	p, err := Leer(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("props: %v", err)
	}
	if p.Ventana != 8192 || p.Vision != llm.Soportada {
		t.Fatalf("props = %+v", p)
	}
}

func TestSinVisionQuedaDesconocida(t *testing.T) {
	// Sin el campo `modalities`, la visión es desconocida, no "no soportada".
	p, err := Decodificar(strings.NewReader(`{"default_generation_settings":{"n_ctx":4096}}`))
	if err != nil {
		t.Fatalf("props: %v", err)
	}
	if p.Ventana != 4096 || p.Vision != llm.Desconocida {
		t.Fatalf("props = %+v", p)
	}
}

func TestServidorCaidoEsErrorClaro(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	url := srv.URL
	srv.Close()

	if _, err := Leer(context.Background(), url); err == nil || !strings.Contains(err.Error(), "no se pudo conectar") {
		t.Fatalf("error = %v", err)
	}
}
