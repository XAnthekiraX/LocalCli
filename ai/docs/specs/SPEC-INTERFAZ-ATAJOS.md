---
title: SPEC — Interfaz, atajos y panel de aprobaciones
tags: [specs, requisito]
depende_de:
  - "[[IDEA]]"
  - "[[specs/SPEC-KEYBINDS]]"
  - "[[specs/SPEC-ARCHIVOS]]"
  - "[[specs/SPEC-INTERFAZ]]"
  - "[[specs/SPEC-SESIONES]]"
---
# SPEC — Interfaz, atajos y panel de aprobaciones

Prioridad: P2

## Propósito

Poder trabajar la interfaz sin soltar el teclado, y ver de un vistazo todo lo que está esperando tu aprobación.

## Alcance

Incluye los atajos de teclado, su configuración y el panel global de aprobaciones.
No incluye las reglas de cuándo se pide permiso, que están en [[specs/SPEC-ARCHIVOS]], ni la disposición general de la pantalla, que está en [[specs/SPEC-INTERFAZ]], ni el mecanismo de resolución de teclas (líder, timeout, contextos), que está en [[specs/SPEC-KEYBINDS]].

El panel de aprobaciones **no es** el panel de datos. El de datos es de lectura y solo información; este son decisiones que esperan tu respuesta.

## Actores

- **Usuario**: navega, reasigna atajos y resuelve aprobaciones.

## Flujo principal — atajos

1. Se muestra la ayuda con los atajos disponibles.
2. El usuario navega, envía, cambia de sesión y abre el panel sin usar el ratón.
3. Si un atajo no sirve, lo reasigna.

## Flujo principal — panel de aprobaciones

1. Una sesión necesita permiso para un cambio.
2. El panel muestra el nombre de la sesión, qué propone y las opciones disponibles.
3. El usuario aprueba o declina.
4. La sesión continúa o se detiene.

## Formato de una línea del panel

`sesión | acción propuesta | aprobar | declinar`

La sesión va por su identificador abreviado (ocho caracteres) y la acción en corto —verbo y ruta, con la carpeta del usuario abreviada a `~`—, para que la línea quepa de un vistazo. La tercera opción de decisiones sigue por definir: no se reserva un hueco vacío que alargue la fila.

Ejemplo: `24afd397 | leer ~/Imágenes/picture.jpeg (fuera) | aprobar | declinar`

## Flujos alternativos

- Dos sesiones esperan a la vez: aparecen como dos líneas independientes.
- El usuario está trabajando en otra sesión: el panel se abre sin interrumpir.
- El usuario declina: la sesión recibe el rechazo.
- La sesión terminó mientras esperaba: la línea se marca como obsoleta.

## Reglas de negocio

- Trae atajos por defecto que se pueden cambiar sin reinstalar.
- El teclado funciona con un keymap central: una acción puede tener varios atajos, y existe una tecla líder (`Ctrl+X`, por defecto) que combina con la siguiente tecla (`Ctrl+X m` abre el modal de modelos, `Ctrl+X l` el de sesiones). Mecanismo completo en [[specs/SPEC-KEYBINDS]].
- El historial del chat se recorre con `↑`/`↓` (línea) y `pgup`/`pgdown` (página). La misma `↑`/`↓` navega la lista cuando hay un modal abierto: son ámbitos distintos.
- Los tres modales de la interfaz son **modelos** (`Ctrl+X m`), **sesiones** (`Ctrl+X l`) y **atajos** (`Ctrl+P`). `Esc` cierra cualquiera de ellos sin cambiar nada.
- `Tab` alterna el agente entre `plan` y `build`; el agente activo se ve a la izquierda del input.
- Un atajo no puede quedar asignado a dos acciones.
- El panel muestra las aprobaciones pendientes de cualquier sesión.
- Resolver una aprobación solo afecta a la sesión de esa línea.
- El panel se puede abrir y cerrar sin detener el trabajo de ninguna sesión.
- Una aprobación pendiente **se muestra pero no secuestra el teclado**: el panel aparece con sus opciones para poder decidir, y el input sigue escribiendo. El foco —que da el teclado al panel para `a`/`d`— se pide con `Ctrl+A`; sin foco, se decide con el ratón.
- Con el panel visible, un clic del ratón sobre «aprobar» o «declinar» de una línea resuelve esa aprobación; un clic fuera de esas palabras no decide nada.
- Un atajo no cambia nunca la regla de permiso: solo la forma de invocarla.

## Criterios de aceptación

- [ ] Se listan los atajos disponibles y la acción de cada uno, incluidas las secuencias con líder (el propio modal de atajos se abre con `Ctrl+P`).
- [ ] `Ctrl+X m` abre el modal de modelos y `Ctrl+X l` el modal de sesiones desde cualquier vista; `Enter` en el modal de sesiones abre la sesión elegida.
- [ ] `Esc` cierra cualquier modal abierto sin cambiar nada.
- [ ] `Tab` recorre los agentes disponibles (los base `plan` y `build` y los propios de `.localcli/agents/<carpeta>/`) y el indicador junto al input se actualiza.
- [ ] Se puede cambiar un atajo y el cambio queda guardado.
- [ ] El panel muestra las aprobaciones pendientes de todas las sesiones.
- [ ] Cada aprobación se resuelve de forma independiente.
- [ ] Una aprobación pendiente se muestra sin bloquear la escritura; `Ctrl+A` le da el teclado (`a`/`d`) y sin foco se decide con el ratón.
- [ ] Con el panel visible, un clic sobre «aprobar» o «declinar» resuelve esa aprobación.
- [ ] El panel no interrumpe el trabajo de ninguna sesión.
- [ ] Una aprobación que ya no aplica se marca como obsoleta.

## Requisitos no funcionales

- Abrir y cerrar el panel no debe afectar a las ejecuciones en curso.

## Dependencias funcionales

- [[specs/SPEC-ARCHIVOS]]
- [[specs/SPEC-INTERFAZ]]
- [[specs/SPEC-SESIONES]]

## Supuestos

- La tercera opción del panel, además de aprobar y declinar, está por definir.

## Referencias

- [[IDEA]]
