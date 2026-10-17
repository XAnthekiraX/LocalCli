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
	if _, err := CrearChatEvento(db, proyecto, s.ID, ChatEventoRespuesta, "la respuesta completa de la etapa", nil); err != nil {
		t.Fatalf("la respuesta de una etapa con `respuesta_en_chat` debería guardarse: %v", err)
	}

	eventos, err := ListarChatEventos(db, proyecto, s.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(eventos) != 3 {
		t.Fatalf("líneas guardadas = %d", len(eventos))
	}
	if eventos[0].Tipo != ChatEventoProceso || eventos[0].DurationMs != nil {
		t.Fatalf("la etapa de flujo no debería llevar duración: %+v", eventos[0])
	}
	if eventos[1].Tipo != ChatEventoHerramienta || eventos[1].DurationMs == nil || *eventos[1].DurationMs != 400 {
		t.Fatalf("la línea de herramienta debería conservar su duración: %+v", eventos[1])
	}
	if eventos[2].Tipo != ChatEventoRespuesta || eventos[2].Content != "la respuesta completa de la etapa" || eventos[2].DurationMs != nil {
		t.Fatalf("la respuesta se guardó mal: %+v", eventos[2])
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
		t.Fatal("un tipo fuera de proceso/herramienta/respuesta debería rechazarse")
	}
}
