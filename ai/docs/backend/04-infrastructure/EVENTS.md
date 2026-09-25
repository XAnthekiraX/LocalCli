# EVENTS — Eventos del motor

No hay webhooks ni cola de mensajes externa. Los "eventos" aquí son notificaciones internas, por canales de Go, que un módulo emite y otro recibe. La TUI no pregunta nada: se le notifica.

## 1. Eventos disponibles

| Evento | Quién lo emite | Para qué |
|---|---|---|
| `token` | `ollama` | Un fragmento de respuesta o de razonamiento, para el streaming en vivo |
| `estado_sesion` | `session` | La sesión cambió de estado (inactiva, trabajando, esperando permiso, terminada, con error) |
| `notificacion` | `session`, `flow` | Aviso de que una sesión espera permiso o ha terminado, aunque el usuario no la esté viendo |
| `peticion_aprobacion` | `fileops`, `exec` | Hay una escritura o un comando esperando tu decisión |
| `aprobacion_resuelta` | `fileops`, `exec` | Se aprobó o se declinó una petición pendiente |
| `etapa_iniciada` | `flow` | Una etapa del flujo empezó |
| `etapa_terminada` | `flow` | Una etapa terminó, con su resultado |
| `etapa_fallida` | `flow` | Una etapa falló; el flujo se detiene |
| `flujo_pausado` | `flow` | El flujo espera una aprobación |
| `flujo_reanudado` | `flow` | El flujo continúa desde donde estaba |
| `flujo_cancelado` | `flow` | El usuario canceló el flujo |
| `cola_actualizada` | `queue` | La cola cambió: se re-derivó, una tarea empezó, terminó o se bloqueó |
| `elemento_bloqueado` | `queue` | Un elemento del TODO no puede arrancar por falta de lo que depende |
| `cambio_aplicado` | `fileops` | Un cambio se aplicó y quedó registrado en `change_history` |
| `contexto_auditado` | `context` | Una etapa terminó de armar su contexto; se registró la auditoría |

Los estados de sesión y sus transiciones están en [[database/01-schema/ENUMS]].

## 2. Productores y consumidores

| Evento | Consumidor principal | Otros consumidores |
|---|---|---|
| `token` | `tui` | `ollama` lo produce; nadie más lo consume |
| `estado_sesion` | `tui` | `queue` lo usa para saber si puede tomar una tarea |
| `notificacion` | `tui` | Llega aunque la sesión no sea la que se está viendo, o el usuario esté en otra carpeta |
| `peticion_aprobacion` | `tui` | Se guarda la fila en `approvals` para que sobreviva y sea visible desde otra sesión |
| `aprobacion_resuelta` | `flow`, `queue` | La etapa pausada continúa |
| `etapa_iniciada` / `terminada` / `fallida` | `tui` | `queue` marca la tarea al terminar |
| `flujo_pausado` / `reanudado` / `cancelado` | `tui` | `session` refleja el estado |
| `cola_actualizada` / `elemento_bloqueado` | `tui` | Visible desde cualquier sesión |
| `cambio_aplicado` | `tui` | `store` ya lo tiene en `change_history` |
| `contexto_auditado` | `tui` | El usuario lo consulta |

## 3. Payloads

Los payloads llevan lo mínimo para que el consumidor pueda pintar o decidir. No duplican lo que ya está en la base: llevan identificadores, y quien necesite el detalle lo lee del esquema.

- **`token`:** el texto del fragmento y si es razonamiento o respuesta final. Es lo único que va token a token, porque la pantalla lo muestra en vivo. Ver [[specs/SPEC-INTERFAZ]].
- **`estado_sesion`:** el identificador de la sesión y el nuevo estado. El nombre de la capa y la marca de tiempo se leen de la base si hacen falta. Ver [[database/01-schema/TABLES]].
- **`notificacion`:** el identificador de la sesión, el motivo (esperando permiso o terminada) y una línea con qué hacer a continuación. No lleva el contenido: la aprobación está en `approvals` y el resultado en el historial de la sesión. Se emite en los dos casos, se vea la sesión o no. Ver [[backend/01-domain/BUSINESS_RULES]].
- **`peticion_aprobacion`:** el identificador de la aprobación y una descripción de lo que se pide (qué archivo, o qué comando). La fila completa está en `approvals`.
- **`etapa_terminada`:** el identificador de la etapa y un resumen de su resultado. Lo que recibió la etapa y qué entregó está en `context_audit`.
- **`cambio_aplicado`:** el identificador del cambio y la ruta del archivo. El antes y el después están en `change_history`.
- **`cola_actualizada`:** la capa y un resumen de la cola (cuántas pendientes, cuál activa, cuál bloqueada). El detalle viene de los archivos de tarea.

## 4. Sincronía y consistencia

- **Los eventos son notificaciones, no comandos.** Un consumidor no le pide nada al productor; decide qué hacer con lo que oye.
- **Son unidireccionales.** Un módulo no espera respuesta por el canal de eventos. Si necesita algo, lo pide por su dependencia declarada, no por un evento de vuelta.
- **La consistencia la garantiza la base, no el evento.** Cuando un evento dice que algo cambió, el cambio ya está escrito. Por ejemplo, `cambio_aplicado` se emite después de que `fileops` registró el cambio en `change_history`, no antes. Ver [[database/03-operations/QUERIES]].
- **El streaming es la excepción a "notificar después":** los tokens se emiten mientras se generan, porque la pantalla los muestra en vivo. Esa asincronía es intencional.
- **Un consumidor lento no bloquea al productor.** Los canales de Go hacen de cola entre la TUI y el trabajo de fondo; una sesión en segundo plano sigue trabajando aunque la pantalla no esté al día. Ver [[specs/SPEC-SESIONES]].

## Referencias

- [[backend/01-domain/DOMAIN]] — los módulos y sus límites.
- [[backend/04-infrastructure/INTEGRATIONS]] — de dónde vienen el streaming y las respuestas.
- [[database/01-schema/ENUMS]] — estados de sesión y transiciones.
- [[specs/SPEC-SESIONES]] — sesiones en segundo plano y su estado visible.
