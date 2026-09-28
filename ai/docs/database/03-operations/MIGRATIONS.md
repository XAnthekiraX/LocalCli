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
- La versión del esquema es el número de la última migración aplicada. `user_version = 2` significa que se aplicaron `001` y `002`.

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

La base se crea ya en su versión actual (v2): las seis tablas de [[database/01-schema/SCHEMA]] más `todos`, la lista de pasos de la sesión. La creación del archivo, su esquema inicial y la migración `002-crear-todo` ocurren en el mismo paso de apertura, así que `user_version` arranca en 2. Los índices de [[database/01-schema/INDEXES]] se aplican junto con la creación, no después.

La migración `002-crear-todo` es aditiva: añade la tabla `todos` y su clave primaria `(session_id, position)`, sin tocar ninguna fila existente. Una base en `user_version = 1` la recibe al abrirse y pasa a la 2 sin perder nada.

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

La incorporación del canal nativo de herramientas **no genera migración**, y esa
ausencia es una decisión documentada, no un descuido.

Lo que añadiría el canal son mensajes con rol de herramienta en la conversación.
Esos mensajes viven solo durante el turno: no se insertan, no hay tabla para
ellos y `messages.role` sigue cerrado a `user` y `agent`.

Las razones para no persistirlos:

- El `CHECK` de `role` no cambia, así que no hay DDL que migrar.
- Al retomar una sesión, lo que el usuario necesita es qué se hizo y qué quedó,
  y eso ya está en el texto del mensaje final del agente. Reinyectar peticiones
  de herramientas de hace tres sesiones no aporta nada y multiplica el contexto.
- Una migración que cambiara `role` y añadiera columnas de argumentos y
  resultados obligaría además a decidir qué se reconstruye y qué se descarta al
  cargar una sesión, que es una política, no un problema de esquema.

El rastro de las ejecuciones existe, pero en otro sitio: los eventos de la capa
universal y `change_history`, que son efímeros y permanentes respectivamente. Ver
[[database/02-rules/DATA_FLOW]] y [[database/01-schema/ENUMS]].

Si en el futuro se quisiera persistir los turnos con herramientas, sería una migración completa: `CHECK` de `role`, columnas nuevas y la política de reconstrucción. No se hace ahora.

## Referencias

- [[database/01-schema/SCHEMA]] — el esquema que migran.
- [[database/01-schema/INDEXES]] — índices que crean las migraciones.
- [[database/01-schema/ENUMS]] — por qué `messages.role` no crece.
- [[database/02-rules/DATA_FLOW]] — qué se guarda de un turno y qué no.
- [[database/DATABASE]] — políticas, en particular la de no borrar el historial.
- [[database/02-rules/BUSINESS_RULES]] — por qué el historial no se toca.
- [[database/03-operations/SEEDING]] — qué se inserta tras crear el esquema.
- [[specs/SPEC-ARCHIVOS]] — la aprobación que exige una migración destructiva.
