package fileops

// history.go — T-B008-07: registrar cada cambio aplicado en `change_history`.
//
// Fuente de verdad: ai/docs/database/02-rules/DATA_FLOW.md §3 y §6 ("Cuando
// `build` aplica un cambio aprobado, se escribe el archivo y se inserta la fila
// en `change_history` con lo anterior y lo nuevo. La base registra lo que pasó,
// no lo que se intentó") y ai/docs/database/01-schema/TABLES.md (la tabla
// guarda antes y después).
//
// El registro vive en `store`; aquí solo se construye la fila. La operación
// (crear/escribir/editar/eliminar) decide qué contenido va en `before` y en
// `after`; `store` comprueba que la combinación es la que admite el esquema.

import (
	"errors"

	"localcli/internal/store"
)

// registrar inserta la fila del cambio aplicado. Sin historial no hay registro
// y, por tanto, tampoco se considera aplicable la escritura: quien llama
// revierte.
func (o *Ops) registrar(operacion, rutaRelativa, before, after string) error {
	if o.Historial == nil {
		return errors.New("fileops: no hay historial donde registrar el cambio")
	}
	return o.Historial.Registrar(&store.ChangeHistory{
		SessionID:     o.SesionID,
		Operation:     operacion,
		FilePath:      rutaRelativa,
		BeforeContent: before,
		AfterContent:  after,
	})
}
