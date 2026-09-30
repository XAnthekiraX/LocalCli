package agent

// dispatch.go — despachar las peticiones de herramienta del modelo hacia
// `tools` y exponer las definiciones que viajan al modelo.
//
// Fuente de verdad: ai/docs/backend/02-interfaces/TOOLS.md §11 (el cable:
// `agent` pide los esquemas a `tools` y los pone en la petición; después entrega
// las peticiones del modelo a `Registro.Ejecutar`) y §8 (la capa universal).
//
// `agent` no ejecuta la herramienta ni decide permisos: convierte la petición
// del modelo en la `Peticion` de `tools` —con las ACCIONES del agente— y deja
// que la capa universal compruebe el permiso, valide y ejecute. El modelo no
// puede concederse permisos: lo único que aporta es el nombre y los argumentos,
// y la política la pone el motor desde el JSON del agente.

import (
	"context"
	"encoding/json"

	"localcli/internal/ollama"
	"localcli/internal/tools"
)

// SolicitudHerramienta es lo que el modelo pide: el nombre de una herramienta
// y sus argumentos ya formados.
type SolicitudHerramienta struct {
	Nombre     string
	Argumentos json.RawMessage
}

// Despachador lleva las peticiones de herramienta de un agente a `tools`.
type Despachador struct {
	registro *tools.Registro
}

// NuevoDespachador envuelve el registro de herramientas de `tools`.
func NuevoDespachador(registro *tools.Registro) *Despachador {
	return &Despachador{registro: registro}
}

// Registro devuelve el registro subyacente (lo usa el cableado).
func (d *Despachador) Registro() *tools.Registro {
	if d == nil {
		return nil
	}
	return d.registro
}

// Definiciones traduce el catálogo efectivo del agente al formato que espera
// `/api/chat`: las herramientas de las acciones que el agente concede. Incluye
// las del usuario, que se reparten con las mismas reglas.
func (d *Despachador) Definiciones(a Agente) []ollama.Herramienta {
	if d == nil || d.registro == nil {
		return nil
	}
	defs := d.registro.Definiciones(a.Acciones())
	out := make([]ollama.Herramienta, 0, len(defs))
	for _, def := range defs {
		out = append(out, ollama.Herramienta{
			Type: "function",
			Function: ollama.Definicion{
				Name:        def.Nombre,
				Description: def.Descripcion,
				Parameters:  def.Esquema,
			},
		})
	}
	return out
}

// Despachar ejecuta una petición de herramienta por la capa universal. El
// primer valor es lo que se le devuelve al modelo; el error es un fallo que el
// modelo no puede corregir (permiso denegado, fallo del harness).
func (d *Despachador) Despachar(ctx context.Context, a Agente, s SolicitudHerramienta) (tools.Resultado, error) {
	if d == nil || d.registro == nil {
		return tools.Resultado{Error: "no hay herramientas conectadas"}, nil
	}
	return d.registro.Ejecutar(ctx, tools.Peticion{
		Agente:      a.Nombre,
		SesionID:    tools.SesionDe(ctx),
		Permisos:    a.Acciones(),
		Herramienta: s.Nombre,
		Argumentos:  s.Argumentos,
	})
}
