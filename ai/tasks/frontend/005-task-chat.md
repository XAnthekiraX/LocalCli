> T-F005 — chat: historial de la sesión activa con razonamiento en vivo arriba de la respuesta y propuestas pendientes.
> Acción de esta descomposición: `crear`.

## Referencias

- [[frontend/01-domain/DOMAIN]] — §1 `chat` y §2: razonamiento en vivo, arriba, distinguible y ocultable sin detener la generación.
- [[specs/SPEC-INTERFAZ]] — zona 1: historial de la sesión activa, intercambio razonamiento+respuesta, propuestas pendientes.
- [[frontend/02-interfaces/INTERFACES]] — §1 eventos `token`, `etapa_*`, `peticion_aprobacion`; roles `user`/`agent` ([[database/01-schema/ENUMS]]).
- [[frontend/05-quality/TESTING]] — comparaciones doradas de formato y prueba «una regla, un test».

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-F005-01 | crear | Tipos del historial: intercambio (mensaje usuario, razonamiento, respuesta) y cola de mensajes pintables | completada | `internal/tui/chat.go` | `go build ./internal/tui/` sin errores |
| T-F005-02 | crear | Acumulación en vivo de `token` en el bloque de razonamiento o de respuesta según su marca | completada | `internal/tui/chat.go` | Test: secuencia de tokens → bloques correctos (chat_test.go) |
| T-F005-03 | crear | Render del intercambio con razonamiento arriba de la respuesta y separación visual garantizada | completada | `internal/tui/chat.go` + `styles.go` | Test golden: razonamiento nunca mezclado con la respuesta |
| T-F005-04 | crear | Ocultar/mostrar razonamiento (`Ctrl+R`) sin detener la acumulación ni alterar la generación | completada | `internal/tui/chat.go` + `app.go` | Test: oculto sigue acumulando, al mostrar aparece todo |
| T-F005-05 | crear | Carga del historial de la sesión activa y reset al cambio (lectura vía `Puerto.Historial`, sin tocar segundo plano) | completada | `chat.go`, `wire.go`, `app.go` | Test: cambiar de sesión carga ese historial; tarde se descarta |
| T-F005-06 | crear | Línea de propuesta pendiente en el chat de la sesión activa; `aprobacion_resuelta` la retira | completada | `chat.go`, `wire.go` | Test: propuesta se ve en chat; resuelta sale; otra sesión no se ve |
| T-F005-07 | crear | Scroll/recorte del chat al alto disponible conservando el final visible | completada | `chat.go` (`recortarAlto`), `app.go` | Test: contenido mayor que la ventana recorta por arriba |

Dependencias: T-F005-02..07 dependen de T-F005-01; T-F005-03..04 dependen de T-F005-02; T-F005-05 depende de T-F005-03.
