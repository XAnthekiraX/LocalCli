> T-F039 — orden del texto y las líneas de herramienta: el chat respeta el orden de ejecución. El texto que el modelo escribió antes de una herramienta queda arriba de su línea y el que escribe después abre un globo nuevo, en vez de fusionarse en un solo globo colocado después de las líneas de herramienta.
> Acción de esta descomposición: `actualizar`.

## Referencias

- [[specs/SPEC-INTERFAZ]] — §Zonas 1 (chat) y criterios de aceptación.
- [[specs/SPEC-TOOLS]] — mientras una herramienta se ejecuta se ve cuál es.
- [[frontend/01-domain/DOMAIN]] — ficha `chat` y reglas de presentación.
- [[frontend/02-interfaces/INTERFACES]] — §1 y §1.1: los eventos y la línea de herramienta.
- [[frontend/05-quality/TESTING]] — qué se prueba del chat.
- [[backend/04-infrastructure/EVENTS]] — payloads de `herramienta_invocada` y `herramienta_resultado`.

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-F039-01 | actualizar | `chat`: `CerrarSegmento(razonamiento)` cierra el texto en curso como un intercambio del agente sin dar el turno por terminado; `CerrarTurno` vuelca solo el segmento final y cuelga la duración del último segmento del agente si el turno termina tras una herramienta | completada | `internal/tui/chat.go` | Test `TestElTextoAntesYDespuésDeLaHerramientaVanEnSuOrden`, `TestCerrarSegmentoSinNadaNoDejaGloboVacio`, `TestLaDuraciónDelTurnoQueTerminaEnHerramientaNoSePierde` |
| T-F039-02 | actualizar | `wire`/`app`: `cerrarSegmentoEnVivo` cierra texto y razonamiento antes de cada línea que se interpone (herramienta, notificación, etapas y flujos, cambios, elemento bloqueado, error) | completada | `internal/tui/wire.go`, `internal/tui/app.go` | Test `TestElChatRespetaElOrdenTextoHerramientaTexto`: el hilo sigue el orden de ejecución |
| T-F039-03 | actualizar | Tests del orden por segmentos, del razonamiento por segmento y de la duración por segmento | completada | `internal/tui/chat_test.go`, `internal/tui/herramientas_test.go` | `go test ./internal/tui/...` en verde |
| T-F039-04 | actualizar | Documentar la regla de orden en `SPEC-INTERFAZ`, `DOMAIN`, `INTERFACES` y `TESTING` | completada | `ai/docs/specs/SPEC-INTERFAZ.md`, `ai/docs/frontend/01-domain/DOMAIN.md`, `ai/docs/frontend/02-interfaces/INTERFACES.md`, `ai/docs/frontend/05-quality/TESTING.md` | Documentación coherente con el código |

Dependencias: T-F039-01 antes de T-F039-02; T-F039-03 revisa las dos; T-F039-04 revisa todas. Depende de `ai/tasks/frontend/037-task-linea-herramienta.md` (la línea compacta que ahora se intercala en el hilo).

## Notas

- La causa era que el texto del asistente se acumulaba en un único búfer en vuelo (`Chat.enCurso`) que solo se volcaba al historial al cerrar el turno, mientras las líneas de herramienta se añadían al historial de inmediato. El bus ya entrega los eventos en orden: el desorden lo introducía la vista.
- El indicador en vivo (`[⠋ Pensando]`, `[⠋ Usando herramienta: X]`, `[⠋ Generando]`) no cambia: al cerrarse el segmento, `EnCurso()` vuelve a vacío y la etiqueta pasa a «Pensando» mientras corre la herramienta.
