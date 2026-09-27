> T-F018 — input: edición del cursor (flechas, home/end) sin perder los atajos de la app.
> Acción de esta descomposición: `actualizar`.

## Referencias

- [[frontend/01-domain/DOMAIN]] — §1 `input`: compone y envía; no valida reglas.
- [[frontend/02-interfaces/INTERFACES]] — §4: con el input enfocado, las letras escriben; las flechas y home/end editan.
- [[specs/SPEC-INTERFAZ]] — zona 2: una sola línea que se edita en cualquier punto.
- [[specs/SPEC-KEYBINDS]] — §Resolución por contexto: las teclas que son atajo no llegan al editor.

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-F018-01 | actualizar | Dejar vivas las teclas de edición del `textinput` (flechas, home/end, borrado por palabra) y anular solo pegar y sugerencias | completada | `internal/tui/input.go` | `go build ./internal/tui/` sin errores |
| T-F018-02 | actualizar | Aplicar el ajuste al construir la línea, no solo en `WindowSizeMsg` | completada | `internal/tui/input.go`, `internal/tui/app.go` | Test: la línea recién creada edita el cursor |
| T-F018-03 | actualizar | Prueba de que flechas/home/end mueven el caret y de que `ctrl+a`/`ctrl+f` siguen siendo acción | completada | `internal/tui/input_test.go` | Test `TestElCursorSeMueveConLasFlechasYHomeEnd` en verde |

Dependencias: T-F018-02 depende de T-F018-01; T-F018-03 depende de T-F018-01..02.
