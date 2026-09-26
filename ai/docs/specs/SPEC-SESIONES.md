---
title: SPEC — Sesiones por proyecto
tags: [specs, requisito]
depende_de:
  - "[[IDEA]]"
  - "[[specs/SPEC-ARCHIVOS]]"
  - "[[specs/SPEC-MOTOR-FLUJOS]]"
  - "[[specs/SPEC-OLLAMA-PERFIL]]"
---
# SPEC — Sesiones por proyecto

Prioridad: P0 (núcleo)

## Propósito

Varias sesiones por proyecto, cada una con su contexto independiente, que pueden trabajar al mismo tiempo.

## Alcance

Incluye crear, cambiar, retomar y cerrar sesiones, el aislamiento entre proyectos y la ejecución en segundo plano con estado visible.
No incluye qué hace el trabajo que corre dentro de una sesión, que está en [[specs/SPEC-MOTOR-FLUJOS]].

## Actores

- **Usuario**: abre, cambia y cierra sesiones.
- **Sistema**: mantiene el estado de cada una.

## Flujo principal

1. El usuario abre la herramienta en una carpeta.
2. Esa carpeta es un proyecto.
3. Se crea una sesión nueva o se retoma una existente.
4. Se trabaja en la sesión.
5. El usuario puede abrir otra sesión del mismo proyecto sin detener la anterior.

## Ejemplo

Se abre una sesión y se deja una tarea de frontend trabajando. Se abre otra sesión y se deja una tarea de backend. Las dos avanzan a la vez y ninguna ve el contexto de la otra.

## Flujos alternativos

- Cerrar y volver: al reabrir la carpeta se ven las sesiones y se retoma cualquiera con su historial.
- Una sesión queda esperando permiso: aparece marcada como tal.
- La sesión terminó: su resultado queda guardado en su historial.
- El usuario cierra la sesión mientras corre un flujo: se pregunta qué hacer con él.

## Reglas de negocio

- Una sesión pertenece a un solo proyecto. El proyecto es la carpeta abierta.
- El contenido de una sesión no se ve desde otra sesión del mismo proyecto.
- Una sesión en segundo plano sigue ejecutándose aunque el usuario cambie de sesión o salga de la vista.
- Cada sesión expone su estado: inactiva, trabajando, esperando permiso, terminada o con error.
- Cambiar de sesión no detiene nada.
- El chat de una carpeta nunca aparece en otra carpeta.
- El contexto acumulado de una sesión no se comparte con las demás del mismo proyecto.

## Criterios de aceptación

- [ ] Al abrir una carpeta, su chat es independiente de cualquier otra carpeta.
- [ ] Un proyecto puede tener varias sesiones simultáneas.
- [ ] Los chats de dos sesiones del mismo proyecto no se mezclan.
- [ ] Una sesión en segundo plano sigue trabajando y su estado es visible desde otra sesión.
- [ ] Al reabrir una carpeta se listan sus sesiones y se puede retomar cualquiera con su historial.
- [ ] El estado de cada sesión es visible en todo momento.

## Requisitos no funcionales

- Aislamiento: no hay fuga de contenido entre proyectos ni entre sesiones del mismo proyecto.
- Robustez: cambiar de sesión no interrumpe la ejecución en curso.

## Dependencias funcionales

- [[specs/SPEC-ARCHIVOS]]
- [[specs/SPEC-MOTOR-FLUJOS]]
- [[specs/SPEC-OLLAMA-PERFIL]]

## Supuestos

- Abrir la misma carpeta en dos instancias a la vez queda por definir.

## Referencias

- [[IDEA]]
