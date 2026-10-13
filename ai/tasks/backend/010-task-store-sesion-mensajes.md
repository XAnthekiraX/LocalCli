# 010-task-store-sesion-mensajes.md

> T-B010 — Persistencia de sesiones y mensajes (store).
> Acción de esta descomposición: `crear`.

## Referencias

- [[database/01-schema/TABLES]] §`sessions`, §`messages`, §`reasoning`
- [[database/02-rules/DATA_FLOW]] §Creación, §Actualización, §Eliminación, §Transacciones
- [[database/03-operations/QUERIES]] §Sesiones del proyecto actual, §Historial de una sesión
- [[database/01-schema/RELATIONSHIPS]] §Cascadas de borrado
- [[backend/01-domain/DOMAIN]] §3 Relaciones de Dominio
- [[backend/DECISIONS]] — razonamiento en tabla propia; ritmo de streaming cada 200 ms y al terminar
- [[specs/SPEC-HISTORIAL-CONVERSACION]] §Requisitos no funcionales — el historial sale de los mensajes ya persistidos
- [[specs/SPEC-SESIONES]] §Reglas de negocio — aislamiento por `project_id`

## Contexto

`store` es el único acceso a SQLite: esquema, WAL y transacciones. Aquí se persiste el estado de sesiones y conversación: la sesión (nombre, proyecto, capa, estado y `updated_at`), sus mensajes (`user`/`agent`, con tokens y duración nullable) y el razonamiento acumulado en tabla propia.

Dos reglas que no admiten excepción: **toda consulta de sesiones filtra por `project_id`** —el aislamiento es lógico, la base es única y global— y **no se borran filas sueltas de `messages` ni de `reasoning`**: el historial se conserva mientras exista la sesión. Al borrar una sesión, en cascada se va todo lo suyo salvo `change_history`, que sobrevive con `session_id` a `NULL`.

El esquema, las migraciones, WAL y el `project_id` ya vienen de T-B001; aquí se completan las consultas de sesión, mensaje y razonamiento que consumen `session` (T-B007) y la TUI (T-B011).

## Tareas pequeñas

| ID        | Acción | Tarea                                                                                                    | Estado    | Archivos                        | Verificación                                                             |
| --------- | ------ | -------------------------------------------------------------------------------------------------------- | --------- | ------------------------------- | ------------------------------------------------------------------------ |
| T-B010-01 | crear  | Resolver el proyecto de la carpeta abierta por `path` único, creando la fila si no existe                 | completada | `internal/store/proyectos.go`   | Abrir una carpeta nueva crea su fila y devuelve su `id`                  |
| T-B010-02 | crear  | Crear sesión (`project_id`, `layer`, `status=inactiva`) y listarlas por proyecto por actividad            | completada | `internal/store/sesiones.go`    | Listar sesiones filtra siempre por `project_id`                          |
| T-B010-03 | crear  | Obtener, renombrar y actualizar el estado de una sesión, tocando `updated_at`                             | completada | `internal/store/sesiones.go`    | Un cambio de estado actualiza `sessions.updated_at`                      |
| T-B010-04 | crear  | Eliminar una sesión en cascada, con `change_history` sobreviviendo (`session_id` a `NULL`)                | completada | `internal/store/sesiones.go`    | Borrar la sesión deja `change_history` con `session_id` NULL            |
| T-B010-05 | crear  | Crear mensaje `user`/`agent` con `input_tokens`/`output_tokens`/`duration_ms` nullable                    | completada | `internal/store/mensajes.go`    | `NULL` se distingue de `0` al leer un mensaje                            |
| T-B010-06 | crear  | Historial de una sesión en orden, con `LEFT JOIN` a `reasoning` sin perder mensajes de agente             | completada | `internal/store/mensajes.go`    | Un mensaje de agente sin razonamiento sigue apareciendo                  |
| T-B010-07 | crear  | Acumular el razonamiento en `reasoning` con límite de frecuencia (cada 200 ms y al terminar)              | completada | `internal/store/razonamiento.go` | No se inserta una fila por token: una fila por mensaje que crece        |

**Dependencias:** `01`→`02`→`03`,`04`; `02`→`05`→`06`; `05`→`07`.
**Fuera de alcance:** el esquema, las migraciones, WAL y el `project_id` ya introducidos en T-B001; la persistencia de `approvals`, `context_audit`, `chat_evento`, `todos`, `flow_context` y `change_history` (las escriben sus módulos: T-B009, T-B012, T-B017); la reconstrucción y compactación del historial (T-B007); la TUI (T-B011).
