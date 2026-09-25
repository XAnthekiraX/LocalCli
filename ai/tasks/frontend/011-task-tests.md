> T-F011 — Pruebas y validaciones: unitarias de render, de componente con arnés Bubble Tea y verificación de criterios de aceptación.
> Acción de esta descomposición: `crear`.

## Referencias

- [[frontend/05-quality/TESTING]] — niveles, reglas para nuevos tests (dorados, sin `sleep`, una prueba una regla) y checklist final.
- [[specs/SPEC-INTERFAZ]] — 17 criterios de aceptación de la disposición y bienvenida.
- [[specs/SPEC-INTERFAZ-ATAJOS]] — 6 criterios de aceptación de atajos y aprobaciones.
- [[frontend/FRONTEND]] — límites de la capa que las pruebas no deben violar.

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-F011-01 | crear | Suite unitaria de funciones puras pendientes: formato del contador y recorte/formato ya iniciados en 000-002 | pendiente | `internal/tui/*_test.go` | `go test ./internal/tui/ -short` en verde |
| T-F011-02 | crear | Arnés de pruebas de componente (programa Bubble Tea in-process con msgs sintéticos, sin base ni red) reutilizable | pendiente | `internal/tui/testing_test.go` | Test propio del arnés: inyecta un msg y lee la view |
| T-F011-03 | crear | Tests de componente por recorrido clave: bienvenida→envío→chat, cambio de sesión en vivo, razonamiento ocultable, panel plegable | pendiente | `internal/tui/app_test.go` | `go test ./internal/tui/ -run Componente` en verde |
| T-F011-04 | crear | Salidas doradas nuevas (vistas de panel, ayuda, selector, aprobaciones) generadas y revisadas a la vista | pendiente | `internal/tui/testdata/*.golden` | Test golden en verde; diff vacío |
| T-F011-05 | verificar | Pasada de verificación de los 23 criterios de aceptación de SPEC-INTERFAZ y SPEC-INTERFAZ-ATAJOS contra las pruebas existentes | pendiente | `ai/tasks/frontend/011-task-tests.md` | Cada criterio enlazado a un test o marcado bloqueado con motivo |
| T-F011-06 | crear | Gates finales: `go vet ./internal/tui/`, `go test ./internal/tui/` y lint si está disponible | pendiente | — | Los tres comandos salen sin errores |

Dependencias: T-F011-02 depende de T-F011-01; T-F011-03 depende de T-F011-02; T-F011-04 depende de T-F011-03; T-F011-05 depende de T-F011-04; T-F011-06 depende de todas. Requiere T-F010 completada.
