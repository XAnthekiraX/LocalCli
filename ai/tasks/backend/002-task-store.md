# 002-task-store.md

> T-B002 — store: SQLite único acceso (esquema 6 tablas, WAL, foreign_keys, user_version, transacciones).
> Acción de esta descomposición: `crear`.

## Referencias

- `ai/docs/database/01-schema/SCHEMA.md` [3-110] (estructura global y campos de las 6 tablas)
- `ai/docs/database/01-schema/TABLES.md` (definición completa de cada tabla)
- `ai/docs/database/01-schema/CONSTRAINTS.md` (claves foráneas y checks)
- `ai/docs/database/01-schema/INDEXES.md` (índices del esquema)
- `ai/docs/database/03-operations/MIGRATIONS.md` [3-46] (user_version y estrategia)
- `ai/docs/database/03-operations/QUERIES.md` [62-96] (patrones de acceso y consultas reutilizables)
- `ai/docs/backend/DECISIONS.md` [12-14, 23] (sin ORM, único punto de escritura, foreign_keys ON, .localcli/state.db)

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-B002-01 | crear | Abrir conexión SQLite vía modernc.org/sqlite con PRAGMA journal_mode=WAL y foreign_keys=ON en cada conexión | pendiente | `internal/store/db.go` | Test: pragmas activos tras abrir (`PRAGMA journal_mode` devuelve wal) |
| T-B002-02 | crear | Derivar la ruta de la base `.localcli/state.db` desde la carpeta del proyecto y respetar `LOCALCLI_DB_PATH` | pendiente | `internal/store/path.go` | Test: con y sin variable resuelve la ruta esperada |
| T-B002-03 | crear | Escribir el DDL del esquema v1: las 6 tablas (sessions, messages, reasoning, approvals, context_audit, change_history) con sus constraints e índices | pendiente | `internal/store/schema.sql`, `internal/store/schema.go` | Test: `sqlitediff`/PRAGMA table_info coincide con SCHEMA.md |
| T-B002-04 | crear | Implementar la migración por `user_version`: aplicar DDL en versión 0 y subir la versión | pendiente | `internal/store/migrate.go` | Test: base nueva queda en user_version esperado; segunda apertura es idempotente |
| T-B002-05 | crear | Implementar repositorio sessions (crear, listar por carpeta/estado, actualizar estado, borrar en cascada) | pendiente | `internal/store/sessions.go` | Test CRUD: borrar sesión elimina messages/reasoning/approvals/context_audit |
| T-B002-06 | crear | Implementar repositorio messages + reasoning (insertar mensaje inmutable, upsert razonamiento con límite 200ms) | pendiente | `internal/store/messages.go`, `internal/store/reasoning.go` | Test: razonamiento ligado a mensaje, máximo uno por mensaje |
| T-B002-07 | crear | Implementar repositorio approvals (pendiente → aprobada/declinada) | pendiente | `internal/store/approvals.go` | Test: consulta de pendientes devuelve solo estado pendiente |
| T-B002-08 | crear | Implementar repositorio context_audit (una fila por documento y etapa, con motivo) | pendiente | `internal/store/contextaudit.go` | Test: inserción y lectura por sesión+etapa |
| T-B002-09 | crear | Implementar repositorio change_history con session_id nullable que sobrevive al borrado de sesión | pendiente | `internal/store/changehistory.go` | Test: borrar sesión deja la fila con session_id NULL |
| T-B002-10 | crear | Envolver las escrituras multi-tabla en transacciones con rollback ante error | pendiente | `internal/store/tx.go` | Test: error a mitad revierte todas las filas |
| T-B002-11 | crear | Mapear errores de driver a los códigos internos de ERRORES.md | pendiente | `internal/store/errors.go` | Test: constraint roto produce el código documentado |
| T-B002-12 | crear | Escribir tests de store con base temporal aislada (modo Pruebas de CONFIGURATION.md) | pendiente | `internal/store/*_test.go` | `go test ./internal/store/...` pasa y destruye la base al terminar |

**Dependencias:** 01→(02,03); 03→04; 04→05..09; {05}→06,07,08,09; {05..09}→10; 10→11; 11→12. Requiere T-B000 completada.
