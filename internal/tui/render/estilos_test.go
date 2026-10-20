package render

import (
	"fmt"
	"testing"
)

// El diff no hereda los tonos del tema: con el agente y el error en rojo, lo
// añadido sigue verde y lo quitado sigue rojo.
func TestElDiffNoHeredaElRojoDelAgente(t *testing.T) {
	t.Cleanup(func() { _ = AplicarTokens(nil) })

	if err := AplicarTokens(map[string]string{
		"color_agente":  "#F05B6E",
		"color_incapaz": "#F7768E",
	}); err != nil {
		t.Fatal(err)
	}

	mas := fmt.Sprint(EstiloDiferenciaMas.GetForeground())
	menos := fmt.Sprint(EstiloDiferenciaMenos.GetForeground())
	if mas != "#4CEE75" {
		t.Fatalf("lo añadido debería seguir verde: %q", mas)
	}
	if menos != "#FF5F5F" {
		t.Fatalf("lo quitado debería ser rojo: %q", menos)
	}
	if mas == menos {
		t.Fatal("las dos caras del diff no pueden compartir color")
	}
}

// Los dos tonos del diff son configurables como cualquier otro token.
func TestLosTonosDelDiffSeConfiguran(t *testing.T) {
	t.Cleanup(func() { _ = AplicarTokens(nil) })

	if err := AplicarTokens(map[string]string{
		"color_diferencia_mas":   "#00FF00",
		"color_diferencia_menos": "#AA0000",
	}); err != nil {
		t.Fatal(err)
	}

	if got := fmt.Sprint(EstiloDiferenciaMas.GetForeground()); got != "#00FF00" {
		t.Fatalf("el verde configurado no se aplicó: %q", got)
	}
	if got := fmt.Sprint(EstiloDiferenciaMenos.GetForeground()); got != "#AA0000" {
		t.Fatalf("el rojo configurado no se aplicó: %q", got)
	}
}
