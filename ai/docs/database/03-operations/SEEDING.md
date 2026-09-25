# SEEDING — Datos iniciales

## 1. Datos iniciales

LocalCli **no necesita seeds**. La base de datos nace vacía y eso es correcto.

La razón es que todo lo que la herramienta necesita para funcionar ya existe antes de abrir la base:

- **El modelo** está en Ollama, no en SQLite. Ver [[specs/SPEC-OLLAMA-PERFIL]].
- **La documentación del proyecto** son archivos, no filas. Ver [[database/02-rules/DATA_FLOW]].
- **Las tareas** son archivos, no filas.
- **El esquema** se crea vacío, con `user_version = 1`. Ver [[database/03-operations/MIGRATIONS]].

No hay catálogos, ni tablas de configuración, ni tipos predefinidos que insertar. El usuario abre una carpeta, y la base asociada está lista.

## 2. Seeds necesarios

Ninguno. No hay ningún seed que la aplicación aplique al crear el esquema.

Lo más cercano a un seed es el valor por defecto de `sessions.status`, que es `inactiva`, y el valor por defecto de `approvals.status`, que es `pendiente`. Son valores por defecto de columna, no filas insertadas. Ver [[database/01-schema/TABLES]].

## 3. Datos de desarrollo y testing

Los datos de prueba no se siembran en la base de un proyecto real. Para pruebas se crea una base temporal, aislada, que se destruye al terminar.

| Propósito | Cómo |
|---|---|
| Probar el esquema y las restricciones | Base en memoria, creada desde el esquema, se descarta al terminar |
| Probar una etapa de un flujo | Base en memoria con datos mínimos: una sesión, un mensaje |
| Probar el aislamiento entre proyectos | Dos bases en memoria distintas, se comprueba que no se ven |
| Probar la supervivencia del historial | Base en memoria: se crea una sesión, se le aplica un cambio, se borra la sesión, y se comprueba que `change_history` sigue ahí |

**No se siembran sesiones ni cambios de ejemplo en la base de un proyecto.** Si se hiciera, un proyecto real empezaría con conversaciones y cambios que el usuario nunca hizo, y `change_history` dejaría de ser un registro fiable. Ver [[database/02-rules/BUSINESS_RULES]].

Si en el futuro hiciera falta un conjunto de datos de demostración para capturas o para probar la interfaz, se generaría con datos falsos y explícitamente etiquetados, nunca mezclado con datos de un proyecto real.

## 4. Dependencias entre seeds

No aplican. No hay seeds, así que no hay orden entre ellos.

El orden que sí existe es el de arranque, que no es de seeds sino de esquema y permisos:

1. Se comprueba que la carpeta abierta es un proyecto.
2. Se crea la base si no existe, con el esquema de [[database/01-schema/SCHEMA]] y `user_version = 1`.
3. Se aplica `PRAGMA foreign_keys = ON` y el modo WAL.
4. Se aplica las migraciones pendientes. Ver [[database/03-operations/MIGRATIONS]].
5. Se reconstruyen el grafo de documentos y la cola leyendo los archivos. Ver [[database/02-rules/DATA_FLOW]].

Ese orden importa: los índices y las claves foráneas se crean con el esquema, antes de que entre ningún dato. La base nunca se usa sin su estructura completa.

## Referencias

- [[database/01-schema/SCHEMA]] — el esquema que se crea vacío.
- [[database/01-schema/TABLES]] — valores por defecto de columna, que es lo único parecido a un seed.
- [[database/DATABASE]] — por qué los archivos son la fuente de verdad y no filas.
- [[database/03-operations/MIGRATIONS]] — el orden de arranque del esquema.
- [[specs/SPEC-OLLAMA-PERFIL]] — el modelo no está en la base.
- [[specs/SPEC-NODO-CONTEXTO]] — el contexto sale de los archivos, no de una consulta.
