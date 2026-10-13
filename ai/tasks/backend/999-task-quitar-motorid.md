# XXX-task-quitar-motorid.md

> T-B0XX — Eliminar columna vestigial `sessions.motor_id` (migración + store).
> Acción de esta descomposición: `actualizar`.

## Referencias

- [[specs/SPEC-MODELOS-LLAMACPP]] — proveedor único, no hay motor por sesión
- [[specs/SPEC-SESIONES]] — el modelo es un recurso de la sesión (`modelo`); el motor no se elige por sesión
- [[database/01-schema/TABLES]] — descripción actual de `sessions` (motor_id/modelo)
- [[database/01-schema/CONSTRAINTS]] — restricciones sobre `motor_id`
- [[database/01-schema/SCHEMA]] — esquema completo
- [[database/03-operations/MIGRATIONS]] — migraciones
- [[backend/DECISIONS]] — decisiones 86-88 (proveedor único)

## Contexto

Con el cambio a un único proveedor (`llama.cpp`), el motor no se elige por sesión. La columna `sessions.motor_id` existe físicamente en el esquema y código, pero nunca se rellena (siempre NULL) y DECISIONS 86/87 + SPEC-SESIONES establecen que el motor es único. La documentación de BD debe reflejar el objetivo (solo `modelo`), por lo que hay que eliminar la columna vestigial.

Esto implica: migración para eliminar `motor_id` (con su tipo/constraints), actualizar `store/schema.go`, `store/migraciones.go`, `store/sesiones.go` (lectura/escritura/Scan), `store/sesiones_test.go` (si procede), y actualizar las docs de BD (TABLES.md, CONSTRAINTS.md, SCHEMA.md) para describir solo `sessions.modelo`.

## Tareas pequeñas

| ID           | Acción     | Tarea                                                                 | Estado    | Archivos                                                 | Verificación                                             |
| ------------ | ---------- | --------------------------------------------------------------------- | --------- | -------------------------------------------------------- | -------------------------------------------------------- |
| T-B0XX-01    | actualizar | Revisar impacto: listar todas las lecturas/escrituras de `motor_id`    | pendiente | `internal/store/*.go`, `internal/store/migraciones.go`, `internal/store/schema.go` | Lista completa de usos                                  |
| T-B0XX-02    | actualizar | Migración: eliminar columna `motor_id` de `sessions` (v8 o siguiente)  | pendiente | `internal/store/migraciones.go`, `internal/store/schema.go` | user_version actualizado                                 |
| T-B0XX-03    | actualizar | `sesiones.go`: dejar de leer/escribir `motor_id`; limpiar `MotorID`    | pendiente | `internal/store/sesiones.go`                             | Compila y pasa tests                                    |
| T-B0XX-04    | actualizar | Pruebas relacionadas con `sessions`                                    | pendiente | `internal/store/sesiones_test.go`                        | Tests pasan                                              |
| T-B0XX-05    | actualizar | Docs BD: TABLES.md, CONSTRAINTS.md, SCHEMA.md                          | pendiente | `ai/docs/database/01-schema/{TABLES,CONSTRAINTS,SCHEMA}.md` | Docs coherentes con objetivo (solo modelo)               |

**Dependencias:** ninguna (actualización de esquema/doc).

**Fuera de alcance:** cambiar la lógica de selección de modelo (ya vive en `modelo`), crear lógica de motor-per-sesión.
