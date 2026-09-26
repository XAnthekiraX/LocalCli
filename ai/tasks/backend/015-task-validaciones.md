# 015-task-validaciones.md

> T-B015 — Validaciones finales: suite de tests por regla/invariante y verificación integral.
> Acción de esta descomposición: `crear`.

## Referencias

- `ai/docs/backend/05-quality/TESTING.md` [5-40] (estrategia, qué debe probarse, fixtures y mocks)
- `ai/docs/backend/01-domain/BUSINESS_RULES.md` [97-123] (invariantes y casos especiales)
- `ai/docs/backend/05-quality/VALIDATION.md` (validación de entradas)
- `ai/docs/backend/05-quality/ERRORS.md` [13-77] (errores conocidos y códigos)
- `ai/docs/DOCS-VALIDATION-REPORT.md` (puntos de control de la documentación)

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-B015-01 | crear | Cobertura de pruebas por regla de negocio de BUSINESS_RULES.md (una prueba por regla) | completada | `internal/**/*_test.go` | Matriz regla→test completa; `go test ./... -race` pasa |
| T-B015-02 | crear | Probar las invariantes: nada se escribe sin aprobación previa de plan | completada | `tests/integration_garantia_test.go` | Test E2E con modelo fake: sin aprobación no hay cambios en disco |
| T-B015-03 | crear | Probar invariantes de frontera: ninguna operación sale de la carpeta del proyecto | completada | `tests/integration_frontera_test.go` | Intentos de evasión (path traversal, symlinks, redirecciones) fallan |
| T-B015-04 | crear | Verificar validaciones de entrada según VALIDATION.md en tools y task | completada | `internal/tools/validate_test.go`, `internal/task/validate_test.go` | Cada caso inválido documentado devuelve su error |
| T-B015-05 | crear | Verificar que los errores emitidos coinciden con ERRORES.md (códigos y mensajes) | completada | `tests/integration_errores_test.go` | Catálogo de errores cubierto; ninguno inventado |
| T-B015-06 | crear | Comprobar restricciones estructurales: sin tabla de cola/grafo/usuarios, sin ORM, store único punto de escritura | completada | `tests/arquitectura_test.go` | go vet + test de imports: tui no importa store; nadie más que store abre SQLite |
| T-B015-07 | crear | Ejecutar la suite completa con `-race` y cobertura mínima de TESTING.md | completada | `Makefile` o script `scripts/verify.sh` | `./scripts/verify.sh` termina en verde con informe de cobertura |
| T-B015-08 | crear | Verificación integral final: compilar binario, arrancar contra un proyecto fixture y recorrer los tres ciclos | completada | `tests/e2e_localcli_test.go` | El binario recorre planificación→trabajo→resolver sobre fixture sin errores |

**Dependencias:** {T-B002..T-B014}→01; 01→{02,03,04,05}; {01..05}→06; 06→07; 07→08. Requiere todas las tareas grandes previas completadas.
