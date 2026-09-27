> T-F025 — línea de estado bajo el input: modelo en uso y acceso a herramientas.
> Acción de esta descomposición: `actualizar`.

## Referencias

- [[frontend/01-domain/DOMAIN]] — §1 `input`: bajo la línea se muestra el modelo y sus herramientas.
- [[specs/SPEC-INTERFAZ]] — zona 2: línea de estado.
- [[specs/SPEC-OLLAMA-PERFIL]] — el modelo y su capacidad de herramientas se informan.

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-F025-01 | actualizar | La vista consulta al puerto si el modelo en uso tiene herramientas y lo refleja (`sí`/`no`/`?`) | completada | `internal/tui/app.go`, `internal/tui/wire.go` | Test `TestLaLineaDeEstadoMuestraModeloYHerramientas` en verde |
| T-F025-02 | actualizar | Elegir modelo actualiza la línea de estado al instante | completada | `internal/tui/app.go` | Test `TestElegirModeloActualizaLaLineaDeEstado` en verde |
| T-F025-03 | actualizar | El motor expone la capacidad del modelo (`Adaptador.CapacidadesModelo`) | completada | `arranque.go` | `go build ./...` sin errores |
