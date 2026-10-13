package session

import "testing"

func TestEstadosCerrados(t *testing.T) {
	for _, e := range []string{EstadoInactiva, EstadoTrabajando, EstadoEsperandoPermiso, EstadoTerminada, EstadoError} {
		if !EsEstadoValido(e) {
			t.Fatalf("%q debería ser un estado válido", e)
		}
	}
	if EsEstadoValido("corriendo") {
		t.Fatal("un estado fuera del catálogo no es válido")
	}
}

func TestTransicionesValidasEInvalidas(t *testing.T) {
	validas := [][2]string{
		{EstadoInactiva, EstadoTrabajando},
		{EstadoTrabajando, EstadoInactiva},
		{EstadoTrabajando, EstadoEsperandoPermiso},
		{EstadoEsperandoPermiso, EstadoTrabajando},
		{EstadoTrabajando, EstadoTerminada},
		{EstadoTrabajando, EstadoError},
		{EstadoTerminada, EstadoInactiva},
		{EstadoError, EstadoInactiva},
	}
	for _, v := range validas {
		if !TransicionValida(v[0], v[1]) {
			t.Fatalf("%s → %s debería ser válida", v[0], v[1])
		}
	}

	invalidas := [][2]string{
		{EstadoTerminada, EstadoTrabajando},
		{EstadoError, EstadoTrabajando},
		{EstadoInactiva, EstadoTerminada},
		{EstadoInactiva, EstadoEsperandoPermiso},
		{EstadoEsperandoPermiso, EstadoTerminada},
		{EstadoTerminada, EstadoTerminada},
	}
	for _, v := range invalidas {
		if TransicionValida(v[0], v[1]) {
			t.Fatalf("%s → %s debería rechazarse", v[0], v[1])
		}
	}
}
