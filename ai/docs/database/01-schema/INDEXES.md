# INDEXES — Índices de la base de datos

## 1. Índices existentes

Los índices se crean en la definición del esquema y se gestionan con las migraciones de [[database/03-operations/MIGRATIONS]]. Las claves primarias de texto y las claves foráneas crean sus propios índices en SQLite, así que aquí solo se listan los explícitos y los compuestos que hacen falta para las consultas de la aplicación.

| Índice | Tabla | Tipo |
|---|---|---|
| `idx_messages_session_created` | `messages` | Compuesto |
| `idx_reasoning_message` | `reasoning` | Simple |
| `idx_approvals_session_status` | `approvals` | Compuesto |
| `idx_approvals_pending` | `approvals` | Simple, parcial |
| `idx_context_audit_session_stage` | `context_audit` | Compuesto |
| `idx_change_history_file` | `change_history` | Simple |
| `idx_sessions_status` | `sessions` | Simple |
| `idx_sessions_updated` | `sessions` | Simple |

## 2. Consultas que optimiza cada índice

| Índice | Consulta que acelera | Quién la ejecuta |
|---|---|---|
| `idx_messages_session_created` | Cargar el historial de una sesión en orden cronológico | La interfaz al abrir o cambiar de sesión |
| `idx_reasoning_message` | Leer el razonamiento de un mensaje | La interfaz al reabrir una respuesta |
| `idx_approvals_session_status` | Ver las aprobaciones de una sesión filtradas por estado | El motor al reanudar un flujo pausado |
| `idx_approvals_pending` | Contar y listar todas las aprobaciones pendientes de cualquier sesión | El panel de aprobaciones y el contador que se ve con el panel de datos cerrado |
| `idx_context_audit_session_stage` | Ver qué documentación recibió una etapa concreta | El usuario al auditar una etapa |
| `idx_change_history_file` | Ver el historial de cambios de un archivo concreto | El usuario al revertir o auditar un archivo |
| `idx_sessions_status` | Listar las sesiones por estado, para el selector | El selector de sesiones |
| `idx_sessions_updated` | Listar las sesiones por actividad reciente | El selector, ordenado por última actividad |

El más importante es `idx_approvals_pending`. La consulta que cuenta las aprobaciones pendientes se ejecuta en cada redibujado de la interfaz para mantener el contador al día, así que necesita ser rápida. Es un índice parcial, `WHERE status = 'pendiente'`, así que solo contiene las filas que importan y ocupa poco.

## 3. Campos indexados

| Índice | Columnas | Notas |
|---|---|---|
| `idx_messages_session_created` | `session_id`, `created_at` | El orden del segundo campo es lo que da el orden cronológico sin `ORDER BY` extra |
| `idx_reasoning_message` | `message_id` | Admite el `UNIQUE` de un razonamiento por mensaje |
| `idx_approvals_session_status` | `session_id`, `status` | Permite filtrar por ambos a la vez |
| `idx_approvals_pending` | `created_at` | Parcial: solo filas con `status = 'pendiente'` |
| `idx_context_audit_session_stage` | `session_id`, `stage` | Agrupa la auditoría por etapa |
| `idx_change_history_file` | `file_path` | Historial de un archivo |
| `idx_sessions_status` | `status` | Filtro por estado |
| `idx_sessions_updated` | `updated_at` | Orden por actividad |

El índice parcial `idx_approvals_pending` ordena por `created_at` y no por `status`, porque el `status` ya está fijado por la propia condición del índice. Indexar la columna que el índice filtra sería redundante.

## 4. Unicidad

| Restricción | Índice que la sustenta |
|---|---|
| `id` único de cada tabla | Índice primario de la clave primaria |
| Un solo razonamiento por mensaje | `idx_reasoning_message`, declarado `UNIQUE` |
| Restricciones `CHECK` de valores | No usan índice; las valida la base al escribir |

El `UNIQUE` de `idx_reasoning_message` es lo que hace efectiva la relación 1:1 entre `messages` y `reasoning` descrita en [[database/01-schema/RELATIONSHIPS]]. Sin él, la base permitiría dos razonamientos para el mismo mensaje.

Los índices no imponen unicidad de negocio más allá de esas dos. No hay índices únicos sobre nombres de sesión ni sobre rutas de archivo, porque nada en las reglas del proyecto lo exige. Dos sesiones pueden llamarse igual y un archivo puede cambiarse muchas veces.

## Notas de diseño

- **Los índices van en la base, no en el código.** La aplicación no crea índices en tiempo de ejecución. Si un índice falta, se añade con una migración.
- **Nada de índices sobre el contenido de texto.** No hay búsquedas de texto completo en la base. La búsqueda por contenido de documentación se hace sobre los archivos, no sobre SQLite, y no es una operación de la base de datos.
- **WAL y los índices.** Con el modo WAL activo, los Readers no bloquean al escritor. Los índices siguen ayudando, pero el cuello de botella real son las escrituras concurrentes de varias sesiones, no las lecturas.

## Referencias

- [[database/01-schema/SCHEMA]] — estructura de las tablas.
- [[database/01-schema/TABLES]] — columnas indexadas.
- [[database/01-schema/RELATIONSHIPS]] — relaciones que los índices aceleran.
- [[database/01-schema/CONSTRAINTS]] — unicidad y claves foráneas.
- [[database/03-operations/QUERIES]] — las consultas que aprovechan estos índices.
- [[database/03-operations/MIGRATIONS]] — cómo se crean los índices.
