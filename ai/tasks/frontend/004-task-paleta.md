# 004-task-paleta.md

> T-F004 — Paleta de comandos de flujo.
> Acción de esta descomposición: `crear`.

## Referencias

- [[frontend/FRONTEND]] §2 Estructura del Proyecto (`comandos.go`)
- [[frontend/01-domain/DOMAIN]] §1 Componentes (`paleta`), §2 Reglas de presentación
- [[frontend/02-interfaces/INTERFACES]] §4 Teclado, §5 Estados de espera
- [[specs/SPEC-INTERFAZ]] §2 Entrada de texto (paleta de comandos)
- [[frontend/05-quality/TESTING]] §2 Qué debe probarse (paleta en bienvenida y vista)

## Tareas pequeñas

| ID         | Acción | Tarea                                                                                     | Estado    | Archivos                | Verificación                                                                      |
| ---------- | ------ | ----------------------------------------------------------------------------------------- | --------- | ----------------------- | --------------------------------------------------------------------------------- |
| T-F004-01  | crear  | Modelo de la paleta: catálogo recibido, texto de filtro y comando resaltado                | completada | `internal/tui/comandos.go` | El modelo guarda los comandos que recibe y el índice resaltado                  |
| T-F004-02  | crear  | Render de la lista desplegada encima de la caja de entrada                                 | completada | `internal/tui/comandos.go` | La lista se pinta sobre la entrada, con el resaltado en el primero                |
| T-F004-03  | crear  | Filtro de la lista por lo escrito tras `/`                                                 | completada | `internal/tui/comandos.go` | La lista se reduce a los comandos que coinciden con lo escrito                    |
| T-F004-04  | crear  | Navegación del resaltado con `↑`/`↓`                                                       | completada | `internal/tui/comandos.go` | `↑`/`↓` mueven el resaltado por la lista                                          |
| T-F004-05  | crear  | `Tab` autocompleta el comando resaltado dejando `/comando [petición]`                      | completada | `internal/tui/comandos.go` | `Tab` escribe el comando resaltado y deja la línea lista para la petición         |
| T-F004-06  | crear  | `Enter` expone el comando resaltado al `app` (sin arrancar nada)                           | completada | `internal/tui/comandos.go` | `Enter` entrega el comando resaltado al `app` y no ejecuta el flujo               |
| T-F004-07  | crear  | Aparición al escribir `/` y retirada al escribir un espacio                                | completada | `internal/tui/comandos.go` | Escribir `/` despliega la paleta; un espacio la retira                            |
| T-F004-08  | crear  | Pintado de la paleta por encima del input en la vista principal y en la bienvenida         | completada | `internal/tui/app.go`   | La paleta se pinta encima de la caja de entrada sin tapar la escritura            |
| T-F004-09  | actualizar | Máximo de elementos visibles y scroll vertical en `PaletaModel.View(ancho, maxAlto)`                  | completada | `internal/tui/comandos.go` | La paleta muestra como máximo 5 elementos a la vez y muestra `▲`/`▼` si hay overflow |
| T-F004-10  | actualizar | Render overlay en bienvenida: `vistaBienvenida` pinta la paleta superpuesta, sin empujar el logo | completada | `internal/tui/app.go` | Al escribir `/` en la bienvenida, la paleta aparece encima sin mover el logotipo ni la caja |
| T-F004-11  | actualizar | Render overlay en vista principal: `vistaPrincipal` pinta la paleta superpuesta, sin empujar el chat | completada | `internal/tui/app.go` | Al escribir `/` en la vista principal, la paleta aparece encima sin desplazar el historial |
| T-F004-12  | actualizar | Navegación scroll en paleta: `↑`/`↓` desplazan viewport al borde, `pgup`/`pgdown` y rueda del ratón | completada | `internal/tui/comandos.go`, `internal/tui/app.go`, `internal/tui/teclado.go`, `internal/tui/selection.go` | Con lista larga, `↓` en el último visible baja el viewport; `pgdown` y rueda desplazan |

**Dependencias:** `01`→`02`,`03`,`07`; `03`→`04`; `03`→`05`; `04`→`06`; `02`→`08`; `09`→`10`,`11`; `10`→`12`; `11`→`12`.
