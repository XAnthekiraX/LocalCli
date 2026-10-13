# 018-task-tareas-cola.md

> T-B018 — Tareas (`NNN-task-<nombre>.md`) y cola por capa.
> Acción de esta descomposición: `crear`.

## Referencias

- [[specs/SPEC-COLA-TAREAS]] — la cola como TODO de la ejecución, orden y bloqueos
- [[specs/SPEC-CICLO-TRABAJO]] — la acción del elemento y el orden por dependencias
- [[backend/01-domain/DOMAIN]] §`task` y §`queue` (módulos y límites)
- [[backend/01-domain/BUSINESS_RULES]] §Detección de trabajo ordenado, §Cola
- [[backend/04-infrastructure/EVENTS]] — `cola_actualizada`, `elemento_bloqueado`
- [[backend/05-quality/ERRORS]] §5 — `E_ELEMENTO_BLOQUEADO`
- [[backend/04-infrastructure/CONFIGURATION]] §4 — el TODO vive en `ai/tasks/<capa>/`
- [[backend/DECISIONS]] — frontmatter del elemento y cola derivada del archivo

## Contexto

Dos módulos. `task` lee y escribe el TODO de una capa —`MAIN-TASKS.md` y sus
`NNN-task-<nombre>.md`— como elementos con los siete campos del frontmatter
aprobado: `id`, `capa`, `accion`, `estado`, `depende_de`, `bloqueada_por` y
`documentos`. `queue` proyecta ese TODO como cola: ordena por dependencias, marca
lo bloqueado y entrega el siguiente elegible, sin guardar estado propio —el TODO
es la única verdad—. El consumo lo autoriza solo el orquestador: no hay modo de
tarea suelta.

## Tareas pequeñas

| ID        | Acción | Tarea                                                                                              | Estado    | Archivos                              | Verificación                                                                  |
| --------- | ------ | -------------------------------------------------------------------------------------------------- | --------- | ------------------------------------- | ----------------------------------------------------------------------------- |
| T-B018-01 | crear  | Modelo del elemento del TODO con sus siete campos y los vocabularios cerrados de capa/acción/estado | completada | `internal/task/modelo.go`             | El round-trip JSON conserva `id, capa, accion, estado, depende_de, bloqueada_por, documentos` |
| T-B018-02 | crear  | Parseo de las tablas Markdown (`MAIN-TASKS.md` y `NNN-task-<nombre>.md`) hacia elementos             | completada | `internal/task/parseo.go`             | Una fila malformada es error; no se descarta en silencio                      |
| T-B018-03 | crear  | Localización de los archivos del TODO de una capa y extracción del prefijo `NNN` del ID             | completada | `internal/task/ubicacion.go`          | `ai/tasks/backend/` lista su `MAIN-TASKS.md` y sus tareas pequeñas en orden   |
| T-B018-04 | crear  | Validación del TODO: vocabulario, regla de `bloqueada_por`, IDs únicos y dependencias resolubles   | completada | `internal/task/validacion.go`         | `bloqueada_por` sin `estado` bloqueada se rechaza                             |
| T-B018-05 | crear  | Orden topológico por `depende_de` con desempate por prefijo `NNN`; elegibles y dependencias faltantes | completada | `internal/task/orden.go`              | Orden determinista; un ciclo es error; un rango no cruza de capa              |
| T-B018-06 | crear  | Escritura quirúrgica del estado de una fila, atómica y con frontera a `ai/tasks/`                  | completada | `internal/task/escritura.go`          | Solo cambia la celda Estado; el resto del archivo queda byte a byte           |
| T-B018-07 | crear  | Pruebas del parser, el orden y la escritura, con testdata de TODO                                  | completada | `internal/task/task_test.go`, `internal/task/testdata/` | `go test ./internal/task/`                                      |
| T-B018-08 | crear  | Proyección en memoria de la cola: raíz y capa, orden y vista                                       | completada | `internal/queue/cola.go`              | La cola no guarda estado propio; la vista refleja el TODO                     |
| T-B018-09 | crear  | Reconstruir y re-derivar la cola desde el TODO                                                     | completada | `internal/queue/derivar.go`           | Un elemento nuevo aparece en su posición                                      |
| T-B018-10 | crear  | Bloqueos y su motivo, sin detener la cola                                                          | completada | `internal/queue/bloqueo.go`           | `E_ELEMENTO_BLOQUEADO`; la cola sigue si nada depende del bloqueado           |
| T-B018-11 | crear  | Siguiente elegible y el objetivo con el que se le pide contexto                                    | completada | `internal/queue/siguiente.go`         | Un elemento empezado se retoma sin repetir lo completado                      |
| T-B018-12 | crear  | Marcar el estado de un elemento en su archivo de TODO                                              | completada | `internal/queue/marcar.go`            | Reanudar no repite subtareas completadas                                      |
| T-B018-13 | crear  | Consumo autorizado: solo el orquestador lanza la cola, sin modo de tarea suelta                    | completada | `internal/queue/consumo.go`           | Fuera del orquestador, consumir la cola se deniega                            |
| T-B018-14 | crear  | Eventos `cola_actualizada` y `elemento_bloqueado` de alcance de proyecto                           | completada | `internal/session/cola.go`            | La vista recibe los eventos sin que la sesión sea la activa                   |
| T-B018-15 | crear  | Pruebas de la cola y cableado de su derivación al arrancar el proyecto                             | completada | `internal/queue/queue_test.go`, `cmd/localcli/main.go` | `go test ./internal/queue/`; la cola se deriva al arrancar    |

**Dependencias:** `01`→`02`,`03`,`04`,`05`; `02`→`06`; `01`–`06`→`07`; `05`→`08`; `08`→`09`; `09`→`10`,`11`; `06`,`09`→`12`; `08`→`13`,`14`; `12`,`13`,`14`→`15`.
**Fuera de alcance (queda para sus tareas):** quién arranca y consume la cola en cada ciclo —detección de trabajo ordenado, flujo `/ejecutar` y el uso de la cola en planificación, trabajo y resolver— es T-B019. Aquí solo se construyen `task` y `queue`, su consumo autorizado y sus eventos.
