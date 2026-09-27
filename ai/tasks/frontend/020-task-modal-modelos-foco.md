> T-F020 — modal de modelos: resaltar el modelo en uso y marcar los que no declaran herramientas.
> Acción de esta descomposición: `actualizar`.

## Referencias

- [[frontend/01-domain/DOMAIN]] — §1 `modals`: `modelsmodal` entrega la elección, no decide.
- [[specs/SPEC-INTERFAZ]] — §Modales: el resaltado arranca en el modelo en uso; los modelos sin herramientas se marcan.
- [[specs/SPEC-OLLAMA-PERFIL]] — aviso, sin bloquear, si el modelo no declara capacidad de herramientas.

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-F020-01 | actualizar | `ModelsModal` recuerda el modelo en uso y enfoca su fila al rellenar la lista (patrón de `SessionsModal`) | completada | `internal/tui/modelsmodal.go`, `internal/tui/app.go` | Test `TestElModalDeModelosEnfocaElModeloEnUso` en verde |
| T-F020-02 | actualizar | Marca «(sin herramientas)» en los modelos que no declaran la capacidad | completada | `internal/tui/modelsmodal.go`, `internal/tui/wire.go` | Test `TestElModalMarcaLosModelosSinHerramientas` en verde |
| T-F020-03 | actualizar | Aviso al elegir un modelo sin herramientas, retirado al elegir uno capaz | completada | `internal/tui/app.go` | Test `TestElegirUnModeloSinHerramientasAvisaSinBloquear` en verde |

Dependencias: T-F020-02..03 dependen de T-F020-01; para el aviso real hace falta T-B018.
