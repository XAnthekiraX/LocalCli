package store

import "testing"

// Revertir al globo de un usuario saca de la conversación ese mensaje y todo lo
// que vino después, dejando intacto lo anterior.
func TestDeshacerDesdeCortaEnElMensajeDelUsuario(t *testing.T) {
	db, proyecto, sesion := sesionDePrueba(t)
	crear := mensajesDePrueba(t, db, proyecto, sesion)
	crear("user", "uno")
	crear("agent", "resp uno")
	segundo, err := CrearMensaje(db, proyecto, sesion, "user", "dos", nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	crear("agent", "resp dos")

	_, n, err := DeshacerDesde(db, proyecto, sesion, segundo.ID)
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("mensajes revertidos = %d, quiero 2", n)
	}
	visibles, err := ListarMensajes(db, proyecto, sesion)
	if err != nil {
		t.Fatal(err)
	}
	if len(visibles) != 2 || visibles[0].Content != "uno" || visibles[1].Content != "resp uno" {
		t.Fatalf("tras revertir quedó: %+v", visibles)
	}

	ms := 300
	if _, err := CrearChatEvento(db, proyecto, sesion, ChatEventoProceso, "[Sub Proceso] Diagnosticar", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := CrearChatEvento(db, proyecto, sesion, ChatEventoHerramienta, "✓ LEER [x] · 5 líneas", &ms); err != nil {
		t.Fatal(err)
	}
	// También crea otro turno con líneas.
	tercero, err := CrearMensaje(db, proyecto, sesion, "user", "tres", nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	crear("agent", "resp tres")
	if _, err := CrearChatEvento(db, proyecto, sesion, ChatEventoHerramienta, "✗ EJEC [cmd] · 0 s", nil); err != nil {
		t.Fatal(err)
	}

	// Revertir desde "tres" debe borrar las líneas del turno "dos" y "tres".
	if _, _, err := DeshacerDesde(db, proyecto, sesion, tercero.ID); err != nil {
		t.Fatal(err)
	}
	if ev, err := ListarChatEventos(db, proyecto, sesion); err != nil || len(ev) != 2 {
		t.Fatalf("tras revertir deberían quedar 2 líneas del turno anterior: %+v %v", ev, err)
	}
}

// Revertir solo tiene sentido desde un globo del usuario: el de un agente no
// corta nada y no toca la conversación.
func TestDeshacerDesdeIgnoraGlobosDelAgente(t *testing.T) {
	db, proyecto, sesion := sesionDePrueba(t)
	crear := mensajesDePrueba(t, db, proyecto, sesion)
	crear("user", "uno")
	agente, err := CrearMensaje(db, proyecto, sesion, "agent", "resp uno", nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	_, n, err := DeshacerDesde(db, proyecto, sesion, agente.ID)
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("revertir a un globo del agente = %d mensajes, quiero 0", n)
	}
	if visibles, _ := ListarMensajes(db, proyecto, sesion); len(visibles) != 2 {
		t.Fatalf("la conversación no debería cambiar: %+v", visibles)
	}
}

// Un mensaje que no es de la sesión no revierte nada.
func TestDeshacerDesdeMensajeAjenoNoHaceNada(t *testing.T) {
	db, proyecto, sesion := sesionDePrueba(t)
	crear := mensajesDePrueba(t, db, proyecto, sesion)
	crear("user", "uno")

	if _, n, err := DeshacerDesde(db, proyecto, sesion, "no-existe"); err != nil || n != 0 {
		t.Fatalf("mensaje ajeno = (%d, %v), quiero (0, nil)", n, err)
	}
	if visibles, _ := ListarMensajes(db, proyecto, sesion); len(visibles) != 1 {
		t.Fatalf("la conversación no debería cambiar: %+v", visibles)
	}
}

// Bifurcar copia en una sesión nueva el historial hasta el mensaje del usuario,
// incluido, con su razonamiento, y deja la sesión de origen como estaba.
func TestBifurcarSesionCopiaHastaElMensaje(t *testing.T) {
	db, proyecto, sesion := sesionDePrueba(t)
	crear := mensajesDePrueba(t, db, proyecto, sesion)
	crear("user", "uno")
	agente, err := CrearMensaje(db, proyecto, sesion, "agent", "resp uno", nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := GuardarRazonamiento(db, agente.ID, "pienso"); err != nil {
		t.Fatal(err)
	}
	// Línea del turno 1 (antes del pivot) que debe copiarse.
	if _, err := CrearChatEvento(db, proyecto, sesion, ChatEventoProceso, "[Sub Proceso] Turno1", nil); err != nil {
		t.Fatal(err)
	}
	segundo, err := CrearMensaje(db, proyecto, sesion, "user", "dos", nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	crear("agent", "resp dos")
	// Línea del turno 2 (después del pivot) que NO debe copiarse.
	if _, err := CrearChatEvento(db, proyecto, sesion, ChatEventoHerramienta, "✓ EJEC [x]", nil); err != nil {
		t.Fatal(err)
	}

	nueva, err := BifurcarSesion(db, proyecto, sesion, segundo.ID)
	if err != nil {
		t.Fatal(err)
	}
	if nueva.Name != "FORK:prueba" {
		t.Fatalf("nombre de la bifurcación = %q, quiero %q", nueva.Name, "FORK:prueba")
	}
	historial, err := Historial(db, proyecto, nueva.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(historial) != 3 {
		t.Fatalf("historial bifurcado = %d mensajes, quiero 3: %+v", len(historial), historial)
	}
	if historial[0].Content != "uno" || historial[1].Content != "resp uno" || historial[2].Content != "dos" {
		t.Fatalf("historial bifurcado inesperado: %+v", historial)
	}
	if historial[1].Razonamiento == nil || *historial[1].Razonamiento != "pienso" {
		t.Fatalf("el razonamiento no se copió: %+v", historial[1])
	}
	if original, _ := ListarMensajes(db, proyecto, sesion); len(original) != 4 {
		t.Fatalf("el origen debería seguir con 4 mensajes: %+v", original)
	}
	// Las líneas del turno 1 se copian; las del turno 2 no.
	if ev, err := ListarChatEventos(db, proyecto, nueva.ID); err != nil || len(ev) != 1 || ev[0].Content != "[Sub Proceso] Turno1" {
		t.Fatalf("las líneas del hilo no se copiaron correctamente: %v %v", ev, err)
	}
}

// Bifurcar desde un mensaje ajeno a la sesión no crea nada.
func TestBifurcarSesionConMensajeDeOtraSesionFalla(t *testing.T) {
	db, proyecto, sesion := sesionDePrueba(t)
	if _, err := CrearMensaje(db, proyecto, sesion, "user", "uno", nil, nil, nil); err != nil {
		t.Fatal(err)
	}

	if _, err := BifurcarSesion(db, proyecto, sesion, "inexistente"); err == nil {
		t.Fatal("bifurcar desde un mensaje que no es de la sesión debería fallar")
	}
	sesiones, err := ListarSesiones(db, proyecto)
	if err != nil {
		t.Fatal(err)
	}
	if len(sesiones) != 1 {
		t.Fatalf("no debería haberse creado ninguna sesión: %+v", sesiones)
	}
}
