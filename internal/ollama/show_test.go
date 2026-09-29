package ollama

// Test de las capacidades del modelo (/api/show). LocalCli no usa
// function-calling nativo —las herramientas van como texto—, así que esta
// capacidad solo alimenta el aviso al usuario (SPEC-OLLAMA-PERFIL).

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestCapacidadesLeeLaFichaDelModelo(t *testing.T) {
	var recibido struct {
		Model string `json:"model"`
	}
	c := nuevoServidorOllama(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/show" {
			t.Errorf("ruta inesperada %s", r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&recibido)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"capabilities":["completion","tools","vision"]}`))
	})

	caps, err := c.Capacidades(context.Background(), "llama3.2")
	if err != nil {
		t.Fatal(err)
	}
	if recibido.Model != "llama3.2" {
		t.Errorf("se pregunta por el modelo pedido: %q", recibido.Model)
	}
	if !PuedeUsarHerramientas(caps) {
		t.Errorf("tools está entre las capacidades: %v", caps)
	}
	if PuedeUsarHerramientas([]string{"completion", "vision"}) {
		t.Error("sin tools no puede usar herramientas")
	}
	if PuedeUsarHerramientas(nil) {
		t.Error("sin capacidades no puede usar herramientas")
	}
}

// PuedeVer reconoce la capacidad «vision», espejo de PuedeUsarHerramientas.
func TestPuedeVerDetectaLaCapacidadDeVision(t *testing.T) {
	if !PuedeVer([]string{"completion", "tools", "vision"}) {
		t.Error("vision está entre las capacidades")
	}
	if PuedeVer([]string{"completion", "tools"}) {
		t.Error("sin vision no puede interpretar imágenes")
	}
	if PuedeVer(nil) {
		t.Error("sin capacidades no puede interpretar imágenes")
	}
}

// PuedePensar reconoce la capacidad «thinking»: solo a estos modelos se les
// puede mandar `think` (SPEC-OLLAMA-PERFIL).
func TestPuedePensarDetectaLaCapacidadDeRazonar(t *testing.T) {
	if !PuedePensar([]string{"completion", "tools", "thinking"}) {
		t.Error("thinking está entre las capacidades")
	}
	if PuedePensar([]string{"completion", "tools", "vision"}) {
		t.Error("sin thinking el modelo no razona")
	}
	if PuedePensar(nil) {
		t.Error("sin capacidades no se le manda `think`")
	}
}

// Ficha que no trae `capabilities` (Ollama antiguo): la visión se deduce de las
// familias, para no dejar sin reconocer a un modelo multimodal.
func TestCapacidadesDeduceLaVisionDeLasFamilias(t *testing.T) {
	c := nuevoServidorOllama(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"details":{"family":"llava","families":["llava","clip"]}}`))
	})
	caps, err := c.Capacidades(context.Background(), "llava")
	if err != nil {
		t.Fatal(err)
	}
	if !PuedeVer(caps) {
		t.Errorf("un modelo de familia multimodal tiene visión: %v", caps)
	}
}

// Una familia solo-texto sí dice algo: no tiene visión, y eso se sabe.
func TestCapacidadesDeUnaFamiliaSinVision(t *testing.T) {
	c := nuevoServidorOllama(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"details":{"family":"llama","families":["llama"]}}`))
	})
	caps, err := c.Capacidades(context.Background(), "llama3")
	if err != nil {
		t.Fatalf("una familia conocida no es un «no se sabe»: %v", err)
	}
	if PuedeVer(caps) {
		t.Errorf("una familia solo-texto no tiene visión: %v", caps)
	}
}

// Sin capacidades ni familias no se puede afirmar nada: es un error, no un «no
// puede». Quien avisa necesita distinguir «no tiene» de «no se sabe».
func TestCapacidadesSinDatosEsUnError(t *testing.T) {
	c := nuevoServidorOllama(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"details":{}}`))
	})
	if _, err := c.Capacidades(context.Background(), "m"); err == nil {
		t.Error("sin capacidades ni familias no se puede afirmar nada")
	}
}
