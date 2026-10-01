> T-B035 — la lista de pasos de la sesión crece por dos vías: `crear_todo` **añade** un paso al final y `actualizar_todo` **sustituye** la lista entera, con el mismo vocabulario, la misma lista devuelta y sin migración de base de datos.
> Acción de esta descomposición: `actualizar`.

## Referencias

- [[specs/SPEC-TOOLS]] — §La lista de pasos: qué hace cada una y por qué son dos; §Catálogo: la entrada de `crear_todo` en `read`.
- [[specs/SPEC-INTERFAZ]] — §El panel: la sección «LISTA DE TAREAS» pinta lo que devuelvan las dos.
- [[backend/02-interfaces/TOOLS]] — §1: el verbo `TODO` de las dos y el tema `contenido` de la nueva.
- [[backend/02-interfaces/dto/TOOLS-DTO]] — §Request Schemas §Sesión: `PeticionCrearTodo` con tres campos planos, sin `elementos`.
- [[backend/01-domain/BUSINESS_RULES]] — §Agentes: añadir y sustituir son contratos distintos.
- [[database/01-schema/TABLES]] — §7 `todos`: por qué insertar no renumera nada y por qué el esquema no cambia.

## Contexto

La lista de pasos de la sesión tiene hoy una sola puerta. `actualizar_todo` recibe la lista entera y la sustituye:

```go
// internal/tools/dto_request.go
type PeticionActualizarTodo struct {
    Elementos []ElementoTodo `json:"elementos"`
}
```

y el store borra y reinserta, en un solo paso atómico:

```go
// internal/store/todos.go
func reemplazarTodos(e ejecutor, sessionID string, items []Todo) error {
    if _, err := e.Exec(`DELETE FROM todos WHERE session_id = ?`, sessionID); err != nil { ... }
    for i, it := range items { INSERT ... position = i }
}
```

De ahí el problema que motiva la tarea: **la única forma de añadir un paso es reescribir la lista entera**. Un modelo que planifica paso a paso tiene que mandar, en cada llamada, todos los pasos que ya越好 —con su contenido y su estado— para añadir uno. Y el modo de fallo es silencioso: si el modelo cree que está «actualizando un paso» y manda lo que recuerda, la lista anterior **se borra sin que nadie lo decida**. `todo_test.go` no lo cubre porque el contrato actual no tiene esa forma de uso.

La solución documentada es una segunda herramienta con un contrato más pequeño, `crear_todo`, que no puede perder nada porque no toca lo que hay. Su esquema no lleva `elementos`: lleva `contenido`, `estado` y `prioridad`.

Dos cosas que hay que tener presentes al implementarlo:

1. **La clave primaria es `(session_id, position)`**, así que añadir es `INSERT` en `position` = número de pasos actuales. No hay que renumerar nada porque `crear_todo` nunca mueve un paso. No hay migración: el DDL no cambia.
2. **La regla de un solo `en_progreso` a la vez hoy no está comprobada**: es una instrucción en la descripción de `actualizar_todo` (`catalog.go`), no una validación en `validarTodo`. Con `crear_todo` pasa a poder romperse desde código —insertar un segundo paso en curso— y eso sí tiene que ser un error corregible que nombre `actualizar_todo`, no un conflicto silencioso.

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-B035-01 | actualizar | `tools`: `PeticionCrearTodo` con `Contenido`, `Estado` y `Prioridad` (`omitempty`), **sin `elementos`**, y su entrada en la tabla `peticiones`; el esquema se deriva del DTO, sin escribirlo a mano | completada | `internal/tools/dto_request.go` | `TestElEsquemaDeCrearTodoNoTieneElementos` en verde |
| T-B035-02 | actualizar | `tools`: entrada de catálogo `crear_todo` con verbo `TODO`, tema `contenido` y unidad `paso`, en `read`; la descripción dice cuándo usarla y que **añade un paso al final sin tocar los demás**, y no se parece a la de `actualizar_todo` | completada | `internal/tools/catalog.go` | `TestCrearTodoEsReadYNoSeConfundeConActualizar` en verde |
| T-B035-03 | actualizar | `tools`: la validación comparte el vocabulario con `actualizar_todo` —`contenido` no vacío, estado en `pendiente`/`en_progreso`/`completada`/`cancelada`, prioridad en `alta`/`media`/`baja` o vacía— y además **rechaza añadir un paso `en_progreso` cuando ya hay uno**, con un `E_BAD_ARGS` corregible que nombre `actualizar_todo` | completada | `internal/tools/validate.go` | `TestCrearTodoConSegundoEnProgresoSeRechaza`, `TestCrearTodoValidaElVocabulario` en verde |
| T-B035-04 | actualizar | `store`: `AgregarTodo` inserta **una fila** en `position` = número de pasos de la sesión y no borra nada; sin migración y sin tocar `ReemplazarTodos` | completada | `internal/store/todos.go` | `TestAgregarTodoNoPierdeLosAnteriores` en verde |
| T-B035-05 | actualizar | `tools`: el handler de `crear_todo` valida, agrega, lee la lista resultante y **devuelve el mismo checklist** que devuelve `actualizar_todo`; emite `todo_actualizada` con la lista entera, sin cambiar el payload del evento | completada | `internal/tools/sesion.go`, `internal/tools/route.go` | `TestAmbasDevuelvenElMismoChecklist` en verde |
| T-B035-06 | actualizar | tests: el caso que hoy no existe —añadir un paso a una lista de tres, verificar que los tres siguen igual y que el nuevo es el último—, más lista vacía, `contenido` vacío, vocabulario inválido y el rechazo del segundo `en_progreso` | completada | `internal/tools/todo_test.go`, `internal/store/todos_test.go` | `go test ./internal/tools/ ./internal/store/ -count=1` en verde |
| T-B035-07 | actualizar | Documentar en `ERRORS.md` el nuevo `E_BAD_ARGS` del segundo `en_progreso` junto a los de la lista de pasos, para que sea reconocible y no una cadena nueva | completada | `ai/docs/backend/05-quality/ERRORS.md` | Documentación coherente con el código |

Dependencias: T-B035-01 antes de T-B035-02 y -03. T-B035-04 antes de T-B035-05. T-B035-07 revisa T-B035-03.

## Fuera de alcance

- **`actualizar_todo` no cambia.** Sigue sustituyendo la lista entera, con el mismo contrato y la misma semántica atómica. No se le añade un modo «parche».
- **No hay identificador de paso.** El orden es la posición; un `id` solo daría material para que un modelo cite mal.
- **No hay migración de SQLite.** Ni columna, ni tabla, ni `user_version` nuevo: `crear_todo` inserta en la misma tabla.
- **No cambia el DDL de `todos`.** `ReemplazarTodos` sigue borrando y reinscribiendo, que es lo que lo hace atómico.
- **No se toca el panel.** Sigue pintando lo que llega en `todo_actualizada`, que sigue siendo la lista entera.
- **No cambia el verbo visual.** Las dos herramientas se muestran como `TODO`: el modelo ya sabe cuál llamó por su nombre, y en el panel la diferencia no aporta nada.
- **No se introduce un tercer verbo** tipo `TODO+`: la lista de pasos sigue siendo un único concepto en pantalla.
- **No se cambia la lista de pasos del proyecto** (`ai/tasks/**/MAIN-TASKS.md`): es otra cosa. Esta tarea es solo la lista de la sesión, la de la tabla `todos`.
