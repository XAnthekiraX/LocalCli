package tui

import "testing"

// --- T-F007-09: el aviso de pendientes ---------------------------------------

// El aviso es nada con cero, singular con una y plural con más.
func TestElFormatoDelAvisoConCeroUnoYVarios(t *testing.T) {
	if got := formatoAvisoPendientes(0); got != "" {
		t.Errorf("sin pendientes no hay aviso: %q", got)
	}
	if got := formatoAvisoPendientes(-1); got != "" {
		t.Errorf("un recuento imposible tampoco avisa: %q", got)
	}
	if got := formatoAvisoPendientes(1); got != "1 aprobación esperando tu decisión" {
		t.Errorf("singular: %q", got)
	}
	if got := formatoAvisoPendientes(3); got != "3 aprobaciones esperando tu decisión" {
		t.Errorf("plural: %q", got)
	}
}

// El número que pinta el aviso sale de las líneas pendientes y se actualiza con
// cada evento: sube con la petición y baja con la resolución, de cualquier sesión.
func TestElAvisoSeActualizaConCadaEvento(t *testing.T) {
	app := appConSesion()

	app.aplicarEvento(eventoDe("s1", EventoPeticionAprobacion, map[string]string{"aprobacion": "a1"}))
	app.aplicarEvento(eventoDe("s2", EventoPeticionAprobacion, map[string]string{"aprobacion": "a2"}))
	if aviso := formatoAvisoPendientes(app.aprobaciones.Pendientes()); aviso != "2 aprobaciones esperando tu decisión" {
		t.Fatalf("con dos pendientes: %q", aviso)
	}

	app.aplicarEvento(eventoDe("s2", EventoAprobacionResuelta, map[string]string{"aprobacion": "a2"}))
	if aviso := formatoAvisoPendientes(app.aprobaciones.Pendientes()); aviso != "1 aprobación esperando tu decisión" {
		t.Fatalf("al resolverse una: %q", aviso)
	}

	app.aplicarEvento(eventoDe("s1", EventoAprobacionResuelta, map[string]string{"aprobacion": "a1"}))
	if aviso := formatoAvisoPendientes(app.aprobaciones.Pendientes()); aviso != "" {
		t.Fatalf("sin pendientes el aviso desaparece: %q", aviso)
	}
}
