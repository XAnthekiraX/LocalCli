> T-F029 — la entrada adjunta imágenes detectadas en el texto y avisa si el modelo no ve.
> Acción de esta descomposición: `actualizar`.

## Referencias

- [[specs/SPEC-INTERFAZ]] — §Reglas: la entrada detecta rutas de imagen del mensaje; la línea de estado muestra visión; el aviso no bloquea.
- [[specs/SPEC-OLLAMA-PERFIL]] — §Reglas: imágenes efímeras por turno y aviso si el modelo no declara `vision`.
- [[frontend/01-domain/DOMAIN]] — §1/§3: el input compone y envía; la vista pinta y avisa, no decide.

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|------|----------|--------------|
| T-F029-01 | actualizar | `adjuntos.go`: `AdjuntosDe` detecta rutas de imagen existentes en el texto y las codifica en base64; lo ilegible se avisa sin cortar | completada | `internal/tui/adjuntos.go` | Tests `TestAdjuntosDeLeeImagenesYLasCodifica`, `TestAdjuntosDeIgnoraRutasInexistentesYNoImagenes` y `TestAdjuntosDeAvisaSiNoPuedeLeer` en verde |
| T-F029-02 | actualizar | `Enviar` del puerto lleva las imágenes; `app.go`/`welcome.go` las calculan al enviar y `ejecutarComando` va sin ellas | completada | `internal/tui/wire.go`, `internal/tui/app.go`, `internal/tui/welcome.go` | Test `TestEnviarAdjuntaLasImagenesDelTexto` en verde |
| T-F029-03 | actualizar | aviso de visión no bloqueante al enviar una imagen con un modelo sin visión; la línea de estado muestra visión | completada | `internal/tui/app.go` | Tests `TestAvisoDeVisionAlEnviarImagenSinVision`, `TestSinAvisoCuandoElModeloVe` y `TestLaLineaDeEstadoMuestraVision` en verde |
| T-F029-04 | actualizar | modal de modelos: marca `(sin visión)` | completada | `internal/tui/modelsmodal.go` | Test `TestElModalMarcaLosModelosSinVision` en verde |

Dependencias: T-F029-02 depende de T-F029-01; T-F029-03 depende de T-F029-02; T-F029-04 depende de T-F029-02.
