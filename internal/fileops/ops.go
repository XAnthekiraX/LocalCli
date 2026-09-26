package fileops

// ops.go — el punto de entrada del módulo: agrupa la carpeta del proyecto, el
// historial, la sesión y el aprobador.
//
// Fuente de verdad: ai/docs/backend/02-interfaces/TOOLS.md §4 y
// ai/docs/backend/03-security/SECURITY.md §3. Cada operación resuelve la ruta
// contra la frontera, pide aprobación cuando escribe y registra el cambio.
//
// Separación de responsabilidades (DECISIONS.md: "Permiso comprobado en tools,
// aplicado en fileops y exec"): `tools` decide *quién puede pedir*; `fileops`
// aplica el efecto y, con él, el permiso concreto (aprobación, frontera,
// confirmación de borrado).
//
// `fileops` no importa `database/sql`: solo `store` abre la base
// (DECISIONS.md: "store es el único que escribe en SQLite"). Aquí se depende de
// interfaces; los adaptadores viven en `store`.

import (
	"context"

	"localcli/internal/store"
)

// Registrador es lo que `fileops` necesita de `store` para dejar rastro de un
// cambio aplicado.
type Registrador interface {
	Registrar(cambio *store.ChangeHistory) error
}

// Ops ejecuta operaciones de archivo sobre un proyecto.
type Ops struct {
	// Proyecto es la carpeta abierta: la frontera de todas las rutas.
	Proyecto string
	// Historial registra cada cambio aplicado. Sin él no se aplica ninguna
	// escritura (no queda un cambio sin rastro).
	Historial Registrador
	// SesionID liga el cambio a su sesión; puede quedar vacío (NULL en la base).
	SesionID string
	// Aprobador obtiene la decisión del usuario. Sin él, ninguna escritura se
	// aplica: el estado por defecto es cerrado.
	Aprobador Aprobador
}

// aprobar pide la decisión y la traduce a un error tipado. Sin aprobador no se
// escribe (E_NEEDS_APPROVAL); declinada es E_APPROVAL_DECLINED; un borrado sin
// confirmación explícita es E_NEEDS_CONFIRM.
func (o *Ops) aprobar(ctx context.Context, operacion, ruta string, borrado bool) error {
	if o.Aprobador == nil {
		return nuevoError(CodigoNecesitaAprobacion,
			"no se aplica ninguna escritura sin aprobación: "+describir(operacion, ruta))
	}
	d, err := o.Aprobador.Pedir(ctx, SolicitudAprobacion{
		SessionID:   o.SesionID,
		Operacion:   operacion,
		Ruta:        ruta,
		Descripcion: describir(operacion, ruta),
		Borrado:     borrado,
	})
	if err != nil {
		return err
	}
	if !d.Aprobada {
		return nuevoError(CodigoAprobacionDeclinada, "el usuario declinó "+describir(operacion, ruta))
	}
	if borrado && !d.Explicita {
		return nuevoError(CodigoNecesitaConfirmacion,
			"el borrado de "+ruta+" necesita confirmación explícita, no solo aprobación")
	}
	return nil
}

// describir resume la operación para el panel de aprobaciones.
func describir(operacion, ruta string) string {
	verbos := map[string]string{
		"crear_archivo":    "crear el archivo",
		"escribir_archivo": "sobrescribir el archivo",
		"editar_archivo":   "editar el archivo",
		"eliminar_archivo": "borrar el archivo",
		"crear_carpeta":    "crear la carpeta",
		"eliminar_carpeta": "borrar la carpeta",
	}
	verbo := verbos[operacion]
	if verbo == "" {
		verbo = operacion
	}
	return verbo + " " + ruta
}
