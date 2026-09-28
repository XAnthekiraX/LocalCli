> T-F040 — ratón: la selección se resalta en video inverso y la rueda no la cancela (se mantiene anclada al texto).
> Acción de esta descomposición: `actualizar`.

## Referencias

- [[frontend/01-domain/DOMAIN]] — §2: el arrastre selecciona (resaltado) y la rueda no cancela la selección.
- [[frontend/02-interfaces/INTERFACES]] — §4: ratón.
- [[specs/SPEC-INTERFAZ]] — reglas de scroll y copia.

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-F040-01 | actualizar | Resaltar en video inverso el tramo seleccionado con el ratón (overlay sobre el marco en `View`) | completada | `internal/tui/selection.go`, `internal/tui/styles.go`, `internal/tui/app.go` | Tests `TestElResaltadoEnvuelveSoloLaSelección` y `TestElArrastrePintaLaSelección` |
| T-F040-02 | actualizar | La rueda no cancela la selección: se mantiene anclada al texto al desplazar el historial | completada | `internal/tui/selection.go`, `internal/tui/app.go` | Tests `TestLaRuedaNoCancelaLaSelecciónEnCurso` y `TestLaSelecciónSeAnclaAlScroll` |
| T-F040-03 | actualizar | Al soltar, el realce desaparece (la selección ya no vive) | completada | `internal/tui/selection.go` | Test `TestLaSelecciónDesapareceAlSoltar` |
| T-F040-04 | actualizar | Aviso transitorio `[Copiado]` arriba a la derecha, que se apaga solo | completada | `internal/tui/selection.go`, `internal/tui/styles.go`, `internal/tui/app.go` | Test `TestElAvisoDeCopiadoSeVeArribaALaDerechaYSeApagaSolo` |
| T-F040-05 | actualizar | Documentar el realce, el anclaje, el borrado al soltar y el aviso en las fuentes de la capa | completada | `ai/docs/specs/SPEC-INTERFAZ.md`, `ai/docs/frontend/02-interfaces/INTERFACES.md`, `ai/docs/frontend/01-domain/DOMAIN.md` | `go build ./...`, `go vet`, `gofmt`, `go test ./... -count=1` |
