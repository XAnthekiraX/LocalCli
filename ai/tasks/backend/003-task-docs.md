# 003-task-docs.md

> T-B003 — docs: carga de documentación, frontmatter y grafo en memoria.
> Acción de esta descomposición: `crear`.

## Referencias

- `ai/docs/backend/BACKEND.md` [22, 45-48] (módulo docs y grafo en memoria)
- `ai/docs/backend/01-domain/DOMAIN.md` [33-48, 85] (entidades en archivos; no hay tabla de grafo)
- `ai/docs/backend/DECISIONS.md` [16] (grafo en memoria, reconstruido al arrancar)
- `ai/docs/database/02-rules/DATA_FLOW.md` (relación documentación ↔ contexto)

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-B003-01 | crear | Recorrer `ai/docs/**/*.md` del proyecto abierto y cargar el contenido en memoria | pendiente | `internal/docs/loader.go` | Test: sobre fixtures devuelve todos los .md encontrados |
| T-B003-02 | crear | Parsear el frontmatter YAML de cada documento (etiquetas y metadatos) | pendiente | `internal/docs/frontmatter.go` | Test: documento con/sin frontmatter parsea correcto |
| T-B003-03 | crear | Extraer los wiki-links `[[...]]` del cuerpo como aristas de dependencia | pendiente | `internal/docs/links.go` | Test: enlaces rotos y válidos se listan por separado |
| T-B003-04 | crear | Construir el grafo dirigido en memoria al arrancar a partir de frontmatter + enlaces | pendiente | `internal/docs/graph.go` | Test: vecinos de un nodo fixture coinciden con lo declarado |
| T-B003-05 | crear | Exponer API de consulta: documentos por etiqueta, dependencias directas y transitivas | pendiente | `internal/docs/query.go` | Test: cierre transitivo de un fixture es el esperado |
| T-B003-06 | crear | Reportar errores de parseo por archivo sin abortar la carga (códigos de ERRORES.md) | pendiente | `internal/docs/errors.go` | Test: YAML inválido produce error localizado y sigue la carga |
| T-B003-07 | crear | Escribir tests del módulo con fixtures en `testdata/` | pendiente | `internal/docs/*_test.go`, `internal/docs/testdata/` | `go test ./internal/docs/...` pasa |

**Dependencias:** 01→02,03; {02,03}→04; 04→05; 01→06; {05,06}→07. Requiere T-B001 completada.
