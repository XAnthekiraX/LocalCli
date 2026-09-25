> T-F007 — sessions: selector momentáneo con nombre y estado de todas las sesiones del proyecto.
> Acción de esta descomposición: `crear`.

## Referencias

- [[frontend/01-domain/DOMAIN]] — §1 `sessions` (aparece y desaparece, sin lista permanente) y §2 (cambiar de sesión no interrumpe).
- [[specs/SPEC-INTERFAZ]] — «Cambiar de sesión»: selector momentáneo con nombre y estado, incluidas sesiones en segundo plano.
- [[frontend/02-interfaces/INTERFACES]] — §2 petición «Cambiar de sesión» al elegir en el selector; teclado `Ctrl+S`.
- [[database/01-schema/ENUMS]] — los cinco estados pintables (`inactiva`, `trabajando`, `esperando_permiso`, `terminada`, `error`).

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-F007-01 | crear | Modelo `sessions` con lista recibida (nombre + estado) y selección navegable arriba/abajo | pendiente | `internal/tui/sessions.go` | `go build ./internal/tui/` sin errores |
| T-F007-02 | crear | Apertura momentánea (`Ctrl+S`) y cierre (Esc/elección): no ocupa espacio permanente ni deja rastro en la vista | pendiente | `internal/tui/sessions.go` | Test de componente: fuera de estado abierto, view no lo pinta |
| T-F007-03 | crear | Render de cada fila con nombre y estado usando solo los enums de sesión, sin estados inventados | pendiente | `internal/tui/sessions.go` | Test golden: tabla de filas ↔ enums de [[database/01-schema/ENUMS]] |
| T-F007-04 | crear | Emisión del comando «Cambiar de sesión» con el id elegido, sin cancelar ejecuciones en curso | pendiente | `internal/tui/sessions.go` | Test: elegir emite exactamente un cmd de cambio |
| T-F007-05 | crear | Actualización de estados en vivo mientras el selector está abierto (`estado_sesion`) | pendiente | `internal/tui/sessions.go` | Test: evento cambia la etiqueta de esa fila sin cerrar el selector |

Dependencias: T-F007-02..05 dependen de T-F007-01; T-F007-05 depende de T-F007-03.
