> T-B020 — sesiones sin «principal»: nombre provisional y título generado por el modelo.
> Acción de esta descomposición: `actualizar`.

## Referencias

- [[specs/SPEC-SESIONES]] — §Reglas: la primera petición crea la sesión; nombre provisional y título; borrar la última vuelve a la bienvenida; el id es permanente y el nombre mutable.
- [[backend/04-infrastructure/EVENTS]] — §1/§3: evento `titulo_sesion` (`sesion`, `nombre`).
- [[backend/BACKEND]] — §3: `session` construye el historial y pide el título; `store` es el único escritor.
- [[database/01-schema/TABLES]] — `sessions.name` es el atributo que cambia; `id` no.

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-B020-01 | actualizar | `store`: `RenombrarSesion` (y método `Sesiones.Renombrar`) que cambia `name` y `updated_at` sin tocar `id` | completada | `internal/store/sessions.go` | `go build ./...` y tests de `internal/store` en verde |
| T-B020-02 | actualizar | `session.Almacen` gana `Renombrar`; `NombreProvisional`, `EsProvisional`, `PromptTitulo` y `LimpiarTitulo` | completada | `internal/session/store.go`, `internal/session/titulo.go` | Test `TestLimpiarTitulo` y `TestEsProvisional` en verde |
| T-B020-03 | actualizar | `session`: evento `EventoTituloSesion`; `Titulador` inyectable; `Conversar` pide el título con la primera petición y lo persiste, sin bloquear la respuesta | completada | `internal/session/channels.go`, `internal/session/bg.go`, `internal/session/run.go` | Tests `TestLaPrimeraPeticionGeneraElTituloDeLaSesion` y `TestUnFalloDelTituloConservaElProvisionalYReintenta` en verde |
| T-B020-04 | actualizar | Adaptador: `ResolverActiva` solo retoma (nil si no hay); `Crear` con nombre provisional; el arranque no crea sesión; `tituladorPorTurno` sobre el cliente y la FIFO | completada | `arranque.go` | `go build ./...` y tests de `localcli` en verde |
| T-B020-05 | actualizar | Configuración: la sesión sin título conserva el provisional y se reintenta | completada | `internal/session/run.go` | Test `TestSinTituladorLaSesionConservaElProvisional` en verde |

Dependencias: T-B020-03 depende de T-B020-01 y T-B020-02; T-B020-04 depende de T-B020-03; T-B020-05 depende de T-B020-03.
