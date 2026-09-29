---
title: LocalCli — superficies del backend
tags: [backend, interfaces]
depende_de:
  - "[[backend/BACKEND]]"
  - "[[specs/SPEC-INTERFAZ]]"
  - "[[specs/SPEC-TOOLS]]"
  - "[[specs/SPEC-OLLAMA-PERFIL]]"
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
| Herramientas del agente | El modelo, vía `agent` | Catálogo de catorce llamadas incluidas, más las que declare el usuario | [[backend/02-interfaces/TOOLS]] |
| Eventos del motor | La TUI y el resto de módulos | Notificaciones unidireccionales por canales | [[backend/04-infrastructure/EVENTS]] |

Los datos que el modelo y la TUI ven en pantalla están definidos en el esquema, no aquí. Ver [[database/01-schema/TABLES]].

## 2. Base y Versión

No hay base path ni versión de API, porque no hay red. Lo equivalente es:

- **La versión del esquema** va en `PRAGMA user_version` de SQLite. Un archivo por proyecto, creado en su versión actual. Ver [[database/03-operations/MIGRATIONS]].
- **La versión del modelo** no la fija el harness. La elige el usuario, y el harness valida que quepa y avisa si no. Ver [[specs/SPEC-OLLAMA-PERFIL]].

Un cambio incompatible en el esquema o en el contrato de las herramientas se resuelve con migración, no con negotiation de versión. Un cambio en el catálogo de herramientas solo es compatible si `plan` sigue sin herramientas que escriban en el proyecto y `build` sigue teniendo todas; si no, rompe la garantía y no es un cambio menor. La acción `tareas` (la lista de pasos de la sesión) no escribe en el proyecto, así que no entra en esa garantía y la tienen los dos agentes. Añadir herramientas del usuario no altera esa condición: se reparten con las mismas reglas. Ver [[backend/03-security/SECURITY]].

## 3. Convenciones

Sin respuestas JSON ni códigos HTTP. Las convenciones son:

- **Toda la comunicación interna es por canales de Go.** Un solo proceso, sin red entre módulos. Un módulo no llama directamente a otro que no sea su dependencia declarada; la excepción es `tui`, que escribe donde el usuario manda.
- **Los datos hacia el modelo son texto plano** en un prompt, más lo que viaja por el canal de herramientas. Ese canal es la excepción: es estructurado, con un esquema por herramienta, y solo `agent` lo construye y `ollama` lo serializa. Las imágenes de un turno de chat viajan codificadas en base64 en el campo `images` de `/api/chat`, y solo `ollama` conoce ese formato. Ver [[backend/01-domain/DOMAIN]] y [[specs/SPEC-OLLAMA-PERFIL]].
- **Los datos hacia la TUI son eventos.** La TUI no pregunta: se le notifica. Ver [[backend/04-infrastructure/EVENTS]].
- **Los errores se propagan como valores**, no como excepciones, y suben por el canal que corresponde. Ver [[backend/05-quality/ERRORS]].
- **El streaming de tokens es la excepción al patrón por etapas:** no espera al final, va token a token desde `ollama` hasta la pantalla. Ver [[specs/SPEC-INTERFAZ]].

## 4. Estilo

No es REST, ni GraphQL, ni RPC. El estilo es **comando en terminal, herramienta estructurada y evento**:

- **Comandos:** el usuario escribe. La TUI los traduce a peticiones a la sesión. Los comandos que empiezan por `/` (`/planificar`, `/crear`, `/actualizar`, `/eliminar`, `/resolver`, `/ejecutar`) arrancan un flujo; el resto se responde como chat. Ver [[specs/SPEC-INTERFAZ]].
- **Herramientas:** el modelo pide una operación por su nombre, con argumentos ya formados contra un esquema declarado, y recibe un resultado estructurado. No hay formato en prosa que el modelo deba imitar. El catálogo incluido son catorce y no se amplía de otra forma; el usuario puede añadir las suyas declarándolas, y respetan el mismo reparto por agente. Ver [[backend/02-interfaces/TOOLS]] y [[specs/SPEC-TOOLS]].
- **Eventos:** el motor notifica cambios de estado, peticiones de aprobación, la invocación de una herramienta y token de streaming. Unidireccionales; el consumidor decide qué hacer. Ver [[backend/04-infrastructure/EVENTS]].

## 5. Contratos entre módulos

Lo que cada módulo expone a los que dependen de él. El detalle de payloads está en [[backend/04-infrastructure/EVENTS]] y en [[backend/02-interfaces/dto/TOOLS-DTO]].

| Módulo | Expone | A quién |
|---|---|---|
| `session` | Crear, cambiar, retomar, cerrar sesión; mandar mensaje (con las imágenes del turno); decidir si es chat o comando de flujo; aprobar o declinar | `tui` |
| `flow` | Lanzar un flujo oficial pedido por el usuario; detectar trabajo ordenado y proponer el TODO; pausar, cancelar, reanudar | `session`, `queue` |
| `context` | Pedir contexto para una etapa y un objetivo; devolver documentos seleccionados y auditados (la etapa viaja con la petición para que `context_audit` distinga una de otra) | `flow` |
| `agent` | Cargar y validar los agentes de `.localcli/agents/*.json` (los base y los propios); construir la llamada de uno; llevar el bucle de pedir al modelo, ejecutar herramientas y volver a pedir; montar los esquemas de las herramientas del agente | `flow`, `arranque` |
| `ollama` | Enviar una petición en streaming con el canal de herramientas (las imágenes del chat viajan en base64, solo aquí se conoce ese formato); devolver tokens, razonamiento y peticiones de herramienta; serializar y limitar la inferencia | `agent`, `context` |
| `tools` | Registrar el catálogo; exponer el esquema de las herramientas de un agente; **envolver toda ejecución** con permiso, validación, recorte y eventos; cargar las herramientas del usuario | `agent` |
| `fileops` | Aplicar una operación de archivo o carpeta; registrar el cambio. **Inyectado** como handler, no importado | `arranque` |
| `exec` | Ejecutar un comando de la lista blanca o aprobado, con Landlock. **Inyectado** como handler, no importado | `arranque` |
| `store` | Abrir, migrar y escribir en SQLite | todos los que persisten |
| `docs` | Cargar documentación; exponer el grafo de frontmatter | `context` |
| `task` | Leer y escribir archivos de tarea | `queue` |

Dos cosas de esta tabla que cambiaron y conviene no pasar por alto:

**`fileops` y `exec` ya no dependen de `tools`, ni al revés.** `tools` define los tipos; las implementaciones se inyectan al cablear. Antes el enrutado era un `switch` de tres categorías y los handlers vivían en los módulos de abajo; ahora cada herramienta lleva su propio `Ejecutar`, y la dependencia va en la dirección del cableado. Sin esto, un handler por herramienta habría producido un ciclo de importación.

**`tools` expone los esquemas, no las ejecuta hacia el modelo.** Es `agent` quien construye la petición y `ollama` quien la serializa. `tools` no conoce el formato de `/api/chat`, igual que antes no conocía el prompt. Ver [[backend/02-interfaces/TOOLS]].

## Referencias

- [[backend/BACKEND]] — mapa de la capa.
- [[backend/02-interfaces/TOOLS]] — el catálogo, la capa universal y las herramientas del usuario.
- [[backend/02-interfaces/dto/TOOLS-DTO]] — los payloads estructurados y los esquemas derivados.
- [[backend/04-infrastructure/EVENTS]] — los eventos del motor, incluidos los de herramienta.
- [[specs/SPEC-TOOLS]] — la especificación funcional del catálogo.
- [[specs/SPEC-OLLAMA-PERFIL]] — el canal de herramientas y la capacidad del modelo.
