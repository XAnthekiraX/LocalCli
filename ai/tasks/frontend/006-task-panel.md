> T-F006 — panel: panel de datos plegable con los nueve datos de la sesión activa.
> Acción de esta descomposición: `crear`.

## Referencias

- [[frontend/01-domain/DOMAIN]] — §1 `panel` (lectura, plegable) y §2 (regla de estimación marcada como tal).
- [[specs/SPEC-INTERFAZ]] — zona 3: tabla de los nueve datos (sesión, contexto, TODO, ruta, git, capa y cola, aprobaciones, agente, proyecto) y reglas de apertura/cierre.
- [[specs/SPEC-PANEL-CONTEXTO]] — significado de tokens/porcentaje, marca de estimación y aviso de límite.
- [[frontend/02-interfaces/INTERFACES]] — §1 eventos que alimentan el panel y §3 lecturas a `store`.

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-F006-01 | crear | Modelo `panel` con los nueve campos de datos y estado plegado/expandido | pendiente | `internal/tui/panel.go` | `go build ./internal/tui/` sin errores |
| T-F006-02 | crear | Plegado y expandido (`Ctrl+D`) devolviendo el ancho liberado al chat, sin interrumpir sesiones | pendiente | `internal/tui/panel.go` | Test golden: panel cerrado → chat a todo el ancho |
| T-F006-03 | crear | Render de las filas sesión, ruta, git, agente y proyecto (nombre y versión siempre visibles con el panel abierto) | pendiente | `internal/tui/panel.go` | Test: las cinco etiquetas aparecen en la salida |
| T-F006-04 | crear | Render de contexto: tokens usados y % de ocupación, marcados como estimación cuando lo sean y con aviso al acercarse al límite | pendiente | `internal/tui/panel.go` | Test: dato estimado sale etiquetado; exacto sin etiqueta |
| T-F006-05 | crear | Render de TODO (elemento en curso y restantes) y de capa y cola (capa activa y tareas grandes restantes) | pendiente | `internal/tui/panel.go` | Test golden de ambas filas con datos sintéticos |
| T-F006-06 | crear | Render de aprobaciones esperando decisión y actualización del panel por eventos (`estado_sesion`, `cola_actualizada`, `cambio_aplicado`) reflejando solo la sesión activa | pendiente | `internal/tui/panel.go` | Test: evento de otra sesión no altera los datos mostrados |

Dependencias: T-F006-02..06 dependen de T-F006-01.
