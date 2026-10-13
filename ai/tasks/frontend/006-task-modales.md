# 006-task-modales.md

> T-F006 — Modales: modelos, sesiones y atajos.
> Acción de esta descomposición: `crear`.

## Referencias

- [[frontend/FRONTEND]] §2 Estructura del Proyecto (`modelsmodal.go`, `sessionsmodal.go`, `keysmodal.go`)
- [[frontend/01-domain/DOMAIN]] §1 Componentes (`modals`)
- [[frontend/02-interfaces/INTERFACES]] §4 Teclado
- [[specs/SPEC-INTERFAZ]] §Pantalla de bienvenida (modal de modelos), §Zonas
- [[specs/SPEC-INTERFAZ-ATAJOS]] §Reglas de negocio
- [[specs/SPEC-SESIONES]]
- [[frontend/05-quality/TESTING]] §2 Qué debe probarse (modales, `Esc` cierra, `Ctrl+P` solo lectura)

## Tareas pequeñas

| ID         | Acción | Tarea                                                                                        | Estado    | Archivos                        | Verificación                                                                      |
| ---------- | ------ | -------------------------------------------------------------------------------------------- | --------- | ------------------------------- | --------------------------------------------------------------------------------- |
| T-F006-01  | crear  | Estado de modales en `app`: uno abierto a la vez y apertura desde acciones                    | completada | `internal/tui/app.go`, `internal/tui/teclado.go` | Solo hay un modal activo; las acciones lo abren y cierran                        |
| T-F006-02  | crear  | Modal de modelos: modelo con lista, resaltado y modelo en uso                                 | completada | `internal/tui/components/modals/modelosmodal.go`, `internal/tui/wire.go` | El modelo guarda la lista recibida, el resaltado y el modelo en uso              |
| T-F006-03  | crear  | Modal de modelos: render centrado, `↑`/`↓` navegan, `Enter` aplica y `Esc` descarta; marca los que no declaran herramientas | completada | `internal/tui/components/modals/modelosmodal.go`, `internal/tui/components/modals/modal.go` | La lista se centra; navegar y aplicar/descartar funcionan; los modelos sin herramientas van marcados |
| T-F006-04  | crear  | Modal de sesiones: modelo con lista (nombre y estado) y resaltado                             | completada | `internal/tui/components/modals/sesionesmodal.go`, `internal/tui/wire.go` | El modelo guarda las sesiones con nombre y estado y el resaltado                 |
| T-F006-05  | crear  | Modal de sesiones: render, navegación y `Enter` que expone la sesión elegida al `app`          | completada | `internal/tui/components/modals/sesionesmodal.go`, `internal/tui/app.go` | `Enter` entrega la sesión resaltada al `app` (sin abrirla por su cuenta)         |
| T-F006-06  | crear  | Modal de sesiones: `Ctrl+D` elimina la resaltada (confirmación si está trabajando)            | completada | `internal/tui/components/modals/sesionesmodal.go`, `internal/tui/app.go` | `Ctrl+D` pide confirmación si la sesión trabaja y expone la eliminación          |
| T-F006-07  | crear  | Modal de atajos: lista de acción + tecla agrupada por categorías, solo lectura                | completada | `internal/tui/components/modals/atajosmodal.go`, `internal/tui/keymap.go` | Cada línea muestra acción y tecla, agrupadas; es de solo lectura                 |
| T-F006-08  | crear  | Integrar los modales en `app`: superposición centrada y robo de foco del teclado              | completada | `internal/tui/app.go`, `internal/tui/teclado.go` | Con un modal abierto, sus teclas no llegan a la vista de abajo                   |

**Dependencias:** `01`→`02`–`07`; `02`→`03`; `04`→`05`→`06`; `01`→`08`.
