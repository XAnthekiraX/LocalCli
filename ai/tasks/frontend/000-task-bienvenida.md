# 000-task-bienvenida.md

> T-F000 — Base de la TUI (modelo raíz, estilos, marco, puerto) y pantalla de bienvenida.
> Acción de esta descomposición: `crear`.

## Referencias

- [[frontend/FRONTEND]] §2 Estructura del Proyecto, §3 Contratos, §4 Decisiones
- [[frontend/01-domain/DOMAIN]] §1 Componentes (`app`, `welcome`), §3 Reglas de la bienvenida
- [[frontend/02-interfaces/INTERFACES]] §4 Teclado, §5 Estados de espera
- [[specs/SPEC-INTERFAZ]] §Pantalla de bienvenida, §Arte canónico del logotipo, §Criterios de aceptación
- [[frontend/05-quality/TESTING]] §2 Qué debe probarse (logotipo, bienvenida sin motor ni base)

## Tareas pequeñas

| ID         | Acción | Tarea                                                                                      | Estado    | Archivos                          | Verificación                                                                       |
| ---------- | ------ | ------------------------------------------------------------------------------------------ | --------- | --------------------------------- | ---------------------------------------------------------------------------------- |
| T-F000-01  | crear  | Inicializar el módulo Go y declarar Bubble Tea y Lip Gloss                                 | completada | `go.mod`, `go.sum`                | `go build ./...` compila sin errores                                               |
| T-F000-02  | crear  | Documentar el paquete `tui`: propósito y frontera                                          | completada | `internal/tui/doc.go`             | `go doc ./internal/tui` responde                                                   |
| T-F000-03  | crear  | Definir la paleta y los estilos base de Lip Gloss                                          | completada | `internal/tui/styles.go`          | Colores canónicos definidos (fondo, usuario `#606CD5`, agente `#4CEE75`, caja `#1E2030`, sutil `#828BB8`) |
| T-F000-04  | crear  | Capa común de render: normaliza el marco a `Ancho`×`Alto` celdas con fondo                 | completada | `internal/tui/marco.go`           | Función pura devuelve exactamente `Ancho`×`Alto` celdas, todas con fondo           |
| T-F000-05  | crear  | Definir el Puerto de la TUI: eventos entrantes y peticiones a `session`                    | completada | `internal/tui/wire.go`            | Compila; declara la frontera sin importar `tools` ni abrir la base                 |
| T-F000-06  | crear  | Escribir el logotipo canónico (6 filas, anchos 69/69/66/66/69/69 columnas)                  | completada | `internal/tui/testdata/logo.txt`  | Comparación byte a byte: 6 filas, sin espacios finales                                |
| T-F000-07  | crear  | Modelo raíz `app`: vista activa y enrutado de eventos y suscripciones                      | completada | `internal/tui/app.go`             | `Init`/`Update`/`View` responden; alterna bienvenida y principal                   |
| T-F000-08  | crear  | Bienvenida: logotipo centrado sobre el fondo                                               | completada | `internal/tui/welcome.go`         | El conjunto se centra en horizontal y vertical; el entorno del logo es el fondo    |
| T-F000-09  | crear  | Bienvenida: caja de entrada con placeholder y línea de estado (agente y modelo)            | completada | `internal/tui/welcome.go`         | Placeholder `"Escribe para iniciar la conversacion"` y línea `[plan] • <modelo>`   |
| T-F000-10  | crear  | Bienvenida: barra de pistas de teclado y versión                                           | completada | `internal/tui/welcome.go`         | Pistas `TAB agentes · Ctrl+P comandos · Ctrl+X M modelos` y versión abajo a la derecha |
| T-F000-11  | crear  | Bienvenida: edición de la línea de entrada en cualquier punto                             | completada | `internal/tui/welcome.go`         | Flechas y `home`/`end`/`ctrl+b`/`ctrl+e` mueven el cursor                          |
| T-F000-12  | crear  | Bienvenida: `Tab` alterna el agente activo                                                 | completada | `internal/tui/welcome.go`         | El indicador `[plan]`/`[build]` se actualiza al instante                           |
| T-F000-13  | crear  | Bienvenida: envío de la primera petición y transición a la vista principal                 | completada | `internal/tui/app.go`, `internal/tui/welcome.go` | Lo escrito viaja como primera petición; la vista cambia sin repetir ni confirmar |
| T-F000-14  | crear  | Arranque del binario                                                                       | completada | `cmd/localcli/main.go`            | `go run ./cmd/localcli` muestra la bienvenida; `Ctrl+C` sale                       |
| T-F000-15  | actualizar | Exponer los límites en pantalla (fila/columna, ancho/alto) de la caja de entrada de la bienvenida | completada | `internal/tui/components/welcome/vista.go`, `internal/tui/app.go` | La caja conoce su rectángulo en el marco en cada repintado |
| T-F000-16  | actualizar | Registrar la zona de origen del arrastre y limitar la selección a sus límites | completada | `internal/tui/selection.go`, `internal/tui/app.go` | Un arrastre que empieza en la caja de entrada no selecciona fuera de ella |
| T-F000-17  | actualizar | Anclar la selección al texto (no a coordenadas) para que un repintado no la invalide | completada | `internal/tui/selection.go`, `internal/tui/app.go` | Con la respuesta generándose token a token, el realce sigue sobre el mismo texto |
| T-F000-18  | actualizar | Probar la selección acotada y anclada (arrastre en la caja y repintado en vivo)            | completada | `internal/tui/selection_test.go`   | Un arrastre en la caja no alcanza el logotipo/pistas/versión; un repintado no mueve el realce |
| T-F000-19  | actualizar | Ajustar la bienvenida a terminales estrechas: logotipo compacto `L.CLI` y anchos acotados  | completada | `internal/tui/testdata/logo-compacto.txt`, `internal/tui/logo.go`, `internal/tui/components/welcome/welcome.go`, `internal/tui/components/welcome/vista.go`, `internal/tui/components/welcome/caja.go`, `internal/tui/render/marco.go` | Por debajo de 70 columnas se pinta el logo compacto y ninguna fila supera el ancho de la pantalla |
| T-F000-20  | actualizar | Envolver la línea de estado y conservar las opciones de aprobación en columnas estrechas | completada | `internal/tui/render/texto.go`, `internal/tui/components/input/input.go`, `internal/tui/components/welcome/caja.go`, `internal/tui/components/welcome/vista.go`, `internal/tui/approvals.go` | En una columna estrecha la línea de estado envuelve sus chapas —el interruptor `pensar` sigue visible y pulsable— y la línea de aprobación conserva siempre `aprobar \| declinar` |

**Dependencias:** `01` antes de `03`/`05`; `03`→`04`; `04`,`05`→`07`; `04`,`06`,`07`→`08`; `08`→`09`,`10`; `09`→`11`,`12`,`13`; `07`→`14`. Actualización: `15`→`16`→`17`→`18`; `19` tras `10` (parte del mismo bloque de bienvenida); `20` tras `19`.

**Nota:** la actualización documentada aquí corrige la selección dentro de la **caja de entrada de la bienvenida** (regla de zona de origen y anclaje al texto de [[specs/SPEC-INTERFAZ]]). La extensión genérica de la selección por zonas (chat y sidebar) corresponde a T-F008 (`008-task-raton.md`), aún pendiente.
