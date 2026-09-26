package fileops

// approval.go — T-B008-04: pedir aprobación antes de aplicar una escritura.
//
// Fuente de verdad: ai/docs/specs/SPEC-ARCHIVOS.md (toda escritura requiere
// aprobación, también dentro de la carpeta; borrar exige confirmación
// explícita) y ai/docs/database/02-rules/DATA_FLOW.md §7 (paso 3: "se crea la
// aprobación y la sesión queda esperando"; paso 5: `build` aplica).
//
// `fileops` decide *cuándo* hace falta una aprobación y qué hacer con la
// decisión; *cómo* se obtiene la decisión vive detrás de la interfaz
// `Aprobador`. La implementación de producción (AprobadorSQLite) crea la fila
// en `approvals` con su transacción de sesión y espera a que se resuelva; la
// espera concreta (el canal de eventos) la aporta `session`. Así `fileops` no
// conoce la TUI ni el motor de flujos, ni abre la base por su cuenta.

import (
	"context"
	"fmt"

	"localcli/internal/store"
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
}

// Aprobador pide una decisión sobre una operación de archivo.
type Aprobador interface {
	Pedir(ctx context.Context, s SolicitudAprobacion) (Decision, error)
}

// AprobadorFunc adapta una función a Aprobador. Útil para tests y wiring.
type AprobadorFunc func(ctx context.Context, s SolicitudAprobacion) (Decision, error)

func (f AprobadorFunc) Pedir(ctx context.Context, s SolicitudAprobacion) (Decision, error) {
	return f(ctx, s)
}

// Permisos es lo que `fileops` necesita de `store` para levantar una petición
// de aprobación. Lo implementa `store.Permisos`.
type Permisos interface {
	PedirPermiso(sessionID, descripcion string) (*store.Approval, error)
}

// Espera bloquea hasta que la aprobación con ese id se resuelve y devuelve su
// estado final (`aprobada`, `declinada` u `obsoleta`). Es el punto que aporta
// `session`, dueño del canal de eventos de aprobación.
type Espera func(ctx context.Context, approvalID string) (string, error)

// AprobadorSQLite es el aprobador de producción: registra la fila de
// `approvals` (y pasa la sesión a esperando_permiso, en la misma transacción)
// y espera la resolución de la persona.
type AprobadorSQLite struct {
	Permisos Permisos
	SesionID string
	Espera   Espera
}

// Pedir crea la aprobación y espera. El estado final decide: aprobada → sí;
// declinada u obsoleta → no; cualquier otro es un error de programación.
func (a *AprobadorSQLite) Pedir(ctx context.Context, s SolicitudAprobacion) (Decision, error) {
	if a.Permisos == nil || a.Espera == nil {
		return Decision{}, fmt.Errorf("fileops: el aprobador no tiene permisos o forma de esperar la decisión")
	}
	sesion := s.SessionID
	if sesion == "" {
		sesion = a.SesionID
	}
	ap, err := a.Permisos.PedirPermiso(sesion, s.Descripcion)
	if err != nil {
		return Decision{}, err
	}
	estado, err := a.Espera(ctx, ap.ID)
	if err != nil {
		return Decision{}, err
	}
	switch estado {
	case store.ApprovalAprobada:
		// Resolver el borrado en el panel es la confirmación explícita.
		return Decision{Aprobada: true, Explicita: true}, nil
	case store.ApprovalDeclinada, store.ApprovalObsoleta:
		return Decision{Aprobada: false}, nil
	default:
		return Decision{}, fmt.Errorf("fileops: estado de aprobación desconocido %q", estado)
	}
}
