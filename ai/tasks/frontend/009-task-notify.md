> T-F009 — notify: línea de aviso de aprobaciones pendientes visible con el panel cerrado.
> Acción de esta descomposición: `crear`.

## Referencias

- [[frontend/01-domain/DOMAIN]] — §1 `notify` (único dato fuera del panel) y §2 (visible y actualizado con cada evento aunque el panel esté cerrado).
- [[specs/SPEC-INTERFAZ]] — «El aviso que no se puede ocultar»: línea discreta con cuántas aprobaciones esperan.
- [[frontend/02-interfaces/INTERFACES]] — §1 eventos `peticion_aprobacion` / `aprobacion_resuelta` / `notificacion` que alimentan el contador.
- [[frontend/05-quality/TESTING]] — unitario: formato del contador.

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-F009-01 | crear | Función pura de formato del contador («N aprobaciones esperando tu decisión», singular/plural, nada si es 0) | completada | `internal/tui/notify.go` (`formatoAvisoPendientes`) | Test: formatos 0/1/N correctos |
| T-F009-02 | crear | Contador derivado de `peticion_aprobacion` (+1) y `aprobacion_resuelta` (−1) por sesión, incluidas sesiones en segundo plano | completada | `wire.go` (contador ya alimentado en T-F006-06) | Test: secuencia de eventos deja el recuento exacto |
| T-F009-03 | crear | Render de la línea discreta solo cuando el panel de datos está cerrado y el contador es > 0 | completada | `app.go` (`viewPrincipal`), `panel.go` | Test: visible con panel cerrado; abierto, el dato vive solo en su fila |

Dependencias: T-F009-02 y T-F009-03 dependen de T-F009-01.
