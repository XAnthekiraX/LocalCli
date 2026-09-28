---
title: LocalCli — circulación de datos
tags: [database, reglas]
depende_de:
  - "[[database/DATABASE]]"
  - "[[database/01-schema/SCHEMA]]"
  - "[[database/02-rules/BUSINESS_RULES]]"
relacionado:
  - "[[database/01-schema/ENUMS]]"
  - "[[database/01-schema/RELATIONSHIPS]]"
  - "[[specs/SPEC-COLA-TAREAS]]"
  - "[[specs/SPEC-NODO-CONTEXTO]]"
  - "[[specs/SPEC-AGENTE-BASE]]"
  - "[[specs/SPEC-TOOLS]]"
  - "[[backend/04-infrastructure/EVENTS]]"
---
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

### Lo que un turno no deja escrito

Un turno con el canal nativo de herramientas produce, por dentro, una secuencia de mensajes: la petición del modelo, la petición de una herramienta, su resultado, la siguiente petición, y así hasta la respuesta final. **De esa secuencia, en la base solo queda el primer mensaje y el último.**

| Mensaje | ¿Se guarda? |
|---|---|
| Lo que escribió el usuario | Sí, en `messages` con `role = user` |
| La petición de una herramienta y su resultado | No |
| La respuesta final del agente | Sí, en `messages` con `role = agent` |
| El razonamiento del turno | Sí, en `reasoning` |

La razón es que el detalle de las ejecuciones pertenece al momento en que ocurren. Al retomar una sesión, lo que interesa es qué se hizo y qué quedó, y eso ya está en el texto del agente: en su respuesta final dice qué archivos leyó, qué comandos corrió y qué cambió.

Lo que se pierde es la posibilidad de reconstruir una ejecución concreta —qué argumento exacto se pasó a qué herramienta—. Para eso están los eventos de la capa universal y, cuando aplica, `change_history`, que sí son permanentes. Ver [[backend/04-infrastructure/EVENTS]] y [[backend/DECISIONS]].

## 3. Creación

**Crear un mensaje.** El usuario escribe, la TUI lo entrega a la sesión y se inserta una fila en `messages` con `role = user`. Si la respuesta del modelo trae razonamiento, se inserta la fila correspondiente en `reasoning` y se va acumulando mientras llega. Al terminar, se inserta el mensaje del agente. Las dos inserciones van en la misma transacción que la actualización de `status` de la sesión.

Un turno con herramientas tiene más pasos por dentro, pero **ninguno escribe en la base**. El bucle pide al modelo, ejecuta lo que le pidió y vuelve a pedir, todo en memoria. Solo el primer y el último mensaje se insertan. Ver [[database/01-schema/ENUMS]].

**Crear una aprobación.** Cuando el agente pide permiso, se inserta una fila en `approvals` con `status = pendiente` y `resolved_at` en `NULL`, y la sesión pasa a `esperando_permiso`. Es un flujo que puede quedar a medias, así que ambas escrituras van en la misma transacción.

Un turno puede necesitar varias aprobaciones, y se resuelven **una a una**: las ejecuciones son secuenciales, así que la segunda herramienta no se intenta hasta que la primera está resuelta. Cada aprobación es su propia transacción, y la sesión vuelve a `trabajando` entre una y otra.

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
2. El agente pide una respuesta al modelo, acumulando el razonamiento.
3. Si el modelo pide una herramienta, la capa de ejecución comprueba el permiso y, si hace falta, crea la aprobación. La sesión queda esperando y la ejecución no ha empezado: **pedir permiso y ejecutar son dos momentos distintos**.
4. Al aprobar, se resuelve la aprobación y **entonces** se ejecuta el handler, que aplica el cambio si lo tenía que aplicar.
5. El agente vuelve a pedir al modelo con el resultado, y repite desde el 3 mientras siga pidiendo herramientas, hasta un máximo de rondas.
6. Cuando el modelo responde en texto, se inserta el mensaje del agente y se cierra el turno.
7. Si el cambio fue una escritura, `build` ya lo aplicó en el 4 y quedó registrado en `change_history`.
8. El motor marca la tarea como completada en su archivo de tarea.
9. La cola vuelve a derivarse y toma la siguiente tarea.

El paso 7 nunca ocurre sin el 3 y el 4. La base no lo impide, pero la capa universal de herramientas sí: el permiso se comprueba antes de ejecutar nada. Ver [[database/02-rules/BUSINESS_RULES]] y [[backend/03-security/SECURITY]].

Los pasos 3 y 5 pueden repetirse hasta un máximo de rondas. Al agotarlo, el turno se cierra con lo conseguido: es un límite duro, no un detalle, porque un modelo puede pedir herramientas en bucle. Ver [[specs/SPEC-AGENTE-BASE]].

**Lo que se escribe en la base en este ciclo es solo:** la auditoría de contexto, el mensaje del usuario, las aprobaciones que se abren y se resuelven, y el mensaje final del agente. El bucle de herramientas en sí no escribe nada.

## Referencias

- [[database/DATABASE]] — los dos planos y las políticas.
- [[database/01-schema/SCHEMA]] — estructura de las tablas implicadas.
- [[database/01-schema/RELATIONSHIPS]] — cascadas de borrado.
- [[database/01-schema/ENUMS]] — estados y transiciones.
- [[database/02-rules/BUSINESS_RULES]] — reglas que condicionan el flujo.
- [[specs/SPEC-NODO-CONTEXTO]] — de dónde sale el contexto.
- [[specs/SPEC-COLA-TAREAS]] — de dónde sale la cola.
- [[specs/SPEC-AGENTE-BASE]] — el ciclo de un turno y su máximo de rondas.
- [[specs/SPEC-TOOLS]] — las herramientas del usuario y su ejecutor.
- [[backend/04-infrastructure/EVENTS]] — los eventos que sí dejan rastro de cada ejecución.
