> T-B030 — los agentes base se cargan de `.localcli/agents/`, junto a las herramientas del usuario, y esa carpeta vuelve a versionarse con el proyecto.
> Acción de esta descomposición: `actualizar`.

## Referencias

- [[specs/SPEC-AGENTE-BASE]] — §Dónde se definen: todos los `*.json` de la carpeta se cargan y el nombre del archivo no decide nada.
- [[backend/04-infrastructure/CONFIGURATION]] — §4 *Rutas* y la sección *Agentes del proyecto*.
- [[backend/DECISIONS]] — la decisión reformulada: de `.localcli/` se ignora el archivo SQLite, no la carpeta entera.
- [[backend/01-domain/DOMAIN]] — la fila del agente base y sus derivados.
- [[backend/02-interfaces/INTERFACES-GENERAL]] — qué carga y valida `agent`.

## Contexto

El commit `499c2ab` movió los agentes de `ai/agents/` a `.localcli/agents/` —para que vivieran junto a las herramientas del usuario— y actualizó **una sola línea**: `agenteBase` (`arranque.go`). Todo lo demás siguió apuntando a la carpeta vieja, y eso tuvo dos efectos distintos.

El primero fueron tres tests en rojo: `TestAgenteBaseCargaAgentesPropios` escribía los JSON de un agente propio en `ai/agents/` de un directorio temporal y `agenteBase` no los veía; `TestAgentesBaseDocumentados` y `TestUnAgenteBaseTieneSuCatalogo` abrían la carpeta del proyecto, que ya no existía.

El segundo, más grave y silencioso: `.gitignore` ignoraba `.localcli` entero, así que `plan.json` y `build.json` dejaron de versionarse. Los dos tests anteriores pasaban solo por el estado de esta copia de trabajo: en un clon nuevo `agenteBase` no encuentra ningún JSON, cae en `agenteDeRespaldo` —un prompt de identidad sin permisos— y `plan` pierde sus ocho herramientas de lectura mientras `build` pierde las catorce. La garantía de que el proyecto trae sus agentes base se sostenía únicamente en un archivo que git no conocía.

Esta tarea cierra el traslado: `.localcli/agents/` es la ruta canónica, la carpeta se versiona con el proyecto como ya se declaraba para `tools/`, y cada referencia del código y de la documentación nombra la ruta nueva.

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-B030-01 | actualizar | `arranque`: `agenteBase` carga los agentes de `.localcli/agents/*.json` | completada | `arranque.go` | `TestAgenteBaseCargaAgentesPropios` en verde |
| T-B030-02 | actualizar | Tests: las tres rutas que leen los agentes del proyecto apuntan a `.localcli/agents` | completada | `arranque_test.go`, `internal/agent/agent_test.go`, `tests/integration_errores_test.go` | `TestAgentesBaseDocumentados`, `TestUnAgenteBaseTieneSuCatalogo` en verde |
| T-B030-03 | actualizar | `.gitignore`: ignora solo `state.db`, `state.db-shm` y `state.db-wal`; `agents/` y `tools/` versionables | completada | `.gitignore` | `git check-ignore` no marca `.localcli/agents/plan.json` y sí `state.db` |
| T-B030-04 | actualizar | Comentarios de código que nombraban `ai/agents` | completada | `arranque.go`, `internal/agent/loader.go`, `internal/agent/model.go`, `internal/flow/engine.go`, `internal/flow/stage.go`, `internal/tui/app.go`, `internal/tui/wire.go` | `go build ./...`; sin referencias a `ai/agents` |
| T-B030-05 | actualizar | Documentación: specs, `DOMAIN` de backend y frontend, `INTERFACES-GENERAL`, `PROJECT`, `AGENTS.md`; `CONFIGURATION` gana la fila y la sección; `DECISIONS` se reformula | completada | `ai/docs/*`, `AGENTS.md` | Sin referencias a `ai/agents`; `go test ./internal/docs/...` en verde |
| T-B030-06 | actualizar | Tareas y auditoría: referencias al lugar de los agentes | completada | `ai/tasks/backend/006-task-agent.md`, `ai/tasks/backend/017-task-permisos-y-ciclo-agente.md`, `ai/tasks/backend/023-task-agentes-configurables.md`, `ai/tasks/backend/025-task-todo-agente.md`, `ai/tasks/backend/MAIN-TASKS.md`, `ai/tasks/frontend/031-task-agentes-lista.md`, `ai/audit/AUDIT.md` | Sin referencias a `ai/agents` |

Dependencias: T-B030-03 antes que T-B030-02 —mientras el contenido de la carpeta siga sin versionarse, los tests que leen los agentes base del proyecto pasan solo en esta copia de trabajo—. T-B030-01 ya venía de `499c2ab`. T-B030-04, T-B030-05 y T-B030-06 revisan el resto.

## Fuera de alcance

- **No se recuperan `explore.json`, `general.json` y `test.json`**, que el traslado descartó. Se decidió que quedan fuera: el flujo `resolver` solo referencia `plan`, y ningún flujo propio depende de ellos.
- **No hay búsqueda en dos carpetas.** `agenteBase` lee una sola ruta fija; no se añade variable de entorno ni carpeta de reserva para `ai/agents/`.
- **No se toca `.localcli/tools/`** más allá de dejar de ignorarla. La decisión ya lo declaraba contenido versionado del proyecto; aquí solo se le quita la regla que lo contradecía.
- **El estado no se toca.** `state.db` sigue en la misma ruta, con los mismos PRAGMA y la misma migración; lo único que cambia es qué se ignora en git.
