> T-F034 — panel: una sección «TODO DEL AGENTE» muestra la lista de pasos de la sesión activa y se repinta con el evento `todo_actualizada`.
> Acción de esta descomposición: `actualizar`.

## Referencias

- [[specs/SPEC-INTERFAZ]] — §Zonas 3 (panel de datos) y §Criterios de aceptación.
- [[specs/SPEC-TOOLS]] — la lista de pasos de la sesión.
- [[frontend/02-interfaces/INTERFACES]] — `todo_actualizada` y las lecturas a `store`.
- [[frontend/05-quality/TESTING]] — qué se prueba del panel.
- [[backend/04-infrastructure/EVENTS]] — `todo_actualizada`.

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-F034-01 | actualizar | `panel`: campo `Tareas` y sección «TODO DEL AGENTE» con `[•]`/`[✓]`/`[x]`/`[ ]`, oculta sin nada accionable | completada | `internal/tui/panel.go` | Test `TestElPanelPintaLaListaDePasosDelAgente` en verde |
| T-F034-02 | actualizar | `wire`: tipo `TareaPanel`, evento `todo_actualizada` y `Puerto.Tareas`; la carga viaja con el historial | completada | `internal/tui/wire.go`, `internal/tui/app.go`, `internal/tui/tui_test.go` | Test de que la lista de otra sesión no entra al panel |
| T-F034-03 | actualizar | documentar la sección en `SPEC-INTERFAZ` §Zonas 3, `INTERFACES` y `TESTING` | completada | `ai/docs/specs/SPEC-INTERFAZ.md`, `ai/docs/frontend/02-interfaces/INTERFACES.md`, `ai/docs/frontend/05-quality/TESTING.md` | Documentación coherente con el código |

Dependencias: T-F034-01 antes de T-F034-02; T-F034-03 revisa las dos.

## Notas

- La lista se pinta debajo de los nueve datos del panel; si no queda ningún paso accionable (vacía, o todo completado/cancelado), la sección no se pinta.
- La sección es de la sesión activa: el evento de otra sesión se descarta, igual que el estado y el historial.
