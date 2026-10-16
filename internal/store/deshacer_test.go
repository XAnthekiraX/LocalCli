package store

import (
	"database/sql"
	"testing"
)

// Un turno son la petición del usuario y la respuesta del agente. Deshacerlo lo
// saca de la conversación y rehacerlo lo devuelve.
func TestDeshacerYRehacerTurno(t *testing.T) {
	db, proyecto, sesion := sesionDePrueba(t)
	crear := mensajesDePrueba(t, db, proyecto, sesion)
	crear("user", "uno")
	crear("agent", "resp uno")
	crear("user", "dos")
	crear("agent", "resp dos")

	turno, n, err := DeshacerTurno(db, proyecto, sesion)
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("mensajes deshechos = %d, quiero 2", n)
	}
	visibles, err := ListarMensajes(db, proyecto, sesion)
	if err != nil {
		t.Fatal(err)
	}
	if len(visibles) != 2 || visibles[0].Content != "uno" || visibles[1].Content != "resp uno" {
		t.Fatalf("tras deshacer quedó: %+v", visibles)
	}

	if err := RehacerTurno(db, turno); err != nil {
		t.Fatal(err)
	}
	visibles, err = ListarMensajes(db, proyecto, sesion)
	if err != nil {
		t.Fatal(err)
	}
	if len(visibles) != 4 || visibles[2].Content != "dos" || visibles[3].Content != "resp dos" {
		t.Fatalf("tras rehacer quedó: %+v", visibles)
	}
}

// Dos deshaceres se rehacen en orden inverso: cada turno capturado vuelve a su
// sitio cuando se rehace.
func TestDeshacerDosSeRehaceEnOrdenInverso(t *testing.T) {
	db, proyecto, sesion := sesionDePrueba(t)
	crear := mensajesDePrueba(t, db, proyecto, sesion)
	crear("user", "uno")
	crear("agent", "resp uno")
	crear("user", "dos")
	crear("agent", "resp dos")

	turno2, n, err := DeshacerTurno(db, proyecto, sesion)
	if err != nil || n != 2 {
		t.Fatalf("primer deshacer = (%d, %v), quiero (2, nil)", n, err)
	}
	turno1, n, err := DeshacerTurno(db, proyecto, sesion)
	if err != nil || n != 2 {
		t.Fatalf("segundo deshacer = (%d, %v), quiero (2, nil)", n, err)
	}
	if visibles, _ := ListarMensajes(db, proyecto, sesion); len(visibles) != 0 {
		t.Fatalf("tras dos deshaceres = %d mensajes, quiero 0", len(visibles))
	}

	if err := RehacerTurno(db, turno1); err != nil {
		t.Fatal(err)
	}
	visibles, _ := ListarMensajes(db, proyecto, sesion)
	if len(visibles) != 2 || visibles[0].Content != "uno" {
		t.Fatalf("el primer rehacer debería devolver el turno «uno»: %+v", visibles)
	}
	if err := RehacerTurno(db, turno2); err != nil {
		t.Fatal(err)
	}
	if visibles, _ = ListarMensajes(db, proyecto, sesion); len(visibles) != 4 {
		t.Fatalf("tras rehacer todo = %d mensajes, quiero 4", len(visibles))
	}
}

// El razonamiento del turno sobrevive al deshacer y al rehacer.
func TestRehacerConservaElRazonamiento(t *testing.T) {
	db, proyecto, sesion := sesionDePrueba(t)
	if _, err := CrearMensaje(db, proyecto, sesion, "user", "hola", nil, nil, nil); err != nil {
		t.Fatal(err)
	}
	agente, err := CrearMensaje(db, proyecto, sesion, "agent", "respuesta", nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := GuardarRazonamiento(db, agente.ID, "pienso"); err != nil {
		t.Fatal(err)
	}

	turno, n, err := DeshacerTurno(db, proyecto, sesion)
	if err != nil || n != 2 {
		t.Fatalf("deshacer = (%d, %v), quiero (2, nil)", n, err)
	}
	if err := RehacerTurno(db, turno); err != nil {
		t.Fatal(err)
	}
	historial, err := Historial(db, proyecto, sesion)
	if err != nil {
		t.Fatal(err)
	}
	if len(historial) != 2 || historial[1].Razonamiento == nil || *historial[1].Razonamiento != "pienso" {
		t.Fatalf("el razonamiento no volvió: %+v", historial)
	}
}

// Las líneas del hilo del turno —herramientas y sub-procesos— acompañan al
// deshacer y al rehacer: el hilo que queda, y el que vuelve, se lee igual que el
// que se vio en vivo (DATA_FLOW §Creación, SPEC-INTERFAZ §/undo).
func TestDeshacerYRehacerTurnoSeLlevaSusLineas(t *testing.T) {
	db, proyecto, sesion := sesionDePrueba(t)
	crear := mensajesDePrueba(t, db, proyecto, sesion)
	crear("user", "uno")
	crear("agent", "resp uno")
	crear("user", "dos")
	crear("agent", "resp dos")

	ms := 400
	if _, err := CrearChatEvento(db, proyecto, sesion, ChatEventoProceso, "[Sub Proceso] X", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := CrearChatEvento(db, proyecto, sesion, ChatEventoHerramienta, "✓ LEER [x] · 70 líneas", &ms); err != nil {
		t.Fatal(err)
	}

	turno, n, err := DeshacerTurno(db, proyecto, sesion)
	if err != nil || n != 2 {
		t.Fatalf("deshacer = (%d, %v), quiero (2, nil)", n, err)
	}
	// Las líneas del turno deshecho no deben quedar huérfanas.
	if eventos, err := ListarChatEventos(db, proyecto, sesion); err != nil || len(eventos) != 0 {
		t.Fatalf("tras deshacer quedaron %d líneas: %v", len(eventos), err)
	}

	if err := RehacerTurno(db, turno); err != nil {
		t.Fatal(err)
	}
	eventos, err := ListarChatEventos(db, proyecto, sesion)
	if err != nil {
		t.Fatal(err)
	}
	if len(eventos) != 2 {
		t.Fatalf("rehacer debería devolver las 2 líneas del turno, hay %d", len(eventos))
	}
	// Las líneas vuelven en su orden y con su contenido.
	if eventos[0].Tipo != ChatEventoProceso || eventos[0].Content != "[Sub Proceso] X" {
		t.Fatalf("la primera línea no volvió igual: %+v", eventos[0])
	}
	if eventos[1].Tipo != ChatEventoHerramienta || eventos[1].DurationMs == nil || *eventos[1].DurationMs != 400 {
		t.Fatalf("la segunda línea no volvió igual: %+v", eventos[1])
	}
}

// Sin un turno que deshacer, deshacer no cambia nada y rehacer no escribe nada.
func TestDeshacerYRehacerSinTurnoNoHacenNada(t *testing.T) {
	db, proyecto, sesion := sesionDePrueba(t)

	if turno, n, err := DeshacerTurno(db, proyecto, sesion); err != nil || n != 0 || len(turno.Mensajes) != 0 {
		t.Fatalf("deshacer sin turno = (%d, %v), quiero (0, nil)", n, err)
	}
	if err := RehacerTurno(db, Turno{}); err != nil {
		t.Fatalf("rehacer sin turno: %v", err)
	}
}

// mensajesDePrueba devuelve un creador de mensajes ligado a la sesión.
func mensajesDePrueba(t *testing.T, db *sql.DB, proyecto, sesion string) func(rol, texto string) {
	t.Helper()
	return func(rol, texto string) {
		if _, err := CrearMensaje(db, proyecto, sesion, rol, texto, nil, nil, nil); err != nil {
			t.Fatal(err)
		}
	}
}
