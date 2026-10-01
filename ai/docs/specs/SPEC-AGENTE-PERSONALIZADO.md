---
title: SPEC — Agente personalizado
tags: [specs, requisito]
depende_de:
  - "[[IDEA]]"
  - "[[specs/SPEC-AGENTE-BASE]]"
  - "[[specs/SPEC-SESIONES]]"
---
# SPEC — Agente personalizado

Prioridad: P3 (personalización, diferible)

## Propósito

Poder usar un agente propio en vez del agente base.

## Alcance

Incluye usar agentes propios: definirlos —creando una carpeta con dos archivos—, borrarlos y elegirlos en una sesión.
No incluye un editor de agentes dentro de la herramienta, que no existe: definirlos es escribir archivos del proyecto. No incluye el agente base, que está en [[specs/SPEC-AGENTE-BASE]], ni los flujos propios.

## Actores

- **Usuario**: define y elige el agente.
- **Sistema**: carga, aplica y expone.

## Flujo principal

1. El usuario crea una carpeta en `.localcli/agents/` con dos archivos: `agent.yaml` —nombre, descripción y permisos— y `prompt.md` —las instrucciones—.
2. El sistema lo carga al arrancar y lo añade a los agentes disponibles.
3. El usuario elige qué agente usa una sesión, con `Tab`.
4. A partir de ese momento, ese agente responde en esa sesión.

## Flujos alternativos

- Editar un agente existente: se edita cualquiera de sus dos archivos y el cambio se ve al reiniciar.
- Borrar un agente: se borra su carpeta.
- Volver al agente base.
- Un agente que intenta usar una herramienta que no tiene permitida.

## Reglas de negocio

- El agente base está siempre disponible.
- Un agente propio es una carpeta con `agent.yaml` y `prompt.md`; los dos archivos son obligatorios y sin los dos no carga.
- Lo que un agente puede hacer no se declara: sale de sus `permissions`. No hay una lista de herramientas que pueda contradecirlas.
- Un agente propio no puede salirse de las herramientas que se le concedieron.
- Un agente propio no cambia las reglas de permiso: no existe una vía para que se conceda más de lo que concede su `agent.yaml`.
- Un `default: allow` en un agente propio no carga, y un campo desconocido tampoco.
- Cambiar de agente en una sesión no borra su historial.
- Los agentes propios solo se ven dentro de su proyecto: viven en la carpeta del proyecto, versionados con él.
- Sin `permissions`, el agente propio es de solo conversación, y eso es válido.

## Criterios de aceptación

- [ ] Crear una carpeta con `agent.yaml` y `prompt.md` válidos añade un agente disponible sin tocar el código.
- [ ] El nombre del agente es el de `name`, no el de la carpeta.
- [ ] Editar los archivos de la carpeta cambia el agente al reiniciar; borrar la carpeta lo quita de la lista.
- [ ] Una carpeta con un solo uno de los dos archivos no carga.
- [ ] Se puede elegir qué agente usa una sesión.
- [ ] Un agente propio no puede usar herramientas que no tenga permitidas.
- [ ] El agente base sigue disponible siempre.
- [ ] Cambiar de agente no borra el historial de la sesión.
- [ ] Un agente sin `permissions` se puede elegir y solo conversa.

## Requisitos no funcionales

- Cargar un agente propio no debe tardar en aplicarse a la sesión.

## Dependencias funcionales

- [[specs/SPEC-AGENTE-BASE]]
- [[specs/SPEC-SESIONES]]

## Supuestos

- El formato y la ubicación ya no están abiertos: una carpeta en `.localcli/agents/` con `agent.yaml` y `prompt.md`. Ver [[specs/SPEC-AGENTE-BASE]] §Dónde se definen.
- Lo que queda abierto es si la herramienta tendrá alguna vez un editor de agentes. Hoy no lo tiene, y esta spec no lo pide: no es necesario para usar un agente propio.
- Un agente propio no declara `skills`: la capacidad no existe. Ver [[specs/SPEC-SKILLS]].

## Referencias

- [[IDEA]]
