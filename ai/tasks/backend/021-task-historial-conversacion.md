> T-B021 — historial de conversación: el chat recuerda y compacta lo que no cabe.
> Acción de esta descomposición: `actualizar`.

## Referencias

- [[specs/SPEC-HISTORIAL-CONVERSACION]] — §Flujo principal y §Reglas: reconstruir el historial, presupuesto de tokens y compactación por resumen.
- [[specs/SPEC-SESIONES]] — el contexto de una sesión no se comparte con las demás.
- [[specs/SPEC-NODO-CONTEXTO]] — el recorte de contexto reutiliza la estimación de tokens.
- [[backend/BACKEND]] — §3/§4: `session` arma el historial; `flow` lo transporta; `agent` lo antepone al turno.
- [[backend/04-infrastructure/CONFIGURATION]] — preferencias del usuario (`~/.config/localcli/config.json`).

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-B021-01 | actualizar | `flow`: tipo `Mensaje` y roles; `Agente.Ejecutar` y `Motor.Conversar` reciben el historial; las etapas de flujo van sin historial | completada | `internal/flow/engine.go` | Tests `TestConversarPasaElHistorialAlAgente` y `TestEjecutarFlujoNoPasaHistorial` en verde |
| T-B021-02 | actualizar | `agent`: `Ejecutor.Ejecutar` antepone el historial al contexto del turno | completada | `internal/agent/loop.go` | Test `TestElBucleAnteponeElHistorialAlContexto` en verde |
| T-B021-03 | actualizar | `session`: `historialPara` reconstruye el historial y lo compacta por presupuesto, reutilizando el resumen; `Resumidor` inyectable y `PromptResumen` | completada | `internal/session/contexto.go`, `internal/session/bg.go`, `internal/session/run.go` | Tests `TestElHistorialQueCabeSeEnviaEntero`, `TestElHistorialLargoSeCompactaConResumen` y `TestLaCompactacionReutilizaElResumenEntreTurnos` en verde |
| T-B021-04 | actualizar | Adaptador: convierte `flow.Mensaje` a `ollama.Mensaje`, `resumidorPorTurno` sobre el cliente y la FIFO, y presupuesto desde preferencias / `LOCALCLI_CONTEXT_LIMIT` | completada | `arranque.go`, `internal/tui/config.go` | `go build ./...` y tests de `localcli` en verde |
| T-B021-05 | actualizar | Fallback no bloqueante: si el resumidor falla o no existe, se envía solo el tramo reciente | completada | `internal/session/contexto.go` | Test `TestSinResumidorSeEntregaSoloElTramoReciente` en verde |

Dependencias: T-B021-02 depende de T-B021-01; T-B021-03 depende de T-B021-01 y T-B021-05; T-B021-04 depende de T-B021-03.
