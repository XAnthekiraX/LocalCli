---
title: LocalCli — capa de datos
tags: [database, indice]
depende_de:
  - "[[PROJECT]]"
relacionado:
  - "[[backend/BACKEND]]"
  - "[[specs/SPEC-ARCHIVOS]]"
---

# DATABASE — LocalCli

## Propósito

Centraliza toda la información de la base de datos de LocalCli. Es el mapa de navegación que leen los agentes: al implementar un endpoint, el agente recorre solo la subcadena que necesita en lugar de cargar toda la capa.

## Almacenamiento: un archivo global

- Un proyecto es la carpeta abierta.
- **Todas las carpetas comparten un único archivo SQLite global**, en `~/.config/localcli/state.db`, junto al resto de la configuración del harness. No hay una base por proyecto.
- El aislamiento entre proyectos es **lógico**: cada carpeta es una fila de `projects` (`path` único) y cada sesión cuelga de su proyecto por `project_id`. Una consulta sin el filtro de proyecto es una fuga entre carpetas, así que `project_id` es obligatorio en `sessions`.
- El modo WAL está activo, para que la interfaz pueda leer mientras las sesiones de segundo plano escriben.
- No hay servidor de base de datos. El driver es puro Go, sin cgo, así que el binario sigue siendo portable.

## Los dos planos

**Archivos: fuente de verdad.** La documentación del proyecto y los archivos de tarea se editan a mano, se versionan en git y se leen sin la herramienta. Si los dos planos divergen, **manda el archivo**.

**SQLite: estado de ejecución.** Solo lo que es efímero o lo que necesita consulta rápida.

**Lo derivado: sin índice persistente.** El grafo de dependencias entre documentos y el orden de la cola se reconstruyen leyendo los archivos al arrancar. No hay un índice en SQLite que pueda quedar desincronizado del archivo que lo originó.

## Inventario de entidades

### En SQLite

| Entidad | Responsabilidad |
|---|---|
| `projects` | Un proyecto: la carpeta abierta, con su `path` único, raíz lógica de las sesiones |
| `sessions` | Una sesión de trabajo: nombre, proyecto, capa y estado |
| `messages` | Mensajes de la conversación, de usuario y de agente |
| `reasoning` | Razonamiento del modelo, ligado a su mensaje, acumulado del streaming |
| `approvals` | Aprobaciones pendientes y resueltas, ligadas a su sesión |
| `context_audit` | Qué documentación entró y salió del contexto de cada etapa, y por qué |
| `change_history` | Qué archivo cambió, qué había antes y qué quedó |

El detalle de columnas de cada una está en [[database/01-schema/TABLES]].

### En archivos (fuente de verdad)

| Entidad | Dónde vive | Qué define |
|---|---|---|
| Documentación del proyecto | `ai/docs/` y la documentación de cada capa | Frontmatter con sus dependencias mediante enlaces, que es la base del grafo de contexto |
| Archivos de tarea | `ai/tasks/<capa>/NNN-task-<nombre>.md` | Frontmatter con id, capa, acción, dependencias y estado, más el bloque de contexto, que es la base de la cola |

Los detalles de ambas convenciones están en [[database/02-rules/DATA_FLOW]].

## Políticas que no se negocian

- **Nada destructivo ni de fuera se aplica sin aprobación**, sin excepción, ni dentro de un flujo en curso. Crear, escribir entero un archivo vacío y editar dentro de la carpeta no interrumpen, pero quedan en `change_history`. Ver [[specs/SPEC-ARCHIVOS]].
- **`change_history` nunca se borra.** Es lo que permite revertir y auditar qué se tocó en el proyecto.
- **El borrado de una sesión es definitivo.** Cuando borras una sesión, sus mensajes y su razonamiento se van con ella. Esto es deliberado y es distinto de `change_history`: la conversación es desechable, el registro de cambios en tus archivos es permanente. Si necesitas la trazabilidad de una sesión, no la has borrado.
- **El proyecto no se borra al cerrar la carpeta.** Cerrar una carpeta o abrir otra no toca la base: la fila de `projects` y sus sesiones siguen ahí, y volver a abrir la carpeta las recupera. Borrar un proyecto es una acción explícita y arrastra sus sesiones en cascada.

## Navegación

La cadena que sigue un agente cuando implementa un endpoint:

```
DATABASE.md → [[database/01-schema/TABLES]] → [[database/01-schema/RELATIONSHIPS]]
            → [[database/02-rules/BUSINESS_RULES]] → [[database/03-operations/QUERIES]]
```

Cada documento de la capa:

- Esquema: [[database/01-schema/SCHEMA]], [[database/01-schema/TABLES]], [[database/01-schema/RELATIONSHIPS]], [[database/01-schema/ENUMS]], [[database/01-schema/CONSTRAINTS]], [[database/01-schema/INDEXES]]
- Reglas: [[database/02-rules/BUSINESS_RULES]], [[database/02-rules/DATA_FLOW]]
- Operaciones: [[database/03-operations/QUERIES]], [[database/03-operations/MIGRATIONS]], [[database/03-operations/SEEDING]]

## Referencias

- [[PROJECT]] — decisiones de persistencia ya aprobadas.
- [[IDEA]] — visión del proyecto.
- [[specs/SPEC-ARCHIVOS]] — reglas de acceso y permiso que se aplican aquí.
- [[specs/SPEC-SESIONES]] — de dónde salen las sesiones.
- [[specs/SPEC-NODO-CONTEXTO]] — qué origina el grafo de documentos.
- [[specs/SPEC-COLA-TAREAS]] — qué origina la cola.
