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
| T-B023 | actualizar | agentes configurables: cargar todos los `.localcli/agents/*.json` (base + propios), orden estable de nombres y validación relajada en `flow` | T-B006, T-B010, T-B018 | completada | `023-task-agentes-configurables.md` |
| T-B024 | actualizar | herramientas nativas y extensibles: canal `tools` de `/api/chat`, capa universal de ejecución y herramientas del usuario en `.localcli/tools/` | T-B005, T-B006, T-B007, T-B017, T-B018 | completada | `024-task-tools-nativas-y-extensibles.md` |
| T-B025 | actualizar | TODO del agente: herramienta `actualizar_todo`, acción `tareas`, tabla `todos` (migración 002) y evento `todo_actualizada` | T-B002, T-B006, T-B007 | completada | `025-task-todo-agente.md` |
| T-B026 | actualizar | línea de herramienta: `herramienta_invocada` lleva el verbo y el tema (el objetivo del catálogo) y `herramienta_resultado` la medida, en vez del resumen ciego de argumentos | T-B024 | completada | `026-task-linea-herramienta.md` |
| T-B027 | actualizar | sub-procesos encadenados: cada etapa devuelve un resumen corto que se encadena, las intermedias sin aprobación corren en silencio (`[Sub Proceso] <nombre>`) y el chat muestra la entrega del último paso | T-B010, T-B013, T-B014 | completada | `027-task-resolver-subprocesos.md` |
| T-B028 | actualizar | `/resolver` determinista: bloque de contexto optimizado por fase en `flow_context` (migración 003), composición final sin herramientas y regla de la ruta raíz de `AGENTS.md` | T-B010, T-B014, T-B027 | completada | `028-task-bloque-contexto-resolver.md` |
| T-B029 | actualizar | hilo del chat persistente: líneas de procesamiento en `chat_evento` (migración 004) sin entrar al contexto, tokens y duración del turno en `messages`, y `contextoSesion` para el panel | T-B002, T-B010, T-B013, T-B014, T-B024, T-B027 | completada | `029-task-hilo-chat-tokens-duracion.md` |
| T-B030 | actualizar | los agentes base se cargan de `.localcli/agents/` y la carpeta se versiona con el proyecto: `.gitignore` ignora solo el SQLite, y código, specs y tareas nombran la ruta nueva | T-B006, T-B023, T-B024 | completada | `030-task-agentes-en-localcli.md` |
| T-B031 | actualizar | la etapa responde a su pregunta (brief + ventana con pregunta/respuesta; reintento y `E_STAGE_FAILED` si sigue muda) y los flujos son dato: fuera los flujos cableados y la cola resuelve su flujo del catálogo | T-B010, T-B012, T-B013, T-B027 | completada | `031-task-etapa-responde-y-flujos-dato.md` |
| T-B032 | actualizar | duración de cada línea de herramienta: `chat_evento.duration_ms` (migración 005), medición en la capa universal, `herramienta_resultado.duracion` y tiempo en la línea pintada y recargada | T-B024, T-B026, T-B029 | completada | `032-task-tiempo-linea-herramienta.md` |
| T-B033 | actualizar | un turno nunca se cierra mudo: la redacción final se reintenta una vez si vuelve a pedir herramientas y, si sigue sin texto, el turno falla con el motivo a la vista en vez de guardar `(respuesta vacía)` | T-B006, T-B031 | completada | `033-task-turno-sin-respuesta.md` |
| T-B034 | actualizar | el agente pasa a ser una carpeta: `agent.yaml` (nombre, descripción y permisos de tres categorías `read`/`write`/`edit`, sin `skills` ni `mode`) más `prompt.md`; el `*.json` antiguo se avisa nombrándolo y los agentes base se migran | T-B006, T-B007, T-B017, T-B023, T-B030 | completada | `034-task-agentes-config-y-prompt.md` |
| T-B035 | actualizar | la lista de pasos de la sesión tiene dos vías: `crear_todo` añade un paso al final y `actualizar_todo` sustituye la lista entera; mismo vocabulario, misma lista devuelta y sin migración | T-B002, T-B006, T-B007, T-B025 | completada | `035-task-crear-todo.md` |
| T-B036 | actualizar | proveedor de modelo intercambiable: frontera `llm.Proveedor`, adaptador OpenAI-compatible (llama.cpp) y `LOCALCLI_PROVEEDOR` | T-B005, T-B018, T-B019, T-B022, T-B024 | completada | `036-task-proveedor-modelo.md` |

## Referencias

- [[backend/BACKEND]] — mapa de la capa y estructura de proyecto (sección 2).
- [[backend/DECISIONS]] — decisiones confirmadas, no se reinventan.
- [[database/DATABASE]] — capa de datos.
- [[specs/SPEC-INTERFAZ]] — superficie que consume `tui`.
