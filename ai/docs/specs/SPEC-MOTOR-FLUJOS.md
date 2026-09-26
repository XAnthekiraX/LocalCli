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

Es el flujo por defecto. Encadena varias tareas grandes de la misma capa sin que el usuario las pida una por una.

1. La cola de la capa se puebla con las tareas grandes disponibles.
2. El motor toma la siguiente tarea cuyas dependencias estén cumplidas.
3. La ejecuta como una secuencia de etapas.
4. Al terminar, marca la tarea y toma la siguiente.
5. Al vaciarse la cola, se detiene y avisa.

## Flujo principal — flujo con objetivo

1. El usuario lanza un flujo con un objetivo.
2. El motor lo descompone en sus etapas.
3. Cada etapa pide su contexto al nodo de contexto.
4. La etapa produce un resultado.
5. El resultado pasa a la etapa siguiente.
6. El flujo termina y se muestra el resultado final.

## Flujos alternativos

- Una etapa falla: el flujo se detiene y el usuario decide.
- Una etapa necesita permiso: la etapa espera y el flujo queda pausado.
- El usuario cancela: se detiene en la etapa actual.
- Una etapa no puede continuar por falta de contexto: avisa y se detiene.
- El usuario vuelve más tarde: el flujo pausado se retoma donde estaba.

## Reglas de negocio

- Las etapas se ejecutan en orden y cada una arranca cuando la anterior terminó.
- Cada etapa recibe solo el contexto que necesita, no todo lo que la etapa anterior produjo.
- Si una etapa falla, el flujo se detiene. El usuario elige reintentar, saltar esa etapa o cancelar.
- Un flujo pausado por un permiso se retoma desde la misma etapa, sin repetir lo ya hecho.
- Todo lo que hace cada etapa queda registrado: qué recibió, qué hizo y qué produjo.
- Los flujos oficiales vienen con la herramienta y funcionan sin configuración.
- Un flujo que se cancela no deja etapas ejecutándose.
- El motor ejecuta la cola por su cuenta. El usuario no lanza tarea por tarea.
- Una tarea de la cola no arranca si sus dependencias no están cumplidas.

## Criterios de aceptación

- [ ] Lanzar un flujo oficial ejecuta sus etapas en orden.
- [ ] El motor toma la siguiente tarea de la cola sin que se la pidan.
- [ ] Una tarea con dependencias sin cumplir no arranca.
- [ ] Cada etapa recibe solo el contexto que necesita.
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
