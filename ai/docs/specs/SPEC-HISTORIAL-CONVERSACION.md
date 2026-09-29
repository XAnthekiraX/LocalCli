---
title: SPEC — Historial de conversación
tags: [specs, requisito]
depende_de:
  - "[[specs/SPEC-SESIONES]]"
  - "[[specs/SPEC-NODO-CONTEXTO]]"
---
# SPEC — Historial de conversación

Prioridad: P0 (núcleo)

## Propósito

Que el chat recuerde: cada petición se responde con la conversación anterior de la sesión, no solo con el mensaje nuevo, y esa conversación se compacta cuando no cabe en el presupuesto de contexto del modelo.

## Alcance

Incluye cómo se reconstruye el historial de una sesión para el modelo, el presupuesto de tokens y la compactación por resumen.
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

## Criterios de aceptación

- [ ] El modelo recibe la conversación anterior de la sesión, no solo el último mensaje.
- [ ] Dos sesiones del mismo proyecto no comparten historial.
- [ ] Un historial corto se entrega entero.
- [ ] Un historial largo se entrega como resumen + tramo reciente, dentro del presupuesto.
- [ ] El resumen se reutiliza entre turnos mientras el tramo reciente siga cabiendo.
- [ ] Un fallo del resumen no impide responder.

## Requisitos no funcionales

- Reconstruir el historial no bloquea la interfaz: ocurre en la goroutine de la sesión.
- No se añaden tablas: el historial sale de los mensajes ya persistidos.

## Dependencias funcionales

- [[specs/SPEC-SESIONES]]
- [[specs/SPEC-NODO-CONTEXTO]]

## Referencias

- [[IDEA]]
