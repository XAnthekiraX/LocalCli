package store

import (
	"database/sql"
	"errors"
	"fmt"
)

// schemaVersion es la versión del esquema que conoce este binario. La base se
// crea ya en su versión 1 con las seis tablas (MIGRATIONS.md, "Nota sobre el
// esquema actual"); user_version = 1 significa que la migración 001 está
// aplicada.
const schemaVersion = 1

// migration es una migración numerada, en orden ascendente, cada una en su
// propia transacción (MIGRATIONS.md §2). to es la versión resultante.
type migration struct {
	num int    // número correlativo de tres dígitos (001, 002, ...)
	nom string // nombre corto: qué hace, no dónde (MIGRATIONS.md §3)
	to  int    // user_version tras aplicarla
	ddl string // sentencias a ejecutar dentro de la transacción
}

// migrations lista las migraciones pendientes de aplicar sobre una base en
// versión 0. Hoy solo existe 001-crear-schema: crear el archivo y su esquema
// inicial ocurren en el mismo paso (MIGRATIONS.md, nota final). Los índices se
// aplican junto con la creación, no después.
var migrations = []migration{
	{num: 1, nom: "001-crear-schema", to: 1, ddl: schemaSQL},
}

// userVersion lee PRAGMA user_version de la conexión.
func userVersion(db *sql.DB) (int, error) {
	var v int
	if err := db.QueryRow("PRAGMA user_version").Scan(&v); err != nil {
		return 0, fmt.Errorf("no se pudo leer user_version: %w", err)
	}
	return v, nil
}

// migrate aplica todas las migraciones que falten, una a una y cada una en su
// propia transacción (MIGRATIONS.md §2). Si una falla, se detiene ahí: las
// anteriores quedan aplicadas y la que falló se revierte sola. El error indica
// el número de la migración que falló, para avisar al usuario.
func migrate(db *sql.DB) error {
	v, err := userVersion(db)
	if err != nil {
		return err
	}
	if v > schemaVersion {
		return fmt.Errorf("la base está en versión %d, más nueva que la que entiende este binario (%d)", v, schemaVersion)
	}
	for _, m := range migrations {
		if v >= m.to {
			continue // ya aplicada: segunda apertura es idempotente
		}
		if err := applyMigration(db, m); err != nil {
			return err
		}
		v = m.to
	}
	return nil
}

// applyMigration ejecuta una migración completa en una transacción y sube
// user_version dentro de ella, de modo que queden atómicos el DDL y la versión.
func applyMigration(db *sql.DB, m migration) (err error) {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("migración %03d (%s): %w", m.num, m.nom, err)
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			err = fmt.Errorf("migración %03d (%s): pánico recuperado: %v", m.num, m.nom, p)
		} else if err != nil {
			tx.Rollback()
		}
	}()
	if _, err = tx.Exec(m.ddl); err != nil {
		return fmt.Errorf("migración %03d (%s) falló: %w", m.num, m.nom, err)
	}
	// PRAGMA user_version no acepta placeholders: se interpola un entero propio.
	if _, err = tx.Exec(fmt.Sprintf("PRAGMA user_version = %d", m.to)); err != nil {
		return fmt.Errorf("migración %03d (%s): no se pudo subir user_version: %w", m.num, m.nom, err)
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("migración %03d (%s): commit: %w", m.num, m.nom, err)
	}
	return nil
}

// ErrMigrationFailed permite detectar un fallo de migración desde fuera del
// paquete; el mensaje siempre lleva el número de migración que falló.
var ErrMigrationFailed = errors.New("fallo de migración del esquema")
