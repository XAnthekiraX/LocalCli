> T-F044 — los motores se administran desde la TUI: nace el **cuarto modal** (`motorsmodal`, `Ctrl+X i`) para listar, añadir, editar, desactivar/reactivar y eliminar motores, y el motor pasa a ser una propiedad visible y cambiable de la sesión (`Ctrl+X i` + `Enter` la aplica sin perder historial). `Ctrl+X m` pasa a listar los modelos del motor de la sesión, rotulados `motor / modelo`.
> Acción de esta descomposición: `actualizar`.

## Referencias

- [[specs/SPEC-MODELO-MOTOR]] — §Gestión de motores, §Modelos, §Motor y modelo por sesión, §Cambiar a mitad de conversación y criterios de aceptación.
- [[specs/SPEC-SESIONES]] — el par motor/modelo por sesión y su supervivencia al cierre.
- [[specs/SPEC-INTERFAZ]] — §Modal de motores, §Modal de modelos, §Línea de estado, §Teclado y §Estados de espera.
- [[specs/SPEC-KEYBINDS]] — §Acción y tabla de keymap: `motor_picker`, `motor_new`, `motor_edit`, `motor_delete`.
- [[specs/SPEC-INTERFAZ-ATAJOS]] — §Modales: los cuatro con la misma mecánica y el aviso de reinicio al editar uno aplicado.
- [[frontend/01-domain/DOMAIN]] — §1 (`motorsmodal`, `modelsmodal`), §2 límites del modal, §3 reglas de la bienvenida, §4 lo que la vista nunca hace.
- [[frontend/02-interfaces/INTERFACES]] — §Teclado (tabla de atajos y §Motores), §Estados de espera.
- [[frontend/05-quality/TESTING]] — estrategia y casos de los dos modales de motor.
- [[backend/04-infrastructure/EVENTS]] — qué eventos repinta la línea de estado al cambiar.

## Contexto

Hoy hay tres modales (`modelsmodal`, `sessionsmodal`, `keysmodal`) y el proveedor no se administra en ningún sitio: se elige al arrancar y la TUI solo lo pinta. La documentación nueva pide lo contrario —el motor se **administra** desde la interfaz y, además, es de la sesión y se cambia con la sesión viva— así que esto es una vista nueva, no una ampliación del modal de modelos.

El mecanismo del modal no se inventa: los cuatro comparten la mecánica ya implementada (uno abierto a la vez, `↑`/`↓` navegan, `Enter` aplica y cierra, `Esc` descarta, las teclas no llegan a la vista de abajo).

El contrato del puerto **cambia de forma** en cuatro sitios y en ninguno cambia el resto de la vista: `ModeloLocal.Proveedor` pasa a `Motor`, `FijarModelo` se acompaña de `FijarMotor`, y entran `ListarMotores`, `FijarMotor`, `RegistrarMotor`, `EditarMotor`, `EliminarMotor` y `AlternarMotor`. Lo que llega son datos ya resueltos: la vista no conoce el archivo, el catálogo cerrado de tipos ni la cola.

El registro y la persistencia son de `llm` (T-B037), no de aquí: el modal **pinta la lista que le llega y devuelve la elección**.

Puntos de comportamiento ya fijados (no se reinventan):

- `Ctrl+X i` abre el modal de motores con **nombre, tipo y estado** de cada uno, y el de la sesión activa resaltado.
- `Enter` aplica el motor a la sesión activa y cierra. **No cambia** el identificador de la sesión, su nombre ni su historial, y no detiene el trabajo en curso.
- `n` añade (nombre, tipo, URL), `e` edita el resaltado, `Ctrl+D` elimina con confirmación **si alguna sesión lo usa**.
- Un motor desactivado se distingue de uno activo, muestra sus modelos como no disponibles y se puede reactivar.
- **Editar un motor ya aplicado avisa de que el cambio se aplica al reiniciar**; añadir, desactivar, reactivar y eliminar sí surtan efecto al momento.
- Si el motor de la sesión se desactivó o se eliminó, la línea de estado lo dice y **no bloquea**: la sesión sigue con su historial y espera.
- `Ctrl+X m` pasa a listar **los modelos del motor de la sesión**, cada fila rotulada `motor / modelo`, no los de un motor global.
- Si el modelo en curso no existe en el motor recién aplicado, el modelo queda vacío y hay que elegir otro: no se arrastra un nombre que el motor nuevo no declara.
- Al volver a una sesión se retoma contra el motor que tenía, y la línea de estado lo refleja.

Lo que **no** se toca: la mecánica de los modales existentes, la línea de entrada, el chat, el panel, las aprobaciones y el resto de la disposición. `internal/tui/` no importa `llm` ni `session` más que hoy.

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-F044-01 | actualizar | `wire.go`: `ModeloLocal.Proveedor` → `Motor string` (nombre visible de la **instancia**, no del tipo); `FijarMotor(id string)` al lado de `FijarModelo`; el puerto sigue sin exponer el archivo ni el catálogo de tipos | completada | `internal/tui/wire.go` | `go build ./internal/tui/` |
| T-F044-02 | actualizar | `wire.go`: `MotorLocal` (`ID`, `Nombre`, `Tipo`, `URL`, `Activo`, `PendienteDeReinicio`) y las peticiones `ListarMotores`, `FijarMotor`, `RegistrarMotor`, `EditarMotor`, `EliminarMotor`, `AlternarMotor`; todo llega ya resuelto desde `llm` | completada | `internal/tui/wire.go` | `go build ./internal/tui/` |
| T-F044-03 | actualizar | `keys.go`/`keyresolver.go`: acción `motor_picker` con la letra líder `i` (de inferencia), más `motor_new` (`<leader>a`), `motor_edit` (`<leader>e`) y `motor_delete` (`<leader>d`) con contexto `modal de motores`; migración de `keys.json` conservando los atajos existentes | completada | `internal/tui/keys.go`, `internal/tui/keyresolver.go` | `TestElKeymapResuelveLasCuatroAccionesDeMotor` |
| T-F044-04 | actualizar | `motorsmodal.go`: la lista con nombre, tipo y estado, el de la sesión resaltado, los desactivados distinguibles y el aviso «se aplica al reiniciar» en los editados; `↑`/`↓`, `Enter`, `Esc` y el resto de la mecánica compartida con los otros modales | completada | `internal/tui/motorsmodal.go` | `TestElModalDistingueActivoDeDesactivadoYAvisaDelReinicio` |
| T-F044-05 | actualizar | `motorsmodal.go`: alta y edición con nombre, tipo (`ollama`/`llamacpp`) y URL, con validación de URL y de nombre repetido; `Ctrl+D` pide confirmación solo si alguna sesión usa el motor | completada | `internal/tui/motorsmodal.go` | `TestAltaYEdicionValidanNombreYURL` |
| T-F044-06 | actualizar | `app.go`: `Ctrl+X i` abre y cierra el modal de motores, en la vista principal y en la bienvenida; con el modal abierto sus teclas no llegan a la vista de abajo y `Ctrl+C` sigue saliendo | completada | `internal/tui/app.go`, `internal/tui/motorsmodal.go` | `go test ./internal/tui/ -count=1` |
| T-F044-07 | actualizar | `modelsmodal.go`: `Ctrl+X m` lista los modelos **del motor de la sesión** y cada fila los rotula `motor / modelo`; si el motor no responde, sigue el aviso «sin modelos» | completada | `internal/tui/modelsmodal.go` | `TestElModalDeModelosRotulaMotorYModelo` |
| T-F044-08 | actualizar | `modelsmodal.go`: al aplicar un motor distinto, el modelo en uso se vacía si el motor nuevo no lo declara y el resaltado vuelve a la lista; no se arrastra un nombre que no existe allí | completada | `internal/tui/modelsmodal.go` | `TestCambiarDeMotorVaciaElModeloSiNoExisteAlli` |
| T-F044-09 | actualizar | `input.go`: la línea de estado muestra el **nombre de la instancia** del motor junto al modelo, y avisa —sin bloquear— si el motor de la sesión está desactivado o ya no existe | completada | `internal/tui/input.go` | `TestLaLineaDeEstadoAvisaDelMotorAusente` |
| T-F044-10 | actualizar | `config.go`: el puerto expone `ultimo_motor` para que la vista arranque en el motor de la sesión; la TUI no escribe el archivo de preferencias, solo lo pide | completada | `internal/tui/config.go` | `go build ./internal/tui/` |
| T-F044-11 | actualizar | tests: el modal de motores abre y aplica sin tocar el historial ni el nombre de la sesión, pide confirmación al borrar con sesiones, avisa del reinicio al editar un aplicado, y la navegación del teclado no filtra a la vista de abajo | completada | `internal/tui/motorsmodal_test.go`, `internal/tui/app_test.go`, `internal/tui/tui_test.go` | `go test ./internal/tui/ -count=1` |
| T-F044-12 | actualizar | tests de integración: cambiar de motor a mitad de conversación mantiene el hilo, y una sesión cuyo motor desaparece sigue existiendo con su historial y su aviso | completada | `tests/e2e_localcli_test.go` | `go test ./tests/ -count=1` |
| T-F044-13 | actualizar | Verificar que la vista y la documentación coinciden: cuatro modales, cuatro acciones de motor, rotulado `motor / modelo` y el aviso de reinicio | completada | `ai/docs/frontend/`, `ai/docs/specs/SPEC-INTERFAZ*.md`, `ai/docs/specs/SPEC-KEYBINDS.md` | `go test ./internal/docs/ -count=1` |

Dependencias: T-F044-01 y -02 antes de -04, -05, -07 y -09. T-F044-03 antes de -04, -05 y -06. T-F044-04 antes de -05 y -11. T-F044-06 antes de -07 y -11. T-F044-08 antes de -09. T-F044-11 y -12 revisan el conjunto.

## Fuera de alcance

- **El registro de motores es de `llm`.** La vista no lee `motores.json`, no conoce el catálogo cerrado de tipos ni decide qué adaptador corresponde: pide y devuelve.
- **La vista no calcula nada de motor**: qué cabe en el hardware, qué ventana hay o qué capacidades declara el modelo llegan resueltas en el puerto.
- **No se reinicia el harness** desde la interfaz: editar un motor aplicado solo avisa de que el cambio surge efecto al reiniciar.
- **No cambia la mecánica de los otros modales** ni la disposición de ninguna vista; `motorsmodal` es un modal más con la mecánica compartida.
- **La vista no borra ni invalida sesiones.** Eliminar un motor con sesiones en uso pide confirmación y deja las sesiones como están.
- **No se toca la base de datos**: el par motor/modelo lo escribe `session` (T-B037).
- **No se muestran las capacidades del motor en el modal**: solo nombre, tipo y estado. Lo que declara cada modelo sigue en el modal de modelos.