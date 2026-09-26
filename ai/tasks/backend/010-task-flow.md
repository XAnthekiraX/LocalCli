# 010-task-flow.md

> T-B010 — flow: motor de etapas, detección de trabajo ordenado y encadenamiento de ciclos.
> Acción de esta descomposición: `crear`.

## Referencias

- `ai/docs/backend/01-domain/BUSINESS_RULES.md` [24-59] (motor de etapas y detección de trabajo ordenado)
- `ai/docs/specs/SPEC-CICLO-TRABAJO.md` [21-127] (un flujo tres entradas; esqueleto paso a paso)
- `ai/docs/specs/SPEC-CICLO-PLANIFICACION.md` (ciclo de planificación)
- `ai/docs/specs/SPEC-RESOLVER.md` (ciclo resolver)
- `ai/docs/backend/04-infrastructure/EVENTS.md` [5-61] (eventos del motor)

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-B010-01 | crear | Definir el tipo Etapa y la secuencia encadenable entre estados seguir/parar/esperar-aprobación | completada | `internal/flow/stage.go` | Test: transiciones inválidas se rechazan |
| T-B010-02 | crear | Implementar la detección de "trabajo ordenado" en una petición del usuario | completada | `internal/flow/detect.go` | Test: petición con lista ordenada → detecta; conversación normal → no |
| T-B010-03 | crear | Generar el TODO de la ejecución (MAIN-TASKS) al detectar trabajo, usando task | completada | `internal/flow/todo.go` | Test: tras detectar, el TODO existe y task lo relee |
| T-B010-04 | crear | Encadenar las etapas del ciclo de planificación pidiendo contexto a context y agente plan | completada | `internal/flow/plan.go` | Test: con stubs, el plan termina sin ninguna escritura |
| T-B010-05 | crear | Encadenar las etapas del ciclo de trabajo con relevo plan→build y aprobaciones | completada | `internal/flow/work.go` | Test: build solo arranca tras aprobar el plan |
| T-B010-06 | crear | Encadenar el ciclo resolver según SPEC-RESOLVER | completada | `internal/flow/resolver.go` | Test: la secuencia de agentes coincide con la documentada |
| T-B010-07 | crear | Decidir continuar/parar/esperar tras cada etapa y emitir los eventos de EVENTS.md | completada | `internal/flow/engine.go` | Test: cada decisión emite su evento con payload correcto |
| T-B010-08 | crear | Aplicar la regla de un elemento por iteración al consumir la cola | completada | `internal/flow/engine.go` | Test: nunca hay dos elementos en_progreso a la vez |
| T-B010-09 | crear | Escribir tests del motor con stubs de context/agent/task | completada | `internal/flow/*_test.go` | `go test ./internal/flow/...` pasa |

**Dependencias:** 01→{02,07}; 02→03; {03}→05; 01→04; 04→05; {05}→06; 07→08; {06,08}→09. Requiere T-B003, T-B004, T-B005, T-B006 completadas.
