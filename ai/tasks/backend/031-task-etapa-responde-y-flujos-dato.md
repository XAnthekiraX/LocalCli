> T-B031 — la etapa responde a su pregunta y los flujos son dato: el brief de cada etapa lleva su `pregunta` y la ventana empareja pregunta y respuesta, una etapa que no responde se reintenta una vez y falla con `E_STAGE_FAILED`, y se eliminan los flujos cableados en el motor (`.localcli/flows/*.json` es la única fuente).
> Acción de esta descomposición: `actualizar`.

## Referencias

- [[specs/SPEC-MOTOR-FLUJOS]] — §Ventana de contexto, §Visibilidad de una respuesta y §Reglas de negocio.
- [[specs/SPEC-FLUJO-PERSONALIZADO]] — los flujos del proyecto son los archivos de `.localcli/flows/`: no hay flujos cableados.
- [[specs/SPEC-RESOLVER]] — `/resolver` con `respuesta_en_chat` en sus nueve pasos.
- [[specs/SPEC-COLA-TAREAS]] — el flujo con el que corre cada elemento de la cola.
- [[backend/01-domain/BUSINESS_RULES]] — §Motor de etapas.
- [[backend/05-quality/ERRORS]] — `E_STAGE_FAILED` de una etapa que no responde.

## Contexto

El motor nunca le entregaba a la etapa su `pregunta`: el brief se armaba con `Reglas + Instruccion + contexto`. La `pregunta` se parseaba y `Validar` la exigía, pero no llegaba al modelo ni entraba en la ventana. Una etapa podía cerrar el turno en blanco, encadenarse como `(sin resultado)` y pasar por buena, sin alimentar a la siguiente.

Además, el motor todavía cableaba los flujos oficiales (`FlujoResolver`, `FlujoPlanificacion`, `FlujoTrabajo`) como respaldo, y la cola usaba `FlujoTrabajo(AccionCrear)` como flujo por defecto. La spec dice que los flujos son los archivos de `.localcli/flows/*.json` y que no hay flujos cableados.

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-B031-01 | actualizar | `flow`: el brief de la etapa lleva su `pregunta`; la ventana (encadenado y bloque) compone `Pregunta:`/`Respuesta:` por etapa; una etapa muda se reintenta una vez y falla con `E_STAGE_FAILED` | completada | `internal/flow/engine.go` | `TestElBriefDeLaEtapaLlevaSuPregunta`, `TestLaVentanaLlevaPreguntaYRespuesta`, `TestUnaEtapaSinRespuestaSeReintentaYFalla`, `TestUnaEtapaQueRespondeTrasElReintento` en verde |
| T-B031-02 | actualizar | `arranque`: `ejecutorPorTurno` devuelve el texto crudo en `Resultado.Texto` (para que el motor vea el vacío) y no persiste el turno de una etapa de flujo muda | completada | `arranque.go` | `go build ./...`; el chat conserva `(respuesta vacía)` |
| T-B031-03 | eliminar | `flow`: fuera `resolver.go`, `plan.go`, `work.go` y el test de sincronía JSON↔respaldo; los flujos son solo `.localcli/flows/*.json` | completada | `internal/flow/resolver.go`, `internal/flow/plan.go`, `internal/flow/work.go`, `internal/flow/ai_flows_test.go` | `go build ./...`; sin referencias a los constructores Go |
| T-B031-04 | actualizar | `flow`/`session`/`queue`/`arranque`: la cola resuelve el flujo del catálogo por la acción del elemento (`ElementoCola.Accion`, `flow.ResolutorFlujo`, `Catalogo.PorComando`, `Adaptador.flujoDeLaCola`); fuera `FlujoPorDefecto`, `Gestor.Flujo`, `flujoOFectivo` y `Gestor.Enviar` | completada | `internal/flow/engine.go`, `internal/flow/catalogo.go`, `internal/queue/next.go`, `internal/session/run.go`, `internal/session/pause.go`, `internal/session/bg.go`, `arranque.go` | `TestConsumirColaUnElementoPorIteracion`, `TestConsumirColaResuelveElFlujoPorAccion` en verde |
| T-B031-05 | actualizar | tests: flujos de prueba en `flow`/`session` (sin depender de flujos Go), e2e leyendo el catálogo real, y la matriz de reglas apuntando al test renombrado | completada | `internal/flow/pruebas_flujos_test.go`, `internal/flow/engine_test.go`, `internal/flow/bloque_test.go`, `internal/session/session_test.go`, `internal/session/chat_test.go`, `tests/e2e_localcli_test.go`, `tests/matriz_reglas_test.go` | `go test ./internal/flow/ ./internal/session/ ./internal/queue/ ./tests/ -count=1` |
| T-B031-06 | actualizar | Documentar: `SPEC-MOTOR-FLUJOS` (regla de la etapa que no responde, ventana), `SPEC-RESOLVER` (flujos como dato, `respuesta_en_chat`) y `ERRORS` (`E_STAGE_FAILED`) | completada | `ai/docs/specs/SPEC-MOTOR-FLUJOS.md`, `ai/docs/specs/SPEC-RESOLVER.md`, `ai/docs/backend/05-quality/ERRORS.md` | Documentación coherente con el código |

Dependencias: T-B031-01 antes de T-B031-05. T-B031-03 y T-B031-04 van juntas (quitar los constructores rompe a quien los use). T-B031-06 revisa el resto.

## Fuera de alcance

- **`continuacion`**: se parsea y el motor no la lee; la pausa antes de la etapa sigue pendiente.
- **`respuesta_en_chat` → `chat_evento`**: la spec dice que su texto se guarda como línea de procesamiento y no entra al contexto del modelo; hoy se persiste en `messages` y se encadena en la ventana. Se trata aparte.
- **Cambiar la regla de que toda etapa debe responder** a las etapas silenciosas existentes: es universal por diseño (una etapa sin respuesta no alimenta la ventana).
