> T-F012 — keymap central: KeyResolver con tecla líder, timeout, múltiples bindings y contexto.
> Acción de esta descomposición: `crear`.

## Referencias

- [[specs/SPEC-KEYBINDS]] — mecanismo completo: estados NORMAL/LEADER, formato de bindings, resolución por contexto, keys.json.
- [[specs/SPEC-INTERFAZ-ATAJOS]] — reglas de reasignación y no duplicados.
- [[frontend/02-interfaces/INTERFACES]] — §4 Teclado: tabla de atajos ampliada (líder + acciones nuevas).
- [[frontend/05-quality/TESTING]] — pruebas unitarias del resolver sin terminal real.

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-F012-01 | crear | IDs de acción estables (`app_exit`, `model_picker`, `session_picker`, `command_palette`, `agent_cycle`, `dismiss`, …) y estructura de binding con literales `<leader>`; migrar los nombres de `keys.go` al nuevo esquema JSON (`leader`, `leader_timeout_ms`, `keybinds` con listas) manteniendo compatibilidad de carga | completada | `internal/tui/keymap.go`, `internal/tui/keys.go` | Test: cargar un keys.json antiguo produce el mapa nuevo equivalente; serializar/deserializar es idempotente |
| T-F012-02 | crear | `KeyResolver` con estado explícito NORMAL/LEADER: la líder (`ctrl+x`) entra en LEADER, la segunda tecla resuelve la secuencia, `esc` cancela, segunda tecla desconocida vuelve a NORMAL sin emitir | completada | `internal/tui/keyresolver.go` | Tests: `ctrl+x m` → AccionModalModelos; `ctrl+x l` → AccionModalSesiones; `ctrl+x` sola → ninguna; `ctrl+x z` → ninguna y NORMAL; `esc` cancela |
| T-F012-03 | crear | Timeout de líder con temporizador de Bubble Tea (`tea.Cmd` con `time.AfterFunc`): al expirar, mensaje interno que devuelve el resolver a NORMAL y quita el indicador | completada | `internal/tui/keyresolver.go`, `internal/tui/app.go` | Test con reloj inyectado: tras el timeout configurado el estado es NORMAL y el indicador desaparece |
| T-F012-04 | crear | Resolución por contexto: orden modal → input → vista → global; el resolver recibe el contexto (modal abierto, input enfocado) y filtra qué bindings aplican; letras sueltas nunca activan acciones con el input enfocado | completada | `internal/tui/keyresolver.go` | Tests: con modal abierto `up/down/enter/esc` van al modal y `tab` no cicla agentes; escribiendo «m» con el input enfocado no abre nada; `<leader>m` funciona desde cualquier contexto |
| T-F012-05 | crear | Validación extendida: multi-binding por acción, lista vacía = deshabilitada, duplicado literal en mismo contexto rechazado, rango de timeout 500–10000 ms | completada | `internal/tui/keymap.go`, `internal/tui/keys.go` | Tests: `"app_exit": ["ctrl+c","ctrl+x q"]` válido; `"panel": []` deshabilita; duplicado devuelve error claro |
| T-F012-06 | crear | Enrutar `Update` de la app por acciones: los componentes dejan de comparar strings de tecla; la barra de estado muestra el indicador «lider » mientras dura LEADER | completada | `internal/tui/app.go`, `internal/tui/welcome.go`, `internal/tui/input.go`, `internal/tui/sessions.go`, `internal/tui/approvals.go` | Grep: ningún `msg.String() == "ctrl+…` fuera del resolver; test de componente: enviar acciones produce los mismos efectos que antes |

Dependencias: T-F012-02..05 dependen de T-F012-01; T-F012-06 depende de todas. La tarea T-F013 (modal de modelos) consume este resolver.

## Notas de alineación

- **Atajos de fábrica**: el código se alineó con [[specs/SPEC-KEYBINDS]] y [[frontend/02-interfaces/INTERFACES]] §4, que son la fuente de verdad: `app_exit` = `ctrl+c` (sin `ctrl+q`), un único `session_picker` = `<leader>l` (sin `ctrl+s` ni una acción `session_modal` duplicada) y `command_palette` = `ctrl+p` (sin `?`).
- **Nombre de la acción de sesiones**: la verificación de T-F012-02 habla de `AccionModalSesiones`; tras la alineación esa acción es `AccionSelector` con el ID estable `session_picker`, que es como la nombra la spec. `<leader>l` resuelve `AccionSelector`.
- **Temporizador de la líder**: es un `tea.Cmd` del framework (SPEC-KEYBINDS: «temporizador del framework, no con bloqueos»), implementado con una espera dentro del comando e inyectable (`KeyResolver.esperar`) para que las pruebas no dependan del reloj real. La spec no exige `time.AfterFunc`; un `AfterFunc` sin poder cancelarse dejaría temporizadores vivos tras resolver la secuencia.

