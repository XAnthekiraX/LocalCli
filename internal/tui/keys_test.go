package tui

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestDefaultKeyMapTieneSieteAtajos(t *testing.T) {
	km := DefaultKeyMap()
	if len(km) != 7 {
		t.Fatalf("se esperaban 7 atajos por defecto, se obtuvieron %d", len(km))
	}
	for _, a := range []Accion{
		AccionPanel, AccionSelector, AccionRazonamiento,
		AccionAprobaciones, AccionCancelar, AccionAyuda, AccionSalir,
	} {
		if _, ok := km[a]; !ok {
			t.Errorf("falta el atajo por defecto de %q", a)
		}
	}
	if got := km[AccionPanel].String(); got != "ctrl+d" {
		t.Errorf("panel por defecto: wanted ctrl+d, got %s", got)
	}
}

func TestMatchesDetectaAtajo(t *testing.T) {
	km := DefaultKeyMap()
	if !km.Matches(AccionPanel, tea.KeyMsg{Type: tea.KeyCtrlD}) {
		t.Error("ctrl+d debe coincidir con la acción panel")
	}
	if !km.Matches(AccionAyuda, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}}) {
		t.Error("? debe coincidir con la acción ayuda")
	}
	if km.Matches(AccionPanel, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}}) {
		t.Error("d sin ctrl no debe coincidir con panel")
	}
	// Reasignado: el cambio se refleja sin reiniciar.
	km[AccionSalir] = Binding{Key: "x"}
	if !km.Matches(AccionSalir, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}}) {
		t.Error("tras reasignar, x debe coincidir con salir")
	}
	if km.Matches(AccionSalir, tea.KeyMsg{Type: tea.KeyCtrlQ}) {
		t.Error("tras reasignar, ctrl+q ya no debe coincidir con salir")
	}
}

func TestFindDuplicateRechazaRepetido(t *testing.T) {
	km := DefaultKeyMap()
	a, dup := km.FindDuplicate(Binding{Ctrl: true, Key: "d"})
	if !dup || a != AccionPanel {
		t.Errorf("ctrl+d duplicado: wanted (panel,true), got (%s,%v)", a, dup)
	}
	if _, dup := km.FindDuplicate(Binding{Ctrl: true, Key: "z"}); dup {
		t.Error("ctrl+z libre no debe reportar duplicado")
	}
	// Guardar un mapa con dos acciones al mismo binding debe fallar.
	km[AccionAyuda] = Binding{Ctrl: true, Key: "q"}
	if err := SaveKeyMap(km); err == nil {
		t.Fatal("SaveKeyMap debe rechazar asignaciones duplicadas")
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	km := DefaultKeyMap()
	km[AccionAyuda] = Binding{Key: "F1"}
	if err := SaveKeyMap(km); err != nil {
		t.Fatalf("SaveKeyMap: %v", err)
	}
	path := filepath.Join(home, ".config", "localcli", "keys.json")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("keys.json no fue escrito: %v", err)
	}
	got, err := LoadKeyMap()
	if err != nil {
		t.Fatalf("LoadKeyMap: %v", err)
	}
	if !sameBinding(got[AccionAyuda], km[AccionAyuda]) {
		t.Errorf("ida y vuelta perdió la reasignación: got %+v", got[AccionAyuda])
	}
	if !sameBinding(got[AccionPanel], DefaultKeyMap()[AccionPanel]) {
		t.Error("los valores por defecto deben conservarse tras el round trip")
	}
}

func TestLoadKeyMapAusenteDevuelveDefectos(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	km, err := LoadKeyMap()
	if err != nil {
		t.Fatalf("archivo ausente no debe ser error: %v", err)
	}
	if len(km) != 7 {
		t.Errorf("wanted 7 defectos, got %d", len(km))
	}
}
