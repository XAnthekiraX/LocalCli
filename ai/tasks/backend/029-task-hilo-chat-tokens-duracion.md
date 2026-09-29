> T-B029 — el hilo del chat deja de ser efímero: las líneas de procesamiento (sub-procesos y herramientas) se guardan en `chat_evento` (migración 004) sin entrar al contexto, el turno guarda sus tokens y su duración, y el panel de contexto pasa a mostrar el total del chat.
> Acción de esta descomposición: `actualizar`.

## Referencias

- [[specs/SPEC-PANEL-CONTEXTO]] — qué significan el contexto y el consumo del turno.
- [[specs/SPEC-HISTORIAL-CONVERSACION]] — el historial que recibe el modelo.
- [[database/01-schema/TABLES]] — §`chat_evento` y §`messages`.
- [[database/01-schema/ENUMS]] — §Tipo de línea de chat.
- [[database/03-operations/MIGRATIONS]] — la migración 004.
- [[database/02-rules/DATA_FLOW]] — qué escribe un turno.
- [[backend/04-infrastructure/EVENTS]] — §3: `tokens_turno`, `etapa_iniciada`, `herramienta_resultado`.
- [[backend/DECISIONS]] — la decisión de no crecer `messages.role`.

## Contexto

El chat mostraba los sub-procesos (`[Sub Proceso] <etapa>`) y las líneas de herramienta (`✓ LEER [ruta] · 70 líneas`) solo en memoria: al cambiar de sesión o recargar se perdían. Además, el turno del agente se cerraba con los tokens en `-1` (nunca se guardaban ni se veía la duración al recargar), y el panel de contexto mostraba el consumo del **último turno**, no el contexto acumulado de la sesión, con `LimiteTokens` sin asignar (el `%` siempre era 0).

Esta tarea persiste el hilo de procesamiento en su propia tabla —nunca en el contexto—, guarda tokens y duración del turno, y cambia la semántica del panel a «total del chat que forma el contexto» (los mensajes de usuario y agente; las líneas de procesamiento no cuentan).

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-B029-01 | actualizar | `store`: tabla `chat_evento`, columna `messages.duration_ms` y migración 004 (aditiva) | completada | `internal/store/chatevento.sql`, `internal/store/schema.go`, `internal/store/migrate.go` | `TestEsquemaTablasEIndices`, `TestMigracionIdempotente` en verde |
| T-B029-02 | actualizar | `store`: repositorio de `chat_evento` (`Registrar`, `Hilo`) y lectura fusionada conversación + procesamiento en orden | completada | `internal/store/chatevento.go` | `internal/store/chatevento_test.go` en verde |
| T-B029-03 | actualizar | `store`: `CerrarTurnoAgente` acepta y persiste tokens y duración; `HistorialSesion` los lee | completada | `internal/store/messages.go`, `internal/store/repos.go` | `TestCerrarTurnoAgenteAtomico` en verde |
| T-B029-04 | actualizar | `flow`: interfaz `Registro` y llamada por etapa (iniciada y fallida); `herramientas`: el publicador guarda la línea cerrada de la herramienta | completada | `internal/flow/engine.go`, `herramientas.go` | `TestElRegistroGuardaElSubProcesoDeCadaEtapa`, `TestElRegistroMarcaLaEtapaQueFalla` en verde |
| T-B029-05 | actualizar | `arranque`: `registroPorTurno`, duración y tokens del turno, y `contextoSesion` (total del chat + límite) | completada | `arranque.go` | `chathilo_test.go` en verde; `go build ./...` |
| T-B029-06 | actualizar | Documentar: `TABLES`, `SCHEMA`, `ENUMS`, `INDEXES`, `RELATIONSHIPS`, `MIGRATIONS`, `DATA_FLOW`, `SPEC-PANEL-CONTEXTO`, `SPEC-HISTORIAL-CONVERSACION`, `EVENTS` | completada | `ai/docs/database/*`, `ai/docs/specs/SPEC-PANEL-CONTEXTO.md`, `ai/docs/specs/SPEC-HISTORIAL-CONVERSACION.md`, `ai/docs/backend/04-infrastructure/EVENTS.md` | Documentación coherente con el código |

Dependencias: T-B029-01 antes de T-B029-02. T-B029-03 y T-B029-04 antes de T-B029-05. T-B029-06 revisa el resto.

## Fuera de alcance

- **No se cambia `messages.role`.** El canal de herramientas no entra a la conversación; sus líneas de pantalla van a `chat_evento`. Crecer el `CHECK` de `role` obligaría a reconstruir `messages` con `reasoning` colgando de ella.
- **No se persisten los argumentos ni el resultado crudo de una herramienta.** Solo su línea de pantalla.
- **No se persisten los avisos ni las notificaciones** como líneas de procesamiento: solo sub-procesos y herramientas.
- **La atribución de sesión sigue usando la sesión activa del adaptador**, como el bloque de contexto y `fileops`; una limitación preexistente.
