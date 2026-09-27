> T-F022 — modal de atajos: agrupado por categorías y con tecla y descripción alineadas.
> Acción de esta descomposición: `actualizar`.

## Referencias

- [[frontend/01-domain/DOMAIN]] — §1 `modals`: `keysmodal`, solo lectura.
- [[specs/SPEC-INTERFAZ]] — §Modales: tabla de atajos por categorías.

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-F022-01 | actualizar | El modal de atajos agrupa por categorías (General, Chat, Vista, Modales, Aprobaciones, Entrada) y calcula el ancho de la columna de teclas | completada | `internal/tui/keysmodal.go` | Test `TestElModalDeAtajosAgrupaPorCategorías` en verde |
| T-F022-02 | actualizar | Las filas se rellenan a un ancho común para que el centrado no las escalone | completada | `internal/tui/modelsmodal.go` | Test `TestLasFilasDeAtajoAlineanLaDescripción` + dorada `atajos.golden` |
