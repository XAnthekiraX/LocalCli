---
title: SPEC — Integración con Ollama y perfil de hardware
tags: [specs, requisito]
depende_de:
  - "[[IDEA]]"
  - "[[specs/SPEC-SESIONES]]"
  - "[[specs/SPEC-NODO-CONTEXTO]]"
  - "[[specs/SPEC-PANEL-CONTEXTO]]"
  - "[[PROJECT]]"
relacionado:
  - "[[specs/SPEC-TOOLS]]"
---
# SPEC — Integración con Ollama y perfil de hardware

Prioridad: P0 (núcleo)

## Propósito

Hablar con el modelo local de forma que quepa y rinda en una máquina con 4 GB de VRAM y 16 GB de RAM.

## Alcance

Incluye conectarse a Ollama, ver los modelos disponibles, proponer un perfil que quepa en el hardware objetivo y avisar cuando no cabe.
No incluye proveedores distintos de Ollama ni la visualización de tokens, que está en [[specs/SPEC-PANEL-CONTEXTO]].

## Actores

- **Usuario**: elige el modelo y acepta o cambia el perfil.
- **Sistema**: detecta, informa de qué cabe y avisa cuando no cabe.

## Flujo principal

1. La herramienta se conecta a Ollama.
2. Lee los modelos disponibles.
3. Muestra cuáles caben en el hardware detectado y cuáles no.
4. **El usuario elige el modelo.** La herramienta no decide por él.
5. Si el modelo elegido no cabe, avisa y no lo carga en silencio.
6. El perfil queda guardado y se aplica a las sesiones nuevas.

## Flujos alternativos

- Ollama no está corriendo: avisa y explica cómo levantarlo.
- No hay ningún modelo que quepa en el hardware.
- El modelo elegido no cabe: lo dice y no lo carga.
- La máquina tiene más recursos: se puede subir el tamaño de contexto.

## Reglas de negocio

- El modelo lo elige el usuario. La herramienta informa y avisa, no decide.
- La herramienta muestra qué modelos caben en el hardware detectado antes de que el usuario elija.
- El perfil por defecto se calcula para 4 GB de VRAM y 16 GB de RAM.
- El tamaño de contexto se limita para que quepa junto con el modelo cargado.
- Si el modelo elegido no cabe, la herramienta avisa y no lo carga en silencio.
- El perfil se aplica a las sesiones nuevas del proyecto.
- Todo el modelo y toda la conversación ocurren en la máquina local: no se envía nada fuera.
- La única excepción es la búsqueda en internet de [[specs/SPEC-TOOLS]], y solo sale la consulta, nunca contenido del proyecto.
- El tamaño de contexto disponible se tiene en cuenta al decidir cuánto contexto entregar.

## Criterios de aceptación

- [ ] La herramienta se conecta a Ollama y lista los modelos disponibles.
- [ ] Indica cuáles caben en el hardware detectado antes de que el usuario elija.
- [ ] El usuario elige el modelo; la herramienta no lo decide por él.
- [ ] Si el modelo elegido no cabe, avisa y no lo carga.
- [ ] El perfil confirmado se aplica a las sesiones nuevas.
- [ ] La herramienta sigue funcionando con 16 GB de RAM sin agotar la memoria.
- [ ] Si Ollama no está disponible, avisa con una instrucción clara.
- [ ] La única información que sale de la máquina es la consulta de una búsqueda, nunca contenido del proyecto.

## Requisitos no funcionales

- Consumo de VRAM dentro de 4 GB en la configuración por defecto.
- No añadir esperas propias al tiempo de respuesta del modelo.

## Dependencias funcionales

- [[specs/SPEC-SESIONES]]
- [[specs/SPEC-NODO-CONTEXTO]]
- [[specs/SPEC-PANEL-CONTEXTO]]
- [[PROJECT]]

## Supuestos

- Los valores concretos del perfil (modelo, tamaño de contexto, parámetros de inferencia) se fijan en FASE 2 y FASE 3.

## Referencias

- [[IDEA]]
