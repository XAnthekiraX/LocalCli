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

// El bloque visual viaja con la línea: se guarda con su tipo y vuelve al leerla,
// para que el hilo recuperado se pinte igual que en vivo.
func TestElBloqueVisualSeGuardaYSeRecupera(t *testing.T) {
	db := nuevaBase(t)
	proyecto := proyectoDePrueba(t, db)
	s, err := CrearSesion(db, proyecto, "s", nil)
	if err != nil {
		t.Fatal(err)
	}

	bloque := Bloque{
		Texto: "@@ -1,2 +1,2 @@\n uno\n-dos\n+dos!",
		Tipo:  BloqueDiferencia,
	}
	if _, err := CrearChatEventoConBloque(db, proyecto, s.ID, ChatEventoHerramienta,
		"✓ EDITAR [doc.md] · 1 reemplazo", nil, bloque); err != nil {
		t.Fatal(err)
	}
	// Una línea sin bloque guarda NULL, no una cadena vacía.
	if _, err := CrearChatEvento(db, proyecto, s.ID, ChatEventoProceso, "[Sub Proceso] X", nil); err != nil {
		t.Fatal(err)
	}

	eventos, err := ListarChatEventos(db, proyecto, s.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(eventos) != 2 {
		t.Fatalf("líneas guardadas = %d", len(eventos))
	}
	if eventos[0].Bloque != bloque.Texto || eventos[0].BloqueTipo != BloqueDiferencia {
		t.Fatalf("el bloque no volvió entero: %+v", eventos[0])
	}
	if eventos[1].Bloque != "" || eventos[1].BloqueTipo != "" {
		t.Fatalf("una línea sin bloque no debería llevar uno: %+v", eventos[1])
	}
}

func TestElTipoDeBloqueFueraDelVocabularioSeRechaza(t *testing.T) {
	db := nuevaBase(t)
	proyecto := proyectoDePrueba(t, db)
	s, err := CrearSesion(db, proyecto, "s", nil)
	if err != nil {
		t.Fatal(err)
	}

	_, err = CrearChatEventoConBloque(db, proyecto, s.ID, ChatEventoHerramienta, "x", nil,
		Bloque{Texto: "x", Tipo: "otro"})
	if err == nil {
		t.Fatal("un bloque_tipo fuera de diagrama/diferencia debería rechazarse")
	}
}

// TestGuardarYListarEjecucionesHerramienta: la línea de una herramienta guarda la
// llamada y su salida; solo esas filas entran al historial del modelo. Las líneas
// de proceso y las herramientas antiguas sin `call_id` no se reenvían.
func TestGuardarYListarEjecucionesHerramienta(t *testing.T) {
	db := nuevaBase(t)
	proyecto := proyectoDePrueba(t, db)
	s, err := CrearSesion(db, proyecto, "s", nil)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := CrearChatEvento(db, proyecto, s.ID, ChatEventoProceso, "[Sub Proceso] X", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := CrearChatEvento(db, proyecto, s.ID, ChatEventoHerramienta, "✓ LEER [viejo.go]", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := CrearChatEventoTool(db, proyecto, s.ID, "✓ LEER [ir.go] · 2 líneas", nil, EjecucionHerramienta{
		CallID: "call-1", Herramienta: "leer_archivo", Argumentos: `{"ruta":"ir.go"}`,
		Salida: "package diagram\n", OK: true,
	}); err != nil {
		t.Fatal(err)
	}

	ejecuciones, err := ListarEjecucionesHerramienta(db, proyecto, s.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(ejecuciones) != 1 {
		t.Fatalf("ejecuciones = %d, quiero 1: %+v", len(ejecuciones), ejecuciones)
	}
	e := ejecuciones[0].Ejecucion
	if e == nil || e.CallID != "call-1" || e.Herramienta != "leer_archivo" ||
		e.Argumentos != `{"ruta":"ir.go"}` || e.Salida != "package diagram\n" || !e.OK || e.Truncado {
		t.Fatalf("la ejecución guardada no coincide: %+v", e)
	}

	// La línea completa también trae la ejecución para pintarla y rehacerla.
	eventos, err := ListarChatEventos(db, proyecto, s.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(eventos) != 3 || eventos[2].Ejecucion == nil || eventos[2].Ejecucion.CallID != "call-1" {
		t.Fatalf("la línea de herramienta perdió su ejecución: %+v", eventos)
	}
}
