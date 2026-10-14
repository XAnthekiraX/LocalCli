package compact

import (
	"strings"
	"testing"
)

func TestPromptInicialPorFormato(t *testing.T) {
	casos := []struct {
		nombre   string
		opciones Opciones
		contiene string
		exacto   string
	}{
		{"opencode por defecto", Opciones{}, "## Objective", ""},
		{"opencode explícito", Opciones{Formato: FormatoOpenCode}, "## Work State", ""},
		{"legacy", Opciones{Formato: FormatoLegacy}, "pocas líneas", ""},
		{"custom con plantilla", Opciones{Formato: FormatoCustom, Plantilla: "MIPROMPT"}, "", "MIPROMPT"},
		{"custom sin plantilla cae en legacy", Opciones{Formato: FormatoCustom}, "pocas líneas", ""},
		{"formato desconocido cae en opencode", Opciones{Formato: "raro"}, "## Next Move", ""},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			p := c.opciones.PromptInicial()
			if c.exacto != "" && p != c.exacto {
				t.Fatalf("prompt = %q, quiero %q", p, c.exacto)
			}
			if c.contiene != "" && !strings.Contains(p, c.contiene) {
				t.Fatalf("el prompt no contiene %q:\n%s", c.contiene, p)
			}
		})
	}
}

func TestPromptIncrementalIncluyeElAnterior(t *testing.T) {
	op := Opciones{Formato: FormatoOpenCode}
	p := op.PromptIncremental("## Objective\n- resumir")
	if !strings.Contains(p, "## Objective\n- resumir") {
		t.Fatalf("el incremental no incluye el resumen previo:\n%s", p)
	}
	if !strings.Contains(p, "Work State") {
		t.Fatalf("el incremental no pide el formato estructurado:\n%s", p)
	}
}

func TestPromptIncrementalSinAnteriorEsElInicial(t *testing.T) {
	op := Opciones{}
	if op.PromptIncremental("   ") != op.PromptInicial() {
		t.Fatal("un incremental sin resumen previo debe equivaler al inicial")
	}
}
