// Package store — módulo store de LocalCli.
//
// Único acceso a SQLite: esquema, modo WAL, foreign_keys, user_version y transacciones. Ningún otro módulo escribe en la base; store no decide nada de negocio.
package store
