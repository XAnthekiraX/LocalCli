> T-F004 — input: línea de entrada de texto que compone y envía la petición a la sesión activa.
> Acción de esta descomposición: `crear`.

## Referencias

- [[frontend/01-domain/DOMAIN]] — §1: `input` compone y envía hacia la sesión activa; no valida reglas de negocio.
- [[specs/SPEC-INTERFAZ]] — zona 2: una sola línea, escribe hacia la sesión activa.
- [[frontend/02-interfaces/INTERFACES]] — §2 Enviar mensaje; §5: con la sesión generando, la entrada sigue operativa y no cancela nada.
- [[frontend/05-quality/TESTING]] — unitario de funciones puras y componente con arnés Bubble Tea.

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-F004-01 | crear | Modelo `input` sobre `bubbles/textinput` de una sola línea con placeholder | pendiente | `internal/tui/input.go` | `go build ./internal/tui/` sin errores |
| T-F004-02 | crear | Update que reenvía teclas imprimibles al campo y limpia/conserva el texto según el estado de espera | pendiente | `internal/tui/input.go` | Test de componente: escribir mientras genera no bloquea ni cancela |
| T-F004-03 | crear | Composición del comando «Enviar mensaje» al confirmar (Enter) solo con texto no vacío | pendiente | `internal/tui/input.go` | Test: Enter con texto emite un cmd; Enter vacío no emite |
| T-F004-04 | crear | Adaptación del ancho de la línea al layout recibido (recorte/relleno) | pendiente | `internal/tui/input.go` | Test: `SetWidth` propaga el ancho al campo |

Dependencias: T-F004-02..04 dependen de T-F004-01.
