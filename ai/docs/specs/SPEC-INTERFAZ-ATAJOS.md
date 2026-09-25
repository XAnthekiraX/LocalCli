# SPEC — Interfaz, atajos y panel de aprobaciones

Prioridad: P2

## Propósito

Poder trabajar la interfaz sin soltar el teclado, y ver de un vistazo todo lo que está esperando tu aprobación.

## Alcance

Incluye los atajos de teclado, su configuración y el panel global de aprobaciones.
No incluye las reglas de cuándo se pide permiso, que están en [[specs/SPEC-ARCHIVOS]], ni la disposición general de la pantalla, que está en [[specs/SPEC-INTERFAZ]].

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

`nombre de la sesión | acción propuesta | aprobar | declinar | <opción por definir>`

## Flujos alternativos

- Dos sesiones esperan a la vez: aparecen como dos líneas independientes.
- El usuario está trabajando en otra sesión: el panel se abre sin interrumpir.
- El usuario declina: la sesión recibe el rechazo.
- La sesión terminó mientras esperaba: la línea se marca como obsoleta.

## Reglas de negocio

- Trae atajos por defecto que se pueden cambiar sin reinstalar.
- Un atajo no puede quedar asignado a dos acciones.
- El panel muestra las aprobaciones pendientes de cualquier sesión.
- Resolver una aprobación solo afecta a la sesión de esa línea.
- El panel se puede abrir y cerrar sin detener el trabajo de ninguna sesión.
- Un atajo no cambia nunca la regla de permiso: solo la forma de invocarla.

## Criterios de aceptación

- [ ] Se listan los atajos disponibles y la acción de cada uno.
- [ ] Se puede cambiar un atajo y el cambio queda guardado.
- [ ] El panel muestra las aprobaciones pendientes de todas las sesiones.
- [ ] Cada aprobación se resuelve de forma independiente.
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
