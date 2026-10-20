---
title: LocalCli — tablas de la base de datos
tags: [database, esquema]
depende_de:
  - "[[database/01-schema/SCHEMA]]"
relacionado:
  - "[[database/01-schema/RELATIONSHIPS]]"
  - "[[database/01-schema/ENUMS]]"
  - "[[database/02-rules/BUSINESS_RULES]]"
  - "[[database/02-rules/DATA_FLOW]]"
  - "[[specs/SPEC-AGENTE-BASE]]"
  - "[[specs/SPEC-INTERFAZ-ATAJOS]]"
  - "[[specs/SPEC-NODO-CONTEXTO]]"
  - "[[specs/SPEC-PANEL-CONTEXTO]]"
  - "[[specs/SPEC-TOOLS]]"
---
# TABLES — Tablas de la base de datos

## 1. Tablas

El esquema tiene diez tablas. La estructura de alto nivel está en [[database/01-schema/SCHEMA]]; aquí cada columna en detalle. Las relaciones entre tablas están en [[database/01-schema/RELATIONSHIPS]] y los valores cerrados, en [[database/01-schema/ENUMS]].

Convenciones que aplican a todas: `id` es `TEXT` con UUID v4, las fechas son `TEXT` en ISO 8601 UTC, y los booleanos son enteros `0`/`1`. La única excepción es `todos`, que no tiene `id`: se identifica por `(session_id, position)`.

## 2. Propósito de cada tabla

- **`projects`** — Un proyecto: la carpeta desde la que se abrió LocalCli. Es la raíz lógica del árbol de la base, porque el archivo es único y global y las sesiones de todos los directorios comparten estructura: toda sesión cuelga de su proyecto por `project_id`. Su `path` es único, así que abrir la misma carpeta dos veces reutiliza la fila. Ver [[specs/SPEC-SESIONES]].
- **`sessions`** — Una sesión de trabajo. Es la unidad de la que cuelgan la conversación, las aprobaciones y la auditoría. Su campo `status` es lo que la interfaz muestra cuando cambias de sesión. Guarda además el `modelo` con el que trabaja: es un recurso de la sesión, no parte de su identidad, así que cambiarlo no la renombra ni le toca el historial.
- **`messages`** — Los turnos de la conversación. Guarda lo que escribió el usuario y lo que respondió el agente, más los tokens de cada turno y su duración. Un turno puede incluir llamadas a herramientas, pero **esas no se guardan en `messages`**: su llamada y su resultado viven en `chat_evento`, junto a su línea, y se reenvían al modelo en los turnos siguientes. Ver [[database/02-rules/DATA_FLOW]].
- **`reasoning`** — El razonamiento del modelo de un mensaje concreto. Va en tabla aparte porque llega token a token mientras se genera y porque se consulta y se mide por separado del texto final. El diseño está en [[specs/SPEC-AGENTE-BASE]].
- **`approvals`** — Lo que una sesión necesita que decidas antes de seguir. El panel de aprobaciones las lee todas, de cualquier sesión. Ver [[specs/SPEC-INTERFAZ-ATAJOS]].
- **`context_audit`** — La traza de qué documentación recibió el modelo en cada etapa y qué se descartó, con el motivo. Es lo que hace auditable el nodo de contexto. Ver [[specs/SPEC-NODO-CONTEXTO]].
- **`change_history`** — Cada cambio aplicado a un archivo del proyecto, con lo que había antes y lo que quedó. Nunca se borra. Ver [[database/02-rules/BUSINESS_RULES]].
- **`todos`** — La lista de pasos de una sesión: el plan que el agente mantiene con `crear_todo` y `actualizar_todo`. Es estado de ejecución de la sesión, no un documento del proyecto: `actualizar_todo` la reescribe entera y `crear_todo` le añade al final un paso o una tarea principal con sus subtareas. Cae con su sesión. Ver [[specs/SPEC-TOOLS]].
- **`flow_context`** — El bloque de contexto de un flujo con `bloque_contexto`: una aportación por etapa, ya optimizada por el modelo. Es lo que permite que la composición final arme la entrega a partir de todo el trabajo de las etapas. Es estado de ejecución de la sesión: se reemplaza en cada ejecución del flujo y cae con su sesión. Ver [[specs/SPEC-ORQUESTADOR-FLUJOS]].
- **`chat_evento`** — Las piezas de pantalla del chat que no son conversación: el sub-proceso de cada etapa de un flujo, la línea de cada herramienta y la respuesta de una etapa que pidió verse en el hilo. Se muestran al usuario y se recuerdan al volver a una sesión. Las herramientas guardan además la llamada y su salida (`call_id`, `argumentos`, `salida`, `error`, `ok`, `truncado`) para reenviarlas al modelo como `assistant(tool_calls)` + `tool(resultado)`; los sub-procesos y las respuestas de etapa siguen siendo solo pantalla. Cae en cascada con su sesión. Ver [[database/02-rules/DATA_FLOW]].

## 3. Columnas

### `projects`

| Columna | Qué representa | Valores permitidos | Nullable | Default |
|---|---|---|---|---|
| `id` | Identificador del proyecto | UUID v4 | No | — |
| `path` | Ruta absoluta de la carpeta del proyecto | Ruta absoluta, única | No | — |
| `created_at` | Cuándo se abrió la carpeta por primera vez | ISO 8601 UTC | No | — |
| `updated_at` | Última actividad registrada en el proyecto | ISO 8601 UTC | No | — |

`path` es `UNIQUE` y es la identidad real del proyecto: `id` es solo la clave surrogada con la que las sesiones lo referencian. Se usa UUID y no la ruta como clave primaria porque una ruta es larga, se repite en cada fila hija y puede cambiar (renombrar la carpeta, moverla); con un id estable la identidad de las sesiones no depende de dónde esté el directorio. La ruta se normaliza a absoluta al abrir, para que la misma carpeta no genere dos filas por escribirse de dos formas.

### `sessions`

| Columna | Qué representa | Valores permitidos | Nullable | Default |
|---|---|---|---|---|
| `id` | Identificador de la sesión | UUID v4 | No | — |
| `project_id` | Proyecto al que pertenece la sesión | UUID v4 de `projects.id` | No | — |
| `name` | Nombre visible de la sesión | Texto libre | No | — |
| `layer` | Capa en la que trabaja la sesión | `backend`, `frontend` | Sí | `NULL` |
| `status` | Estado de la sesión | Ver [[database/01-schema/ENUMS]] | No | `inactiva` |
| `created_at` | Cuándo se creó | ISO 8601 UTC | No | — |
| `updated_at` | Última actividad | ISO 8601 UTC | No | — |
| `modelo` | Modelo que usa la sesión | Nombre tal cual lo declara el motor | Sí | `NULL` |
| `summary` | Resumen compactado del historial, ya optimizado por el modelo | Texto libre | Sí | `NULL` |
| `summary_upto` | `id` del último mensaje que el resumen abarca | UUID v4 de `messages.id` | Sí | `NULL` |

`layer` es `NULL` para una sesión general, por ejemplo cuando solo conversas o revisas contexto sin ejecutar tareas de una capa concreta.

`modelo` es la **huella de ejecución de la sesión**: es lo que se lee al retomar y lo que se actualiza al cambiar de modelo. El motor no se elige por sesión —el proveedor es único (llama.cpp)—, así que la columna `motor_id` que existió en versiones anteriores se eliminó (migración `009-quitar-motor-id`). Ver [[specs/SPEC-SESIONES]] y [[specs/SPEC-MODELOS-LLAMACPP]].

`summary` y `summary_upto` son **el resumen persistido del historial**. Viven en la sesión y no en una tabla aparte porque son un único valor derivado por sesión: el resumen compactado y hasta qué mensaje llega dan lo necesario para reutilizarlo entre turnos y ampliarlo de forma incremental sin resumir dos veces. Van nullable: una sesión sin compactar tiene las dos en `NULL` —no hay resumen ni, por tanto, un punto de corte—. Guardarlo aquí no sustituye a los mensajes, que siguen completos; es una caché del tramo antiguo ya resumido. Una sesión deshecha hasta un punto anterior al corte deja el resumen obsoleto y se limpia. Ver [[specs/SPEC-HISTORIAL-CONVERSACION]].

Un detalle que no es obvio:

- **`modelo` no se valida contra el motor.** El mismo nombre puede no existir en otro runtime. La comprobación —«¿este motor tiene este modelo?»— es del momento en que se usa, no de la escritura.

### `messages`

| Columna | Qué representa | Valores permitidos | Nullable | Default |
|---|---|---|---|---|
| `id` | Identificador del mensaje | UUID v4 | No | — |
| `session_id` | Sesión a la que pertenece | UUID v4 de `sessions.id` | No | — |
| `role` | Quién escribió | `user`, `agent` | No | — |
| `content` | Texto del mensaje | Texto libre | No | — |
| `input_tokens` | Tokens consumidos en este turno | Entero ≥ 0 | Sí | `NULL` |
| `output_tokens` | Tokens generados en este turno | Entero ≥ 0 | Sí | `NULL` |
| `duration_ms` | Cuánto tardó el turno, en milisegundos | Entero ≥ 0 | Sí | `NULL` |
| `created_at` | Cuándo se escribió el mensaje | ISO 8601 UTC | No | — |

Los dos campos de tokens son `NULL` cuando el modelo no los reporta. Se distinguen del `0`: `NULL` significa "no lo sé", `0` significa "cero". `duration_ms` sigue la misma convención: `NULL` es "no se midió". El panel de contexto marca la estimación como tal. Ver [[specs/SPEC-PANEL-CONTEXTO]].

**Lo que no hay en esta tabla.** No hay columnas para las peticiones de herramientas ni para sus resultados, y `role` no tiene un valor de herramienta. Es una decisión, no un olvido: la conversación guarda los turnos de usuario y agente, y el detalle de las ejecuciones vive en `chat_evento`, donde además de la línea de pantalla se guardan la llamada y su salida para reenviarlas al modelo. Ver [[database/01-schema/ENUMS]] y [[backend/DECISIONS]].

### `reasoning`

| Columna | Qué representa | Valores permitidos | Nullable | Default |
|---|---|---|---|---|
| `id` | Identificador del razonamiento | UUID v4 | No | — |
| `message_id` | Mensaje al que pertenece | UUID v4 de `messages.id` | No | — |
| `content` | Razonamiento acumulado del modelo | Texto libre | No | — |
| `created_at` | Cuándo se empezó a acumular | ISO 8601 UTC | No | — |

Como el razonamiento llega en streaming, `content` se va escribiendo token a token mientras la respuesta se genera. Un mensaje de agente tiene como máximo un registro aquí.

### `approvals`

| Columna | Qué representa | Valores permitidos | Nullable | Default |
|---|---|---|---|---|
| `id` | Identificador de la aprobación | UUID v4 | No | — |
| `session_id` | Sesión que espera la decisión | UUID v4 de `sessions.id` | No | — |
| `description` | Qué se propone hacer | Texto libre | No | — |
| `status` | Estado de la aprobación | Ver [[database/01-schema/ENUMS]] | No | `pendiente` |
| `created_at` | Cuándo se pidió | ISO 8601 UTC | No | — |
| `resolved_at` | Cuándo se decidió | ISO 8601 UTC | Sí | `NULL` |

`resolved_at` es `NULL` mientras la aprobación siga `pendiente` y se rellena al pasar a `aprobada`, `declinada` u `obsoleta`.

### `context_audit`

| Columna | Qué representa | Valores permitidos | Nullable | Default |
|---|---|---|---|---|
| `id` | Identificador del registro | UUID v4 | No | — |
| `session_id` | Sesión que ejecutó la etapa | UUID v4 de `sessions.id` | No | — |
| `stage` | Etapa del flujo | Texto libre | No | — |
| `document` | Ruta del documento | Ruta relativa al proyecto | No | — |
| `decision` | Si entró o se descartó | `incluido`, `descartado` | No | — |
| `reason` | Motivo del descarte | Texto libre | Sí | `NULL` |
| `tokens` | Tamaño estimado del documento | Entero ≥ 0 | Sí | `NULL` |
| `created_at` | Cuándo se decidió la selección | ISO 8601 UTC | No | — |

Hay un registro por documento y por etapa. `reason` es `NULL` cuando `decision` es `incluido`, y obligatorio cuando es `descartado`: el nodo de contexto exige que todo descarte quede justificado. `tokens` puede ser `NULL` si no se pudo estimar el tamaño.

### `change_history`

| Columna | Qué representa | Valores permitidos | Nullable | Default |
|---|---|---|---|---|
| `id` | Identificador del cambio | UUID v4 | No | — |
| `session_id` | Sesión que aplicó el cambio | UUID v4 de `sessions.id` | Sí | `NULL` |
| `operation` | Qué operación se hizo | Ver [[database/01-schema/ENUMS]] | No | — |
| `file_path` | Archivo afectado | Ruta relativa al proyecto | No | — |
| `before_content` | Contenido previo | Texto libre | Sí | `NULL` |
| `after_content` | Contenido nuevo | Texto libre | Sí | `NULL` |
| `created_at` | Cuándo se aplicó | ISO 8601 UTC | No | — |

`session_id` es `NULL` cuando la sesión que aplicó el cambio ya fue eliminada. El registro sobrevive a la sesión porque esta tabla nunca se borra. `before_content` es `NULL` si el archivo no existía y `after_content` es `NULL` si el archivo fue eliminado.

### `todos`

| Columna | Qué representa | Valores permitidos | Nullable | Default |
|---|---|---|---|---|
| `session_id` | Sesión dueña de la lista | UUID v4 de `sessions.id` | No | — |
| `position` | Orden del paso dentro de la lista | Entero ≥ 0 | No | — |
| `parent_position` | Posición de la tarea principal que agrupa este paso | `NULL` o `position` de otra fila de la misma sesión | Sí | `NULL` |
| `content` | Qué hay que hacer | Texto libre | No | — |
| `status` | Estado del paso | `pendiente`, `en_progreso`, `completada`, `cancelada` | No | — |
| `priority` | Prioridad del paso | `alta`, `media`, `baja` | No | `media` |
| `created_at` | Cuándo se escribió la lista | ISO 8601 UTC | No | — |
| `updated_at` | Última reescritura | ISO 8601 UTC | No | — |

La clave primaria es `(session_id, position)` y no hay `id` de fila: los pasos se numera por su índice, así que no hay identificadores que puedan quedar obsoletos. `session_id` es `NOT NULL` con `ON DELETE CASCADE`: borrar la sesión se lleva su lista.

**Las dos herramientas de la lista escriben de forma distinta, y la tabla solo lo nota como filas.** `actualizar_todo` borra la lista de la sesión y la reinserta con sus posiciones, en un solo paso atómico: o queda la lista nueva entera, o queda la anterior. `crear_todo` inserta **una o varias filas** en la posición siguiente —un paso suelto o una tarea principal con su grupo de subtareas—, y con eso añade: no borra nada, así que no puede perder los pasos que ya estaban ni dejar la lista a medias.

Que `crear_todo` inserte y no reescriba es justo lo que hace segura la operación: añadir no tiene forma de degenerar en «se me olvidó alguno». Y como la clave primaria es `(session_id, position)`, añadir es insertar en `position` = número de filas actuales; no hace falta renumerar nada porque nada se mueve.

**La jerarquía es de un nivel y vive en `parent_position`.** Un paso raíz la lleva en `NULL`; un paso agrupado apunta a la `position` de su tarea principal dentro de la misma sesión. No es una clave foránea declarada —SQLite no puede añadir por `ALTER TABLE` una autorreferencia compuesta ni un `CHECK` que mire otras filas—, así que la validez de la referencia la impone la capa de datos al escribir. El índice `idx_todos_padre` de [[database/01-schema/INDEXES]] sirve justo para leer el grupo de una tarea principal y para verificar esa referencia. Ver [[database/02-rules/BUSINESS_RULES]].

La jerarquía es un cambio de esquema: la columna `parent_position` la añade la migración `009-todo-jerarquia` ([[database/03-operations/MIGRATIONS]], pendiente de la tarea T-B022), nullable y sin valor por defecto, así que las filas anteriores quedan con `NULL`, que es exactamente «paso raíz». Una base v8 se lee como una lista plana, sin ningún cambio de comportamiento: la jerarquía es una forma nueva de crear la lista, no una lectura nueva ([[specs/SPEC-TOOLS]] §Criterios de aceptación).

### `flow_context`

| Columna | Qué representa | Valores permitidos | Nullable | Default |
|---|---|---|---|---|
| `id` | Identificador de la aportación | UUID v4 | No | — |
| `session_id` | Sesión que ejecutó el flujo | UUID v4 de `sessions.id` | No | — |
| `flow` | Flujo al que pertenece el bloque | Nombre del flujo (p. ej. `resolver`) | No | — |
| `stage` | Etapa que produjo la aportación | Id de la etapa | No | — |
| `stage_name` | Nombre visible de la etapa | Texto libre | No | — |
| `position` | Orden de la etapa en el flujo | Entero ≥ 0 | No | — |
| `content` | Aportación ya optimizada | Texto libre | No | — |
| `created_at` | Cuándo se guardó | ISO 8601 UTC | No | — |

Hay una aportación por etapa y flujo dentro de la sesión: el índice único `(session_id, flow, stage)` hace que volver a ejecutar el flujo **reemplace** la fila en vez de duplicarla. El orquestador vacía el bloque al arrancar cada ejecución. `session_id` es `NOT NULL` con `ON DELETE CASCADE`: borrar la sesión se lleva su bloque. Ver [[specs/SPEC-ORQUESTADOR-FLUJOS]].

### `chat_evento`

| Columna | Qué representa | Valores permitidos | Nullable | Default |
|---|---|---|---|---|
| `id` | Identificador de la línea | UUID v4 | No | — |
| `session_id` | Sesión a la que pertenece el hilo | UUID v4 de `sessions.id` | No | — |
| `tipo` | Qué clase de pieza es | `proceso`, `herramienta`, `respuesta` | No | — |
| `content` | La línea tal como se pinta | Texto libre | No | — |
| `bloque` | El bloque visual ya renderizado que se pinta bajo la línea | Texto libre | Sí | `NULL` |
| `bloque_tipo` | Qué clase de bloque es | `diagrama`, `diferencia` | Sí | `NULL` |
| `duration_ms` | Cuánto tardó lo que la línea describe, en milisegundos | Entero ≥ 0 | Sí | `NULL` |
| `created_at` | Cuándo se añadió al hilo | ISO 8601 UTC | No | — |
| `call_id` | Id de la llamada que pidió la herramienta | Texto libre | Sí | `NULL` |
| `herramienta` | Nombre de la herramienta ejecutada | Texto libre | Sí | `NULL` |
| `argumentos` | Argumentos con los que se pidió | JSON | Sí | `NULL` |
| `salida` | Lo que devolvió la herramienta | Texto libre | Sí | `NULL` |
| `error` | El error corregible, si falló | Texto libre | Sí | `NULL` |
| `ok` | Si la ejecución terminó bien | `0` o `1` | Sí | `NULL` |
| `truncado` | Si la salida se recortó | `0` o `1` | Sí | `NULL` |

`tipo` distingue el sub-proceso de una etapa (`proceso`) de la línea de una herramienta (`herramienta`) y de la respuesta completa de una etapa con `respuesta_en_chat` (`respuesta`). `content` guarda el texto ya compuesto —«[Sub Proceso] Entender el problema», «✓ LEER [AGENTS.md] · 93 líneas» o la respuesta entera de la etapa—, para que al recargar la sesión el hilo se pinte igual que en vivo. En una lectura por rango el tema incluye el rango real leído (por ejemplo `AGENTS.md:10-50`). Las piezas son inmutables una vez completas, como los mensajes: la línea de una herramienta se guarda cerrada, con su marca, su medida y su tiempo; la respuesta de una etapa se guarda completa. `session_id` es `NOT NULL` con `ON DELETE CASCADE`: borrar la sesión se lleva su hilo.

`bloque` y `bloque_tipo` guardan el bloque visual que se pinta bajo la línea: el diagrama de `diagram` (`diagrama`) o el antes y el después de un cambio aplicado (`diferencia`). El texto llega ya renderizado —la TUI solo lo coloca— y `bloque_tipo` dice cómo pintarlo. Como puede repetir contenido del archivo, el bloque se recorta a un tope y el antes y el después completos siguen en `change_history`; el bloque no entra al contexto del modelo, se guarda solo para que el hilo recuperado se lea igual que en vivo. Las filas anteriores a la migración `013-bloque-linea` quedan con ambos en `NULL` y se pintan sin bloque.

Las columnas `call_id`, `herramienta`, `argumentos`, `salida`, `error`, `ok` y `truncado` se llenan en las filas `tipo = herramienta` que corren en el turno principal: son la llamada y su desenlace, y permiten reconstruir el par `assistant(tool_calls)` + `tool(resultado)` que el modelo ve en los turnos siguientes. Las herramientas que corren dentro de una etapa de un flujo también dejan su línea, pero quedan en `NULL`: su resultado lo resume el `ejecutar_flujo` que las lanzó y no se reenvían por separado. Las filas anteriores a la migración `012-ejecucion-herramienta` quedan en `NULL` y se siguen pintando, pero no se reenvían. Los sub-procesos y las respuestas de etapa no tienen ejecución y también quedan en `NULL`. La salida reenviada se acota en la capa de sesión; `truncado` conserva si la herramienta ya había recortado su propia salida.

`duration_ms` es el tiempo de **lo que esa pieza describe**, no el del turno: en una línea de herramienta es el de su ejecución, y en las demás no se mide y queda `NULL`. Sigue la misma convención que `messages.duration_ms` —`NULL` es «no se midió», no «tardó cero»—, y como cada pieza ya guarda su texto compuesto, el tiempo está ahí dos veces: como número, para poder consultarlo, y dentro de `content`, para que pintar el hilo no dependa de recomponerlo. Ver [[database/02-rules/DATA_FLOW]] y [[database/03-operations/MIGRATIONS]] §Nota sobre el esquema actual.

## 4. Relaciones

- `projects` 1:N `sessions`: una carpeta tiene muchas sesiones y cada sesión pertenece a un solo proyecto.
- `sessions` 1:N `messages`; `messages` 1:1 opcional `reasoning`.
- `sessions` 1:N `approvals`.
- `sessions` 1:N `context_audit`.
- `sessions` 1:N `flow_context`.
- `sessions` 1:N `todos`.
- `sessions` 1:N `chat_evento`.
- `sessions` 1:N opcional `change_history`, que sobrevive a la sesión.

El detalle de cardinalidades, claves foráneas y cascadas está en [[database/01-schema/RELATIONSHIPS]].

## 5. Ejemplo conceptual

Dos sesiones de un mismo proyecto, una de backend trabajando y otra de frontend esperando permiso. Un mensaje de agente con su razonamiento, una aprobación pendiente y un cambio de archivo ya aplicado.

```
projects
  p1 | "/home/ana/proyectos/pedidos" | 2026-01-15T08:55:00Z | 2026-01-15T10:12:00Z

sessions
  s1 | p1 | "backend-pedidos"  | backend | trabajando    | 2026-01-15T09:00:00Z | 2026-01-15T10:12:00Z
  s2 | p1 | "frontend-carrito" | frontend| esperando_permiso | 2026-01-15T09:05:00Z | 2026-01-15T10:10:00Z

messages
  m1 | s1 | user  | "añade un endpoint de pedidos"          |  null | null | 09:00
  m2 | s1 | agent | "propongo endpoint GET /pedidos"          |  820 |  140 | 09:01
  m3 | s2 | user  | "aplica el diseño del carrito"           |  null | null | 09:05

reasoning
  r1 | m2 | "el usuario pide pedidos; el schema ya tiene la tabla..." | 09:01

approvals
  a1 | s2 | "escribir frontend/Carrito.tsx" | pendiente | 10:10 | NULL

context_audit
  ca1| s1 | "implementar-endpoint" | "backend/PEDIDOS.md"        | incluido  | NULL  | 2100 | 09:01
  ca2| s1 | "implementar-endpoint" | "database/PEDIDOS.md"      | incluido  | NULL  | 1800 | 09:01
  ca3| s1 | "implementar-endpoint" | "frontend/CARRITO.md"      | descartado| "el endpoint es de backend" | 1500 | 09:01

change_history
  h1 | s1 | editar      | "backend/pedidos.go" | "old content..." | "new content..." | 09:40
  h2 | NULL| crear       | "backend/pedidos_test.go" | NULL     | "package..."       | 09:41
```

`h2` conserva `session_id` a `NULL` porque su sesión fue eliminada después, y el registro sigue ahí.
