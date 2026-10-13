package llm

import "testing"

func TestCapacidadDeLista(t *testing.T) {
	// Lista declarada: lo ausente es "no soportada", no desconocida.
	c := DeLista([]string{"tools", "reasoning"})
	if c.Herramientas != Soportada || c.Razonamiento != Soportada || c.Vision != NoSoportada {
		t.Fatalf("capacidades = %+v", c)
	}
	// Lista vacía: declara que no hay ninguna.
	if v := DeLista([]string{}); v.Herramientas != NoSoportada {
		t.Fatalf("lista vacía = %+v", v)
	}
}

func TestCapacidadSinDatoEsDesconocida(t *testing.T) {
	// El valor cero es Desconocida: la ausencia de dato nunca es permiso.
	var cero Capacidad
	if cero != Desconocida {
		t.Fatalf("el valor cero de Capacidad = %v, quiero Desconocida", cero)
	}
	c := DeLista(nil)
	if c != TDesconocidas() {
		t.Fatalf("lista nula = %+v", c)
	}
	if c.Herramientas != Desconocida {
		t.Fatal("sin dato, herramientas no puede leerse como soportada")
	}
}

func TestVentanaDeContexto(t *testing.T) {
	casos := []struct {
		declarada, tope, tokens int
		conocida                bool
	}{
		{8192, 16384, 8192, true},   // cabe: manda el modelo
		{32768, 16384, 16384, true}, // no cabe: manda el tope
		{0, 16384, 16384, false},    // desconocida: manda el tope
		{-1, 16384, 16384, false},   // sin dato
		{4096, 0, 4096, true},       // sin tope
	}
	for _, c := range casos {
		tokens, conocida := VentanaDeContexto(c.declarada, c.tope)
		if tokens != c.tokens || conocida != c.conocida {
			t.Errorf("VentanaDeContexto(%d, %d) = (%d, %v), quiero (%d, %v)",
				c.declarada, c.tope, tokens, conocida, c.tokens, c.conocida)
		}
	}
}

func TestVentanaEfectiva(t *testing.T) {
	casos := []struct {
		override, contexto, defecto, quiero int
	}{
		{0, 32768, 16384, 32768},   // el modelo manda sobre el defecto
		{0, 0, 16384, 16384},       // sin dato: el defecto
		{8192, 32768, 16384, 8192}, // el override del usuario manda
		{8192, 0, 16384, 8192},     // el override manda aun sin contexto
	}
	for _, c := range casos {
		if got := VentanaEfectiva(c.override, c.contexto, c.defecto); got != c.quiero {
			t.Errorf("VentanaEfectiva(%d, %d, %d) = %d, quiero %d",
				c.override, c.contexto, c.defecto, got, c.quiero)
		}
	}
}

func TestEstimarTokens(t *testing.T) {
	if n := EstimarTokens(""); n != 0 {
		t.Fatalf("texto vacío = %d, quiero 0", n)
	}
	if n := EstimarTokens("abcd"); n != 1 {
		t.Fatalf("4 caracteres = %d, quiero 1", n)
	}
	if n := EstimarTokens("abcdefghij"); n != 3 {
		t.Fatalf("10 caracteres = %d, quiero 3", n)
	}
}
