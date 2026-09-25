> T-F000 — Verificación del estado base: `internal/tui/` con su testdata/logo.txt dorado y compilación del paquete.
> Acción de esta descomposición: `crear`.

## Referencias

- [[frontend/FRONTEND]] — estructura de `internal/tui/` (sección 2).
- [[specs/SPEC-INTERFAZ]] — arte canónico del logotipo (6 filas × 53 columnas).
- [[frontend/05-quality/TESTING]] — comparación byte a byte contra `testdata/logo.txt`.

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-F000-01 | crear | Inicializar módulo Go `localcli` con dependencias Bubble Tea, Lip Gloss y Bubbles | completada | `go.mod`, `go.sum` | `go build ./...` sin errores |
| T-F000-02 | verificar | Dorado `logo.txt`: 6 líneas UTF-8 × 53 columnas, sin espacios finales | completada | `internal/tui/testdata/logo.txt` | `awk 'END{print NR==6}'` y grep sin espacios finales |
| T-F000-03 | crear | Placeholder mínimo que compile del paquete `internal/tui` (doc.go) | completada | `internal/tui/doc.go` | `go vet ./internal/tui/` sin errores |

Dependencias: T-F000-02 y T-F000-03 dependen de T-F000-01; ninguna otra.
