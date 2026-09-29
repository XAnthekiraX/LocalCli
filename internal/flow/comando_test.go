package flow

import (
	"context"
	"testing"

	"localcli/internal/tools"
)

// --- T-B016-02: comandos explícitos ----------------------------------------

// Un comando es una línea que empieza por `/`. Qué comandos existen de verdad
// lo decide el catálogo cargado de `.localcli/flows/`, no esta función.
func TestEsComandoReconoceLaBarraInicial(t *testing.T) {
	for _, texto := range []string{"/resolver", "  /crear x", "/ejecutar"} {
		if !esComando(texto) {
			t.Errorf("%q debe ser un comando", texto)
		}
	}
	for _, texto := range []string{"", "hola", "dime de qué se trata este proyecto"} {
		if esComando(texto) {
			t.Errorf("%q no debe ser un comando", texto)
		}
	}
}

func TestObjetivoDeQuitaElComando(t *testing.T) {
	if got := ObjetivoDe("/crear añadir marca de agua"); got != "añadir marca de agua" {
		t.Errorf("objetivo = %q", got)
	}
	// Sin resto, el objetivo es el propio comando: nunca se entrega vacío.
	if got := ObjetivoDe("/planificar"); got != "/planificar" {
		t.Errorf("objetivo = %q", got)
	}
}

// --- T-B016-03: la detección solo propone ----------------------------------

func TestSugerenciaTrabajoNoArranca(t *testing.T) {
	if _, ok := SugerenciaTrabajo("primero esto, luego esto"); !ok {
		t.Error("una petición ordenada debe proponerse")
	}
	if _, ok := SugerenciaTrabajo("dime de qué se trata este proyecto"); ok {
		t.Error("una conversación normal no debe proponerse")
	}
	if _, ok := SugerenciaTrabajo("/ejecutar"); ok {
		t.Error("un comando explícito no es una sugerencia")
	}
}

// --- T-B016-04: el chat corre con el agente activo -------------------------

func TestConversarUsaElAgenteActivo(t *testing.T) {
	var traza []string
	m := &Motor{
		Contexto: &stubContexto{},
		Agente:   &stubAgente{traza: &traza},
	}
	res, err := m.Conversar(context.Background(), tools.AgenteBuild, "dime de qué va", nil, nil)
	if err != nil {
		t.Fatalf("Conversar: %v", err)
	}
	if len(traza) != 1 || traza[0] != "agente:build" {
		t.Errorf("el agente activo debe correr el turno: %v", traza)
	}
	if res.Texto == "" {
		t.Error("la respuesta del agente no puede quedar vacía")
	}
	if _, err := m.Conversar(context.Background(), "", "x", nil, nil); err == nil {
		t.Error("un chat sin agente debe rechazarse")
	}
}
