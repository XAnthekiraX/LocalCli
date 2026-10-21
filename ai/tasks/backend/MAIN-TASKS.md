# MAIN-TASKS.md

Descomposición de la capa backend, **en el orden de construcción acordado**. Cada bloque se apoya en el anterior: no se empieza uno sin cerrar el que tiene delante.

1. **Arranque y archivo global** — validación y creación de la raíz `~/.config/localcli/`.
2. **Conexión y carga de modelos desde `models/`** — descarga, catálogo, ciclo de vida de `llama-server` y cliente de inferencia.
3. **Chat funcional** — sesión, agente, contexto y TUI.
4. **Herramientas** — catálogo, ejecución, terminal y archivos.
5. **Flujos** — orquestador de etapas, tareas y cola.
6. **Resto** — documentación, grafo de frontmatter y lo que quede pendiente.

## Bloque 1 — Arranque y archivo global

| ID     | Acción     | Tarea                                                                                                       | Dep    | Estado     | Detalle                        |
| ------ | ---------- | ----------------------------------------------------------------------------------------------------------- | ------ | ---------- | ------------------------------ |
| T-B000 | crear      | Comandos de arranque y siembra: `localcli` / `-h` / `-v` / `init` / `create`; la raíz global la crea `init` | —      | completada | `000-task-arranque-siembra.md` |
| T-B001 | actualizar | Store global: `projects` + `sessions.project_id` (v7/007) y consultas de aislamiento                        | T-B000 | completada | `001-task-store-global.md`     |
| T-B002 | actualizar | Carga aditiva de agentes y flujos (globales + propios del proyecto)                                         | T-B000 | completada | `002-task-carga-aditiva.md`    |

## Bloque 2 — Conexión y carga de modelos desde `models/`

| ID     | Acción | Tarea                                                                                                        | Dep    | Estado     | Detalle                          |
| ------ | ------ | ------------------------------------------------------------------------------------------------------------ | ------ | ---------- | -------------------------------- |
| T-B003 | crear  | Descarga de modelos GGUF desde Hugging Face (repositorio + variante, sin duplicados, reanudable)             | T-B000 | completada | `003-task-descarga-modelos.md`   |
| T-B004 | crear  | Catálogo de modelos (`models/` + `models.json`) y selección del modelo activo                                | T-B003 | completada | `004-task-catalogo-modelos.md`   |
| T-B028 | actualizar | Eliminación de modelos: borrar la carpeta `models/<id>/` y su entrada de `models.json`, con el modelo en uso protegido | T-B004 | completada | `028-task-eliminar-modelos.md`   |
| T-B005 | crear  | Ciclo de vida de `llama-server`: arranque y detención con el GGUF elegido, puerto, disponibilidad y reinicio | T-B004 | completada | `005-task-llama-server.md`       |
| T-B006 | crear  | Cliente de inferencia HTTP contra `llama-server` (núcleo OpenAI + extensión `props`, errores de conexión)    | T-B005 | completada | `006-task-cliente-inferencia.md` |
| T-B023 | actualizar | Ventana de contexto real del modelo (GGUF) y recorte efectivo del turno por ronda                       | T-B005 | completada | `023-task-ventana-contexto-real.md` |
| T-B026 | crear  | Rechazo de cabezas MTP (descarga, catálogo y arranque) y diagnóstico de caídas de `llama-server`         | T-B005 | completada | `026-task-guardia-cabezas-mtp.md`   |

## Bloque 3 — Chat funcional

| ID     | Acción | Tarea                                                                   | Dep                    | Estado     | Detalle                             |
| ------ | ------ | ----------------------------------------------------------------------- | ---------------------- | ---------- | ----------------------------------- |
| T-B007 | crear  | Sesión y memoria de conversación (crear, cambiar, historial)            | T-B006                 | completada | `007-task-sesion-memoria.md`        |
| T-B008 | crear  | Agente base y construcción de la llamada (prompt de sistema + esquemas) | T-B002, T-B013, T-B012 | completada | `008-task-agente-base.md`           |
| T-B009 | crear  | Nodo de contexto: selección y recorte por etapa                         | T-B008                 | completada  | `009-task-nodo-contexto.md`         |
| T-B010 | crear  | Persistencia de sesiones y mensajes (store)                             | T-B001                 | completada | `010-task-store-sesion-mensajes.md` |
| T-B011 | crear  | TUI de chat: bienvenida, streaming en vivo y panel                      | T-B007                 | completada | `011-task-tui-chat.md`              |
| T-B024 | actualizar | Compactación del historial: resumen persistido en la sesión, disparo por umbral y por desbordamiento, formato y modelo configurables, `/compactar` | T-B007, T-B023 | completada | `024-task-compactacion-contexto.md` |

## Bloque 4 — Herramientas

| ID     | Acción | Tarea                                                              | Dep    | Estado    | Detalle                             |
| ------ | ------ | ------------------------------------------------------------------ | ------ | --------- | ----------------------------------- |
| T-B013 | crear  | Canal de herramientas en `llm/openai` (tipos + tool_calls)         | T-B006 | completada | `013-task-canal-herramientas.md`    |
| T-B012 | crear  | Catálogo de herramientas y capa universal de ejecución             | T-B013 | completada | `012-task-catalogo-herramientas.md` |
| T-B015 | crear  | Terminal (`exec`): lista blanca y bloqueo estructural con Landlock | T-B012 | completada | `015-task-terminal.md`              |
| T-B014 | crear  | Acceso a archivos (`fileops`) y aprobaciones                       | T-B012 | completada | `014-task-fileops.md`               |
| T-B016 | crear  | Lista de pasos de la sesión (`crear_todo` / `actualizar_todo`)     | T-B012 | completada | `016-task-lista-pasos.md`           |
| T-B025 | crear  | Herramienta `diagram`: parser, IR, layout, render, handler, integración al catálogo y emisión del bloque en `herramienta_resultado` | T-B012 | pendiente  | `025-task-diagrama.md`              |
| T-B022 | actualizar | Jerarquía de un nivel en la lista de pasos: tarea principal con subtareas (`titulo`/`subtareas`, `todos.parent_position`) | T-B016 | completada | `022-task-todo-subtareas.md` |
| T-B027 | actualizar | La diferencia a la vista: diff de `editar_archivo`/`escribir_archivo`/`crear_archivo`, `bloque`/`bloque_tipo` en `herramienta_resultado` y persistencia en `chat_evento` | T-B014 | completada | `027-task-diferencia-cambios.md` |

## Bloque 5 — Flujos

| ID     | Acción | Tarea                                                                                                                                                                     | Dep            | Estado    | Detalle         |
| ------ | ------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------- | --------- | --------------- |
| T-B017 | crear  | Orquestador de etapas y flujos oficiales; entrada por comando y por la herramienta `ejecutar_flujo` (permiso derivado, sub-proceso, frontera de salida y profundidad uno) | T-B009, T-B012 | completada | `017-task-orquestador-flujos.md` |
| T-B018 | crear  | Tareas (`NNN-task-<nombre>.md`) y cola por capa                                                                                                                           | T-B017         | completada | `018-task-tareas-cola.md` |
| T-B019 | crear  | Ciclos: planificación, trabajo y resolver                                                                                                                                 | T-B018         | completada | `019-task-ciclos.md` |

## Bloque 6 — Resto

| ID      | Acción     | Tarea                                                              | Dep    | Estado    | Detalle                      |
| ------- | ---------- | ------------------------------------------------------------------ | ------ | --------- | ---------------------------- |
| T-B019b | actualizar | Eliminar columna vestigial `sessions.motor_id` (migración + store) | T-B006 | completada | `999-task-quitar-motorid.md` |
| T-B020  | crear      | Documentación (`docs`) y grafo de frontmatter                      | T-B009 | completada | `020-task-docs-grafo.md`     |
| T-B021  | crear      | Lo que quede pendiente (panel de contexto, atajos, skills)         | T-B020 | pendiente | por descomponer              |

> Los detalles de los bloques 3–6 se descomponen **uno a uno** cuando toque, con aprobación; los bloques 1–2 ya están descompuestos.
