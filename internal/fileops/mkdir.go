package fileops

// mkdir.go — T-B008-06: crear y eliminar carpetas con la misma regla de
// aprobación.
//
// Fuente de verdad: ai/docs/backend/05-quality/VALIDATION.md §1 ("`crear_carpeta`
// falla si ya existe") y ai/docs/specs/SPEC-ARCHIVOS.md (toda escritura pasa
// por aprobación; borrar exige confirmación explícita).
//
// El borrado de una carpeta se niega sobre la raíz del proyecto: la frontera ya
// impide salir de ella, pero borrar la propia carpeta abierta destruiría todo
// el proyecto y no es algo que una herramienta deba poder hacer.

import (
	"context"
	"fmt"
	"os"

	"localcli/internal/store"
	"localcli/internal/tools"
)

// CrearCarpeta crea una carpeta. Falla si ya existe.
func (o *Ops) CrearCarpeta(ctx context.Context, ruta string) (tools.RespuestaEscritura, error) {
	abs, err := Resolver(o.Proyecto, ruta)
	if err != nil {
		return tools.RespuestaEscritura{}, err
	}
	if _, statErr := os.Stat(abs); statErr == nil {
		return tools.RespuestaEscritura{}, nuevoError(CodigoRutaExiste,
			"la carpeta "+ruta+" ya existe")
	}
	if err := o.aprobar(ctx, store.OpCrearCarpeta, ruta, false); err != nil {
		return tools.RespuestaEscritura{}, err
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return tools.RespuestaEscritura{}, fmt.Errorf("no se pudo crear la carpeta %s: %w", ruta, err)
	}
	if err := o.registrar(store.OpCrearCarpeta, mustRelativa(o.Proyecto, abs), "", ""); err != nil {
		_ = os.Remove(abs)
		return tools.RespuestaEscritura{}, err
	}
	return tools.RespuestaEscritura{Confirmacion: "carpeta creada", Ruta: ruta}, nil
}

// EliminarCarpeta borra una carpeta tras confirmación explícita.
func (o *Ops) EliminarCarpeta(ctx context.Context, ruta string) (tools.RespuestaEliminar, error) {
	abs, err := Resolver(o.Proyecto, ruta)
	if err != nil {
		return tools.RespuestaEliminar{}, err
	}
	raiz, err := Resolver(o.Proyecto, ".")
	if err != nil {
		return tools.RespuestaEliminar{}, err
	}
	if abs == raiz {
		return tools.RespuestaEliminar{}, nuevoError(CodigoArgumentosInvalidos,
			"no se puede borrar la carpeta del proyecto")
	}
	info, err := os.Stat(abs)
	if err != nil {
		return tools.RespuestaEliminar{}, nuevoError(CodigoArgumentosInvalidos,
			"no se pudo borrar "+ruta+": "+err.Error())
	}
	if !info.IsDir() {
		return tools.RespuestaEliminar{}, nuevoError(CodigoArgumentosInvalidos,
			ruta+" no es una carpeta")
	}
	if err := o.aprobar(ctx, store.OpEliminarCarpeta, ruta, true); err != nil {
		return tools.RespuestaEliminar{}, err
	}
	if err := os.RemoveAll(abs); err != nil {
		return tools.RespuestaEliminar{}, fmt.Errorf("no se pudo borrar la carpeta %s: %w", ruta, err)
	}
	// Una carpeta no tiene contenido que revertir, así que el registro va
	// después del borrado. Si falla no se puede deshacer, pero tampoco se
	// oculta: el error sube.
	if err := o.registrar(store.OpEliminarCarpeta, mustRelativa(o.Proyecto, abs), "", ""); err != nil {
		return tools.RespuestaEliminar{}, err
	}
	return tools.RespuestaEliminar{Confirmacion: "carpeta borrada"}, nil
}
