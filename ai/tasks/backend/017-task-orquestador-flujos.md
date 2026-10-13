# 017-task-orquestador-flujos.md

> T-B017 — Orquestador de etapas y flujos oficiales; entrada por comando y por la
> herramienta `ejecutar_flujo`.
> Acción de esta descomposición: `crear`.

## Referencias

- [[specs/SPEC-ORQUESTADOR-FLUJOS]] — modelo, ventana, pausas, entrega
- [[specs/SPEC-TOOLS]] §Los flujos como herramienta
- [[specs/SPEC-CICLO-TRABAJO]] — ciclos oficiales
- [[backend/02-interfaces/TOOLS]] §Los flujos como herramienta
- [[backend/04-infrastructure/EVENTS]] §3 (`etapa_*`, `flujo_*`)
- [[database/01-schema/TABLES]] §`flow_context`
- [[backend/05-quality/ERRORS]] §5 (§Flujo)

## Contexto

`internal/flow` ya tenía el **modelo** (flujo, etapas, catálogo, validación). Aquí
se añade el **runner**: recorre las etapas, compone la ventana de cada una (la
petición original + la contribución de las etapas anteriores), corre su agente,
respeta las pausas de `continuacion` y `aprobacion` y publica los eventos.

El runner vive en `session` porque necesita la base, el registro de herramientas,
el emisor de eventos y el canal de aprobación, y porque **el flujo corre como
sub-proceso del turno**: el turno espera mientras el flujo avanza. Devuelve solo
la entrega más el bloque resumido; el interior no cruza.

## Tareas pequeñas

| ID         | Acción | Tarea                                                                                                    | Estado    | Archivos                              | Verificación                                                                  |
| ---------- | ------ | -------------------------------------------------------------------------------------------------------- | --------- | ------------------------------------- | ----------------------------------------------------------------------------- |
| T-B017-01  | crear  | Permiso derivado: `PermisoNecesario` / `PuedeLanzar` (plan → `read`; alguna build → `write`)              | completada | `internal/flow/permiso.go`            | `plan` no lanza un flujo que escribe; `build` sí                           |
| T-B017-02  | crear  | Bloque de contexto del flujo en `flow_context`: guardar, listar y vaciar por corrida                      | completada | `internal/store/flowcontext.go`       | Cada corrida empieza en blanco y guarda una contribución por etapa            |
| T-B017-03  | crear  | Enum de flujos en el esquema: `tools.Esquema.Propiedad.Enum` y el esquema por turno de `ejecutar_flujo`   | completada | `internal/tools/esquema.go`, `internal/session/orquestador.go` | El esquema de `ejecutar_flujo` lista solo los flujos que el agente puede lanzar |
| T-B017-04  | crear  | Runner de etapas: ventana por etapa, agente por etapa, `entrega` sin herramientas, profundidad uno         | completada | `internal/session/flujo.go`           | Las etapas corren en orden; una etapa nunca ve `ejecutar_flujo`               |
| T-B017-05  | crear  | Pausas `continuacion` (antes) y `aprobacion` (después) reutilizando el canal de aprobación                 | completada | `internal/session/flujo.go`           | Aprobar continúa; declinar cancela el flujo                                   |
| T-B017-06  | crear  | `respuesta_en_chat`: la respuesta se pinta sin entrar al bloque                                            | completada | `internal/session/flujo.go`           | La etapa con `respuesta_en_chat` no suma al bloque resumido                   |
| T-B017-07  | crear  | Eventos de etapa y flujo: `etapa_iniciada/terminada/fallida`, `flujo_pausado/reanudado/cancelado`          | completada | `internal/session/flujo.go`           | La vista pinta `[Sub Proceso]` y los avisos de pausa                          |
| T-B017-08  | crear  | Entrada por comando: `Enviar` reconoce `/comando` y arranca el flujo; la entrega se guarda como respuesta  | completada | `internal/session/orquestador.go`     | `/resolver …` corre etapas y deja la entrega como turno                       |
| T-B017-09  | crear  | Handler `ejecutar_flujo`: resuelve, comprueba el permiso derivado y devuelve entrega + bloque              | completada | `internal/session/flujo.go`           | `E_FLOW_UNKNOWN` / `E_FLOW_NOT_ALLOWED`; la salida trae la entrega y el bloque |
| T-B017-10  | crear  | Cableado: cargar el catálogo de flujos (`flow.CargarAditivo`) e inyectarlo; instalar el handler            | completada | `cmd/localcli/main.go`                | Los flujos oficiales sembrados se cargan y aparecen en el enum                |

**Dependencias:** `01`,`02`,`03`→`04`; `04`→`05`,`06`,`07`; `04`,`08`,`09`,`10`.
**Fuera de alcance (queda para sus tareas):**
- Los **documentos del nodo de contexto por etapa** dependen del grafo de docs
  (`docs`, T-B020). Sin él, cada etapa corre con su ventana, su prompt y sus
  herramientas; el `context.Nodo` se deja para cuando exista su `Fuente`.
- La **persistencia de líneas en `chat_evento`** (sub-proceso y herramienta) se
  difiere a cuando la vista las relea al retomar una sesión: hoy nadie las lee.
- La **cola y las tareas** de la capa (`queue`) son T-B018; los **ciclos**, T-B019.
