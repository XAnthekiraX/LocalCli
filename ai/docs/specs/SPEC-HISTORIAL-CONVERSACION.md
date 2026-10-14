---
title: SPEC — Historial de conversación
tags: [specs, requisito]
depende_de:
  - "[[specs/SPEC-SESIONES]]"
  - "[[specs/SPEC-NODO-CONTEXTO]]"
relacionado:
  - "[[backend/04-infrastructure/CONFIGURATION]]"
---
# SPEC — Historial de conversación

Prioridad: P0 (núcleo)

## Propósito

Que el chat recuerde: cada petición se responde con la conversación anterior de la sesión, no solo con el mensaje nuevo, y esa conversación se compacta cuando no cabe en el presupuesto de contexto del modelo.

## Alcance

Incluye cómo se reconstruye el historial de una sesión para el modelo, el presupuesto de tokens, la compactación por resumen y su configuración (disparadores, formato y modelo).
No incluye qué documentos entran al contexto, que está en [[specs/SPEC-NODO-CONTEXTO]]; ni el ciclo de vida de la sesión, que está en [[specs/SPEC-SESIONES]]; ni el almacenamiento de los mensajes, que está en [[database/DATABASE]].

## Actores

- **Usuario**: mantiene una conversación con la sesión activa.
- **Sistema**: reconstruye el historial, lo compacta y lo entrega al modelo.

## Flujo principal

1. El usuario envía una petición a la sesión activa.
2. El sistema lee los mensajes ya cerrados de esa sesión, en orden.
3. Estima sus tokens y los compara con el presupuesto.
4. Si caben, los entrega enteros junto con el mensaje nuevo.
5. Si no, resume los más antiguos y entrega el resumen seguido de los recientes.
6. El modelo responde con esa conversación como contexto.
7. La respuesta y el mensaje del usuario quedan guardados para el turno siguiente.

## Reglas de negocio

- Solo la sesión activa aporta historial: el contexto de una sesión nunca se mezcla con el de otra.
- El historial se arma solo de los mensajes de usuario y agente. Las **líneas de procesamiento** del hilo (sub-procesos y herramientas) se muestran al usuario y se recuperan al abrir una sesión, pero **no** viajan al modelo: viven en `chat_evento`, aparte de la conversación. Ver [[database/02-rules/DATA_FLOW]].
- El historial es la conversación ya cerrada; el mensaje en curso viaja aparte, como contexto del turno, y no se duplica.
- El historial que se entrega al modelo no supera el presupuesto de tokens configurado.
- Cuando el historial no cabe, se resume lo antiguo y lo reciente viaja entero y literal.
- El resumen es un dato derivado: no sustituye a los mensajes, que siguen completos en el historial de la sesión y en la pantalla.
- El resumen se reutiliza entre turnos mientras el tramo reciente siga cabiendo en el presupuesto.
- Un fallo al resumir no bloquea la respuesta: se entrega el tramo reciente sin resumen.
- El presupuesto se configura en las preferencias del usuario (`~/.config/localcli/config.json`); sin preferencia se usa `LOCALCLI_CONTEXT_LIMIT` y, en su defecto, un valor por defecto.
- La compactación se dispara por dos vías: **automática**, cuando el tamaño estimado del historial supera el umbral de la ventana; y **por desbordamiento**, cuando el servidor rechaza el turno porque la petición excede su contexto. La vía automática resume antes de enviar; la de desbordamiento resume con un tramo reciente menor y **reintenta el turno una sola vez**.
- El **umbral** de la vía automática es una fracción de la ventana efectiva (60 % por defecto), configurable.
- El **tramo reciente que viaja literal** tras el resumen se mide en tokens (`conservar_tokens`, 15000 por defecto), configurable; nunca puede superar el presupuesto del historial.
- El resumen se pide a un **modelo de compactación** configurable, pensado para ser ligero; sin preferencia es el modelo activo de la sesión.
- El resumen tiene un **formato** configurable: `opencode` (por defecto, con las secciones Objective / Important Details / Work State / Next Move / Relevant Files), `legacy` (resumen en prosa) o `custom` con una plantilla propia. El prompt pide devolver solo el resumen.
- El resumen se **persiste con la sesión** (`sessions.summary` y `sessions.summary_upto`): no se resume dos veces lo mismo. Si llegan mensajes nuevos que no caben, el resumen persistido se **amplía** con lo añadido en lugar de rehacerse.
- Un resumen **obsoleto** —cuando se deshace hasta un punto anterior a su corte— se descarta y se vuelve a compactar desde cero.
- `/compactar` **fuerza** la compactación de la sesión activa aunque el historial quepa en el presupuesto.

## Disparadores y configuración

- El umbral, el tramo conservado, el formato, la plantilla y el modelo de compactación viven en el bloque `compactacion` de `~/.config/localcli/config.json`; su ausencia no rompe el arranque y aplica los valores por defecto. Ver [[backend/04-infrastructure/CONFIGURATION]].
- La compactación automática respeta la ventana efectiva: el umbral se aplica sobre esa ventana, no sobre un valor fijo.
- La compactación por desbordamiento es la red de seguridad: cubre el caso en que la estimación de tokens se queda corta y el servidor rechaza la petición real.

## Formato y modelo del resumen

- El formato por defecto es `opencode`, elegido porque conserva lo que un turno posterior necesita: el objetivo, los detalles que no se pueden redescubrir, el estado del trabajo (hecho, en curso, bloqueado), el siguiente movimiento y los archivos relevantes.
- El formato `legacy` mantiene el resumen anterior en pocas líneas para no romper configuraciones existentes; `custom` usa la plantilla literal que declare el usuario.
- El modelo de compactación puede ser distinto del que conversa —uno más pequeño basta para resumir— y se le pide sin razonamiento y con una salida acotada.

## Criterios de aceptación

- [ ] El modelo recibe la conversación anterior de la sesión, no solo el último mensaje.
- [ ] Dos sesiones del mismo proyecto no comparten historial.
- [ ] Un historial corto se entrega entero.
- [ ] Un historial largo se entrega como resumen + tramo reciente, dentro del presupuesto.
- [ ] El resumen se reutiliza entre turnos mientras el tramo reciente siga cabiendo.
- [ ] Un fallo del resumen no impide responder.
- [ ] Al superar el umbral, el historial se compacta antes de enviar el turno.
- [ ] Si el servidor rechaza el turno por desbordamiento, se compacta y se reintenta una vez.
- [ ] El resumen persistido se amplía de forma incremental en lugar de rehacerse entero.
- [ ] `/compactar` resume la sesión activa aunque el historial quepa.
- [ ] El formato y el modelo del resumen se toman de la preferencia; sin ella, del valor por defecto.
- [ ] Deshacer hasta un punto anterior al corte descarta el resumen obsoleto.

## Requisitos no funcionales

- Reconstruir el historial no bloquea la interfaz: ocurre en la goroutine de la sesión.
- No se añaden tablas: el historial sale de los mensajes ya persistidos.

## Dependencias funcionales

- [[specs/SPEC-SESIONES]]
- [[specs/SPEC-NODO-CONTEXTO]]

## Referencias

- [[IDEA]]
