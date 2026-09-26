---
title: SPEC — Agentes incluidos
tags: [specs, requisito]
depende_de:
  - "[[IDEA]]"
  - "[[specs/SPEC-TOOLS]]"
  - "[[specs/SPEC-NODO-CONTEXTO]]"
  - "[[specs/SPEC-ARCHIVOS]]"
  - "[[specs/SPEC-OLLAMA-PERFIL]]"
  - "[[specs/SPEC-SESIONES]]"
  - "[[specs/SPEC-COLA-TAREAS]]"
relacionado:
  - "[[backend/DECISIONS]]"
  - "[[specs/SPEC-CICLO-TRABAJO]]"
  - "[[specs/SPEC-RESOLVER]]"
  - "[[specs/SPEC-SKILLS]]"
---
# SPEC — Agentes incluidos

Prioridad: P0 (núcleo)

## Propósito

Los dos agentes incluidos, con permisos distintos: uno que solo mira y propone, otro que escribe.

**Vienen incluidos, pero no están hardcodeados.** Los dos se definen en archivos JSON con estructura fija, en `ai/agents/`. El usuario puede modificarlos o derivar otros agentes de ellos sin tocar el código.

**No hay skills por defecto.** Los agentes vienen sin ninguna; las crea el usuario en markdown. Ver [[specs/SPEC-SKILLS]].

## Alcance

Incluye `plan` y `build`: qué hace cada uno, a qué herramientas tiene acceso y cómo se pasa de uno a otro.
No incluye agentes propios, skills ni flujos.

## Actores

- **Usuario**: elige agente, responde preguntas, aprueba propuestas.
- **Agente `plan`**: investiga, propone y pide aprobación. No escribe.
- **Agente `build`**: aplica lo aprobado. Es el único que escribe.

## Dónde se definen

Cada agente es un `ai/agents/*.json` con campos fijos: `nombre`, `descripcion`, `prompt`, `herramientas` y `skills`. Se eligió JSON y no un README porque un README deja margen a interpretación de qué significa cada parte.

El campo `herramientas` es el que sostiene la garantía de permisos: si `plan` no lista una herramienta de escritura, no la tiene, aunque se la pidan. La garantía vive en los datos, no en el código.

El formato exacto de los campos sigue por confirmar. Ver [[backend/DECISIONS]].

## Los dos agentes

### `plan` — solo lee

Tiene **solo herramientas que recopilan información**. No tiene ninguna forma de escribir, y no la necesita: existe para entender el proyecto y decidir qué hay que hacer.

- Lee archivos, lista carpetas y busca por contenido o por nombre.
- Ejecuta comandos de consulta: compilar, correr pruebas, revisar estilo, ver el estado del repositorio.
- Busca en internet y abre páginas.
- Pregunta al usuario lo que no sabe y propone ideas en vez de suponer.
- Nunca escribe. Su salida es una propuesta, no un cambio.

Su trabajo termina en una propuesta aprobada.

### `build` — escribe

Tiene **el catálogo completo**. Es el único que crea, modifica y borra.

- Aplica exactamente lo que `plan` propuso y tú aprobaste.
- Toma el siguiente elemento del TODO de la ejecución en curso, sin que se lo pidan.
- Trata **un elemento del TODO por iteración**, sin importar si es de código o de documentación.
- La documentación es su única fuente de verdad.
- Si falta información, marca la tarea como bloqueada y dice exactamente qué falta. No rellena huecos inventando.
- Verifica el resultado antes de dar la tarea por terminada.
- Al terminar la tarea, toma la siguiente de la cola.

## El relevo

El paso de `plan` a `build` es explícito y es el mecanismo central de la herramienta.

```
plan lee  →  plan propone  →  tú apruebas  →  cambias a build  →  build aplica
```

1. `plan` investiga con herramientas de lectura.
2. `plan` propone el cambio concreto y pide aprobación.
3. Tú apruebas.
4. Cambias a `build`.
5. `build` aplica exactamente lo aprobado.

**Una aprobación vale para el cambio propuesto, no para lo que siga.** Si `build` necesita hacer algo que `plan` no propuso, vuelve a preguntar. Una aprobación no es un permiso general.

Por eso `plan` no escribe: nada cambia en el proyecto sin que antes alguien lo propusiera y tú lo aceptaras.

## Flujo principal

1. El usuario abre la herramienta en una carpeta.
2. Elige agente: `plan` para decidir, `build` para aplicar.
3. Escribe su petición.
4. El agente recibe el contexto que entrega el nodo de contexto.
5. `plan` investiga, propone y pide aprobación.
6. El usuario cambia a `build`.
7. `build` aplica el cambio.
8. `build` verifica y continúa con la cola.

## Flujos alternativos

- Elige `build` sin documentación suficiente: se lo dice y ofrece cambiar a `plan`.
- Elige `plan` en un proyecto ya documentado: pregunta qué hay que cambiar.
- `plan` termina sin proponer cambios: la tarea se cierra sin cambios.
- `build` se queda sin contexto: se lo dice y vuelve a pedir contexto.
- Necesita salir de la carpeta del proyecto → se aplica [[specs/SPEC-ARCHIVOS]].
- Necesita cambiar, actualizar o eliminar una funcionalidad → se aplica [[specs/SPEC-CICLO-TRABAJO]].
- Tiene que resolver algo que está roto → se aplica [[specs/SPEC-RESOLVER]].

## Reglas de negocio

- Ambos agentes vienen incluidos y están siempre disponibles.
- `plan` no tiene herramientas de escritura. `build` no cambia decisiones técnicas.
- El relevo entre agentes es explícito: propuesta, aprobación, cambio, aplicación.
- Una aprobación habilita solo el cambio propuesto.
- Ambos responden en el idioma en que el usuario escribió.
- Ninguno busca archivos por su cuenta: esperan el contexto que entrega el nodo de contexto.
- Si el contexto no alcanza, lo dicen y piden más. No rellenan los huecos con suposiciones.
- Ninguna escritura pasa sin que `plan` la haya propuesto y tú la hayas aprobado.
- `build` no inventa endpoints, entidades, reglas de negocio ni relaciones.
- Un intercambio de chat empieza con el contexto de esa sesión, no con el de otra.
- `plan` alimenta la cola; `build` la drena. Ninguno espera a que se le pida el siguiente trabajo.

## Criterios de aceptación

- [ ] Al abrir la herramienta sin configurar nada, los dos agentes están disponibles.
- [ ] `plan` no tiene ninguna herramienta que cree, modifique o borre.
- [ ] `plan` produce una propuesta, nunca un cambio.
- [ ] `build` tiene el catálogo completo y aplica lo aprobado.
- [ ] Nada se escribe sin que `plan` lo haya propuesto y tú lo hayas aprobado.
- [ ] Una aprobación no habilita cambios distintos de los propuestos.
- [ ] `build` implementa y nunca inventa requisitos.
- [ ] Ambos responden en el idioma en que el usuario escribió.
- [ ] Si le falta información del contexto, la pide en vez de inventarla.
- [ ] `build` marca la tarea como bloqueada y dice qué falta cuando no puede continuar.
- [ ] `build` verifica el resultado antes de cerrar la tarea.
- [ ] `build` toma la siguiente tarea de la cola sin que se lo pidan.
- [ ] El contenido de una sesión no aparece en otra sesión del mismo proyecto.

## Requisitos no funcionales

- Ambos operan solo con el contexto que reciben, no con el proyecto completo.
- No añaden esperas propias al tiempo de respuesta del modelo.

## Dependencias funcionales

- [[specs/SPEC-TOOLS]]
- [[specs/SPEC-NODO-CONTEXTO]]
- [[specs/SPEC-ARCHIVOS]]
- [[specs/SPEC-OLLAMA-PERFIL]]
- [[specs/SPEC-SESIONES]]
- [[specs/SPEC-COLA-TAREAS]]

## Supuestos

- Las instrucciones concretas de cada agente (tono, formato de respuesta, longitud) se ajustan después de verlos en uso.

## Referencias

- [[IDEA]]
