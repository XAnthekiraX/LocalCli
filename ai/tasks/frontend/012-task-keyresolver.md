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
| T-F012-01 | crear | IDs de acción estables (`app_exit`, `model_picker`, `session_picker`, `command_palette`, `agent_cycle`, `dismiss`, …) y estructura de binding con literales `<leader>`; migrar los nombres de `keys.go` al nuevo esquema JSON (`leader`, `leader_timeout_ms`, `keybinds` con listas) manteniendo compatibilidad de carga | pendiente | `internal/tui/keymap.go`, `internal/tui/keys.go` | Test: cargar un keys.json antiguo produce el mapa nuevo equivalente; serializar/deserializar es idempotente |
| T-F012-02 | crear | `KeyResolver` con estado explícito NORMAL/LEADER: la líder (`ctrl+x`) entra en LEADER, la segunda tecla resuelve la secuencia, `esc` cancela, segunda tecla desconocida vuelve a NORMAL sin emitir | pendiente | `internal/tui/keyresolver.go` | Tests: `ctrl+x m` → AccionModalModelos; `ctrl+x l` → AccionModalSesiones; `ctrl+x` sola → ninguna; `ctrl+x z` → ninguna y NORMAL; `esc` cancela |
| T-F012-03 | crear | Timeout de líder con temporizador de Bubble Tea (`tea.Cmd` con `time.AfterFunc`): al expirar, mensaje interno que devuelve el resolver a NORMAL y quita el indicador | pendiente | `internal/tui/keyresolver.go`, `internal/tui/app.go` | Test con reloj inyectado: tras el timeout configurado el estado es NORMAL y el indicador desaparece |
| T-F012-04 | crear | Resolución por contexto: orden modal → input → vista → global; el resolver recibe el contexto (modal abierto, input enfocado) y filtra qué bindings aplican; letras sueltas nunca activan acciones con el input enfocado | pendiente | `internal/tui/keyresolver.go` | Tests: con modal abierto `up/down/enter/esc` van al modal y `tab` no cicla agentes; escribiendo «m» con el input enfocado no abre nada; `<leader>m` funciona desde cualquier contexto |
| T-F012-05 | crear | Validación extendida: multi-binding por acción, lista vacía = deshabilitada, duplicado literal en mismo contexto rechazado, rango de timeout 500–10000 ms | pendiente | `internal/tui/keymap.go`, `internal/tui/keys.go` | Tests: `"app_exit": ["ctrl+c","ctrl+x q"]` válido; `"panel": []` deshabilita; duplicado devuelve error claro |
| T-F012-06 | crear | Enrutar `Update` de la app por acciones: los componentes dejan de comparar strings de tecla; la barra de estado muestra el indicador «lider » mientras dura LEADER | pendiente | `internal/tui/app.go`, `internal/tui/welcome.go`, `internal/tui/input.go`, `internal/tui/sessions.go`, `internal/tui/approvals.go` | Grep: ningún `msg.String() == "ctrl+…` fuera del resolver; test de componente: enviar acciones produce los mismos efectos que antes |

Dependencias: T-F012-02..05 dependen de T-F012-01; T-F012-06 depende de todas. La tarea T-F013 (modal de modelos) consume este resolver.
