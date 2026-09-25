# TABLES — Tablas de la base de datos

## 1. Tablas

El esquema tiene seis tablas. La estructura de alto nivel está en [[database/01-schema/SCHEMA]]; aquí cada columna en detalle. Las relaciones entre tablas están en [[database/01-schema/RELATIONSHIPS]] y los valores cerrados, en [[database/01-schema/ENUMS]].

Convenciones que aplican a todas: `id` es `TEXT` con UUID v4, las fechas son `TEXT` en ISO 8601 UTC, y los booleanos son enteros `0`/`1`.

## 2. Propósito de cada tabla

- **`sessions`** — Una sesión de trabajo. Es la unidad de la que cuelgan la conversación, las aprobaciones y la auditoría. Su campo `status` es lo que la interfaz muestra cuando cambias de sesión.
- **`messages`** — Los turnos de la conversación. Guarda lo que escribió el usuario y lo que respondió el agente, más los tokens de cada turno para el panel de contexto.
- **`reasoning`** — El razonamiento del modelo de un mensaje concreto. Va en tabla aparte porque llega token a token mientras se genera y porque se consulta y se mide por separado del texto final. El diseño está en [[specs/SPEC-AGENTE-BASE]].
- **`approvals`** — Lo que una sesión necesita que decidas antes de seguir. El panel de aprobaciones las lee todas, de cualquier sesión. Ver [[specs/SPEC-INTERFAZ-ATAJOS]].
- **`context_audit`** — La traza de qué documentación recibió el modelo en cada etapa y qué se descartó, con el motivo. Es lo que hace auditable el nodo de contexto. Ver [[specs/SPEC-NODO-CONTEXTO]].
- **`change_history`** — Cada cambio aplicado a un archivo del proyecto, con lo que había antes y lo que quedó. Nunca se borra. Ver [[database/02-rules/BUSINESS_RULES]].

## 3. Columnas

### `sessions`

| Columna | Qué representa | Valores permitidos | Nullable | Default |
|---|---|---|---|---|
| `id` | Identificador de la sesión | UUID v4 | No | — |
| `name` | Nombre visible de la sesión | Texto libre | No | — |
| `layer` | Capa en la que trabaja la sesión | `backend`, `frontend` | Sí | `NULL` |
| `status` | Estado de la sesión | Ver [[database/01-schema/ENUMS]] | No | `inactiva` |
| `created_at` | Cuándo se creó | ISO 8601 UTC | No | — |
| `updated_at` | Última actividad | ISO 8601 UTC | No | — |

`layer` es `NULL` para una sesión general, por ejemplo cuando solo conversas o revisas contexto sin ejecutar tareas de una capa concreta.

### `messages`

| Columna | Qué representa | Valores permitidos | Nullable | Default |
|---|---|---|---|---|
| `id` | Identificador del mensaje | UUID v4 | No | — |
| `session_id` | Sesión a la que pertenece | UUID v4 de `sessions.id` | No | — |
| `role` | Quién escribió | `user`, `agent` | No | — |
| `content` | Texto del mensaje | Texto libre | No | — |
| `input_tokens` | Tokens consumidos en este turno | Entero ≥ 0 | Sí | `NULL` |
| `output_tokens` | Tokens generados en este turno | Entero ≥ 0 | Sí | `NULL` |
| `created_at` | Cuándo se escribió el mensaje | ISO 8601 UTC | No | — |

Los dos campos de tokens son `NULL` cuando el modelo no los reporta. Se distinguen del `0`: `NULL` significa "no lo sé", `0` significa "cero". El panel de contexto marca la estimación como tal. Ver [[specs/SPEC-PANEL-CONTEXTO]].

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

## 4. Relaciones

- `sessions` 1:N `messages`; `messages` 1:1 opcional `reasoning`.
- `sessions` 1:N `approvals`.
- `sessions` 1:N `context_audit`.
- `sessions` 1:N opcional `change_history`, que sobrevive a la sesión.

El detalle de cardinalidades, claves foráneas y cascadas está en [[database/01-schema/RELATIONSHIPS]].

## 5. Ejemplo conceptual

Dos sesiones de un mismo proyecto, una de backend trabajando y otra de frontend esperando permiso. Un mensaje de agente con su razonamiento, una aprobación pendiente y un cambio de archivo ya aplicado.

```
sessions
  s1 | "backend-pedidos"  | backend | trabajando    | 2026-01-15T09:00:00Z | 2026-01-15T10:12:00Z
  s2 | "frontend-carrito" | frontend| esperando_permiso | 2026-01-15T09:05:00Z | 2026-01-15T10:10:00Z

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
