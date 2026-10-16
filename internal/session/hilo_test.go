package session

import (
	"testing"

	"localcli/internal/tui"
)

// Al retomar una sesión, el hilo guardado trae mensajes y líneas del sistema
// —herramientas y sub-procesos— intercalados en su orden cronológico, incluso
// cuando caen dentro del mismo segundo (DATA_FLOW §Creación). Antes, las líneas
// no se guardaban y desaparecían al reabrir la sesión.
func TestHistorialIntercalaMensajesYLineasDelHilo(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	o, sesionID := sesionDePrueba(t, "http://127.0.0.1:1")

	// El reloj real no da este orden de forma fiable: se siembra a mano para
	// fijar el caso difícil —todo dentro del mismo segundo.
	mensaje := func(id, rol, created string) {
		t.Helper()
		_, err := o.db.Exec(
			"INSERT INTO messages (id, session_id, role, content, created_at) VALUES (?, ?, ?, ?, ?)",
			id, sesionID, rol, id, created,
		)
		if err != nil {
			t.Fatal(err)
		}
	}
	linea := func(id, tipo, created string) {
		t.Helper()
		_, err := o.db.Exec(
			"INSERT INTO chat_evento (id, session_id, tipo, content, created_at) VALUES (?, ?, ?, ?, ?)",
			id, sesionID, tipo, id, created,
		)
		if err != nil {
			t.Fatal(err)
		}
	}

	mensaje("user", "user", "2026-01-01T10:00:00.000000100Z")
	linea("LEER", "herramienta", "2026-01-01T10:00:00.000000200Z")
	mensaje("agent", "agent", "2026-01-01T10:00:00.000000300Z")
	linea("[Sub Proceso] X", "proceso", "2026-01-01T10:00:00.000000400Z")

	piezas, err := o.Historial(sesionID)
	if err != nil {
		t.Fatal(err)
	}
	if len(piezas) != 4 {
		t.Fatalf("piezas del hilo = %d", len(piezas))
	}

	// El orden esperado: mensaje, herramienta, mensaje, sub-proceso.
	if m, ok := piezas[0].(*tui.MensajeHistorial); !ok || m.Rol != "user" {
		t.Fatalf("la pieza 0 debería ser el mensaje del usuario: %#v", piezas[0])
	}
	if l, ok := piezas[1].(*tui.LineaHistorial); !ok || l.Tipo != "herramienta" {
		t.Fatalf("la pieza 1 debería ser la línea de herramienta: %#v", piezas[1])
	}
	if m, ok := piezas[2].(*tui.MensajeHistorial); !ok || m.Rol != "agent" {
		t.Fatalf("la pieza 2 debería ser la respuesta del agente: %#v", piezas[2])
	}
	if l, ok := piezas[3].(*tui.LineaHistorial); !ok || l.Tipo != "proceso" {
		t.Fatalf("la pieza 3 debería ser el sub-proceso: %#v", piezas[3])
	}
}
