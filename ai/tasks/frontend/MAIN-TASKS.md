# MAIN-TASKS — Frontend LocalCli (TUI)

Fuente de verdad del progreso del frontend. Derivada exclusivamente de `ai/docs/` (frontend/FRONTEND.md, 01-domain, 02-interfaces, 05-quality, specs/SPEC-INTERFAZ, specs/SPEC-INTERFAZ-ATAJOS y backend/04-infrastructure/EVENTS). La TUI es Go + Bubble Tea + Lip Gloss en `internal/tui/`; no hay stack web que instalar ni configurar.

| ID | Acción | Tarea | Dep | Estado | Detalle |
|----|--------|-------|-----|--------|---------|
| T-F000 | crear | Verificación del estado base: `internal/tui/` con su testdata/logo.txt dorado y compilación del paquete | — | completada | `000-task-base.md` |
| T-F001 | crear | keys: mapa de teclas reasignable con valores por defecto, carga/guardado en ~/.config/localcli/keys.json y rechazo de duplicados | T-F000 | completada | `001-task-keys.md` |
| T-F002 | crear | styles: estilos Lip Gloss y render de bloques (razonamiento distinguible de la respuesta) | T-F000 | en_progreso | `002-task-styles.md` |
| T-F003 | crear | welcome: pantalla de bienvenida con logotipo ASCII dorado, nombre con versión y primera petición | T-F000, T-F002 | pendiente | `003-task-welcome.md` |
| T-F004 | crear | input: línea de entrada de texto que compone y envía la petición a la sesión activa | T-F001 | pendiente | `004-task-input.md` |
| T-F005 | crear | chat: historial de la sesión activa con razonamiento en vivo arriba de la respuesta y propuestas pendientes | T-F002, T-F004 | pendiente | `005-task-chat.md` |
| T-F006 | crear | panel: panel de datos plegable con los nueve datos de la sesión activa | T-F002, T-F005 | pendiente | `006-task-panel.md` |
| T-F007 | crear | sessions: selector momentáneo con nombre y estado de todas las sesiones del proyecto | T-F001, T-F005 | pendiente | `007-task-sessions.md` |
| T-F008 | crear | approvals: panel global de aprobaciones pendientes, resolubles una a una (a/d) con líneas obsoletas | T-F001, T-F005 | pendiente | `008-task-approvals.md` |
| T-F009 | crear | notify: línea de aviso de aprobaciones pendientes visible con el panel cerrado | T-F008 | pendiente | `009-task-notify.md` |
| T-F010 | crear | app: modelo raíz Bubble Tea, enrutado de eventos del motor, transición bienvenida↔principal y suscripciones | T-F001..T-F009 | pendiente | `010-task-app.md` |
| T-F011 | crear | Pruebas y validaciones: unitarias de render, de componente con arnés Bubble Tea y verificación de criterios de aceptación | T-F010 | pendiente | `011-task-tests.md` |

## Referencias

- [[frontend/FRONTEND]] — mapa de la capa, estructura de `internal/tui/` (sección 2) y contratos (sección 3).
- [[frontend/01-domain/DOMAIN]] — componentes, límites y reglas de presentación/bienvenida.
- [[frontend/02-interfaces/INTERFACES]] — eventos consumidos, peticiones a `session`, lecturas a `store` y teclado.
- [[frontend/05-quality/TESTING]] — estrategia de pruebas y salidas doradas.
- [[specs/SPEC-INTERFAZ]] — disposición, zonas, bienvenida y arte canónico del logotipo.
- [[specs/SPEC-INTERFAZ-ATAJOS]] — reglas de atajos y panel de aprobaciones.
- [[backend/04-infrastructure/EVENTS]] — eventos que produce el motor (productor de la TUI).
- [[PROJECT]] — stack aprobado: Bubble Tea + Lip Gloss, un único binario Go.
