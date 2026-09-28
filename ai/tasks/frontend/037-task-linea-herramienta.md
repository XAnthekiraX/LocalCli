> T-F037 — línea de herramienta compacta: el chat pinta una sola línea que nace al invocar con el verbo y el tema (la ruta que se busca) y se completa al terminar con la marca, la medida del resultado y, si se recortó, el aviso. Sin agente y sin volcar la salida.
> Acción de esta descomposición: `actualizar`.

## Referencias

- [[specs/SPEC-INTERFAZ]] — §Zonas 1 (chat) y criterios de aceptación.
- [[specs/SPEC-TOOLS]] — mientras una herramienta se ejecuta se ve cuál es.
- [[frontend/01-domain/DOMAIN]] — ficha `chat` y reglas de presentación.
- [[frontend/02-interfaces/INTERFACES]] — §1 y §1.1: los eventos y la línea de herramienta.
- [[frontend/05-quality/TESTING]] — qué se prueba del chat.
- [[backend/04-infrastructure/EVENTS]] — payloads de `herramienta_invocada` y `herramienta_resultado` (los fija `ai/tasks/backend/026-task-linea-herramienta.md`).

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-F037-01 | actualizar | `chat`: una sola línea que se completa —`AnotarInvocacion(verbo, tema)` abre y `CerrarHerramienta` reescribe la misma con marca y medida— con una cola FIFO de líneas abiertas | completada | `internal/tui/chat.go` | Test `TestLosEventosDeHerramientaSePintan`: una sola línea, sin agente |
| T-F037-02 | actualizar | `wire`: reaccionar a `verbo`/`tema` al invocar y a `medida` al cerrar | completada | `internal/tui/wire.go` | El chat recibe verbo, tema y medida del bus |
| T-F037-03 | actualizar | Tests: `herramientas_test.go` al formato nuevo (línea única, sin `(agente …)`, con medida y recorte) | completada | `internal/tui/herramientas_test.go` | `go test ./internal/tui/...` en verde |
| T-F037-04 | actualizar | Documentar la línea en `INTERFACES` §1.1, `DOMAIN` y `TESTING` | completada | `ai/docs/frontend/02-interfaces/INTERFACES.md`, `ai/docs/frontend/01-domain/DOMAIN.md`, `ai/docs/frontend/05-quality/TESTING.md` | Documentación coherente con el código |

Dependencias: T-F037-01 antes de T-F037-02; T-F037-03 revisa las dos; T-F037-04 revisa todas. Depende de `ai/tasks/backend/026-task-linea-herramienta.md` (el payload nuevo).

## Notas

- La línea del sistema se pinta como `· ` + texto (`chat.go`, `Render`): `· LEER [internal/tui/chat.go]` y, al cerrar, `· ✓ LEER [internal/tui/chat.go] · 70 líneas`.
- El indicador en vivo (`[⠋ Usando herramienta: X]`) no cambia: sigue nombrando la herramienta por su nombre.
