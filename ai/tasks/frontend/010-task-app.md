> T-F010 — app: modelo raíz Bubble Tea, enrutado de eventos del motor, transición bienvenida↔principal y suscripciones.
> Acción de esta descomposición: `crear`.

## Referencias

- [[frontend/FRONTEND]] — §1 arquitectura Elm (un modelo, un update, un view; sin goroutines fuera de las suscripciones) y §2 `app.go`.
- [[frontend/01-domain/DOMAIN]] — §1 `app` (solo enruta, mantiene vista activa) y §3 (transición bienvenida→principal sin repetir ni confirmar).
- [[frontend/02-interfaces/INTERFACES]] — §1 tabla completa de eventos y su reacción; §4 teclado global (`?`, `Ctrl+Q`, `Ctrl+F` con confirmación); §5 pregunta al cerrar sesión con flujo en marcha.
- [[backend/04-infrastructure/EVENTS]] — payloads reales de cada evento a traducir a mensajes internos.

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-F010-01 | crear | Estructura `model` raíz con composición de componentes (welcome/input/chat/panel/sessions/approvals/notify), keys y estilos | pendiente | `internal/tui/app.go` | `go build ./internal/tui/` sin errores |
| T-F010-02 | crear | Vista inicial = bienvenida al arrancar; cambio a la interfaz principal tras enviar la primera petición, sin repetirla | pendiente | `internal/tui/app.go` | Test de componente: bienvenida → mensaje enviado → chat muestra ese primer mensaje una sola vez |
| T-F010-03 | crear | Layout: reparto de ancho entre chat y panel plegado/expandido, alto para entrada y línea de aviso | pendiente | `internal/tui/app.go` | Test golden de `WindowMsg` con panel abierto y cerrado |
| T-F010-04 | crear | Enrutado de teclado por el mapa reasignable hacia acciones globales (`Ctrl+D/S/R/A/F/Q`, `?`) y teclas locales al componente con foco | pendiente | `internal/tui/app.go` | Test: atajo reasignado dispara la misma acción |
| T-F010-05 | crear | Traducción de los 15 eventos del motor a mensajes internos y distribución según la tabla de INTERFACES §1 | pendiente | `internal/tui/app.go` | Test: cada evento llega al componente que le corresponde |
| T-F010-06 | crear | Suscripción al canal de eventos como `tea.Cmd` continuo (cola Go → msgs), sin goroutines propias fuera de ella | pendiente | `internal/tui/app.go` | Test: evento publicado en el canal aparece en el modelo |
| T-F010-07 | crear | Overlay de ayuda `?` con la lista de atajos disponibles y su acción, y cierre sin efectos | pendiente | `internal/tui/app.go` | Test golden de la pantalla de ayuda |
| T-F010-08 | crear | Confirmación de `Ctrl+F` (cancelar flujo) y presentación de la pregunta de qué hacer al cerrar sesión con flujo en marcha | pendiente | `internal/tui/app.go` | Test: cancelar emite el cmd solo tras confirmar |
| T-F010-09 | crear | Composición final de `View`: chat + entrada + panel a la derecha + selector/aprobaciones/ayuda superpuestas + línea de aviso | pendiente | `internal/tui/app.go` | Test golden de la vista completa en 80×24 |

Dependencias: T-F010-02..09 dependen de T-F010-01; T-F010-04 depende de T-F010-03; T-F010-05..06 van juntas; T-F010-07..09 dependen de T-F010-04 y T-F010-05. Requiere los componentes de T-F002..T-F009 completados.
