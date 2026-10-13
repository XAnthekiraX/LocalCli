package exec

import (
	"context"

	"localcli/internal/tools"
)

// aprobar pide permiso para un comando que no está en la lista blanca. Nunca se
// ejecuta sin aprobación: si no hay mecanismo, o el usuario declina, se corta.
func aprobar(ctx context.Context, c tools.Contexto, s tools.Solicitud) error {
	if c.Ask == nil {
		return nuevaError(tools.CodCmdNotWhitelisted, "el comando no está en la lista blanca y no hay forma de aprobarlo")
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
