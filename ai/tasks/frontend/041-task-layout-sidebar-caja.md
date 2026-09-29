> T-F041 — rediseño de la vista principal: chat con burbujas e icono, sidebar separado por línea vertical y caja de entrada con borde y pie (agente · modelo).
> Acción de esta descomposición: `actualizar`.

## Referencias

- [[specs/SPEC-INTERFAZ]] — §Disposición, §Zonas 1 (chat), §Zonas 2 (entrada) y §Zonas 3 (sidebar).
- [[frontend/01-domain/DOMAIN]] — §1 (`chat`, `input`, `panel`) y §2 (reglas de presentación).
- [[frontend/02-interfaces/INTERFACES]] — §4 (teclado) y §5 (estados de espera).
- [[frontend/FRONTEND]] — §2 (estructura de `internal/tui/`).

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-F041-01 | actualizar | Chat: cada mensaje con un icono al lado (delante el agente, detrás el usuario, globo pegado a la derecha) | completada | `internal/tui/chat.go`, `internal/tui/styles.go` | Test `TestElChatPintaCadaMensajeEnSuGlobo` |
| T-F041-02 | actualizar | Sidebar: título de la conversación, CONTEXTO, TODO, LISTA DE TAREAS, ESTADO (git, capa y cola, aprobaciones, agente, proyecto) y la ruta al pie; recortar a una línea | completada | `internal/tui/panel.go`, `internal/tui/styles.go` | Tests `TestElPanelMuestraLosNueveDatos` y `TestElPanelPintaLaListaDePasosDelAgente`; `testdata/panel.golden` |
| T-F041-03 | actualizar | Caja de entrada con borde, con el texto y el pie (agente, modelo y capacidades); tokens justo debajo | completada | `internal/tui/input.go`, `internal/tui/styles.go`, `internal/tui/app.go` | Tests de `input_test.go`, `estado_modelo_test.go` y `agente_test.go`; `testdata/entrada.golden` |
| T-F041-04 | actualizar | Layout: columna principal y sidebar unidos por una línea vertical; divisor horizontal entre chat y caja; todo un solo marco | completada | `internal/tui/app.go` | Test `TestElMarcoCabeEnLaTerminal`; `testdata/principal.golden` |
| T-F041-05 | actualizar | Ratón: la banda del chat exige la columna principal, y la selección cruza chat y sidebar (marco único) | completada | `internal/tui/selection.go` | Test `TestLaSeleccionCruzaChatYSidebar` |
| T-F041-06 | actualizar | Documentar el rediseño en las fuentes de la capa | completada | `ai/docs/specs/SPEC-INTERFAZ.md`, `ai/docs/frontend/01-domain/DOMAIN.md`, `ai/docs/frontend/02-interfaces/INTERFACES.md`, `ai/docs/frontend/FRONTEND.md` | `go build ./...`, `go vet`, `gofmt`, `go test ./... -count=1` |
| T-F041-07 | actualizar | Márgenes a los lados de los globos y del chat; la caja de entrada queda siempre pegada al pie aunque el historial sea corto | completada | `internal/tui/chat.go`, `internal/tui/styles.go`, `internal/tui/app.go` | Tests `TestLosGlobosDejanMargenALosLados` y `TestLaCajaDeEntradaQuedaPegadaAbajo` |
| T-F041-08 | actualizar | Recorte y envoltura por ancho de pantalla (no por runas): un emoji o un ideograma ya no desalinea el divisor del sidebar | completada | `internal/tui/styles.go`, `internal/tui/selection.go` | Test `TestLasColumnasQuedanAlineadasConAnchoDePantalla` |
| T-F041-09 | actualizar | Resolver: las pruebas que seguían suponiendo el sidebar cerrado (recorrido, arnés, apertura/cierre, atajo reasignado, bienvenida) y el anclaje al scroll, que comparaba la fila entera con el texto del sidebar | completada | `internal/tui/app_test.go`, `internal/tui/testing_test.go`, `internal/tui/tui_test.go`, `internal/tui/welcome_test.go`, `internal/tui/selection_test.go`; `ai/docs/specs/SPEC-INTERFAZ.md`, `ai/docs/frontend/01-domain/DOMAIN.md`, `ai/docs/frontend/02-interfaces/INTERFACES.md` | `go test ./... -count=1` |
