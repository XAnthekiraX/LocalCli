> T-F017 — atajos de sesión: `Ctrl+X n` crea una sesión nueva y la deja activa; `Ctrl+D` elimina la sesión resaltada en el modal, pidiendo confirmación si está trabajando.
> Acción de esta descomposición: `actualizar` (sobre T-F001 keys y T-F014 modales).

## Referencias

- [[specs/SPEC-KEYBINDS]] — §Acción (`session_new`, `session_delete`) y §Resolución por contexto (el modal de sesiones añade `session_delete`).
- [[specs/SPEC-SESIONES]] — crear desde la vista principal y borrado en cascada con confirmación si trabaja.
- [[specs/SPEC-INTERFAZ]] — §Modales y §Teclado.
- [[frontend/02-interfaces/INTERFACES]] — §4 teclado.

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-F017-01 | actualizar | Añadir las acciones `session_new` (`<leader>n`) y `session_delete` (`ctrl+d`) al mapa y al resolver | completada | `internal/tui/keymap.go`, `internal/tui/keys.go`, `internal/tui/keyresolver.go` | Test: las dos teclas resuelven su acción |
| T-F017-02 | actualizar | `Ctrl+X n` crea la sesión, la deja activa y abre el chat en blanco; para eso el puerto gana `Crear` | completada | `internal/tui/wire.go`, `internal/tui/app.go`, `arranque.go` | Test: `TestCtrlXNCreaUnaSesionNuevaYLaDejaActiva` |
| T-F017-03 | actualizar | `Ctrl+D` borra la resaltada; si trabaja pide confirmación (`s`/`n`/`esc`) y entonces se borra en cascada, retomando otra sesión si era la activa | completada | `internal/tui/app.go`, `internal/tui/wire.go`, `arranque.go` | Tests: `TestCtrlDBorraSinConfirmarSiNoTrabaja`, `TestCtrlDPideConfirmacionCuandoLaSesionTrabaja` |
| T-F017-04 | actualizar | Registrar el teclado nuevo en `welcome` y en la vista principal, y cubrirlo con pruebas | completada | `internal/tui/welcome.go`, `internal/tui/sesiones_ctrl_test.go` | Test: `TestLaBienvenidaAbreElModalDeSesionesYAlElegirPasaALaPrincipal` |

Dependencias: T-F017-02 y T-F017-03 dependen de T-F017-01.
