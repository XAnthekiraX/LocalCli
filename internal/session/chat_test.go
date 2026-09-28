package session

// Tests de T-B016-01: la vista principal responde como chat y solo un comando
// explícito arranca etapas. Una prueba, una regla (TESTING.md §3).

import (
	"context"
	"testing"

	"localcli/internal/flow"
)

// Una petición sin comando no dispara `flow` ni consume la cola.
func TestConversarRespondeComoChatYNoArrancaFlujo(t *testing.T) {
	motor := nuevoMotor(flow.EstadoTerminado)
	g, alm := gestorDe(t, motor)
	s, err := g.Crear("sesión", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := g.Conversar(context.Background(), s.ID, "plan", "  dime de qué va el proyecto  ", nil); err != nil {
		t.Fatal(err)
	}
	esperarEstado(t, g, s.ID, EstadoTerminada)

	if ll := motor.llamadasHechas(); len(ll) != 0 {
		t.Errorf("el chat no arranca etapas: %v", ll)
	}
	if got := motor.conversacionesHechas(); len(got) != 1 || got[0] != "plan:dime de qué va el proyecto" {
		t.Errorf("el chat corre con el agente activo y el mensaje recortado: %v", got)
	}
	if motor.cola != nil {
		t.Error("el chat no consume la cola")
	}
	h, err := alm.Historial(s.ID)
	if err != nil || len(h) != 1 || h[0].Role != "user" {
		t.Errorf("el historial debe tener el mensaje del usuario: %v (%d)", err, len(h))
	}
}

// Un turno de chat cierra con el aviso de chat: aquí no corrió ninguna etapa,
// y decir «terminó el trabajo» haría parecer que arrancó un flujo que el
// usuario no pidió.
func TestElChatCierraSinAnunciarTrabajoTerminado(t *testing.T) {
	g, _ := gestorDe(t, nuevoMotor(flow.EstadoTerminado))
	eventos, baja := g.Bus.Suscribir()
	defer baja()

	s, _ := g.Crear("sesión", "")
	if err := g.Conversar(context.Background(), s.ID, "plan", "hola", nil); err != nil {
		t.Fatal(err)
	}
	esperarEstado(t, g, s.ID, EstadoTerminada)

	recibidos := recoger(eventos)
	if !contieneEvento(recibidos, EventoNotificacion, "terminó de responder") {
		t.Errorf("falta el aviso de cierre de chat: %v", recibidos)
	}
	for _, e := range recibidos {
		if e.Nombre == EventoNotificacion && e.Datos["motivo"] == "la sesión terminó el trabajo" {
			t.Errorf("el chat no debe anunciar «terminó el trabajo»: %v", e.Datos)
		}
	}
}

// Un comando explícito arranca su flujo, no el por defecto.
func TestArrancarFlujoCorreElFlujoExplicito(t *testing.T) {
	motor := nuevoMotor(flow.EstadoTerminado)
	g, _ := gestorDe(t, motor)
	s, _ := g.Crear("sesión", "")
	if err := g.ArrancarFlujo(context.Background(), s.ID, flow.FlujoResolver(), "arregla el fallo"); err != nil {
		t.Fatal(err)
	}
	esperarEstado(t, g, s.ID, EstadoTerminada)
	if got := motor.llamadasHechas(); len(got) != 1 || got[0] != "arregla el fallo" {
		t.Errorf("llamadas = %v", got)
	}
}

// Una sesión que quedó en `trabajando` sin trabajo vivo (cierre abrupto del
// proceso) acepta una petición nueva: el estado obsoleto se baja a inactiva
// antes del turno, en lugar de fallar con trabajando → trabajando.
func TestUnaSesionAtascadaEnTrabajandoAceptaPeticion(t *testing.T) {
	motor := nuevoMotor(flow.EstadoTerminado)
	g, alm := gestorDe(t, motor)
	s, _ := g.Crear("sesión", "")
	if err := alm.CambiarEstado(s.ID, EstadoTrabajando); err != nil {
		t.Fatal(err)
	}
	if err := g.Conversar(context.Background(), s.ID, "plan", "hola", nil); err != nil {
		t.Fatalf("una sesión obsoleta en trabajando debe aceptar el turno: %v", err)
	}
	esperarEstado(t, g, s.ID, EstadoTerminada)
}

// El trabajo ordenado se propone y no arranca la cola.
func TestConversarProponeTrabajoOrdenadoSinArrancarCola(t *testing.T) {
	motor := nuevoMotor(flow.EstadoTerminado)
	g, alm := gestorDe(t, motor)
	s, _ := g.Crear("sesión", "")
	if err := g.Conversar(context.Background(), s.ID, "plan", "primero esto, luego esto", nil); err != nil {
		t.Fatal(err)
	}
	esperarEstado(t, g, s.ID, EstadoTerminada)

	if motor.cola != nil {
		t.Error("la detección no puede arrancar la cola")
	}
	h, _ := alm.Historial(s.ID)
	hayAviso := false
	for _, m := range h {
		if m.Role == "system" {
			hayAviso = true
		}
	}
	if !hayAviso {
		t.Error("el trabajo ordenado debe proponerse")
	}
}
