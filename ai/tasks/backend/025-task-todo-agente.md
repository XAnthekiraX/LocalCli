> T-B025 — TODO del agente: una herramienta `actualizar_todo` deja que el agente mantenga una lista de pasos de la sesión, que se persiste por sesión y se ve en el panel.
> Acción de esta descomposición: `actualizar`.

## Referencias

- [[specs/SPEC-TOOLS]] — §Catálogo (Sesión), §El reparto y §Reglas de negocio.
- [[specs/SPEC-AGENTE-BASE]] — la acción `tareas` y el reparto de `plan`/`build`.
- [[backend/02-interfaces/TOOLS]] — §1 (la herramienta), §2 (la acción `tareas`) y §7.
- [[backend/02-interfaces/dto/TOOLS-DTO]] — el esquema de `actualizar_todo`.
- [[backend/04-infrastructure/EVENTS]] — `todo_actualizada`.
- [[backend/03-security/SECURITY]] — la acción `tareas` no escribe en el proyecto.
- [[database/01-schema/TABLES]] — la tabla `todos`.
- [[database/03-operations/MIGRATIONS]] — la migración `002-crear-todo`.

## Contexto

El agente necesita planear trabajo de varios pasos y que el usuario lo vea. Se añade una catorceava herramienta, `actualizar_todo`, que reescribe entera la lista de pasos de la sesión (como el `todowrite` de opencode), más la acción de permiso `tareas` para que la tengan `plan` y `build` sin abrir una vía de escritura al proyecto. La lista vive en SQLite por sesión (tabla `todos`, migración 002) y se pinta en una sección del panel.

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-B025-01 | actualizar | `tools`: categoría `CatTareas`, acción `AccionTareas`, derivación en `Accion()` y `SoloBuild()` por acción | completada | `internal/tools/catalog.go` | Tests de catálogo y de `plan`/`build` en verde |
| T-B025-02 | actualizar | `tools`: DTO `PeticionActualizarTodo`/`ElementoTodo`, entrada en `peticiones` y validación del vocabulario | completada | `internal/tools/dto_request.go`, `internal/tools/validate.go` | Test `TestActualizarTodoValidaElVocabulario` en verde |
| T-B025-03 | actualizar | `store`: tabla `todos` (migración 002), `LeerTodos`/`ReemplazarTodos` y el repositorio `Todos` | completada | `internal/store/todo.sql`, `internal/store/todos.go`, `internal/store/migrate.go`, `internal/store/verify.go`, `internal/store/repos.go` | Test `TestReemplazarYLeerTodos` en verde; `user_version = 2` |
| T-B025-04 | actualizar | cableado: handler `herramientaActualizarTodo` (persiste, emite `todo_actualizada`, devuelve el checklist) y el método `Adaptador.Tareas` | completada | `herramientas.go`, `arranque.go` | El registro tiene catorce herramientas |
| T-B025-05 | actualizar | agentes: conceder `tareas` a `plan` y `build` | completada | `ai/agents/plan.json`, `ai/agents/build.json` | `plan.TieneEscritura()` sigue falso; `plan` ve ocho herramientas |
| T-B025-06 | actualizar | tests: actualizar los conteos (14/8), `TestSinTablasProhibidas` (7 tablas) y añadir los del repositorio y la validación | completada | `internal/tools/*_test.go`, `internal/agent/*_test.go`, `tests/*_test.go`, `internal/store/store_test.go` | `go test ./...` en verde |

Dependencias: T-B025-01 y T-B025-02 antes de T-B025-04. T-B025-03 antes de T-B025-04. T-B025-04 antes de T-B025-05. T-B025-06 revisa todas.

## Fuera de alcance

- **No hay reinyección de la lista en el contexto.** El modelo ve la lista en el resultado de la herramienta; inyectarla en cada turno queda para más adelante.
- **No se valida en código "un solo `en_progreso`".** La política vive en la descripción de la herramienta, no en la capa universal.
- **No se persisten los turnos de herramienta.** `messages.role` sigue siendo `user`/`agent`. Ver [[database/03-operations/MIGRATIONS]].
