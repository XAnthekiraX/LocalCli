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
| Herramientas del agente | El modelo, vía `agent` | Catálogo cerrado de trece llamadas | [[backend/02-interfaces/TOOLS]] |
| Eventos del motor | La TUI y el resto de módulos | Notificaciones unidireccionales por canales | [[backend/04-infrastructure/EVENTS]] |

Los datos que el modelo y la TUI ven en pantalla están definidos en el esquema, no aquí. Ver [[database/01-schema/TABLES]].

## 2. Base y Versión

No hay base path ni versión de API, porque no hay red. Lo equivalente es:

- **La versión del esquema** va en `PRAGMA user_version` de SQLite. Un archivo por proyecto, creado en su versión actual. Ver [[database/03-operations/MIGRATIONS]].
- **La versión del modelo** no la fija el harness. La elige el usuario, y el harness valida que quepa y avisa si no. Ver [[specs/SPEC-OLLAMA-PERFIL]].

Un cambio incompatible en el esquema o en el contrato de las herramientas se resuelve con migración, no con negotiation de versión. Un cambio en el catálogo de herramientas solo es compatible si `plan` sigue sin herramientas de escritura y `build` sigue teniendo todas; si no, rompe la garantía y no es un cambio menor. Ver [[backend/03-security/SECURITY]].

## 3. Convenciones

Sin respuestas JSON ni códigos HTTP. Las convenciones son:

- **Toda la comunicación interna es por canales de Go.** Un solo proceso, sin red entre módulos. Un módulo no llama directamente a otro que no sea su dependencia declarada; la excepción es `tui`, que escribe donde el usuario manda.
- **Los datos hacia el modelo son texto plano** en un prompt. No hay serialización especial: el modelo recibe contexto y responde texto. La excepción son las imágenes de un turno de chat, que viajan codificadas en base64 en el campo `images` de `/api/chat` (y solo `ollama` conoce ese formato). Ver [[backend/01-domain/DOMAIN]].
- **Los datos hacia la TUI son eventos.** La TUI no pregunta: se le notifica. Ver [[backend/04-infrastructure/EVENTS]].
- **Los errores se propagan como valores**, no como excepciones, y suben por el canal que corresponde. Ver [[backend/05-quality/ERRORS]].
- **El streaming de tokens es la excepción al patrón por etapas:** no espera al final, va token a token desde `ollama` hasta la pantalla. Ver [[specs/SPEC-INTERFAZ]].

## 4. Estilo

No es REST, ni GraphQL, ni RPC. El estilo es **comando en terminal, herramienta estructurada y evento**:

- **Comandos:** el usuario escribe. La TUI los traduce a peticiones a la sesión. Los comandos que empiezan por `/` (`/planificar`, `/crear`, `/actualizar`, `/eliminar`, `/resolver`, `/ejecutar`) arrancan un flujo; el resto se responde como chat. Ver [[specs/SPEC-INTERFAZ]].
- **Herramientas:** el modelo pide una operación con argumentos estructurados (ruta, contenido, comando), y recibe un resultado estructurado. El catálogo es cerrado y cada herramienta pertenece a un agente. Ver [[backend/02-interfaces/TOOLS]].
- **Eventos:** el motor notifica cambios de estado, peticiones de aprobación y token de streaming. Unidireccionales; el consumidor decide qué hacer. Ver [[backend/04-infrastructure/EVENTS]].

## 5. Contratos entre módulos

Lo que cada módulo expone a los que dependen de él. El detalle de payloads está en [[backend/04-infrastructure/EVENTS]] y en [[backend/02-interfaces/dto/TOOLS-DTO]].

| Módulo | Expone | A quién |
|---|---|---|
| `session` | Crear, cambiar, retomar, cerrar sesión; mandar mensaje (con las imágenes del turno); decidir si es chat o comando de flujo; aprobar o declinar | `tui` |
| `flow` | Lanzar un flujo oficial pedido por el usuario; detectar trabajo ordenado y proponer el TODO; pausar, cancelar, reanudar | `session`, `queue` |
| `context` | Pedir contexto para una etapa y un objetivo; devolver documentos seleccionados y auditados (la etapa viaja con la petición para que `context_audit` distinga una de otra) | `flow` |
| `agent` | Cargar y validar los agentes de `ai/agents/*.json` (los base y los propios); construir la llamada de uno y despachar sus herramientas | `flow`, `arranque` |
| `ollama` | Enviar una petición en streaming (las imágenes del chat viajan en base64, solo aquí se conoce ese formato); devolver tokens y razonamiento | `agent`, `context` |
| `tools` | Registrar y enrutar una herramienta; comprobar permiso | `agent` |
| `fileops` | Aplicar una operación de archivo o carpeta; registrar el cambio | `tools` |
| `exec` | Ejecutar un comando de la lista blanca o aprobado | `tools` |
| `store` | Abrir, migrar y escribir en SQLite | todos los que persisten |
| `docs` | Cargar documentación; exponer el grafo de frontmatter | `context` |
| `task` | Leer y escribir archivos de tarea | `queue` |

## Referencias

- [[backend/BACKEND]] — mapa de la capa.
- [[backend/02-interfaces/TOOLS]] — el catálogo de trece herramientas.
- [[backend/02-interfaces/dto/TOOLS-DTO]] — los payloads estructurados.
- [[backend/04-infrastructure/EVENTS]] — los eventos del motor.
- [[specs/SPEC-TOOLS]] — la especificación funcional del catálogo.
