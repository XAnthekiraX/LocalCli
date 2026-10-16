package exec

import (
	"context"

	"localcli/internal/tools"
)

// aprobar pide permiso para un comando destructivo o que toca rutas fuera de la
// carpeta. Nunca se ejecuta sin aprobación: si no hay mecanismo, o el usuario
// declina, se corta.
func aprobar(ctx context.Context, c tools.Contexto, s tools.Solicitud) error {
	if c.Ask == nil {
		return nuevaError(tools.CodNeedsApproval, "el comando requiere aprobación y no hay forma de darla")
	}
	decision, err := c.Ask(ctx, s)
	if err != nil {
		return err
	}
	if decision != tools.Aprobar {
		return nuevaError(tools.CodApprovalDeclined, "el usuario declinó el comando")
	}
	return nil
}
