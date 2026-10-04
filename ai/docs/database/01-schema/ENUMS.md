---
title: LocalCli — valores cerrados de la base de datos
tags: [database, esquema]
depende_de:
  - "[[database/01-schema/SCHEMA]]"
  - "[[database/01-schema/TABLES]]"
relacionado:
  - "[[database/01-schema/RELATIONSHIPS]]"
  - "[[database/01-schema/CONSTRAINTS]]"
  - "[[specs/SPEC-SESIONES]]"
  - "[[specs/SPEC-TOOLS]]"
  - "[[specs/SPEC-INTERFAZ-ATAJOS]]"
---
# ENUMS — Valores cerrados de la base de datos

## 1. Enums

La base de datos tiene cinco columnas con un conjunto cerrado de valores. Cada una se valida con `CHECK`, no solo por convención. Ver [[database/01-schema/CONSTRAINTS]].

| Enum | Columna | Valores |
|---|---|---|
| Estado de sesión | `sessions.status` | `inactiva`, `trabajando`, `esperando_permiso`, `terminada`, `error` |
| Rol de mensaje | `messages.role` | `user`, `agent` |
| Tipo de línea de chat | `chat_evento.tipo` | `proceso`, `herramienta` |
| Estado de aprobación | `approvals.status` | `pendiente`, `aprobada`, `declinada`, `obsoleta` |
| Operación de archivo | `change_history.operation` | `crear_archivo`, `escribir_archivo`, `editar_archivo`, `eliminar_archivo`, `crear_carpeta`, `eliminar_carpeta` |

`context_audit.decision` también está cerrado a `incluido` y `descartado`, pero son solo dos valores y se documenta junto a la columna en [[database/01-schema/TABLES]].

## 2. Valores válidos

### Estado de sesión — `sessions.status`

| Valor | Significado |
|---|---|
| `inactiva` | La sesión existe pero no está ejecutando nada |
| `trabajando` | Está generando una respuesta o ejecutando una etapa |
| `esperando_permiso` | Está detenida hasta que decidas una aprobación |
| `terminada` | Su trabajo terminó correctamente |
| `error` | Se detuvo por un fallo |

### Rol de mensaje — `messages.role`

| Valor | Significado |
|---|---|
| `user` | Lo escribió la persona en la entrada de texto |
| `agent` | Lo respondió el modelo |

El razonamiento solo existe en mensajes con `role = agent`.

**No hay un rol `tool` en `messages`, y es deliberado.** Con el canal nativo de herramientas, cada turno puede incluir mensajes de herramienta: la petición del modelo y el resultado de cada ejecución. Esos mensajes existen **solo mientras dura el turno** y no entran a la conversación.

Lo que sí se guarda del turno es el mensaje `agent` final, que ya dice en su texto qué hizo y con qué resultado —lo que el usuario lee y lo que el modelo recibe como historial—, y, aparte, las **líneas de pantalla** de la ejecución en `chat_evento`. Esa separación es la clave: `messages` es la conversación, que es lo que va al contexto; `chat_evento` es lo que se pinta. Ver «Tipo de línea de chat», [[database/02-rules/DATA_FLOW]] y [[backend/DECISIONS]].

### Tipo de línea de chat — `chat_evento.tipo`

| Valor | Significado |
|---|---|
| `proceso` | El sub-proceso de una etapa de un flujo: «[Sub Proceso] Entender el problema» |
| `herramienta` | La línea de una herramienta, ya cerrada: «✓ LEER [AGENTS.md] · 93 líneas» |

Son líneas del hilo que se muestran al usuario, no turnos de conversación: al modelo no se le entregan. Viven en su propia tabla, `chat_evento`, para que esa frontera —se muestra / se envía al modelo— sea la frontera de la tabla y no un filtro que alguien pueda olvidar. Ver [[database/01-schema/TABLES]].

### Estado de aprobación — `approvals.status`

| Valor | Significado |
|---|---|
| `pendiente` | Espera tu decisión. Cuenta para el contador del panel |
| `aprobada` | La aceptaste y `build` aplicó el cambio |
| `declinada` | La rechazaste |
| `obsoleta` | Ya no aplica, porque la sesión terminó o el cambio se cayó |

`obsoleta` existe porque una sesión puede terminar mientras su aprobación esperaba: la línea se marca como obsoleta en vez de quedarse pidiendo una decisión que ya no sirve. Ver [[specs/SPEC-INTERFAZ-ATAJOS]].

### Operación de archivo — `change_history.operation`

Los seis valores coinciden con las seis herramientas de escritura del catálogo, para que el registro diga exactamente qué herramienta se aplicó.

| Valor | Herramienta | Qué registra |
|---|---|---|
| `crear_archivo` | `crear_archivo` | Un archivo nuevo. `before_content` es `NULL` |
| `escribir_archivo` | `escribir_archivo` | Sobrescritura. `before_content` guarda lo anterior |
| `editar_archivo` | `editar_archivo` | Cambio parcial. `before_content` guarda lo anterior |
| `eliminar_archivo` | `eliminar_archivo` | Borrado. `after_content` es `NULL` |
| `crear_carpeta` | `crear_carpeta` | Una carpeta nueva |
| `eliminar_carpeta` | `eliminar_carpeta` | Una carpeta borrada |

`crear_archivo` y `escribir_archivo` están separados a propósito: el primero falla si el archivo ya existe, el segundo lo sobrescribe. La base refleja esa diferencia.

**Una herramienta del usuario no genera una operación nueva.** Sus comandos se ejecutan bajo Landlock, que no le permite escribir en el proyecto, así que nunca hay una escritura que registrar. Si en el futuro se permitiera, sería un valor nuevo aquí y una migración; hoy el conjunto de seis sigue completo. Ver [[backend/03-security/SECURITY]].

## 3. Estados y transiciones

### Transiciones de sesión

```
inactiva ──▶ trabajando ──▶ inactiva
                │  ▲
                │  └──▶ esperando_permiso ──▶ trabajando
                ▼
        terminada
        error
```

- `inactiva` → `trabajando`: la sesión recibe una petición o el orquestador toma trabajo.
- `trabajando` → `inactiva`: terminó la etapa y espera otra.
- `trabajando` → `esperando_permiso`: el agente pidió una aprobación.
- `esperando_permiso` → `trabajando`: decidiste y el flujo continúa.
- `trabajando` → `terminada`: el trabajo acabó bien.
- Cualquier estado → `error`: hubo un fallo irrecuperable en la etapa.

`terminada` y `error` son finales: una sesión no vuelve a `trabajando` sin pasar antes por `inactiva`. La interfaz muestra `terminada` y `error` como estados finales distinguibles.

Una transición no válida se rechaza en la base con `CHECK`, no solo en el código.

### Transiciones de aprobación

```
pendiente ──▶ aprobada
           ──▶ declinada
           ──▶ obsoleta
```

- `aprobada`, `declinada` y `obsoleta` son finales. Una aprobación resuelta no vuelve a `pendiente`.
- Solo `pendiente` tiene `resolved_at` en `NULL`. Al resolver, se rellena.
- Pasar a `aprobada` no significa que el cambio se aplicó: significa que lo autorizaste. Lo que pasó después está en `change_history`, si el cambio llegó a aplicarse.

Cuando una sesión se elimina, sus aprobaciones `pendiente` se borran en cascada con ella y no pasan por `obsoleta`. La fila desaparece porque la sesión desaparece.

### Herramientas y aprobaciones

Un turno puede pedir varias herramientas, y cada una que lo necesite abre su propia aprobación. Las aprobaciones de un turno se resuelven **una a una, en el orden en que el modelo las pidió**: las ejecuciones son secuenciales, así que la segunda ni siquiera se intenta hasta que la primera está resuelta.

Una aprobación se vuelve `obsoleta` si la sesión termina mientras esperaba, o si el turno se cancela. El motivo es el mismo que con las aprobaciones normales: la decisión ya no sirve para nada.

## Referencias

- [[database/01-schema/TABLES]] — columnas donde se usan estos valores.
- [[database/01-schema/RELATIONSHIPS]] — cascadas y supervivencia del historial.
- [[database/01-schema/CONSTRAINTS]] — los `CHECK` que los validan.
- [[database/02-rules/DATA_FLOW]] — qué se guarda de un turno y qué no.
- [[specs/SPEC-SESIONES]] — de dónde salen los estados de sesión.
- [[specs/SPEC-TOOLS]] — el catálogo del que salen las operaciones de archivo y las herramientas del usuario.
