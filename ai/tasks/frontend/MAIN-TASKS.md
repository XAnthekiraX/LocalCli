# MAIN-TASKS — Frontend LocalCli (TUI)

Fuente de verdad del progreso del frontend. Derivada exclusivamente de `ai/docs/` (frontend/FRONTEND.md, 01-domain, 02-interfaces, 05-quality, specs/SPEC-INTERFAZ, specs/SPEC-INTERFAZ-ATAJOS y backend/04-infrastructure/EVENTS). La TUI es Go + Bubble Tea + Lip Gloss en `internal/tui/`; no hay stack web que instalar ni configurar.

| ID | Acción | Tarea | Dep | Estado | Detalle |
|----|--------|-------|-----|--------|---------|
| T-F000 | crear | Verificación del estado base: `internal/tui/` con su testdata/logo.txt dorado y compilación del paquete | — | completada | `000-task-base.md` |
| T-F001 | crear | keys: mapa de teclas reasignable con valores por defecto, carga/guardado en ~/.config/localcli/keys.json y rechazo de duplicados | T-F000 | completada | `001-task-keys.md` |
| T-F002 | crear | styles: estilos Lip Gloss y render de bloques (razonamiento distinguible de la respuesta) | T-F000 | completada | `002-task-styles.md` |
| T-F003 | crear | welcome: pantalla de bienvenida con logotipo ASCII dorado, nombre con versión y primera petición | T-F000, T-F002 | completada | `003-task-welcome.md` |
| T-F004 | crear | input: línea de entrada de texto que compone y envía la petición a la sesión activa | T-F001 | completada | `004-task-input.md` |
| T-F005 | crear | chat: historial de la sesión activa con razonamiento en vivo arriba de la respuesta y propuestas pendientes | T-F002, T-F004 | completada | `005-task-chat.md` |
| T-F006 | crear | panel: panel de datos plegable con los nueve datos de la sesión activa | T-F002, T-F005 | completada | `006-task-panel.md` |
| T-F007 | crear | sessions: selector momentáneo con nombre y estado de todas las sesiones del proyecto | T-F001, T-F005 | completada | `007-task-sessions.md` |
| T-F008 | crear | approvals: panel global de aprobaciones pendientes, resolubles una a una (a/d) con líneas obsoletas | T-F001, T-F005 | completada | `008-task-approvals.md` |
| T-F009 | crear | notify: línea de aviso de aprobaciones pendientes visible con el panel cerrado | T-F008 | completada | `009-task-notify.md` |
| T-F010 | crear | app: modelo raíz Bubble Tea, enrutado de eventos del motor, transición bienvenida↔principal y suscripciones | T-F001..T-F009 | completada | `010-task-app.md` |
| T-F011 | crear | Pruebas y validaciones: unitarias de render, de componente con arnés Bubble Tea y verificación de criterios de aceptación | T-F010 | completada | `011-task-tests.md` |
| T-F012 | crear | keymap central: KeyResolver con tecla líder (Ctrl+X), timeout 2000 ms, múltiples bindings por acción y resolución por contexto; migración de keys.json | T-F001, T-F010 | completada | `012-task-keyresolver.md` |
| T-F013 | actualizar | modal de modelos: la bienvenida muestra solo el modelo en uso; Ctrl+X m abre un modal con la lista de Ollama cargada bajo demanda | T-F012 | completada | `013-task-modal-modelos.md` |
| T-F014 | actualizar | modales de sesiones y atajos: Ctrl+X l abre el modal de sesiones (Enter abre la elegida), Ctrl+P el listado de atajos, Esc cierra cualquiera | T-F012, T-F013 | completada | `014-task-modales-sesiones-atajos.md` |
| T-F015 | actualizar | indicador de agente a la izquierda del input ([plan]/[build]) en bienvenida y vista principal; Tab alterna el agente | T-F012 | completada | `015-task-indicador-agente.md` |
| T-F016 | actualizar | modal de sesiones en la bienvenida: `Ctrl+X l` lo abre y al elegir una sesión la vista pasa a la principal con su historial | T-F003, T-F010 | completada | `016-task-sesiones-bienvenida.md` |
| T-F017 | actualizar | atajos de sesión: `Ctrl+X n` crea una sesión y la deja activa; `Ctrl+D` elimina la resaltada en el modal, pidiendo confirmación si trabaja | T-F014 | completada | `017-task-sesiones-ctrl.md` |
| T-F018 | actualizar | input: edición del cursor (flechas, home/end) sin perder los atajos de la app | T-F004 | completada | `018-task-edicion-input.md` |
| T-F019 | actualizar | chat: scroll del historial (ventana con seguimiento del final) y sus atajos ↑/↓ y PgUp/PgDn | T-F005 | completada | `019-task-scroll-chat.md` |
| T-F020 | actualizar | modal de modelos: resalta el modelo en uso, marca los que no declaran herramientas y avisa al elegirlos | T-F013 | completada | `020-task-modal-modelos-foco.md` |
| T-F021 | actualizar | preferencias de usuario: arranca con el último agente y persiste modelo/agente en `~/.config/localcli/config.json` | T-F004, T-F010 | completada | `021-task-preferencias-tui.md` |
| T-F022 | actualizar | modal de atajos: agrupado por categorías y con tecla y descripción alineadas | T-F014 | completada | `022-task-modal-atajos-orden.md` |
| T-F023 | actualizar | bienvenida: la línea de entrada se edita en cualquier punto (flechas, home/end) | T-F003 | completada | `023-task-bienvenida-edicion.md` |
| T-F024 | actualizar | ratón: rueda para el historial, arrastre para seleccionar y copia al portapapeles | T-F005 | completada | `024-task-raton-seleccion-copia.md` |
| T-F025 | actualizar | línea de estado bajo el input: modelo en uso y acceso a herramientas | T-F004, T-F020 | completada | `025-task-estado-modelo.md` |
| T-F026 | actualizar | doble `esc` para cancelar el trabajo en curso, con confirmación | T-F010 | completada | `026-task-doble-esc-cancelar.md` |
| T-F027 | actualizar | la bienvenida crea la sesión; borrar la última vuelve a la bienvenida; evento `titulo_sesion` | T-F003, T-F014, T-F017 | completada | `027-task-sesiones-bienvenida-titulo.md` |
| T-F028 | actualizar | aprobaciones: el panel se muestra sin robar el teclado; `Ctrl+A` enfoca (`a`/`d`) y un clic sobre «aprobar»/«declinar» decide | T-F008 | completada | `028-task-aprobaciones-input-raton.md` |

## Referencias

- [[frontend/FRONTEND]] — mapa de la capa, estructura de `internal/tui/` (sección 2) y contratos (sección 3).
- [[frontend/01-domain/DOMAIN]] — componentes, límites y reglas de presentación/bienvenida.
- [[frontend/02-interfaces/INTERFACES]] — eventos consumidos, peticiones a `session`, lecturas a `store` y teclado.
- [[frontend/05-quality/TESTING]] — estrategia de pruebas y salidas doradas.
- [[specs/SPEC-INTERFAZ]] — disposición, zonas, bienvenida y arte canónico del logotipo.
- [[specs/SPEC-INTERFAZ-ATAJOS]] — reglas de atajos y panel de aprobaciones.
- [[specs/SPEC-KEYBINDS]] — keymap central, leader key, timeout y resolución por contexto.
- [[backend/04-infrastructure/EVENTS]] — eventos que produce el motor (productor de la TUI).
- [[PROJECT]] — stack aprobado: Bubble Tea + Lip Gloss, un único binario Go.
