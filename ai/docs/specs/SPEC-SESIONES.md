---
title: SPEC — Sesiones por proyecto
tags: [specs, requisito]
depende_de:
  - "[[IDEA]]"
  - "[[specs/SPEC-ARCHIVOS]]"
  - "[[specs/SPEC-ORQUESTADOR-FLUJOS]]"
  - "[[specs/SPEC-MODELO-MOTOR]]"
---
# SPEC — Sesiones por proyecto

Prioridad: P0 (núcleo)

## Propósito

Varias sesiones por proyecto, cada una con su contexto independiente, que pueden trabajar al mismo tiempo.

## Alcance

Incluye crear, cambiar, retomar y cerrar sesiones, el aislamiento entre proyectos y la ejecución en segundo plano con estado visible. Incluye qué motor y qué modelo usa cada sesión y qué le pasa cuando se cambian o cuando el motor desaparece.
No incluye qué hace el trabajo que corre dentro de una sesión, que está en [[specs/SPEC-ORQUESTADOR-FLUJOS]], ni la administración de los motores, que está en [[specs/SPEC-MODELO-MOTOR]].

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
- No existe una sesión previa al uso: al abrir el proyecto la vista es la bienvenida y no hay sesión activa. La primera petición desde la bienvenida crea una sesión nueva; `Ctrl+X n` desde la vista principal también crea una y la deja activa.
- Toda sesión nace con un nombre provisional (`Nueva sesión`). Con su primera petición, el modelo genera un título breve a partir de ella y ese pasa a ser su nombre.
- El título se pide una sola vez: una sesión ya titulada no vuelve a pedirlo. Si el modelo falla, se conserva el nombre provisional y se reintenta con la siguiente petición.
- El identificador de una sesión es permanente; su nombre es un atributo mutable. Renombrar una sesión no cambia sus mensajes ni ninguna referencia interna.
- Al borrar la última sesión del proyecto, la vista vuelve a la bienvenida de inmediato.
- El contenido de una sesión no se ve desde otra sesión del mismo proyecto.
- Una sesión en segundo plano sigue ejecutándose aunque el usuario cambie de sesión o salga de la vista.
- Cada sesión expone su estado: inactiva, trabajando, esperando permiso, terminada o con error.
- Cambiar de sesión no detiene nada.
- El chat de una carpeta nunca aparece en otra carpeta.
- El contexto acumulado de una sesión no se comparte con las demás del mismo proyecto; su historial se reconstruye para el modelo según [[specs/SPEC-HISTORIAL-CONVERSACION]].
- **El motor y el modelo son recursos de la sesión, no parte de su identidad.** Cada sesión guarda el par que está usando, así que puede cambiarlo sin que cambien su identificador, su nombre ni su historial, y la conversación continúa donde estaba. Ver [[specs/SPEC-MODELO-MOTOR]].
- Una sesión puede cambiar de motor, cambiar de modelo o cambiar ambos a la vez. Al hacerlo, su historial sigue siendo el mismo.
- El par motor/modelo de una sesión se conserva al salir y volver: al retomarla se trabaja contra el motor que tenía.
- Una sesión nueva no nace con motor propio: parte del último usado, que es una preferencia global del usuario.
- **Eliminar el motor de una sesión no elimina ni invalida la sesión.** Sigue existiendo, con su historial; lo que hace es avisar de que su motor ya no está y esperar a que el usuario elija otro.
- Desactivar el motor de una sesión tampoco la elimina: avisa y espera, con el historial intacto.
- Que el motor sea por sesión no significa que dos sesiones generen a la vez. Es para trabajar por separado con un motor en una sesión y otro en otra. Ver [[specs/SPEC-MODELO-MOTOR]].

## Criterios de aceptación

- [ ] Al abrir una carpeta, su chat es independiente de cualquier otra carpeta.
- [ ] Un proyecto puede tener varias sesiones simultáneas.
- [ ] Los chats de dos sesiones del mismo proyecto no se mezclan.
- [ ] Una sesión en segundo plano sigue trabajando y su estado es visible desde otra sesión.
- [ ] Al reabrir una carpeta se listan sus sesiones y se puede retomar cualquiera con su historial.
- [ ] El estado de cada sesión es visible en todo momento.
- [ ] `Ctrl+X n` crea una sesión nueva y la deja activa.
- [ ] La primera petición desde la bienvenida crea una sesión nueva, sin que exista ninguna antes.
- [ ] Una sesión recién creada se llama «Nueva sesión» y pasa a llevar un título generado por el modelo tras su primera petición.
- [ ] Si el modelo no puede generar el título, la sesión conserva el nombre provisional y lo reintenta con la siguiente petición.
- [ ] Al borrar la última sesión del proyecto, la vista vuelve a la bienvenida.
- [ ] Al retomar una sesión, se trabaja con el motor y el modelo que tenía guardados.
- [ ] Cambiar de motor o de modelo no cambia el identificador, el nombre ni el historial de la sesión, y la conversación continúa.
- [ ] Una sesión nueva arranca con el último motor y modelo usados.
- [ ] Si el motor de una sesión se elimina o se desactiva, la sesión sigue existiendo con su historial y avisa de que necesita otro motor, sin bloquear el resto de sesiones.
- [ ] Reactivar un motor previamente desactivado devuelve las sesiones que lo usaban a poder trabajar sin que el usuario tenga que reconfigurarlas.

## Requisitos no funcionales

- Aislamiento: no hay fuga de contenido entre proyectos ni entre sesiones del mismo proyecto.
- Robustez: cambiar de sesión no interrumpe la ejecución en curso.

## Dependencias funcionales

- [[specs/SPEC-ARCHIVOS]]
- [[specs/SPEC-ORQUESTADOR-FLUJOS]]
- [[specs/SPEC-MODELO-MOTOR]]

## Supuestos

- Abrir la misma carpeta en dos instancias a la vez queda por definir.

## Referencias

- [[IDEA]]
