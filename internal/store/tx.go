package store

import (
	"database/sql"
	"fmt"
)

// tx.go — escrituras multi-tabla envueltas en transacción con rollback ante
// error (T-B002-10).
//
// DATA_FLOW.md fija el caso principal: "Las dos inserciones [mensaje del
// agente + razonamiento] van en la misma transacción que la actualización de
// status de la sesión". Este archivo da el envoltorio genérico; los helpers de
// arriba son las escrituras multi-tabla concretas.

// EjecutarTX corre fn dentro de una transacción. Si fn devuelve error o entra
// en pánico, se hace rollback y ninguna fila queda escrita; si devuelve nil,
// se hace commit. Es el patrón único de escritura multi-tabla del paquete.
func EjecutarTX(db *sql.DB, fn func(tx *sql.Tx) error) (err error) {
	tx, err := db.Begin()
	if err != nil {
		return traducirError(err)
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			err = fmt.Errorf("pánico recuperado en transacción: %v", p)
			return
		}
		if err != nil {
			tx.Rollback()
		}
	}()
	if err = fn(tx); err != nil {
		return err // ya traducido por fn; el defer hace rollback
	}
	if err = tx.Commit(); err != nil {
		return traducirError(err)
	}
	return nil
}
