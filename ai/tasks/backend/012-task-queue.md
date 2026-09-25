# 012-task-queue.md

> T-B012 — queue: cola derivada del TODO, orden por dependencias y elementos bloqueados.
> Acción de esta descomposición: `crear`.

## Referencias

- `ai/docs/specs/SPEC-COLA-TAREAS.md` [21-115] (elemento, cola como TODO de la ejecución, flujo, reglas)
- `ai/docs/backend/01-domain/BUSINESS_RULES.md` [45-59] (reglas de la cola)
- `ai/docs/backend/DECISIONS.md` [15, 27] (cola derivada de archivos; solo el motor lanza colas)
- `ai/docs/backend/01-domain/DOMAIN.md` [79, 84] (no hay tabla de cola; proyección en memoria)

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-B012-01 | crear | Definir el tipo Cola como proyección en memoria reconstruida desde task al arrancar | pendiente | `internal/queue/model.go` | Test: no existe ninguna escritura a SQLite desde queue |
| T-B012-02 | crear | Reconstruir la cola leyendo MAIN-TASKS.md y NNN-task-*.md de la ejecución activa | pendiente | `internal/queue/rebuild.go` | Test: fixture de TODO reconstruye los elementos en orden |
| T-B012-03 | crear | Ordenar por dependencias (`depende_de`) con desempate por prefijo NNN | pendiente | `internal/queue/order.go` | Test: ciclo de dependencias → error; orden estable verificado |
| T-B012-04 | crear | Marcar elementos bloqueados cuando su dependencia está bloqueada o incompleta | pendiente | `internal/queue/block.go` | Test: elemento tras uno bloqueado aparece bloqueado con motivo |
| T-B012-05 | crear | Proveer la siguiente tarea ejecutable al flow (un elemento por iteración) | pendiente | `internal/queue/next.go` | Test: next devuelve el primero elegible; vacío si todo bloqueado |
| T-B012-06 | crear | Sincronizar el estado ejecutado de vuelta al archivo del TODO vía task | pendiente | `internal/queue/sync.go` | Test: completar un elemento actualiza la fila del archivo |
| T-B012-07 | crear | Rechazar cualquier lanzamiento de cola que no venga del motor (sin tarea suelta) | pendiente | `internal/queue/api.go` | Test: intento de lanzar elemento suelto → operación denegada |
| T-B012-08 | crear | Escribir tests del módulo con fixtures de TODO | pendiente | `internal/queue/*_test.go`, `internal/queue/testdata/` | `go test ./internal/queue/...` pasa |

**Dependencias:** 01→{02,03}; 03→04; {02,04}→05; 05→06; 06→07; 07→08. Requiere T-B004 y T-B010 completadas.
