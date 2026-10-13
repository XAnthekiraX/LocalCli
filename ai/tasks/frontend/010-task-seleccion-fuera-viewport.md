# 010-task-seleccion-fuera-viewport.md

> T-F010 — Selección con ratón que rebasa el viewport.
> Acción de esta descomposición: `actualizar`.

## Referencias

- [[frontend/FRONTEND]] §2 Estructura del Proyecto (`selection.go`, `components/chat/vista.go`), §4 Decisiones (ratón capturado en la TUI)
- [[frontend/01-domain/DOMAIN]] §2 Reglas de presentación (selección que rebasa el viewport)
- [[frontend/02-interfaces/INTERFACES]] §4 Ratón
- [[specs/SPEC-INTERFAZ]] §Reglas de negocio (ratón y copia) y criterio de aceptación
- [[frontend/05-quality/TESTING]] §2 Qué debe probarse (selección por zona, anclada y copia)

## Tareas pequeñas

| ID        | Acción    | Tarea                                                                                          | Estado  | Archivos                              | Verificación                                                                                                                                    |
| --------- | --------- | ---------------------------------------------------------------------------------------------- | ------- | ------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------- |
| T-F010-01 | actualizar | Traducción contenido↔ventana en `chat.Model`: qué línea del hilo está bajo una fila de la ventana `Vista` y qué fila de ventana ocupa una línea del hilo, con el `offset` del scroll | completada | `internal/tui/components/chat/vista.go` | Los ayudantes son puros y acotan fuera de rango; se prueban con tallas y offsets distintos (tras `Subir`/`Bajar`)                        |
| T-F010-02 | actualizar | La selección del chat se rastrea en **coordenadas de contenido** (línea del hilo, columna); la caja y el sidebar conservan el ancla de marco actual | completada | `internal/tui/selection.go`         | Un arrastre en el chat ancla y extiende en líneas de contenido; la selección en caja/sidebar no cambia de comportamiento               |
| T-F010-03 | actualizar | Al iniciar un arrastre en el chat se suspende el seguimiento del final (`scrolleado`) hasta soltar | completada | `internal/tui/selection.go`, `internal/tui/app.go` | Mientras se arrastra, un token nuevo no mueve la vista; al soltar, el seguimiento vuelve a su estado anterior                         |
| T-F010-04 | actualizar | Auto-scroll al arrastrar contra los bordes de `zonaChat` (margen de 2 filas): `Subir(1)`/`Bajar(1)` mientras el puntero esté en el margen y la selección se prolonga por las líneas que entran | completada | `internal/tui/selection.go`         | Un arrastre que llega a un borde desplaza el historial y abarca líneas que estaban fuera del viewport; el realce no sale de la zona del chat |
| T-F010-05 | actualizar | Realce y copia resueltos desde el contenido del chat: `resolver` mapea contenido→ventana (solo realza lo visible) y la copia extrae el tramo completo del hilo, incluidas las líneas fuera del viewport; se mantiene la copia al soltar (herramientas del sistema u OSC 52) y el aviso `[Copiado]` | completada | `internal/tui/selection.go`, `internal/tui/selection_test.go` | El tramo copiado incluye las líneas fuera del viewport; el realce no cruza de zona; `[Copiado]` aparece y se apaga solo          |
| T-F010-06 | actualizar | Pruebas de la selección fuera del viewport: traducción contenido↔ventana, suspensión del seguimiento, auto-scroll en bordes, no mezcla de zonas y copia del tramo completo | completada | `internal/tui/selection_test.go`    | `go test ./internal/tui/...` en verde; las pruebas de T-F008 siguen pasando                                              |

**Dependencias:** `01`→`02`; `02`,`03`→`04`; `04`→`05`; `02`–`05`→`06`.