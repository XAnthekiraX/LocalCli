> T-F028 — aprobaciones: mostrar sin robar el teclado y decidir con el ratón.
> Acción de esta descomposición: `actualizar`.

## Referencias

- [[specs/SPEC-INTERFAZ-ATAJOS]] — §Reglas: la aprobación se muestra sin secuestrar el teclado; un clic sobre «aprobar»/«declinar» decide.
- [[specs/SPEC-INTERFAZ]] — §"El aviso que no se puede ocultar" y §Reglas: una aprobación pendiente se ve sin bloquear la escritura.
- [[frontend/01-domain/DOMAIN]] — §1 `approvals`/`notify` y §2 reglas de presentación.
- [[frontend/02-interfaces/INTERFACES]] — §4 atajos y ratón.

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-F028-01 | actualizar | Separar «verse» (`Abierto`) de «tener el teclado» (`Enfocado`): `Fijar` muestra el panel sin enfocarlo, así la decisión se ve y el input sigue escribiendo. `Ctrl+A` enfoca o suelta el foco; `Esc` lo cierra | completada | `internal/tui/approvals.go`, `internal/tui/app.go` | Tests `TestUnaAprobaciónPendienteNoBloqueaElInput` y `TestMostrarYEnfocarElPanelDeAprobacionesNoDetieneNada` en verde |
| T-F028-02 | actualizar | `Aprobaciones.Resolver` cierra el panel y suelta el foco cuando se queda sin pendientes | completada | `internal/tui/approvals.go` | Test `TestClicEnDeclinarDeclinaLaAprobacion` (comprueba el cierre) en verde |
| T-F028-03 | actualizar | El ratón decide sin foco: un clic sobre «aprobar» o «declinar» de una fila resuelve esa aprobación; un arrastre sigue seleccionando texto | completada | `internal/tui/selection.go`, `internal/tui/app.go`, `internal/tui/approvals.go` | Tests `TestClicEnAprobarResuelveLaFilaPulsada`, `TestClicEnDeclinarDeclinaLaAprobacion` y `TestClicFueraDeLasOpcionesNoResuelve` en verde |
| T-F028-04 | actualizar | El pie del panel dice qué teclas valen según el foco; se regenera la salida dorada | completada | `internal/tui/approvals.go`, `internal/tui/testdata/aprobaciones.golden` | `go test ./internal/tui/` en verde |

Dependencias: T-F028-02 y T-F028-03 dependen de T-F028-01; T-F028-04 depende de T-F028-01.
