package exec

import "testing"

// TestCapturaCortaAlLimite — T-B009-04: la salida se corta al tope y queda
// marcada, sin superarlo.
func TestCapturaCortaAlLimite(t *testing.T) {
	c := nuevaCaptura(10)
	if _, err := c.Stdout().Write([]byte("123456789012345")); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Stderr().Write([]byte("error")); err != nil {
		t.Fatal(err)
	}
	salida, errorSalida := c.Salidas()
	if len(salida) != 10 {
		t.Errorf("stdout = %q (%d bytes), quiero 10", salida, len(salida))
	}
	if errorSalida != "" {
		t.Errorf("stderr = %q, quiero vacío: el tope es compartido", errorSalida)
	}
	if !c.Truncado() {
		t.Error("la salida cortada debe quedar marcada como truncada")
	}
}

// TestCapturaNoTruncaSiCabe — si todo cabe, no se marca nada.
func TestCapturaNoTruncaSiCabe(t *testing.T) {
	c := nuevaCaptura(100)
	_, _ = c.Stdout().Write([]byte("corto"))
	salida, _ := c.Salidas()
	if salida != "corto" || c.Truncado() {
		t.Fatalf("salida=%q truncado=%v", salida, c.Truncado())
	}
}
