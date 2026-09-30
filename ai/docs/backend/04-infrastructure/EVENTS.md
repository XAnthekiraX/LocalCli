---
title: LocalCli — eventos del motor
tags: [backend, infraestructura]
depende_de:
  - "[[backend/01-domain/DOMAIN]]"
  - "[[specs/SPEC-INTERFAZ]]"
  - "[[specs/SPEC-SESIONES]]"
relacionado:
  - "[[backend/01-domain/BUSINESS_RULES]]"
  - "[[backend/04-infrastructure/INTEGRATIONS]]"
  - "[[database/01-schema/ENUMS]]"
  - "[[database/01-schema/TABLES]]"
  - "[[database/03-operations/QUERIES]]"
  - "[[backend/02-interfaces/TOOLS]]"
---
# EVENTS — Eventos del motor

No hay webhooks ni cola de mensajes externa. Los "eventos" aquí son notificaciones internas, por canales de Go, que un módulo emite y otro recibe. La TUI no pregunta nada: se le notifica.

## 1. Eventos disponibles

| Evento | Quién lo emite | Para qué |
|---|---|---|
| `token` | `ollama` | Un fragmento de respuesta o de razonamiento, para el streaming en vivo |
| `estado_sesion` | `session` | La sesión cambió de estado (inactiva, trabajando, esperando permiso, terminada, con error) |
| `titulo_sesion` | `session` | La sesión cambió de nombre: el modelo generó su título a partir de la primera petición |
| `notificacion` | `session`, `flow` | Aviso de que una sesión espera permiso o ha terminado, aunque el usuario no la esté viendo |
| `peticion_aprobacion` | `fileops`, `exec` | Hay una escritura o un comando esperando tu decisión |
| `aprobacion_resuelta` | `fileops`, `exec` | Se aprobó o se declinó una petición pendiente |
| `herramienta_invocada` | `tools` | El modelo pidió una herramienta y va a ejecutarse; con qué nombre, verbo y objetivo |
| `herramienta_resultado` | `tools` | La ejecución terminó, con o sin error, con la medida del resultado, si su salida se recortó y cuánto tardó |
| `tokens_turno` | adaptador de arranque | El consumo del turno (tokens de entrada y de salida) y el total del contexto del chat con su límite, para el panel de contexto y la línea de estado |
| `etapa_iniciada` | `flow` | Una etapa del flujo empezó; la vista la anuncia como sub-proceso con su nombre |
| `etapa_terminada` | `flow` | Una etapa terminó, con su nombre y un resumen de su resultado |
| `etapa_fallida` | `flow` | Una etapa falló; el flujo se detiene |
| `flujo_pausado` | `flow` | El flujo espera una aprobación |
| `flujo_reanudado` | `flow` | El flujo continúa desde donde estaba |
| `flujo_cancelado` | `flow` | El usuario canceló el flujo |
| `cola_actualizada` | `queue` | La cola cambió: se re-derivó, una tarea empezó, terminó o se bloqueó |
| `elemento_bloqueado` | `queue` | Un elemento del TODO no puede arrancar por falta de lo que depende |
| `todo_actualizada` | adaptador de arranque | La lista de pasos de la sesión se reescribió; el panel la repinta |
| `cambio_aplicado` | `fileops` | Un cambio se aplicó y quedó registrado en `change_history` |
| `contexto_auditado` | `context` | Una etapa terminó de armar su contexto; se registró la auditoría |

Los estados de sesión y sus transiciones están en [[database/01-schema/ENUMS]].

### Por qué hacen falta los dos eventos de herramienta

Con el contrato en prosa, la llamada a una herramienta viajaba dentro del texto que el modelo escribía, y el usuario la veía porque era texto en pantalla. Con el canal nativo la llamada **sale del texto**: el modelo pide la herramienta y lo que escribe es otra cosa.

Sin un evento, el efecto sería que mientras el agente lee un archivo o corre las pruebas la pantalla se queda quieta, y de repente aparece una respuesta como si nada. En el caso peor —una escritura esperando tu aprobación— el silencio es justo lo que más confunde.

Por eso los emite `tools` y no `agent`: la capa universal ya está en ese punto para todas las herramientas, incluidas las del usuario, y es el único sitio donde se puede emitir **antes** de ejecutar y **después**, sin instrumentar catorce handlers.

## 2. Productores y consumidores

| Evento | Consumidor principal | Otros consumidores |
|---|---|---|
| `token` | `tui` | `ollama` lo produce; nadie más lo consume |
| `estado_sesion` | `tui` | `queue` lo usa para saber si puede tomar una tarea |
| `titulo_sesion` | `tui` | Actualiza el nombre en el panel (si es la sesión activa) y en la fila del modal de sesiones |
| `notificacion` | `tui` | Llega aunque la sesión no sea la que se está viendo, o el usuario esté en otra carpeta |
| `peticion_aprobacion` | `tui` | Se guarda la fila en `approvals` para que sobreviva y sea visible desde otra sesión |
| `aprobacion_resuelta` | `flow`, `queue` | La etapa pausada continúa |
| `herramienta_invocada` | `tui` | Los hooks de `tools` lo usan para auditoría y métricas |
| `herramienta_resultado` | `tui` | Los hooks de `tools` lo usan para auditoría y métricas |
| `tokens_turno` | `tui` | Alimenta la fila de contexto del panel y la línea de tokens bajo la entrada |
| `etapa_iniciada` / `terminada` / `fallida` | `tui` | `etapa_iniciada` pinta la línea `[Sub Proceso] <nombre>`; `queue` marca la tarea al terminar |
| `flujo_pausado` / `reanudado` / `cancelado` | `tui` | `session` refleja el estado |
| `cola_actualizada` / `elemento_bloqueado` | `tui` | Visible desde cualquier sesión |
| `todo_actualizada` | `tui` | Repinta la lista de pasos del panel; es de la sesión activa, la de otra no entra |
| `cambio_aplicado` | `tui` | `store` ya lo tiene en `change_history` |
| `contexto_auditado` | `tui` | El usuario lo consulta |

`tui` recibe estos eventos por el bus. No importa `tools`: la prohibición de imports entre módulos ([[backend/01-domain/DOMAIN]]) lo impide, y además no lo necesita —pinta lo que le llega.

## 3. Payloads

Los payloads llevan lo mínimo para que el consumidor pueda pintar o decidir. No duplican lo que ya está en la base: llevan identificadores, y quien necesite el detalle lo lee del esquema.

**Todo evento de una sesión lleva su `sesion`** (el identificador de la sesión que lo produjo): la vista pinta la sesión activa y descarta lo de las demás, que siguen vivas en segundo plano. La sesión viaja en el contexto del turno (`tools.ConSesion`), estampada una vez por turno, y no en una variable global: con varias sesiones concurrentes, cada evento se atribuye a la suya. Los eventos de alcance de proyecto (`cola_actualizada`, `elemento_bloqueado`, `cambio_aplicado`) no la llevan; `notificacion` sí, y aun así se pinta aunque no sea la sesión activa ([[specs/SPEC-SESIONES]]).

- **`token`:** el texto del fragmento y si es razonamiento o respuesta final. Es lo único que va token a token, porque la pantalla lo muestra en vivo. Ver [[specs/SPEC-INTERFAZ]].
- **`estado_sesion`:** el identificador de la sesión y el nuevo estado. El nombre de la capa y la marca de tiempo se leen de la base si hacen falta. Ver [[database/01-schema/TABLES]].
- **`titulo_sesion`:** el identificador de la sesión y su nuevo nombre (el título que el modelo generó a partir de la primera petición). El id no viaja cambiado: renombrar solo toca el nombre. Ver [[specs/SPEC-SESIONES]].
- **`notificacion`:** el identificador de la sesión, el motivo (esperando permiso o terminada) y una línea con qué hacer a continuación. No lleva el contenido: la aprobación está en `approvals` y el resultado en el historial de la sesión. Se emite en los dos casos, se vea la sesión o no. El motivo de «terminada» distingue el camino: un flujo o la cola anuncian «la sesión terminó el trabajo» y un turno de chat anuncia «la sesión terminó de responder» (NotificacionDeChat), porque el chat no ejecuta etapas. Ver [[backend/01-domain/BUSINESS_RULES]].
- **`peticion_aprobacion`:** el identificador de la aprobación y una descripción de lo que se pide (qué archivo, o qué comando). La fila completa está en `approvals`.
- **`etapa_iniciada`:** el identificador de la etapa y su **nombre visible**. La vista lo pinta como `[Sub Proceso] <nombre>`; el texto del paso no viaja —los pasos intermedios de un flujo corren en silencio—. La última etapa de un flujo con `bloque_contexto` es la composición: corre sin herramientas y su entrega es la que se muestra. Ver [[specs/SPEC-MOTOR-FLUJOS]].
- **`etapa_terminada`:** el identificador de la etapa, su nombre y un resumen de su resultado. Lo que recibió la etapa y qué entregó está en `context_audit`.
- **`cambio_aplicado`:** el identificador del cambio y la ruta del archivo. El antes y el después están en `change_history`.
- **`cola_actualizada`:** la capa y un resumen de la cola (cuántas pendientes, cuál activa, cuál bloqueada). El detalle viene de los archivos de tarea.
- **`todo_actualizada`:** el identificador de la sesión y la lista completa de pasos, cada uno con su contenido y su estado. No lleva la prioridad: el panel no la muestra. La lista entera reemplaza a la anterior; el panel solo la aplica si la sesión es la activa.
- **`herramienta_invocada`:** el nombre de la herramienta, el agente que la pidió, su **verbo** de pantalla («LEER», «EJEC») y su **tema**: el objetivo declarado por el catálogo (la ruta, el patrón, el comando, la consulta), colapsado y recortado. El tema es lo único de los argumentos que viaja —y solo ese campo—: el resto (cuerpos de archivo, credenciales) no se expone, porque esto va a la pantalla y a la auditoría.
- **`herramienta_resultado`:** el nombre de la herramienta, si terminó bien, si hubo error, la **medida** del resultado («70 líneas», «3 coincidencias»), si la salida se recortó y la **duración** de la ejecución. **No lleva la salida.** El resultado va al modelo, no a la pantalla: en la TUI bastan el nombre, el estado, la medida y el tiempo que tardó. La duración es el dato que permite pintar cuánto tardó cada línea del hilo y conservarlo al recuperar la sesión; lo mide la capa universal, no la herramienta.
- **`tokens_turno`:** los tokens de entrada y de salida del turno, más el total del contexto del chat (`contexto`) y su límite (`limite`). Mientras el turno corre, la TUI aproxima el consumo del turno contando los fragmentos de `token`; al llegar este evento, el valor exacto lo corrige. `contexto` es la estimación del chat que forma el contexto de la sesión (mensajes de usuario y agente; las líneas de procesamiento no cuentan) y `limite`, la ventana del modelo: juntos alimentan la fila CONTEXTO del panel. Ver [[specs/SPEC-PANEL-CONTEXTO]].

Los dos los emite `tools` en la capa universal, y solo cuando hay un consumidor. En producción, la TUI; si no hay nadie escuchando, no se construyen los payloads. Los hooks de `tools` los aprovechan sin ser consumidores del bus.

## 4. Sincronía y consistencia

- **Los eventos son notificaciones, no comandos.** Un consumidor no le pide nada al productor; decide qué hacer con lo que oye.
- **Son unidireccionales.** Un módulo no espera respuesta por el canal de eventos. Si necesita algo, lo pide por su dependencia declarada, no por un evento de vuelta.
- **La consistencia la garantiza la base, no el evento.** Cuando un evento dice que algo cambió, el cambio ya está escrito. Por ejemplo, `cambio_aplicado` se emite después de que `fileops` registró el cambio en `change_history`, no antes. Ver [[database/03-operations/QUERIES]].
- **Las líneas de procesamiento se guardan, no solo se emiten.** Además de emitir `herramienta_resultado` y `etapa_iniciada`, el adaptador persiste la línea en `chat_evento`: es parte del hilo que se recupera al volver a una sesión y **no** entra al contexto del modelo. La línea de herramienta se persiste **con su duración**, para que el hilo recuperado lleve los mismos tiempos que se vieron en vivo. Ver [[database/02-rules/DATA_FLOW]].
- **El streaming es la excepción a "notificar después":** los tokens se emiten mientras se generan, porque la pantalla los muestra en vivo. Esa asincronía es intencional.
- **Un consumidor lento no bloquea al productor.** Los canales de Go hacen de cola entre la TUI y el trabajo de fondo; una sesión en segundo plano sigue trabajando aunque la pantalla no esté al día. Ver [[specs/SPEC-SESIONES]].
- **Los eventos de herramienta no pausean el turno.** `herramienta_invocada` se emite y la ejecución sigue; es una notificación, no un punto de espera. La única interrupción real es `peticion_aprobacion`, y esa la decide el usuario.

## Referencias

- [[backend/01-domain/DOMAIN]] — los módulos y sus límites.
- [[backend/02-interfaces/TOOLS]] — la capa universal que emite los eventos de herramienta, y los hooks que los aprovechan.
- [[backend/04-infrastructure/INTEGRATIONS]] — de dónde vienen el streaming y las respuestas.
- [[database/01-schema/ENUMS]] — estados de sesión y transiciones.
- [[specs/SPEC-SESIONES]] — sesiones en segundo plano y su estado visible.
