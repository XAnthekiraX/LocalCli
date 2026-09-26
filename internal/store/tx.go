package store

import (
	"database/sql"
	"fmt"
)

// tx.go — escrituras multi-tabla envueltas en transacción con rollback ante
// error (T-B002-10).
//
// DATA_FLOW.md fija los casos que deben ser atómicos: el turno del agente, la
// petición de permiso, su resolución y el registro de un cambio. Este archivo
// da el envoltorio genérico (EjecutarTX) y las operaciones compuestas que lo
// usan; los repositorios de una sola tabla son las escrituras simples.

// PedirPermiso es la operación compuesta de DATA_FLOW.md §2: insertar la
// aprobación y pasar la sesión a esperando_permiso en la MISMA transacción. Es
// un flujo que puede quedar a medias, así que si la segunda escritura falla no
// queda una aprobación huérfana, y si la primera falla la sesión no se queda
// esperando una decisión que nadie le pidió.
func PedirPermiso(db *sql.DB, sessionID, descripcion string) (*Approval, error) {
	var a *Approval
	err := EjecutarTX(db, func(tx *sql.Tx) error {
		nueva, err := pedirAprobacion(tx, sessionID, descripcion)
		if err != nil {
			return err
		}
		if err := actualizarEstadoSesion(tx, sessionID, StatusEsperandoPermiso); err != nil {
			return err
		}
		a = nueva
		return nil
	})
	if err != nil {
		return nil, err
	}
	return a, nil
}

// ResolverPermiso es la operación compuesta de DATA_FLOW.md §2: resolver la
// aprobación y devolver la sesión a su estado normal en la misma transacción,
// para que no queden aprobaciones resueltas con la sesión todavía en
// esperando_permiso (BUSINESS_RULES.md).
//
// estadoSesion es explícito porque el destino depende de la decisión y del
// punto del flujo en que se tome; la transición sigue validándose contra
// ENUMS.md §3.
func ResolverPermiso(db *sql.DB, approvalID, estado, estadoSesion string) error {
	return EjecutarTX(db, func(tx *sql.Tx) error {
		if err := resolverAprobacion(tx, approvalID, estado); err != nil {
			return err
		}
		if estadoSesion == "" {
			return nil
		}
		a, err := obtenerAprobacion(tx, approvalID)
		if err != nil {
			return err
		}
		return actualizarEstadoSesion(tx, a.SessionID, estadoSesion)
	})
}

// ObsoletarPendientesDeSesion pasa a obsoleta todas las aprobaciones pendientes
// de una sesión (BUSINESS_RULES.md: al terminar una sesión, lo que no se
// decidió deja de estar pendiente). Es una sola sentencia y por tanto una sola
// transacción implícita; se expone aquí para que la regla tenga un solo sitio.
func ObsoletarPendientesDeSesion(db *sql.DB, sessionID string) (int, error) {
	res, err := db.Exec(
		`UPDATE approvals SET status = ?, resolved_at = ?
		 WHERE session_id = ? AND status = ?`,
		ApprovalObsoleta, nowISO(), sessionID, ApprovalPendiente)
	if err != nil {
		return 0, traducirError(err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, traducirError(err)
	}
	return int(n), nil
}

// RegistrarCambioEnTX mete la fila de change_history en la transacción que abre
// el llamador. Existe porque el otro lado del paso lógico —escribir el archivo—
// no es una operación de SQL: fileops lo aplica en disco y luego registra. Con
// esta via, todo lo demás del paso (una segunda fila, una actualización de
// estado) queda en la misma transacción que el registro, y si el registro falla
// el llamador puede revertir lo que ya aplicó en disco.
func RegistrarCambioEnTX(db *sql.DB, h *ChangeHistory, antesDeRegistrar func(tx *sql.Tx) error) error {
	return EjecutarTX(db, func(tx *sql.Tx) error {
		if antesDeRegistrar != nil {
			if err := antesDeRegistrar(tx); err != nil {
				return err
			}
		}
		return registrarCambio(tx, h)
	})
}

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
