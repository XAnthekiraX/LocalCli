package agent

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"localcli/internal/tools"
)

// stubTool construye un registro con destinos que capturan lo que reciben.
func stubTool(capturado *tools.Peticion) *tools.Registro {
	handler := func(ctx context.Context, p tools.Peticion) (any, error) {
		*capturado = p
		return "resultado", nil
	}
	r, err := tools.NuevoRegistro(tools.Destinos{
		Archivos: handler,
		Terminal: handler,
		Internet: handler,
	})
	if err != nil {
		panic(err)
	}
	return r
}

// TestDespachoLLevaLaSolicitudATools — T-B006-06: la petición del modelo llega
// a `tools` con su nombre y sus argumentos ya decodificados.
func TestDespachoLLevaLaSolicitudATools(t *testing.T) {
	var capturado tools.Peticion
	d := NuevoDespachador(stubTool(&capturado))
	a := Agente{Nombre: tools.AgenteBuild, Herramientas: tools.HerramientasDeBuild()}

	got, err := d.Despachar(context.Background(), a, SolicitudHerramienta{
		Nombre:     "leer_archivo",
		Argumentos: json.RawMessage(`{"ruta":"ai/docs/PROJECT.md"}`),
	})
	if err != nil {
		t.Fatalf("Despachar: %v", err)
	}
	if got != "resultado" {
		t.Errorf("resultado = %v, quiero el del handler", got)
	}
	if capturado.Herramienta != "leer_archivo" {
		t.Errorf("herramienta = %q, quiero leer_archivo", capturado.Herramienta)
	}
	peticion, ok := capturado.Argumentos.(*tools.PeticionLeerArchivo)
	if !ok || peticion.Ruta != "ai/docs/PROJECT.md" {
		t.Errorf("argumentos = %#v, quiero la ruta decodificada", capturado.Argumentos)
	}
}

// TestDespachoNoSaltaElPermiso — un agente que no declara la herramienta no
// llega al handler de `tools`.
func TestDespachoNoSaltaElPermiso(t *testing.T) {
	var capturado tools.Peticion
	d := NuevoDespachador(stubTool(&capturado))
	charla := Agente{Nombre: "charlatan"} // sin herramientas

	_, err := d.Despachar(context.Background(), charla, SolicitudHerramienta{
		Nombre:     "leer_archivo",
		Argumentos: json.RawMessage(`{"ruta":"a.md"}`),
	})
	if !errors.Is(err, tools.ErrHerramientaNoPermitida) {
		t.Fatalf("err = %v, quiero E_TOOL_NOT_ALLOWED", err)
	}
	if capturado.Herramienta != "" {
		t.Errorf("el handler se llamó sin permiso: %+v", capturado)
	}
}

// TestDespachoRechazaHerramientaDesconocida — un nombre fuera del catálogo no
// tiene contrato con el que decodificar y se rechaza.
func TestDespachoRechazaHerramientaDesconocida(t *testing.T) {
	var capturado tools.Peticion
	d := NuevoDespachador(stubTool(&capturado))
	a := Agente{Nombre: tools.AgenteBuild, Herramientas: tools.HerramientasDeBuild()}

	_, err := d.Despachar(context.Background(), a, SolicitudHerramienta{
		Nombre:     "inventada",
		Argumentos: json.RawMessage(`{}`),
	})
	if !errors.Is(err, tools.ErrHerramientaDesconocida) {
		t.Fatalf("err = %v, quiero E_TOOL_UNKNOWN", err)
	}
}
