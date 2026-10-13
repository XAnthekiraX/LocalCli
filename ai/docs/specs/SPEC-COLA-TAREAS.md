---
title: SPEC — Cola de tareas
tags: [specs, requisito]
depende_de:
  - "[[IDEA]]"
  - "[[specs/SPEC-CICLO-TRABAJO]]"
  - "[[specs/SPEC-ORQUESTADOR-FLUJOS]]"
  - "[[specs/SPEC-SESIONES]]"
  - "[[specs/SPEC-ARCHIVOS]]"
  - "[[specs/SPEC-NODO-CONTEXTO]]"
relacionado:
  - "[[backend/DECISIONS]]"
---
# SPEC — Cola de tareas

Prioridad: P0 (núcleo)

## Propósito

Convertir las tareas grandes del proyecto en una cola que se ejecuta en orden cuando tú la lanzas, sin lanzar cada tanda por separado.

## Alcance

Incluye poblar la cola, ordenarla por dependencias, consumirla, manejar bloqueos y re-derivarla cuando las tareas cambian.
No incluye qué tarea se crea ni qué documentación se escribe, que es de [[specs/SPEC-CICLO-TRABAJO]], ni las reglas de permiso, que están en [[specs/SPEC-ARCHIVOS]].

## Actores

- **Usuario**: pausa, reanuda o cancela una tarea.
- **Sistema**: mantiene la cola y su orden.
- **Agente `plan`**: produce el TODO con los elementos que alimentan la cola.
- **Agente `build`**: consume la cola.

## Qué es un elemento del TODO

Una unidad de trabajo del TODO: una fase, una tarea o un paso. Se identifica, declara sus dependencias y declara su acción.

El elemento se identifica por su ID y vive en la capa de su archivo de tarea, de modo que dos capas no se mezclan.

## La cola es la de cada capa

La cola es **por capa**: cada capa tiene la suya y se deriva de sus archivos de tarea —el `MAIN-TASKS.md` y los `NNN-task-<nombre>.md`—, que son la fuente de verdad. No hay una cola global.

```
archivos de tarea  →  se derivan  →  cola de la capa  →  se consume en orden
```

No hay una cola esperando en la base: se reconstruye al arrancar y cuando un archivo de tarea cambia, de modo que lo pendiente es literalmente lo que dice el archivo.

Qué cuenta como petición ordenada:

- "Documentar capa por capa".
- "Ejecutar tarea 1, tarea 2, tarea 3".
- "Primero esto, luego esto".

Y en qué flujo se propone:

- En el **flujo de planificar**, se propone el TODO con todas las fases de la planificación.
- En el **flujo de ejecutar tareas**, se proponen los pasos para ejecutar cada tarea.

Una petición sin orden ni lista no propone nada: se responde en el chat, como cualquier otra.

**Por eso es por capa y no global:** dos capas avanzan en paralelo sin pisarse, y una capa que termina antes se detiene y avisa sin bloquear a la otra.

## Elementos de código y de documentación

Un elemento del TODO es de código o de documentación, y los dos se tratan igual: se ejecutan con el mismo criterio, pasan por las mismas reglas de permiso y se verifican igual.

Ejemplo: un elemento "implementar API de pedidos" puede necesitar elementos de documentación (definir el contrato, sus tipos) y de código (repositorio, servicio, pruebas).

## Flujo principal

1. Pides algo que implica una lista ordenada: "documentar capa por capa", "ejecutar tarea 1, tarea 2, tarea 3".
2. El sistema lo detecta y **propone** el TODO con los elementos que van a seguir, en orden.
3. La cola arranca cuando tú la lanzas: confirmas la propuesta o escribes el comando `/ejecutar`.
4. `build` toma el siguiente elemento disponible cuyas dependencias ya estén cumplidas.
5. Ejecuta **un elemento por iteración**: pide su contexto, propone el cambio, espera aprobación, aplica, verifica y lo marca completado.
6. Toca el siguiente.
7. Al vaciarse la cola, se detiene y avisa.

## Flujos alternativos

- Un elemento queda bloqueado: se marca y se informa qué falta. Si nada depende de él, la cola sigue con el siguiente.
- Aparece un elemento nuevo: la cola se re-deriva y el nuevo entra en su posición.
- El usuario cancela un elemento en curso: se detiene y lo que estaba en curso no se aplica.
- El usuario pausa la cola: se detiene después del elemento actual.
- Un elemento ya estaba empezado: se retoma sin repetir lo ya completado.

## Reglas de negocio

- La cola es por capa: cada capa tiene la suya, y ninguna es global.
- La cola se deriva de los archivos de tarea de su capa: no hay una representación paralela que pueda quedar vieja.
- Al detectar una petición ordenada, el sistema propone el TODO; el usuario decide si se ejecuta (confirmación o `/ejecutar`).
- Ningún texto arranca la cola por sí solo: sin comando o confirmación, la petición se responde en el chat.
- Dos capas tienen cada una su cola, y una no ve el estado de la otra.
- El orden del TODO es el orden de ejecución, y se respeta.
- Se ejecuta un elemento por iteración, nunca varios a la vez.
- Los elementos de documentación y de código siguen las mismas reglas de permiso.
- Nada se aplica sin aprobación, en ninguna de las dos clases de elemento.
- Un elemento bloqueado no detiene la cola si nada depende de él.
- La cola refleja siempre el estado real del TODO.
- El estado de la cola es visible desde cualquier sesión.
- Al vaciarse, la cola se detiene sola y avisa.

## Criterios de aceptación

- [ ] Una petición ordenada propone un TODO; no lo ejecuta por sí sola.
- [ ] "Documentar capa por capa", "ejecutar tarea 1, tarea 2, tarea 3" y "primero esto, luego esto" proponen un TODO.
- [ ] La cola solo arranca con la confirmación del usuario o el comando `/flow:ejecutar` —que el agente también puede lanzar con la herramienta `ejecutar_flujo`—.
- [ ] En el flujo de planificar, el TODO contiene todas las fases de la planificación.
- [ ] En el flujo de ejecutar tareas, el TODO contiene los pasos de cada tarea.
- [ ] Una petición sin orden no crea TODO y se responde en el chat.
- [ ] Los elementos se toman en el orden que marcan sus dependencias.
- [ ] `build` ejecuta un elemento del TODO por iteración.
- [ ] Los elementos de documentación y de código se ejecutan con el mismo criterio.
- [ ] Cada elemento pide aprobación antes de aplicarse.
- [ ] Un elemento bloqueado no detiene la cola si nada depende de él.
- [ ] Un elemento nuevo añadido aparece en la cola en su posición correcta.
- [ ] El usuario puede pausar y reanudar la cola.
- [ ] El usuario puede cancelar un elemento en curso.
- [ ] Al vaciarse la cola, se detiene y avisa.
- [ ] El estado de la cola se ve desde cualquier sesión.
- [ ] Retomar no repite elementos ya completados.
- [ ] Cada capa tiene su propia cola y el estado de una no se mezcla con el de la otra.

## Requisitos no funcionales

- Reanudar no duplica trabajo: los elementos completados no se vuelven a ejecutar.
- La cola de una capa no bloquea a la de otra.

## Dependencias funcionales

- [[specs/SPEC-CICLO-TRABAJO]]
- [[specs/SPEC-ORQUESTADOR-FLUJOS]]
- [[specs/SPEC-SESIONES]]
- [[specs/SPEC-ARCHIVOS]]
- [[specs/SPEC-NODO-CONTEXTO]]

## Supuestos

- Cómo se declara un elemento de documentación frente a uno de código se fija en FASE 2 y FASE 3.
- El formato exacto del frontmatter de un elemento del TODO sigue por decidir. Ver [[backend/DECISIONS]].

## Referencias

- [[IDEA]]
