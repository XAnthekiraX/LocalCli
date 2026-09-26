# 011-task-context.md

> T-B011 — context: nodo de contexto (selección por modelo, recorte y auditoría).
> Acción de esta descomposición: `crear`.

## Referencias

- `ai/docs/specs/SPEC-NODO-CONTEXTO.md` [21-66] (flujo principal, ejemplo, reglas, criterios)
- `ai/docs/backend/01-domain/BUSINESS_RULES.md` [61-74] (reglas del nodo de contexto)
- `ai/docs/backend/DECISIONS.md` [16] (grafo en memoria desde docs)
- `ai/docs/backend/04-infrastructure/CONFIGURATION.md` [22] (LOCALCLI_CONTEXT_LIMIT)

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-B011-01 | crear | Definir el tipo SolicitudContexto (objetivo de etapa + candidatos desde el grafo de docs) | completada | `internal/context/model.go` | Test: construcción desde fixture del grafo |
| T-B011-02 | crear | Preguntar al modelo qué documentos son relevantes para el objetivo (sin llamarlo por cuenta propia fuera del flujo) | completada | `internal/context/select.go` | Test con stub ollama: devuelve la selección pedida |
| T-B011-03 | crear | Estimar tokens de cada documento candidato | completada | `internal/context/tokens.go` | Test: estimación estable sobre fixture conocido |
| T-B011-04 | crear | Recortar la selección hasta el límite (modelo o LOCALCLI_CONTEXT_LIMIT), priorizando relevancia | completada | `internal/context/trim.go` | Test: selección que excede el límite queda dentro sin vaciar |
| T-B011-05 | crear | Registrar en context_audit qué entró, qué salió y el motivo, una fila por documento y etapa | completada | `internal/context/audit.go` | Test: tras un recorte hay filas de entrada y salida con motivo |
| T-B011-06 | crear | Ensamblar el bloque de contexto final entregable a agent | completada | `internal/context/assemble.go` | Test: el bloque contiene solo los documentos aprobados por el recorte |
| T-B011-07 | crear | Escribir tests del módulo con grafo y modelo simulados | completada | `internal/context/*_test.go`, `internal/context/testdata/` | `go test ./internal/context/...` pasa |

**Dependencias:** 01→{02,03}; {02,03}→04; 04→{05,06}; {05,06}→07. Requiere T-B003 y T-B005 completadas.
