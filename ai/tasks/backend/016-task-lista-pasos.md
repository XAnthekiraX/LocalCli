# 016-task-lista-pasos.md

> T-B016 — Lista de pasos de la sesión (`crear_todo` y `actualizar_todo`).
> Acción de esta descomposición: `crear`.

## Referencias

- [[backend/02-interfaces/TOOLS]] §1 Tareas, §8 La capa universal
- [[backend/02-interfaces/dto/TOOLS-DTO]] §3 Request Schemas (§Tareas), §4 Response Schemas
- [[specs/SPEC-TOOLS]] §`crear_todo`, §`actualizar_todo`
- [[database/01-schema/TABLES]] §`todos` — lista de pasos de una sesión
- [[database/01-schema/ENUMS]] — estado y prioridad del paso
- [[backend/04-infrastructure/EVENTS]] §3 `todo_actualizada`
- [[backend/05-quality/ERRORS]] §5 (§Lista de pasos)

## Contexto

La lista de pasos es estado de ejecución de la **sesión**, no un cambio en el
proyecto: por eso `crear_todo` y `actualizar_todo` van con permiso `read` y las
tienen los dos agentes. `crear_todo` **añade un paso al final** sin tocar los que
ya había; `actualizar_todo` **sustituye la lista entera** en un solo paso atómico.
Las dos devuelven el checklist resultante.

Los handlers viven en `session` —necesitan la base, la sesión y el emisor de
eventos— y el orquestador los instala en el registro al cablear.

## Tareas pequeñas

| ID         | Acción | Tarea                                                                                                          | Estado    | Archivos                       | Verificación                                                                         |
| ---------- | ------ | ------------------------------------------------------------------------------------------------------------- | --------- | ------------------------------ | ------------------------------------------------------------------------------------ |
| T-B016-01  | crear  | Persistencia: `ListarTodos`, `CrearTodo` (inserta en `position = COUNT(*)`, sin borrar) y `ReemplazarTodos` (borra e inserta atómico) | completada | `internal/store/todos.go`      | Añadir un paso no pierde los anteriores; el reemplazo es todo o nada                 |
| T-B016-02  | crear  | Handler `crear_todo`: valida contenido, estado y prioridad, rechaza un segundo `en_progreso` nombrando `actualizar_todo`, y devuelve el checklist | completada | `internal/session/todos.go`    | Un segundo `en_progreso` da `E_BAD_ARGS`; el checklist usa `[ ]`/`[•]`/`[✓]`/`[x]`    |
| T-B016-03  | crear  | Handler `actualizar_todo`: valida cada elemento y admite como mucho un `en_progreso`; sustituye la lista y devuelve el checklist | completada | `internal/session/todos.go`    | Dos `en_progreso` dan `E_BAD_ARGS`; la lista vacía deja el checklist en blanco        |
| T-B016-04  | crear  | Emitir `todo_actualizada` con la lista completa (`elementos`), que el panel repinta                            | completada | `internal/session/todos.go`    | El evento llega con la lista resultante tras cada escritura                          |
| T-B016-05  | crear  | Cableado: instalar los dos handlers en el registro desde `ConHerramientas`                                     | completada | `internal/session/orquestador.go` | Ejecutar `crear_todo` por el registro devuelve el checklist sin "no implementada" | 

**Dependencias:** `01`→`02`,`03`; `02`,`03`→`04`; `04`→`05`.
**Fuera de alcance:** la cola y las tareas de la capa (`queue`, T-B018); el panel de
la vista, que ya consume `todo_actualizada` (capa frontend).
