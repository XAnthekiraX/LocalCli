---
title: LocalCli — gestión del esquema
tags: [database, operaciones]
depende_de:
  - "[[database/01-schema/SCHEMA]]"
  - "[[database/DATABASE]]"
relacionado:
  - "[[database/01-schema/INDEXES]]"
  - "[[database/01-schema/ENUMS]]"
  - "[[database/02-rules/BUSINESS_RULES]]"
  - "[[database/02-rules/DATA_FLOW]]"
  - "[[database/03-operations/SEEDING]]"
  - "[[specs/SPEC-ARCHIVOS]]"
---
# MIGRATIONS — Gestión del esquema

## 1. Estrategia de migraciones

La versión del esquema se guarda en el `PRAGMA user_version` de SQLite, un entero que ya viene en la base y que no se usa para otra cosa. No hay tabla de migraciones.

La razón de no usar una tabla propia es que este proyecto no necesita el historial de qué migraciones se aplicaron: solo necesita saber en qué versión está el esquema y poder pasar al siguiente. `user_version` cubre exactamente eso sin añadir una tabla de control de migraciones al esquema.

Las migraciones se ejecutan automáticamente al abrir un proyecto, antes de usar la base. El usuario no ejecuta migraciones a mano.

Subir una migración aplicada no es lo mismo que tener un esquema válido: `user_version` registra qué se aplicó, no qué hay en las tablas. Por eso, además de aplicar lo que falte, al abrir un proyecto se verifica el esquema real contra la versión actual, y si no cuadra la apertura falla en lugar de declarar un esquema que no es. Ver
[[database/03-operations/MIGRATIONS#4-datos-existentes]] y la nota sobre el esquema actual, más abajo.

## 2. Modificación del esquema

- El esquema se define una vez, en la creación de la base. Si el archivo no existe, se crea con la versión actual.
- Cada cambio del esquema es una migración numerada, en orden ascendente.
- Al abrir un proyecto, la herramienta lee `user_version`, compara con la versión que espera y aplica todas las migraciones que falten, una a una y cada una en su propia transacción.
- Si una migración falla, se detiene ahí. Las anteriores que sí se aplicaron quedan; la que falló se revierte sola porque va en su propia transacción. Se avisa al usuario con el número de la migración que falló.
- Las migraciones se ejecutan una vez. `user_version` se incrementa a la versión destino tras cada una.

## 3. Convenciones de nombres

- Cada migración tiene un número y un nombre corto: `001-crear-schema`, `002-agregar-indice-approvals`, y así sucesivamente.
- El número es correlativo y de tres dígitos. El nombre describe qué hace, no dónde.
- El orden de los números es el orden de ejecución. No se reutiliza un número ni se renumera una migración ya publicada.
- La versión del esquema es el número de la última migración aplicada. `user_version = 5` significa que se aplicaron `001`, `002`, `003`, `004` y `005`.

## 4. Datos existentes

- **Una migración aditiva no toca los datos.** Añadir una tabla, una columna con valor por defecto, o un índice no requiere tocar las filas que ya existen. Una columna nueva sin default se añade nullable, y las filas viejas quedan con `NULL` hasta que se rellenen.
- **Una migración que cambia datos lleva su propia actualización.** Si un cambio de esquema obliga a transformar filas existentes, la migración lo hace en la misma transacción, después de cambiar el esquema y antes de subir `user_version`.
- **Los datos de `change_history` no se transforman ni se descartan.** Es el registro permanente de los cambios en los archivos del proyecto. Ninguna migración lo reescribe. Si un cambio de esquema lo afectara, la migración se rehace antes de aplicarse, no sobre datos ya guardados.

## 5. Migraciones destructivas

Una migración destructiva es la que borra o altera datos que no se pueden recuperar. Este esquema tiene pocos datos y casi todos son reconstruibles, pero uno no lo es: `change_history`.

Reglas:

- **Ninguna migración borra `change_history`.** Jamás. Es el único dato cuya pérdida no se puede recuperar, porque no se puede reconstruir qué se tocó en los archivos del proyecto.
- **Ninguna migración borra masivamente filas de `sessions`, `messages` o `reasoning`** para simplificar el esquema. Si una columna sobra, se deja o se ignora; no se tira lo que hay.
- **Una migración que destruya datos la escribe una persona, no el agente durante una fase de trabajo.** `build` no crea ni aplica migraciones destructivas por su cuenta. Es una operación que requiere una decisión explícita del usuario, porque afecta al historial.
- **Una migración destructiva se anuncia antes de aplicarse.** El usuario sabe qué se va a perder y lo aprueba, con la misma mecánica de aprobación que el resto de escrituras. Ver [[specs/SPEC-ARCHIVOS]].
- **Antes de una migración destructiva se recomienda una copia del archivo SQLite.** Es un archivo, se copia entero. La base es de un proyecto, no de un servidor, así que la copia es trivial.

## Nota sobre el esquema actual

La base se crea ya en su versión actual (v5): las seis tablas de [[database/01-schema/SCHEMA]] más `todos`, la lista de pasos de la sesión, `flow_context`, el bloque de contexto de un flujo, y `chat_evento`, las líneas de procesamiento del chat, además de las columnas `messages.duration_ms` y `chat_evento.duration_ms`. La creación del archivo, su esquema inicial y las migraciones `002-crear-todo`, `003-crear-flow-context`, `004-crear-chat-evento` y `005-duracion-linea-herramienta` ocurren en el mismo paso de apertura, así que `user_version` arranca en 5. Los índices de [[database/01-schema/INDEXES]] se aplican junto con la creación, no después.

La migración `002-crear-todo` es aditiva: añade la tabla `todos` y su clave primaria `(session_id, position)`, sin tocar ninguna fila existente. Una base en `user_version = 1` la recibe al abrirse y pasa a la 2 sin perder nada.

La migración `003-crear-flow-context` también es aditiva: añade la tabla `flow_context` y su índice único `(session_id, flow, stage)`, sin tocar ninguna fila existente. Una base en `user_version = 2` la recibe al abrirse y pasa a la 3 sin perder nada.

La migración `004-crear-chat-evento` también es aditiva: añade la tabla `chat_evento` con su índice `(session_id, created_at)` y la columna `messages.duration_ms` (nullable, sin valor por defecto: las filas anteriores quedan con `NULL`, que es "no se midió"). No toca ninguna fila existente. Una base en `user_version = 3` la recibe al abrirse y pasa a la 4 sin perder nada.

La migración `005-duracion-linea-herramienta` es aditiva y solo añade una columna: `chat_evento.duration_ms`, nullable y sin valor por defecto. No crea tabla, no crea índice y no actualiza filas. Las líneas de herramienta ya guardadas quedan con `NULL` y se pintan sin tiempo, que es exactamente lo que significa "no se midió": no se rellena un tiempo inventado para tapar el hueco. Una base en `user_version = 4` la recibe al abrirse y pasa a la 5 sin perder nada.

Añadir la columna como número y no solo dentro de `content` es deliberado: `content` ya guarda la línea compuesta, así que el tiempo se podría leer de ahí. Guardarlo también como columna es lo que permite ordenarlo o filtrar por él sin parsear texto de pantalla, que es la razón de ser de una columna y no un adorno del `content`.

### Un `user_version` correcto no basta para saber que el esquema es el de la versión

`user_version` dice qué migración se aplicó por última vez, no qué hay en las
tablas. Un archivo creado por una build anterior a la corrección del DDL de la
v1 tiene `user_version` correcto para su momento y, sin embargo:

- sus seis tablas declaran `id TEXT PRIMARY KEY` **sin `NOT NULL`**, así que
  admiten filas con `id` nulo, y
- no tienen el índice `idx_context_audit_unico`, así que admiten una segunda
  auditoría de la misma terna.

Subir el `user_version` no arregla eso: declararía un esquema que no es el que
tiene el archivo, y la próxima apertura volvería a mirar el `user_version` y a
creérselo. Es peor que no detectar nada.

Por eso, además de la versión, al abrir un proyecto se **verifica el esquema
real**: se comprueba que estén las tablas esperadas, que su `id` sea `NOT NULL` y
que existan la tabla `todos` y el índice único de auditoría. Si algo no cuadra,
la apertura falla con `E_DB_SCHEMA_OUTDATED` diciendo qué archivo borrar. No se
corrige solo y no se borra solo: el archivo es del usuario, y la política de no
tocar datos es de [[database/DATABASE]].

La corrección del DDL de la v1 no se aplica como migración; el camino previsto es
borrar `.localcli/state.db` y dejar que se vuelva a crear. La base es caché y
traza local: el contenido que no se puede recuperar de ella está en git. Si
alguna vez esto ya no fuera cierto, la corrección pasaría a ser una migración con
su transformación de datos, y esta verificación seguiría siendo necesaria para
detectar bases ajenas.

La comparación contra el esquema real es también un test: fija el número de
versión esperado contra el literal que dice esta sección, para que subir la
constante sin actualizar el documento salga en rojo.

## Nota sobre el canal de herramientas

El canal nativo de herramientas **no añade filas a `messages`** y `messages.role`
sigue cerrado a `user` y `agent`. Eso no cambia con la 004: un turno con
herramientas no se guarda como conversación.

Lo que sí añadió la `004-crear-chat-evento` es una tabla para las **líneas de
pantalla** de ese canal: el sub-proceso de cada etapa (`chat_evento.tipo =
proceso`) y la línea de cada herramienta (`tipo = herramienta`). Son lo que el
chat muestra y lo que se recupera al volver a una sesión, pero **no son
contexto**: al modelo solo se le entregan los turnos de `messages`.

Se eligió una tabla nueva en vez de crecer `messages.role` por dos razones:

- La separación «se muestra / se envía al modelo» queda en la frontera de la
  tabla, no en un filtro que alguien pueda olvidar: `session.historialPara` arma
  el contexto solo de `messages`. Con un rol nuevo en `messages`, cada lectura
  del contexto tendría que acordarse de excluirlo.
- La migración es **aditiva**. Cambiar el `CHECK` de `role` en SQLite obliga a
  reconstruir `messages`, y `reasoning` tiene una clave foránea hacia ella
  (`ON DELETE CASCADE`): una reconstrucción mal hecha arrastraría el
  razonamiento guardado. Añadir una tabla no toca ninguna fila existente
  (MIGRATIONS.md §4 y §5).

El rastro permanente de las ejecuciones sigue estando, además, en los eventos de
la capa universal y en `change_history`. Ver [[database/02-rules/DATA_FLOW]],
[[database/01-schema/ENUMS]] y [[backend/DECISIONS]].

## Referencias

- [[database/01-schema/SCHEMA]] — el esquema que migran.
- [[database/01-schema/INDEXES]] — índices que crean las migraciones.
- [[database/01-schema/ENUMS]] — por qué `messages.role` no crece.
- [[database/02-rules/DATA_FLOW]] — qué se guarda de un turno y qué no.
- [[database/DATABASE]] — políticas, en particular la de no borrar el historial.
- [[database/02-rules/BUSINESS_RULES]] — por qué el historial no se toca.
- [[database/03-operations/SEEDING]] — qué se inserta tras crear el esquema.
- [[specs/SPEC-ARCHIVOS]] — la aprobación que exige una migración destructiva.
