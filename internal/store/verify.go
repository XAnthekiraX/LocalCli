package store

import (
	"database/sql"
	"errors"
	"fmt"
)

// ErrEsquemaDesfasado indica que el archivo .localcli/state.db existe pero su
// esquema no es el que este binario define, sin que user_version lo revele.
//
// Situación: la v1 se corrigió después de haberla creado (las seis tablas
// declaraban `id TEXT PRIMARY KEY` sin NOT NULL, y faltaba el UNIQUE de
// context_audit). Corregir la definición no cambia user_version, así que una
// base ya creada conservaría el esquema viejo y aceptaría un `id` nulo sin
// avisar. Detectar el desfase aquí es preferible a una migración que reconstruya
// seis tablas: SQLite no puede añadir NOT NULL a una clave primaria de texto
// sin recrear la tabla, y MIGRATIONS.md §5 no permite destruir datos en
// silencio (change_history es irrecuperable). La base es estado desechable del
// proyecto, no un dato del usuario, así que se pide recrearla.
var ErrEsquemaDesfasado = errors.New("el esquema de la base no corresponde a la versión del binario")

// tablasEsperadas son las seis tablas del esquema, en el orden de SCHEMA.md.
var tablasEsperadas = []string{
	"sessions", "messages", "reasoning", "approvals", "context_audit", "change_history",
}

// verifyEsquema comprueba que el esquema vivo coincide con schema.sql. Solo
// comprueba los puntos que se corrigieron tras la primera publicación de la v1
// —la nulabilidad de `id` y el UNIQUE de context_audit—, porque son los que un
// user_version correcto no puede distinguir. Se ejecuta tras migrate().
func verifyEsquema(db *sql.DB) error {
	for _, tabla := range tablasEsperadas {
		notnull, err := idEsNotNull(db, tabla)
		if err != nil {
			return err
		}
		if !notnull {
			return fmt.Errorf("%w: la tabla %q admite `id` nulo; el archivo %s es de una versión anterior del binario. Bórralo para que se regenere (no contiene datos del proyecto, solo el historial de cambios de archivos, y ese módulo aún no está en uso)",
				ErrEsquemaDesfasado, tabla, dbFileName)
		}
	}
	var unico int
	err := db.QueryRow(
		"SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name='idx_context_audit_unico'",
	).Scan(&unico)
	if err != nil {
		return traducirError(err)
	}
	if unico == 0 {
		return fmt.Errorf("%w: falta el índice único de auditoría de contexto; borra %s para que se regenere",
			ErrEsquemaDesfasado, dbFileName)
	}
	return nil
}

// idEsNotNull lee PRAGMA table_info de una tabla y dice si su columna `id`
// declara NOT NULL. En SQLite un `TEXT PRIMARY KEY` NO lo implica: sin la
// declaración explícita la columna admite NULL, y una fila con id nulo entra
// sin que ninguna restricción lo note (CONSTRAINTS.md §1 la exige).
func idEsNotNull(db *sql.DB, tabla string) (bool, error) {
	filas, err := db.Query("PRAGMA table_info(" + tabla + ")")
	if err != nil {
		return false, traducirError(err)
	}
	defer filas.Close()

	vista := false
	for filas.Next() {
		var (
			cid        int
			nombre     string
			tipo       string
			notNull    int
			defecto    sql.NullString
			primaryKey int
		)
		if err := filas.Scan(&cid, &nombre, &tipo, &notNull, &defecto, &primaryKey); err != nil {
			return false, traducirError(err)
		}
		if nombre == "id" {
			vista = true
			if notNull == 1 {
				return true, nil
			}
			return false, nil
		}
	}
	if err := filas.Err(); err != nil {
		return false, traducirError(err)
	}
	if !vista {
		return false, fmt.Errorf("%w: la tabla %q no tiene columna `id`", ErrEsquemaDesfasado, tabla)
	}
	return false, nil
}
