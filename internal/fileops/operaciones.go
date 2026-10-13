// Package fileops implementa las herramientas de archivo: resuelve rutas, aplica
// la frontera de la carpeta y registra los cambios. El permiso y la aprobación
// llegan desde la capa universal de tools.
package fileops

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"

	"localcli/internal/tools"
)

// Operaciones agrupa el estado que necesitan las herramientas de archivo.
type Operaciones struct {
	Raiz      string
	DB        *sql.DB
	ProjectID string
}

// Resolver convierte una ruta en su forma absoluta normalizada. No expande `~`
// ni `$HOME`: los rechaza para que el modelo use la forma corta.
func Resolver(raiz, ruta string) (string, string) {
	r := strings.TrimSpace(ruta)
	if r == "" {
		return "", tools.CodBadArgs + ": falta la ruta"
	}
	if strings.HasPrefix(r, "~") || strings.Contains(r, "$HOME") {
		return "", tools.CodBadArgs + ": no se expande ~ ni $HOME; usa la forma relativa (main.go)"
	}
	if filepath.IsAbs(r) {
		return filepath.Clean(r), ""
	}
	return filepath.Clean(filepath.Join(raiz, r)), ""
}

// Canonica devuelve la ruta en forma canónica: relativa a la raíz si está dentro,
// absoluta si queda fuera.
func Canonica(raiz, abs string) string {
	if Dentro(raiz, abs) {
		if rel, err := filepath.Rel(raiz, abs); err == nil {
			return rel
		}
	}
	return abs
}

// ValidarAcceso comprueba la frontera: dentro de la carpeta es accesible; fuera
// exige el motivo y la aprobación del usuario.
func ValidarAcceso(ctx context.Context, raiz, abs, motivo string, c tools.Contexto) error {
	if Dentro(raiz, abs) {
		return nil
	}
	if strings.TrimSpace(motivo) == "" {
		return nuevaError(tools.CodPathOutside, "la ruta está fuera de la carpeta del proyecto y falta el `motivo`")
	}
	return pedirPermiso(ctx, c, tools.Solicitud{
		Titulo:      "Acceso fuera de la carpeta",
		Descripcion: Canonica(raiz, abs),
		Motivo:      motivo,
	})
}

// Dentro dice si una ruta absoluta cuelga de la raíz.
func Dentro(raiz, abs string) bool {
	rel, err := filepath.Rel(raiz, abs)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

// pedirPermiso pide aprobación por el mecanismo único del contexto.
func pedirPermiso(ctx context.Context, c tools.Contexto, s tools.Solicitud) error {
	if c.Ask == nil {
		return nuevaError(tools.CodNeedsApproval, "no hay forma de aprobar la operación")
	}
	decision, err := c.Ask(ctx, s)
	if err != nil {
		return err
	}
	if decision != tools.Aprobar {
		return nuevaError(tools.CodApprovalDeclined, "el usuario declinó la operación")
	}
	return nil
}
