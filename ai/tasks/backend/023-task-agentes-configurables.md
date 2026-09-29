> T-B023 — agentes configurables: cualquier `.localcli/agents/*.json` se carga y está disponible.
> Acción de esta descomposición: `actualizar`.

## Referencias

- [[specs/SPEC-AGENTE-BASE]] — §Alcance y §Dónde se definen: todos los `.localcli/agents/*.json` se cargan; un agente propio está disponible sin tocar el código.
- [[specs/SPEC-MOTOR-FLUJOS]] — la etapa declara su agente; el motor no decide el reparto.
- [[backend/DECISIONS]] — el catálogo efectivo se deriva de los permisos.
- [[backend/04-infrastructure/CONFIGURATION]] — la vista sigue viva aunque falte o falle un archivo.

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-B023-01 | actualizar | `agent`: `OrdenarNombres` para un orden estable (base primero, resto alfabético) | completada | `internal/agent/loader.go` | Test `TestOrdenarNombresPoneLosBasePrimero` en verde |
| T-B023-02 | actualizar | Arranque: `agenteBase` carga todos los `.localcli/agents/*.json` (un archivo inválido se ignora; `plan`/`build` siempre, con respaldo) y expone `Adaptador.Agentes()` | completada | `arranque.go` | Test `TestAgenteBaseCargaAgentesPropios` en verde |
| T-B023-03 | actualizar | `flow`: la validación de etapa y de chat exige solo un agente con nombre; el ejecutor rechaza el desconocido en tiempo de ejecución | completada | `internal/flow/stage.go`, `internal/flow/engine.go` | Tests `TestFlujoValidar` y `TestConversarUsaElAgenteActivo` en verde |

Dependencias: T-B023-02 depende de T-B023-01; T-B023-03 depende de T-B023-01.
