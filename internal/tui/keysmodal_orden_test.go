package tui

// Test de la organización del modal de atajos (keysmodal.go).

import (
	"strings"
	"testing"
)

func TestElModalDeAtajosAgrupaPorCategorías(t *testing.T) {
	km := &KeysModal{}
	km.AbrirAtajos(KeymapPorDefecto().Entradas(), LíderPorDefecto)
	v := sinEstilo(km.Render(100, 40))
	for _, grupo := range []string{"General", "Chat", "Vista", "Modales", "Aprobaciones", "Entrada"} {
		if !strings.Contains(v, grupo) {
			t.Errorf("falta el grupo %q:\n%s", grupo, v)
		}
	}
	if !strings.Contains(v, "subir por el historial del chat") || !strings.Contains(v, "salir") {
		t.Errorf("las filas siguen listándose:\n%s", v)
	}
}

func TestLasFilasDeAtajoAlineanLaDescripción(t *testing.T) {
	km := &KeysModal{}
	km.lider = LíderPorDefecto
	a1, _ := NuevoAtajo(LíderPorDefecto, AccionSalir, "salir", "ctrl+c")
	a2, _ := NuevoAtajo(LíderPorDefecto, AccionModalModelos, "modal de modelos", "<leader>m")
	f1 := km.filaDeAtajo(a1, 12)
	f2 := km.filaDeAtajo(a2, 12)
	if strings.Index(f1, "salir") != strings.Index(f2, "modal de modelos") {
		t.Errorf("la descripción empieza en columnas distintas:\n%q\n%q", f1, f2)
	}
}
