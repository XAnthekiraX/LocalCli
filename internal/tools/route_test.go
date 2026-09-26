package tools

import (
	"context"
	"errors"
	"testing"
)

// TestEnrutadoLlegaAlDestinoCorrecto — T-B007-06: cada categoría llega a su
// módulo: archivos → fileops, terminal → exec, internet → cliente de internet.
func TestEnrutadoLlegaAlDestinoCorrecto(t *testing.T) {
	var archivos, terminal, internet int
	r, err := NuevoRegistro(Destinos{
		Archivos: stubDestino("archivos", &archivos),
		Terminal: stubDestino("terminal", &terminal),
		Internet: stubDestino("internet", &internet),
	})
	if err != nil {
		t.Fatalf("NuevoRegistro: %v", err)
	}
	t.Setenv(VarInternet, "1")

	peticiones := []struct {
		nombre string
		args   any
		quiero string
	}{
		{"leer_archivo", &PeticionLeerArchivo{Ruta: "a.md"}, "archivos"},
		{"ejecutar_comando", &PeticionEjecutarComando{Comando: "go test ./..."}, "terminal"},
		{"buscar_en_internet", &PeticionBuscarInternet{Consulta: "docs"}, "internet"},
	}
	for _, c := range peticiones {
		got, err := r.Enrutar(context.Background(), Peticion{
			Agente:      AgenteBuild,
			Permitidas:  HerramientasDeBuild(),
			Herramienta: c.nombre,
			Argumentos:  c.args,
		})
		if err != nil {
			t.Errorf("%s: %v", c.nombre, err)
			continue
		}
		if got != c.quiero {
			t.Errorf("%s llegó a %v, quiero %v", c.nombre, got, c.quiero)
		}
	}
	if archivos == 0 || terminal == 0 || internet == 0 {
		t.Errorf("destinos llamados: archivos=%d terminal=%d internet=%d", archivos, terminal, internet)
	}
}

// TestInternetDesactivadaDeniega — T-B007-07: sin LOCALCLI_ALLOW_INTERNET,
// una herramienta de internet se deniega antes de llegar al cliente.
func TestInternetDesactivadaDeniega(t *testing.T) {
	t.Setenv(VarInternet, "")
	var internet int
	r, err := NuevoRegistro(Destinos{
		Archivos: stubDestino("archivos", new(int)),
		Terminal: stubDestino("terminal", new(int)),
		Internet: stubDestino("internet", &internet),
	})
	if err != nil {
		t.Fatalf("NuevoRegistro: %v", err)
	}
	_, err = r.Enrutar(context.Background(), Peticion{
		Agente:      AgenteBuild,
		Permitidas:  HerramientasDeBuild(),
		Herramienta: "buscar_en_internet",
		Argumentos:  &PeticionBuscarInternet{Consulta: "docs"},
	})
	if !errors.Is(err, ErrHerramientaNoPermitida) {
		t.Fatalf("err = %v, quiero E_TOOL_NOT_ALLOWED", err)
	}
	if internet != 0 {
		t.Errorf("el cliente de internet se llamó con la salida desactivada")
	}
}

// TestInternetHabilitadaPasa — con la variable puesta, la petición llega al
// cliente de internet.
func TestInternetHabilitadaPasa(t *testing.T) {
	t.Setenv(VarInternet, "true")
	var internet int
	r, err := NuevoRegistro(Destinos{
		Archivos: stubDestino("archivos", new(int)),
		Terminal: stubDestino("terminal", new(int)),
		Internet: stubDestino("internet", &internet),
	})
	if err != nil {
		t.Fatalf("NuevoRegistro: %v", err)
	}
	if _, err := r.Enrutar(context.Background(), Peticion{
		Agente:      AgenteBuild,
		Permitidas:  HerramientasDeBuild(),
		Herramienta: "abrir_pagina",
		Argumentos:  &PeticionAbrirPagina{Direccion: "https://example.com"},
	}); err != nil {
		t.Fatalf("abrir_pagina: %v", err)
	}
	if internet != 1 {
		t.Errorf("el cliente de internet recibió %d llamadas, quiero 1", internet)
	}
}

// TestEnrutarNoLlegaAlHandlerSinPermisoNiPayload — una petición rechazada no
// toca el destino: ni por permiso ni por argumentos inválidos.
func TestEnrutarNoLlegaAlHandlerSinPermisoNiPayload(t *testing.T) {
	var archivos int
	r, err := NuevoRegistro(Destinos{
		Archivos: stubDestino("archivos", &archivos),
		Terminal: stubDestino("terminal", new(int)),
		Internet: stubDestino("internet", new(int)),
	})
	if err != nil {
		t.Fatalf("NuevoRegistro: %v", err)
	}

	// plan pidiendo escritura: E_TOOL_NOT_ALLOWED.
	if _, err := r.Enrutar(context.Background(), Peticion{
		Agente:      AgentePlan,
		Permitidas:  HerramientasDePlan(),
		Herramienta: "crear_archivo",
		Argumentos:  &PeticionCrearArchivo{Ruta: "a.md", Contenido: "x"},
	}); !errors.Is(err, ErrHerramientaNoPermitida) {
		t.Errorf("plan escribiendo: err = %v, quiero E_TOOL_NOT_ALLOWED", err)
	}

	// payload incompleto: E_BAD_ARGS.
	if _, err := r.Enrutar(context.Background(), Peticion{
		Agente:      AgenteBuild,
		Permitidas:  HerramientasDeBuild(),
		Herramienta: "leer_archivo",
		Argumentos:  &PeticionLeerArchivo{},
	}); !errors.Is(err, ErrArgumentosInvalidos) {
		t.Errorf("payload incompleto: err = %v, quiero E_BAD_ARGS", err)
	}

	if archivos != 0 {
		t.Errorf("el handler de archivos recibió %d llamadas que debían rechazarse", archivos)
	}
}
