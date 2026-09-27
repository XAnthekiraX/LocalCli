> T-F027 — la bienvenida crea la sesión, el título llega por evento y borrar la última vuelve a la bienvenida.
> Acción de esta descomposición: `actualizar`.

## Referencias

- [[specs/SPEC-SESIONES]] — la primera petición desde la bienvenida crea la sesión; título generado por el modelo; borrar la última vuelve a la bienvenida.
- [[specs/SPEC-INTERFAZ]] — §Pantalla de bienvenida y §Cambiar de sesión.
- [[frontend/01-domain/DOMAIN]] — §2/§3: reglas de presentación y de la bienvenida.
- [[frontend/02-interfaces/INTERFACES]] — §1 eventos que consume (`titulo_sesion`) y §2 peticiones (crear sesión).
- [[backend/04-infrastructure/EVENTS]] — payload de `titulo_sesion`.

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-F027-01 | actualizar | La primera petición desde la bienvenida (y un envío sin sesión) crea la sesión con `Crear` en lugar de retomarla | completada | `internal/tui/welcome.go`, `internal/tui/app.go` | Tests `TestEnviarDesdeLaBienvenidaCreaLaSesionYEnvíaUnaVez` y `TestEnviarVacioDesdeLaBienvenidaNoHaceNada` en verde |
| T-F027-02 | actualizar | Al borrar la activa sin sesiones restantes, la vista vuelve a la bienvenida y limpia sesión/chat/modales | completada | `internal/tui/app.go` | Test `TestCtrlDSobreLaUltimaSesionVuelveALaBienvenida` en verde |
| T-F027-03 | actualizar | El evento `titulo_sesion` renombra la sesión en el panel (si es la activa) y en su fila del modal | completada | `internal/tui/wire.go`, `internal/tui/sessionsmodal.go` | Tests `TestElEventoTituloRenombraLaSesionEnPanelYModal` y `TestUnTituloDeOtraSesionNoTocaElPanel` en verde |
| T-F027-04 | actualizar | `ResolverActiva` pasa a ser solo-retomar (nil sin sesiones) y el doble de pruebas lo refleja | completada | `internal/tui/wire.go`, `internal/tui/tui_test.go` | `go test ./internal/tui/` en verde |

Dependencias: T-F027-02 y T-F027-03 dependen de T-B020 (evento y semántica de `ResolverActiva`).
