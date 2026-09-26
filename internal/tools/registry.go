package tools

// registry.go — T-B007-04: el registro que asocia cada herramienta con el
// handler de su destino.
//
// Fuente de verdad: ai/docs/backend/02-interfaces/TOOLS.md §7 ("archivo →
// fileops; terminal → exec; internet → el cliente de internet") y
// ai/docs/backend/02-interfaces/INTERFACES-GENERAL.md §5 (contratos entre
// módulos).
//
// `tools` no aplica permisos ni ejecuta la herramienta: solo conoce el
// catálogo y sabe a qué destino pertenece cada categoría. Los handlers reales
// los conecta la raíz de composición; aquí solo se enrutan. Esa separación es
// deliberada (DECISIONS.md): "quién puede pedir" se decide en `tools`, "quién
// aplica" se decide en `fileops` y `exec`.

import (
	"context"
	"errors"
)

// Peticion es lo que `agent` entrega a `tools`: quién pide, qué herramienta y
// con qué argumentos. `Permitidas` es el catálogo del agente, tal cual lo
// declara su JSON; la comprobación de permiso parte de ahí.
type Peticion struct {
	Agente      string   // "plan" o "build", para trazabilidad
	Permitidas  []string // herramientas del catálogo del agente
	Herramienta string   // nombre exacto dentro del catálogo cerrado
	Argumentos  any      // valor de la tabla `peticiones` de dto_request.go
}

// Handler procesa una petición ya validada. Lo implementan `fileops`, `exec` y
// el cliente de internet. Recibe la Peticion completa porque `fileops`
// necesita saber qué operación concreta se le pide.
type Handler func(ctx context.Context, p Peticion) (any, error)

// Destinos son los tres handlers a los que `tools` puede enrutar. Cada uno es
// la frontera de un módulo distinto.
type Destinos struct {
	Archivos Handler // fileops
	Terminal Handler // exec
	Internet Handler // cliente de internet
}

// Registro resuelve el destino de una herramienta por su categoría.
type Registro struct {
	destinos Destinos
}

// NuevoRegistro conecta los tres destinos. Falla si falta alguno: un destino
// sin conectar no puede quedarse en nil y reventar en tiempo de ejecución; se
// detecta al montar el motor.
func NuevoRegistro(d Destinos) (*Registro, error) {
	if d.Archivos == nil || d.Terminal == nil || d.Internet == nil {
		return nil, errors.New("tools: falta conectar un destino (archivos, terminal o internet)")
	}
	return &Registro{destinos: d}, nil
}

// Handler devuelve el handler del destino que corresponde a la herramienta.
// El segundo valor es un error E_TOOL_UNKNOWN si el nombre no está en el
// catálogo cerrado: una herramienta inventada no tiene destino.
func (r *Registro) Handler(nombre string) (Handler, error) {
	h, ok := Buscar(nombre)
	if !ok {
		return nil, nuevoError(CodigoHerramientaDesconocida,
			"la herramienta "+nombre+" no está en el catálogo cerrado")
	}
	switch h.Categoria {
	case CatArchivos:
		return r.destinos.Archivos, nil
	case CatTerminal:
		return r.destinos.Terminal, nil
	case CatInternet:
		return r.destinos.Internet, nil
	}
	return nil, nuevoError(CodigoHerramientaDesconocida,
		"la herramienta "+nombre+" tiene una categoría sin destino")
}
