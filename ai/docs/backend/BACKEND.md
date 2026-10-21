---
title: LocalCli — capa backend
tags: [backend, arquitectura]
depende_de:
  - "[[PROJECT]]"
  - "[[backend/DECISIONS]]"
relacionado:
  - "[[frontend/FRONTEND]]"
  - "[[database/DATABASE]]"
---

# BACKEND — LocalCli

Capa de orquestador de `LocalCli`: un proceso único de terminal, en Go, que ejecuta trabajos de varias etapas por ti. No es un servidor web: no expone HTTP, no tiene usuarios, no tiene login. La TUI (`tui`) es su único consumidor.

Este documento es el mapa de navegación del backend. Para los datos, mira [[database/DATABASE]]; para la pantalla, [[frontend/FRONTEND]].

## 1. Visión General

El backend se divide en dos planos, y la separación es deliberada:

- **Lo que vive en archivos.** La documentación del proyecto y los archivos de tarea. Se editan a mano, se versionan en git, se leen sin la herramienta. Fuentes de verdad. Ver [[backend/01-domain/DOMAIN]].
- **Lo que vive en SQLite.** El estado de ejecución: sesiones, mensajes, razonamiento, aprobaciones, auditoría de contexto e historial de cambios. Ver [[database/DATABASE]].

Sobre esos dos planos trabajan quince módulos, cada uno con una responsabilidad única:

| Grupo | Módulos | Qué hacen |
|---|---|---|
| Interfaz y ciclo | `tui`, `session`, `flow` | Reciben lo que escribes, encadenan etapas y muestran el resultado |
| Contexto e inteligencia | `context`, `agent`, `llm`, `openai`, `llamacpp` | Deciden qué se le entrega al modelo, quién es el agente y cómo se le habla |
| Ejecución | `tools`, `fileops`, `exec` | Aplican lo que el agente pide, dentro de las fronteras permitidas |
| Trabajo en serie | `queue`, `task` | Ordenan las tareas grandes y las ejecutan solas |
| Datos | `store`, `docs` | Persisten y leen lo que hace falta para saber qué está pasando |

## 2. Estructura del Proyecto

```
localcli           comando global para arrancar la aplicación
internal/
  tui/             pantalla Bubble Tea: chat, panel, selector, aprobaciones
  session/         ciclo de vida de sesiones y ejecución en segundo plano
  flow/            orquestador de etapas y encadenamiento
  queue/           cola de trabajo en forma de TODO y su orden
  context/         grafo de frontmatter, selección de contexto y auditoría
  agent/           agentes desde carpeta: agent.yaml + prompt.md, permisos y ciclo conversacional
  llm/             frontera neutra del modelo: tipos, capacidades de tres estados, cola de inferencia y regla de la ventana
  llm/openai/      núcleo común: transporte HTTP y SSE contra /v1/chat/completions y /v1/models
  llm/llamacpp/    extensión nativa de llamacpp (llama-server): /props
  task/            archivos de tarea
  tools/           catálogo, capa universal de ejecución y hooks
  fileops/         validación de rutas, acceso a archivos e historial
  exec/            terminal: confirmación selectiva (destructivo o fuera) y bloqueo estructural
  store/           persistencia SQLite
  docs/            carga de documentación y lectura de frontmatter
```

## 3. Responsabilidades de los Componentes

Cada módulo tiene un límite. Ver [[backend/01-domain/DOMAIN]] para el detalle.

- **`tui`** — Toda la pantalla. Recibe teclado, muestra chat, panel de datos, selector de sesiones, aprobaciones y razonamiento en vivo. No decide nada: solo pinta lo que le llega y manda lo que pulsas.
- **`session`** — Ciclo de vida de las sesiones: crear, cambiar, retomar, cerrar, y su estado. Corre en segundo plano aunque cambies de vista. Reconstruye el historial de conversación de la sesión y lo compacta (resumen) cuando no cabe en el presupuesto antes de cada turno de chat, y pide al modelo el título de la sesión con su primera petición. No ejecuta tareas; eso es `flow` y `queue`.
- **`flow`** — Orquestador de etapas. Encadena las etapas de los ciclos oficiales (planificación, trabajo, resolver), decide si sigue, si para o si espera tu aprobación. No sabe de herramientas ni de SQL.
- **`queue`** — Cola de la ejecución en curso. Consume el TODO que creó esta solicitud, respeta su orden y marca lo bloqueado. No inventa trabajo: el TODO se lo creó `flow` al detectar tu petición.
- **`context`** — Nodo de contexto. Lee el grafo de dependencias, deja que el modelo decida qué es relevante, recorta hasta el límite y registra qué entró y qué salió. No llama al modelo por su cuenta.
- **`agent`** — Carga las definiciones de agente desde carpeta: cada agente es un directorio con un `agent.yaml` (identidad y permisos) y un `prompt.md` (instrucciones del modelo), ambos obligatorios. Los base (`plan`, `build`, `chat`) se siembran desde el binario en `~/.config/localcli/agents/`; los propios del proyecto se añaden en `<proyecto>/.localcli/agents/` de forma aditiva. El nombre lo decide `name`, y lo que no se nombre en `permissions` queda denegado porque `default` solo admite `deny`; un campo desconocido —`skills`, `herramientas`, `prompt`— o un `default: allow` hacen que el agente no cargue. Corre el ciclo conversacional (modelo → herramienta → resultado → modelo, con un máximo de rondas) y antepone al contexto del turno el historial de conversación que le entrega `flow`. Si un turno de herramientas acaba en fallos corregibles, `agent` lo empuja a continuar en vez de rendirse: inyecta un mensaje de sistema con la instrucción de corregir o probar otra herramienta, escalando hasta tres empujones —el último reintroduce la petición original del usuario con sus rutas—, y un acierto de herramienta vuelve a dar el control al modelo. Construye la petición al modelo con los esquemas de las herramientas del agente activo y no las ejecuta: se las pide a `tools`. Si el modelo en uso no admite herramientas, el agente cae a modo conversación. No hay skills: la capacidad no está implementada.
- **`llm`** — La frontera neutra con el modelo: los tipos (`Mensaje`, `Herramienta`, `Evento`, `Modelo`), el tipo `Capacidad` de tres estados (`soportada`, `no soportada`, `desconocida`), una cola de inferencia única y la regla de la ventana. Ni `agent` ni `context` saben en qué formato viaja la respuesta: hablan con esta frontera. El motor es único —llama.cpp—, así que no hay registro de motores, ni adaptador por tipo, ni motor por sesión: el proveedor se declara una vez en `localcli.json`. Ver [[specs/SPEC-MODELOS-LLAMACPP]].
- **`llm/openai`** — El núcleo común con `llama-server`: transporte HTTP, lectura SSE de `/v1/chat/completions` y listado en `/v1/models`. Aquí viven el streaming, el razonamiento, las imágenes en `image_url`, el canal estructurado de herramientas y los tokens de uso. Habla el protocolo OpenAI-compatible que expone `llama-server`; no sabe de runtimes concretos.
- **`llm/llamacpp`** — Extensión nativa de llama.cpp, para `llama-server`. No implementa la conversación, que es del núcleo común: aporta `/props`, de donde lee `default_generation_settings.n_ctx` y las capacidades del servidor. Si no se puede leer —porque el router aún no tiene ningún modelo cargado— la ventana queda **desconocida**, no cero, y no se muta el servidor.
- **`tools`** — Registro del catálogo y capa universal de ejecución. Envuelve **toda** ejecución con la misma secuencia: buscar, comprobar permiso, validar argumentos, ejecutar el handler, recortar la salida y emitir el evento. Define los tipos; las implementaciones se inyectan al cablear, así que no importa `fileops` ni `exec`. El catálogo es cerrado y vive embebido en el código. No inventa herramientas y no inventa permisos: comprueba los que el agente trae.
- **`fileops`** — Aplica operaciones de archivo y carpeta, valida la frontera de rutas y guarda el historial de cambios. Donde vive la frontera de la carpeta del proyecto.
- **`exec`** — Terminal. Lista blanca de comandos y bloqueo estructural de escritura con Landlock. Lo único que puede lanzar procesos.
- **`store`** — Único acceso a SQLite: esquema, WAL y transacciones. Ningún otro módulo escribe en la base.
- **`docs`** — Carga la documentación del proyecto, lee el frontmatter y construye el grafo de dependencias. No guarda nada.
- **`task`** — Lee y escribe el TODO de la ejecución: elementos, orden y estado.

## 4. Relaciones entre Componentes

Los módulos se comunican por canales de Go, no por red. Un solo proceso. La forma exacta de cada contrato entre módulos está en [[backend/02-interfaces/INTERFACES-GENERAL]].

- `tui` → `session`: la TUI pide a la sesión activa que mande lo escrito; pide cambiar de sesión, aprobar o declinar.
- `session` → `flow`: al arrancar un flujo, la sesión pide al orquestador que encadene etapas; en el chat, le entrega además el historial de la sesión (completo o compactado).
- `flow` → `context`: cada etapa pide su contexto por un objetivo concreto.
- `flow` → `agent`: el orquestador dice qué agente corre (`plan` o `build`) en cada etapa y le pasa el contexto del turno junto con el historial de la conversación.
- `context` → `llm` y `store`: para pedir al modelo qué documentos necesita, y para registrar la auditoría.
- `llm` → `openai`, `llamacpp`: la frontera usa el núcleo común (`openai` para el chat y el listado de modelos) y la extensión `llamacpp` (`/props`) para la ventana y las capacidades. El motor es único; no hay adaptador por tipo ni motor por sesión.
- `agent` → `llm`: el agente construye la llamada con su prompt, sus esquemas de herramientas y el historial, y la envía. Vuelve por streaming, token a token, y con las peticiones de herramienta acumuladas.
- `agent` → `tools`: el agente pide los esquemas de las herramientas del agente activo, y despacha cada ejecución. `tools` decide si se permite y qué pasa.
- `arranque` → `fileops`, `exec`: sus implementaciones se **inyectan** en el registro de herramientas como handlers. `tools` no los importa; es lo que permite que cada herramienta lleve su propio `Ejecutar` sin un ciclo de importación.
- `fileops` → `store`: cada cambio aplicado se registra en `change_history`.
- `queue` → `flow`: la cola toma la siguiente tarea y el orquestador la ejecuta.
- `docs` → `context` y `task`: proveen el grafo y las tareas.
- `store` es transversal: lo usan `session`, `context`, `fileops` y `queue`.

### Concurrencia

Una goroutine por sesión en ejecución, con el orquestador de etapas encima. Los canales de Go hacen de cola entre la TUI y el trabajo de fondo. SQLite en WAL para que la interfaz pueda leer mientras las sesiones de fondo escriben. Con un único motor (llama.cpp), las respuestas de las sesiones se serializan por orden de llegada: mientras una genera, la otra espera. Es una consecuencia del hardware, no del diseño. Ver [[specs/SPEC-MODELOS-LLAMACPP]].

**El turno de inferencia se toma por petición al modelo, no por ejecución completa.** Un turno puede necesitar varias peticiones —pedir, ejecutar herramientas, volver a pedir—, y entre una y otra puede haber una espera de aprobación. Si el testigo se tomara alrededor de la ejecución entera, una sesión bloqueada esperando al usuario retendría el modelo y las demás sesiones no podrían generar durante ese rato. Ver [[specs/SPEC-MODELOS-LLAMACPP]] y [[backend/04-infrastructure/INTEGRATIONS]].

## Stack

- Lenguaje: Go, binario único, sin runtime externo.
- Framework: ninguno para el backend. La TUI usa Bubble Tea + Lip Gloss.
- ORM: ninguno. `store` habla SQL directo con el driver puro Go.
- Base de datos: SQLite en modo WAL, un único archivo global en `~/.config/localcli/state.db` con las sesiones separadas por carpeta, vía `modernc.org/sqlite`.
- Otras dependencias: cliente HTTP estándar para el motor de inferencia (`llama-server`) y para internet; Landlock del sistema para el bloqueo de escritura en la terminal.
- Aislamiento: Landlock en Linux da garantía fuerte; en otros sistemas la garantía es más débil y queda documentada como tal. Ver [[backend/03-security/SECURITY]].

## Referencia rápida

- Módulos y entidades → [[backend/01-domain/DOMAIN]]
- Reglas del orquestador → [[backend/01-domain/BUSINESS_RULES]]
- Contratos entre módulos y superficies → [[backend/02-interfaces/INTERFACES-GENERAL]]
- Las diecisiete herramientas → [[backend/02-interfaces/TOOLS]]
- Seguridad y permisos → [[backend/03-security/SECURITY]]
- Configuración → [[backend/04-infrastructure/CONFIGURATION]]
- Integraciones externas → [[backend/04-infrastructure/INTEGRATIONS]]
- Eventos internos → [[backend/04-infrastructure/EVENTS]]
- Pruebas → [[backend/05-quality/TESTING]]
- Validación de entradas → [[backend/05-quality/VALIDATION]]
- Errores → [[backend/05-quality/ERRORS]]
- Decisiones de la capa → [[backend/DECISIONS]]

## Mapa de Navegación

Guía de lectura para la IA según la tarea:

- Implementar o cambiar una etapa → este documento, [[backend/01-domain/DOMAIN]], [[backend/02-interfaces/INTERFACES-GENERAL]].
- Añadir o revisar una herramienta → [[backend/02-interfaces/TOOLS]], [[backend/03-security/SECURITY]].
- Cambiar qué recibe el modelo → [[backend/01-domain/DOMAIN]], [[database/02-rules/DATA_FLOW]].
- Tocar la persistencia → [[database/DATABASE]], [[database/01-schema/SCHEMA]].
- Tocar la cola o el encadenamiento → [[backend/01-domain/BUSINESS_RULES]], [[backend/04-infrastructure/EVENTS]].
- Cambiar permisos, aprobación o aislamiento → [[backend/03-security/SECURITY]].

Al implementar una herramienta o una etapa, leer la subcadena necesaria:

```
BACKEND.md → DOMAIN → INTERFACES-GENERAL → TOOLS → BUSINESS_RULES
```

## Referencias

- [[PROJECT]] — decisiones técnicas ya aprobadas, no se reinventan.
- [[database/DATABASE]] — la capa de datos, no se duplica aquí.
