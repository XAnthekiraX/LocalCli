# 014-task-tui.md

> T-B014 — tui: pantalla Bubble Tea (chat, panel, selector, aprobaciones, streaming).
> Acción de esta descomposición: `crear`.

## Referencias

- `ai/docs/specs/SPEC-INTERFAZ.md` (superficies de la pantalla)
- `ai/docs/specs/SPEC-INTERFAZ-ATAJOS.md` (atajos de teclado y panel de aprobaciones)
- `ai/docs/specs/SPEC-PANEL-CONTEXTO.md` (panel de contexto/auditoría)
- `ai/docs/backend/BACKEND.md` [29, 43] (tui→session; no decide nada, solo pinta)
- `ai/docs/backend/01-domain/DOMAIN.md` [11] (límite del módulo tui)

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-B014-01 | crear | Montar el programa Bubble Tea raíz con modelo/update/view vacío y ciclo de arranque/salida | completada | `internal/tui/app.go` | Test: tea.InitModel avanza sin pánico (teatest) |
| T-B014-02 | crear | Implementar la vista de chat: entrada de texto, mensajes propios y del agente | completada | `internal/tui/chat.go` | Test teatest: enviar texto añade burbujas en orden |
| T-B014-03 | crear | Renderizar el razonamiento en vivo desde los eventos de streaming | completada | `internal/tui/reasoning.go` | Test: tokens fake aparecen progresivamente en la vista |
| T-B014-04 | crear | Implementar el panel de datos (estado de sesión, cola, auditoría de contexto) | completada | `internal/tui/panel.go` | Test: fixture de cola renderiza filas esperadas |
| T-B014-05 | crear | Implementar el selector de sesiones (listar por carpeta, cambiar, crear, cerrar) | completada | `internal/tui/sessions.go` | Test: navegación con teclas documentada cambia de sesión |
| T-B014-06 | crear | Implementar la vista de aprobaciones con aceptar/declinar según SPEC-INTERFAZ-ATAJOS | completada | `internal/tui/approvals.go` | Test: pulsar aprobar envía la decisión a session |
| T-B014-07 | crear | Conectar los atajos de teclado declarados en la spec | completada | `internal/tui/keymap.go` | Test: cada tecla del mapa produce el msg interno correcto |
| T-B014-08 | crear | Unir tui con session por canales (sin lógica de negocio en la vista) | completada | `internal/tui/wire.go` | Test: tui compila sin importar flow/tools/store |
| T-B014-09 | crear | Escribir tests de vista con teatest sobre fixtures de eventos | completada | `internal/tui/*_test.go`, `internal/tui/testdata/` | `go test ./internal/tui/...` pasa |

**Dependencias:** 01→{02,04,05,07}; 02→03; {02}→06; {04,05,06}→08; 08→09. Requiere T-B013 completada.
