package tui

// Tests de las preferencias del usuario (config.json): se recuerdan entre
// ejecuciones, un archivo ausente o corrupto no rompe el arranque y el agente
// se normaliza (SPEC-OLLAMA-PERFIL, FRONTEND.md §3).

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLasPreferenciasSeRecuerdanYSeToleran(t *testing.T) {
	dir := t.TempDir()

	// Sin archivo: preferencias vacías, sin error.
	if p, err := CargarPreferenciasDesde(dir); err != nil || p.Modelo != "" || p.Agente != "" {
		t.Fatalf("sin config.json, vacío y sin error: %+v, %v", p, err)
	}

	// Round-trip: lo guardado se vuelve a leer tal cual.
	if err := GuardarPreferenciasEn(dir, Preferencias{Modelo: "llama3.2", Agente: AgenteBuild}); err != nil {
		t.Fatal(err)
	}
	p, err := CargarPreferenciasDesde(dir)
	if err != nil || p.Modelo != "llama3.2" || p.Agente != AgenteBuild {
		t.Fatalf("round-trip de preferencias: %+v, %v", p, err)
	}

	// Un archivo corrupto no rompe el arranque: se ignora.
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte("{no es json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if p, err := CargarPreferenciasDesde(dir); err != nil || p.Modelo != "" || p.Agente != "" {
		t.Errorf("un config.json corrupto no rompe: %+v, %v", p, err)
	}
}

func TestElAgenteRecordadoSeNormaliza(t *testing.T) {
	casos := map[string]string{
		AgenteBuild: AgenteBuild,
		AgentePlan:  AgentePlan,
		"":          AgentePlan,
		"otro":      AgentePlan,
	}
	for entrada, quiere := range casos {
		if got := ValidarAgente(entrada); got != quiere {
			t.Errorf("ValidarAgente(%q) = %q, quiero %q", entrada, got, quiere)
		}
	}
}
