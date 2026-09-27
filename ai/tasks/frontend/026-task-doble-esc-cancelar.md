> T-F026 — doble `esc` para cancelar el trabajo en curso.
> Acción de esta descomposición: `actualizar`.

## Referencias

- [[frontend/01-domain/DOMAIN]] — §2: el primer esc pide confirmación, el segundo cancela.
- [[frontend/02-interfaces/INTERFACES]] — §4/§5: doble esc y estados de espera.
- [[specs/SPEC-INTERFAZ]] — reglas: doble esc.

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-F026-01 | actualizar | El primer `esc` (sesión trabajando) pide confirmación con aviso bajo el input; el segundo cancela | completada | `internal/tui/app.go` | Test `TestElDobleEscCancelaElTrabajoEnCurso` en verde |
| T-F026-02 | actualizar | Cualquier otra tecla descarta la confirmación; cambiar de sesión o cerrar el turno la limpia | completada | `internal/tui/app.go`, `internal/tui/wire.go` | Test `TestElDobleEscSeDescartaConOtraTecla` y `TestSinTrabajoElEscNoPideCancelar` |
