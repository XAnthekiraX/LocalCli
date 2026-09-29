> T-F032 — paleta de comandos de flujo en la entrada: escribir `/` despliega encima del input el catálogo del motor (los seis oficiales más los propios de `.localcli/flows/*.json`); `↑`/`↓` la recorren, `Tab` autocompleta el comando resaltado y `Enter` lo ejecuta.
> Acción de esta descomposición: `actualizar` (sobre T-F004 input y T-F010 app; no se modifican porque están completadas).

## Referencias

- [[specs/SPEC-INTERFAZ]] — §Zonas 2 (paleta de comandos), §Pantalla de bienvenida, §Reglas de negocio y §Criterios de aceptación.
- [[specs/SPEC-MOTOR-FLUJOS]] — un flujo no arranca solo: lo solicita un comando explícito.
- [[specs/SPEC-FLUJO-PERSONALIZADO]] — los flujos propios se declaran en `.localcli/flows/*.json` y entran en el catálogo.
- [[frontend/01-domain/DOMAIN]] — §1 (`input` y `paleta`), §2 y §3.
- [[frontend/02-interfaces/INTERFACES]] — §2 y §4: flechas, `Tab` y `Enter` con la paleta desplegada.
- [[frontend/05-quality/TESTING]] — §2: qué se prueba de la paleta.

## Tareas pequeñas

| ID        | Acción     | Tarea                                                                                                            | Estado     | Archivos                              | Verificación                                                                                                |
| --------- | ---------- | ---------------------------------------------------------------------------------------------------------------- | ---------- | ------------------------------------- | ----------------------------------------------------------------------------------------------------------- |
| T-F032-01 | actualizar | Componente `Paleta`: catálogo del puerto, filtro por lo escrito y render encima del input                        | completada | `internal/tui/comandos.go`            | Tests `TestLaPaletaSeDespliegaAlEscribirBarraYListaLosComandos` y `TestLaPaletaFiltraPorLoEscrito` en verde |
| T-F032-02 | actualizar | `Tab` autocompleta el comando resaltado y `Enter` lo ejecuta por el motor; las flechas la recorren               | completada | `internal/tui/app.go`                 | Tests `TestTabAutocompletaElComandoResaltado` y `TestEnterEjecutaElComandoYLanzaElFlujo` en verde           |
| T-F032-03 | actualizar | La paleta también se despliega en la bienvenida; ejecutar un comando crea la sesión y lleva a la vista principal | completada | `internal/tui/welcome.go`             | Test `TestLaBienvenidaDespliegaLaPaletaYEjecutaElComando` en verde                                          |
| T-F032-04 | actualizar | Catálogo desde el puerto: `Comandos()` sirve los flujos oficiales más los de `.localcli/flows/*.json`            | completada | `internal/tui/wire.go`, `arranque.go` | Test `TestLaPaletaListaElCatalogoDelPuerto` en verde                                                        |

Dependencias: T-F032-02 depende de T-F032-01; T-F032-03 depende de T-F032-01.

## Notas

- La paleta es descubrimiento, no una vía nueva de arranque (SPEC-MOTOR-FLUJOS): el flujo sigue arrancando solo con su comando explícito. `Enter` con la paleta desplegada ejecuta el comando resaltado; sin paleta, el texto se reconoce con `ComandoFlujoDe`.
- El catálogo sale del puerto (`Puerto.Comandos()`, servido por `Adaptador.Comandos` en `arranque.go`), no de una lista en la vista: los seis comandos de `comandosDeFlujo()` son solo el respaldo cuando el puerto no ofrece catálogo.
- Con la paleta desplegada, `Tab` autocompleta el comando resaltado en vez de ciclar el agente y `↑`/`↓` la recorren en vez de desplazar el historial; un espacio retira la paleta y deja paso a la petición.
- Esta entrada registra una implementación ya presente en el árbol de trabajo (commit «Añade la paleta de comandos de flujo a la entrada»); la documentación de la capa y la spec se pusieron al día en esta misma actualización.
