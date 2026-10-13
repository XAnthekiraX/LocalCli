# 009-task-pruebas.md

> T-F009 — Pruebas y salidas doradas de la TUI.
> Acción de esta descomposición: `crear`.

## Referencias

- [[frontend/05-quality/TESTING]] §1 Estrategia, §2 Qué debe probarse, §3 Reglas para nuevos tests
- [[frontend/FRONTEND]] §2 Estructura del Proyecto (`testdata/`)
- [[frontend/02-interfaces/INTERFACES]] §1 Eventos que consume, §2 Peticiones que envía, §4 Teclado
- [[specs/SPEC-INTERFAZ]] §Criterios de aceptación, §Logotipo (salida dorada `logo.txt`)
- [[specs/SPEC-INTERFAZ-ATAJOS]] §Criterios de aceptación
- [[backend/05-quality/TESTING]] §estrategia global y dobles de motor

## Tareas pequeñas

| ID        | Acción | Tarea                                                                                                   | Estado    | Archivos                                    | Verificación                                                                     |
| --------- | ------ | ------------------------------------------------------------------------------------------------------- | --------- | ------------------------------------------- | -------------------------------------------------------------------------------- |
| T-F009-01 | crear  | Arnés de salidas doradas: comparar el render con `testdata/<vista>.golden` y regenerarlo con una bandera | completada | `internal/tui/golden_test.go`               | Un cambio de formato falla salvo que se regenere el dorado a propósito           |
| T-F009-02 | crear  | Salida dorada de la bienvenida (logotipo, caja de entrada, pistas y versión)                             | completada | `internal/tui/testdata/bienvenida.golden`   | El `view` de la bienvenida coincide con el archivo                               |
| T-F009-03 | crear  | Salida dorada de la vista principal (chat, entrada y sidebar)                                            | completada | `internal/tui/testdata/principal.golden`    | El `view` de la vista principal coincide con el archivo                          |
| T-F009-04 | crear  | Salida dorada de la caja de entrada al crecer en filas al desbordar el ancho                             | completada | `internal/tui/testdata/entrada.golden`      | La entrada crece en filas sin recortar lo escrito y coincide con el archivo       |
| T-F009-05 | crear  | Salida dorada del panel lateral abierto (título, contexto, TODO, tareas, estado y pie)                   | completada | `internal/tui/testdata/panel.golden`        | El panel pintado coincide con el archivo                                         |
| T-F009-06 | crear  | Salida dorada del modal de modelos                                                                       | completada | `internal/tui/testdata/modelos.golden`      | El modal de modelos coincide con el archivo                                      |
| T-F009-07 | crear  | Salida dorada del modal de sesiones                                                                      | completada | `internal/tui/testdata/sesiones.golden`     | El modal de sesiones coincide con el archivo                                     |
| T-F009-08 | crear  | Salida dorada del modal de atajos                                                                        | completada | `internal/tui/testdata/atajos.golden`       | El modal de atajos coincide con el archivo                                       |
| T-F009-09 | crear  | Prueba del logotipo byte a byte contra `testdata/logo.txt` (6 filas, anchos 69/69/66/66/69/69, sin espacios finales) | completada | `internal/tui/testdata/logo.txt`            | El logotipo coincide byte a byte con la definición canónica                       |
| T-F009-10 | crear  | Unitarias de las funciones puras de render: recorte/alineado, formato del razonamiento, contador de tokens y duraciones | completada | `internal/tui/render/`, `internal/tui/components/chat/`, `internal/tui/components/panel/` | Cada función pura con sus casos; una prueba, una regla     |
| T-F009-11 | crear  | Unitarias del reparto de la bienvenida y de la cuenta de filas envueltas de la entrada                  | completada | `internal/tui/components/welcome/`, `internal/tui/components/input/` | El reparto y el conteo se comprueban de forma aislada        |
| T-F009-12 | crear  | Unitarias del mapa de teclas: duplicados por contexto al cargar y al guardar, y persistencia del cambio | completada | `internal/tui/keys_test.go`                 | Un literal repetido en el mismo contexto se rechaza; el cambio queda persistido   |
| T-F009-13 | crear  | Pruebas de componente: `update` y `view` ante secuencias fijas de eventos, sin base ni red               | completada | `internal/tui/components/**/*_test.go`      | Cada componente reacciona a los mensajes inyectados; sin `sleep` ni esperas fijas |

**Dependencias:** `01`→`02`–`08`; `13`→`02`–`08`; `09`,`10`,`11`,`12` sin dependencias.
