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

// flowContextSQL es el DDL del bloque de contexto de un flujo (migración 003),
// embebido igual que el esquema base para que el binario siga siendo portable.
//
// source: flowcontext.sql — única copia del DDL del bloque de contexto.
//
//go:embed flowcontext.sql
var flowContextSQL string

// chatEventoSQL es el DDL de las líneas de procesamiento del chat (migración
// 004), embebido igual que el esquema base para que el binario siga siendo
// portable.
//
// source: chatevento.sql — única copia del DDL del hilo de procesamiento.
//
//go:embed chatevento.sql
var chatEventoSQL string

// duracionLineaSQL es el DDL de la duración de una línea de herramienta
// (migración 005), embebido igual que el esquema base para que el binario siga
// siendo portable.
//
// source: duracionlinea.sql — única copia del DDL de la columna.
//
//go:embed duracionlinea.sql
var duracionLineaSQL string

// motorSQL es el DDL del par motor/modelo de la sesión (migración 006),
// embebido igual que el esquema base para que el binario siga siendo portable.
//
// source: motor.sql — única copia del DDL de las dos columnas.
//
//go:embed motor.sql
var motorSQL string
