> T-F013 — modal de modelos: la bienvenida deja de listar modelos inline; `Ctrl+X m` abre un modal con la lista de Ollama.
> Acción de esta descomposición: `actualizar` (sobre T-F003 welcome y T-F010 app).

## Referencias

- [[specs/SPEC-INTERFAZ]] — §Pantalla de bienvenida: línea de modelo + «Modal de modelos» (criterios actualizados).
- [[specs/SPEC-KEYBINDS]] — la acción `model_picker` llega resuelta por el KeyResolver.
- [[frontend/01-domain/DOMAIN]] — §3 reglas de la bienvenida actualizadas.
- [[frontend/05-quality/TESTING]] — golden de bienvenida sin lista de modelos; test del modal.

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-F013-01 | actualizar | Quitar el selector inline de `welcome.go`: la vista muestra solo la línea «modelo: X (Ctrl+X m cambiar)»; las flechas dejan de navegar modelos en la bienvenida | completada | `internal/tui/welcome.go`, `internal/tui/testdata/*` | Golden: la bienvenida no contiene la lista de modelos; escribir y enviar sigue funcionando |
| T-F013-02 | crear | Componente `modelsmodal`: modal centrado con lista, resaltado, `↑`/`↓` navegan, `Enter` aplica y cierra, `Esc` cierra sin cambios; estados «cargando…» y «sin modelos». Mecánica de modal reutilizable (estado abierto, lista, resaltado, `dismiss` con `Esc`) que heredarán los modales de sesiones y atajos (T-F014) | completada | `internal/tui/modelsmodal.go` | Test de componente: navegación, aplicar, descartar y aviso sin modelos |
| T-F013-03 | actualizar | Carga bajo demanda: al abrir el modal se pide la lista a Ollama de forma asíncrona (ya no en el arranque de la bienvenida); el mensaje de resultados rellena el modal sin bloquear la UI | completada | `internal/tui/app.go`, `arranque.go` | Test: abrir el modal emite una sola petición; la respuesta tarda y la bienvenida sigue escribiéndose |
| T-F013-04 | actualizar | Enrutar la acción `AccionModalModelos` del resolver en bienvenida y vista principal; mientras el modal está abierto su contexto captura las teclas (ninguna llega al input) | completada | `internal/tui/app.go` | Test: `ctrl+x m` abre desde ambas vistas; con el modal abierto `enter` no envía la petición |
| T-F013-05 | actualizar | El modelo aplicado viaja con la primera petición y se muestra en la línea de modelo; prevalece sobre la autodetección del arranque | completada | `internal/tui/welcome.go`, `internal/tui/app.go` | Test: elegir «qwen» en el modal y enviar produce una petición con modelo qwen |

Dependencias: T-F013-01..05 dependen de T-F012 (KeyResolver). T-F013-04 depende de T-F013-02.

## Notas

- `internal/tui/modelsmodal.go` introduce `ModelsModal`, que embebe una `Modal`
  genérica (`Modal.Abierto`, `Modal.Aviso`, `Modal.LineaActual`,
  `Modal.Mover`, `Modal.Cerrar`, `Modal.Render`). Esa mecánica compartida es la
  que heredarán los modales de sesiones y de atajos de T-F014, que ya no
  necesitan implementar su propia lista ni su propio `Render`.
- La carga es bajo demanda de verdad: `App.Init` ya no emite ningún comando de
  modelos, así que arrancar no toca Ollama. `App.cargarModelos` produce un
  único `modelosMsg` (sin `tea.Batch`, sin temporizador) que rellena el modal y
  respeta el ciclo de vida: una respuesta que llega con el modal cerrado se
  descarta.
- `Puerto` ganó `ModeloActual() string` para que la línea de modelo de la
  bienvenida se pinte sin llamar a Ollama: es una lectura en memoria del modelo
  que el arranque autodetectó o que el usuario eligió. Es un añadido a la
  interfaz, no una excepción: si aparece un tercer implementador, hay que
  implementarlo también.
- `Enter` es ambiguo y el contexto lo separa (INTERFACES §4). Para expresarlo,
  `keyresolver.go` tiene ahora `AmbitoEntrada`: `AccionEnviar` resuelve con
  cualquier contexto, y `despachar` decide qué hacer según lo que haya abierto —
  con el modal de modelos aplica el resaltado, con el selector de sesiones abre
  esa sesión y en la interfaz principal envía. Antes `AccionEnviar`Vivía en
  `AmbitoVista` y con un modal abierto no resolvía, así que el selector de
  sesiones atrapaba `enter` al final de `despachar`; con este cambio ese
  `case tea.KeyEnter` sobra y se eliminó.
- `accionesDeBienvenida` ahora declara `AccionModalModelos` y
  `AccionCerrarSelector` además de enviar y salir; las flechas siguen
  declaradas porque con el modal abierto son las suyas, pero sin modal ya no
  eligen modelo: la línea de entrada no navega nada.
- La verificación de la suite completa mantiene los dos fallos anteriores a esta
  tarea (`internal/docs` y `internal/task`), comprobados en un worktree limpio
  y ajenos a T-F013.
