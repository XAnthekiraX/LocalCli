> T-B037 — el proveedor deja de ser único y elige al arrancar: nace el **registro de motores** (`~/.config/localcli/motores.json`), `llm.Proveedor` pasa a `llm.Motor`, cada motor declarado tiene su propio adaptador y su propia cola de inferencia, y cada sesión guarda el par motor/modelo (`sessions.motor_id`, `sessions.modelo`, migración 006) para poder cambiar de runtime con la conversación ya escrita. Sin SDK nuevo y sin proxy.
> Acción de esta descomposición: `actualizar`.

## Referencias

- [[specs/SPEC-MODELO-MOTOR]] — §Qué es un motor, §Varias instancias del mismo tipo, §Gestión de motores (agregar, editar, desactivar, reactivar, eliminar), §Motor y modelo por sesión, §Cambiar a mitad de conversación, §Varios modelos a la vez y criterios de aceptación.
- [[specs/SPEC-SESIONES]] — §El par motor/modelo por sesión: cambio en caliente, supervivencia al cierre y a la eliminación del motor.
- [[specs/SPEC-ORQUESTADOR-FLUJOS]] — el orquestador de etapas; `flow` no toca el registro ni sabe el tipo de motor.
- [[backend/DECISIONS]] — par motor/modelo en la sesión, registro global en archivo, editar exige reiniciar y una cola por motor.
- [[backend/04-infrastructure/CONFIGURATION]] — §2 `LOCALCLI_MOTOR`, §3 el motor se administra, §7 `ultimo_motor`, §8 estructura de `motores.json`.
- [[backend/04-infrastructure/INTEGRATIONS]] — contrato por motor, `modelos_por_motor` como caché y normalización de capacidades.
- [[database/01-schema/TABLES]] — `sessions.motor_id` y `sessions.modelo`, sin clave foránea; [[database/03-operations/MIGRATIONS]] — migración `006-motor-y-modelo-de-sesion`.
- [[backend/05-quality/ERRORS]] — `E_MOTOR_NO_DISPONIBLE`, `E_MOTOR_TIPO_DESCONOCIDO`, `E_REGISTRO_MOTORES_INVALIDO`.
- [[specs/SPEC-INTERFAZ]] — los cuatro modales y el rotulado `motor / modelo`.

## Contexto

T-B036 dejó el harness hablando con un contrato neutro (`llm.Proveedor`) pero con **un solo proveedor por ejecución**, elegido con `LOCALCLI_PROVEEDOR` al arrancar y compartido por todas las sesiones. Eso no cubre lo que ahora exige la documentación: una máquina puede tener varios runtimes y varios servidores de cada tipo, el usuario los administra, y cada sesión trabaja contra el suyo y puede cambiar sin perder la conversación.

El cambio tiene tres partes que van juntas porque por separado no resuelven nada:

| | Hoy (T-B036) | Ahora |
|---|---|---|
| Motores | Uno por ejecución, elegido al arrancar | Varias instancias declaradas, cada una con nombre, tipo, URL y estado |
| Persistencia | `ultimo_proveedor` en `config.json` | Registro `motores.json` + `ultimo_motor`/`ultimo_modelo` en `config.json` |
| Sesión | El motor es del arranque | El par motor/modelo es de la sesión (`sessions.motor_id`, `sessions.modelo`) |
| Concurrencia | Una cola global | Una `ColaInferencia` por motor |
| Cambio en caliente | Imposible (reiniciar) | Sí, y es el caso normal: `Ctrl+X i` + `Enter` |

Puntos de comportamiento ya fijados (no se reinventan):

- El **catálogo de tipos es cerrado**: `ollama` y `llamacpp`. Un `tipo` desconocido se rechaza al cargar, se salta esa entrada y se avisa; si no queda ninguna, se usa `ollama` en `localhost:11434`.
- Sin `motores.json` se registran `ollama` y `llamacpp` con sus direcciones por defecto. Un registro **ilegible nunca deja al usuario sin motor**: se avisa el error y se sigue con los valores por defecto.
- `id` estable e independiente del nombre y de la URL: las sesiones lo nombran y no quedan huérfanas al editarlo.
- `modelos_por_motor` es una **caché del último catálogo visto**, no una verdad: al conectar se pregunta al motor y lo que responda manda.
- **Editar** un motor aplicado no cambia el comportamiento hasta reiniciar; agregar, desactivar, reactivar y eliminar son inmediatos. La razón está en DECISIONS: el adaptador vive en memoria y dos versiones del mismo motor sirviendo turnos a la vez no tienen frontera.
- Desactivar **no** borra: el motor sigue visible con sus modelos no disponibles, y las sesiones que lo usaban avisan y esperan con su historial.
- **Eliminar** un motor borra su configuración y sus modelos registrados, y **no elimina ni invalida ninguna sesión**: la sesión avisa de que su motor falta y espera a que se elija otro.
- `LOCALCLI_MOTOR` es una comodidad de arranque para sesiones nuevas: el `id` de una entrada, o `auto` para el primero activo que responde. Un `id` que no existe se ignora y se avisa.
- `ultimo_motor` y `ultimo_modelo` van juntos: el mismo nombre de modelo no significa lo mismo en runtimes distintos, así que el modelo recordado solo se reutiliza si su motor registrado es el que corresponde.
- La ventana, la ficha de capacidades y la que declara el modelo se **cachean por motor**: dos motores del mismo tipo pueden declarar cosas distintas del mismo nombre de modelo.
- Cambiar de motor recalcula el presupuesto del historial a la ventana efectiva del motor nuevo y relee capacidades; **no reconstruye el contexto del turno** y no toca el historial de la sesión.

Lo que **no** se toca: `tui.Puerto` ya habla en `tui.ModeloLocal`; `agent.Generador` y `context.Modelo` no cambian de firma. `session`, `store`, `flow`, `queue`, `tools`, `fileops`, `exec`, `task` y `docs` no importan `llm` y quedan fuera.

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-B037-01 | actualizar | `llm`: renombrar la interfaz `Proveedor` a `Motor` (`proveedor.go` → `motor.go`) y sus implementaciones en `agent`/`context`; el contrato no cambia de firma | completada | `internal/llm/motor.go`, `internal/agent/run.go`, `internal/context/select.go` | `go build ./internal/llm/ ./internal/agent/ ./internal/context/` |
| T-B037-02 | actualizar | `llm`: `errores.go` con `ErrorMotor`, `CodigoMotorNoDisponible`, `CodigoMotorTipoDesconocido`, `CodigoRegistroMotoresInvalido`, sus centinelas y `mensajeLevantar` | completada | `internal/llm/errores.go` | `go build ./internal/llm/` |
| T-B037-03 | actualizar | `llm/motores.go`: el registro de motores —`Motor` declarado (`ID`, `Nombre`, `Tipo`, `URL`, `Activo`), `Registro` con `Cargar`, `Guardar`, `Agregar`, `Editar`, `Desactivar`, `Reactivar`, `Eliminar`, `PorID`, `Activos`— sobre `~/.config/localcli/motores.json`, con `modelos_por_motor` como caché; sin registro ni mal formado se cae a `ollama`+`llamacpp` por defecto | completada | `internal/llm/motores.go` | `TestRegistroIlegibleArrancaConLosDosPorDefecto` |
| T-B037-04 | actualizar | `llm`: un adaptador vivo por motor registrado, construido por su tipo cerrado y cacheado por `id`; agregar, desactivar, reactivar y eliminar lo construyen o lo destruyen al momento, y **editar marca `pendienteDeReinicio`** sin tocar el adaptador en uso | completada | `internal/llm/motores.go`, `internal/llm/motor.go` | `TestEditarUnMotorAplicadoNoCambiaHastaReiniciar` |
| T-B037-05 | actualizar | `llm`: `ColaInferencia` por `id` de motor en vez de una global; dos motores distintos generan a la vez y las sesiones del mismo se atienden por orden de llegada; el testigo se toma por petición y se suelta al terminar | completada | `internal/llm/cola.go` | `TestDosMotoresGeneranSimultaneamente` |
| T-B037-06 | actualizar | `llm`: cachear ventana y ficha de capacidades **por motor**, no por nombre de modelo; `VentanaDeContexto` devuelve la efectiva (la menor entre modelo, servidor y `LOCALCLI_CONTEXT_LIMIT`) | completada | `internal/llm/ventana.go`, `internal/llm/motores.go` | `TestDosMotoresDelMismoTipoNoCompartenVentana` |
| T-B037-07 | actualizar | `ollama` y `llamacpp`: renombrar a adaptadores de motor y exponer `Tipo() string` (`ollama`/`llamacpp`); el resto del comportamiento (NDJSON/SSE, herramientas, razonamiento, imágenes, capacidades, ventana declarada/leída) no cambia | completada | `internal/ollama/*.go`, `internal/llamacpp/*.go` | `go test ./internal/ollama/ ./internal/llamacpp/ -count=1` |
| T-B037-08 | actualizar | `store`: migración `006-motor-y-modelo-de-sesion` (`ALTER TABLE sessions ADD COLUMN motor_id TEXT` y `modelo TEXT`, ambas nullable), `user_version` a 6; leer y escribir el par en crear sesión, cambiar motor/modelo y retomar | completada | `internal/store/migrate.go`, `internal/store/motor.sql`, `internal/store/schema.go`, `internal/store/sessions.go`, `internal/store/store_test.go`, `internal/store/chatevento_test.go` | `TestLaMigracion006DejaLasColumnasEnNULL` |
| T-B037-09 | actualizar | `session`: `ParMotorModelo(idMotor, modelo)`, `CambiarMotor`, `CambiarModelo`, el par de la sesión en `ResolverActiva` y en el arranque del turno; pide al registro el adaptador de ese `id` en cada turno y **nunca elige motor por su cuenta** | completada | `internal/session/model.go`, `internal/session/store.go`, `internal/session/bg.go`, `internal/session/session_test.go` | `TestCambiarDeMotorNoTocaElHistorialNiElNombre` |
| T-B037-10 | actualizar | `session`: una sesión cuyo motor está desactivado o eliminado no muere: avisa con `E_MOTOR_NO_DISPONIBLE`, conserva el historial y espera; no invalida ni borra la fila | completada | `internal/session/run.go`, `internal/session/bg.go`, `internal/session/session_test.go` | `TestEliminarElMotorNoEliminaLaSesion` |
| T-B037-11 | actualizar | `arranque`: cargar el registro, conectar los motores activos, refrescar `modelos_por_motor`; resolver `LOCALCLI_MOTOR` (id, `auto`, ausente → `ultimo_motor` → `auto`); si el motor recordado ya no está registrado, autodetectar y avisar | completada | `arranque.go` | `TestMotorRecordadoInexistenteNoFallaElArranque` |
| T-B037-12 | actualizar | `arranque`: preferencias `ultimo_motor` en `~/.config/localcli/config.json` junto a `ultimo_modelo`; el par solo se reutiliza si su motor sigue registrado y ambos cuadran, si no se autodetecta; eliminar `LOCALCLI_PROVEEDOR` y `LOCALCLI_LLAMACPP_URL` | completada | `arranque.go`, `internal/tui/config.go` | `TestElParRecordadoSoloSeReusaSiSuMotorEstaRegistrado` |
| T-B037-13 | actualizar | `arranque`: `Adaptador` pasa a `Adaptadores` (un `llm.Motor` por `id`); quitar las cachés globales `ventanaServidor`/`ventanaLeida`, que ahora viven en el registro por motor; `etiquetaProveedor` → `etiquetaMotor` | completada | `arranque.go` | `go test . -count=1` |
| T-B037-14 | actualizar | `tui`: exponer en el puerto el par de la sesión y las peticiones `CambiarMotor`/`CambiarModelo`, y `RegistrarMotor`/`EditarMotor`/`EliminarMotor`/`ListarMotores`; `MotorActual()` devuelve el nombre visible de la **instancia**, no el del tipo | completada | `internal/tui/wire.go`, `internal/tui/config.go` | `go test ./internal/tui/ -count=1` |
| T-B037-15 | actualizar | `estructura_test.go`: la lista de módulos incluye `llamacpp` en lugar de `openai` y refleja que `llm` expone ahora el registro; actualizar `PROJECT.md` si el recuento de módulos cambia | completada | `internal/estructura_test.go`, `ai/docs/PROJECT.md` | `go test ./internal/ -count=1` |
| T-B037-16 | actualizar | tests: dobles de `llm.Motor` que se registran en el registro como uno real; casos de motor que no responde, tipo desconocido, registro ilegible, editar-aplicado, desactivar/reactivar, eliminar-con-sesiones, cambio de motor a mitad de conversación y dos motores generando a la vez | completada | `arranque_test.go`, `tests/e2e_localcli_test.go`, `tests/integration_errores_test.go`, `internal/llm/*_test.go`, `internal/session/*_test.go` | `go test ./... -count=1` |
| T-B037-17 | actualizar | Verificar que código y docs coinciden: `llm.Motor`, `motores.json`, `LOCALCLI_MOTOR`, migración 006, `E_MOTOR_*` y los cuatro modales | completada | `ai/docs/backend/`, `ai/docs/database/`, `ai/docs/specs/` | `go test ./internal/docs/ -count=1` |

Dependencias: T-B037-01 antes de -02, -03, -04 y -05. T-B037-03 antes de -04 y -11. T-B037-04 antes de -09 y -13. T-B037-05 antes de -09. T-B037-06 antes de -09. T-B037-07 antes de -04. T-B037-08 antes de -09. T-B037-09 antes de -10 y -14. T-B037-11 antes de -12 y -13. T-B037-15 y -16 revisan el conjunto.

## Fuera de alcance

- **El motor es un recurso de la sesión, no del arranque.** No hay que reiniciar para cambiar de motor o de modelo: se cambia con la sesión viva y la conversación sigue.
- **No hay más tipos de motor.** Añadir un runtime es código, no configuración: el catálogo es cerrado a propósito.
- **No se muta el servidor.** Ni ventana, ni modelos cargados o descargados, ni `POST /props`, ni descarga de pesos.
- **Sin proxy ni emulación de `/api/*`** sobre `llama-server`: el adaptador habla el protocolo de su motor.
- **No se recuerda qué motor respondió cada turno.** Solo el par actual de la sesión; `messages` no gana columnas ni una marca de autor por motor.
- **Sin SDK de terceros.** El SSE se lee con `bufio`; el transporte es el cliente HTTP de la estándar.
- **El registro no vive en SQLite.** Ni tabla, ni columna, ni clave foránea: `sessions.motor_id` es un `id` sin integridad referencial, a propósito.
- **Ningún módulo por debajo de `session` elige motor.** `agent`, `context`, `flow`, `queue`, `tools`, `fileops`, `exec`, `task` y `docs` no se tocan.
- **La interfaz del modal de motores es T-F044**, no esta tarea: aquí solo se expone el puerto.