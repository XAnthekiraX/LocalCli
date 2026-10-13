package llama

import "testing"

// parDe busca el par de bandera y valor en los argumentos de arranque.
func parDe(args []string, bandera string) (string, bool) {
	for i := 0; i+1 < len(args); i++ {
		if args[i] == bandera {
			return args[i+1], true
		}
	}
	return "", false
}

func TestArgsDeArranqueUsaElContextoDelModelo(t *testing.T) {
	s, err := Nuevo("llama-server", "http://localhost:8080", 16384)
	if err != nil {
		t.Fatalf("nuevo: %v", err)
	}

	if v, _ := parDe(s.argsDeArranque(Modelo{Nombre: "m", Ruta: "/m.gguf", Contexto: 32768}), "-c"); v != "32768" {
		t.Fatalf("-c = %q, quiero 32768 (el contexto del modelo)", v)
	}

	if v, _ := parDe(s.argsDeArranque(Modelo{Nombre: "m", Ruta: "/m.gguf"}), "-c"); v != "16384" {
		t.Fatalf("-c = %q, quiero 16384 (el del servidor)", v)
	}
}
