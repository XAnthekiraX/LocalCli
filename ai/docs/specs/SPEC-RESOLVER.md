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

Investigar un problema detectado durante el desarrollo, diagnosticar su causa y entregar un **plan de solución**, sin confundirlo con un cambio de funcionalidad y **sin implementar nada**.

## Alcance

Incluye recibir la tarea, entender el problema, buscar contexto, investigar, diagnosticar, identificar los archivos afectados, diseñar la solución, armar el plan de ejecución y **entregar el PLAN**.
No implementa: no aplica cambios ni crea tareas; entregar el PLAN es el final del ciclo. Es un ciclo aparte: no amplía el alcance del proyecto. Eso lo hace [[specs/SPEC-CICLO-TRABAJO]].
Su flujo se declara en `.localcli/flows/resolver.json`; el orquestador lo carga al arrancar y el comando `/resolver` existe mientras ese archivo esté. Los flujos son dato: el orquestador no cablea ninguno ([[specs/SPEC-FLUJO-PERSONALIZADO]]).

Es un flujo con **bloque de contexto** ([[specs/SPEC-ORQUESTADOR-FLUJOS]] §Bloque de contexto: todos los flujos llevan bloque): cada paso deja su resultado optimizado en el bloque y el paso 9 compone el PLAN a partir de todo el bloque, sin herramientas. Así la entrega es determinista: el resolver cierra siempre con la salida estándar, no con un preámbulo del modelo.

## Actores

- **Usuario**: reporta el problema y recibe el PLAN.
- **Agente `plan`**: investiga, diagnostica y propone el plan. No escribe ni implementa.
- **Sistema**: aporta el contexto y ejecuta las pruebas que el agente pida.

## Qué lo distingue de un cambio de funcionalidad

- Un cambio de funcionalidad amplía lo que el proyecto puede hacer. Va por el ciclo de trabajo.
- Un problema es algo que ya debía funcionar y no funciona. El resolver lo diagnostica y propone cómo arreglarlo; no lo arregla.
- **Si al diagnosticar se descubre que hace falta una funcionalidad nueva, se dice en el PLAN y la funcionalidad se pide aparte por el ciclo de trabajo.**

## Flujo principal

El flujo se declara en `.localcli/flows/resolver.json` (ver [[specs/SPEC-FLUJO-PERSONALIZADO]]); así está declarado. Es un recorrido de nueve pasos, todos con `plan`:

1. **Recibir la tarea.**
2. **Entender el problema.** Qué se pidió y qué comportamiento se espera.
3. **Buscar contexto.** Documentación, estructura del código, archivos relacionados, tests y configuración. Solo lo necesario.
4. **Investigar el problema.** Seguir el flujo real, encontrar dónde ocurre, verificar hipótesis y ejecutar pruebas si hace falta.
5. **Diagnosticar.** La causa, el comportamiento actual y el comportamiento esperado.
6. **Identificar los archivos afectados**, en tres grupos: MODIFICAR, CONSULTAR y NO TOCAR, cada uno con su motivo.
7. **Diseñar la solución.** Qué cambiar, dónde cambiarlo y cómo se relacionan los cambios.
8. **Crear el plan de ejecución.** Pasos concretos, en orden, con su validación.
9. **Entregar el PLAN. No se implementa.** Es la composición: recibe el bloque con lo que dejaron los pasos 1-8 y lo redacta con la salida estándar, sin herramientas.

Todos los pasos declaran `respuesta_en_chat: true`: cada uno responde a su pregunta y esa respuesta se ve en la pantalla, se guarda y alimenta la ventana y el bloque que reciben los pasos siguientes. El paso 9 es la composición: recibe el bloque con lo que dejaron los pasos 1-8 y lo redacta con la salida estándar, sin herramientas. La vista anuncia cada paso (`[Sub Proceso] <nombre>`) y sus herramientas, para que el usuario vea que el sistema trabaja.

### La regla central

Antes de terminar, el resolver debe poder responder estas seis preguntas:

1. ¿Qué está pasando?
2. ¿Qué debería pasar?
3. ¿Dónde está el problema?
4. ¿Por qué está ocurriendo?
5. ¿Qué archivos están involucrados?
6. ¿Qué hay que hacer para solucionarlo?

### Salida estándar

Toda resolución cierra con esta estructura, para que sea reutilizable:

```
## Diagnóstico

[Qué ocurre y cuál es la causa.]

## Archivos involucrados

### Modificar
- `archivo` — motivo

### Consultar
- `archivo` — relación con el problema

### No modificar
- `archivo` — por qué no es necesario

## Solución

[Descripción concreta de la solución.]

## Plan

1. [Cambio]
2. [Cambio]
3. [Validación]

## Resultado esperado

[Cómo debe comportarse después del cambio.]
```

Las reglas del flujo —entre ellas, cómo descubrir la documentación y cerrar con la salida estándar— viajan con el contexto de cada etapa.

## Flujos alternativos

- El error no viene del código: se explica y se propone la corrección fuera del proyecto.
- El problema es ambiguo: se pregunta antes de proponer.
- La causa no se encuentra: se informa qué se descartó y qué falta, sin inventar un arreglo.
- El diagnóstico descubre que hace falta una funcionalidad nueva: se dice en el PLAN; la funcionalidad se pide aparte por el ciclo de trabajo.
- El PLAN cambia el comportamiento del sistema: se avisa en el plan; aplicarlo es cosa del ciclo de trabajo, no del resolver.

## Reglas de negocio

- El resolver **no implementa**: su salida es un PLAN.
- Nunca se afirma nada sin evidencia que lo respalde: se cita el archivo y la línea.
- Antes de terminar, se responden las seis preguntas.
- El PLAN cierra con la salida estándar (Diagnóstico; Archivos involucrados; Solución; Plan; Resultado esperado).
- Los archivos se clasifican en MODIFICAR, CONSULTAR y NO TOCAR, cada uno con su motivo.
- Un diagnóstico nunca amplía el alcance del proyecto por su cuenta.
- No se ejecutan cambios destructivos; el resolver no escribe.
- El PLAN queda registrado como parte del historial de la tarea afectada.

## Criterios de aceptación

- [ ] El problema se investiga con evidencia, no con suposiciones.
- [ ] El PLAN indica qué archivos se modificarían, cuáles se consultan y cuáles no se tocan, con su motivo.
- [ ] Antes de terminar quedan respondidas las seis preguntas.
- [ ] La salida cierra con la estructura estándar.
- [ ] El resolver no escribe ni implementa nada: entrega un PLAN.
- [ ] Si la causa no se encuentra, se informa en vez de inventar un arreglo.
- [ ] Si el arreglo requiere una funcionalidad nueva, se dice en el PLAN para el ciclo de trabajo.
- [ ] Si el arreglo cambia el comportamiento del sistema, se avisa en el PLAN.

## Requisitos no funcionales

- El diagnóstico se apoya en evidencia, también cuando el problema parece obvio.

## Dependencias funcionales

- [[specs/SPEC-AGENTE-BASE]]
- [[specs/SPEC-TOOLS]]
- [[specs/SPEC-ARCHIVOS]]
- [[specs/SPEC-CICLO-TRABAJO]]
- [[specs/SPEC-NODO-CONTEXTO]]

## Supuestos

- El formato del PLAN es la salida estándar de arriba; los detalles de redacción se ajustan en uso.

## Referencias

- [[IDEA]]
