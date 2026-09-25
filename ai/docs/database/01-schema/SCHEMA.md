# SCHEMA — Estructura de la base de datos

## 1. Estructura global

La base de datos de LocalCli es un archivo SQLite por proyecto, en la carpeta de estado de la carpeta abierta. El modo WAL está activo. No hay servidor de base de datos.

El esquema tiene dos naturalezas y conviene no confundirlas:

- **Estado en SQLite.** Sesiones, conversaciones, aprobaciones, auditoría de contexto e historial de cambios. Son datos que nacen y mueren con la ejecución.
- **Datos en archivos.** La documentación del proyecto y los archivos de tarea son la fuente de verdad y viven fuera de SQLite. El grafo de documentos y el orden de la cola se derivan de ellos y se reconstruyen al arrancar, sin tabla propia.

SQLite solo modela el primer grupo. El segundo se documenta en [[database/02-rules/DATA_FLOW]].

No hay usuarios ni autenticación: el proyecto es una carpeta y el acceso lo controla el sistema de archivos. El aislamiento entre proyectos es físico, un archivo por carpeta.

### Convenciones

- **Clave primaria.** Todas las tablas usan `id` de tipo `TEXT` con un UUID v4. No hay enteros autoincrementales: los identificadores se referencian en logs, aprobaciones y tareas, y un UUID opaco evita depender de una secuencia.
- **Fechas.** `created_at` y `updated_at` son `TEXT` en ISO 8601 con zona UTC (`2026-01-15T10:30:00Z`). Legible en crudo y sin ambigüedad de zona.
- **Booleanos.** Donde hacen falta se usan enteros `0` y `1`, no `BOOLEAN`, que SQLite no tiene.
- **Ausencia de valor.** Una columna opcional se deja `NULL`. No se usan cadenas vacías como sustituto de "sin valor".

La versión del esquema no ocupa tabla: se guarda en el `PRAGMA user_version` de SQLite. Ver [[database/03-operations/MIGRATIONS]].

## 2. Tablas

| Tabla | Propósito |
|---|---|
| `sessions` | Una sesión de trabajo: nombre, capa y estado |
| `messages` | Los mensajes de la conversación, de usuario y de agente |
| `reasoning` | El razonamiento del modelo de cada mensaje de agente, acumulado del streaming |
| `approvals` | Las aprobaciones pendientes y resueltas de cada sesión |
| `context_audit` | Qué documentación entró o salió del contexto de cada etapa, y por qué |
| `change_history` | Qué archivo se cambió, qué había antes y qué quedó |

El detalle de cada columna está en [[database/01-schema/TABLES]]. Las relaciones entre ellas, en [[database/01-schema/RELATIONSHIPS]]. Los valores cerrados, en [[database/01-schema/ENUMS]].

## 3. Campos

Resumen estructural. Los valores permitidos, los `NULL` y los valores por defecto de cada columna están en [[database/01-schema/TABLES]].

### `sessions`

| Campo | Tipo | Clave | Notas |
|---|---|---|---|
| `id` | TEXT | PK | UUID v4 |
| `name` | TEXT | | Nombre de la sesión |
| `layer` | TEXT | | Capa en la que trabaja. `NULL` si la sesión es general |
| `status` | TEXT | | Ver [[database/01-schema/ENUMS]] |
| `created_at` | TEXT | | ISO 8601 UTC |
| `updated_at` | TEXT | | ISO 8601 UTC |

### `messages`

| Campo | Tipo | Clave | Notas |
|---|---|---|---|
| `id` | TEXT | PK | UUID v4 |
| `session_id` | TEXT | FK → `sessions.id` | `ON DELETE CASCADE` |
| `role` | TEXT | | `user` o `agent` |
| `content` | TEXT | | Texto del mensaje |
| `input_tokens` | INTEGER | | `NULL` si el modelo no lo reporta |
| `output_tokens` | INTEGER | | `NULL` si el modelo no lo reporta |
| `created_at` | TEXT | | ISO 8601 UTC |

### `reasoning`

| Campo | Tipo | Clave | Notas |
|---|---|---|---|
| `id` | TEXT | PK | UUID v4 |
| `message_id` | TEXT | FK → `messages.id` | `ON DELETE CASCADE` |
| `content` | TEXT | | Razonamiento acumulado del streaming |
| `created_at` | TEXT | | ISO 8601 UTC |

### `approvals`

| Campo | Tipo | Clave | Notas |
|---|---|---|---|
| `id` | TEXT | PK | UUID v4 |
| `session_id` | TEXT | FK → `sessions.id` | `ON DELETE CASCADE` |
| `description` | TEXT | | Qué propone la aprobación |
| `status` | TEXT | | `pendiente`, `aprobada`, `declinada` u `obsoleta` |
| `created_at` | TEXT | | ISO 8601 UTC |
| `resolved_at` | TEXT | | `NULL` mientras siga pendiente |

### `context_audit`

| Campo | Tipo | Clave | Notas |
|---|---|---|---|
| `id` | TEXT | PK | UUID v4 |
| `session_id` | TEXT | FK → `sessions.id` | `ON DELETE CASCADE` |
| `stage` | TEXT | | Etapa del flujo a la que pertenece |
| `document` | TEXT | | Ruta del documento, relativa al proyecto |
| `decision` | TEXT | | `incluido` o `descartado` |
| `reason` | TEXT | | Motivo del descarte. `NULL` si se incluyó |
| `tokens` | INTEGER | | Tamaño estimado del documento en el contexto |
| `created_at` | TEXT | | ISO 8601 UTC |

### `change_history`

| Campo | Tipo | Clave | Notas |
|---|---|---|---|
| `id` | TEXT | PK | UUID v4 |
| `session_id` | TEXT | FK → `sessions.id` | `NULL` si la sesión ya no existe. `ON DELETE SET NULL` |
| `operation` | TEXT | | Operación de archivo aplicada |
| `file_path` | TEXT | | Ruta del archivo, relativa al proyecto |
| `before_content` | TEXT | | `NULL` si el archivo no existía |
| `after_content` | TEXT | | `NULL` si el archivo fue eliminado |
| `created_at` | TEXT | | ISO 8601 UTC |

`change_history` es la única tabla cuya `session_id` no se borra en cascada: usa `ON DELETE SET NULL`. Así, al eliminar una sesión de forma definitiva, el registro de los cambios que aplicó en los archivos del proyecto sobrevive, tal como exige [[specs/SPEC-ARCHIVOS]].

## Referencias

- [[database/DATABASE]] — panorama de los dos planos y las políticas.
- [[database/01-schema/TABLES]] — detalle por columna.
- [[database/01-schema/RELATIONSHIPS]] — relaciones y cascadas.
- [[database/01-schema/ENUMS]] — valores permitidos y transiciones.
- [[database/01-schema/CONSTRAINTS]] — restricciones que impone la base.
- [[database/01-schema/INDEXES]] — índices y consultas que optimizan.
- [[database/03-operations/MIGRATIONS]] — gestión del `user_version`.
