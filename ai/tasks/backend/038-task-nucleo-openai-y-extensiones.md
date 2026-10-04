> T-B038 — el harness habla **un solo protocolo** con todos los motores: nace `llm/openai` como núcleo común (`/v1/chat/completions` y `/v1/models`), y `ollama` y `llamacpp` dejan de traducir la conversación para aportar **solo** sus extensiones nativas (`num_ctx`, `show`, `tags` / `props`). Además, las capacidades pasan a **tres estados** (`soportada`, `no soportada`, `desconocida`), el recorte previo a la petición pasa a ser real y del harness, y **ningún motor queda activo por defecto**: el registro se entrega escrito con `activo: false`.
> Acción de esta descomposición: `actualizar`.

## Referencias

- [[specs/SPEC-MODELO-MOTOR]] — §Un solo protocolo, varios runtimes, §Las extensiones nativas, §Qué se sabe y qué no, §El recorte lo hace el harness, §Ningún motor activo por defecto, §Ciclo de vida de un motor y criterios de aceptación.
- [[backend/DECISIONS]] — la fila del núcleo OpenAI-compatible, la del recorte local, la de tríada de capacidades y la de ausencia de motor por defecto.
- [[backend/BACKEND]] — §2 estructura del proyecto (`llm/openai`, `llm/ollama`, `llm/llamacpp`) y las fichas de `llm`, `llm/openai`, `llm/ollama` y `llm/llamacpp`.
- [[backend/01-domain/DOMAIN]] — §Módulos: la fila `openai`, `ollama`, `llamacpp` y su frontera.
- [[backend/02-interfaces/INTERFACES-GENERAL]] — §5: `llm`, `llm/openai` y las extensiones, más los dos párrafos sobre el tipo que ya no decide el transporte y la tríada.
- [[backend/04-infrastructure/INTEGRATIONS]] — §Contrato con el motor: lo que el harness da por cierto y lo que depende de las extensiones.
- [[backend/04-infrastructure/CONFIGURATION]] — §2 `LOCALCLI_MOTOR`, §7 `ultimo_motor`, §8 el registro y el `motores.json` de ejemplo con `activo: false`.
- [[backend/04-infrastructure/EVENTS]] — §1 `token` lo emite el núcleo común, §3 por qué las capacidades no son un evento.
- [[backend/02-interfaces/TOOLS]] — §11 El cable: el paso 3 es el núcleo común, y qué se hace en cada estado de la capacidad.
- [[backend/05-quality/TESTING]] — filas de `llm` y de `openai`/`ollama`/`llamacpp`: la misma carga útil contra los dos motores.
- [[specs/SPEC-AGENTE-BASE]] — qué hace el bucle con una capacidad de herramientas desconocida.
- [[specs/SPEC-INTERFAZ]] — el rotulado `?` y el aviso de razonamiento desconocido.
- [[frontend/02-interfaces/INTERFACES]] — §3.2 Lecturas al motor: la tabla de los tres estados y qué se pinta.

## Contexto

T-B037 dejó varios motores declarables, con su registro, su cola y su par motor/modelo por sesión. Pero cada motor seguía siendo **una traducción completa**: `internal/ollama` hablaba NDJSON por `/api/chat` y `internal/llamacpp` SSE por `/v1/chat/completions`. Eso obliga a mantener y probar dos lecturas de stream, dos formatos de herramientas y dos rutas de imágenes para una diferencia que el protocolo OpenAI ya resolvió y que **los dos runtimes ya exponen** —Ollama en `/v1`, `llama-server` en `/v1`—.

El tipo cerrado convirtió además «saber más sobre un motor» en «escribir otro adaptador». Y el registro sembraba `ollama` y `llamacpp` con direcciones por defecto: presupone un runtime que puede no existir, convierte el arranque en una cascada de fallos y hace que instalar LocalCli sea instalar medio Ollama.

| | Hoy (T-B037) | Ahora |
|---|---|---|
| Conversación | Dos traducciones: NDJSON en `ollama`, SSE en `llamacpp` | Un núcleo común `llm/openai` para todos; el runtime no se ramifica |
| Lo específico del runtime | Todo mezclado en el adaptador | **Solo** extensiones opcionales: `num_ctx`, `show`, `tags` (`ollama`); `props` (`llamacpp`) |
| Capacidades | Dos estados (`sí` / `no`) | Tres estados; `false` **no** significa nunca `desconocida` |
| Ventana sin extensión | Se trata como si se conociera | **Desconocida**: manda el tope y avisa de que el límite es del harness |
| Recorte | Confiado en el servidor | El harness recorta de verdad antes de enviar |
| Registro ausente | Se siembran `ollama`+`llamacpp` | Registro vacío, y es un estado válido |
| Motor por defecto | `ollama` en `localhost:11434` | Ninguno; el registro llega escrito con `activo: false` |
| Runtime nuevo | Código nuevo | Solo el núcleo, sin código |

Puntos de comportamiento ya fijados (no se reinventan):

- **El núcleo obligatorio es `/v1/chat/completions` y `/v1/models`.** Un runtime que no los exponga no es admisible: no hay una segunda forma de hablarle al harness.
- **`tipo` ya no decide el transporte.** Sobrevivió porque sigue nombrando **qué extensiones se pueden declarar**, y por eso el catálogo sigue siendo cerrado; lo que se cae es el `switch` de protocolo.
- Las extensiones se **declaran por instancia** en el campo `extensiones` de `motores.json`. Ausente se lee como «solo núcleo común».
- Cada punto es **independiente**: un motor puede declarar `num_ctx` y no `show`, y entonces conoce su ventana y no sus capacidades. Nunca se hereda nada.
- **`desconocida` no colapsa.** Herramientas: no se ofrecen y se marca `?`. Visión: no se ofrece y se marca `?`. Razonamiento: se desactiva y **se avisa**, porque afecta al turno que se está haciendo.
- La ventana efectiva sigue siendo la menor entre lo declarado, lo del servidor y `LOCALCLI_CONTEXT_LIMIT`. Sin extensión que la declare queda **desconocida**, y entonces manda el tope.
- **El recorte es del harness y es real**: antes de enviar, la lista de mensajes se recorta de más nuevo a más viejo hasta que quepa, y se avisa. Reutiliza las estimadoras y los recortadores que ya existen; no se escribe un truncador nuevo.
- **Un registro ausente o ilegible deja el registro vacío.** No hay motor de arranque y no se inventa uno. No es un error a tapar.
- **El registro se entrega escrito** con las dos entradas de referencia, su tipo, su dirección y sus extensiones, y **`activo: false`**. Es un dato, no un default activo.
- `LOCALCLI_MOTOR` sigue siendo una comodidad de arranque: sin ninguna entrada activa no hay motor de arranque, y un `id` inexistente se ignora y avisa — **no** se sustituye por otro.
- `ultimo_motor`/`ultimo_modelo` que ya no estén registrados no se reutilizan, y tampoco se sustituyen por otro automático.
- **Sin migración.** `sessions.motor_id` y `sessions.modelo` ya existen y bastan: el par motor/modelo por sesión no cambia, y `llm` no persiste.
- La cola por motor, la advertencia de proxy, «editar un motor aplicado exige reiniciar», desactivar sin borrar y eliminar sin invalidar sesiones **no cambian**: son de T-B037 y siguen vigentes.

Lo que **no** se toca: `tui.Puerto` ya habla en `tui.ModeloLocal`, así que la tríada de capacidades solo necesita el tipo nuevo; `agent.Generador` y `context.Modelo` no cambian de firma. `session`, `store`, `flow`, `queue`, `tools`, `fileops`, `exec`, `task` y `docs` no importan `llm` y quedan fuera.

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-B038-01 | actualizar | `llm`: el tipo `Capacidad` de tres estados (`CapacidadSoportada`, `CapacidadNoSoportada`, `CapacidadDesconocida`) y `Motor.Capacidades` devolviéndolo; `ventana.go` distingue desconocida de cero y aplica el tope con aviso | pendiente | `internal/llm/motor.go`, `internal/llm/ventana.go`, `internal/llm/doc.go` | `go test ./internal/llm/ -count=1` |
| T-B038-02 | actualizar | `llm/motores.go`: `Motor` declarado gana `Extensiones []string` (ausente = solo núcleo común) y la resolución usa **las extensiones declaradas**, no el `tipo`, para saber qué puntos puede preguntar; el catálogo de tipos sigue cerrado | pendiente | `internal/llm/motores.go` | `TestExtensionAusenteNoSeAsumePorElTipo` |
| T-B038-03 | actualizar | `llm/motores.go`: **sin `motores.json` el registro queda vacío**, sin siembra de motores; si está mal formado, avisa y también queda vacío; se borra el fallback a `ollama` en `localhost:11434` | pendiente | `internal/llm/motores.go` | `TestRegistroAusenteNoSiembraMotores` |
| T-B038-04 | actualizar | `llm/openai`: **núcleo común** con el cliente HTTP y la lectura SSE de `/v1/chat/completions`, el listado en `/v1/models`, el razonamiento distinguido del texto final, las imágenes en `image_url`, el canal estructurado de herramientas y los tokens de uso; no sabe de runtimes ni los importa | pendiente | `internal/openai/client.go`, `internal/openai/modelos.go`, `internal/openai/stream.go`, `internal/openai/doc.go` | `TestElNucleoComunNoImportaRuntimes` |
| T-B038-05 | actualizar | `llm/ollama`: deja de implementar la conversación y pasa a **extensión**; solo aporta `options.num_ctx` en la petición (cuando está declarada), `/api/show` y `/api/tags`; cada consulta va detrás de su extensión declarada | pendiente | `internal/ollama/client.go`, `internal/ollama/show.go`, `internal/ollama/aliases.go`, `internal/ollama/doc.go`, `internal/ollama/errors.go` | `TestNumCtxSoloSeMandaSiEstaDeclarado` |
| T-B038-06 | actualizar | `llm/llamacpp`: deja de implementar la conversación y pasa a **extensión**; solo aporta `/props`, de donde lee `default_generation_settings.n_ctx` y las capacidades del servidor; si no se puede leer devuelve **desconocida**, no cero, y no muta el servidor | pendiente | `internal/llamacpp/props.go`, `internal/llamacpp/doc.go` | `TestPropsSinModeloDevuelveDesconocida` |
| T-B038-07 | actualizar | `llm`: **recorte real antes de enviar** — la lista de mensajes se acorta de más nuevo a más viejo hasta que quepa en la ventana efectiva, y se avisa al hacerlo; reutiliza las estimadoras y los recortadores existentes (`context/trim.go`, `tools/truncado.go`) en vez de escribir un truncador nuevo | pendiente | `internal/llm/motor.go`, `internal/agent/run.go`, `internal/context/trim.go` | `TestElTurnoSeRecortaAntesDeEnviarYSeAvisa` |
| T-B038-08 | actualizar | `arranque`: sin motor activo no se elige motor ni se intenta conectar a nadie; la sesión nueva nace sin motor y espera; `LOCALCLI_MOTOR` inexistente se ignora y avisa; `ultimo_motor` no registrado no se sustituye por otro automático | pendiente | `arranque.go` | `TestArranqueSinMotoresNoIntentaConectar` |
| T-B038-09 | actualizar | `llm`: **el registro se entrega escrito** — el binario incluye un `motores.json` de ejemplo con las dos entradas de referencia, tipo, dirección y extensiones, y **`activo: false`**; se copia al `~/.config/localcli/motores.json` del usuario solo si no existe, y nunca sobrescribe lo suyo | pendiente | `internal/llm/motores.go`, `internal/llm/motores_test.go`, assets del binario | `TestElEjemploTraeLosDosMotoresInactivos` |
| T-B038-10 | actualizar | `agent`: una capacidad de herramientas `desconocida` **cae a modo conversación** sin enviar el canal estructurado, igual que `no soportada`; con `soportada` usa el esquema. El razonamiento `desconocida` se desactiva y **avisa** | pendiente | `internal/agent/loop.go`, `internal/agent/run.go` | `TestHerramientasDesconocidasNoSeOfrecen` |
| T-B038-11 | actualizar | `tui`: `tui.Capacidades` pasa a tres estados y pinta `?` donde el estado es `desconocida`, sin ofrecer herramientas ni visión; el aviso de razonamiento desconocido lo emite el backend y la vista lo muestra | pendiente | `internal/tui/app.go`, `internal/tui/wire.go`, `internal/tui/modelsmodal.go`, `internal/tui/motorsmodal.go` | `TestCapacidadDesconocidaSePintaConInterrogacion` |
| T-B038-12 | actualizar | `tui`: el modal de motores muestra el campo `extensiones` editable y explica que sin extensión declarada lo que no se informa queda desconocido | pendiente | `internal/tui/motorsmodal.go`, `internal/tui/motorsmodal_test.go` | `TestElModalDeclaraExtensionesPorMotor` |
| T-B038-13 | actualizar | Tests: la misma `llm.Peticion` serializada produce **la misma carga útil** contra un motor `ollama` y uno `llamacpp`; cada extensión ausente degrada su punto y solo el suyo; el registro vacío no inventa motor; el recorte previo reduce los mensajes hasta que caben | pendiente | `internal/openai/openai_test.go`, `internal/llm/motores_test.go`, `internal/llm/ventana_test.go`, `tests/motores_test.go` | `go test ./... -count=1` |

## Fuera de alcance

- **Sin migración de base de datos.** `sessions.motor_id` y `sessions.modelo` siguen como están.
- **Sin proxy ni proceso traductor.**
- **Sin SDK nuevo**: el núcleo usa la biblioteca HTTP y SSE estándar.
- **El catálogo de tipos sigue cerrado** (`ollama`, `llamacpp`). Abrirlo a un tercer runtime es trabajo aparte y no lo pide esta tarea.
- **No se comprueba el encaje con el hardware** sin la extensión `tags`: sin ella el harness no afirma que el modelo quepa ni que no quepa.
- `arranque`, `session`, `flow`, `queue`, `tools`, `fileops`, `exec`, `task` y `docs` no cambian de comportamiento más allá de lo que exige la ausencia de motor por defecto.