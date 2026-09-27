> T-F019 — chat: scroll del historial (ventana con seguimiento del final) y sus atajos.
> Acción de esta descomposición: `actualizar` (supersede el recorte de T-F005-07).

## Referencias

- [[frontend/01-domain/DOMAIN]] — §2: el historial se recorre sin perder la línea de entrada de vista.
- [[frontend/02-interfaces/INTERFACES]] — §4: `↑`/`↓` y `pgup`/`pgdown` en la vista principal.
- [[specs/SPEC-INTERFAZ]] — zona 1: el historial se puede recorrer; sigue el final mientras no se sube.
- [[specs/SPEC-KEYBINDS]] — acciones `chat_scroll_up/down` y `chat_page_up/down`.

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-F019-01 | actualizar | Ventana del historial con `offset`/`maxOffset` y bandera de seguimiento del final | completada | `internal/tui/chat.go` | Test: `Ventana` devuelve el final y `Subir`/`Bajar` lo recorren |
| T-F019-02 | actualizar | Acciones `chat_scroll_up/down` y `chat_page_up/down` en el keymap y su despacho | completada | `keymap.go`, `keys.go`, `app.go` | Test: `↑`/`↓` recorren el historial en la vista y navegan el modal con un modal abierto |
| T-F019-03 | actualizar | Reservar el alto del bloque inferior (entrada, avisos, propuestas) y pintar el indicador de líneas arriba/abajo | completada | `internal/tui/app.go` | Test: la línea de entrada sigue visible con historial largo |
| T-F019-04 | actualizar | Pruebas de la ventana, del seguimiento del final y del scroll por teclado | completada | `internal/tui/chat_test.go`, `keyresolver_test.go` | `go test ./internal/tui/` en verde |

Dependencias: T-F019-02..03 dependen de T-F019-01; T-F019-04 depende de T-F019-01..03.
