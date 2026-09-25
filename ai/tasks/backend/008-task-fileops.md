# 008-task-fileops.md

> T-B008 — fileops: operaciones de archivo, frontera de rutas, aprobación y change_history.
> Acción de esta descomposición: `crear`.

## Referencias

- `ai/docs/backend/02-interfaces/TOOLS.md` [60-63] (herramientas de archivo)
- `ai/docs/backend/03-security/SECURITY.md` [25-38] (toda escritura pasa por aprobación; frontera de rutas)
- `ai/docs/database/01-schema/TABLES.md` (change_history: antes/después)
- `ai/docs/backend/DECISIONS.md` [18, 22] (aplica en fileops; historial sobrevive a la sesión)

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-B008-01 | crear | Implementar la frontera de rutas: resolver y rechazar cualquier ruta fuera de la carpeta del proyecto (incluido `..` y symlinks) | pendiente | `internal/fileops/boundary.go` | Test: `/etc/passwd`, `../x` y symlink externo → rechazo |
| T-B008-02 | crear | Implementar lectura de archivo con límite de tamaño y paginación según TOOLS-DTO | pendiente | `internal/fileops/read.go` | Test: archivo fixture devuelve contenido y metadatos exactos |
| T-B008-03 | crear | Implementar listado/búsqueda de archivos y directorios dentro de la frontera | pendiente | `internal/fileops/list.go` | Test: patrón sobre testdata coincide con lo esperado |
| T-B008-04 | crear | Crear la fila de approvals y esperar decisión antes de aplicar cualquier escritura | pendiente | `internal/fileops/approval.go` | Test: write sin aprobación no toca el disco; declinada tampoco |
| T-B008-05 | crear | Aplicar escritura/edición de archivo solo tras aprobación, calculando el diff | pendiente | `internal/fileops/write.go` | Test: aprobada crea/modifica el archivo pedido |
| T-B008-06 | crear | Aplicar creación/eliminación de carpetas y movimiento con la misma regla de aprobación | pendiente | `internal/fileops/mkdir.go`, `internal/fileops/move.go` | Test: cada operación deja el sistema como lo aprobado |
| T-B008-07 | crear | Registrar cada cambio aplicado en change_history vía store (antes, después, sesión) | pendiente | `internal/fileops/history.go` | Test: tras write aprobada existe la fila con antes/después |
| T-B008-08 | crear | Escribir tests del módulo con proyecto temporal aislado | pendiente | `internal/fileops/*_test.go` | `go test ./internal/fileops/...` pasa |

**Dependencias:** 01→{02,03}; {02}→04; 04→{05,06}; 05→07; 07→08. Requiere T-B002 y T-B007 completadas.
