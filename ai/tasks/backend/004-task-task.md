# 004-task-task.md

> T-B004 — task: lectura/escritura del TODO (MAIN-TASKS.md y NNN-task-*.md) con su frontmatter.
> Acción de esta descomposición: `crear`.

## Referencias

- `ai/docs/backend/01-domain/DOMAIN.md` [38] (entidad TODO en MAIN-TASKS.md)
- `ai/docs/backend/DECISIONS.md` [28] (frontmatter del elemento: id, capa, accion, estado, depende_de, bloqueada_por, documentos)
- `ai/docs/specs/SPEC-CICLO-TRABAJO.md` [37-101] (estructura del TODO y entradas crear/actualizar/eliminar)
- `ai/docs/backend/BACKEND.md` [49] (ruta ai/tasks/ dentro del proyecto)

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-B004-01 | crear | Definir el tipo Elemento con los 7 campos del frontmatter aprobado | pendiente | `internal/task/model.go` | Test: JSON/YAML round-trip conserva los campos |
| T-B004-02 | crear | Parsear tablas Markdown de MAIN-TASKS.md y NNN-task-*.md hacia Elementos | pendiente | `internal/task/parse.go` | Test: parsea el MAIN-TASKS.md real sin perder filas |
| T-B004-03 | crear | Localizar los archivos del TODO bajo `ai/tasks/<capa>/` por prefijo NNN | pendiente | `internal/task/locate.go` | Test: encuentra 000-task-*.md tras T-B000..T-B015 |
| T-B004-04 | crear | Escribir el estado de un elemento (pendiente/en_progreso/completada/bloqueada) preservando el resto del archivo | pendiente | `internal/task/write.go` | Test: cambiar solo el estado no altera otras celdas |
| T-B004-05 | crear | Validar `bloqueada_por` presente solo cuando `estado = bloqueada` | pendiente | `internal/task/validate.go` | Test: violación devuelve error de validación |
| T-B004-06 | crear | Resolver dependencias (`depende_de`) y orden por prefijo NNN como desempate | pendiente | `internal/task/deps.go` | Test: orden topológico estable sobre fixture |
| T-B004-07 | crear | Escribir tests del módulo con fixtures de TODO | pendiente | `internal/task/*_test.go`, `internal/task/testdata/` | `go test ./internal/task/...` pasa |

**Dependencias:** 01→02; 02→{03,04}; 04→05; {02}→06; {05,06}→07. Requiere T-B001 completada.
