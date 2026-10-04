> T-F043 — el modal de modelos y la línea de estado nombran al proveedor: `ModeloLocal` gana su proveedor, cada fila del modal lo rotula y el nombre visible del modelo en uso lo lleva incluido.
> Acción de esta descomposición: `actualizar`.

## Referencias

- [[specs/SPEC-INTERFAZ]] — §Modal de modelos y §Caja de entrada (la línea de estado con el modelo y su proveedor).
- [[frontend/01-domain/DOMAIN]] — §1 (`modelsmodal`, `input`) y §2/§3 (línea de modelo y modal).
- [[frontend/FRONTEND]] — §2 estructura de `internal/tui/`.
- [[backend/04-infrastructure/CONFIGURATION]] — §7 `ultimo_proveedor`.
- [[backend/01-domain/DOMAIN]] — de dónde sale el par modelo/proveedor que pinta la TUI.

## Contexto

El modal de modelos muestra hoy solo el nombre del modelo. Con dos proveedores, «el mismo nombre no significa lo mismo» en los dos runtimes, y la spec pide que se sepa contra cuál se habla ([[specs/SPEC-MODELO-PROVEEDOR]]: el nombre del proveedor se ve en el modal y en la línea de modelo).

El contrato del puerto **no cambia de forma**: siguen `Modelos() []ModeloLocal` y `CapacidadesModelo(nombre)`. Lo que cambia es que cada `ModeloLocal` lleva además su proveedor, y que `ModeloActual()` devuelve el nombre con el proveedor incluido (`qwen3:8b (llama.cpp)`). Así la línea de estado no cambia de código: pinta lo que el puerto le da.

Esto no decide el proveedor: lo elige el arranque (T-B036). La TUI solo lo muestra.

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-F043-01 | actualizar | `puerto`: `ModeloLocal` gana `Proveedor string`; `Modelos()` y `CapacidadesModelo(nombre)` conservan su forma | completada | `internal/tui/wire.go` | `go build ./internal/tui/` |
| T-F043-02 | actualizar | `modelsmodal`: cada fila rotula el proveedor del modelo, además del nombre y la marca de los que no declaran herramientas | completada | `internal/tui/modelsmodal.go` | `TestElModalRotulaElProveedor` |
| T-F043-03 | actualizar | `arranque`: `ModeloActual()` devuelve el modelo con su proveedor (`qwen3:8b (llama.cpp)`) para que la línea de estado lo muestre sin lógica nueva | completada | `arranque.go` | `TestModeloActualIncluyeElProveedor` |
| T-F043-04 | actualizar | `input`: la línea de modelo pinta lo que devuelve el puerto, incluido el proveedor; sin cambios de disposición | completada | `internal/tui/input.go` | `go test ./internal/tui/ -count=1` |
| T-F043-05 | actualizar | tests: el modal se abre igual contra los dos proveedores, rotula cada fila y no cambia el resto de la mecánica (`↑`/`↓`, `Enter`, `Esc`) | completada | `internal/tui/modelsmodal_test.go`, `internal/tui/tui_test.go` | `go test ./internal/tui/ -count=1` |
| T-F043-06 | actualizar | Documentación ya actualizada por el comando `/actualizar`: verificar que la spec y la documentación de la capa siguen describiendo el modal y la línea de modelo rotulados | completada | `ai/docs/specs/SPEC-INTERFAZ.md`, `ai/docs/frontend/01-domain/DOMAIN.md`, `ai/docs/frontend/05-quality/TESTING.md` | `go test ./internal/docs/ -count=1` |

Dependencias: T-F043-01 antes de -02 y de -04. T-F043-03 antes de -04. T-F043-05 revisa -02 y -04. T-F043-06 revisa el conjunto.

## Fuera de alcance

- **No se añade un selector de proveedor.** El proveedor se elige al arrancar (T-B036); el modal elige modelo dentro del proveedor ya elegido.
- **No cambia la mecánica del modal** (líder, `↑`/`↓`, `Enter`, `Esc`) ni la disposición de la TUI.
- **No se persiste nada desde la vista**: `ultimo_proveedor` lo escribe el arranque; la TUI solo lo muestra.
- **No se toca la base de datos.**
