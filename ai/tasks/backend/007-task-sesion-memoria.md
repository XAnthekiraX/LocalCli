# 007-task-sesion-memoria.md

> T-B007 — Sesión y memoria de conversación (crear, cambiar, historial).
> Acción de esta descomposición: `crear`.

## Referencias

- [[specs/SPEC-SESIONES]] §Flujo principal, §Reglas de negocio, §Criterios de aceptación
- [[specs/SPEC-HISTORIAL-CONVERSACION]] §Flujo principal, §Reglas de negocio, §Criterios de aceptación
- [[backend/01-domain/DOMAIN]] §1 Módulos (`session`), §3 Relaciones de Dominio
- [[backend/01-domain/BUSINESS_RULES]] §Sesiones, §Notificaciones, §Invariantes
- [[backend/02-interfaces/INTERFACES-GENERAL]] §5 Contratos (`session`)
- [[backend/04-infrastructure/CONFIGURATION]] §2 `LOCALCLI_CONTEXT_LIMIT`, §7 Preferencias (`historial_tokens`)
- [[database/01-schema/ENUMS]] §2 Valores válidos (estado de sesión), §3 Transiciones de sesión
- [[database/02-rules/DATA_FLOW]] §Qué se guarda de un turno

## Contexto

`session` es el ciclo de vida de las sesiones y su memoria. Una sesión pertenece a un solo proyecto y su contenido no se ve desde otra sesión ni desde otra carpeta: el aislamiento lo garantiza `store` filtrando siempre por `project_id`. El módulo no ejecuta tareas (eso es `flow` y `queue`), ni decide qué documentación entra al contexto (eso es `context`), ni construye la llamada del agente (eso es `agent`): mantiene las sesiones, su estado, su historial de conversación y su título, y expone el resultado por eventos a la TUI.

El historial que se entrega al modelo son solo los mensajes ya cerrados (`role = user`/`agent`); las líneas de procesamiento del hilo viven en `chat_evento` y **no** viajan al modelo. Cuando el historial no cabe en el presupuesto se resume lo antiguo y lo reciente viaja entero.

La persistencia en SQLite (sesiones y mensajes) y el par motor/modelo de la sesión quedan fuera de esta tarea; aquí solo se consume lo que proveen `store` (T-B010) y el catálogo (Bloque 2).

## Tareas pequeñas

| ID        | Acción | Tarea                                                                                             | Estado    | Archivos                        | Verificación                                                                          |
| --------- | ------ | ------------------------------------------------------------------------------------------------- | --------- | ------------------------------- | ------------------------------------------------------------------------------------- |
| T-B007-01 | crear  | Estructura del módulo `session`: entidad sesión, estados cerrados y transiciones válidas          | completada | `internal/session/sesion.go`    | Una transición no válida (`terminada`→`trabajando`) se rechaza                        |
| T-B007-02 | crear  | Crear sesión («Nueva sesión»), retomar/cambiar y listar las del proyecto                          | completada | `internal/session/sesion.go`    | Al abrir una carpeta se listan sus sesiones y se retoma cualquiera con su historial   |
| T-B007-03 | crear  | Ejecución en segundo plano: cambiar de sesión no detiene lo que corre                             | completada | `internal/session/sesion.go`    | Cambiar de sesión mientras otra trabaja no interrumpe su ejecución                    |
| T-B007-04 | crear  | Publicar y persistir el estado de la sesión; notificar al terminar                                | completada | `internal/session/sesion.go`    | El cambio de estado se emite y una sesión terminada notifica                          |
| T-B007-05 | crear  | Reconstruir el historial para el modelo: solo `user`/`agent`, `chat_evento` excluido              | completada | `internal/session/historial.go` | El historial armado no incluye líneas de proceso ni de herramienta                    |
| T-B007-06 | crear  | Presupuesto de tokens del historial: `historial_tokens` → `LOCALCLI_CONTEXT_LIMIT` → por defecto  | completada | `internal/session/historial.go` | Sin preferencia se usa el tope de entorno; sin ambos, el valor por defecto            |
| T-B007-07 | crear  | Compactación por resumen reutilizable; un fallo no bloquea (tramo reciente sin resumen)           | completada | `internal/session/historial.go` | Un historial que no cabe se entrega como resumen + reciente, dentro del presupuesto   |
| T-B007-08 | crear  | Título de la sesión con la primera petición: una sola vez; si falla, conserva provisional         | completada | `internal/session/titulo.go`    | Tras la primera petición el nombre lo genera el modelo; si falla sigue «Nueva sesión» |
| T-B007-09 | actualizar | Guardar `ultima_peticion_usuario` (texto) en sesión/modelo para soportar `/redo` sin argumento | pendiente | `internal/session/sesion.go` | La sesión recuerda la última petición de usuario enviada y la expone para re-ejecución |

**Dependencias:** `01`→`02`→`03`,`04`; `01`→`05`→`06`→`07`; `02`→`08`.
**Fuera de alcance:** la persistencia SQLite de sesiones y mensajes (T-B010); la selección de documentos del contexto (T-B009); el ciclo conversacional del agente (T-B008); la TUI de chat (T-B011); el modelo por sesión y el ciclo de `llama-server` (Bloque 2).
