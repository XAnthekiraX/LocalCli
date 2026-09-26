---
title: SPEC — Panel de contexto y tokens
tags: [specs, requisito]
depende_de:
  - "[[IDEA]]"
  - "[[specs/SPEC-INTERFAZ]]"
  - "[[specs/SPEC-OLLAMA-PERFIL]]"
  - "[[specs/SPEC-NODO-CONTEXTO]]"
relacionado:
  - "[[specs/SPEC-INTERFAZ-ATAJOS]]"
---
# SPEC — Panel de contexto y tokens

Prioridad: P1 (importante)

## Propósito

Ver lo que está pasando por dentro: cuánto contexto se usó, cuánto queda libre y qué pensó el modelo.

## Alcance

Incluye el conteo de tokens, la ocupación del contexto y la visualización del razonamiento del modelo.
No incluye dónde se coloca cada dato en pantalla, que está en [[specs/SPEC-INTERFAZ]], ni el panel de aprobaciones, que está en [[specs/SPEC-INTERFAZ-ATAJOS]].

Aquí se define **qué significan** los números y de dónde salen. **Dónde se muestran** es responsabilidad de [[specs/SPEC-INTERFAZ]].

## Actores

- **Usuario**: observa.
- **Sistema**: mide y muestra.

## Flujo principal

1. Se envía una petición al modelo.
2. Mientras responde, se muestra su razonamiento a medida que llega.
3. Al terminar, se muestra cuántos tokens se usaron y qué parte del contexto quedó ocupada.

## Flujos alternativos

- El modelo no expone el conteo de tokens: se muestra una estimación marcada como estimada.
- El modelo no expone razonamiento: se indica que no está disponible.
- El contexto se acerca al límite: se avisa antes de que la petición falle.

## Reglas de negocio

- El conteo de tokens se muestra siempre. Si es una estimación, aparece marcado como estimación.
- El razonamiento se muestra tal como lo devuelve el modelo, sin editarlo.
- La ocupación del contexto se expresa sobre el límite del modelo.
- Los datos corresponden a la petición actual, no a la suma de la sesión.
- Los datos se pueden ocultar sin detener nada.

## Criterios de aceptación

- [ ] Se ve el razonamiento del modelo mientras genera la respuesta.
- [ ] Se ve cuántos tokens se usaron en la petición.
- [ ] Se ve qué porcentaje del contexto está ocupado.
- [ ] Si el conteo es estimado, aparece marcado como estimación.
- [ ] Si el modelo no entrega razonamiento, se indica que no está disponible.
- [ ] Se avisa cuando el contexto se acerca a su límite.

## Requisitos no funcionales

- Mostrar los datos no debe retrasar la respuesta del modelo.

## Dependencias funcionales

- [[specs/SPEC-INTERFAZ]]
- [[specs/SPEC-OLLAMA-PERFIL]]
- [[specs/SPEC-NODO-CONTEXTO]]

## Referencias

- [[IDEA]]
