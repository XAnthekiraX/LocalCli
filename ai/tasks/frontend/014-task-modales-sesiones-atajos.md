> T-F014 — modales de sesiones y de atajos: `Ctrl+X l` abre la lista de sesiones y `Enter` abre la elegida; `Ctrl+P` abre el listado de atajos; `Esc` cierra cualquier modal.
> Acción de esta descomposición: `actualizar` (sobre T-F007 sessions y T-F010 app).

## Referencias

- [[specs/SPEC-INTERFAZ]] — §Modales: los tres modales comparten mecánica; tabla de apertura y aplicación.
- [[specs/SPEC-KEYBINDS]] — acciones `session_picker`, `command_palette` y `dismiss`.
- [[frontend/01-domain/DOMAIN]] — componente `modals`.
- [[frontend/05-quality/TESTING]] — pruebas de los modales.

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-F014-01 | actualizar | Convertir el selector momentáneo de sesiones en modal (`sessionsmodal.go` a partir de `sessions.go`): centrado, ↑/↓ navega, `Enter` abre la sesión seleccionada y cierra, `Esc` cierra sin cambios | pendiente | `internal/tui/sessions.go`→`sessionsmodal.go`, `app.go`, testdata | Test de componente: elegir sesión produce la petición de cambio de sesión; `Esc` no cambia nada |
| T-F014-02 | crear | Modal de atajos (`keysmodal.go`): tabla acción + tecla(s) generada desde el keymap vigente (incluidas las secuencias con líder); solo lectura, `Esc` lo cierra | pendiente | `internal/tui/keysmodal.go` | Golden: el listado contiene cada acción del mapa con su literal; sin navegación que altere estado |
| T-F014-03 | actualizar | Enrutar `AccionModalSesiones` (`ctrl+x l`) y `AccionModalAtajos` (`ctrl+p`), y un `dismiss` único que cierre cualquiera de los tres modales; solo un modal abierto a la vez | pendiente | `internal/tui/app.go` | Test: abrir modelos y luego sesiones deja uno solo visible; `esc` desde cada modal devuelve la vista previa |
| T-F014-04 | actualizar | Lista de sesiones bajo demanda al abrir el modal (lecturas a `store` por `tea.Cmd`, estados «cargando…» y «sin sesiones») | pendiente | `internal/tui/app.go`, `sessionsmodal.go` | Test: abrir emite una sola lectura; la respuesta rellena el modal sin bloquear la escritura |

Dependencias: T-F014 consume T-F012 (KeyResolver). Comparte la mecánica de modal creada en T-F013-02.
