> T-B022 — imágenes por turno de chat para los modelos multimodales de Ollama.
> Acción de esta descomposición: `actualizar`.

## Referencias

- [[specs/SPEC-OLLAMA-PERFIL]] — §Reglas: imágenes base64 en `/api/chat`, efímeras, y aviso no bloqueante si el modelo no declara visión.
- [[specs/SPEC-INTERFAZ]] — §Reglas: la entrada detecta rutas de imagen del mensaje y la línea de estado muestra visión.
- [[backend/04-infrastructure/INTEGRATIONS]] — §1/§5: el campo `images` de `/api/chat` y la capacidad `vision`.
- [[backend/02-interfaces/INTERFACES-GENERAL]] — §3/§5: las imágenes como excepción al texto plano, solo conocidas por `ollama`.

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-B022-01 | actualizar | `ollama`: `Mensaje.Images` (base64, `images` de `/api/chat`) y capacidad `vision` (`CapacidadVision`, `PuedeVer`) | completada | `internal/ollama/client.go`, `internal/ollama/show.go` | Tests `TestChatEnviaImagenes`, `TestChatOmiteImagenesSinAdjuntos` y `TestPuedeVerDetectaLaCapacidadDeVision` en verde |
| T-B022-02 | actualizar | `agent`: `Ejecutor.Ejecutar` recibe las imágenes del turno y las adjunta solo al mensaje de usuario del turno actual | completada | `internal/agent/loop.go` | Test `TestElTurnoLlevaImagenesAlModelo` en verde |
| T-B022-03 | actualizar | `flow`: `Agente.Ejecutar` y `Motor.Conversar` transportan las imágenes; las etapas de flujo van sin ellas | completada | `internal/flow/engine.go` | Tests `TestConversarReenviaLasImagenesAlAgente` y `TestEjecutarFlujoNoPasaImagenes` en verde |
| T-B022-04 | actualizar | `session`: `Motor.Conversar` y `Gestor.Conversar` reciben las imágenes y las pasan al motor; son efímeras (no se guardan en `store`) | completada | `internal/session/run.go` | Tests de `localcli/internal/session` en verde |
| T-B022-05 | actualizar | Adaptador: `Enviar` propaga las imágenes al chat; `Modelos` y `CapacidadesModelo` exponen `SinVision`/`Vision` | completada | `arranque.go`, `internal/tui/wire.go` | `go build ./...` y tests de `localcli` en verde |
| T-B022-06 | actualizar | `tui`: detección de rutas de imagen en el texto (`AdjuntosDe`), envío de las imágenes y aviso de visión | completada | `internal/tui/adjuntos.go`, `internal/tui/app.go`, `internal/tui/welcome.go`, `internal/tui/modelsmodal.go` | Tests `TestAdjuntosDeLeeImagenesYLasCodifica`, `TestEnviarAdjuntaLasImagenesDelTexto` y `TestAvisoDeVisionAlEnviarImagenSinVision` en verde |

Dependencias: T-B022-02 depende de T-B022-01; T-B022-03 depende de T-B022-02; T-B022-04 depende de T-B022-03; T-B022-05 depende de T-B022-04; T-B022-06 depende de T-B022-05.
