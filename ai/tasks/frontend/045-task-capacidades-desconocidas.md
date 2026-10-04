> T-F045 — la vista aprende a decir **«no lo sé»**: las capacidades del modelo pasan a tres estados, y donde el estado es `desconocida` la TUI **no ofrece** herramientas ni visión, las **marca con `?`** y avisa de que el razonamiento está desactivado porque no se pudo comprobar. En el modal de motores, el campo `extensiones` es editable y visible, y la bienvenida explica cómo activar el runtime en una instalación sin motores.
> Acción de esta descomposición: `actualizar`.

## Referencias

- [[specs/SPEC-MODELO-MOTOR]] — §Qué se sabe y qué no (los tres estados y el `?`), §Ningún motor activo por defecto y criterios de aceptación.
- [[specs/SPEC-INTERFAZ]] — el rotulado `?` en la línea de estado, el aviso de razonamiento desconocido y la bienvenida sin motores.
- [[frontend/02-interfaces/INTERFACES]] — §3.2 Lecturas al motor: la tabla de los tres estados y qué se pinta en cada uno.
- [[frontend/01-domain/DOMAIN]] — §2 límites del modal y §4 lo que la vista nunca hace.
- [[frontend/05-quality/TESTING]] — los casos de `?`, del aviso y del modal con extensiones.
- [[specs/SPEC-KEYBINDS]] — §Acción y tabla de keymap: las acciones de motor de T-F044, sin ninguna nueva aquí.
- [[specs/SPEC-INTERFAZ-ATAJOS]] — §Modales: la mecánica compartida, incluida la validación del campo nuevo.
- [[backend/02-interfaces/INTERFACES-GENERAL]] — el tipo `Capacidad` de tres estados que llega al puerto.
- [[backend/04-infrastructure/CONFIGURATION]] — §8 el campo `extensiones` del registro y el ejemplo con `activo: false`.

## Contexto

T-F044 dejó el modal de motores y la línea de estado nombrando motor y modelo. Pero la vista sigue suponiendo que **toda capacidad es sí o no**, y eso es justo lo que la arquitectura nueva retira: una extensión que no se declara no es una capacidad ausente. Si la vista no cambia, aparecen los dos fallos que la tríada evita —ofrecer herramientas que quizá no existen, o esconderlas sin explicación—, y ninguno se puede distinguir de un límite real del modelo.

El `?` no es decoración: es el estado de primera clase que permite a la TUI pintar «no lo sé». Sin él, «no lo soporta» y «no lo sé» son la misma línea, y el usuario no puede distinguir el motivo.

Aquí no se toca la mecánica de los modales ni la disposición de las vistas. El cambio es de **datos que llegan y de cómo se pintan**, más el campo `extensiones` en el alta y la edición del modal.

Lo que llega por el puerto cambia de forma en un sitio: `tui.Capacidades` pasa de banderas booleanas a tres estados. Todo lo demás —`ModeloLocal.Motor`, las peticiones de `ListarMotores` a `AlternarMotor`, la navegación, el reinicio— sigue como lo dejó T-F044.

El tipo de motor, el catálogo cerrado, la cola y el registro siguen siendo de `llm` (T-B038): la vista **pinta la lista que le llega y devuelve la elección**.

Puntos de comportamiento ya fijados (no se reinventan):

- Herramientas `desconocida`: **no se ofrecen** y se marcan con `?`. No se pide al usuario que adivine si funcionan.
- Visión `desconocida`: **no se ofrece** adjuntar imágenes y la opción sale con `?`.
- Razonamiento `desconocida`: **se desactiva** y se avisa. Es el único de los tres que genera aviso, porque afecta al turno en curso y no bastaría con una marca: el usuario tiene que saber por qué el modelo está pensando menos.
- `no soportada` **no lleva `?`**: ahí sí hay dato, y la ausencia es información.
- El `?` se pinta junto al nombre de la capacidad en el pie del sidebar y en la línea de estado, no dentro del mensaje: es un atributo del modelo, no del turno.
- La vista **no decide** qué hacer con un estado desconocido de razonamiento: lo recibe ya resuelto y lo pinta. La decisión es del backend.
- La ventana de contexto sigue la misma historia: si el motor no la declara, el panel marca el límite como el del harness, no como el del modelo.
- El campo `extensiones` es **editable y visible** en el alta y la edición del modal de motores, con los valores válidos del tipo elegido, y una línea que explique que sin extensión declarada lo que no se informa queda desconocido.
- Un motor de la entrega con `activo: false` se ve **inactivo** como cualquier otro: no hay un estado «de fábrica» que la vista tenga que conocer.
- La bienvenida, cuando no hay ningún motor activo, **no bloquea** y **no inventa uno**: dice que no hay ningún motor dado de alta y ofrece abrir el modal. El resto de la interfaz sigue viva.

Lo que **no** se toca: la mecánica compartida de los modales, el chat, el panel, las aprobaciones, la caja de entrada, el selector de sesiones y el resto de la disposición. `internal/tui/` no importa `llm` ni `session` más que hoy.

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-F045-01 | actualizar | `wire.go`: `tui.Capacidades` pasa de banderas booleanas a los tres estados (`Soportada`, `NoSoportada`, `Desconocida`) y llega por el puerto ya resuelto; el tipo en sí no lo calcula la vista | completada | `internal/tui/wire.go` | `go build ./internal/tui/` |
| T-F045-02 | actualizar | `wire.go`: `MotorLocal` gana `Extensiones []string`, y las peticiones de alta y edición lo aceptan; la vista no decide qué extensión puede declarar cada tipo | completada | `internal/tui/wire.go` | `go build ./internal/tui/` |
| T-F045-03 | actualizar | `motorsmodal.go`: alta y edición con el campo **`extensiones`**, editable y con los valores válidos del tipo elegido, más la línea que explica que sin extensión declarada lo que no se informa queda desconocido; se valida contra el tipo, sin lista cerrada en la vista | completada | `internal/tui/motorsmodal.go` | `TestElAltaValidaLasExtensionesDelTipo` |
| T-F045-04 | actualizar | `motorsmodal.go`: la lista muestra las extensiones declaradas de cada motor y distingue la lista vacía —«solo núcleo común»— de una declarada; el motor de la entrega aparece inactivo como cualquier otro, sin estado de fábrica | completada | `internal/tui/motorsmodal.go`, `internal/tui/motorsmodal_test.go` | `TestLaListaDistingueSinExtensionesDeConExtensiones` |
| T-F045-05 | actualizar | `app.go` y el pie del sidebar: las capacidades se pintan con `?` cuando el estado es `desconocida`, y sin marca cuando es `no soportada`; el `?` va junto al nombre, no dentro del mensaje | completada | `internal/tui/app.go`, `internal/tui/modelsmodal.go` | `TestCapacidadDesconocidaSePintaConInterrogacionYNoSoportadaSin` |
| T-F045-06 | actualizar | razonamiento `desconocida`: la TUI **no ofrece** el `think` y **muestra el aviso** de que está desactivado porque no se pudo comprobar; es el único aviso de los tres | completada | `internal/tui/input.go`, `internal/tui/app.go` | `TestElThinkDesconocidoSeDesactivaYAvisa` |
| T-F045-07 | actualizar | `modelsmodal.go`: herramientas y visión `desconocida` **no se ofrecen** para adjuntar, con `?` en su rótulo; `no soportada` se oculta sin marca | completada | `internal/tui/modelsmodal.go` | `TestLasCapacidadesDesconocidasNoSeOfrecen` |
| T-F045-08 | actualizar | `welcome.go`: sin ningún motor activo la bienvenida **no bloquea ni inventa**: dice que no hay motor dado de alta y ofrece abrir el modal (`Ctrl+X i`); el resto de la interfaz sigue viva y se puede escribir | completada | `internal/tui/welcome.go`, `internal/tui/app.go` | `TestLaBienvenidaSinMotoresOfreceAbrirElModal` |
| T-F045-09 | actualizar | `input.go` y el panel: si la ventana no la declara el motor, el límite se marca como el del harness, no como el del modelo; es el mismo `?` de las capacidades | completada | `internal/tui/input.go` | `TestElLimiteDelHarnessSeMarcaComoTal` |
| T-F045-10 | actualizar | tests: los tres estados de cada capacidad pintan lo que corresponde, el aviso de razonamiento aparece solo en `desconocida`, el modal valida `extensiones` y la bienvenida sin motores no bloquea | completada | `internal/tui/motorsmodal_test.go`, `internal/tui/app_test.go`, `internal/tui/modales_test.go`, `internal/tui/golden_test.go`, `internal/tui/testdata/atajos.golden` | `go test ./internal/tui/ -count=1` |

Dependencias: T-F045-01 antes de -05, -06, -07 y -09. T-F045-02 antes de -03 y -04. T-F045-03 antes de -04 y -10.

## Fuera de alcance

- **No se decide el comportamiento de una capacidad desconocida.** Lo recibe resuelto por el puerto y lo pinta; la regla (no ofrecer, marcar, avisar) viene del backend (T-B038-10).
- **La vista no lee `motores.json`**, no conoce el archivo de ejemplo ni decide qué motor se activa: pide y devuelve.
- **La vista no valida qué extensión puede declarar cada tipo** con una lista propia: valida contra lo que el puerto dice, no contra una copia de la regla.
- **No se comprueba el encaje con el hardware** ni se avisa de que un modelo no cabe: sin la extensión `tags` no hay dato, y la vista no lo inventa.
- **No cambia la mecánica de los modales** ni la disposición de ninguna vista.
- **No se añaden atajos**: las acciones de motor de T-F044 siguen siendo las cuatro.
- **No se toca la base de datos** ni el historial de las sesiones.