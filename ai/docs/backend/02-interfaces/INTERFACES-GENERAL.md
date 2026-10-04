---
title: LocalCli — superficies del backend
tags: [backend, interfaces]
depende_de:
  - "[[backend/BACKEND]]"
  - "[[specs/SPEC-INTERFAZ]]"
  - "[[specs/SPEC-TOOLS]]"
  - "[[specs/SPEC-MODELO-MOTOR]]"
relacionado:
  - "[[specs/SPEC-INTERFAZ-ATAJOS]]"
  - "[[backend/02-interfaces/TOOLS]]"
  - "[[backend/02-interfaces/dto/TOOLS-DTO]]"
  - "[[backend/01-domain/DOMAIN]]"
  - "[[backend/03-security/SECURITY]]"
  - "[[backend/04-infrastructure/EVENTS]]"
  - "[[backend/05-quality/ERRORS]]"
  - "[[database/01-schema/TABLES]]"
  - "[[database/03-operations/MIGRATIONS]]"
---
# INTERFACES-GENERAL — Superficies del backend

`LocalCli` no tiene API HTTP. No expone puertos, no recibe peticiones de red y no tiene clientes externos. Sus "interfaces" son otra cosa: la superficie que consume la TUI, la superficie que consume el modelo (las herramientas) y los contratos entre módulos.

Este documento las agrupa. Para el detalle de las herramientas, [[backend/02-interfaces/TOOLS]].

## 1. Recursos

No hay recursos REST. Hay tres superficies, cada una con su consumidor:

| Superficie | Consumidor | Naturaleza | Dónde se define |
|---|---|---|---|
| Comandos de la CLI | El usuario, en la terminal | Comandos escritos, no llamadas | [[specs/SPEC-INTERFAZ-ATAJOS]] |
| Herramientas del agente | El modelo, vía `agent` | Catálogo de quince llamadas incluidas, más las que declare el usuario | [[backend/02-interfaces/TOOLS]] |
| Eventos del orquestador | La TUI y el resto de módulos | Notificaciones unidireccionales por canales | [[backend/04-infrastructure/EVENTS]] |

Los datos que el modelo y la TUI ven en pantalla están definidos en el esquema, no aquí. Ver [[database/01-schema/TABLES]].

## 2. Base y Versión

No hay base path ni versión de API, porque no hay red. Lo equivalente es:

- **La versión del esquema** va en `PRAGMA user_version` de SQLite. Un archivo por proyecto, creado en su versión actual. Ver [[database/03-operations/MIGRATIONS]].
- **La versión del modelo** no la fija el harness. La elige el usuario, y el harness valida que quepa y avisa si no. Ver [[specs/SPEC-MODELO-MOTOR]].

Un cambio incompatible en el esquema o en el contrato de las herramientas se resuelve con migración, no con negotiation de versión. Un cambio en el catálogo de herramientas solo es compatible si `plan` sigue sin herramientas que escriban en el proyecto y `build` sigue teniendo todas; si no, rompe la garantía y no es un cambio menor. La lista de pasos de la sesión cae en `read` —es estado de la sesión, no del proyecto—, así que no entra en esa garantía y la tienen los dos agentes. Añadir herramientas del usuario no altera esa condición: se reparten con las mismas reglas. Ver [[backend/03-security/SECURITY]].

## 3. Convenciones

Sin respuestas JSON ni códigos HTTP. Las convenciones son:

- **Toda la comunicación interna es por canales de Go.** Un solo proceso, sin red entre módulos. Un módulo no llama directamente a otro que no sea su dependencia declarada; la excepción es `tui`, que escribe donde el usuario manda.
- **Los datos hacia el modelo son texto plano** en un prompt, más lo que viaja por el canal de herramientas. Ese canal es la excepción: es estructurado, con un esquema por herramienta, y solo `agent` lo construye y el adaptador del motor lo serializa. Las imágenes de un turno de chat viajan en el formato que declara su runtime —`images` en base64 con Ollama, `image_url` con data URI con llama.cpp—, y solo el adaptador conoce ese formato. Ver [[backend/01-domain/DOMAIN]] y [[specs/SPEC-MODELO-MOTOR]].
- **Los datos hacia la TUI son eventos.** La TUI no pregunta: se le notifica. Ver [[backend/04-infrastructure/EVENTS]].
- **Los errores se propagan como valores**, no como excepciones, y suben por el canal que corresponde. Ver [[backend/05-quality/ERRORS]].
- **El streaming de tokens es la excepción al patrón por etapas:** no espera al final, va token a token desde `ollama` hasta la pantalla. Ver [[specs/SPEC-INTERFAZ]].

## 4. Estilo

No es REST, ni GraphQL, ni RPC. El estilo es **comando en terminal, herramienta estructurada y evento**:

- **Comandos:** el usuario escribe. La TUI los traduce a peticiones a la sesión. Los comandos que empiezan por `/` (`/planificar`, `/crear`, `/actualizar`, `/eliminar`, `/resolver`, `/ejecutar`) arrancan un flujo; el resto se responde como chat. Ver [[specs/SPEC-INTERFAZ]].
- **Herramientas:** el modelo pide una operación por su nombre, con argumentos ya formados contra un esquema declarado, y recibe un resultado estructurado. No hay formato en prosa que el modelo deba imitar. El catálogo incluido son quince y no se amplía de otra forma; el usuario puede añadir las suyas declarándolas, y respetan el mismo reparto por agente. Ver [[backend/02-interfaces/TOOLS]] y [[specs/SPEC-TOOLS]].
- **Eventos:** el orquestador notifica cambios de estado, peticiones de aprobación, la invocación de una herramienta y token de streaming. Unidireccionales; el consumidor decide qué hacer. Ver [[backend/04-infrastructure/EVENTS]].

## 5. Contratos entre módulos

Lo que cada módulo expone a los que dependen de él. El detalle de payloads está en [[backend/04-infrastructure/EVENTS]] y en [[backend/02-interfaces/dto/TOOLS-DTO]].

| Módulo | Expone | A quién |
|---|---|---|
| `session` | Crear, cambiar, retomar, cerrar sesión; mandar mensaje (con las imágenes del turno); decidir si es chat o comando de flujo; aprobar o declinar | `tui` |
| `flow` | Lanzar un flujo oficial pedido por el usuario; detectar trabajo ordenado y proponer el TODO; pausar, cancelar, reanudar | `session`, `queue` |
| `context` | Pedir contexto para una etapa y un objetivo; devolver documentos seleccionados y auditados (la etapa viaja con la petición para que `context_audit` distinga una de otra) | `flow` |
| `agent` | Cargar y validar los agentes de `.localcli/agents/<carpeta>/` (los base y los propios), leyendo de cada carpeta su `agent.yaml` y su `prompt.md`; construir la llamada de uno (con el `think` que decida el usuario, solo si el modelo razona); llevar el bucle de pedir al modelo, ejecutar herramientas y volver a pedir; montar los esquemas de las herramientas del agente | `flow`, `arranque` |
| `llm` | La frontera neutra con el modelo: tipos (`Mensaje`, `Herramienta`, `Evento`, `Modelo`), interfaz `Motor` (`Nombre`, `BaseURL`, `Chat`, `ListarModelos`, `Capacidades`, `VentanaDeContexto`), el tipo `Capacidad` de tres estados, el registro de motores declarados con sus extensiones nativas y su persistencia global (`~/.config/localcli/motores.json`) y una cola de inferencia por motor; no conoce ningún endpoint | `agent`, `context`, `tui` |
| `llm/openai` | El núcleo común: transporte HTTP y lectura SSE contra `/v1/chat/completions` y `/v1/models`. **Lo comparten todos los motores**, así que no sabe de runtime: no tiene `ollama` ni llama.cpp dentro | `llm` y los adaptadores |
| `ollama`, `llamacpp` | Los dos adaptadores de motor. Ninguno implementa la ruta de la conversación: esa la da `llm/openai`. Cada uno aporta **solo sus extensiones nativas** —`ollama`: `options.num_ctx`, `/api/show` y `/api/tags`; `llamacpp`: `/props`— y devuelve tokens, razonamiento y peticiones de herramienta. Solo el registro de `llm` los instancia, por tipo y por URL | `llm` |
| `tools` | Registrar el catálogo; exponer el esquema de las herramientas de un agente; **envolver toda ejecución** con permiso, validación, recorte y eventos; cargar las herramientas del usuario | `agent` |
| `fileops` | Aplicar una operación de archivo o carpeta; registrar el cambio. **Inyectado** como handler, no importado | `arranque` |
| `exec` | Ejecutar un comando de la lista blanca o aprobado, con Landlock. **Inyectado** como handler, no importado | `arranque` |
| `store` | Abrir, migrar y escribir en SQLite | todos los que persisten |
| `docs` | Cargar documentación; exponer el grafo de frontmatter | `context` |
| `task` | Leer y escribir archivos de tarea | `queue` |

Dos cosas de esta tabla que cambiaron y conviene no pasar por alto:

**`fileops` y `exec` ya no dependen de `tools`, ni al revés.** `tools` define los tipos; las implementaciones se inyectan al cablear. Antes el enrutado era un `switch` de tres categorías y los handlers vivían en los módulos de abajo; ahora cada herramienta lleva su propio `Ejecutar`, y la dependencia va en la dirección del cableado. Sin esto, un handler por herramienta habría producido un ciclo de importación.

**`tools` expone los esquemas, no las ejecuta hacia el modelo.** Es `agent` quien construye la petición y el adaptador del motor quien la serializa. `tools` no conoce el formato de ningún runtime, igual que antes no conocía el prompt. Ver [[backend/02-interfaces/TOOLS]].

**El motor es una propiedad de la sesión, no del arranque.** `llm` mantiene el registro de motores declarados y puede tener varios vivos a la vez; `session` es quien dice con qué motor y con qué modelo trabaja cada sesión, y quien pide al registro el adaptador correspondiente en cada turno. Ningún módulo elige motor por su cuenta, y ningún módulo comprueba el tipo: eso es del registro. Ver [[specs/SPEC-MODELO-MOTOR]] y [[specs/SPEC-SESIONES]].

**El tipo de motor ya no decide el transporte.** Antes el tipo cerrado elegía el adaptador y cada uno traducía un protocolo distinto. Ahora todos hablan el núcleo OpenAI-compatible, así que `tipo` solo dice **qué extensiones nativas se pueden declarar** y el transporte no se ramifica por él. Un runtime nuevo que hable el núcleo no necesita código nuevo; uno que declare más extensiones da más información sin que el otro tenga que interpretarla. Ver [[backend/DECISIONS]].

**Las capacidades viajan en tres estados, no en dos.** `Motor.Capacidades` devuelve un tipo `Capacidad` con `soportada`, `no soportada` y `desconocida`, y `false` no significa nunca `desconocida`. El estado `desconocida` existe porque una extensión ausente no es una capacidad ausente, y tratarla como si lo fuera produce dos fallos opuestos según el sentido. Quien consume decide qué hacer con cada estado, y la regla vigente es: herramientas y visión no se ofrecen sin dato, el razonamiento se apaga y avisa. Ver [[specs/SPEC-MODELO-MOTOR]].

## Referencias

- [[backend/BACKEND]] — mapa de la capa.
- [[backend/02-interfaces/TOOLS]] — el catálogo, la capa universal y las herramientas del usuario.
- [[backend/02-interfaces/dto/TOOLS-DTO]] — los payloads estructurados y los esquemas derivados.
- [[backend/04-infrastructure/EVENTS]] — los eventos del motor, incluidos los de herramienta.
- [[specs/SPEC-TOOLS]] — la especificación funcional del catálogo.
- [[specs/SPEC-MODELO-MOTOR]] — el canal de herramientas y la capacidad del modelo.
