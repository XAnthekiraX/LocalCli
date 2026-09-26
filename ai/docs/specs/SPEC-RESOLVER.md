---
title: SPEC — Resolver problemas
tags: [specs, requisito]
depende_de:
  - "[[IDEA]]"
  - "[[specs/SPEC-AGENTE-BASE]]"
  - "[[specs/SPEC-TOOLS]]"
  - "[[specs/SPEC-ARCHIVOS]]"
  - "[[specs/SPEC-CICLO-TRABAJO]]"
  - "[[specs/SPEC-NODO-CONTEXTO]]"
---
# SPEC — Resolver problemas

Prioridad: P1 (importante)

## Propósito

Resolver un problema detectado durante el desarrollo, sin que eso se confunda con un cambio de funcionalidad.

## Alcance

Incluye analizar la causa, proponer una solución, ejecutarla y verificar que quedó resuelta.
Es un ciclo aparte: no amplía el alcance del proyecto ni crea tareas por su cuenta. Eso lo hace [[specs/SPEC-CICLO-TRABAJO]].

## Actores

- **Usuario**: reporta el problema y aprueba la solución.
- **Agente `plan`**: investiga, propone la solución y pide aprobación. No escribe.
- **Agente `build`**: aplica la solución aprobada y verifica.
- **Sistema**: aporta el contexto y ejecuta la verificación.

## Qué lo distingue de un cambio de funcionalidad

- Un cambio de funcionalidad amplía lo que el proyecto puede hacer. Va por el ciclo de trabajo.
- Un problema es algo que ya debía funcionar y no funciona. Se arregla en su lugar.
- **Si al arreglar un problema se descubre que hace falta una funcionalidad nueva, el problema se resuelve primero y la funcionalidad se pide aparte por el ciclo de trabajo.**

## Flujo principal

1. El usuario describe el problema, o pega el error.
2. `build` captura evidencia: logs, mensajes de error, fallos de verificación.
3. Localiza el código relacionado.
4. Identifica la causa raíz. No propone soluciones sin evidencia.
5. Presenta la propuesta: qué archivos cambian y cómo.
6. Espera aprobación.
7. Cambias a `build`, que aplica la solución.
8. Verifica que el problema quedó resuelto ejecutando la comprobación que aplique.

## Flujos alternativos

- El error no viene del código: se explica y se propone la corrección fuera del proyecto.
- El problema es ambiguo: se pregunta antes de proponer.
- La causa no se encuentra: se informa qué se descartó y qué falta, sin inventar un arreglo.
- La solución cambia el comportamiento del sistema: se avisa y pasa por el ciclo de trabajo antes de aplicarse.
- El arreglo toca documentación: se actualiza después de aprobar, como cualquier otro cambio.

## Reglas de negocio

- Nunca se propone una solución sin evidencia que la respalde.
- La propuesta se muestra antes de aplicarse, siempre.
- La solución la aplica `build` después del relevo, no `plan`.
- Después de aplicar, se verifica que el problema quedó resuelto ejecutando pruebas, linter, tipos o build. Sin verificación, la tarea no se cierra.
- Un arreglo nunca amplía el alcance del proyecto por su cuenta.
- Un arreglo nunca se aplica sin pasar por las reglas de permiso.
- No se ejecutan cambios destructivos sin confirmación explícita.
- El arreglo queda registrado como parte del historial de la tarea afectada.

## Criterios de aceptación

- [ ] El problema se investiga con evidencia, no con suposiciones.
- [ ] La propuesta indica qué archivos cambian y cómo, antes de aplicarse.
- [ ] `plan` propone y `build` aplica, con el relevo de por medio.
- [ ] Nada se aplica sin aprobación.
- [ ] Tras aplicar, se ejecuta una verificación y se informa su resultado.
- [ ] Si la causa no se encuentra, se informa en vez de inventar un arreglo.
- [ ] Si el arreglo requiere una funcionalidad nueva, se deriva al ciclo de trabajo.
- [ ] Si el arreglo cambia el comportamiento del sistema, se avisa antes de aplicarlo.

## Requisitos no funcionales

- La verificación ocurre siempre, también cuando el arreglo parece obvio.

## Dependencias funcionales

- [[specs/SPEC-AGENTE-BASE]]
- [[specs/SPEC-TOOLS]]
- [[specs/SPEC-ARCHIVOS]]
- [[specs/SPEC-CICLO-TRABAJO]]
- [[specs/SPEC-NODO-CONTEXTO]]

## Supuestos

- Qué comprobaciones concretas se ejecutan para verificar (pruebas, linter, tipos, build) se fija en FASE 2 y FASE 3.

## Referencias

- [[IDEA]]
