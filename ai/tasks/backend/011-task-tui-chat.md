# 011-task-tui-chat.md

> T-B011 — TUI de chat: bienvenida, streaming en vivo y panel.
> Acción de esta descomposición: `crear`.

## Referencias

- [[frontend/FRONTEND]] §1 Visión General, §2 Estructura, §4 Decisiones
- [[frontend/01-domain/DOMAIN]] §1 Componentes, §2 Reglas de presentación, §3 Reglas de la bienvenida
- [[frontend/02-interfaces/INTERFACES]] §1 Eventos que consume, §1.1 La línea de herramienta, §2 Peticiones, §3.1 Lecturas del arranque, §5 Estados de espera
- [[specs/SPEC-INTERFAZ]] — disposición de pantalla, zonas, bienvenida y cambio de sesión
- [[specs/SPEC-INTERFAZ-ATAJOS]] — `Ctrl+D` (sidebar), `Ctrl+R` (razonamiento), `Tab` (agente)
- [[backend/04-infrastructure/EVENTS]] — `token`, `estado_sesion`, `herramienta_invocada`/`resultado`
- [[database/03-operations/QUERIES]] §Historial de una sesión, §Datos del panel
- [[backend/BACKEND]] §`tui` — la TUI es presentación pura: solo pinta y envía

## Contexto

La TUI es una arquitectura Elm de Bubble Tea: un modelo de estado, un bucle de eventos (`update`) y una función de pintado (`view`). Es **presentación pura**: no habla con el motor, no escribe en SQLite ni toca archivos; recibe eventos y manda peticiones a `session`. Aquí se construye el esqueleto visible del chat: la **bienvenida**, el **streaming en vivo** dentro del hilo, la **línea de actividad** fija y el **sidebar de datos** con su pie.

Esta tarea monta la vista principal como un chat y su transición desde la bienvenida. Deja fuera los modales, el panel de aprobaciones, los atajos reasignables, la selección con ratón y los adjuntos, que llegan más adelante (panel de contexto, atajos y skills, T-B020). La paleta de comandos se alimenta del catálogo de flujos del orquestador (T-B011-09). El envío de tokens del motor y el ciclo del agente son de T-B008; la memoria de la sesión, de T-B007; el contexto, de T-B009; el historial persistido, de T-B010.

## Tareas pequeñas

| ID        | Acción | Tarea                                                                                                                                                | Estado    | Archivos                       | Verificación                                                                  |
| --------- | ------ | ---------------------------------------------------------------------------------------------------------------------------------------------------- | --------- | ------------------------------ | ----------------------------------------------------------------------------- |
| T-B011-01 | crear  | Bienvenida centrada: logotipo sobre el fondo, caja de entrada con placeholder, línea de estado (agente y modelo), barra de pistas y versión           | completada | `internal/tui/welcome.go`      | `localcli` pinta la bienvenida sin esperar al motor ni a la base              |
| T-B011-02 | crear  | Enviar desde la bienvenida crea la sesión (por el puerto) y cambia a la vista principal con la petición como primer mensaje, sin repetir ni confirmar | completada | `internal/tui/app.go`, `internal/tui/wire.go` | Enviar en la bienvenida lleva a la vista principal con la petición en el chat |
| T-B011-03 | crear  | Historial de la sesión activa: globos por autor (franja de color y fondo), líneas del sistema sueltas y orden del hilo (el texto previo a una herramienta queda arriba) | completada | `internal/tui/chat.go`, `internal/tui/marco.go` | Un texto previo a una herramienta queda en su globo arriba de su línea        |
| T-B011-04 | crear  | Streaming en vivo: cada evento `token` crece dentro del hilo como un bloque más; el hilo sigue el final y el scroll (`↑`/`↓`) lo recorre            | completada | `internal/tui/chat.go`         | La respuesta crece token a token y se recorre con el scroll                   |
| T-B011-05 | crear  | Línea de actividad fija justo encima de la caja (`[⠋ Pensando]`/`[⠋ Usando herramienta: X]`/`[⠋ Generando]`) con el tiempo transcurrido             | completada | `internal/tui/chat.go`, `internal/tui/app.go` | El indicador no se desplaza con el scroll del chat y queda fuera de su zona   |
| T-B011-06 | crear  | Razonamiento con `Ctrl+R`: indicador siempre visible, texto revelable/oculto sin detener la generación, distinguible de la respuesta                 | completada | `internal/tui/reasoning.go`    | `Ctrl+R` revela y oculta el razonamiento sin perderlo                          |
| T-B011-07 | crear  | Sidebar de datos visible por defecto y plegable con `Ctrl+D`: título (sesión y estado), CONTEXTO, TODO, LISTA DE TAREAS y ESTADO                      | completada | `internal/tui/panel.go`        | El panel muestra los datos de la sesión activa y se pliega con `Ctrl+D`        |
| T-B011-08 | crear  | Pie del panel de tres filas: git (rama activa o `sin iniciar`), ruta del proyecto abreviada con `~` y firma del harness con nombre y versión         | completada | `internal/tui/panel.go`        | En un proyecto sin git, la fila del pie dice `sin iniciar`                    |
| T-B011-09 | actualizar | Comandos slash en tres familias: `/comando` (de la interfaz, se resuelve en la vista: `models`, `sessions`, `redo`, `init`), `/flow:<nombre>` (se deja pasar al orquestador) y `/skill:<nombre>` (reconocida, solo avisa). Paleta alimentada por `Puerto.Flujos()`, no por un catálogo fijo. | completada  | `internal/tui/slash.go`, `internal/tui/comandos.go`, `internal/tui/app.go`, `internal/tui/wire.go`, `internal/session/flujo.go`, `internal/session/orquestador.go` | `/flow:planificar` arranca el flujo; `/models` abre su modal; `/skill:x` y un `/comando` desconocido solo avisan |
| T-B011-10 | actualizar | Recordar última petición de usuario (texto) en la TUI/app y lanzar `/redo` sin argumento re-ejecutándola como turno nuevo, sin historial | completada  | `internal/tui/app.go`, `internal/tui/slash.go` | Al enviar texto, el app la guarda; `/redo` la reenvía |
| T-B011-11 | actualizar | Soporte click sobre mensaje de usuario en chat para re-ejecutar ese mensaje concreto (nuevo turno, sin modificar historial)                        | pendiente  | `internal/tui/components/chat/chat.go`, `internal/tui/components/chat/vista.go`, `internal/tui/app.go` | Click en mensaje usuario reenvía ese texto como nueva petición               |
| T-B011-12 | actualizar | Pegado en input (algoritmo de tokenización): lógica de adjuntos ya implementada según especificación (1 línea tokeniza rutas; varias líneas: todas rutas → tokens por elemento; mezcla → `[PEGADO N líneas]`; archivo `[nombre.ext]`; carpeta `[CARPETA N elementos]`). | completada | `internal/tui/components/input/adjuntos.go` | Algoritmo de tokenización coincide con lo documentado                           |
| T-B011-13 | actualizar | Conectar pegado de terminal: manejar `tea.PasteMsg` en input/app y usar `InsertarPegado/AnotarPegado` para que pegados grandes/múltiples líneas se tokenicen (evitar inserción bruta con `\n`). | pendiente  | `internal/tui/components/input/input.go`, `internal/tui/app.go` | Al pegar texto multilínea aparece `[PEGADO N líneas]` o tokens por ruta según corresponda |
| T-B011-14 | crear  | Comando `/init`: inicializa el proyecto sin red —crea `AGENTS.md` y la estructura SDD (`ai/docs/`, specs/planes/tareas) y avisa del resultado—. Hoy solo se reconoce y avisa «pendiente». | pendiente  | `internal/tui/slash.go`, `internal/docs` | `/init` en un proyecto sin estructura la crea; no llega al agente |

**Dependencias:** `01`→`02`; `02`→`03`,`07`; `03`→`04`→`05`; `04`→`06`; `07`→`08`.
**Fuera de alcance:** los modales de modelos/sesiones/atajos (`Ctrl+X m`/`Ctrl+X l`/`Ctrl+P`); el panel de aprobaciones y la línea de aviso; los atajos y el `KeyResolver` con `keys.json`; la selección con ratón, el copiado y los adjuntos (tokens pegados o arrastrados); el envío de tokens y el ciclo del agente (T-B008); la memoria de la sesión (T-B007); el nodo de contexto (T-B009); el historial persistido y las consultas (T-B010); el render de contexto y la interpretación del TODO (panel de contexto, T-B020). La paleta arranca con el catálogo de flujos que sirve el orquestador y los comandos de la interfaz (T-B011-09).
