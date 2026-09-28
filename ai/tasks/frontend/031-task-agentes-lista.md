> T-F031 — `Tab` recorre todos los agentes disponibles, no solo `plan` y `build`.
> Acción de esta descomposición: `actualizar`.

## Referencias

- [[specs/SPEC-INTERFAZ]] — §Zonas 2: el indicador muestra el agente activo y `Tab` recorre los disponibles.
- [[specs/SPEC-KEYBINDS]] — §Acción `agent_cycle`.
- [[specs/SPEC-AGENTE-BASE]] — los agentes se cargan de `ai/agents/*.json`.
- [[frontend/01-domain/DOMAIN]] — §3: la vista cicla el agente y actualiza el indicador.

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-F031-01 | actualizar | Puerto: `Agentes() []string` y el adaptador lo sirve desde el arranque | completada | `internal/tui/wire.go`, `arranque.go` | `go build ./...` en verde |
| T-F031-02 | actualizar | `App`: lista de agentes, `ciclarAgente` recorre la lista y el agente recordado se valida contra ella | completada | `internal/tui/app.go`, `internal/tui/config.go`, `internal/tui/input.go` | Tests `TestTabRecorreTodosLosAgentesDisponibles` y `TestValidarAgenteContraLaLista` en verde |
| T-F031-03 | actualizar | Dorada del modal de atajos regenerada (`cambiar de agente`) | completada | `internal/tui/testdata/atajos.golden` | Test `TestLasDoradasDeVistaSeMantienen` en verde |

Dependencias: T-F031-02 depende de T-F031-01.
