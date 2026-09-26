package tools

import (
	"context"
	"errors"
	"testing"
)

// stubDestino devuelve un handler que registra que fue llamado y devuelve una
// marca, para distinguir a qué destino llegó cada categoría.
func stubDestino(marca string, llamado *int) Handler {
	return func(ctx context.Context, p Peticion) (any, error) {
		*llamado++
		return marca, nil
	}
}

// TestRegistroExigeDestinos — montar el registro sin los tres destinos es un
// error de wiring, no algo que deba reventar en tiempo de ejecución.
func TestRegistroExigeDestinos(t *testing.T) {
	var llamado int
	h := stubDestino("x", &llamado)
	casos := []Destinos{
		{},
		{Archivos: h},
		{Archivos: h, Terminal: h},
	}
	for i, d := range casos {
		if _, err := NuevoRegistro(d); err == nil {
			t.Errorf("caso %d: quiero error por destino sin conectar", i)
		}
	}
}

// TestHandlerPorNombre — el registro asocia cada herramienta con el handler de
// su categoría, y una herramienta desconocida no tiene handler.
func TestHandlerPorNombre(t *testing.T) {
	var archivos, terminal, internet int
	r, err := NuevoRegistro(Destinos{
		Archivos: stubDestino("archivos", &archivos),
		Terminal: stubDestino("terminal", &terminal),
		Internet: stubDestino("internet", &internet),
	})
	if err != nil {
		t.Fatalf("NuevoRegistro: %v", err)
	}

	casos := []struct {
		herramienta string
		quiero      string
	}{
		{"leer_archivo", "archivos"},
		{"crear_carpeta", "archivos"},
		{"ejecutar_comando", "terminal"},
		{"buscar_en_internet", "internet"},
		{"abrir_pagina", "internet"},
	}
	for _, c := range casos {
		h, err := r.Handler(c.herramienta)
		if err != nil {
			t.Errorf("%s: %v", c.herramienta, err)
			continue
		}
		got, err := h(context.Background(), Peticion{Herramienta: c.herramienta})
		if err != nil {
			t.Errorf("%s: handler devolvió %v", c.herramienta, err)
			continue
		}
		if got != c.quiero {
			t.Errorf("%s llegó a %v, quiero %v", c.herramienta, got, c.quiero)
		}
	}

	if _, err := r.Handler("inventada"); !errors.Is(err, ErrHerramientaDesconocida) {
		t.Errorf("herramienta desconocida: err = %v, quiero E_TOOL_UNKNOWN", err)
	}
}
