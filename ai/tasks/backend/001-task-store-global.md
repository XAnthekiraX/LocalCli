# 001-task-store-global.md

> T-B001 — Store global: `projects` + `sessions.project_id` (v7/007) y consultas de aislamiento.
> Acción de esta descomposición: `actualizar`.

## Referencias

- [[database/DATABASE]] — file único y global; aislamiento lógico
- [[database/01-schema/SCHEMA]] y [[database/01-schema/TABLES]] — `projects`, `sessions.project_id`
- [[database/01-schema/INDEXES]] — `idx_projects_path`, `idx_sessions_project`, `idx_sessions_status`
- [[database/01-schema/CONSTRAINTS]] — FK `sessions.project_id → projects.id` ON DELETE CASCADE
- [[database/03-operations/MIGRATIONS]] — `user_version` v7, migración `007-crear-proyectos`
- [[database/03-operations/QUERIES]] — resolver proyecto, sesiones por `project_id`, filtro obligatorio
- [[backend/DECISIONS]] — decisión 4: BD única global
- [[backend/04-infrastructure/CONFIGURATION]] §4 Rutas, §2 `LOCALCLI_DB_PATH`
- Referencia vieja: `git show HEAD:internal/store/path.go` (la DB vivía en `<proyecto>/.localcli/state.db`)

## Contexto

`store` pasa de abrir una base por proyecto (derivada de la carpeta) a abrir un único archivo global en `~/.config/localcli/state.db`. El aislamiento deja de ser físico: cada carpeta es una fila de `projects` con su `path` único y toda sesión cuelga de ella por `project_id`. Toda consulta de sesiones queda acotada por el proyecto resuelto de la carpeta abierta.

## Tareas pequeñas

| ID         | Acción | Tarea                                                                                          | Estado    | Archivos                       | Verificación                                                                          |
| ---------- | ------ | ---------------------------------------------------------------------------------------------- | --------- | ------------------------------ | ------------------------------------------------------------------------------------- |
| T-B001-01  | actualizar | Abrir la BD en la ruta global resuelta (`LOCALCLI_DB_PATH` o `~/.config/localcli/state.db`)      | completada | `internal/store/base.go`       | `store` abre el archivo global; no existe ruta por proyecto                            |
| T-B001-02  | actualizar | Esquema a v7: tabla `projects` (`id` TEXT PK, `path` UNIQUE, `created_at`, `updated_at`)         | completada | `internal/store/schema.go`     | Al crear la base nueva, queda ya en v7 con `projects`                                  |
| T-B001-03  | actualizar | `sessions.project_id`: añadir la columna NOT NULL con FK a `projects.id` ON DELETE CASCADE       | completada | `internal/store/schema.go`     | Toda sesión exige `project_id`                                                         |
| T-B001-04  | actualizar | Migración `007-crear-proyectos` en el camino de `user_version`                                   | completada | `internal/store/migraciones.go` | Una base v6 migra a v7; una base nueva nace en v7; sin importación de bases antiguas   |
| T-B001-05  | actualizar | Índices: `idx_projects_path` (UNIQUE), `idx_sessions_project` (`project_id, updated_at`), `idx_sessions_status` (`project_id, status`) | completada | `internal/store/schema.go` | Los tres índices existen con esas columnas                                             |
| T-B001-06  | crear  | `ResolucionProyecto`: resolver/crear la fila de `projects` por el `path` de la carpeta abierta   | completada | `internal/store/proyectos.go`  | Abrir dos carpetas distintas → dos filas de `projects`; reabrir una no la duplica      |
| T-B001-07  | actualizar | Consultas de sesiones acotadas por `project_id`: listar, retomar, crear, borrar del proyecto    | completada | `internal/store/sesiones.go`   | Las sesiones de un proyecto no aparecen en las consultas de otro                        |
| T-B001-08  | actualizar | Consultas de historial, aprobaciones y auditoría de contexto del proyecto actual con filtro `project_id` obligatorio | completada | `internal/store/*` | Ninguna consulta de sesión vuelve datos de otra carpeta                                 |
| T-B001-09  | crear  | Modo pruebas: base temporal aislada que se destruye al terminar                                  | completada | `internal/store`               | Los tests corren contra una base propia y no tocan la global                            |

**Dependencias:** `01`→`02`→`03`,`04`,`05`; `02`→`06`→`07`,`08`; `06`→`09`.