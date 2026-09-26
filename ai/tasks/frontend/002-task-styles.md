> T-F002 — styles: estilos Lip Gloss y render de bloques (razonamiento distinguible de la respuesta).
> Acción de esta descomposición: `crear`.

## Referencias

- [[frontend/FRONTEND]] — `styles.go` en la estructura (sección 2).
- [[frontend/01-domain/DOMAIN]] — §2: razonamiento en vivo, arriba de la respuesta, distinguible y ocultable.
- [[specs/SPEC-INTERFAZ]] — Razonamiento del modelo y reglas (nunca mezclado visualmente con la respuesta).
- [[frontend/05-quality/TESTING]] — unitario: recorte de líneas y formato del bloque de razonamiento.

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-F002-01 | crear | Estilos Lip Gloss base: mensaje de usuario, razonamiento (atenuado/cursiva con prefijo), respuesta (normal), avisos y paneles | completada | `internal/tui/styles.go` | `go build ./internal/tui/` sin errores |
| T-F002-02 | crear | Recorte de líneas al ancho de pantalla (`clipLine`) como función pura | completada | `internal/tui/styles.go` | Test `TestClipLineRecortaAlAncho` en verde |
| T-F002-03 | crear | Render del bloque de razonamiento distinguible y ocultable sin tocar la generación (`renderReasoning`) | completada | `internal/tui/styles.go` | Test `TestRenderRazonamientoDistinguibleYOcultable` en verde |
| T-F002-04 | crear | Render del intercambio razonamiento-arriba-de-respuesta garantizando separación visual | completada | `internal/tui/styles.go` | Test `TestRenderIntercambioRazonamientoAntesQueRespuesta` en verde |
| T-F002-05 | crear | Pruebas unitarias puras de estilos y recorte | completada | `internal/tui/styles_test.go` | `go test ./internal/tui/ -run Styles\|Clip\|Render` en verde |

Dependencias: T-F002-02..04 dependen de T-F002-01; T-F002-05 depende de T-F002-01..04.
