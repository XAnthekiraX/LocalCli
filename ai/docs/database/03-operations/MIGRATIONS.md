---
title: LocalCli — gestión del esquema
tags: [database, operaciones]
depende_de:
  - "[[database/01-schema/SCHEMA]]"
  - "[[database/DATABASE]]"
relacionado:
  - "[[database/01-schema/INDEXES]]"
  - "[[database/02-rules/BUSINESS_RULES]]"
  - "[[database/03-operations/SEEDING]]"
  - "[[specs/SPEC-ARCHIVOS]]"
---
# MIGRATIONS — Gestión del esquema

## 1. Estrategia de migraciones

La versión del esquema se guarda en el `PRAGMA user_version` de SQLite, un entero que ya viene en la base y que no se usa para otra cosa. No hay tabla de migraciones.

La razón de no usar una tabla propia es que este proyecto no necesita el historial de qué migraciones se aplicaron: solo necesita saber en qué versión está el esquema y poder pasar al siguiente. `user_version` cubre exactamente eso sin añadir una séptima tabla a un esquema de seis.

Las migraciones se ejecutan automáticamente al abrir un proyecto, antes de usar la base. El usuario no ejecuta migraciones a mano.

Subir una migración aplicada no es lo mismo que tener un esquema válido: `user_version` registra qué se aplicó, no qué hay en las tablas. Por eso, además de aplicar lo que falte, al abrir un proyecto se verifica el esquema real contra la versión actual, y si no cuadra la apertura falla en lugar de declarar un esquema que no es. Ver
[[database/03-operations/MIGRATIONS#3-datos-existentes]] y la nota sobre el esquema actual, más abajo.

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

La base se crea ya en su versión 1, con las seis tablas de [[database/01-schema/SCHEMA]]. No hay migración de creación pendiente: la creación del archivo y su esquema inicial ocurren en el mismo paso, y `user_version` arranca en 1. Las migraciones de [[database/01-schema/INDEXES]] existen, pero se aplican junto con la creación inicial, no después.

### Un `user_version` a 1 no basta para saber que el esquema es el de la v1

`user_version` dice qué migración se aplicó por última vez, no qué hay en las
tablas. Un archivo creado por una build anterior a la corrección del DDL de la
v1 tiene `user_version = 1` y, sin embargo:

- sus seis tablas declaran `id TEXT PRIMARY KEY` **sin `NOT NULL`**, así que
  admiten filas con `id` nulo, y
- no tienen el índice `idx_context_audit_unico`, así que admiten una segunda
  auditoría de la misma terna.

Aplicar una migración `002` no arregla eso: subir el `user_version` a 2 sobre un
esquema que nunca tuvo la corrección declararía un esquema que no es el de la v2,
y la próxima apertura volvería a mirar el `user_version` y a creérselo. Es peor
que no detectar nada.

Por eso, además de la versión, al abrir un proyecto se **verifica el esquema
real**: se comprueba que estén las seis tablas y que su `id` sea `NOT NULL`. Si
algo no cuadra, la apertura falla con `E_DB_SCHEMA_OUTDATED` diciendo qué
archivo borrar. No se corrige solo y no se borra solo: el archivo es del
usuario, y la política de no tocar datos es de [[database/DATABASE]].

Como la v1 es la versión inicial y todavía no hay una v2, el camino previsto es
borrar `.localcli/state.db` y dejar que se vuelva a crear. La base es caché y
traza local: el contenido que no se puede recuperar de ella está en git. Si
alguna vez esto ya no fuera cierto, la corrección pasaría a ser la migración
`002` con su transformación de datos, y esta verificación seguiría siendo
necesaria para detectar bases ajenas.

La comparación contra el esquema real es también un test: fija el número de
versión esperado contra el literal que dice esta sección, para que subir la
constante sin actualizar el documento salga en rojo.

## Referencias

- [[database/01-schema/SCHEMA]] — el esquema que migran.
- [[database/01-schema/INDEXES]] — índices que crean las migraciones.
- [[database/DATABASE]] — políticas, en particular la de no borrar el historial.
- [[database/02-rules/BUSINESS_RULES]] — por qué el historial no se toca.
- [[database/03-operations/SEEDING]] — qué se inserta tras crear el esquema.
- [[specs/SPEC-ARCHIVOS]] — la aprobación que exige una migración destructiva.
