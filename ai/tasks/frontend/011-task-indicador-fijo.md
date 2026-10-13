# 011-task-indicador-fijo.md

> T-F011 — Línea de actividad fija sobre la caja de entrada.
> Acción de esta descomposición: `actualizar`.

## Referencias

- [[frontend/FRONTEND]] §2 Estructura del Proyecto (`app.go`, `components/chat/vista.go`)
- [[frontend/01-domain/DOMAIN]] §2 Reglas de presentación (línea de actividad fija) y §1 (componente `chat`)
- [[frontend/02-interfaces/INTERFACES]] §5 Estados de espera
- [[specs/SPEC-INTERFAZ]] §Zonas 1 y 2, §Razonamiento del modelo y criterio de aceptación

## Tareas pequeñas

| ID        | Acción    | Tarea                                                                                                   | Estado  | Archivos                                    | Verificación                                                                                              |
| --------- | --------- | ------------------------------------------------------------------------------------------------------- | ------- | ------------------------------------------- | --------------------------------------------------------------------------------------------------------- |
| T-F011-01 | actualizar | Quitar la línea de actividad del bloque en vivo del chat: `bloqueVivo` deja de añadir `Actividad(frame)` al hilo; la respuesta en vivo sigue creciendo dentro del hilo | completada | `internal/tui/components/chat/vista.go`    | El scroll del chat ya no desplaza el indicador; sin turno no queda fila fantasma                           |
| T-F011-02 | actualizar | Pintar la línea de actividad como **fila fija** en `vistaPrincipal` (`app.go`): justo encima de la caja de entrada, descontando su alto del alto del chat (`arriba`) | completada | `internal/tui/app.go`                      | Con turno activo la línea aparece junto a la caja y no se mueve al desplazar el historial; con el turno cerrado desaparece |
| T-F011-03 | actualizar | Excluir la fila fija de la zona de selección del chat: `zonaChat` termina donde empieza la línea de actividad | completada | `internal/tui/app.go`, `internal/tui/selection.go` | Un arrastre en el chat no invade la fila del indicador                                                     |
| T-F011-04 | actualizar | Salidas doradas y pruebas: el indicador fijo no se desplaza con el scroll y desaparece al cerrarse el turno | completada | `internal/tui/components/chat/*_test.go`, `internal/tui/*_test.go`, `internal/tui/testdata/*.golden` | `go test ./internal/tui/...` en verde; salidas doradas regeneradas                                          |

**Dependencias:** `01`→`02`; `02`→`03`; `01`–`03`→`04`.