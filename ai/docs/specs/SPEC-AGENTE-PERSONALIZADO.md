# SPEC — Agente personalizado

Prioridad: P3 (personalización, diferible)

## Propósito

Poder usar un agente propio en vez del agente base.

## Alcance

Incluye definir, guardar, editar, borrar y usar agentes propios.
No incluye el agente base, que está en [[specs/SPEC-AGENTE-BASE]], ni los flujos propios.

## Actores

- **Usuario**: define y elige el agente.
- **Sistema**: guarda y aplica.

## Flujo principal

1. El usuario define un agente con nombre, instrucciones y herramientas permitidas.
2. Lo guarda.
3. Elige qué sesión lo usa.
4. A partir de ese momento, ese agente responde en esa sesión.

## Flujos alternativos

- Editar un agente existente.
- Borrar un agente.
- Volver al agente base.
- Un agente que intenta usar una herramienta que no tiene permitida.

## Reglas de negocio

- El agente base está siempre disponible.
- Un agente propio no puede salirse de las herramientas que se le permitieron.
- Un agente propio no cambia las reglas de permiso.
- Cambiar de agente en una sesión no borra su historial.
- Los agentes propios solo se ven dentro de su proyecto.

## Criterios de aceptación

- [ ] Se puede crear un agente con nombre, instrucciones y herramientas.
- [ ] Se puede editar y borrar un agente propio.
- [ ] Se puede elegir qué agente usa una sesión.
- [ ] Un agente propio no puede usar herramientas que no tenga permitidas.
- [ ] El agente base sigue disponible siempre.
- [ ] Cambiar de agente no borra el historial de la sesión.

## Requisitos no funcionales

- Cargar un agente propio no debe tardar en aplicarse a la sesión.

## Dependencias funcionales

- [[specs/SPEC-AGENTE-BASE]]
- [[specs/SPEC-SESIONES]]

## Supuestos

- El formato de definición y dónde se guardan los agentes quedan en FASE 2 y FASE 3.

## Referencias

- [[IDEA]]
