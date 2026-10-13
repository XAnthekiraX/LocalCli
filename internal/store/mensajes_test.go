package store

import "testing"

func TestTokensNuloDistintoDeCero(t *testing.T) {
	db := nuevaBase(t)
	proyecto := proyectoDePrueba(t, db)
	s, err := CrearSesion(db, proyecto, "s", nil)
	if err != nil {
		t.Fatal(err)
	}

	cero := 0
	if _, err := CrearMensaje(db, proyecto, s.ID, "agent", "respuesta", &cero, nil, nil); err != nil {
		t.Fatal(err)
	}

	mensajes, err := ListarMensajes(db, proyecto, s.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(mensajes) != 1 {
		t.Fatalf("mensajes = %d", len(mensajes))
	}
	if mensajes[0].InputTokens == nil || *mensajes[0].InputTokens != 0 {
		t.Fatal("un 0 guardado debería leerse como 0, no como NULL")
	}
	if mensajes[0].OutputTokens != nil {
		t.Fatal("un dato ausente debería leerse como nil")
	}
}

func TestHistorialConservaMensajesSinRazonamiento(t *testing.T) {
	db := nuevaBase(t)
	proyecto := proyectoDePrueba(t, db)
	s, err := CrearSesion(db, proyecto, "s", nil)
	if err != nil {
		t.Fatal(err)
	}

	usuario, err := CrearMensaje(db, proyecto, s.ID, "user", "hola", nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	agente, err := CrearMensaje(db, proyecto, s.ID, "agent", "qué tal", nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := GuardarRazonamiento(db, agente.ID, "pensando"); err != nil {
		t.Fatal(err)
	}

	historial, err := Historial(db, proyecto, s.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(historial) != 2 {
		t.Fatalf("el historial no debería perder mensajes: %d", len(historial))
	}
	if historial[0].ID != usuario.ID || historial[0].Razonamiento != nil {
		t.Fatalf("el mensaje de usuario no lleva razonamiento: %+v", historial[0])
	}
	if historial[1].ID != agente.ID || historial[1].Razonamiento == nil || *historial[1].Razonamiento != "pensando" {
		t.Fatalf("el razonamiento del agente no se leyó: %+v", historial[1])
	}
}

func TestHistorialAislaPorProyecto(t *testing.T) {
	db := nuevaBase(t)
	uno := proyectoDePrueba(t, db)
	dos := proyectoDePrueba(t, db)
	s, err := CrearSesion(db, uno, "s", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Historial(db, dos, s.ID); err == nil {
		t.Fatal("una sesión de otro proyecto no debería leerse")
	}
}
