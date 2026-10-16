package store

import "testing"

func TestGuardarYListarLineasDelHilo(t *testing.T) {
	db := nuevaBase(t)
	proyecto := proyectoDePrueba(t, db)
	s, err := CrearSesion(db, proyecto, "s", nil)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := CrearChatEvento(db, proyecto, s.ID, ChatEventoProceso, "[Sub Proceso] Diagnosticar", nil); err != nil {
		t.Fatal(err)
	}
	ms := 400
	if _, err := CrearChatEvento(db, proyecto, s.ID, ChatEventoHerramienta, "✓ LEER [AGENTS.md] · 93 líneas", &ms); err != nil {
		t.Fatal(err)
	}

	eventos, err := ListarChatEventos(db, proyecto, s.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(eventos) != 2 {
		t.Fatalf("líneas guardadas = %d", len(eventos))
	}
	if eventos[0].Tipo != ChatEventoProceso || eventos[0].DurationMs != nil {
		t.Fatalf("la etapa de flujo no debería llevar duración: %+v", eventos[0])
	}
	if eventos[1].Tipo != ChatEventoHerramienta || eventos[1].DurationMs == nil || *eventos[1].DurationMs != 400 {
		t.Fatalf("la línea de herramienta debería conservar su duración: %+v", eventos[1])
	}
}

func TestCrearLineaRechazaTipoDesconocido(t *testing.T) {
	db := nuevaBase(t)
	proyecto := proyectoDePrueba(t, db)
	s, err := CrearSesion(db, proyecto, "s", nil)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := CrearChatEvento(db, proyecto, s.ID, "ocurrencia", "x", nil); err == nil {
		t.Fatal("un tipo fuera de proceso/herramienta debería rechazarse")
	}
}
