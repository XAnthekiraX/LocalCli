---
title: SPEC — Skills
tags: [specs, requisito]
depende_de:
  - "[[IDEA]]"
  - "[[specs/SPEC-AGENTE-BASE]]"
  - "[[specs/SPEC-SESIONES]]"
  - "[[specs/SPEC-NODO-CONTEXTO]]"
---
# SPEC — Skills

Prioridad: P3 (personalización, diferible)

## Propósito

Cargar skills que añaden una capacidad concreta al agente cuando la tarea lo necesita.

**No hay ninguna skill por defecto.** `LocalCli` no viene con skills de fábrica: las crea el usuario, en markdown, como en opencode. Esta spec describe cómo cargarlas, no cuáles existen.

## Alcance

Incluye ver las skills disponibles, cargarlas y quitarlas en una sesión.
No incluye el agente base, los agentes propios ni los flujos propios.

## Actores

- **Usuario**: carga y quita skills.
- **Sistema**: las aplica a la sesión.
- **Agente**: las usa cuando la tarea lo pide.

## Flujo principal

1. El usuario ve las skills disponibles.
2. Carga una en una sesión.
3. El agente la usa cuando la tarea lo requiere.
4. El usuario la quita.

## Flujos alternativos

- La skill no aplica a la tarea en curso: el agente sigue sin ella.
- La skill necesita un contexto que no está disponible: avisa.
- Varias skills cargadas en la misma sesión.

## Reglas de negocio

- Cargar una skill solo afecta a la sesión donde se carga.
- Una skill no puede cambiar las reglas de permiso.
- Una skill no amplía lo que el agente puede hacer fuera de la carpeta del proyecto.
- Cargar o quitar una skill no borra el historial de la sesión.
- Solo se cargan skills que estén disponibles en la máquina.
- No hay skills por defecto. Las crea el usuario.
- Una skill se escribe en markdown, como en opencode.
- La skill se aplica como contexto adicional de la etapa, no como sustituto del contexto del nodo.

## Criterios de aceptación

- [ ] Se listan las skills disponibles.
- [ ] Se puede cargar y quitar una skill en una sesión.
- [ ] La skill cargada solo afecta a esa sesión.
- [ ] Una skill no cambia las reglas de permiso.
- [ ] Cargar o quitar una skill no borra el historial.
- [ ] Si la skill necesita contexto que no está, avisa en lugar de suponer.

## Requisitos no funcionales

- Cargar una skill no debe degradar el consumo de contexto de la sesión.

## Dependencias funcionales

- [[specs/SPEC-AGENTE-BASE]]
- [[specs/SPEC-SESIONES]]
- [[specs/SPEC-NODO-CONTEXTO]]

## Supuestos

- De dónde vienen las skills y en qué formato se escriben queda en FASE 2 y FASE 3.

## Referencias

- [[IDEA]]
