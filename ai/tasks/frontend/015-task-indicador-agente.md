> T-F015 — indicador de agente junto al input: `[plan]` / `[build]` a la izquierda de la línea de entrada y ciclo con `Tab` (acción `agent_cycle`).
> Acción de esta descomposición: `actualizar` (sobre T-F004 input, T-F003 welcome y T-F010 app).

## Referencias

- [[specs/SPEC-INTERFAZ]] — §Zonas 2 (Entrada de texto): indicador a la izquierda; criterios de `Tab`.
- [[specs/SPEC-KEYBINDS]] — acción `agent_cycle` con binding `tab`; no cicla con un modal abierto.
- [[frontend/01-domain/DOMAIN]] — componentes `input` y `welcome`.
- [[frontend/05-quality/TESTING]] — prueba del indicador y del ciclo.

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-F015-01 | actualizar | Pintar el agente activo a la izquierda del input en la vista principal (`[plan] > ` / `[build] > `), estilo Lip Gloss distinguible del texto escrito | pendiente | `internal/tui/input.go` | Golden: la línea de entrada empieza por el indicador del agente actual |
| T-F015-02 | actualizar | Mismo indicador en la línea de entrada de la bienvenida, dentro del bloque centrado | pendiente | `internal/tui/welcome.go`, testdata | Golden de bienvenida incluye `[plan] >` por defecto |
| T-F015-03 | actualizar | Enrutar `AccionCicloAgente` (`tab`): alterna plan↔build en bienvenida y vista principal; con un modal abierto la acción se ignora; la decisión de qué hace cada agente sigue en el motor | pendiente | `internal/tui/app.go`, `keyresolver.go` | Test: `tab` cambia el indicador; con modal abierto no; `tab` con el input enfocado nunca escribe un tabulador |
| T-F015-04 | actualizar | El agente elegido viaja con la petición enviada desde el input y desde la bienvenida | pendiente | `internal/tui/app.go` | Test: tras `tab` a build, la petición lleva agente build |

Dependencias: T-F015-03 depende de T-F012. El resto puede hacerse en paralelo sobre los componentes ya existentes.
