# RELATIONSHIPS — Relaciones entre tablas

## 1. Relaciones

| Origen | Destino | Cardinalidad | Descripción |
|---|---|---|---|
| `sessions` | `messages` | 1:N | Una sesión tiene muchos mensajes |
| `messages` | `reasoning` | 1:1 opcional | Un mensaje de agente puede tener un razonamiento |
| `sessions` | `approvals` | 1:N | Una sesión puede tener muchas aprobaciones |
| `sessions` | `context_audit` | 1:N | Una sesión genera muchos registros de auditoría |
| `sessions` | `change_history` | 1:N opcional | Una sesión aplica cambios; el registro sobrevive a ella |

No hay relaciones de N:N. La base de datos es un árbol que cuelga de `sessions`, sin tablas puente.

Hay dos entidades que **no** están en SQLite: la documentación del proyecto y los archivos de tarea. Son la fuente de verdad en archivos, y el grafo de documentos y el orden de la cola se derivan de ellos al arrancar, sin tabla que los modele. Ver [[database/02-rules/DATA_FLOW]].

## 2. Foreign Keys

| Tabla | Columna | Referencia | On delete | On update |
|---|---|---|---|---|
| `messages` | `session_id` | `sessions.id` | `CASCADE` | `CASCADE` |
| `reasoning` | `message_id` | `messages.id` | `CASCADE` | `CASCADE` |
| `approvals` | `session_id` | `sessions.id` | `CASCADE` | `CASCADE` |
| `context_audit` | `session_id` | `sessions.id` | `CASCADE` | `CASCADE` |
| `change_history` | `session_id` | `sessions.id` | `SET NULL` | `CASCADE` |

Todas las claves foráneas son `NOT NULL` salvo la de `change_history`, que es nullable a propósito. Las cascadas en `update` son irrelevantes en la práctica porque los UUID no se reescriben nunca, pero se declaran por completitud.

## 3. Cascadas

La diferencia entre `CASCADE` y `SET NULL` en `change_history` es la que sostiene una política del proyecto: **el historial de cambios de los archivos nunca se borra**.

- **Al eliminar una sesión**, `messages`, `approvals` y `context_audit` se borran en cascada con ella, porque son parte de la conversación y del estado de ejecución. Lo mismo ocurre con `reasoning`, que cuelga de `messages`.
- **`change_history` sobrevive.** Su `session_id` pasa a `NULL` con `SET NULL`. El registro de qué se tocó en los archivos del proyecto se conserva aunque la sesión desaparezca, tal como exige [[specs/SPEC-ARCHIVOS]].

Esto no contradice que el borrado de sesiones sea definitivo. Son dos planos distintos:

| Tabla | Al borrar la sesión | Por qué |
|---|---|---|
| `messages`, `reasoning`, `approvals`, `context_audit` | Desaparecen | Son la conversación y el estado de ejecución, desechables |
| `change_history` | Sobrevive con `session_id` en `NULL` | Es el registro permanente de los cambios en los archivos del proyecto |

Cuando una fila queda con `session_id` en `NULL`, `session_id` deja de ser una relación utilizable: ya no apunta a ninguna sesión. Los índices sobre esa columna lo tratan como `NULL` y no lo incluyen, lo que es el comportamiento deseado.

`reasoning` no tiene `session_id` propio: llega a la sesión a través de `messages`, y por eso se borra en la misma cascada.

## 4. Dependencias

Orden de borrado natural, de arriba abajo. Ninguna tabla depende de una que se borre después que ella.

```
sessions
  ├─ messages ── reasoning
  ├─ approvals
  ├─ context_audit
  └─ change_history   (sobrevive: SET NULL)
```

Dependencias entre entidades:

- `reasoning` depende de `messages`. No puede existir un razonamiento sin el mensaje al que pertenece.
- `messages`, `approvals` y `context_audit` dependen de `sessions`. Una sesión borrada se lleva a sus hijos en cascada.
- `change_history` depende de `sessions` solo de forma opcional y la relación se rompe al borrar, en vez de propagarse.

Ninguna tabla referencia a `change_history` ni depende de ella. Es una hoja del esquema: solo se escribe y se lee, nunca se navega desde otra tabla hacia algo que la referencie.

Los índices que soportan estas relaciones están en [[database/01-schema/INDEXES]].

## Referencias

- [[database/01-schema/SCHEMA]] — estructura de las tablas.
- [[database/01-schema/TABLES]] — columnas y claves.
- [[database/01-schema/ENUMS]] — estados de sesión y aprobación.
- [[database/01-schema/CONSTRAINTS]] — las claves foráneas como restricción.
- [[database/DATABASE]] — políticas que sostienen estas cascadas.
