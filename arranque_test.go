package main

// Test de la elección del modelo en el arranque: el último usado se recuerda y
// prevalece si sigue instalado (SPEC-OLLAMA-PERFIL).

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"localcli/internal/ollama"
)

func servidorDeTags(t *testing.T, modelos ...string) *ollama.Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tags" {
			t.Errorf("ruta inesperada %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		items := make([]string, 0, len(modelos))
		for _, m := range modelos {
			items = append(items, `{"name":"`+m+`","size":0}`)
		}
		_, _ = w.Write([]byte(`{"models":[` + strings.Join(items, ",") + `]}`))
	}))
	t.Cleanup(srv.Close)
	return ollama.NewClient(srv.URL)
}

func TestElegirModeloPrefiereElUltimoUsado(t *testing.T) {
	c := servidorDeTags(t, "llama3.2", "qwen2.5")
	got, err := elegirModelo(c, "qwen2.5")
	if err != nil {
		t.Fatal(err)
	}
	if got != "qwen2.5" {
		t.Errorf("el último modelo usado prevalece si sigue instalado: %q", got)
	}
}

func TestElegirModeloIgnoraUnPreferidoDesinstalado(t *testing.T) {
	c := servidorDeTags(t, "llama3.2")
	got, err := elegirModelo(c, "desinstalado")
	if err != nil {
		t.Fatal(err)
	}
	if got != "llama3.2" {
		t.Errorf("si el preferido no está, se autodetecta: %q", got)
	}
}
