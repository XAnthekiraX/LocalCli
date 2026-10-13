package models

import "testing"

func TestNombreDeUsaLaCarpetaNoElArchivo(t *testing.T) {
	casos := map[string]string{
		"/home/u/.config/localcli/models/qwen3-14b/qwen3-14b-q4_k_m.gguf": "qwen3-14b",
		"models/Qwen3.8/m.gguf":                                          "Qwen3.8",
		"models/llama-3-8b/modelo.gguf":                                  "llama-3-8b",
	}
	for ruta, quiero := range casos {
		if got := NombreDe(ruta); got != quiero {
			t.Errorf("NombreDe(%q) = %q, quiero %q", ruta, got, quiero)
		}
	}
}
