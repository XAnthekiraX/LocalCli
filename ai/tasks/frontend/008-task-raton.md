# 008-task-raton.md

> T-F008 — Selección con ratón y copia al portapapeles.
> Acción de esta descomposición: `crear`.

## Referencias

- [[frontend/FRONTEND]] §2 Estructura del Proyecto (`selection.go`), §4 Decisiones (ratón capturado en la TUI)
- [[frontend/01-domain/DOMAIN]] §2 Reglas de presentación (selección acotada y anclada, copia)
- [[frontend/02-interfaces/INTERFACES]] §4 Ratón
- [[specs/SPEC-INTERFAZ]] §Ratón y copia (criterio de aceptación)
- [[frontend/05-quality/TESTING]] §2 Qué debe probarse (selección por zona, anclada y copia)

## Tareas pequeñas

| ID        | Acción | Tarea                                                                                       | Estado    | Archivos                              | Verificación                                                                     |
| --------- | ------ | ------------------------------------------------------------------------------------------- | --------- | ------------------------------------- | -------------------------------------------------------------------------------- |
| T-F008-01 | crear  | Captura del ratón en pantalla completa y desplazamiento del historial con la rueda           | completada | `internal/tui/arranque.go`, `internal/tui/selection.go` | La rueda recorre el historial del chat; con el ratón capturado la selección nativa queda con `Shift` |
| T-F008-02 | crear  | Modelo del arrastre: inicio/fin, instantánea del marco («ancla») y acotado a la zona de origen | completada | `internal/tui/selection.go`         | El arrastre guarda el ancla y se acota a su zona desde el primer punto           |
| T-F008-03 | crear  | Zonas de selección —caja de entrada, chat y sidebar— registradas en cada repintado             | completada | `internal/tui/selection.go`, `internal/tui/app.go` | Un punto dentro de una zona acota el arrastre; fuera de las tres no lo acota    |
| T-F008-04 | crear  | Anclaje al texto: el realce sigue al texto elegido aunque el marco se repinte o se desplace    | completada | `internal/tui/selection.go`         | Un repintado no cancela ni mueve el realce a otro texto                          |
| T-F008-05 | crear  | Realce en video inverso del tramo seleccionado mientras se arrastra                            | completada | `internal/tui/app.go`               | El tramo elegido se pinta en video inverso mientras se arrastra                  |
| T-F008-06 | crear  | Copia al soltar (herramientas del sistema y, si no hay ninguna, OSC 52) y apagado del realce   | completada | `internal/tui/selection.go`         | Al soltar se copia el texto elegido y el realce desaparece                       |
| T-F008-07 | crear  | Aviso transitorio `[Copiado]` arriba a la derecha, que se apaga solo                           | completada | `internal/tui/selection.go`, `internal/tui/app.go` | Tras copiar aparece `[Copiado]` y se apaga solo al cabo de un instante          |
| T-F008-08 | crear  | Pruebas de la selección: límites de la caja, no mezcla de zonas, anclaje y copia al soltar      | completada | `internal/tui/selection_test.go`    | Las pruebas cubren acotado por zona, anclaje ante repintado y copia al soltar   |

**Dependencias:** `01`→`02`; `02`,`03`→`04`,`05`; `04`,`05`→`06`; `06`→`07`; `02`–`07`→`08`.
