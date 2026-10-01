package tools

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

// TestInternetDesactivadaDeniega — sin LOCALCLI_ALLOW_INTERNET, una herramienta
// de internet se deniega antes de llegar al cliente.
func TestInternetDesactivadaDeniega(t *testing.T) {
	t.Setenv(VarInternet, "")
	var llamado int
	r := registroStub("x", &llamado)
	_, err := r.Ejecutar(context.Background(), Peticion{
		Agente:      AgenteBuild,
		Permisos:    PermisosDeBuild(),
		Herramienta: "buscar_en_internet",
		Argumentos:  json.RawMessage(`{"consulta":"docs"}`),
	})
	if !errors.Is(err, ErrHerramientaNoPermitida) {
		t.Fatalf("err = %v, quiero E_TOOL_NOT_ALLOWED", err)
	}
	if llamado != 0 {
		t.Error("el cliente de internet se llamó con la salida desactivada")
	}
}

// TestInternetHabilitadaPasa — con la variable puesta, la petición llega al
// handler.
func TestInternetHabilitadaPasa(t *testing.T) {
	t.Setenv(VarInternet, "true")
	var llamado int
	r := registroStub("x", &llamado)
	if _, err := r.Ejecutar(context.Background(), Peticion{
		Agente:      AgenteBuild,
		Permisos:    PermisosDeBuild(),
		Herramienta: "abrir_pagina",
		Argumentos:  json.RawMessage(`{"direccion":"https://example.com"}`),
	}); err != nil {
		t.Fatalf("abrir_pagina: %v", err)
	}
	if llamado != 1 {
		t.Errorf("el handler recibió %d llamadas, quiero 1", llamado)
	}
}

// TestInternetPermitidaAceptaValoresConocidos — el interruptor es tolerante en
// la forma, estricto en el fondo: solo los valores afirmativos la encienden.
func TestInternetPermitidaAceptaValoresConocidos(t *testing.T) {
	for _, v := range []string{"1", "true", "YES", "on", "sí"} {
		t.Setenv(VarInternet, v)
		if !InternetPermitida() {
			t.Errorf("%q debería habilitarla", v)
		}
	}
	for _, v := range []string{"", "0", "false", "nope"} {
		t.Setenv(VarInternet, v)
		if InternetPermitida() {
			t.Errorf("%q no debería habilitarla", v)
		}
	}
}
