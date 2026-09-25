# DATA_FLOW — Circulación de datos

## 1. Circulación de datos

Los datos de LocalCli vienen de dos sitios y van a dos sitios. Entender de dónde sale cada cosa evita tratarlos igual cuando no lo son.

```
Documentación y tareas (archivos)  ──┐
                                     ├──▶  Nodo de contexto  ──▶  Modelo
                                     │         │
                                     │         ▼
                                     │    context_audit
                                     │
Usuario (teclado) ──▶ TUI ──▶ Sesión ──▶ Agent ──▶ Herramientas
                                 │                         │
                                 │                         ▼
                                 │                   Archivos del proyecto
                                 │                         │
                                 │                         ▼
                                 │                   change_history
                                 ▼
                     sessions, messages, reasoning, approvals
```

El contexto que consume el modelo no sale de la base: sale de los archivos del proyecto. La base guarda el rastro de qué se eligió, no el contenido. Ver [[specs/SPEC-NODO-CONTEXTO]].

## 2. Los dos planos y de dónde viene cada dato

### Archivos: fuente de verdad

| Dato | Dónde vive | Cómo se usa |
|---|---|---|
| Documentación del proyecto | `ai/docs/` y la documentación de las capas | El nodo de contexto la lee para armar el contexto de cada etapa |
| Archivos de tarea | `ai/tasks/<capa>/NNN-task-<nombre>.md` | La cola los lee para saber qué hacer |

**Documentación.** Cada archivo declara en su frontmatter de qué depende, mediante enlaces. Eso forma el grafo: un documento que declara que usa las entidades de otro apunta al que las define. El grafo se reconstruye leyendo los frontmatter al arrancar.

**Tareas.** Cada archivo declara su id, su capa, su acción, sus dependencias y su estado, más un bloque de contexto. La cola se reconstruye con esos archivos, ordenando por dependencias y respetando el estado.

Los dos se editan a mano y se versionan. Nada de esto se duplica en SQLite: si los planos divergen, manda el archivo. Ver [[database/DATABASE]].

### SQLite: estado de ejecución

| Dato | Quién lo escribe | Para qué |
|---|---|---|
| `sessions` | El módulo de sesión | Saber qué sesiones hay y en qué estado |
| `messages` | El agente y el usuario, al conversar | Reconstruir el historial |
| `reasoning` | El modelo, token a token | Mostrar y auditar el razonamiento |
| `approvals` | El agente, al pedir permiso | Saber qué espera tu decisión |
| `context_audit` | El nodo de contexto | Saber qué documentación entró y qué se descartó |
| `change_history` | Las herramientas de archivo | Poder revertir y auditar cambios |

Nada de esto es fuente de verdad. Es estado que se puede perder y reconstruir, salvo `change_history`, que es permanente por decisión de diseño.

## 3. Creación

**Crear un mensaje.** El usuario escribe, la TUI lo entrega a la sesión y se inserta una fila en `messages` con `role = user`. Si la respuesta del modelo trae razonamiento, se inserta la fila correspondiente en `reasoning` y se va acumulando mientras llega. Al terminar, se inserta el mensaje del agente. Las dos inserciones van en la misma transacción que la actualización de `status` de la sesión.

**Crear una aprobación.** Cuando el agente pide permiso, se inserta una fila en `approvals` con `status = pendiente` y `resolved_at` en `NULL`, y la sesión pasa a `esperando_permiso`. Es un flujo que puede quedar a medias, así que ambas escrituras van en la misma transacción.

**Crear un cambio.** Cuando `build` aplica un cambio aprobado, se escribe el archivo, y en la misma transacción se inserta la fila en `change_history` con lo anterior y lo nuevo. Si la escritura del archivo falla, no se inserta nada. La base registra lo que pasó, no lo que se intentó.

**Crear una auditoría de contexto.** El nodo de contexto decide qué documentos entran en una etapa e inserta una fila por documento en `context_audit`, con `incluido` o `descartado` y su motivo.

## 4. Actualización

- **Estado de la sesión.** Se actualiza `sessions.status` y `sessions.updated_at` en cada cambio: al empezar a trabajar, al pedir permiso, al terminar, al fallar. Las transiciones válidas están en [[database/01-schema/ENUMS]].
- **Acumulación del razonamiento.** La fila en `reasoning` se va actualizando con el texto acumulado. No se inserta una fila por token: es una fila por mensaje que crece.
- **Resolver una aprobación.** Se actualiza `approvals.status` y se rellena `resolved_at` en la misma transacción que cambia el estado de la sesión, para que no queden approvals resueltas con la sesión todavía en `esperando_permiso`.
- **Avanzar la cola.** Cuando una tarea se completa, se actualiza su archivo de tarea, no la base. La cola vuelve a derivarse. El estado de la tarea vive en el archivo, no en SQLite.

## 5. Eliminación

- **Borrar una sesión.** Es definitivo. En cascada se van `messages`, y con ellos `reasoning`, más `approvals` y `context_audit`. Las filas de `change_history` sobreviven con `session_id` en `NULL`. El detalle de las cascadas está en [[database/01-schema/RELATIONSHIPS]].
- **Nunca se borra `change_history`.** No hay ninguna operación que la borre. Es lo que permite revertir y auditar.
- **No se borran filas sueltas de `messages` ni de `reasoning`.** El historial de una sesión se conserva mientras la sesión exista; si quieres quitártelo, borras la sesión entera.

## 6. Transacciones

- **Una transacción por operación lógica.** Un cambio de estado de sesión, una inserción de mensaje, una resolución de aprobación, un cambio de archivo: cada uno es una transacción. Nunca media transacción.
- **Cambio de archivo y registro, juntos.** La escritura del archivo y la inserción en `change_history` son un solo paso lógico. Si la escritura falla, no hay registro. Si el registro falla, se revierte la escritura. No queda el archivo cambiado sin rastro.
- **Lecturas en WAL.** Con el modo WAL, las consultas de la interfaz no bloquean a las escrituras de las sesiones de segundo plano. Varias sesiones pueden escribir a la vez.
- **Errores visibles.** Si una transacción falla, la etapa se marca con error. Nunca se pierde un cambio en silencio.

## 7. Relaciones entre operaciones y su orden

El orden importa cuando una operación prepara a la siguiente:

1. El nodo de contexto arma el contexto de la etapa y registra la auditoría.
2. El agente produce la respuesta, acumulando el razonamiento.
3. Si pide una herramienta de escritura, se crea la aprobación y la sesión queda esperando.
4. Al aprobar, se resuelve la aprobación y la sesión vuelve a trabajar.
5. `build` aplica el cambio: escribe el archivo y registra en `change_history`.
6. El motor marca la tarea como completada en su archivo de tarea.
7. La cola vuelve a derivarse y toma la siguiente tarea.

El paso 5 nunca ocurre sin el 3 y el 4. La base no lo impide, pero el módulo de herramientas sí. Ver [[database/02-rules/BUSINESS_RULES]].

## Referencias

- [[database/DATABASE]] — los dos planos y las políticas.
- [[database/01-schema/SCHEMA]] — estructura de las tablas implicadas.
- [[database/01-schema/RELATIONSHIPS]] — cascadas de borrado.
- [[database/01-schema/ENUMS]] — estados y transiciones.
- [[database/02-rules/BUSINESS_RULES]] — reglas que condicionan el flujo.
- [[specs/SPEC-NODO-CONTEXTO]] — de dónde sale el contexto.
- [[specs/SPEC-COLA-TAREAS]] — de dónde sale la cola.
