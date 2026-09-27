> T-F016 — modal de sesiones en la bienvenida: `Ctrl+X l` lo abre desde la bienvenida y, al elegir una sesión, la vista pasa a la principal con su historial.
> Acción de esta descomposición: `actualizar` (sobre T-F003 welcome y T-F010 app).

## Referencias

- [[specs/SPEC-INTERFAZ]] — §Pantalla de bienvenida y §Modales: el modal de sesiones se abre también desde la bienvenida.
- [[frontend/01-domain/DOMAIN]] — §1 `welcome`/`modals` y §3 reglas de la bienvenida.
- [[frontend/02-interfaces/INTERFACES]] — §2 peticiones y §4 teclado.

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-F016-01 | actualizar | Permitir `session_picker` en la bienvenida y abrir el modal de sesiones con `Ctrl+X l` | completada | `internal/tui/keys.go`, `internal/tui/welcome.go` | Test: `Ctrl+X l` en la bienvenida abre el modal de sesiones |
| T-F016-02 | actualizar | Al elegir una sesión desde la bienvenida, pasar a la vista principal con el historial de esa sesión | completada | `internal/tui/app.go` | Test: elegir una sesión cambia a la vista principal y carga su historial |
| T-F016-03 | actualizar | Cubrir el modal de sesiones en la bienvenida y la transición con pruebas | completada | `internal/tui/welcome_test.go`, `internal/tui/sesiones_ctrl_test.go` | Test: la bienvenida con el modal abierto y tras elegir sesión |

Dependencias: T-F016-02 depende de T-F016-01.
