> T-F036 — chat: cada mensaje se pinta en un globo con el color de quien habla (usuario vs. agente) y las líneas del sistema quedan sueltas.
> Acción de esta descomposición: `actualizar`.

## Referencias

- [[specs/SPEC-INTERFAZ]] — §Zonas 1 (chat) y §Criterios de aceptación.
- [[frontend/01-domain/DOMAIN]] — ficha `chat` y reglas de presentación.
- [[frontend/02-interfaces/INTERFACES]] — §5 (estados de espera).
- [[frontend/05-quality/TESTING]] — qué se prueba del chat.

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-F036-01 | actualizar | `styles`: estilos de globo (borde redondeado y color por rol), helper `burbuja` y `anchoGlobo` que reserva borde y relleno | completada | `internal/tui/styles.go` | Tests `TestLosGlobosDistinguenAlUsuarioDelAgente`, `TestElAnchoDelGloboReservaBordeYRelleno` |
| T-F036-02 | actualizar | `chat`: el historial pinta cada mensaje en su globo, el sistema va suelto y la respuesta en vivo también en globo | completada | `internal/tui/chat.go`, `internal/tui/app.go` | Test `TestElChatPintaCadaMensajeEnSuGlobo` |
| T-F036-03 | actualizar | documentar los globos en `SPEC-INTERFAZ`, `DOMAIN`, `INTERFACES` y `TESTING` | completada | `ai/docs/specs/SPEC-INTERFAZ.md`, `ai/docs/frontend/01-domain/DOMAIN.md`, `ai/docs/frontend/02-interfaces/INTERFACES.md`, `ai/docs/frontend/05-quality/TESTING.md` | Documentación coherente con el código |

Dependencias: T-F036-01 antes de T-F036-02; T-F036-03 revisa las dos.

## Notas

- Colores: azul (12) para el usuario y verde (10) para el agente, los mismos que ya distinguían a los interlocutores antes de los globos.
- Las líneas del sistema (`[→ herramienta: X]`, avisos) no son un turno: se pintan sueltas, sin globo.
- El ancho interior del globo es `ancho-4` (borde y relleno); el contenido se envuelve a esa medida antes de pintarse, así el globo nunca desborda la terminal.
