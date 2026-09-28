package main

// Test de la elección del modelo en el arranque: el último usado se recuerda y
// prevalece si sigue instalado (SPEC-OLLAMA-PERFIL).

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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

// agenteBase carga los agentes base y cualquier agente propio de
// `ai/agents/*.json`; un archivo roto se ignora sin tumbar el arranque, y los
// base siguen disponibles aunque falte su JSON.
func TestAgenteBaseCargaAgentesPropios(t *testing.T) {
	raiz := t.TempDir()
	dir := filepath.Join(raiz, "ai", "agents")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	escribir := func(nombre, contenido string) {
		if err := os.WriteFile(filepath.Join(dir, nombre), []byte(contenido), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	escribir("plan.json", `{"nombre":"plan","prompt":"soy plan","permisos":[{"accion":"leer","efecto":"permitir"}]}`)
	escribir("revisor.json", `{"nombre":"revisor","prompt":"soy revisor","permisos":[{"accion":"leer","efecto":"permitir"}]}`)
	escribir("roto.json", `{no es json`)

	agentes := agenteBase(raiz)
	if _, ok := agentes["plan"]; !ok {
		t.Error("plan debe estar")
	}
	if _, ok := agentes["revisor"]; !ok {
		t.Error("el agente propio de ai/agents se carga")
	}
	if _, ok := agentes["roto"]; ok {
		t.Error("un JSON de agente roto se ignora")
	}
	// build no tiene archivo: queda el de respaldo, para que los flujos
	// oficiales sigan teniendo a quién referirse.
	if _, ok := agentes["build"]; !ok {
		t.Error("build debe existir aunque falte su JSON")
	}
}
