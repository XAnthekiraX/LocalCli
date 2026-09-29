package fileops

// approval.go — pedir aprobación antes de aplicar una escritura.
//
// Fuente de verdad: ai/docs/specs/SPEC-ARCHIVOS.md (toda escritura requiere
// aprobación, también dentro de la carpeta; borrar exige confirmación
// explícita) y ai/docs/backend/02-interfaces/TOOLS.md §7 ("La aprobación vive en
// `Contexto.Ask`, no dentro de los handlers").
//
// `fileops` decide *cuándo* hace falta una aprobación y qué hacer con la
// decisión; *cómo* se obtiene la decisión vive detrás de la interfaz
// `Aprobador`. Desde T-B024 el aprobador de producción es un adaptador sobre
// `tools.Contexto.Ask`: el mecanismo queda en un solo sitio y cualquier
// herramienta —incluida una del usuario— pide permiso por el mismo camino. Por
// eso aquí ya no hay una implementación que hable con `store`.

import (
	"context"
)

// Decision es la respuesta del usuario a una solicitud de aprobación.
type Decision struct {
	// Aprobada es la aprobación genérica.
	Aprobada bool
	// Explicita es la confirmación explícita, obligatoria para borrar
	// (E_NEEDS_CONFIRM). Una aprobación genérica no basta para un borrado.
	Explicita bool
}

// SolicitudAprobacion describe lo que se pide, para que el panel lo muestre.
type SolicitudAprobacion struct {
	SessionID   string
	Operacion   string
	Ruta        string
	Descripcion string
	Borrado     bool
	// Fuera marca que la ruta sale de la carpeta del proyecto: la operación
	// necesita permiso (SPEC-ARCHIVOS §Reglas), y el panel lo dice.
	Fuera bool
	// Motivo es la explicación del agente: obligatoria cuando la ruta sale de la
	// carpeta, y visible en el panel junto a la propuesta.
	Motivo string
}

// Aprobador pide una decisión sobre una operación de archivo.
type Aprobador interface {
	Pedir(ctx context.Context, s SolicitudAprobacion) (Decision, error)
}

// AprobadorFunc adapta una función a Aprobador. Es lo que usa el cableado para
// llevar la petición a `tools.Contexto.Ask`.
type AprobadorFunc func(ctx context.Context, s SolicitudAprobacion) (Decision, error)

func (f AprobadorFunc) Pedir(ctx context.Context, s SolicitudAprobacion) (Decision, error) {
	return f(ctx, s)
}
