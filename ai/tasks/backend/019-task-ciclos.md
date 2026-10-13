# 019-task-ciclos.md

> T-B019 — Ciclos: planificación, trabajo y resolver.
> Acción de esta descomposición: `crear`.

## Referencias

- [[specs/SPEC-CICLO-PLANIFICACION]] — el TODO con las fases y el arranque de `build`
- [[specs/SPEC-CICLO-TRABAJO]] — el elemento entra en la cola y `build` lo ejecuta
- [[specs/SPEC-RESOLVER]] — el resolver entrega un PLAN y no implementa
- [[specs/SPEC-COLA-TAREAS]] — detección de trabajo ordenado y arranque de la cola
- [[specs/SPEC-ORQUESTADOR-FLUJOS]] — los flujos como dato y la entrega
- [[backend/01-domain/BUSINESS_RULES]] §Motor de etapas, §Detección de trabajo ordenado, §Cola
- [[backend/02-interfaces/INTERFACES-GENERAL]] §4 — los comandos de flujo
- [[backend/04-infrastructure/EVENTS]] — `notificacion` de trabajo terminado

## Contexto

Los tres ciclos oficiales son flujos de datos y T-B017 ya los corre. Aquí se
conecta cada ciclo con la cola de T-B018: quién detecta el trabajo ordenado y
propone el TODO —sin arrancarlo—, quién lo puebla al terminar la planificación, y
el flujo `/ejecutar` que lo consume un elemento por iteración. El resolver ya es
un flujo y su entrega es el PLAN: no crea tareas ni implementa.

## Tareas pequeñas

| ID        | Acción | Tarea                                                                                          | Estado    | Archivos                                                       | Verificación                                                             |
| --------- | ------ | ---------------------------------------------------------------------------------------------- | --------- | -------------------------------------------------------------- | ------------------------------------------------------------------------ |
| T-B019-01 | crear  | Detección de trabajo ordenado: una petición con lista propone el TODO sin arrancarlo            | completada | `internal/session/deteccion.go`                                | "documentar capa por capa" propone; no ejecuta por sí sola               |
| T-B019-02 | crear  | Flujo oficial `/ejecutar`: arranca la cola de la capa y la consume                              | completada | `internal/arranque/datos/flows/ejecutar.json`                  | `/ejecutar` aparece en el catálogo y en el enum de `ejecutar_flujo`      |
| T-B019-03 | crear  | Bucle de la cola: un elemento por iteración, con contexto, aprobación, verificación y marcado  | completada | `internal/session/cola.go`                                     | `build` ejecuta un elemento y lo marca completado                        |
| T-B019-04 | crear  | Arranque de la cola por confirmación o `/ejecutar`, y cierre al vaciarse con aviso             | completada | `internal/session/cola.go`                                     | Sin confirmación se responde en el chat; la cola vacía avisa y termina   |
| T-B019-05 | crear  | El flujo `/planificar` puebla el TODO con sus fases al cerrar                                   | completada | `internal/arranque/datos/flows/planificar.json`, `internal/session/ciclos.go` | Al terminar la planificación existe el TODO con todas sus fases |
| T-B019-06 | crear  | Los flujos de trabajo (`/crear`, `/actualizar`, `/eliminar`) meten la tarea aprobada en la cola | completada | `internal/arranque/datos/flows/{crear,actualizar,eliminar}.json` | La tarea aprobada entra en la cola en su posición                       |
| T-B019-07 | crear  | El ciclo `/resolver` entrega el PLAN y no crea tareas ni implementa                             | completada | `internal/arranque/datos/flows/resolver.json`, `internal/session/ciclos.go` | `/resolver` cierra con el PLAN y no escribe ni crea tareas      |
| T-B019-08 | crear  | Pruebas de los ciclos: detección, ejecución de la cola y resolver                               | completada | `internal/session/ciclos_test.go`                              | `go test ./internal/session/`                                            |

**Dependencias:** `01`→`03`; `02`,`03`→`04`; `03`→`05`,`06`,`07`; todos → `08`.
**Fuera de alcance (queda para sus tareas):** los módulos `task` y `queue` y la derivación de la cola son T-B018; aquí solo se conectan los ciclos con esa cola.
