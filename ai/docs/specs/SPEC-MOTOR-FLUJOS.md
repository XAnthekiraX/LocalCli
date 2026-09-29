---
title: SPEC — Motor de flujos
tags: [specs, requisito]
depende_de:
  - "[[IDEA]]"
  - "[[specs/SPEC-NODO-CONTEXTO]]"
  - "[[specs/SPEC-SESIONES]]"
  - "[[specs/SPEC-AGENTE-BASE]]"
  - "[[specs/SPEC-COLA-TAREAS]]"
relacionado:
  - "[[specs/SPEC-FLUJO-PERSONALIZADO]]"
---
# SPEC — Motor de flujos

Prioridad: P0 (núcleo)

## Propósito

Ejecutar trabajos de varias etapas encadenadas, entregando a cada etapa solo el contexto que necesita.

## Alcance

Incluye el motor de etapas, los flujos oficiales, el encadenamiento, el control cuando una etapa falla y el registro de lo que hizo cada etapa.
No incluye la cola que el motor ejecuta, que está en [[specs/SPEC-COLA-TAREAS]], ni que el usuario defina sus propios flujos, que está en [[specs/SPEC-FLUJO-PERSONALIZADO]].

## Actores

- **Usuario**: lanza el flujo y decide cuando algo falla.
- **Motor**: encadena las etapas y conserva el estado.
- **Etapa**: pide su contexto, trabaja y produce un resultado.

## Flujo principal — ejecutar cola

Se arranca con el comando explícito `/ejecutar`. Encadena las tareas grandes disponibles de la capa, una tras otra.

1. El usuario escribe `/ejecutar`.
2. La cola de la capa se puebla con las tareas grandes disponibles.
3. El motor toma la siguiente tarea cuyas dependencias estén cumplidas.
4. La ejecuta como una secuencia de etapas.
5. Al terminar, marca la tarea y toma la siguiente.
6. Al vaciarse la cola, se detiene y avisa.

## Flujo principal — flujo con objetivo

1. El usuario escribe el comando que nombra el flujo con un objetivo.
2. El motor lo descompone en sus etapas.
3. Cada etapa pide su contexto al nodo de contexto, identificándose con su etapa: es lo que hace auditable qué recibió cada una (`context_audit`, una fila por documento y etapa).
4. La etapa produce su resultado y lo deja como un **resumen corto**.
5. El resumen pasa a la etapa siguiente, que lo recibe además de su propio contexto. No se arrastra la salida completa: cada etapa recibe solo lo que necesita.
6. El flujo termina y el chat muestra el resultado final: la entrega del último paso.

Un flujo con `bloque_contexto` (hoy solo el resolver) cambia los pasos 4-6: cada etapa guarda su resultado en un **bloque de contexto** persistido, el modelo lo **optimiza** al cerrar la etapa y la última etapa **compone** la entrega a partir de todo el bloque, sin herramientas. Ver §Bloque de contexto.

### Bloque de contexto

Un flujo declara `bloque_contexto: true` cuando necesita que el trabajo de sus etapas no se pierda por el camino: el resolver lo hace para entregar el PLAN.

1. Cada etapa intermedia entrega su resultado y el modelo lo **optimiza** —una generación corta que lo condensa sin inventar— antes de guardarlo y de encadenarlo.
2. El bloque guarda una aportación por etapa en `flow_context` (sesión, flujo, etapa, orden y contenido). Una ejecución nueva del mismo flujo lo vacía: no hereda la anterior.
3. Las etapas siguientes reciben las aportaciones ya optimizadas.
4. La **última etapa es la composición**: recibe el bloque entero y corre **sin herramientas**, solo para redactar la entrega. Así el cierre es determinista y no depende de que el modelo deje de pedir herramientas.
5. La vista no cambia: sigue anunciando `[Sub Proceso] <nombre>` y las líneas de herramienta; solo el texto de las fases intermedias queda oculto.

## Flujos alternativos

- Una etapa falla: el flujo se detiene y el usuario decide.
- Una etapa necesita permiso: la etapa espera y el flujo queda pausado.
- El usuario cancela: se detiene en la etapa actual.
- Una etapa no puede continuar por falta de contexto: avisa y se detiene.
- El usuario vuelve más tarde: el flujo pausado se retoma donde estaba.

## Reglas de negocio

- Las etapas se ejecutan en orden y cada una arranca cuando la anterior terminó.
- Cada etapa recibe solo el contexto que necesita, no todo lo que la etapa anterior produjo.
- Cada etapa es un sub-proceso: corre sin el historial del chat y recibe de las anteriores solo sus resúmenes cortos, encadenados.
- Una etapa intermedia que no pide aprobación corre en silencio: su texto no se muestra en el chat ni se persiste; la vista solo anuncia su nombre (`[Sub Proceso] <nombre>`). La última etapa y las que piden aprobación sí se muestran.
- En un flujo normal, el estado de la cadena —los resúmenes de las etapas— vive en el motor. En un flujo con `bloque_contexto` se persiste en `flow_context` (una aportación por etapa y sesión) y la composición final lo lee entero; al terminar, el chat muestra la entrega del último paso.
- Si una etapa falla, el flujo se detiene. El usuario elige reintentar, saltar esa etapa o cancelar.
- Un flujo pausado por un permiso se retoma desde la misma etapa, sin repetir lo ya hecho.
- Todo lo que hace cada etapa queda registrado: qué recibió, qué hizo y qué produjo.
- Los flujos oficiales vienen con la herramienta y funcionan sin configuración. Los flujos propios y las personalizaciones de los oficiales se declaran en `ai/flows/*.json` y se cargan al arrancar: un JSON con el mismo `comando` reemplaza al oficial y uno con un `comando` nuevo añade un flujo ([[specs/SPEC-FLUJO-PERSONALIZADO]]).
- Un flujo que se cancela no deja etapas ejecutándose.
- Un flujo no arranca solo: lo solicita el usuario con un comando explícito (`/planificar`, `/crear`, `/actualizar`, `/eliminar`, `/resolver` o `/ejecutar`).
- Una petición que no es un comando de flujo se responde en el chat, no arranca etapas.
- El motor puede detectar trabajo ordenado y proponer un TODO, pero no lo ejecuta hasta que el usuario confirme o escriba el comando.
- Una tarea de la cola no arranca si sus dependencias no están cumplidas.

## Criterios de aceptación

- [ ] Lanzar un flujo oficial ejecuta sus etapas en orden.
- [ ] Un flujo solo arranca con el comando explícito del usuario.
- [ ] La cola avanza cuando el usuario escribe `/ejecutar`.
- [ ] Una petición sin comando se responde en el chat y no arranca etapas.
- [ ] Una tarea con dependencias sin cumplir no arranca.
- [ ] Cada etapa recibe solo el contexto que necesita.
- [ ] El resultado de cada etapa se pasa a la siguiente como un resumen corto.
- [ ] Un flujo con `bloque_contexto` guarda una aportación optimizada por etapa en `flow_context`.
- [ ] La última etapa de un flujo con `bloque_contexto` compone la entrega sin herramientas, a partir del bloque entero.
- [ ] Una etapa intermedia sin aprobación no se muestra ni se persiste; la vista solo anuncia su nombre.
- [ ] Al terminar el flujo, el chat muestra la entrega del último paso.
- [ ] Si una etapa falla, el flujo se detiene y el usuario elige qué hacer.
- [ ] Un flujo pausado por un permiso se retoma exactamente donde estaba.
- [ ] El usuario puede cancelar un flujo en cualquier momento.
- [ ] Queda registro de lo que hizo cada etapa.
- [ ] Un flujo oficial se puede lanzar sin configurar nada.

## Requisitos no funcionales

- Aislamiento: un flujo no recibe el contexto de otro flujo.
- Recuperación: retomar un flujo pausado no repite etapas ya completadas.

## Dependencias funcionales

- [[specs/SPEC-NODO-CONTEXTO]]
- [[specs/SPEC-SESIONES]]
- [[specs/SPEC-AGENTE-BASE]]
- [[specs/SPEC-COLA-TAREAS]]

## Referencias

- [[IDEA]]
