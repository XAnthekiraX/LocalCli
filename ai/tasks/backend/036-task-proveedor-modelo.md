> T-B036 — el harness deja de hablar solo con Ollama: nace una frontera neutra (`llm.Proveedor`) con dos adaptadores (`ollama` y `openai` para `llama-server`) y el proveedor se elige al arrancar con `LOCALCLI_PROVEEDOR`. Sin cambios de esquema y sin SDK nuevo.
> Acción de esta descomposición: `actualizar`.

## Referencias

- [[specs/SPEC-MODELO-PROVEEDOR]] — §Proveedores, §Lo que cambia por proveedor, §La ventana de contexto, §Flujo principal y criterios de aceptación.
- [[backend/DECISIONS]] — frontera `llm.Proveedor`, ventana declarada vs leída, proveedor fijo por ejecución y «sin proxy».
- [[backend/04-infrastructure/INTEGRATIONS]] — §1 subsección por proveedor, §3 SSE con `bufio`, §5 contrato por proveedor.
- [[backend/04-infrastructure/CONFIGURATION]] — §2 `LOCALCLI_PROVEEDOR`/`LOCALCLI_LLAMACPP_URL`, §3 detección, §6 servicios, §7 `ultimo_proveedor`.
- [[backend/03-security/SECURITY]] — §3.2 banderas inadmisibles de `llama-server` y no mutar el servidor.
- [[backend/02-interfaces/INTERFACES-GENERAL]] — §5 contratos por módulo; [[backend/02-interfaces/TOOLS]] — §11 el cable hacia el modelo.
- [[backend/05-quality/ERRORS]] — `E_PROVEEDOR_NO_DISPONIBLE`; [[backend/05-quality/TESTING]] — dobles de proveedor.
- [[specs/SPEC-INTERFAZ]] — el modal y la línea de modelo nombran el proveedor.

## Contexto

Hoy `ollama` es el único módulo que habla con el modelo: `agent` construye la petición contra sus tipos (`ollama.Mensaje`, `ollama.Evento`, `ollama.Herramienta`) y `context` recorta por su ventana. El bloqueante es que `llama-server` eliminó en abril de 2026 (commit `cf8b0db`, PR #22165) los endpoints de compatibilidad con Ollama (`/api/*`), así que **no es un cambio de URL base**: es un adaptador nuevo con otro transporte.

La forma aprobada es una **frontera neutra** con un adaptador por runtime:

| | `ollama` | `openai` (llama.cpp) |
|---|---|---|
| Transporte | NDJSON por `/api/chat` | SSE por `/v1/chat/completions` |
| Razonamiento | campo `think` por petición | `delta.reasoning_content` |
| Herramientas | `tool_calls` | `delta.tool_calls` por índice (se acumulan) |
| Imágenes | `images` en base64 | `image_url` con data URI |
| Capacidades | `/api/show` | `/props` (`chat_template_caps`, `modalities`) |
| Ventana | **se declara** en `options.num_ctx` | **se lee** de `/props` (`n_ctx`) |

Puntos de comportamiento ya fijados (no se reinventan):

- `Nombre()` devuelve `ollama` o `llamacpp`; la etiqueta visible es `Ollama` o `llama.cpp`.
- `VentanaDeContexto` devuelve `(ventana, declarada)`. Con Ollama `declarada=true` y la ventana viaja en `options.num_ctx`. Con llama.cpp `declarada=false`, se lee de `/props` y la efectiva es `min(n_ctx del servidor, tope)`; si es menor que el tope se avisa con `-c 16384` y el historial se recorta al valor real. **Nunca `POST /props`.**
- El adaptador no implementa `POST /models` (descarga) ni `POST /models/load`; el router autocarga con `--models-autoload`. El harness no pide pesos a la red ni muta el servidor.
- `LOCALCLI_PROVEEDOR` sin valor es `ollama`; `auto` es opt-in y usa el primero que responde empezando por Ollama. El proveedor no cambia con la sesión viva.
- El modelo recordado solo se reutiliza si su proveedor es el elegido; si no, se autodetecta.

Lo que **no** se toca: `tui.Puerto` ya habla en `tui.ModeloLocal`; `agent.Generador` y `context.Modelo` no cambian de firma. `session`, `store`, `flow`, `queue`, `tools`, `fileops`, `exec`, `task` y `docs` no importan `ollama` y quedan fuera.

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-B036-01 | actualizar | `llm`: crear el paquete con `doc.go` (el límite del núcleo neutro: tipos y contrato, sin conocer ningún runtime) y `tipos.go` con `Mensaje`, `Herramienta`, `Definicion`, `ToolCall`, `Evento`/`TipoEvento`, `RespuestaFinal`, `Modelo` y `Peticion`, movidos desde `ollama` | completada | `internal/llm/doc.go`, `internal/llm/tipos.go` | `go build ./internal/llm/` |
| T-B036-02 | actualizar | `llm`: `proveedor.go` con la interfaz `Proveedor` —`Nombre`, `BaseURL`, `Chat`, `ListarModelos`, `Capacidades`, `VentanaDeContexto`— y `ventana.go` con `TopeVentanaPorDefecto`, `VentanaDeModelo` y `PuedeUsarHerramientas`/`PuedeVer`/`PuedePensar` | completada | `internal/llm/proveedor.go`, `internal/llm/ventana.go` | `go build ./internal/llm/` |
| T-B036-03 | actualizar | `llm`: `cola.go` con `ColaInferencia` y `OpcionesEncolar`, moviendo la FIFO de capacidad 1 desde `ollama`; `errores.go` con `ErrorProveedor`, `CodigoProveedorNoDisponible`, `CodigoModeloNoCabe`, sus centinelas y `mensajeLevantar` | completada | `internal/llm/cola.go`, `internal/llm/errores.go` | `go build ./internal/llm/` |
| T-B036-04 | actualizar | `ollama`: pasar a adaptador de `llm.Proveedor` conservando `client`, `stream`, `models`, `show`, `profile` y `reasoning`; los tipos pasan a ser de `llm`; `conNumCtx` intacto (la ventana se sigue declarando) | completada | `internal/ollama/*.go` | `go test ./internal/ollama/ -count=1` |
| T-B036-05 | actualizar | `openai`: adaptador nuevo de `llm.Proveedor` con `doc.go`; `client.go` (`POST /v1/chat/completions`, `GET /models`, `GET /props`); `modelos.go` (`/models` con reserva `/v1/models`); `props.go` (`n_ctx`; `chat_template_caps` → herramientas/razonamiento; `modalities.vision` → visión) | completada | `internal/openai/doc.go`, `internal/openai/client.go`, `internal/openai/modelos.go`, `internal/openai/props.go` | `go test ./internal/openai/ -count=1` |
| T-B036-06 | actualizar | `openai`: `stream.go` con el parser SSE —`delta.content`, `delta.reasoning_content`, `delta.tool_calls` acumulados por índice, `usage` y `[DONE]`— devolviendo `llm.Evento`; sin dependencia nueva (el SSE se lee con `bufio` de la estándar) | completada | `internal/openai/stream.go` | `TestElSSEAcumulaToolCallsPorIndice` |
| T-B036-07 | actualizar | `openai`: la ventana se lee de `/props` y, si no se puede leer, se trata como desconocida y el turno se recorta al tope; nunca `POST /props`, `POST /models` ni `POST /models/load`; aviso con `-c 16384` cuando la ventana efectiva es menor que el tope | completada | `internal/openai/props.go`, `internal/openai/client.go` | `TestLaVentanaSeLeeYNuncaSeMutaElServidor` |
| T-B036-08 | actualizar | `agent`: `Runner{Cliente}` → `Runner{Proveedor}` y `loop.go`/`dispatch.go` pasan de `ollama.*` a `llm.*`; `Generador` no cambia de firma | completada | `internal/agent/run.go`, `internal/agent/loop.go`, `internal/agent/dispatch.go` | `go test ./internal/agent/ -count=1` |
| T-B036-09 | actualizar | `context`: `ModeloOllama` → `ModeloLLM{Cliente llm.Proveedor}` conservando la firma de `Modelo`; el recorte se deriva de la ventana efectiva | completada | `internal/context/select.go` | `go test ./internal/context/ -count=1` |
| T-B036-10 | actualizar | `arranque`: factory de proveedor desde `LOCALCLI_PROVEEDOR`/`LOCALCLI_LLAMACPP_URL` (sin valor → `ollama`; `auto` → primero que responde), `Adaptador.proveedor`, `elegirModelo` y los cuatro literales «Ollama»; `ModeloActual()` devuelve `qwen3:8b (llama.cpp)` cuando toca | completada | `arranque.go` | `go test . -count=1` |
| T-B036-11 | actualizar | preferencias: `ultimo_proveedor` en `~/.config/localcli/config.json` (leído/escrito por el arranque); el modelo recordado solo se reutiliza si su proveedor es el elegido | completada | `arranque.go`, `internal/tui/config.go` | `TestElModeloRecordadoSoloSeReusaConSuProveedor` |
| T-B036-12 | actualizar | `estructura_test.go`: la lista de módulos pasa de 13 a 15 e incluye `llm` y `openai` | completada | `internal/estructura_test.go` | `go test ./internal/ -count=1` |
| T-B036-13 | actualizar | tests: dobles de `llm.Proveedor` para los que hoy doblan a `ollama`; renombrar el código `E_OLLAMA_UNAVAILABLE` por `E_PROVEEDOR_NO_DISPONIBLE` en tests e2e; caso de proveedor que no responde con aviso y arranque vivo | completada | `arranque_test.go`, `tests/e2e_localcli_test.go`, `tests/integration_errores_test.go`, `internal/agent/*_test.go` | `go test ./... -count=1` |
| T-B036-14 | actualizar | Documentación ya actualizada por el comando `/actualizar`: verificar que código y docs coinciden (módulos 15, `E_PROVEEDOR_NO_DISPONIBLE`, variables nuevas) | completada | `ai/docs/backend/BACKEND.md`, `ai/docs/backend/05-quality/ERRORS.md`, `ai/docs/backend/04-infrastructure/CONFIGURATION.md` | `go test ./internal/docs/ -count=1` |

Dependencias: T-B036-01 antes de -02, -03, -04 y -05. T-B036-02 antes de -04, -05 y -09. T-B036-03 antes de -04 y -05. T-B036-04 y -05 antes de -06, -07 y -10. T-B036-08 y -09 antes de -10. T-B036-10 antes de -11. T-B036-12 y -13 revisan el conjunto.

## Fuera de alcance

- **Nada de base de datos.** Ni tabla, ni columna, ni migración: `messages` no guarda el modelo y las preferencias viven en un archivo fuera del proyecto.
- **Sin proxy ni emulación de `/api/*`** sobre `llama.cpp`: el adaptador habla el protocolo del proveedor.
- **Sin SDK de terceros.** El SSE se lee con `bufio`; el transporte es el cliente HTTP de la estándar.
- **El proveedor no cambia en caliente.** El modal elige modelo, no runtime; cambiar de proveedor es reiniciar.
- **Ningún módulo por debajo de `agent`/`context` aprende el contrato neutro**: `session`, `store`, `flow`, `queue`, `tools`, `fileops`, `exec`, `task` y `docs` no se tocan.
- **No se muta el servidor**: ni descarga, ni carga, ni cambio de ventana.
