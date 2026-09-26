> T-F008 — approvals: panel global de aprobaciones pendientes, resolubles una a una (a/d) con líneas obsoletas.
> Acción de esta descomposición: `crear`.

## Referencias

- [[frontend/01-domain/DOMAIN]] — §1 `approvals`: decisiones de todas las sesiones; no decide, envía la decisión del usuario.
- [[specs/SPEC-INTERFAZ-ATAJOS]] — flujo del panel, formato de línea (`sesión | acción propuesta | aprobar | declinar | <opción por definir>`), líneas independientes y obsoletas.
- [[frontend/02-interfaces/INTERFACES]] — §1 eventos `peticion_aprobacion`/`aprobacion_resuelta`; §2 Aprobar/Declinar; teclado `Ctrl+A`, `a`, `d`.
- [[frontend/05-quality/TESTING]] — resuelve cada línea por separado y lista pendientes de cualquier sesión.

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-F008-01 | crear | Modelo `approvals` con líneas (sesión, descripción, estado) y selección navegable | completada | `internal/tui/approvals.go` | `go build ./internal/tui/` sin errores |
| T-F008-02 | crear | Apertura/cierre del panel (`Ctrl+A`) sin detener el trabajo de ninguna sesión | completada | `keymap.go` (`AccionAprobaciones`), `app.go` | Test: abrir/cerrar no emite ningún control de sesión |
| T-F008-03 | crear | Alta de línea al recibir `peticion_aprobacion` de cualquier sesión, como líneas independientes | completada | `wire.go`, `approvals.go` | Test: dos sesiones → dos líneas independientes |
| T-F008-04 | crear | Render de la línea con el formato documentado, dejando visible la tercera opción por definir según spec | completada | `approvals.go` (`aprobar | declinar | —`) | Test golden del formato de línea |
| T-F008-05 | crear | Resolución con `a`/`d` sobre la línea seleccionada emitiendo Aprobar/Declinar solo a esa sesión | completada | `app.go` (rama de teclado con el panel abierto) | Tests: resolver una línea no afecta a las demás |
| T-F008-06 | crear | Retiro de línea al recibir `aprobacion_resuelta` y marcado de obsoleta si la sesión terminó | completada | `wire.go` (`MarcarObsoleta` en `estado_sesion`) | Tests: resuelta sale; terminada → obsoleta, no se manda |

Dependencias: T-F008-02..06 dependen de T-F008-01; T-F008-05 depende de T-F008-04; T-F008-06 depende de T-F008-03.
