> T-F030 — token `[nombre.ext]` al pegar o arrastrar una imagen.
> Acción de esta descomposición: `actualizar`.

## Referencias

- [[specs/SPEC-INTERFAZ]] — §Reglas: al pegar o arrastrar una imagen la línea muestra `[nombre.ext]` resaltado y al enviar se usa la ruta real.
- [[frontend/01-domain/DOMAIN]] — §3: la vista pinta y no decide; el input compone y envía.

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-F030-01 | actualizar | `RutasImagen` (detección reutilizable) y tipo `adjuntos` con `Anotar`/`Expandir`/`Resaltar`/`Olvidar`; estilo `estiloAdjunto` | completada | `internal/tui/adjuntos.go`, `internal/tui/styles.go` | Tests `TestAdjuntosAnotaYExpandeVariasRutas` y `TestAdjuntosNoMezclaDosArchivosConElMismoNombre` en verde |
| T-F030-02 | actualizar | `Entrada`: guarda adjuntos, expande en `Texto`, resalta en `View` y los olvida al limpiar/`FijarTexto`; `actualizarEntrada` convierte el pegado en tokens | completada | `internal/tui/input.go`, `internal/tui/app.go` | Test `TestPegarUnaImagenMuestraElTokenYEnviaLaRuta` en verde |
| T-F030-03 | actualizar | `Bienvenida`: mismo comportamiento (tokens al pegar, `TextoExpandido` al enviar, resaltado y olvido) | completada | `internal/tui/welcome.go` | Test `TestPegarUnaImagenEnLaBienvenidaMuestraElToken` en verde |

Dependencias: T-F030-02 depende de T-F030-01; T-F030-03 depende de T-F030-01.
