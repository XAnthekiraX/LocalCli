---
title: LocalCli — restricciones de la base de datos
tags: [database, esquema]
depende_de:
  - "[[database/01-schema/SCHEMA]]"
  - "[[database/01-schema/TABLES]]"
relacionado:
  - "[[database/01-schema/ENUMS]]"
  - "[[database/01-schema/RELATIONSHIPS]]"
  - "[[database/01-schema/INDEXES]]"
  - "[[database/02-rules/BUSINESS_RULES]]"
  - "[[specs/SPEC-ARCHIVOS]]"
  - "[[specs/SPEC-NODO-CONTEXTO]]"
---
# CONSTRAINTS — Restricciones de la base de datos

## 1. Constraints

### NOT NULL

| Tabla | Columnas |
|---|---|
| `sessions` | `id`, `name`, `status`, `created_at`, `updated_at` |
| `messages` | `id`, `session_id`, `role`, `content`, `created_at` |
| `reasoning` | `id`, `message_id`, `content`, `created_at` |
| `approvals` | `id`, `session_id`, `description`, `status`, `created_at` |
| `context_audit` | `id`, `session_id`, `stage`, `document`, `decision`, `created_at` |
| `change_history` | `id`, `operation`, `file_path`, `created_at` |

Las columnas que sí admiten `NULL` están justificadas en [[database/01-schema/TABLES]]: `sessions.layer` para una sesión general, `messages.input_tokens` y `output_tokens` cuando el modelo no los reporta, `reasoning` no tiene ninguno opcional, `approvals.resolved_at` mientras está pendiente, `context_audit.reason` y `tokens`, y las dos columnas de contenido de `change_history` más su `session_id`.

### UNIQUE

| Restricción | Propósito |
|---|---|
| `sessions.id` (PK) | Identificador único de sesión |
| `messages.id` (PK) | Identificador único de mensaje |
| `reasoning.id` (PK) | Identificador único de razonamiento |
| `approvals.id` (PK) | Identificador único de aprobación |
| `context_audit.id` (PK) | Identificador único de registro de auditoría |
| `change_history.id` (PK) | Identificador único de cambio |
| `messages (session_id, id)` | Índice único que garantiza un solo `reasoning` por mensaje |

La última es la que hace efectiva la relación 1:1 opcional entre `messages` y `reasoning`: aunque el modelo no lo imponga en la base, un mensaje no puede tener dos razonamientos.

`PRAGMA foreign_keys = ON` es obligatorio en cada conexión. Sin este pragma, SQLite ignora las claves foráneas y las cascadas no se aplican. Es un detalle fácil de olvidar que rompería toda la política de borrado.

### CHECK

| Tabla | Columna | Valores permitidos |
|---|---|---|
| `sessions` | `status` | `inactiva`, `trabajando`, `esperando_permiso`, `terminada`, `error` |
| `messages` | `role` | `user`, `agent` |
| `messages` | `input_tokens`, `output_tokens` | `>= 0` o `NULL` |
| `approvals` | `status` | `pendiente`, `aprobada`, `declinada`, `obsoleta` |
| `context_audit` | `decision` | `incluido`, `descartado` |
| `context_audit` | `tokens` | `>= 0` o `NULL` |
| `change_history` | `operation` | Los seis valores de [[database/01-schema/ENUMS]] |

Los valores permitidos y su significado están en [[database/01-schema/ENUMS]].

### Foreign keys

Están definidas en [[database/01-schema/RELATIONSHIPS]]. Todas con `ON UPDATE CASCADE` y con `ON DELETE CASCADE` salvo `change_history.session_id`, que usa `SET NULL` para que el historial sobreviva a la sesión.

## 2. Restricciones especiales

- **`resolved_at` consistente con el estado.** Una aprobación `pendiente` tiene `resolved_at` en `NULL`, y cualquier otro estado lo tiene relleno. Se comprueba con un `CHECK` que cruza las dos columnas.
- **`reason` obligatorio al descartar.** En `context_audit`, si `decision = 'descartado'` entonces `reason` no es `NULL`; si `decision = 'incluido'` entonces `reason` sí es `NULL`. El nodo de contexto exige que todo descarte esté justificado, y la base lo hace cumplir. Ver [[specs/SPEC-NODO-CONTEXTO]].
- **`before_content` y `after_content` según la operación.** En `change_history`, `crear_archivo` y `crear_carpeta` tienen `before_content` en `NULL`; `eliminar_archivo` y `eliminar_carpeta` tienen `after_content` en `NULL`; `escribir_archivo` y `editar_archivo` tienen ambos relleno. Se comprueba con `CHECK` por operación.
- **Transiciones de estado.** La validación de que una transición de sesión o de aprobación sea legal no se comprueba con `CHECK`, porque `CHECK` solo mira la fila nueva. La legibilidad de la transición se aplica en el código, en el módulo que escribe. Ver [[database/02-rules/BUSINESS_RULES]].

## 3. Reglas que la base impone

Lo que la base rechaza por sí sola, sin ayuda del código:

- Un mensaje, una aprobación o un registro de auditoría sin sesión: la clave foránea lo rechaza.
- Un rol, un estado o una operación fuera del catálogo: el `CHECK` lo rechaza.
- Un conteo negativo de tokens: el `CHECK` lo rechaza.
- Un descarte sin motivo: el `CHECK` lo rechaza.
- Una aprobación resuelta sin `resolved_at`, o pendiente con `resolved_at` relleno: el `CHECK` lo rechaza.
- Un `reasoning` de un mensaje que no es de agente: lo rechaza una clave foránea compuesta entre `reasoning` y `messages`, que exige que el mensaje referenciado tenga `role = agent`.
- Un segundo `reasoning` para el mismo mensaje: el `UNIQUE` compuesto lo rechaza.
- Una sesión con un hijo `reasoning` colgando de un mensaje suyo cuando se borra: lo resuelve la cascada, no lo rechaza.

Lo que la base **no** impone y depende del código: la legalidad de las transiciones de estado, que un cambio de archivo pasara por aprobación antes de llegar aquí, y que `reasoning` solo exista si el modelo devolvió razonamiento. La aprobación es una política de [[specs/SPEC-ARCHIVOS]] y vive en el módulo de herramientas, no en un `CHECK`.

## Referencias

- [[database/01-schema/SCHEMA]] — estructura.
- [[database/01-schema/TABLES]] — columnas y nulabilidad.
- [[database/01-schema/RELATIONSHIPS]] — claves foráneas y cascadas.
- [[database/01-schema/ENUMS]] — valores permitidos.
- [[database/01-schema/INDEXES]] — unicidad ligada a índices.
- [[database/02-rules/BUSINESS_RULES]] — reglas que la base no puede imponer.
