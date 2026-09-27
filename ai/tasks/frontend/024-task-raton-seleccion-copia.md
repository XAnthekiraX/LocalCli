> T-F024 — ratón: rueda para el historial, arrastre para seleccionar y copia al portapapeles.
> Acción de esta descomposición: `actualizar`.

## Referencias

- [[frontend/01-domain/DOMAIN]] — §2: la rueda recorre el historial y el arrastre copia.
- [[frontend/02-interfaces/INTERFACES]] — §4: ratón.
- [[specs/SPEC-INTERFAZ]] — reglas de scroll y copia.

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-F024-01 | actualizar | Captura del ratón en el arranque (`tea.WithMouseCellMotion`) | completada | `main.go` | `go build ./...` sin errores |
| T-F024-02 | actualizar | Rueda arriba/abajo desplaza el historial del chat | completada | `internal/tui/selection.go` | Test `TestLaRuedaDesplazaElHistorial` en verde |
| T-F024-03 | actualizar | Arrastre con el botón izquierdo selecciona texto y al soltar se copia (wl-copy/xclip/xsel/pbcopy u OSC 52) | completada | `internal/tui/selection.go` | Tests `TestTextoSeleccionadoExtraeElRangoSinCódigos` y `TestElArrastreDelRatónDisparaLaCopia` |
