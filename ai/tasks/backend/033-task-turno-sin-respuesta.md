> T-B033 — un turno nunca se cierra mudo: la redacción final no se queda sin comprobar, se reintenta una vez y, si el modelo sigue sin entregar texto, el turno falla con el motivo a la vista en vez de guardar `(respuesta vacía)`.
> Acción de esta descomposición: `actualizar`.

## Referencias

- [[specs/SPEC-AGENTE-BASE]] — §El ciclo de un turno: qué es la pasada de redacción y qué pasa si vuelve a pedir herramientas.
- [[specs/SPEC-TOOLS]] — §Cómo se le pide una herramienta al modelo: al agotar las pasadas, el turno avisa y nunca termina en silencio.
- [[specs/SPEC-INTERFAZ]] — §El chat: un turno que no entrega texto se ve, no se guarda como respuesta.
- [[backend/05-quality/ERRORS]] — el error del turno sin respuesta, reconocible con `errors.Is`.
- [[backend/01-domain/BUSINESS_RULES]] — §Sesiones: el cierre de un turno de chat no se anuncia como trabajo.
- [[database/02-rules/DATA_FLOW]] — §7: un turno agotado redacta y, si no puede, falla.

## Contexto

`internal/agent/loop.go` ya cierra los turnos agotados con una **pasada de redacción sin herramientas** (T-B031): en vez de devolver el preámbulo que acompañaba a la última petición, obliga al modelo a escribir su resultado. Esa pasada está bien, pero su resultado se descarta:

```go
texto, razon, _, err := e.unaPasada(ctx, a, modelo, mensajes, nil, numCtx, pensar, sink, &tokensIn, &tokensOut)
```

El `_` son los `pedidos` de esa pasada. Si el modelo contesta con otra petición de herramienta —que es justo lo que hace un modelo pequeño que no entiende que se le está pidiendo prosa—, sus peticiones se descartan en silencio y el turno devuelve un `Resultado` **vacío y exitoso**: `Texto == ""`, `err == nil`.

De ahí el síntoma que se vio: `arranque.ejecutorPorTurno` convierte ese texto vacío en el marcador `(respuesta vacía)` y lo **persiste como si fuera la respuesta del agente** (el chat no es una etapa de flujo, así que no cae en el `etapaMuda` que evita el mensaje de relleno). La base muestra un turno de 156 s con 7 416 tokens de entrada, 737 de salida y razonamiento en `reasoning`, y un mensaje de dos palabras que el modelo nunca escribió. El `Resultado` de `agent` no distingue «no dijo nada» de «dijo poco»: por eso el fallo llega hasta el historial.

El reintento de una etapa muda que ya existe es de otra capa y no cubre este caso: `flow.ejecutarEtapa` reintenta una vez y falla con `E_STAGE_FAILED`, pero solo se aplica a `p.Etapa != ""` (una etapa de flujo). El chat normal no pasa por ahí.

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-B033-01 | actualizar | `agent`: la pasada de redacción deja de descartar los `pedidos`; si el modelo vuelve a pedir herramientas, se le responde con una instrucción explícita de redactar en prosa y se reintenta **una sola vez** | completada | `internal/agent/loop.go` | `TestUnaRedaccionQueVuelveAPedirHerramientasSeReintenta`, `TestLaRedaccionNoRepiteElReintento` en verde |
| T-B033-02 | actualizar | `agent`: si el reintento tampoco entrega texto, el turno devuelve un error distinguible (`ErrSinRespuesta`) en vez de un `Resultado` vacío exitoso; el aviso dice que se agotaron las rondas y que el modelo no entregó nada | completada | `internal/agent/loop.go`, `internal/agent/errors.go` (nuevo, siguiendo `internal/tools/errors.go`) | `TestUnTurnoSinTextoFinalFalla` en verde |
| T-B033-03 | actualizar | `arranque`: `ejecutorPorTurno` traduce el error según el tipo de turno — una etapa de flujo (`p.Etapa != ""`) vuelve a texto vacío y deja que el motor la reintente como hasta ahora; el chat deja de persistir `(respuesta vacía)` y propaga el error, con lo que la sesión queda en `error` y el motivo se ve | completada | `arranque.go` | `TestElChatNoGuardaRespuestaVacia`, `TestUnaEtapaMudaSeReintentaYfallaComoAntes` en verde |
| T-B033-04 | actualizar | Documentar el error del turno sin respuesta en `ERRORS.md` (tabla y reconocible con `errors.Is`), junto a `E_STAGE_FAILED` | completada | `ai/docs/backend/05-quality/ERRORS.md` | Documentación coherente con el código |
| T-B033-05 | actualizar | tests: el bucle completo con un modelo que solo pide herramientas (redacción → reintento → error), el chat que ya no persiste el marcador, y la etapa muda de flujo que conserva su semántica | completada | `internal/agent/loop_test.go`, `arranque_test.go` (`internal/flow/engine_test.go` ya cubría el reintento de la etapa muda: sin cambios) | `go test ./internal/agent/ ./internal/flow/ . -count=1` en verde |

Dependencias: T-B033-01 antes de T-B033-02 (sin el reintento, el error salta en la primera redacción). T-B033-02 antes de T-B033-03. T-B033-04 y T-B033-05 revisan las tres.

## Fuera de alcance

- **No se cambia la regla de las etapas mudas de flujo.** `flow.ejecutarEtapa` sigue reintentando una vez y fallando con `E_STAGE_FAILED`; el error nuevo solo sustituye al marcador `(respuesta vacía)` del chat.
- **No se sube `pasadasPorDefecto`.** El fallo no era que el turno gastara tres rondas, sino que su cierre no comprobara si había entrega.
- **No se reintenta un turno que sí entregó texto.** El reintento es solo para el caso «redactó, y volvió a pedir herramientas»; si hubo texto, el turno se cierra.
- **No se reintenta más de una vez.** El segundo fallo es visible y definitivo, para que un modelo atascado no consuma el doble de tokens callando igual.
- **No se toca el silencio de las etapas intermedias de un flujo** (`p.Silenciosa`): sin sink siguen sin verse, y sin persistir, que es lo que hace que su reintento no deje mensajes de relleno.
- **`(respuesta vacía)` no se borra del código como texto.** Sigue siendo el resumen de una etapa muda de flujo, que es un caso distinto del que resuelve esta tarea.
