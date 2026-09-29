> T-F042 — la vista recuerda el hilo de procesamiento y mide el contexto real: al cargar una sesión se pintan sub-procesos y líneas de herramienta con el tiempo guardado, y el CONTEXTO del panel muestra el total del chat (no el último turno).
> Acción de esta descomposición: `actualizar`.

## Referencias

- [[specs/SPEC-PANEL-CONTEXTO]] — qué significan el contexto del panel y el consumo del turno.
- [[specs/SPEC-HISTORIAL-CONVERSACION]] — el historial que recibe el modelo.
- [[backend/04-infrastructure/EVENTS]] — §3: `tokens_turno` con `contexto` y `limite`.
- [[database/02-rules/DATA_FLOW]] — qué se guarda del hilo.
- [[frontend/01-domain/DOMAIN]] — §1 (`chat`, `panel`) y §2 (reglas de presentación).
- [[frontend/02-interfaces/INTERFACES]] — §3 (lecturas a `session`).

## Contexto

La TUI pintaba las líneas de procesamiento solo mientras llegaban los eventos: al cambiar de sesión (`Chat.Cargar`) se perdían, y una respuesta recargada no mostraba su duración. El panel mostraba en CONTEXTO el consumo del último turno y, como `LimiteTokens` nunca se asignaba desde el motor, el porcentaje era siempre 0.

Esta tarea adapta la vista al hilo persistido: `Puerto.Historial` devuelve mensajes + líneas de procesamiento + números del contexto, `Chat.Cargar` reparte cada rol (incluido `proceso`) y recupera el tiempo, y el panel usa `ContextoTokens` para el total del chat.

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-F042-01 | actualizar | `chat`: formateadores compartidos `LineaHerramientaAbierta`/`LineaHerramientaCerrada`/`LineaProceso` (una sola forma para lo que se pinta y lo que se guarda) | completada | `internal/tui/chat.go` | `go test ./internal/tui/`; `testdata/*.golden` sin cambios |
| T-F042-02 | actualizar | `chat`: `Cargar` pinta el rol `proceso` como línea de sistema y recupera la duración guardada | completada | `internal/tui/chat.go` | `TestCargarPintaElProcesamientoYElTiempoGuardado` |
| T-F042-03 | actualizar | `panel`: `ContextoTokens` para la fila CONTEXTO y el `%` sobre `LimiteTokens`; `Tokens` queda como consumo del turno | completada | `internal/tui/panel.go`, `internal/tui/app.go`, `internal/tui/wire.go` | `TestElContextoDelPanelSonLosTokensDelChatDeLaSesion` |
| T-F042-04 | actualizar | `puerto`: `Historial` devuelve `HistorialSesion` (mensajes + contexto + límite) y `tokens_turno` alimenta el contexto del panel | completada | `internal/tui/wire.go`, `internal/tui/app.go`, `internal/tui/tui_test.go` | `go test ./internal/tui/` en verde |
| T-F042-05 | actualizar | Documentar el hilo y el contexto del panel en las fuentes de la capa | completada | `ai/docs/specs/SPEC-PANEL-CONTEXTO.md`, `ai/docs/specs/SPEC-HISTORIAL-CONVERSACION.md`, `ai/docs/backend/04-infrastructure/EVENTS.md` | `go test ./... -count=1` |

Dependencias: T-F042-01 antes de T-F042-02. T-F042-03 y T-F042-04 en paralelo. T-F042-05 revisa el resto.

## Fuera de alcance

- **No se cambia la disposición** de la vista principal (eso es T-F041).
- **No se añade un tokenizador exacto**: el contexto es una estimación, marcada como tal.
- **No se persiste nada desde la vista**: la escritura del hilo es del backend; la TUI solo pinta lo que llega.
