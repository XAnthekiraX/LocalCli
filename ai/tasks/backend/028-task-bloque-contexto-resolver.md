> T-B028 — `/resolver` determinista: el flujo construye un bloque de contexto con el resultado de cada etapa, el modelo lo optimiza por fase, el bloque se persiste en `flow_context` (migración 003) y la última etapa compone el PLAN sin herramientas.
> Acción de esta descomposición: `actualizar`.

## Referencias

- [[specs/SPEC-MOTOR-FLUJOS]] — §Bloque de contexto.
- [[specs/SPEC-RESOLVER]] — la salida estándar y el paso de composición.
- [[backend/DECISIONS]] — la decisión del bloque de contexto y el cierre determinista.
- [[database/01-schema/TABLES]] — §`flow_context`.
- [[database/03-operations/MIGRATIONS]] — la migración 003.
- [[backend/04-infrastructure/EVENTS]] — §3: `etapa_iniciada` y la etapa de composición.

## Contexto

`/resolver` no entregaba el PLAN por dos motivos independientes:

1. La regla «`AGENTS.md` es la puerta de entrada» no llevaba ruta y el modelo la resolvía como `ai/docs/AGENTS.md`, que no existe (`E_BAD_ARGS`).
2. En `agent.Ejecutor` el texto se resetea en cada pasada y, al agotar las rondas con el modelo pidiendo herramientas, el turno devolvía el texto de la última pasada —un preámbulo («Voy a investigar…»)— en vez de su resultado. Encima, el motor encadenaba resúmenes recortados y el bloque vivía solo en memoria: el cierre dependía de que el modelo dejara de pedir herramientas por sí solo.

Esta tarea hace el cierre determinista: cada fase deja su aportación optimizada en un bloque persistido y la última etapa compone la entrega sin herramientas. No hay migración de datos: `flow_context` es aditiva.

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-B028-01 | actualizar | `agent`: pasada final de síntesis SIN herramientas al agotar las rondas y parámetro `sinHerramientas` por turno | completada | `internal/agent/loop.go` | `TestElBucleCierraConSintesisAlAgotarPasadas`, `TestElBucleSinHerramientasNoLasOfrece` en verde |
| T-B028-02 | actualizar | `flow`: flag `bloque_contexto`, tipos `Bloque`/`Optimizador` y pipeline del motor (optimización por fase, composición final) | completada | `internal/flow/bloque.go`, `internal/flow/engine.go`, `internal/flow/flujo.go`, `internal/flow/stage.go` | `internal/flow/bloque_test.go` en verde |
| T-B028-03 | actualizar | `resolver`: regla con la ruta raíz de `AGENTS.md`, `bloque_contexto` y JSON sincronizado | completada | `internal/flow/resolver.go`, `ai/flows/resolver.json` | `TestElFlujoResolverDelProyectoCoincideConElRespaldo` en verde |
| T-B028-04 | actualizar | `store`: tabla `flow_context`, migración 003 y repositorio | completada | `internal/store/flowcontext.sql`, `internal/store/flowcontext.go`, `internal/store/repos.go`, `internal/store/migrate.go`, `internal/store/schema.go` | `internal/store/flowcontext_test.go`, `TestEsquemaTablasEIndices`, `TestMigracionIdempotente` en verde |
| T-B028-05 | actualizar | `arranque`: `bloquePorTurno`, `optimizadorPorTurno` y `SinHerramientas` por turno | completada | `arranque.go` | `go build ./...` en verde |
| T-B028-06 | actualizar | Documentar: `SPEC-MOTOR-FLUJOS`, `SPEC-RESOLVER`, `DECISIONS`, `database/*` y `EVENTS` | completada | `ai/docs/specs/SPEC-MOTOR-FLUJOS.md`, `ai/docs/specs/SPEC-RESOLVER.md`, `ai/docs/backend/DECISIONS.md`, `ai/docs/backend/04-infrastructure/EVENTS.md`, `ai/docs/database/*` | Documentación coherente con el código |

Dependencias: T-B028-01 y T-B028-04 antes de T-B028-02. T-B028-02 antes de T-B028-05. T-B028-03 con T-B028-02. T-B028-06 revisa el resto.

## Fuera de alcance

- **No se cambia la visibilidad de la TUI.** La vista sigue anunciando `[Sub Proceso] <nombre>` y las líneas de herramienta; lo que se oculta es el texto de las fases intermedias (ya era así).
- **No se persisten las llamadas a herramienta ni el historial de fases en `messages`.** El bloque solo guarda la aportación de cada etapa.
- **Solo el resolver usa el bloque.** `trabajo` y `planificacion` no activan `bloque_contexto`: su última etapa no cambia.
- **No se expone el flujo como herramienta del modelo.** El resolver arranca solo con su comando explícito.
