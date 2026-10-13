# 022-task-todo-subtareas.md

> T-B022 — Jerarquía de un nivel en la lista de pasos: tarea principal con subtareas (`crear_todo`/`actualizar_todo`).
> Acción de esta descomposición: `actualizar`.

## Referencias

- [[specs/SPEC-TOOLS]] §La lista de pasos — las dos formas de crear y la jerarquía de un nivel
- [[backend/02-interfaces/TOOLS]] §1 Sesión — contrato de `crear_todo` y `actualizar_todo`
- [[backend/02-interfaces/dto/TOOLS-DTO]] §3 Sesión, §4 Sesión — payloads y checklist anidado
- [[database/01-schema/TABLES]] §`todos` — columna `parent_position` y cómo se impone la referencia
- [[database/01-schema/SCHEMA]] §`todos` — el campo en el esquema
- [[database/01-schema/INDEXES]] §`idx_todos_padre` — lectura del grupo de subtareas
- [[database/03-operations/MIGRATIONS]] §`008-todo-jerarquia` — columna e índice aditivos
- [[backend/04-infrastructure/EVENTS]] §3 `todo_actualizada` — payload con `padre`
- [[backend/01-domain/BUSINESS_RULES]] §Agentes — reglas de las dos herramientas

## Contexto

`crear_todo` y `actualizar_todo` mantienen la lista de pasos de la sesión, y hasta
ahora la lista es plana: cada paso es una fila de `todos`. El flujo de planificación
pide al modelo estructurar una petición con fases, y el modelo mete la lista entera
en el `contenido` de un solo paso porque no tiene forma de declarar jerarquía.

Esta tarea la añade, con una **jerarquía de un nivel**: `crear_todo` acepta, además
del paso suelto (`contenido`), una tarea principal con `titulo` y `subtareas`,
insertada como un grupo atómico; `actualizar_todo` admite `subtareas` opcionales en
cada elemento. En la base, la jerarquía vive en la columna nullable
`todos.parent_position` (la migración `008-todo-jerarquia` la añade junto con el
índice `idx_todos_padre`); la referencia no se puede declarar como clave foránea con
`ALTER TABLE`, así que su validez se impone en la capa de datos al escribir.

No cambia el comportamiento de una llamada sin subtareas: un paso suelto se inserta
igual que hoy, y las bases v7 se leen como listas planas.

## Tareas pequeñas

| ID         | Acción    | Tarea                                                                                                                   | Estado    | Archivos                              | Verificación                                                                                                                   |
| ---------- | --------- | ----------------------------------------------------------------------------------------------------------------------- | --------- | ------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------ |
| T-B022-01  | actualizar | Migración `008-todo-jerarquia`: `ALTER TABLE todos ADD COLUMN parent_position INTEGER NULL` + índice `idx_todos_padre` (`session_id, parent_position`); DDL de creación en v8 y `user_version = 8` | pendiente | `internal/store/migraciones.go`, `internal/store/schema.go` | Una base v7 abre y pasa a v8 sin tocar filas; el DDL de creación nueva y la migración dejan el mismo esquema                     |
| T-B022-02  | actualizar | Store: `Todo` gana `PosicionPadre` (`*int`); `CrearTodo` inserta el grupo (principal + subtareas) en una transacción; `ReemplazarTodos` escribe `parent_position`; la lectura devuelve `parent_position` y valida que apunte a una fila existente de la sesión | pendiente | `internal/store/todos.go`              | Crear un grupo deja la principal y sus subtareas seguidas en una transacción; un `parent_position` huérfano da error al escribir  |
| T-B022-03  | actualizar | Peticiones y esquemas: `PeticionCrearTodo` gana `Titulo` y `Subtareas []ElementoTodo` (excluyentes con `Contenido`); `actualizar_todo` admite `subtareas` opcional por elemento; rellenar `Propiedad.Enum` para `estado`/`prioridad` en `esquema.go` | pendiente | `internal/tools/peticiones.go`, `internal/tools/catalogo.go`, `internal/tools/esquema.go` | `titulo`+`subtareas` conviven con `contenido`+`estado`+`prioridad`; mezclar ambos o ninguno da `E_BAD_ARGS`; el esquema generado muestra el enum de estado y prioridad |
| T-B022-04  | actualizar | Handlers de sesión: `crear_todo` acepta las dos formas y devuelve el checklist con las subtareas anidadas bajo su título; `actualizar_todo` acepta `subtareas` por elemento; el `en_progreso` único se cuenta sobre toda la lista, incluidas las subtareas | pendiente | `internal/session/todos.go`            | «Crea un TODO con las fases del ciclo de vida del software» produce «Ciclo de vida del software» con sus 7 fases como subtareas; sin `subtareas` la llamada se comporta como hoy                                  |
| T-B022-05  | actualizar | Emitir `todo_actualizada` con `padre` (posición de la tarea principal, `NULL` si es raíz) en cada elemento                  | pendiente | `internal/session/todos.go`            | El evento lleva la lista resultante con el `padre` correcto; el panel lo repinta anidado                                        |
| T-B022-06  | actualizar | Prompts de `plan` y `build`: instruir a estructurar las peticiones con fases como una tarea principal con sus subtareas, en vez de meter la lista en un solo paso | pendiente | `internal/arranque/datos/agents/plan/prompt.md`, `internal/arranque/datos/agents/build/prompt.md` | La descripción de `crear_todo` y el prompt del agente piden `titulo` + `subtareas` para una lista ordenada                          |
| T-B022-07  | actualizar | Tests de la jerarquía: caso «fases del ciclo de vida del software»; compatibilidad de `crear_todo` sin `subtareas`; exclusión `contenido`/`titulo`; máximo un nivel; checklist y evento con `padre` | pendiente | tests de `internal/tools`, `internal/session`, `internal/store` | La suite cubre el caso aprobado (título «Ciclo de vida del software» + 7 subtareas), la no regresión de la lista plana y los errores corregibles |

**Fuente de verdad:** [[specs/SPEC-TOOLS]] (aprobada) y la documentación por capas
actualizada en esta misma fase. No se implementa nada que no esté documentado ahí.

**Dependencias:** `01`→`02`,`03`; `02`,`03`→`04`; `04`→`05`,`06`; `07` cierra y valida sobre las anteriores.
**Fuera de alcance:** la detección automática de peticiones ordenadas (T-B018/T-B019); el render anidado del panel, que es de la capa frontend.