> T-B024 — herramientas nativas y extensibles: el modelo pide herramientas por el canal estructurado de Ollama, toda ejecución pasa por una capa universal y el usuario puede declarar las suyas.
> Acción de esta descomposición: `actualizar`.

## Referencias

- [[specs/SPEC-TOOLS]] — §Contrato, §Ejecución y §Herramientas del usuario.
- [[specs/SPEC-OLLAMA-PERFIL]] — §Capacidades y §Concurrencia: el canal de herramientas y el turno de inferencia.
- [[specs/SPEC-AGENTE-BASE]] — §Ciclo conversacional: el bucle de pedir, ejecutar y volver a pedir.
- [[backend/02-interfaces/TOOLS]] — la capa universal, los hooks y la carga de herramientas del usuario.
- [[backend/02-interfaces/dto/TOOLS-DTO]] — payloads y el esquema derivado por reflexión.
- [[backend/04-infrastructure/CONFIGURATION]] — §Herramientas del usuario: dónde viven y qué declaran.
- [[backend/04-infrastructure/EVENTS]] — `herramienta_invocada` y `herramienta_resultado`.
- [[backend/05-quality/VALIDATION]] — qué error se puede corregir y cuál no.
- [[database/02-rules/DATA-FLOW]] — el turno de herramienta no se persiste.

## Contexto

Esta tarea **reescribe** lo que hicieron T-B007, T-B009, T-B017 y T-B018, y no los sustituye: el catálogo de trece, la lista blanca, el aislamiento y el reparto por agente se mantienen. Lo que cambia es el canal, la forma de ejecutar y el punto de extensión.

Tres decisiones de [[backend/DECISIONS]] gobiernan todo lo demás: tool-calling nativo sin reserva en prosa, capa universal de ejecución, y herramientas del usuario como subproceso de solo lectura. No hay migración de base de datos: `messages.role` sigue siendo `user`/`agent` y los turnos de herramienta viven solo en memoria.

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-B024-01 | actualizar | Corregir el proyecto que se pasa al ejecutor en el arranque: hoy usa la carpeta de la home y no la del proyecto, lo que deja la terminal sin frontera | completada | `arranque.go` | Test `TestEjecutorUsaLaCarpetaDelProyecto` en verde |
| T-B024-02 | actualizar | `tools`: derivar el JSON Schema por reflexión desde los DTO, con la descripción del tag `desc` y `required` en los campos sin `omitempty` | completada | `internal/tools/esquema.go`, `internal/tools/dto.go` | Test `TestEsquemaDesdeDTO` en verde; un campo `desc` aparece como `description` |
| T-B024-03 | actualizar | `tools`: `Def`, `Contexto`, `Resultado` y el wrapper de ejecución por herramienta | completada | `internal/tools/def.go` | Compila; los 13 handlers existentes implementan la interfaz |
| T-B024-04 | actualizar | `tools`: truncado universal de la salida por tokens y distinción entre error corregible (`Resultado.Error`) y error duro (error de Go) | completada | `internal/tools/truncado.go`, `internal/tools/def.go` | Test `TestTruncadoRespetaElLimite` en verde |
| T-B024-05 | actualizar | `tools`: migrar los 13 handlers al contrato `Ejecutar`, cada uno con su propio, en vez del enrutado por categorías | completada | `internal/tools/catalog.go`, `internal/tools/describe.go`, `internal/tools/validate.go` | `TestCatalogo` en verde: siguen siendo exactamente 13 |
| T-B024-06 | actualizar | Unificar la aprobación: las herramientas que necesitan permiso piden por `Contexto.Ask` en vez de montar su propia notificación | completada | `internal/fileops/approval.go`, `internal/exec/run.go` | Test de aprobación de una herramienta de escritura en verde |
| T-B024-07 | actualizar | `tools`: ganchos opcionales antes y después de ejecutar, sin que el catálogo tenga que saber quién los instala | completada | `internal/tools/hooks.go` | Un hook se invoca y puede abortar la ejecución |
| T-B024-08 | actualizar | `tools`: cargar los `.localcli/tools/*.json` del proyecto, separando las declaradas de las 13 integradas | completada | `internal/tools/usuario.go` | Test `TestCargaHerramientasDelUsuario`: un JSON válido se carga, uno inválido se salta sin error |
| T-B024-09 | actualizar | `tools`: ejecutar una herramienta del usuario como subproceso argv sobre `exec`, heredando lista blanca, Landlock, tiempo y recorte | completada | `internal/tools/usuario.go`, `internal/exec/exec.go` | Test de que `modo: "escribe"` se rechaza al cargar; test de que una lectura pide permiso siempre |
| T-B024-10 | actualizar | `ollama`: añadir `tools` a la petición de `/api/chat` y los mensajes de rol `tool` al transporte | completada | `internal/ollama/client.go` | Test de serialización: la petición lleva un objeto por herramienta |
| T-B024-11 | actualizar | `ollama`: acumular `message.tool_calls` durante el streaming y devolverlos al terminar, sin esperar al final para el texto | completada | `internal/ollama/stream.go` | Test `TestAcumulaToolCalls` en verde |
| T-B024-12 | actualizar | `agent`: bucle secuencial de pedir, ejecutar y volver a pedir, con máximo de rondas y sin contrato en prosa | completada | `internal/agent/loop.go` | Test del turno con dos herramientas encadenadas en verde |
| T-B024-13 | actualizar | `agent`: consumir `EventoDone` en vez de descartar el conteo de tokens | completada | `internal/agent/loop.go`, `internal/agent/run.go` | El panel muestra los tokens del turno |
| T-B024-14 | actualizar | Tomar el turno de inferencia por petición al modelo, no por ejecución completa, para que una espera de aprobación no retenga el modelo; quitar el `Encolar` de la capa de `Chat` para evitar el auto-bloqueo | completada | `arranque.go`, `internal/ollama/client.go` | Test `TestElTestigoSeTomaPorPeticion` en verde |
| T-B024-15 | actualizar | `agent`: quitar del prompt el catálogo y el formato de solicitud, y rellenar `tools.Peticion.Agente` —hoy nadie lo pone ni lo lee fuera de los tests— para que la capa universal sepa a qué agente se pide | completada | `internal/agent/run.go`, `internal/agent/loop.go` | Test `TestPromptNoLlevaCatalogo` en verde |
| T-B024-16 | actualizar | `agent`: si el modelo no declara capacidad de herramientas, degradar a `SoloConversacion()` y avisar por el puerto en vez de fallar | completada | `internal/agent/loader.go`, `internal/agent/run.go` | Test `TestSinHerramientasDegradaAConversacion` en verde |
| T-B024-17 | actualizar | `tools`: emitir `herramienta_invocada` y `herramienta_resultado` desde la capa universal, con un resumen de argumentos sin valores sensibles | completada | `internal/tools/def.go`, `arranque.go` | Test de que los dos eventos salen de toda ejecución, incluidas las del usuario |
| T-B024-18 | actualizar | `tui`: pintar la línea de herramienta desde los eventos, sin importar `tools` y sin volcar la salida cruda | completada | `internal/tui/chat.go` | Test de arquitectura en verde: `tui` no importa `tools` |
| T-B024-19 | actualizar | Eliminar `Solicitudes` y `CatalogoTexto`, y toda la ruta de parseo en prosa | completada | `internal/agent/loop.go`, `internal/tools/describe.go` | `rg "CatalogoTexto\|Solicitudes"` sin resultados |
| T-B024-20 | actualizar | Tests: reescribir `loop_test.go` y `run_test.go` al contrato nativo, añadir los de esquema, truncado, carga y subprocess, y mantener el catálogo en 13 | completada | `internal/agent/loop_test.go`, `internal/agent/run_test.go`, `internal/tools/tools_test.go`, `internal/tools/usuario_test.go` | `go test ./...` en verde; suite de arquitectura en verde |

Dependencias: T-B024-02 y T-B024-03 antes de T-B024-04, T-B024-05 y T-B024-17. T-B024-01 antes de T-B024-09. T-B024-10 y T-B024-11 antes de T-B024-12. T-B024-12 antes de T-B024-13, T-B024-15, T-B024-16 y T-B024-19. T-B024-17 antes de T-B024-18. T-B024-20 revisa todas.

## Fuera de alcance

- **No hay migración de la base de datos.** El turno de herramienta no se persiste y `messages.role` no cambia. Ver [[database/03-operations/MIGRATIONS]].
- **No hay herramientas de escritura entre las del usuario.** `modo: "escribe"` se rechaza, no se implementa. Si algún día se admite, es una decisión de seguridad nueva.
- **No hay reserva en prosa.** Si un modelo no pide herramientas por el canal, el agente conversa; no intenta adivinar la intención.
- **No se toca la lista blanca ni el aislamiento de la terminal.** Solo se les da otro consumidor.
