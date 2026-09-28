---
title: SPEC — Flujo personalizado
tags: [specs, requisito]
depende_de:
  - "[[IDEA]]"
  - "[[specs/SPEC-MOTOR-FLUJOS]]"
  - "[[specs/SPEC-NODO-CONTEXTO]]"
---
# SPEC — Flujo personalizado

Prioridad: P3 (personalización, diferible)

## Propósito

Poder definir flujos propios con las etapas que uno necesite.

## Alcance

Incluye definir, guardar, editar, borrar y ejecutar flujos propios.
No incluye los flujos oficiales, que están en [[specs/SPEC-MOTOR-FLUJOS]], ni los agentes propios.

## Actores

- **Usuario**: define y lanza el flujo.
- **Motor**: ejecuta las etapas.

## Flujo principal

1. El usuario define un flujo con nombre, etapas y el orden en que se ejecutan.
2. Cada etapa indica qué necesita y qué produce.
3. Lo guarda.
4. Lo lanza con un objetivo.

## Flujos alternativos

- Editar un flujo existente.
- Borrarlo.
- Una etapa que depende de algo que no existe.
- Un flujo con una etapa que se repite en bucle.

## Reglas de negocio

- Un flujo propio no puede saltarse el nodo de contexto ni las reglas de permiso.
- Las etapas de un flujo propio reciben contexto igual que las de un flujo oficial.
- Un flujo con un bucle se detiene y avisa; no se repite.
- Los flujos propios solo se ven dentro de su proyecto.
- Un flujo propio se detiene y pregunta ante un fallo, igual que uno oficial.

## Criterios de aceptación

- [ ] Se puede crear un flujo con varias etapas en orden.
- [ ] Se puede lanzar un flujo propio con un objetivo.
- [ ] Las etapas de un flujo propio reciben contexto mediante el nodo de contexto.
- [ ] Un flujo propio pide aprobación igual que uno oficial.
- [ ] Se puede editar y borrar un flujo propio.
- [ ] Un flujo con un bucle se detiene y avisa.

## Requisitos no funcionales

- Un flujo propio no puede degradar el consumo de contexto frente a un flujo oficial.

## Dependencias funcionales

- [[specs/SPEC-MOTOR-FLUJOS]]
- [[specs/SPEC-NODO-CONTEXTO]]

## Supuestos

- El formato de definición está fijado: un JSON por flujo en `ai/flows/`, con `comando`, `nombre`, `descripcion`, `peticion`, `reglas` y `etapas`. Cada etapa lleva `id`, `nombre`, `agente` (`plan` o `build`), `aprobacion` e `instruccion`.
- Los flujos oficiales vienen con la herramienta y funcionan sin configuración. Un JSON del proyecto con el mismo `comando` personaliza el oficial; con un `comando` nuevo añade un flujo propio.
- Las `reglas` del flujo y la `instruccion` de cada etapa viajan con el contexto de la etapa: es donde se declara, por ejemplo, cómo descubrir la documentación.
- Los flujos se cargan al arrancar. Un JSON roto no tumba el arranque: se usan los oficiales y se avisa.

## Referencias

- [[IDEA]]
