# MAIN-TASKS.md

| ID     | Acción | Tarea                                                              | Dep                | Estado    | Detalle                     |
| ------ | ------ | ------------------------------------------------------------------ | ------------------ | --------- | --------------------------- |
| T-F000 | actualizar | Base de la TUI (modelo raíz, estilos, marco, puerto) y bienvenida  | —                  | completada | `000-task-bienvenida.md`    |
| T-F001 | crear  | Caja de entrada: edición, pegado y adjuntos                        | T-F000             | completada | `001-task-entrada.md`       |
| T-F002 | crear  | Mapa y resolución de teclas (keymap, keyresolver, keys, config)    | T-F000             | completada | `002-task-teclas.md`        |
| T-F003 | actualizar | Chat: globos, líneas de herramienta y razonamiento             | T-F000, T-F001     | completada | `003-task-chat.md`          |
| T-F004 | crear  | Paleta de comandos de flujo                                        | T-F001, T-F002     | completada | `004-task-paleta.md`        |
| T-F005 | crear  | Panel lateral de datos plegable                                    | T-F003             | completada | `005-task-panel.md`         |
| T-F006 | crear  | Modales: modelos, sesiones y atajos                                | T-F002             | completada | `006-task-modales.md`       |
| T-F007 | crear  | Panel de aprobaciones y aviso de pendientes                        | T-F003             | completada | `007-task-aprobaciones.md`  |
| T-F008 | crear  | Selección con ratón y copia al portapapeles                        | T-F003             | completada | `008-task-raton.md`         |
| T-F009 | crear  | Pruebas y salidas doradas de la TUI                                | T-F000 … T-F006, T-F008 | completada | `009-task-pruebas.md`   |
| T-F010 | actualizar | Selección con ratón que rebasa el viewport (desplazamiento automático en bordes) | T-F008 | completada | `010-task-seleccion-fuera-viewport.md` |
| T-F011 | actualizar | Línea de actividad fija sobre la caja de entrada               | T-F003             | completada | `011-task-indicador-fijo.md` |
| T-F012 | actualizar | Cableado del pegado (bracketed paste) y resaltado de tokens en la caja de entrada | T-F001 | completada | `012-task-pegado-bracketed.md` |
| T-F013 | actualizar | Pegado tokenizado en la bienvenida con el motor de adjuntos compartido | T-F001, T-F012 | completada | `013-task-pegado-bienvenida.md` |
| T-F014 | crear  | Vista de código real: Markdown de la respuesta y botón de copiar por bloque | T-F003 | completada | `014-task-vista-codigo.md` |
| T-F015 | crear  | Bloque de diagrama: pintar el diagrama Unicode bajo la línea de herramienta `diagram` en el chat | T-F003, T-F008 | completada | `015-task-diagrama.md` |
| T-F016 | actualizar | Bloque de diferencia: pintar el antes y el después del cambio bajo la línea de herramienta, con su tipo, color y recuperación | T-F003, T-F015, T-B027 | completada | `016-task-diferencia-cambios.md` |
| T-F017 | actualizar | Modal de modelos: eliminar el resaltado con `Ctrl+D` y confirmación `s/n`, con recarga de la lista y bloqueo del modelo en uso | T-F006, T-B028 | pendiente | `017-task-eliminar-modelos-modal.md` |
