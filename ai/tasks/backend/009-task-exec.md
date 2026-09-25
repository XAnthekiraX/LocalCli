# 009-task-exec.md

> T-B009 — exec: terminal con lista blanca, límites (120s/10KB) y bloqueo Landlock.
> Acción de esta descomposición: `crear`.

## Referencias

- `ai/docs/backend/02-interfaces/TOOLS.md` [64-93] (herramientas de terminal, controles 1 y 2)
- `ai/docs/backend/03-security/SECURITY.md` [39-47] (terminal bloqueada estructuralmente)
- `ai/docs/backend/DECISIONS.md` [19, 26] (Landlock; límites fijos 120s y 10KB con marca truncado)
- `ai/docs/backend/04-infrastructure/INTEGRATIONS.md` (Landlock Linux 5.13+, degradación documentada)

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-B009-01 | crear | Definir la lista blanca cerrada de comandos (compilar, probar, lint, git de lectura) según TOOLS.md | pendiente | `internal/exec/whitelist.go` | Test: comando fuera de lista → rechazo antes de ejecutar |
| T-B009-02 | crear | Validar el comando parseando argv (no texto plano) para que `sh -c`, tuberías o `tee` no lo salten | pendiente | `internal/exec/validate.go` | Test: `sh -c "rm -rf"` y variantes con pipe → rechazo |
| T-B009-03 | crear | Ejecutar el proceso con límite de 120 segundos y corte con marca de tiempo agotado | pendiente | `internal/exec/run.go` | Test: comando dormilón se corta a los 120s (ajustable en test) |
| T-B009-04 | crear | Capturar stdout/stderr con tope de 10KB y marcar `truncado` al superarlo | pendiente | `internal/exec/output.go` | Test: salida de 11KB llega con truncado=true y 10KB |
| T-B009-05 | crear | Aplicar Landlock de escritura al lanzar el proceso (solo lectura en Linux ≥5.13) | pendiente | `internal/exec/landlock_linux.go` | Test: intento de escribir desde el comando falla en kernel soportado |
| T-B009-06 | crear | Proveer stub de degradación sin Landlock que avisa garantía más débil sin bloquear | pendiente | `internal/exec/landlock_other.go` | `go build` pasa en GOOS no-linux y emite el aviso |
| T-B009-07 | crear | Garantizar que exec nunca escribe archivos del proyecto (control 2) | pendiente | `internal/exec/run.go` | Test: redirección `>` dentro de la frontera sigue sin escribir por Landlock |
| T-B009-08 | crear | Escribir tests del módulo sobre comandos fake | pendiente | `internal/exec/*_test.go` | `go test ./internal/exec/...` pasa |

**Dependencias:** 01→02; {02}→03,04; 03→07; 05↔06 (build tags); {04,05}→07; 07→08. Requiere T-B007 completada.
