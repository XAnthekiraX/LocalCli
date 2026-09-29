> T-B027 — sub-procesos encadenados: cada etapa de un flujo corre sin el historial del chat y devuelve un resumen corto que se pasa a la siguiente; las etapas intermedias sin aprobación no se muestran ni se persisten, y el chat termina con la entrega del último paso.
> Acción de esta descomposición: `actualizar`.

## Referencias

- [[specs/SPEC-MOTOR-FLUJOS]] — §Flujo principal (flujo con objetivo) y §Reglas: el resultado pasa a la etapa siguiente y se muestra el final.
- [[specs/SPEC-RESOLVER]] — la salida estándar y las seis preguntas del resolver.
- [[backend/04-infrastructure/EVENTS]] — §3: los payloads de `etapa_iniciada` y `etapa_terminada`.
- [[backend/DECISIONS]] — sub-procesos encadenados por resúmenes cortos.
- [[frontend/02-interfaces/INTERFACES]] — §1.2: la línea `[Sub Proceso]`.

## Contexto

El motor corría las etapas en orden y sin historial del chat, pero descartaba el resultado de cada una (solo viajaba como `resumen` del evento `etapa_terminada`) y `ejecutorPorTurno` transmitía y persistía la salida de CADA etapa: el chat mostraba nueve globos, la base guardaba nueve mensajes y no había entrega final. Esta tarea implementa el encadenado que la spec ya pedía: cada etapa deja un resumen corto (acotado con el recortador de `tools`) que el motor inyecta en la siguiente; las etapas intermedias sin aprobación corren en silencio y el chat muestra solo `[Sub Proceso] <nombre>` y la entrega del último paso. Un flujo correcto cierra el turno con el estado, sin el mensaje de relleno «El turno terminó.».

No hay migración: el estado de la cadena vive en el motor.

## Tareas pequeñas

| ID        | Acción     | Tarea                                                                                                                                                             | Estado     | Archivos                                                                                                                                                            | Verificación                                                                                  |
| --------- | ---------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------- |
| T-B027-01 | actualizar | `flow`: `PeticionEtapa` (agente, contexto, historial, imágenes, etapa, silenciosa), `Agente.Ejecutar` con esa petición, encadenado de resúmenes y `bloqueResumen` | completada | `internal/flow/engine.go`                                                                                                                                           | Tests `TestCadaEtapaRecibeElResumenDeLaAnterior`, `TestEtapasIntermediasSilenciosas` en verde |
| T-B027-02 | actualizar | `arranque`: `ejecutorPorTurno.Ejecutar` con la petición nueva; una etapa silenciosa no transmite ni persiste                                                      | completada | `arranque.go`                                                                                                                                                       | `go build ./...` en verde                                                                     |
| T-B027-03 | actualizar | `session`: un flujo y una cola correctos cierran el turno cambiando el estado, sin mensaje de relleno                                                             | completada | `internal/session/run.go`, `internal/session/pause.go`                                                                                                              | `TestEnviarInvocaFlowUnaVez` en verde                                                         |
| T-B027-04 | actualizar | `tui`: `etapa_iniciada` pinta `[Sub Proceso] <nombre>`; `etapa_terminada` no deja línea                                                                           | completada | `internal/tui/wire.go`                                                                                                                                              | Tests de `internal/tui` en verde                                                              |
| T-B027-05 | actualizar | `resolver`: regla de resumen por paso y composición final; JSON sincronizado                                                                                      | completada | `internal/flow/resolver.go`, `.localcli/flows/resolver.json`                                                                                                        | `TestElFlujoResolverDelProyectoCoincideConElRespaldo` en verde                                |
| T-B027-06 | actualizar | Documentar: `SPEC-MOTOR-FLUJOS`, `EVENTS`, `INTERFACES` y `DECISIONS`                                                                                             | completada | `ai/docs/specs/SPEC-MOTOR-FLUJOS.md`, `ai/docs/backend/04-infrastructure/EVENTS.md`, `ai/docs/frontend/02-interfaces/INTERFACES.md`, `ai/docs/backend/DECISIONS.md` | Documentación coherente con el código                                                         |

Dependencias: T-B027-01 antes de T-B027-02. T-B027-02 antes de T-B027-03. T-B027-04 y T-B027-05 en paralelo. T-B027-06 revisa el resto.

## Fuera de alcance

- **No se expone el flujo como herramienta del modelo.** El resolver (y los demás oficiales) siguen arrancando solo con su comando explícito.
- **No se añade configuración por etapa.** El silencio se deriva (intermedia y sin aprobación); no hay un campo nuevo en el JSON del flujo.
- **No cambia el catálogo de herramientas ni los permisos.**
