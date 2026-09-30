> T-B032 — cada línea de herramienta dice cuánto tardó: `chat_evento` gana `duration_ms` (migración 005), la capa universal mide la ejecución y el hilo recargado conserva el tiempo.
> Acción de esta descomposición: `actualizar`.

## Referencias

- [[specs/SPEC-TOOLS]] — §La capa universal: la medición va en el paso 5, en la capa y no en cada handler.
- [[specs/SPEC-INTERFAZ]] — §El chat: los dos tiempos, cómo se miden y que los dos son opcionales.
- [[backend/02-interfaces/TOOLS]] — §8 §Duración: qué se mide, qué no, y que el dato es de pantalla.
- [[backend/02-interfaces/dto/TOOLS-DTO]] — §4: la duración no está en `Resultado` y no viaja al modelo.
- [[backend/04-infrastructure/EVENTS]] — §3: `herramienta_resultado` lleva `duracion`.
- [[database/01-schema/TABLES]] — §`chat_evento`: `duration_ms` nullable, `NULL` es «no se midió».
- [[database/03-operations/MIGRATIONS]] — la migración 005 y el literal que el test debe seguir.
- [[database/02-rules/DATA_FLOW]] — §Creación: la línea cerrada se guarda con su duración.
- [[frontend/02-interfaces/INTERFACES]] — §1.1: las cuatro reglas de la línea de herramienta.
- [[frontend/05-quality/TESTING]] — §1: el tiempo se pinta también al fallar y se recarga con la línea.

## Contexto

`herramienta_resultado` lleva la **medida** del resultado —«70 líneas», «3 coincidencias»—, que dice cuánto devolvió la herramienta, y no cuánto tardó. El turno sí guarda su tiempo en `messages.duration_ms` desde T-B029, así que hoy el hilo dice «tardó 2 m 36 s» sin decir qué parte de eso fue una herramienta que se quedó colgada. Quien espera no puede distinguir una búsqueda de tres segundos de un `go test` de dos minutos: son la misma línea.

La documentación de este comportamiento **ya está actualizada** (esta tarea la implementa, no la describe): `SPEC-TOOLS`, `SPEC-INTERFAZ`, `TABLES`, `SCHEMA`, `MIGRATIONS`, `DATA_FLOW`, `EVENTS`, `TOOLS`, `TOOLS-DTO`, `INTERFACES`, `DOMAIN` y `TESTING`. Por eso esta descomposición no tiene subtarea de documentación.

Dos detalles que la documentación fija y que el código todavía contradice:

- `internal/tui/styles.go` afirma en el comentario de `sufijoDuracion` que «un historial recargado no la trae (la base no guarda el tiempo)». Es falso desde T-B029 para `messages`, y lo será doble con esta tarea.
- `internal/tui/chat.go` documenta la línea de herramienta con ejemplos sin duración.

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-B032-01 | actualizar | `store`: `chat_evento.duration_ms` (nullable, sin valor por defecto) y migración 005 `duracion-linea-herramienta`, aditiva y sin tocar filas; el literal `versionEsperada` de `TestMigracionIdempotente` sube a 5, como exige MIGRATIONS.md §Un `user_version` correcto no basta | completada | `internal/store/duracionlinea.sql` (nuevo), `internal/store/schema.go`, `internal/store/migrate.go`, `internal/store/store_test.go` | `TestEsquemaTablasEIndices`, `TestMigracionIdempotente` en verde |
| T-B032-02 | actualizar | `store`: `Registrar` persiste la duración de la línea; las de sub-proceso van en `NULL` («no se midió»), no en cero | completada | `internal/store/chatevento.go` | `internal/store/chatevento_test.go` en verde |
| T-B032-03 | actualizar | `tools`: `Registro.Ejecutar` mide el handler y pasa la duración a `Eventos.HerramientaResultado` en los dos caminos, el de error y el de éxito; la medición no cubre ni la búsqueda de la herramienta ni la espera de permiso | completada | `internal/tools/def.go` | `go test ./internal/tools/... -count=1` en verde |
| T-B032-04 | actualizar | cableado: `publicadorBus.HerramientaResultado` acepta la duración, la emite en el bus y compone la línea cerrada con ella antes de guardarla en `chat_evento`; `wire` la pasa del payload a la línea | completada | `herramientas.go`, `internal/tui/wire.go` | `chathilo_test.go` en verde |
| T-B032-05 | actualizar | `tui`: `LineaHerramientaCerrada` añade la duración al final con el separador de la línea (`· 0.4 s`), con el atenuado de sistema y sin paréntesis, reutilizando `formatearDuracion`; sin duración no imprime nada ni deja hueco | completada | `internal/tui/chat.go` | `internal/tui/duracion_test.go` en verde |
| T-B032-06 | actualizar | comentarios: `sufijoDuracion` ya no afirma que la base no guarda el tiempo, y el ejemplo de la línea de herramienta incluye la duración | completada | `internal/tui/styles.go`, `internal/tui/chat.go`, `herramientas.go` | Comentarios que no contradicen al código |
| T-B032-07 | actualizar | tests: el payload del evento con duración, la línea pintada con ella y la persistencia con la duración junto a la línea | completada | `internal/tools/registry_test.go`, `internal/tui/duracion_test.go`, `internal/store/chatevento_test.go`, `chathilo_test.go` | `go test ./internal/tools/ ./internal/tui/ ./internal/store/ -count=1` |

Dependencias: T-B032-01 antes de T-B032-02. T-B032-03 antes de T-B032-04, y T-B032-04 antes de T-B032-05 (la línea se compone en el cableado). T-B032-05 antes de T-B032-06. T-B032-07 revisa las tres.

## Fuera de alcance

- **El hilo recargado no necesita trabajo de TUI.** `LineaChat` solo lleva `content` y la vista lo pinta tal cual, así que la línea guardada —que ya viene compuesta con su tiempo— se muestra igual al recargar. `duration_ms` no se lee todavía por nadie.
- **La duración no se consulta ni se ordena por ella.** La columna existe para poder hacerlo sin parsear texto de pantalla; la primera consulta que la use es otra tarea.
- **No se persiste la duración de una etapa de flujo ni de un sub-proceso.** Esas líneas van en `NULL`.
- **No se mide la espera de aprobación.** El tiempo del handler no incluye el rato que el usuario tardó en contestar, que es de la persona, no de la herramienta.
- **El tiempo de la respuesta del turno no cambia.** Solo se le añade el de cada línea de herramienta, y los dos se formatean con la misma función.
- **No se toca `messages.duration_ms`**: ni su nombre, ni su semántica, ni su migración.
