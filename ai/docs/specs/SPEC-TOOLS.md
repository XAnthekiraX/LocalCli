---
title: SPEC — Catálogo de herramientas
tags: [specs, requisito]
depende_de:
  - "[[IDEA]]"
  - "[[specs/SPEC-AGENTE-BASE]]"
  - "[[specs/SPEC-ARCHIVOS]]"
  - "[[specs/SPEC-NODO-CONTEXTO]]"
  - "[[specs/SPEC-RESOLVER]]"
---
# SPEC — Catálogo de herramientas

Prioridad: P0 (núcleo)

## Propósito

Definir qué herramientas tiene el agente para trabajar, cuáles puede usar para mirar y cuáles para escribir, y en qué casos necesita permiso.

## Alcance

Incluye el catálogo completo de herramientas, el reparto por agente y las reglas de la terminal y de internet.
No incluye las reglas de permiso sobre archivos, que están en [[specs/SPEC-ARCHIVOS]], ni la selección de contexto, que está en [[specs/SPEC-NODO-CONTEXTO]].

## Actores

- **Usuario**: aprueba lo que lo requiere.
- **Agente `plan`**: solo mira, propone y pide aprobación.
- **Agente `build`**: escribe, modifica y borra.
- **Sistema**: ejecuta la herramienta y devuelve el resultado.

## Catálogo

### Archivos

| Herramienta | Lee o escribe | Agente |
|---|---|---|
| `leer_archivo` | Lee | Ambos |
| `listar_carpeta` | Lee | Ambos |
| `buscar_archivos` | Lee | Ambos |
| `buscar_en_archivos` | Lee | Ambos |
| `crear_archivo` | Escribe | Solo `build` |
| `escribir_archivo` | Escribe | Solo `build` |
| `editar_archivo` | Escribe | Solo `build` |
| `eliminar_archivo` | Escribe | Solo `build` |
| `crear_carpeta` | Escribe | Solo `build` |
| `eliminar_carpeta` | Escribe | Solo `build` |

### Terminal

| Herramienta | Lee o escribe | Agente |
|---|---|---|
| `ejecutar_comando` | Lee | Ambos |

### Internet

| Herramienta | Lee o escribe | Agente |
|---|---|---|
| `buscar_en_internet` | Lee | Ambos |
| `abrir_pagina` | Lee | Ambos |

## El reparto: `plan` mira, `build` escribe

`plan` solo recibe herramientas que **recopilan información**. No tiene ninguna que escriba. Existe para entender el proyecto, decidir qué hay que hacer y proponerlo.

`build` recibe el catálogo completo. Es el único que crea, modifica y borra.

Esto no es una restricción de estilo: es la garantía de que nada cambia en tu proyecto sin que `plan` lo haya propuesto antes y tú lo hayas aprobado.

## El relevo entre agentes

`plan` no escribe. Termina su trabajo proponiendo, y a partir de ahí el relevo es explícito.

```
plan lee  →  plan propone  →  tú apruebas  →  cambias a build  →  build aplica
```

1. `plan` investiga con herramientas de lectura.
2. `plan` propone el cambio concreto.
3. Tú lo apruebas.
4. Cambias a `build`.
5. `build` aplica exactamente lo aprobado.

**La aprobación vale para el cambio propuesto, no para lo que siga.** Si tras el relevo `build` necesita hacer algo que `plan` no propuso, vuelve a preguntar. Una aprobación no es un permiso general.

## Herramientas de archivo

Siguen las reglas de [[specs/SPEC-ARCHIVOS]]: dentro de la carpeta del proyecto son accesibles, fuera se pide permiso y el agente explica por qué, y **toda escritura pasa por aprobación**.

Una diferencia importante: `crear_archivo` falla si el archivo ya existe, y `escribir_archivo` lo sobrescribe. Están separadas a propósito. Así el agente no destruye algo por accidente cuando pretendía crear.

## Herramientas de terminal

Tiene **dos controles independientes**: una lista blanca que decide qué comandos existen, y un bloqueo que garantiza que no puede tocar tus archivos.

### Control 1 — lista blanca

Estos comandos se ejecutan sin preguntar, porque son los que se repiten en cada iteración:

- Compilar el proyecto.
- Correr las pruebas.
- Revisar estilo y tipos.
- Ver el estado y las diferencias del repositorio: estado, diferencias, historial.

Cualquier otro comando pide aprobación antes de ejecutarse.

### Control 2 — no toca tus archivos

El bloqueo no depende de revisar el texto del comando. **La terminal no puede crear, editar ni borrar ningún archivo tuyo**, por más indirecto que sea el comando. Es una garantía del entorno, no una convención.

Un filtro de texto no sirve: `find -delete`, `tee`, una tubería hacia un archivo, `python3 -c` o un `sh -c` con redirección se cuelan sin esfuerzo. Por eso el bloqueo es estructural.

Lo que sí puede hacer:

- Usar su propio espacio: caché de compilación, temporales y una carpeta de trabajo por sesión.
- Correr los comandos de la lista blanca.

Lo que no puede hacer, aunque se le pida:

- Crear, editar o borrar archivos del proyecto. Para eso están las herramientas de escritura, que solo tiene `build` y que sí pasan por aprobación.
- Descartar cambios del repositorio.
- Hacer commits, subir cambios o publicar.

### Otras reglas

- El comando corre en la carpeta del proyecto.
- La salida está limitada. Un comando que no termina o que genera salida sin fin no puede colgar el harness ni llenar el contexto. Si se corta, se avisa de que se cortó.
- El resultado indica si el comando terminó bien o mal, con la salida de error cuando la hay.
- Un comando que falla no detiene el trabajo: el agente ve el error y sigue.

## Herramientas de internet

Son las únicas que hacen salir información de la máquina.

### Qué sale

**Solo la consulta que redacta el modelo.** Nada más.

Nunca sale:

- El contenido de tus archivos.
- El contenido de la documentación del proyecto.
- El historial de la conversación.
- Los datos de tus tareas.

Si una búsqueda necesita el contenido del proyecto para ser útil, el agente te lo pide a ti en lugar de mandarlo.

### Qué vuelve

`buscar_en_internet` devuelve resultados: títulos, direcciones y fragmentos. `abrir_pagina` devuelve el contenido de una página.

Ese contenido entra en el contexto, **ocupa el mismo espacio que el resto** y está sujeto al mismo recorte y al mismo registro de auditoría. Ningún agente vierte el contenido de una página sin procesarlo en su respuesta.

### A quién tiene acceso

Las usan los dos. `plan` las necesita para consultar documentación de librerías mientras planifica. `build` las necesita para verificar versiones y APIs.

## Reglas de negocio

- El catálogo es cerrado: el agente no puede inventar herramientas que no estén aquí.
- `plan` solo tiene herramientas de lectura. No tiene ninguna forma de escribir.
- `build` tiene el catálogo completo.
- El relevo de `plan` a `build` es explícito: propuesta, aprobación, cambio de agente, aplicación.
- Una aprobación vale para el cambio propuesto. No habilita nada más.
- Las herramientas de terminal nunca se saltan las reglas de permiso.
- La terminal no puede modificar nada del proyecto, y esa garantía no depende de revisar el comando.
- La salida de un comando está limitada para no desbordar el contexto.
- Por internet, solo sale la consulta, nunca nada del proyecto.
- El contenido que vuelve de internet se recorta y se audita igual que el resto.

## Criterios de aceptación

- [ ] `plan` puede leer, listar, buscar, ejecutar comandos de consulta y buscar en internet.
- [ ] `plan` no tiene ninguna herramienta que cree, modifique o borre.
- [ ] `build` tiene el catálogo completo.
- [ ] Nada se escribe sin que `plan` lo haya propuesto y tú lo hayas aprobado.
- [ ] Una aprobación no habilita cambios distintos de los propuestos.
- [ ] `crear_archivo` falla si el archivo ya existe; `escribir_archivo` lo sobrescribe.
- [ ] Los comandos de la lista blanca se ejecutan sin pedir permiso.
- [ ] Un comando fuera de la lista blanca pide permiso antes de ejecutarse.
- [ ] La terminal no puede crear, editar ni borrar archivos del proyecto, ni con comandos indirectos.
- [ ] La terminal sí puede compilar, correr pruebas y revisar estilo y tipos.
- [ ] No puede descartar cambios del repositorio ni hacer commits.
- [ ] Un comando sin fin o con salida sin fin se corta y se avisa, sin colgar el harness.
- [ ] La búsqueda por internet envía solo la consulta, nunca contenido del proyecto.
- [ ] `abrir_pagina` devuelve el contenido de una página y ese contenido se audita.
- [ ] El agente no tiene herramientas fuera de este catálogo.

## Requisitos no funcionales

- Ninguna respuesta del modelo se agota por un comando descuidado.
- El contenido de una página web cuenta para el mismo presupuesto de contexto que el resto.

## Dependencias funcionales

- [[specs/SPEC-AGENTE-BASE]]
- [[specs/SPEC-ARCHIVOS]]
- [[specs/SPEC-NODO-CONTEXTO]]
- [[specs/SPEC-RESOLVER]]

## Supuestos

- Los comandos concretos de la lista blanca, el límite de tiempo y el límite de salida se fijan en FASE 2 y FASE 3.
- Cómo se garantiza el bloqueo de escritura en la terminal se define en FASE 2 y FASE 3.

## Referencias

- [[IDEA]]
