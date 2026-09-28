package agent

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"localcli/internal/tools"
)

// registroCaptura arma el registro con un handler que recuerda los argumentos
// ya decodificados.
func registroCaptura(capturado *any) *tools.Registro {
	var impls []tools.Herramienta
	for _, nombre := range tools.NombresCatalogo() {
		n := nombre
		h, _ := tools.NuevaHerramienta(n, func(ctx context.Context, args any, c tools.Contexto) (tools.Resultado, error) {
			if capturado != nil {
				*capturado = args
			}
			return tools.Resultado{Salida: "resultado"}, nil
		})
		impls = append(impls, h)
	}
	r, err := tools.NuevoRegistro(impls)
	if err != nil {
		panic(err)
	}
	return r
}

// TestDespachoLLevaLaSolicitudATools — la petición del modelo llega a `tools`
// con su nombre y sus argumentos ya decodificados.
func TestDespachoLLevaLaSolicitudATools(t *testing.T) {
	var capturado any
	d := NuevoDespachador(registroCaptura(&capturado))
	a := Agente{Nombre: tools.AgenteBuild, Permisos: []Permiso{
		{Accion: "leer", Efecto: EfectoPermitir},
		{Accion: "editar", Efecto: EfectoPermitir},
		{Accion: "ejecutar", Efecto: EfectoPermitir},
		{Accion: "internet", Efecto: EfectoPermitir},
	}}

	res, err := d.Despachar(context.Background(), a, SolicitudHerramienta{
		Nombre:     "leer_archivo",
		Argumentos: json.RawMessage(`{"ruta":"ai/docs/PROJECT.md"}`),
	})
	if err != nil {
		t.Fatalf("Despachar: %v", err)
	}
	if res.Salida != "resultado" {
		t.Errorf("resultado = %q", res.Salida)
	}
	peticion, ok := capturado.(*tools.PeticionLeerArchivo)
	if !ok || peticion.Ruta != "ai/docs/PROJECT.md" {
		t.Errorf("argumentos = %#v, quiero la ruta decodificada", capturado)
	}
}

// TestDespachoNoSaltaElPermiso — un agente sin la acción `leer` no llega al
// handler.
func TestDespachoNoSaltaElPermiso(t *testing.T) {
	d := NuevoDespachador(registroCaptura(nil))
	charla := Agente{Nombre: "charlatan"} // sin permisos

	_, err := d.Despachar(context.Background(), charla, SolicitudHerramienta{
		Nombre:     "leer_archivo",
		Argumentos: json.RawMessage(`{"ruta":"a.md"}`),
	})
	if !errors.Is(err, tools.ErrHerramientaNoPermitida) {
		t.Fatalf("err = %v, quiero E_TOOL_NOT_ALLOWED", err)
	}
}

// TestDefinicionesDelAgente — las definiciones que viajan al modelo salen de
// las acciones: `plan` no ve ninguna de escritura.
func TestDefinicionesDelAgente(t *testing.T) {
	d := NuevoDespachador(registroCaptura(nil))
	plan := Agente{Nombre: tools.AgentePlan, Permisos: []Permiso{
		{Accion: "leer", Efecto: EfectoPermitir},
		{Accion: "ejecutar", Efecto: EfectoPermitir},
		{Accion: "internet", Efecto: EfectoPermitir},
		{Accion: "tareas", Efecto: EfectoPermitir},
	}}
	defs := d.Definiciones(plan)
	if len(defs) != 8 {
		t.Fatalf("plan ve %d herramientas, quiero 8", len(defs))
	}
	for _, def := range defs {
		if h, ok := tools.Buscar(def.Function.Name); ok && h.SoloBuild() {
			t.Errorf("plan no puede ver %s", def.Function.Name)
		}
		if def.Type != "function" || def.Function.Parameters == nil {
			t.Errorf("la definición de %s debe llevar esquema", def.Function.Name)
		}
	}
}
