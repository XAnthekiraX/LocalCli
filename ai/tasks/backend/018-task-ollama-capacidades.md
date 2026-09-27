> T-B018 — ollama: capacidades del modelo (`/api/show`) y su exposición a la TUI.
> Acción de esta descomposición: `actualizar`.

## Referencias

- [[backend/04-infrastructure/INTEGRATIONS]] — Ollama: `/api/show` y capacidades.
- [[specs/SPEC-OLLAMA-PERFIL]] — aviso si el modelo no declara capacidad de herramientas.
- [[backend/BACKEND]] — `ollama` habla con el modelo; `tui` no importa `ollama`.

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-B018-01 | actualizar | `Capacidades(ctx, modelo)` sobre `/api/show` y `PuedeUsarHerramientas` | completada | `internal/ollama/show.go` | Test `TestCapacidadesLeeLaFichaDelModelo` en verde |
| T-B018-02 | actualizar | `Adaptador.Modelos()` rellena `SinHerramientas` (fichas en paralelo, con tope; un fallo no alarma) | completada | `arranque.go` | `go build ./...` sin errores |
| T-B018-03 | actualizar | `Adaptador.CapacidadesModelo` da la capacidad del modelo en uso a la línea de estado | completada | `arranque.go` | `go build ./...` sin errores |

Dependencias: T-B018-02 depende de T-B018-01; la TUI lo consume en T-F020.
