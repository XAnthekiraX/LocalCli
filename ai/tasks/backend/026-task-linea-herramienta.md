> T-B026 — línea de herramienta: `herramienta_invocada` lleva el verbo y el tema (el objetivo declarado por el catálogo: la ruta, el patrón, el comando) en vez de un resumen ciego de argumentos, y `herramienta_resultado` lleva la medida del resultado.
> Acción de esta descomposición: `actualizar`.

## Referencias

- [[specs/SPEC-TOOLS]] — §Contrato: mientras una herramienta se ejecuta se ve cuál es.
- [[backend/02-interfaces/TOOLS]] — §1 y §8: el verbo, el tema y la unidad de cada herramienta, y la secuencia de la capa universal.
- [[backend/02-interfaces/dto/TOOLS-DTO]] — §4: de dónde sale la medida del resultado.
- [[backend/04-infrastructure/EVENTS]] — §3: los payloads de `herramienta_invocada` y `herramienta_resultado`.
- [[backend/DECISIONS]] — la línea muestra el tema declarado, no los argumentos en crudo.
- [[frontend/02-interfaces/INTERFACES]] — §1.1: la línea de herramienta que pinta la TUI.

## Contexto

T-B024-17 emitió los dos eventos con un «resumen de argumentos» que solo decía los nombres de campo y su tamaño (`ruta: 4 car.`): ocultaba justo el dato que el usuario necesita —sobre qué actúa la herramienta—. Esta tarea cambia el payload: el catálogo declara un **verbo**, un **tema** (el campo cuyo valor es el objetivo) y una **unidad** (cómo se mide el resultado), y la capa universal calcula el tema y la medida. Sigue sin exponerse el resto de argumentos —cuerpos de archivo, cambios— ni la salida.

No hay migración ni cambio de esquema: los eventos viven solo en memoria.

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-B026-01 | actualizar | `tools`: añadir `Verbo`, `Tema` y `Unidad` a `Herramienta` y rellenarlos en las catorce del catálogo | completada | `internal/tools/catalog.go` | El catálogo sigue en catorce y con sus acciones intactas |
| T-B026-02 | actualizar | `tools`: `VerboDe`, `TemaDe` (solo el campo declarado, colapsado y recortado) y `MedidaDe`; `Registro.Ejecutar` emite verbo/tema/medida y `Publicador` estrena firma | completada | `internal/tools/def.go` | Tests `TestTemaDeSoloExponeElCampoDeclarado`, `TestTemaDeRecortaLoEnorme`, `TestMedidaDeCuentaEnSuUnidad` en verde |
| T-B026-03 | actualizar | cableado: `publicadorBus` emite `verbo`, `tema` y `medida` en el bus | completada | `herramientas.go` | Test `TestEventosDeTodaEjecucion` en verde |
| T-B026-04 | actualizar | Tests: `registry_test` al payload nuevo y unitarios de tema y medida | completada | `internal/tools/registry_test.go`, `internal/tools/def_test.go` | `go test ./internal/tools/...` en verde |
| T-B026-05 | actualizar | Documentar: payloads en `EVENTS`, el verbo/tema/unidad en `TOOLS` y `TOOLS-DTO`, y la decisión en `DECISIONS` | completada | `ai/docs/backend/04-infrastructure/EVENTS.md`, `ai/docs/backend/02-interfaces/TOOLS.md`, `ai/docs/backend/02-interfaces/dto/TOOLS-DTO.md`, `ai/docs/backend/DECISIONS.md` | Documentación coherente con el código |

Dependencias: T-B026-01 antes de T-B026-02. T-B026-02 antes de T-B026-03. T-B026-03 antes de T-B026-04. T-B026-05 revisa las dos.

## Fuera de alcance

- **No se persiste nada nuevo.** Los mensajes de herramienta siguen sin guardarse.
- **No se toca el indicador en vivo de la TUI** (`[⠋ Usando herramienta: X]`): solo cambia la línea del chat.
- **El resto de argumentos no viaja.** Solo el tema declarado por el catálogo; cuerpo de archivo, reemplazos y credenciales quedan fuera.
