---
title: LocalCli — consultas a la base de datos
tags: [database, operaciones]
depende_de:
  - "[[database/01-schema/SCHEMA]]"
  - "[[database/01-schema/TABLES]]"
  - "[[database/01-schema/INDEXES]]"
relacionado:
  - "[[database/01-schema/ENUMS]]"
  - "[[database/01-schema/RELATIONSHIPS]]"
  - "[[database/02-rules/BUSINESS_RULES]]"
  - "[[database/02-rules/DATA_FLOW]]"
  - "[[specs/SPEC-NODO-CONTEXTO]]"
  - "[[specs/SPEC-PANEL-CONTEXTO]]"
  - "[[specs/SPEC-INTERFAZ-ATAJOS]]"
---
# QUERIES — Consultas a la base de datos

## 1. Consultas complejas

La base es pequeña y las consultas son directas. No hay necesidad de `JOIN` complejo, salvo el filtro constante por proyecto. Estas son las que merecen atención.

### Resolver el proyecto de la carpeta abierta

Antes que nada, traducir el directorio actual a su `project_id`. Si no existe la fila, se crea.

```sql
SELECT id FROM projects WHERE path = ?;
```

Acelera `idx_projects_path`. Es la primera consulta del arranque y de la que depende el aislamiento: sin este `project_id`, cualquier consulta de sesiones podría cruzar carpetas. Ver [[database/02-rules/BUSINESS_RULES]].

### Sesiones del proyecto actual

Alimenta el selector de sesiones, acotado a la carpeta abierta.

```sql
SELECT id, name, layer, status, created_at, updated_at
FROM sessions
WHERE project_id = ?
ORDER BY updated_at DESC;
```

Acelera `idx_sessions_project`. El filtro `project_id` no es opcional: sin él se verían las sesiones de otros proyectos.

### Historial de una sesión, en orden

Cargar la conversación de una sesión, con el razonamiento de cada mensaje de agente.

```sql
SELECT m.id, m.role, m.content, m.input_tokens, m.output_tokens, m.created_at,
       r.content AS reasoning
FROM messages m
LEFT JOIN reasoning r ON r.message_id = m.id
WHERE m.session_id = ?
ORDER BY m.created_at;
```

El `LEFT JOIN` con `reasoning` es necesario porque un mensaje de agente puede no tener razonamiento, si el modelo no lo devolvió. Un `INNER JOIN` perdería esos mensajes. Ver [[database/01-schema/ENUMS]].

### Aprobaciones pendientes del proyecto actual

La consulta que alimenta el panel de aprobaciones y el contador que se ve con el panel de datos cerrado.

```sql
SELECT a.id, a.session_id, s.name AS session_name, a.description, a.created_at
FROM approvals a
JOIN sessions s ON s.id = a.session_id
WHERE a.status = 'pendiente' AND s.project_id = ?
ORDER BY a.created_at;
```

Acelera el índice parcial `idx_approvals_pending` para el estado y `idx_sessions_project` para el proyecto. La ejecuta el panel global, así que abarca las sesiones de cualquier capa **del proyecto actual**, nunca las de otra carpeta. Ver [[specs/SPEC-INTERFAZ-ATAJOS]].

### Auditoría de una etapa concreta

Ver qué documentación recibió el modelo en una etapa, con lo incluido y lo descartado y su motivo.

```sql
SELECT document, decision, reason, tokens
FROM context_audit
WHERE session_id = ? AND stage = ?
ORDER BY decision, document;
```

Acelera `idx_context_audit_session_stage`. Es la consulta con la que el usuario audita por qué una respuesta fue mala. Ver [[specs/SPEC-NODO-CONTEXTO]].

### Historial de cambios de un archivo

Para revertir o auditar un archivo concreto.

```sql
SELECT id, session_id, operation, before_content, after_content, created_at
FROM change_history
WHERE file_path = ?
ORDER BY created_at;
```

Acelera `idx_change_history_file`. Devuelve cambios de sesiones distintas, y está bien: el archivo cambió aunque la sesión que lo cambió ya no exista.

## 2. Patrones de acceso

| Patrón | Tabla | Quién |
|---|---|---|
| Leer por sesión | `messages`, `reasoning` | La TUI al abrir una sesión |
| Contar pendientes | `approvals` | El panel y el contador, en cada redibujado |
| Escribir estado | `sessions` | El orquestador de flujos, en cada transición |
| Escribir streaming | `reasoning` | El modelo, token a token |
| Registrar y leer auditoría | `context_audit` | El nodo de contexto y el usuario |
| Registrar y leer historial | `change_history` | Las herramientas de archivo y el usuario |
| Resolver el proyecto de la carpeta | `projects` | El arranque, una vez por apertura |
| Listar por estado y actividad | `sessions` | El selector de sesiones, acotado a `project_id` |

El patrón dominante es **leer por sesión** y **escribir por operación**. Casi ninguna consulta cruza más de dos tablas. La excepción estructural es el filtro por `project_id` en las consultas de sesiones: aparece siempre que se listan o agregan datos de la carpeta actual.

## 3. Joins y agregaciones

- El historial de una sesión hace un `LEFT JOIN` entre `messages` y `reasoning`. Es el único `JOIN` de la aplicación que une dos tablas hijas de la misma sesión.
- El panel de aprobaciones hace un `JOIN` de `approvals` con `sessions` para mostrar el nombre de la sesión que espera. Necesita el nombre, que solo está en `sessions`, y el `project_id`, que es lo que acota el panel a la carpeta actual.
- El historial de un archivo no hace `JOIN`: si la sesión ya no existe, `session_id` es `NULL` y no hay nada que unir. Ver [[database/01-schema/RELATIONSHIPS]].
- **No hay agregaciones.** No se cuenta ni se suma nada en la base. El contador de aprobaciones pendientes cuenta filas, y el panel de contexto calcula la ocupación del contexto en memoria a partir de los tokens que trae el modelo. Ver [[specs/SPEC-PANEL-CONTEXTO]].

## 4. Consultas reutilizables

Estas consultas se usan desde más de un sitio y conviene tenerlas en un solo lugar:

| Consulta | Dónde se usa |
|---|---|
| Resolver el proyecto de la carpeta abierta | El arranque y cualquier consulta que liste sesiones |
| Contar aprobaciones pendientes | El panel de aprobaciones y el contador del panel de datos |
| Historial de una sesión | La TUI al abrir y al cambiar de sesión |
| Auditoría de una etapa | El registro que consulta el usuario |
| Historial de cambios de un archivo | La reversión y la auditoría de archivos |
| Listar sesiones por actividad | El selector de sesiones, con `project_id` |

Ninguna consulta se escribe dos veces en el código. Vienen de la base de datos en [[database/01-schema/SCHEMA]] y no leen nada de la base que no esté en esa estructura.

## 5. Casos a evitar

- **No contar approvals pendientes con un `SELECT COUNT(*)` sobre toda la tabla.** Se filtra por `status = 'pendiente'` para aprovechar el índice parcial. Contar sobre la tabla entera es más lento y no cambia el resultado.
- **No cargar el historial completo de una sesión para mostrar la última línea.** Se pide el último mensaje con `ORDER BY created_at DESC LIMIT 1`. Cargar todo para descartar casi todo llena la memoria y el contexto sin necesidad.
- **No recuperar el contenido del razonamiento si no se va a mostrar.** El `LEFT JOIN` de `reasoning` trae el texto completo. Para listar mensajes sin razonamiento, se consulta `messages` sin el `JOIN`. El razonamiento puede ser largo y se paga en memoria.
- **No hacer `JOIN` de `change_history` con `sessions` para saber qué sesión hizo un cambio.** Si la sesión fue borrada, el `JOIN` devuelve nada y se pierde el registro. El `session_id` se muestra tal cual, o `NULL`. Ver [[database/01-schema/RELATIONSHIPS]].
- **No escribir en `change_history` desde la base directamente.** Solo el módulo de herramientas de archivo escribe ahí, y solo tras una aprobación. Ver [[database/02-rules/BUSINESS_RULES]].
- **No listar ni contar sesiones sin filtrar por `project_id`.** La base es única y global: una consulta de sesiones sin el filtro de proyecto devuelve las de todas las carpetas. Es la única regla que no admite excepción, porque el aislamiento es lógico y depende de esa cláusula. Ver [[database/02-rules/BUSINESS_RULES]].
- **No borrar `change_history`.** No hay caso en el que se borre. Si parece que hay, el problema es el diseño de la consulta, no la operación.

## Referencias

- [[database/01-schema/TABLES]] — columnas usadas en las consultas.
- [[database/01-schema/RELATIONSHIPS]] — las uniones y su cardinalidad.
- [[database/01-schema/INDEXES]] — índices que estas consultas aprovechan.
- [[database/01-schema/ENUMS]] — valores por los que se filtra.
- [[database/02-rules/DATA_FLOW]] — quién escribe y quién lee.
- [[database/02-rules/BUSINESS_RULES]] — reglas que condicionan algunas consultas.
