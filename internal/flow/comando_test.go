package flow

import (
	"context"
	"testing"

	"localcli/internal/tools"
)

// --- T-B016-02: comandos explícitos ----------------------------------------

func TestComandoDeReconoceLosSeisComandos(t *testing.T) {
	casos := map[string]struct {
		flujo string
		cola  bool
	}{
		"/planificar": {"planificacion", false},
		"/crear":      {"trabajo/crear", false},
		"/actualizar": {"trabajo/actualizar", false},
		"/eliminar":   {"trabajo/eliminar", false},
		"/resolver":   {"resolver", false},
		"/ejecutar":   {"", true},
	}
	for texto, quiere := range casos {
		cmd, ok := ComandoDe(texto)
		if !ok {
			t.Errorf("%q debe reconocerse como comando", texto)
			continue
		}
		if cmd.Consumir != quiere.cola {
			t.Errorf("%q: consumir = %v, quería %v", texto, cmd.Consumir, quiere.cola)
		}
		if !quiere.cola && cmd.Flujo.Nombre != quiere.flujo {
			t.Errorf("%q: flujo = %q, quería %q", texto, cmd.Flujo.Nombre, quiere.flujo)
		}
	}
}

func TestTextoDesconocidoNoEsComando(t *testing.T) {
	for _, texto := range []string{"", "hola", "/otro", "dime de qué se trata este proyecto", "/planificarx"} {
		if _, ok := ComandoDe(texto); ok {
			t.Errorf("%q no debe ser un comando de flujo", texto)
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
