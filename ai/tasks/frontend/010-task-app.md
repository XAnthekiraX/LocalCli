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
| T-F010-01 | crear | Estructura `model` raíz con composición de componentes (welcome/input/chat/panel/sessions/approvals/notify), keys y estilos | completada | `internal/tui/app.go` | `go build ./internal/tui/` sin errores |
| T-F010-02 | crear | Vista inicial = bienvenida al arrancar; cambio a la interfaz principal tras enviar la primera petición, sin repetirla | completada | `app.go`, `welcome.go` (T-F003) | Tests T-F003: primer mensaje una sola vez |
| T-F010-03 | crear | Layout: reparto de ancho entre chat y panel plegado/expandido, alto para entrada y línea de aviso | completada | `app.go` (WindowSizeMsg + AccionPanel, T-F006) | Tests: entrada cede y recupera el ancho |
| T-F010-04 | crear | Enrutado de teclado por el mapa reasignable alineado con INTERFACES §4 (`Ctrl+D/S/R/A/F/Q`, `?`) y persistido en `~/.config/localcli/keys.json` | completada | `keymap.go`, `keys.go`, `app.go`, `welcome.go` | Test: atajo reasignado dispara la misma acción; ida y vuelta del JSON |
| T-F010-05 | crear | Traducción de los eventos del motor a mensajes internos y distribución según la tabla de INTERFACES §1 | completada | `wire.go` (incluye `flujo_pausado/reanudado/cancelado`) | Test: los eventos del canal llegan a la vista |
| T-F010-06 | crear | Suscripción al canal de eventos como `tea.Cmd` continuo (cola Go → msgs), sin goroutines propias fuera de ella | completada | `wire.go` (`escucharCmd`), `app.go` (`Init` + re-arma en `eventoMsg`) | Test: suscripción única, re-armada tras cada evento |
| T-F010-07 | crear | Overlay de ayuda `?` con la lista de atajos disponibles y su acción, y cierre sin efectos | completada | `app.go` (`viewAyuda`) | Test: lista cada atajo, cierra con cualquier tecla sin escribir |
| T-F010-08 | crear | Confirmación de `Ctrl+F` (cancelar flujo) antes de cortar el trabajo | completada | `app.go` (`PidiendoCancelar`) | Test: cancela solo tras confirmar; el no descarta |
| T-F010-09 | crear | Composición final de `View`: chat + entrada + panel a la derecha + selector/aprobaciones/ayuda superpuestas + línea de aviso | completada | `app.go` (`viewPrincipal`, `View`) | Test de composición con panel abierto y cerrado |

Dependencias: T-F010-02..09 dependen de T-F010-01; T-F010-04 depende de T-F010-03; T-F010-05..06 van juntas; T-F010-07..09 dependen de T-F010-04 y T-F010-05. Requiere los componentes de T-F002..T-F009 completados.
