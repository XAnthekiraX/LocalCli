# 013-task-session.md

> T-B013 — session: ciclo de vida de sesiones, segundo plano, estados y notificaciones.
> Acción de esta descomposición: `crear`.

## Referencias

- `ai/docs/specs/SPEC-SESIONES.md` [19-78] (flujo, ejemplo, flujos alternativos, reglas)
- `ai/docs/backend/01-domain/BUSINESS_RULES.md` [7-23] (reglas de sesiones y notificaciones)
- `ai/docs/backend/BACKEND.md` [43, 92] (session→flow; una goroutine por sesión en ejecución)
- `ai/docs/backend/04-infrastructure/EVENTS.md` [27-53] (productores/consumidores de eventos)

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-B013-01 | crear | Definir el tipo Sesión con nombre, capa y estados del ciclo crear/cambiar/retomar/cerrar | pendiente | `internal/session/model.go` | Test: transiciones inválidas rechazadas |
| T-B013-02 | crear | Persistir sesiones y mensajes vía store (crear, listar por carpeta, borrar en cascada) | pendiente | `internal/session/store.go` | Test: borrar sesión elimina sus messages/reasoning/approvals |
| T-B013-03 | crear | Aislar sesiones por carpeta: las de otra carpeta nunca aparecen | pendiente | `internal/session/scope.go` | Test: dos proyectos temporales no comparten listado |
| T-B013-04 | crear | Ejecutar cada sesión activa en su goroutine de segundo plano que sobrevive al cambio de vista | pendiente | `internal/session/bg.go` | Test: la sesión sigue avanzando mientras la vista está en otra |
| T-B013-05 | crear | Arrancar el flujo de la sesión delegando en flow (session no ejecuta tareas) | pendiente | `internal/session/run.go` | Test con stub flow: enviar mensaje invoca flow una vez |
| T-B013-06 | crear | Gestionar pausar/cancelar/re-derivar de la cola en curso | pendiente | `internal/session/pause.go` | Test: pausar detiene tras el elemento en curso |
| T-B013-07 | crear | Emitir notificaciones según BUSINESS_RULES (término, espera de aprobación, fallo) | pendiente | `internal/session/notify.go` | Test: cada disparador produce su evento de EVENTS.md |
| T-B013-08 | crear | Exponer canales hacia tui (mensajes, estado, razonamiento en vivo) | pendiente | `internal/session/channels.go` | Test: el consumidor recibe los eventos en orden |
| T-B013-09 | crear | Escribir tests del módulo con flow y store simulados/temporales | pendiente | `internal/session/*_test.go` | `go test ./internal/session/...` pasa |

**Dependencias:** 01→{02,03}; {02}→05; 05→{04,06,08}; {04}→07; {06,07,08}→09. Requiere T-B002 y T-B010 completadas.
