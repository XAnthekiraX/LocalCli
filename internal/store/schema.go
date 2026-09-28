package store

import _ "embed"

// schemaSQL es el DDL del esquema v1, embebido desde schema.sql para que el
// binario siga siendo portable y la base se cree sin archivos externos.
//
// source: schema.sql — única copia del DDL; no se duplica en código.
//
//go:embed schema.sql
var schemaSQL string

// todoSQL es el DDL de la tabla `todos` (migración 002), embebido igual que el
// esquema base para que el binario siga siendo portable.
//
// source: todo.sql — única copia del DDL de la lista de pasos.
//
//go:embed todo.sql
var todoSQL string
