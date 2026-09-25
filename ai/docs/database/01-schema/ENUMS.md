# ENUMS — Valores cerrados de la base de datos

## 1. Enums

La base de datos tiene cuatro columnas con un conjunto cerrado de valores. Cada una se valida con `CHECK`, no solo por convención. Ver [[database/01-schema/CONSTRAINTS]].

| Enum | Columna | Valores |
|---|---|---|
| Estado de sesión | `sessions.status` | `inactiva`, `trabajando`, `esperando_permiso`, `terminada`, `error` |
| Rol de mensaje | `messages.role` | `user`, `agent` |
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

- `inactiva` → `trabajando`: la sesión recibe una petición o el motor toma trabajo.
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

## Referencias

- [[database/01-schema/TABLES]] — columnas donde se usan estos valores.
- [[database/01-schema/RELATIONSHIPS]] — cascadas y supervivencia del historial.
- [[database/01-schema/CONSTRAINTS]] — los `CHECK` que los validan.
- [[specs/SPEC-SESIONES]] — de dónde salen los estados de sesión.
- [[specs/SPEC-TOOLS]] — el catálogo del que salen las operaciones de archivo.
