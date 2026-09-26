package tui

// Tests de keys (T-F001-05, completada con la persistencia de T-F010-04):
// ida y vuelta del mapa, archivo ausente, rechazo de duplicados al guardar y
// de acciones desconocidas al cargar.

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	atajos := AtajosPorDefecto()
	// El usuario mueve el panel a ctrl+p y asigna pausar a ctrl+o.
	for i := range atajos {
		if atajos[i].Accion == AccionPanel {
			atajos[i].Tecla = "ctrl+p"
		}
	}
	atajos = append(atajos, Atajo{Tecla: "ctrl+o", Accion: AccionPausar, Descripcion: "pausar la cola en curso"})

	if err := GuardarKeysEn(dir, atajos); err != nil {
		t.Fatalf("guardar: %v", err)
	}
	cargados, err := CargarKeysDesde(dir)
	if err != nil {
		t.Fatalf("cargar: %v", err)
	}
	if accion, ok := AccionDe(cargados, "ctrl+p"); !ok || accion != AccionPanel {
		t.Errorf("ctrl+p debe ser panel tras la ida y vuelta: %d, %v", accion, ok)
	}
	if accion, ok := AccionDe(cargados, "ctrl+o"); !ok || accion != AccionPausar {
		t.Errorf("ctrl+o debe ser pausar tras la ida y vuelta: %d, %v", accion, ok)
	}
	// La reasignación no cambia reglas de permiso, solo la invocación: el
	// resto del mapa queda intacto.
	if accion, ok := AccionDe(cargados, "ctrl+s"); !ok || accion != AccionSelector {
		t.Errorf("el resto del mapa se conserva: %d, %v", accion, ok)
	}
}

func TestCargarSinArchivoDevuelveLosDeFabrica(t *testing.T) {
	cargados, err := CargarKeysDesde(t.TempDir())
	if err != nil {
		t.Fatalf("sin archivo no hay error: %v", err)
	}
	fabrica := AtajosPorDefecto()
	if len(cargados) != len(fabrica) {
		t.Fatalf("devuelve el mapa de fábrica: %d contra %d", len(cargados), len(fabrica))
	}
}

func TestGuardarRechazaElDuplicadoYNoEscribe(t *testing.T) {
	dir := t.TempDir()
	malo := []Atajo{
		{Tecla: "ctrl+d", Accion: AccionPanel},
		{Tecla: "ctrl+d", Accion: AccionSalir},
	}
	if err := GuardarKeysEn(dir, malo); err == nil {
		t.Fatal("dos acciones con la misma tecla se rechazan al guardar")
	}
	if _, err := os.Stat(filepath.Join(dir, "keys.json")); !os.IsNotExist(err) {
		t.Error("un mapa inválido no deja nada escrito")
	}
}

func TestCargarRechazaAccionesDesconocidas(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "keys.json"), []byte(`{"atajos":{"inventada":"ctrl+z"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := CargarKeysDesde(dir); err == nil {
		t.Error("una acción desconocida invalida el mapa entero, no se adivina")
	}
}
