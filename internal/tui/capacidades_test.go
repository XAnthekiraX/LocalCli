package tui

import (
	"testing"

	"localcli/internal/tui/components/input"
	"localcli/internal/tui/components/modals"
)

// El chip de visión se fija al arrancar desde el puerto y se actualiza con el
// evento `capacidades_modelo`.
func TestElChipDeVisionReflejaLasCapacidades(t *testing.T) {
	p := &puertoFalso{capacidades: Capacidades{Vision: CapacidadSoportada}}
	app := Nueva(p)
	if app.entrada.Capacidades.Vision != input.Soportada {
		t.Fatalf("el chip de visión debería estar soportado: %v", app.entrada.Capacidades.Vision)
	}

	app.aplicarEvento(Evento{Tipo: EventoCapacidadesModelo, Datos: map[string]string{"vision": "no_soportada"}})
	if app.entrada.Capacidades.Vision != input.NoSoportada {
		t.Fatalf("el evento no actualizó el chip: %v", app.entrada.Capacidades.Vision)
	}

	app.aplicarEvento(Evento{Tipo: EventoCapacidadesModelo, Datos: map[string]string{"vision": "desconocida"}})
	if app.entrada.Capacidades.Vision != input.Desconocida {
		t.Fatalf("el estado desconocido no se aplicó: %v", app.entrada.Capacidades.Vision)
	}
}

// Al cambiar de modelo la visión queda desconocida hasta que el motor la resuelva.
func TestAplicarModeloDesconoceLaVisionHastaElEvento(t *testing.T) {
	p := &puertoFalso{}
	app := appDePrueba(p)
	app.entrada.Capacidades.Vision = input.Soportada

	app.aplicarModelo(modals.Modelo{Nombre: "otro", Motor: "llama.cpp"})
	if app.entrada.Capacidades.Vision != input.Desconocida {
		t.Fatalf("al cambiar de modelo la visión debería quedar desconocida: %v", app.entrada.Capacidades.Vision)
	}
}
