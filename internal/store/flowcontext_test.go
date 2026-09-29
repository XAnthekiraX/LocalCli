package store

import (
	"testing"
)

// TestBloqueDeContextoGuardaReemplazaYLimpia — T-B0XX: el bloque de contexto de
// un flujo guarda una aportación por etapa, la reemplaza al volver a correr el
// flujo y se vacía con Limpiar. Es el estado que usa la composición final.
func TestBloqueDeContextoGuardaReemplazaYLimpia(t *testing.T) {
	db := abrirBaseTemporal(t)
	ses, err := CrearSesion(db, "prueba", "")
	if err != nil {
		t.Fatalf("CrearSesion: %v", err)
	}
	sid := ses.ID

	bloques := NuevosBloques(db)
	if err := bloques.Guardar(sid, &FlowContext{
		Flow: "resolver", Stage: "diagnosticar", StageName: "Diagnosticar", Position: 1, Content: "causa: X",
	}); err != nil {
		t.Fatalf("Guardar: %v", err)
	}
	if err := bloques.Guardar(sid, &FlowContext{
		Flow: "resolver", Stage: "diseno", StageName: "Diseñar la solución", Position: 2, Content: "cambiar Y",
	}); err != nil {
		t.Fatalf("Guardar: %v", err)
	}

	got, err := bloques.Leer(sid, "resolver")
	if err != nil {
		t.Fatalf("Leer: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("entradas = %d, quiero 2", len(got))
	}
	if got[0].Stage != "diagnosticar" || got[1].Stage != "diseno" {
		t.Errorf("orden = [%s, %s], quiero el de position", got[0].Stage, got[1].Stage)
	}

	// Re-ejecutar el flujo reemplaza la aportación de la misma etapa, no la duplica.
	if err := bloques.Guardar(sid, &FlowContext{
		Flow: "resolver", Stage: "diagnosticar", StageName: "Diagnosticar", Position: 1, Content: "causa: Z",
	}); err != nil {
		t.Fatalf("Guardar (reemplazo): %v", err)
	}
	got, err = bloques.Leer(sid, "resolver")
	if err != nil {
		t.Fatalf("Leer: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("tras reemplazar, entradas = %d, quiero 2", len(got))
	}
	if got[0].Content != "causa: Z" {
		t.Errorf("contenido = %q, quiero el reemplazado", got[0].Content)
	}

	// Limpiar deja el bloque vacío para la ejecución siguiente.
	if err := bloques.Limpiar(sid, "resolver"); err != nil {
		t.Fatalf("Limpiar: %v", err)
	}
	got, err = bloques.Leer(sid, "resolver")
	if err != nil {
		t.Fatalf("Leer tras limpiar: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("tras limpiar, entradas = %d, quiero 0", len(got))
	}
}

// TestBloqueDeContextoAisladoPorSesion — el bloque está acotado a la sesión: no
// se mezclan las aportaciones de dos sesiones.
func TestBloqueDeContextoAisladoPorSesion(t *testing.T) {
	db := abrirBaseTemporal(t)
	s1, err := CrearSesion(db, "s1", "")
	if err != nil {
		t.Fatal(err)
	}
	s2, err := CrearSesion(db, "s2", "")
	if err != nil {
		t.Fatal(err)
	}
	bloques := NuevosBloques(db)
	if err := bloques.Guardar(s1.ID, &FlowContext{Flow: "resolver", Stage: "a", StageName: "A", Position: 0, Content: "de s1"}); err != nil {
		t.Fatal(err)
	}
	if err := bloques.Guardar(s2.ID, &FlowContext{Flow: "resolver", Stage: "a", StageName: "A", Position: 0, Content: "de s2"}); err != nil {
		t.Fatal(err)
	}
	got, err := bloques.Leer(s1.ID, "resolver")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Content != "de s1" {
		t.Errorf("el bloque de s1 no debe ver el de s2: %+v", got)
	}
}
