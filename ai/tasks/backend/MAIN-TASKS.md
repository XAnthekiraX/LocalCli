# MAIN-TASKS — Backend LocalCli

Fuente de verdad del progreso del backend. Derivada exclusivamente de `ai/docs/` (BACKEND.md, DECISIONS.md, 01-domain, 02-interfaces, 03-security, 04-infrastructure, 05-quality, database/ y specs/).

| ID | Acción | Tarea | Dep | Estado | Detalle |
|----|--------|-------|-----|--------|---------|
| T-B000 | crear | Configuración del stack Go: go.mod, dependencias fijas y binario localcli | — | completada | `000-task-stack.md` |
| T-B001 | crear | Estructura de carpetas base: los 13 módulos de internal/ con límites por paquete | T-B000 | completada | `001-task-estructura.md` |
| T-B002 | crear | store: SQLite único acceso (esquema 6 tablas, WAL, foreign_keys, user_version, transacciones) | T-B000 | completada | `002-task-store.md` |
| T-B003 | crear | docs: carga de documentación, frontmatter y grafo en memoria | T-B001 | completada | `003-task-docs.md` |
| T-B004 | crear | task: lectura/escritura del TODO (MAIN-TASKS.md y NNN-task-*.md) con su frontmatter | T-B001 | completada | `004-task-task.md` |
| T-B005 | crear | ollama: cliente HTTP streaming, razonamiento, perfil de hardware y serialización FIFO | T-B001 | completada | `005-task-ollama.md` |
| T-B007 | crear | tools: registro de las 13 herramientas, comprobación de permiso y enrutado (+ DTOs) | T-B001 | completada | `007-task-tools.md` |
| T-B006 | crear | agent: agentes JSON (plan/build), catálogo cerrado y despacho de herramientas a tools | T-B001, T-B007 | completada | `006-task-agent.md` |
| T-B008 | crear | fileops: operaciones de archivo, frontera de rutas, aprobación y change_history | T-B002, T-B007 | completada | `008-task-fileops.md` |
| T-B009 | crear | exec: terminal con lista blanca, límites (120s/10KB) y bloqueo Landlock | T-B007 | completada | `009-task-exec.md` |
| T-B010 | crear | flow: motor de etapas, detección de trabajo ordenado y encadenamiento de ciclos | T-B003, T-B004, T-B005, T-B006 | completada | `010-task-flow.md` |
| T-B011 | crear | context: nodo de contexto (selección por modelo, recorte y auditoría) | T-B003, T-B005 | completada | `011-task-context.md` |
| T-B012 | crear | queue: cola derivada del TODO, orden por dependencias y elementos bloqueados | T-B004, T-B010 | completada | `012-task-queue.md` |
| T-B013 | crear | session: ciclo de vida de sesiones, segundo plano, estados y notificaciones | T-B002, T-B010 | completada | `013-task-session.md` |
| T-B014 | crear | tui: pantalla Bubble Tea (chat, panel, selector, aprobaciones, streaming) | T-B013 | completada | `014-task-tui.md` |
| T-B015 | crear | Validaciones finales: suite de tests por regla/invariante y verificación integral | T-B002..T-B014 | completada | `015-task-validaciones.md` |
| T-B016 | actualizar | chat normal por defecto y flujos solo con comando explícito; el chat usa las herramientas del agente activo | T-B006, T-B010, T-B013 | completada | `016-task-chat-y-flujos.md` |
| T-B017 | actualizar | permisos explícitos por acción con catálogo derivado; ciclo conversacional único en `agent` y catálogo inyectado en el mensaje de sistema | T-B006, T-B007 | completada | `017-task-permisos-y-ciclo-agente.md` |
| T-B018 | actualizar | ollama: capacidades del modelo (`/api/show`) y `SinHerramientas` en la lista del modal | T-B005 | completada | `018-task-ollama-capacidades.md` |
| T-B019 | actualizar | preferencias de usuario: el arranque reutiliza el último modelo y expone el último agente por el puerto | T-B005 | completada | `019-task-preferencias-usuario.md` |
| T-B020 | actualizar | sesiones sin «principal»: nombre provisional y título generado por el modelo; `ResolverActiva` solo retoma | T-B002, T-B013 | completada | `020-task-sesiones-titulos.md` |
| T-B021 | actualizar | historial de conversación al modelo con compactación por presupuesto | T-B006, T-B010, T-B013 | completada | `021-task-historial-conversacion.md` |
| T-B022 | actualizar | imágenes por turno de chat: `ollama.Mensaje.Images` (base64 en `/api/chat`), `PuedeVer` y el transporte efímero por `agent`/`flow`/`session` | T-B005, T-B018, T-B021 | completada | `022-task-imagenes-ollama.md` |
| T-B023 | actualizar | agentes configurables: cargar todos los `ai/agents/*.json` (base + propios), orden estable de nombres y validación relajada en `flow` | T-B006, T-B010, T-B018 | completada | `023-task-agentes-configurables.md` |

## Referencias

- [[backend/BACKEND]] — mapa de la capa y estructura de proyecto (sección 2).
- [[backend/DECISIONS]] — decisiones confirmadas, no se reinventan.
- [[database/DATABASE]] — capa de datos.
- [[specs/SPEC-INTERFAZ]] — superficie que consume `tui`.
