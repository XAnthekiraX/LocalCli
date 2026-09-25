package store

import _ "embed"

// schemaSQL es el DDL del esquema v1, embebido desde schema.sql para que el
// binario siga siendo portable y la base se cree sin archivos externos.
//
// source: schema.sql — única copia del DDL; no se duplica en código.
//
//go:embed schema.sql
var schemaSQL string
