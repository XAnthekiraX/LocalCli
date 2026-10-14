package store

import "testing"

// El resumen del historial va y viene por la sesión: nace ausente, se guarda y se
// limpia. Ver [[specs/SPEC-HISTORIAL-CONVERSACION]].
func TestResumenHistorialIdaYVuelta(t *testing.T) {
	db := nuevaBase(t)
	projectID := proyectoDePrueba(t, db)
	sesion, err := CrearSesion(db, projectID, "s", nil)
	if err != nil {
		t.Fatal(err)
	}

	if _, ok, err := ObtenerResumenHistorial(db, projectID, sesion.ID); err != nil || ok {
		t.Fatalf("sin resumen: ok=%v err=%v", ok, err)
	}

	if err := GuardarResumenHistorial(db, projectID, sesion.ID, "## Objective\n- algo", "m-9"); err != nil {
		t.Fatal(err)
	}
	res, ok, err := ObtenerResumenHistorial(db, projectID, sesion.ID)
	if err != nil || !ok {
		t.Fatalf("resumen guardado: ok=%v err=%v", ok, err)
	}
	if res.Texto != "## Objective\n- algo" || res.HastaID != "m-9" {
		t.Fatalf("resumen leído = %+v", res)
	}

	if err := LimpiarResumenHistorial(db, projectID, sesion.ID); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := ObtenerResumenHistorial(db, projectID, sesion.ID); err != nil || ok {
		t.Fatalf("tras limpiar: ok=%v err=%v", ok, err)
	}
}
