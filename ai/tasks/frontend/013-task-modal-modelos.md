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
| T-F013-01 | actualizar | Quitar el selector inline de `welcome.go`: la vista muestra solo la línea «modelo: X (Ctrl+X m cambiar)»; las flechas dejan de navegar modelos en la bienvenida | pendiente | `internal/tui/welcome.go`, `internal/tui/testdata/*` | Golden: la bienvenida no contiene la lista de modelos; escribir y enviar sigue funcionando |
| T-F013-02 | crear | Componente `modelsmodal`: modal centrado con lista, resaltado, `↑`/`↓` navegan, `Enter` aplica y cierra, `Esc` cierra sin cambios; estados «cargando…» y «sin modelos». Mecánica de modal reutilizable (estado abierto, lista, resaltado, `dismiss` con `Esc`) que heredarán los modales de sesiones y atajos (T-F014) | pendiente | `internal/tui/modelsmodal.go` | Test de componente: navegación, aplicar, descartar y aviso sin modelos |
| T-F013-03 | actualizar | Carga bajo demanda: al abrir el modal se pide la lista a Ollama de forma asíncrona (ya no en el arranque de la bienvenida); el mensaje de resultados rellena el modal sin bloquear la UI | pendiente | `internal/tui/app.go`, `arranque.go` | Test: abrir el modal emite una sola petición; la respuesta tarda y la bienvenida sigue escribiéndose |
| T-F013-04 | actualizar | Enrutar la acción `AccionModalModelos` del resolver en bienvenida y vista principal; mientras el modal está abierto su contexto captura las teclas (ninguna llega al input) | pendiente | `internal/tui/app.go` | Test: `ctrl+x m` abre desde ambas vistas; con el modal abierto `enter` no envía la petición |
| T-F013-05 | actualizar | El modelo aplicado viaja con la primera petición y se muestra en la línea de modelo; prevalece sobre la autodetección del arranque | pendiente | `internal/tui/welcome.go`, `internal/tui/app.go` | Test: elegir «qwen» en el modal y enviar produce una petición con modelo qwen |

Dependencias: T-F013-01..05 dependen de T-F012 (KeyResolver). T-F013-04 depende de T-F013-02.
