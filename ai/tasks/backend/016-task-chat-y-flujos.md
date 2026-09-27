> T-B016 — chat normal por defecto y flujos por comando explícito: la vista principal responde como conversación con las herramientas del agente activo; solo `/planificar`, `/crear`, `/actualizar`, `/eliminar`, `/resolver` o `/ejecutar` arrancan un flujo.
> Acción de esta descomposición: `actualizar` (sobre T-B010 flow/T-B013 session/T-B006 agent).

## Referencias

- [[specs/SPEC-INTERFAZ]] — §Zonas 2: chat normal, comando explícito y agente por herramientas.
- [[specs/SPEC-MOTOR-FLUJOS]] — los flujos arrancan por comando, no por su cuenta.
- [[specs/SPEC-COLA-TAREAS]] — el TODO se propone y solo se ejecuta por confirmación o `/ejecutar`.
- [[specs/SPEC-TOOLS]] — el chat usa el catálogo del agente activo.
- [[backend/01-domain/BUSINESS_RULES]] — §Detección de trabajo ordenado y §Cola.
- [[backend/02-interfaces/INTERFACES-GENERAL]] — contrato `session`/`flow`.

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-B016-01 | actualizar | Tratar toda petición como chat; solo un comando de flujo arranca etapas | completada | `arranque.go`, `internal/session` | Test: una petición sin comando no dispara `flow` ni `queue` |
| T-B016-02 | actualizar | Reconocer los comandos explícitos `/planificar`, `/crear`, `/actualizar`, `/eliminar`, `/resolver` y `/ejecutar` y enrutarlos a su flujo | completada | `arranque.go`, `internal/flow` | Test: cada comando arranca su flujo; texto desconocido se responde como chat |
| T-B016-03 | actualizar | Convertir la detección de trabajo ordenado en sugerencia: propone el TODO y no lo ejecuta sin confirmación | completada | `internal/flow` | Test: una petición ordenada propone el TODO y no arranca la cola |
| T-B016-04 | actualizar | El chat responde con las herramientas declaradas en el JSON del agente activo | completada | `internal/agent` | Test: `plan` lee y propone; `build` escribe con aprobación |

Dependencias: T-B016-02 depende de T-B016-01; T-B016-03 y T-B016-04 pueden hacerse en paralelo tras T-B016-01.
