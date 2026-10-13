package agent

import (
	"context"
	"strings"
	"testing"

	"localcli/internal/llm"
)

func TestAjustarAlContextoDescartaLoAntiguo(t *testing.T) {
	e := &Ejecutor{Agente: Agente{Nombre: "chat", Prompt: ""}, Ventana: 1100}
	mensajes := []Mensaje{
		{Rol: RolUsuario, Contenido: strings.Repeat("a", 4000)},
		{Rol: RolUsuario, Contenido: "reciente"},
	}
	ajustados := e.ajustarAlContexto(mensajes, nil)
	if len(ajustados) != 1 || ajustados[0].Contenido != "reciente" {
		t.Fatalf("debería quedarse con el más nuevo: %+v", ajustados)
	}
}

func TestAjustarAlContextoSinVentanaNoToca(t *testing.T) {
	e := &Ejecutor{Agente: Agente{Nombre: "chat"}, Ventana: 0}
	mensajes := []Mensaje{
		{Rol: RolUsuario, Contenido: strings.Repeat("a", 4000)},
		{Rol: RolUsuario, Contenido: "reciente"},
	}
	ajustados := e.ajustarAlContexto(mensajes, nil)
	if len(ajustados) != 2 {
		t.Fatalf("sin ventana no debería recortar: %d", len(ajustados))
	}
}

func TestAjustarAlContextoTruncaElMasNuevoSiNoCabe(t *testing.T) {
	e := &Ejecutor{Agente: Agente{Nombre: "chat"}, Ventana: 1100}
	mensajes := []Mensaje{{Rol: RolUsuario, Contenido: strings.Repeat("a", 4000)}}
	ajustados := e.ajustarAlContexto(mensajes, nil)
	if len(ajustados) != 1 {
		t.Fatalf("debería conservar el más nuevo truncado: %+v", ajustados)
	}
	if !strings.Contains(ajustados[0].Contenido, "recortado") {
		t.Fatalf("debería marcar el recorte: %q", ajustados[0].Contenido)
	}
	if len(ajustados[0].Contenido) >= 4000 {
		t.Fatalf("debería haber acortado el contenido, no %d runas", len(ajustados[0].Contenido))
	}
}

func TestAjustarAlContextoNoDejaHerramientaHuerfana(t *testing.T) {
	e := &Ejecutor{Agente: Agente{Nombre: "chat"}, Ventana: 1100}
	mensajes := []Mensaje{
		{Rol: RolAsistente, Llamadas: []Llamada{{ID: "1", Nombre: "leer_archivo", Argumentos: "{}"}}},
		{Rol: RolHerramienta, IDLlamada: "1", Contenido: strings.Repeat("x", 4000)},
	}
	ajustados := e.ajustarAlContexto(mensajes, nil)
	if len(ajustados) != 1 || ajustados[0].Rol == RolHerramienta {
		t.Fatalf("no debería quedar un resultado de herramienta sin su llamada: %+v", ajustados)
	}
}

func TestElTurnoRecortaLoQueNoCabeEnLaVentana(t *testing.T) {
	motor := &motorGuion{respuestas: []Respuesta{{Texto: "ok"}}}
	var avisos int
	e := &Ejecutor{
		Agente:    Agente{Nombre: "chat", Prompt: ""},
		Motor:     motor,
		Capacidad: llm.NoSoportada,
		Ventana:   1100,
		Avisar:    func(string, string) { avisos++ },
	}
	historial := []Mensaje{
		{Rol: RolUsuario, Contenido: strings.Repeat("a", 4000)},
		{Rol: RolUsuario, Contenido: "reciente"},
	}
	if _, err := e.Turno(context.Background(), historial, nil); err != nil {
		t.Fatalf("turno: %v", err)
	}
	msgs := motor.pedidos[0].Mensajes
	if len(msgs) != 2 {
		t.Fatalf("esperaba sistema + 1 mensaje, tengo %d: %+v", len(msgs), msgs)
	}
	if msgs[1].Contenido != "reciente" {
		t.Fatalf("debería conservar el más nuevo: %q", msgs[1].Contenido)
	}
	if avisos == 0 {
		t.Fatal("debería avisar del recorte")
	}
}
