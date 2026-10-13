package models

import (
	"os"
	"path/filepath"
	"testing"
)

// El proyector multimodal se detecta por convención en la carpeta del modelo.
func TestEntradaDetectaProyector(t *testing.T) {
	dir := t.TempDir()
	gguf := filepath.Join(dir, "m.gguf")
	if err := os.WriteFile(gguf, []byte("gguf"), 0o644); err != nil {
		t.Fatal(err)
	}
	e := Entrada{ID: "m", Ruta: gguf}
	if e.Proyector() != "" || e.TieneMMProj() {
		t.Fatalf("sin proyector no debería haber ninguno: %q", e.Proyector())
	}

	mm := filepath.Join(dir, "mmproj-f16.gguf")
	if err := os.WriteFile(mm, []byte("gguf"), 0o644); err != nil {
		t.Fatal(err)
	}
	if e.Proyector() != mm || !e.TieneMMProj() {
		t.Fatalf("Proyector() = %q, quiero %q", e.Proyector(), mm)
	}
}

// Un proyector registrado explícitamente manda sobre la convención.
func TestEntradaProyectorRegistrado(t *testing.T) {
	dir := t.TempDir()
	gguf := filepath.Join(dir, "m.gguf")
	if err := os.WriteFile(gguf, []byte("gguf"), 0o644); err != nil {
		t.Fatal(err)
	}
	otro := filepath.Join(dir, "mmproj-f16.gguf")
	if err := os.WriteFile(otro, []byte("gguf"), 0o644); err != nil {
		t.Fatal(err)
	}
	e := Entrada{ID: "m", Ruta: gguf, MMProj: "mmproj-f16.gguf"}
	if e.Proyector() != otro {
		t.Fatalf("Proyector() = %q, quiero %q", e.Proyector(), otro)
	}
}

func TestVarianteEsProyector(t *testing.T) {
	if !(Variante{Archivo: "mmproj-f16.gguf"}).EsProyector() {
		t.Fatal("mmproj-f16.gguf debería ser proyector")
	}
	if (Variante{Archivo: "modelo-q4_k_m.gguf"}).EsProyector() {
		t.Fatal("un modelo de lenguaje no es proyector")
	}
}
