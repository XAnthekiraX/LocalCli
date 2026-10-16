# 015-task-diagrama.md

> T-F015 — Bloque de diagrama: pintar el diagrama Unicode bajo la línea de herramienta `diagram` en el chat.
> Acción de esta descomposición: `crear`.

## Referencias

- [[frontend/02-interfaces/INTERFACES]] §1.1 «La línea de herramienta», apartado «El bloque de diagrama».
- [[backend/04-infrastructure/EVENTS]] §3 `herramienta_resultado` — campo opcional `bloque` solo para `diagram`.
- [[backend/02-interfaces/TOOLS]] §1 «Diagramas», §1.1 tabla TUI (verbo `DIAGRAMAR`, tema `code`, unidad `diagrama`).
- [[specs/SPEC-DIAGRAMA]] §Presentación en la conversación.
- [[specs/SPEC-INTERFAZ]] §Zonas 1 (chat) — integración al hilo.
- [[specs/SPEC-VISTA-CODIGO]] (referencia para patrón de bloque renderizado).
- [[database/02-rules/DATA_FLOW]] §Creación — persistencia de líneas con duración y recuperación.

## Contexto

Cuando `herramienta_resultado` llega con `herramienta = "diagram"` y `bloque` no vacío, la TUI debe pintar el diagrama Unicode justo debajo de la línea de herramienta (`DIAGRAMAR [...] · medida · tiempo`), como bloque adicional (no sustituto). Es la única excepción a «la salida, nunca» y el bloque ya viene renderizado desde backend —la TUI no interpreta Mermaid ni hace layout.

El bloque debe respetar el ancho de columna real de la terminal (se confía al texto renderizado), aparecer en orden cronológico (después de cerrar el globo de texto anterior y antes del siguiente), mantenerse al recuperar la sesión (persistido con la línea), y no mezclar zonas al seleccionar con ratón.

## Tareas pequeñas

| ID         | Acción | Tarea                                                                 | Estado    | Archivos                                                                                  | Verificación                                                                                        |
| ---------- | ------ | --------------------------------------------------------------------- | --------- | ----------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------- |
| T-F015-01  | crear  | Extender el modelo de evento/tipo de herramienta resultado en la capa TUI (wire/events) para aceptar `Bloque` opcional cuando el nombre es `diagram`. No romper otros campos. | completada | `internal/tui/wire.go`, `internal/tui/eventos.go` o tipos usados por eventos              | Otros resultados siguen sin bloque; `diagram` puede recibirlo.                                       |
| T-F015-02  | crear  | Al cerrar la línea de herramienta en `herramienta_resultado`, si es `diagram` y hay bloque, renderizar debajo en el componente de chat: insertar un bloque visual separado del globo de texto (espaciado razonable, sin sangría excesiva). | completada | `internal/tui/components/chat/*` (vista/chat/formato/lineas)                              | La línea permanece y el bloque aparece debajo, en el mismo hilo cronológico.                         |
| T-F015-03  | crear  | Asegurar que el bloque no rompe la selección con ratón: el bloque se comporta como parte del mismo mensaje de herramienta (no inicia zona distinta), y el scrolling sigue funcionando. Seguir el patrón de bloques existentes (código). | completada | `internal/tui/components/chat/vista.go`, `internal/tui/raton*.go` según corresponda        | Selección con ratón limitada a la zona; copiar sigue operativo; arrastre a bordes desplaza.           |
| T-F015-04  | crear  | Persistencia/recuperación: al cargar historial, si la línea guardada corresponde a `diagram` con bloque asociado, este se pinta al reabrir sesión (coincide con duración/medida). | completada | `internal/tui/components/chat/*`, recuperación desde store (eventos/chat)                  | El bloque aparece al volver a la sesión, igual que se vio en vivo.                                   |
| T-F015-05  | crear  | Pruebas doradas o comprobación visual de estructura: que el bloque no sustituye la línea; la línea conserva verbo `DIAGRAMAR`, tema `code`, medida y tiempo. | completada | `internal/tui/*_test.go`, golden files existentes                                           | No rompe golden existentes; estructura del hilo correcta.                                              |

## Dependencias
`T-F015-01→02→03→04`; `T-F015-05` valida integración.

## Notas

- El bloque ya viene en Unicode con ancho calculado: **no** re-renderizar, **pintar tal cual**. 
- Medida: el evento `herramienta_resultado` llevará la medida declarada por backend ("X líneas" del bloque) y la unidad acorde al catálogo (tabla TUI usa `diagrama` como tema/tipo presentación).
- Si bloque está vacío (no debería ocurrir), simplemente no pintar bloque extra.
- Mantener separación entre "resultado va al modelo" (backend ya lo hace) y "bloque va a pantalla" (sólo TUI).