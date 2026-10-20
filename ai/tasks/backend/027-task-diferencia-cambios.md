# 027-task-diferencia-cambios.md

> T-B027 — La diferencia del cambio: calcular el diff al aplicar una escritura, emitirlo con la línea de herramienta y persistirlo en `chat_evento`.
> Acción de esta descomposición: `actualizar`.

## Referencias

- [[specs/SPEC-TOOLS]] §La diferencia a la vista.
- [[specs/SPEC-ARCHIVOS]] §Reglas de negocio — «Cada cambio aplicado deja ver el antes y el después».
- [[backend/02-interfaces/TOOLS]] §La diferencia que se ve y §Lo que se ve en la TUI.
- [[backend/02-interfaces/dto/TOOLS-DTO]] §Escritura de archivos.
- [[backend/04-infrastructure/EVENTS]] §3 `herramienta_resultado` (`bloque`, `bloque_tipo`) y §4.
- [[database/01-schema/TABLES]] §`chat_evento`, [[database/01-schema/ENUMS]] (tipo de bloque) y [[database/01-schema/CONSTRAINTS]].
- [[database/03-operations/MIGRATIONS]] §`013-bloque-linea`.
- [[database/02-rules/DATA_FLOW]] §Creación.
- [[backend/DECISIONS]] — la fila de la diferencia como bloque.

## Contexto

`fileops` ya tiene el antes y el después en memoria al aplicar una escritura y los guarda
en `change_history`, pero la pantalla solo recibe la confirmación corta. Esta tarea
calcula el diff en el backend, lo adjunta al evento y lo persiste con la línea, para que
la TUI solo tenga que pintarlo.

El bloque visual bajo la línea ya existe en el chat (T-F015): `Linea.Bloque` se pinta
debajo y la línea conserva su texto. Esta tarea añade el productor y el tipo:
`bloque` y `bloque_tipo` (`diagrama` | `diferencia`) viajan en
`herramienta_resultado`, se persisten en `chat_evento` y se usan para las
herramientas de escritura. T-B025 —la herramienta `diagram`— sigue pendiente y,
cuando se implemente, solo tendrá que emitir su bloque con
`bloque_tipo = "diagrama"`.

## Tareas pequeñas

| ID        | Acción     | Tarea                                                                                                                                                                                                 | Estado    | Archivos                                                                 | Verificación                                                                                   |
| --------- | ---------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------- | ------------------------------------------------------------------------ | ---------------------------------------------------------------------------------------------- |
| T-B027-01 | actualizar | Implementar el diff por líneas con 3 de contexto, cabeceras `@@ -a,b +c,d @@`, marcas `-`/`+` y tope de 40 líneas con `… N líneas más`; sin dependencias nuevas y con el texto normalizado (CRLF/BOM) | completada | `internal/fileops/diferencia.go` (nuevo), `internal/fileops/diferencia_test.go` | Un cambio a media línea, un archivo nuevo y un cambio mayor que el tope producen el diff esperado |
| T-B027-02 | actualizar | Adjuntar el bloque al resultado de `editar_archivo`, `escribir_archivo` y `crear_archivo` en `Meta["bloque"]` con `Meta["bloque_tipo"] = "diferencia"`, solo después de aplicar y nunca en errores ni borrados | completada | `internal/fileops/escribir.go`                                            | Cada escritura aplicada devuelve el bloque; un fallo o un borrado no lo devuelven             |
| T-B027-03 | actualizar | Propagar `bloque` y `bloque_tipo` por la capa universal hasta el evento `herramienta_resultado`, sin sumarlos a la salida que recibe el modelo ni al recorte | completada | `internal/tools/registro.go`, `internal/tools/eventos.go`, `internal/session/orquestador.go` | El evento llega con el bloque; el resultado reenviado al modelo no cambia                     |
| T-B027-04 | actualizar | Migración `013-bloque-linea`: columnas `chat_evento.bloque` y `chat_evento.bloque_tipo` (CHECK `diagrama`/`diferencia`), `user_version = 13`, y lectura/escritura en `store` (crear, listar, deshacer y bifurcar) y en `Historial` | completada | `internal/store/schema.go`, `internal/store/chat_evento.go`, `internal/session/orquestador.go`, `internal/tui/wire.go` | El bloque sobrevive al recuperar la sesión; las filas antiguas quedan sin bloque y se pintan igual |
| T-B027-05 | actualizar | Pruebas de integración de la escritura: el evento lleva el bloque, un cambio fallido no lo lleva y la sesión recuperada conserva el bloque con su línea | completada | `internal/fileops/*_test.go`, `internal/session/*_test.go`, `internal/store/*_test.go` | Las tres rutas quedan cubiertas sin tocar las pruebas de otras herramientas           |

**Dependencias:** `01` → `02` → `03`; `04` en paralelo con `01–03`; `05` cierra la integración.

**Fuera de alcance:** mostrar la diferencia en el panel de aprobaciones (solo aplica a
rutas fuera del proyecto), el diff de los borrados y cualquier editor de cambios en la
TUI.

## Notas

- El bloque es de pantalla: **no** entra al contexto del modelo ni se reenvía. La salida
  que recibe el modelo sigue siendo la confirmación corta.
- El antes y el después completos no se duplican: siguen en `change_history`; en
  `chat_evento` se guarda solo el bloque ya recortado.
- `bloque` y `bloque_tipo` se escriben juntos o no se escriben; la base valida el
  vocabulario del tipo, la pareja la garantiza el código.
- T-B025 sigue pendiente y no se toca aquí: la herramienta `diagram` no está
  implementada, pero el mecanismo de bloque (`bloque`, `bloque_tipo`, persistencia y
  pintado) queda listo para que la emita con `bloque_tipo = "diagrama"`.
