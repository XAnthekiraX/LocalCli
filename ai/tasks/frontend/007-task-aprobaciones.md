# 007-task-aprobaciones.md

> T-F007 — Panel de aprobaciones y aviso de pendientes.
> Acción de esta descomposición: `crear`.

## Referencias

- [[frontend/FRONTEND]] §2 Estructura del Proyecto (`approvals.go`, `notify.go`), §3 Contratos
- [[frontend/01-domain/DOMAIN]] §1 Componentes (`approvals`, `notify`), §2 Reglas de presentación
- [[frontend/02-interfaces/INTERFACES]] §1 Eventos que consume, §2 Peticiones que envía, §4 Teclado, §5 Estados de espera
- [[specs/SPEC-INTERFAZ-ATAJOS]] §Flujo principal — panel de aprobaciones, §Formato de una línea del panel, §Reglas de negocio
- [[specs/SPEC-KEYBINDS]] §Modelo de datos (Acción: `approvals_toggle`, `approve`/`decline`)
- [[specs/SPEC-INTERFAZ]] §Zonas, §Aviso de pendientes
- [[frontend/05-quality/TESTING]] §2 Qué debe probarse (contador con panel cerrado, panel de aprobaciones)

## Tareas pequeñas

| ID        | Acción | Tarea                                                                                                             | Estado    | Archivos                        | Verificación                                                                             |
| --------- | ------ | ---------------------------------------------------------------------------------------------------------------- | --------- | ------------------------------- | ---------------------------------------------------------------------------------------- |
| T-F007-01 | crear  | Modelo del panel: pendientes de todas las sesiones (identificador, acción propuesta) y línea resaltada            | completada | `internal/tui/approvals.go`     | El modelo guarda cada pendiente con su sesión y acción, y cuál está resaltado            |
| T-F007-02 | crear  | Formato de una línea como función pura: `sesión \| acción \| aprobar \| declinar`, sesión abreviada a ocho y carpeta del usuario a `~` | completada | `internal/tui/approvals.go` | El formateador compone la línea con la sesión abreviada y `~`; no reserva hueco para la tercera opción |
| T-F007-03 | crear  | Render del panel: lista de pendientes; se cierra solo cuando no queda ninguna                                     | completada | `internal/tui/approvals.go`     | Con pendientes se pinta la lista; sin pendientes el panel desaparece sin dejar hueco     |
| T-F007-04 | crear  | Alta y baja por evento: `peticion_aprobacion` añade la línea y `aprobacion_resuelta` la retira; la sesión terminada la marca obsoleta | completada | `internal/tui/approvals.go` | Un pendiente aparece, se retira al resolverse y se marca obsoleto si la sesión terminó   |
| T-F007-05 | crear  | Mover el resaltado dentro del panel con el teclado cuando el panel tiene el foco                                   | completada | `internal/tui/approvals.go`     | Con el foco, arriba/abajo mueven el resaltado sin salirse de la lista                    |
| T-F007-06 | crear  | Foco del panel: `Ctrl+A` da el teclado y `a`/`d` resuelven la línea resaltada, sin que el input deje de escribir   | completada | `internal/tui/app.go`, `internal/tui/teclado.go` | Con foco `a`/`d` resuelven; sin foco el input sigue escribiendo     |
| T-F007-07 | crear  | Resolución hacia `session`: el `Puerto` expone aprobar/declinar con el identificador y la decisión                 | completada | `internal/tui/wire.go`          | La decisión viaja a `session` con el id y solo afecta a la sesión de esa línea           |
| T-F007-08 | crear  | Clic del ratón: sobre «aprobar»/«declinar» de una línea resuelve esa aprobación; un clic fuera no decide nada      | completada | `internal/tui/selection.go`     | El clic sobre la palabra resuelve; el clic fuera no decide                               |
| T-F007-09 | crear  | Aviso de pendientes con el panel cerrado (`notify`): número visible y actualizado con cada evento                  | completada | `internal/tui/notify.go`        | El contador se ve en el sidebar ESTADO y se actualiza con cada evento                    |
| T-F007-10 | crear  | Integrar panel y aviso en `app` sobre la vista, sin secuestrar el teclado ni interrumpir sesiones                  | completada | `internal/tui/app.go`           | El panel se pinta encima de la caja con pendientes; sin foco no captura el teclado        |

**Dependencias:** `01`→`02`,`03`,`04`,`05`,`09`; `03`→`08`; `05`→`06`; `06`→`07`; `03`,`06`,`09`→`10`.
