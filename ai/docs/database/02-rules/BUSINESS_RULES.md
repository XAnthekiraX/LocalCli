---
title: LocalCli — reglas de negocio de la base de datos
tags: [database, reglas]
depende_de:
  - "[[database/01-schema/SCHEMA]]"
  - "[[database/01-schema/TABLES]]"
  - "[[database/01-schema/CONSTRAINTS]]"
relacionado:
  - "[[database/DATABASE]]"
  - "[[database/01-schema/ENUMS]]"
  - "[[database/01-schema/RELATIONSHIPS]]"
  - "[[database/02-rules/DATA_FLOW]]"
  - "[[specs/SPEC-ARCHIVOS]]"
  - "[[specs/SPEC-AGENTE-BASE]]"
  - "[[specs/SPEC-NODO-CONTEXTO]]"
  - "[[specs/SPEC-PANEL-CONTEXTO]]"
---
# BUSINESS_RULES — Reglas de negocio de la base de datos

## 1. Reglas de negocio

Reglas que la base no puede imponer sola, porque dependen del contexto o de lo que decidió el usuario.

- **Toda escritura pasa por aprobación.** Nada llega a `change_history` sin que antes un usuario lo autorizara. La regla vive en el módulo de herramientas, no en la base. Ver [[specs/SPEC-ARCHIVOS]].
- **`plan` no escribe.** El razonamiento de un agente de planificación no produce entradas en `change_history`. Solo el agente de construcción, tras el relevo, lo hace. Ver [[specs/SPEC-AGENTE-BASE]].
- **Una aprobación habilita solo el cambio propuesto.** Que exista una fila `aprobada` no autoriza cambios distintos de los que se describen en su campo `description`.
- **Nada se borra de la base.** El contenido de `change_history` no se modifica ni se elimina nunca, ni siquiera al corregir un cambio posterior. Ver [[database/DATABASE]].
- **El borrado de una sesión es definitivo.** Borra la conversación y su estado, y deja `change_history` con `session_id` en `NULL`. Ver [[database/01-schema/RELATIONSHIPS]].
- **`change_history` es la memoria del proyecto.** Un cambio se aplica, y su registro queda con lo que había antes, para poder revertir y auditar. No es un log temporal.
- **La selección de contexto siempre está justificada.** Una fila `context_audit` con `decision = 'descartado'` necesita `reason`; el nodo de contexto no descarta nada sin explicar por qué. Ver [[specs/SPEC-NODO-CONTEXTO]].
- **El grafo y la cola derivan de los archivos.** SQLite no guarda un índice de documentos ni de tareas. Si un archivo y la base discrepan, manda el archivo.
- **`obsoleta` no es `declinada`.** Una aprobación que ya no aplica porque su sesión terminó se marca `obsoleta`, no `declinada`: el usuario no la rechazó, dejó de tener sentido.

## 2. Invariantes

Condiciones que siempre deben cumplirse.

- Una sesión no tiene dos razonamientos para el mismo mensaje.
- Un mensaje con razonamiento tiene `role = agent`.
- Una aprobación `pendiente` tiene `resolved_at` en `NULL`, y resuelta lo tiene relleno.
- Un cambio con `crear_archivo` tiene `before_content` en `NULL`; con `eliminar_archivo`, `after_content` en `NULL`.
- Toda fila de `change_history` se puede leer aunque su sesión ya no exista.
- La suma de `context_audit` de una etapa en `incluido` no supera el límite de contexto del modelo. Esto no lo comprueba la base: lo garantiza el nodo de contexto antes de escribir. Ver [[specs/SPEC-NODO-CONTEXTO]].
- Una sesión con estado `esperando_permiso` tiene al menos una aprobación `pendiente`.
- El estado de una sesión refleja su situación real: `terminada` solo cuando terminó su trabajo, `error` solo ante un fallo.

## 3. Casos especiales

- **La sesión termina con una aprobación pendiente.** Las aprobaciones `pendiente` de esa sesión pasan a `obsoleta` en lugar de desaparecer, para que el historial de la sesión siga siendo coherente. Si la sesión se borra, en cambio, sí desaparecen en cascada.
- **Se borra una sesión con cambios aplicados.** Los registros de `change_history` quedan con `session_id` en `NULL`. Ya no se sabe qué sesión los hizo, pero se sigue sabiendo qué se cambió en los archivos.
- **Una etapa descarta un documento y luego lo necesita.** La auditoría de la etapa siguiente es un registro nuevo. Nada se corrige en la auditoría anterior, porque es un registro de lo que pasó, no del estado actual.
- **Un modelo no reporta tokens.** `input_tokens` y `output_tokens` quedan en `NULL`, no en `0`. El panel de contexto lo muestra como estimación o como no disponible. Ver [[specs/SPEC-PANEL-CONTEXTO]].
- **Un modelo no devuelve razonamiento.** No se crea fila en `reasoning`, y la interfaz indica que no está disponible. Un mensaje de agente puede no tener razonamiento.
- **Una operación de archivo falla a mitad.** Se escribe una sola fila en `change_history`, con el resultado real: si el archivo quedó como estaba, la operación no se registra. La base registra lo que ocurrió, no lo que se intentó.
- **Dos sesiones en segundo plano escriben a la vez.** WAL y las transacciones por operación lógica evitan que se pisen. Si una transacción falla, la etapa se marca con error y no se pierde en silencio. Ver [[database/01-schema/SCHEMA]].

## Referencias

- [[database/DATABASE]] — políticas de la base.
- [[database/01-schema/SCHEMA]] — estructura.
- [[database/01-schema/TABLES]] — columnas donde se aplican estas reglas.
- [[database/01-schema/ENUMS]] — estados y transiciones.
- [[database/01-schema/CONSTRAINTS]] — lo que la base sí impone.
- [[database/02-rules/DATA_FLOW]] — cómo circula y de dónde vienen los datos.
- [[specs/SPEC-ARCHIVOS]] — reglas de permiso que aquí no se imponen.
