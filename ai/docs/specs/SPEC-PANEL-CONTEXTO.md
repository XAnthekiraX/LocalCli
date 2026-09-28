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
2. Mientras responde, se muestra un **indicador en vivo** (`[⠋ Pensando]`); el texto del razonamiento se revela con `Ctrl+R` a medida que llega.
3. Mientras responde y al terminar, se muestran los tokens del turno (el consumo vivo se corrige con el total exacto al cerrarse) y qué parte del contexto quedó ocupada.

## Flujos alternativos

- El modelo no expone el conteo de tokens: se muestra una estimación marcada como estimada.
- El modelo no expone razonamiento: se indica que no está disponible.
- El contexto se acerca al límite: se avisa antes de que la petición falle.

## Reglas de negocio

- El conteo de tokens se muestra siempre: en el panel de datos y, cuando hay consumo, también bajo la línea de entrada (`tokens: 54k`). Si es una estimación, aparece marcado como estimación.
- El razonamiento se revela tal como lo devuelve el modelo, sin editarlo (`Ctrl+R`); por defecto se ve el indicador en vivo, no el texto.
- La ocupación del contexto se expresa sobre el límite del modelo.
- Los datos corresponden a la petición actual, no a la suma de la sesión.
- Los datos se pueden ocultar sin detener nada.

## Criterios de aceptación

- [ ] Se ve el indicador en vivo mientras el modelo genera, y `Ctrl+R` revela el texto del razonamiento.
- [ ] Se ve cuántos tokens se usaron en la petición, bajo la línea de entrada (`tokens: 54k`).
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
