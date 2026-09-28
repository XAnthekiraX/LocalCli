---
title: LocalCli — servicios externos
tags: [backend, infraestructura]
depende_de:
  - "[[backend/DECISIONS]]"
  - "[[backend/03-security/SECURITY]]"
  - "[[backend/05-quality/VALIDATION]]"
  - "[[specs/SPEC-OLLAMA-PERFIL]]"
  - "[[specs/SPEC-TOOLS]]"
relacionado:
  - "[[backend/01-domain/DOMAIN]]"
  - "[[backend/04-infrastructure/CONFIGURATION]]"
  - "[[backend/04-infrastructure/EVENTS]]"
  - "[[specs/SPEC-INTERFAZ]]"
---
# INTEGRATIONS — Servicios externos

Dos integraciones: Ollama, en local, y la búsqueda por internet. Ninguna otra sale de la máquina. Ver [[backend/03-security/SECURITY]] para las fronteras de seguridad.

## 1. APIs y servicios externos

### Ollama

- **Propósito:** es el modelo. Genera las respuestas, el razonamiento, las decisiones de contexto y las peticiones de herramientas. Sin Ollama no hay respuestas; el harness es un cliente de él.
- **Dónde:** API HTTP en local, por defecto `http://localhost:11434`.
- **Streaming:** obligatorio. Peticiones en streaming para poder mostrar el razonamiento mientras llega, que es el requisito de la interfaz. Ver [[specs/SPEC-INTERFAZ]].
- **Perfil de hardware:** el harness detecta la máquina, avisa si el modelo elegido no cabe en la VRAM y limita el contexto. El modelo lo elige el usuario, no el harness. Ver [[specs/SPEC-OLLAMA-PERFIL]].
- **Canal de herramientas:** es la parte de la que LocalCli depende por completo. Cada petición a `/api/chat` lleva las definiciones de las herramientas del agente activo —nombre, descripción y esquema de argumentos— y el modelo responde pidiendo una por su nombre con los argumentos ya formados. Un turno puede necesitar varias peticiones: pedir, ejecutar, volver a pedir. No hay formato en prosa de reserva, así que **un modelo que no declare esta capacidad no puede trabajar con herramientas**, y el agente cae a modo conversación. Ver [[specs/SPEC-TOOLS]].
- **Capacidades:** el harness consulta `/api/show` para leer qué declara saber cada modelo (`completion`, `tools`, `vision`…). La diferencia entre las dos es que **`tools` decide y `vision` solo informa**: sin `vision`, el harness adjunta la imagen y avisa de que el modelo no la interpretará; sin `tools`, el agente no recibe ninguna y conversa. Ninguna de las dos bloquea la elección del modelo. Ver [[backend/02-interfaces/TOOLS]].
- **Imágenes:** para los modelos multimodales, cada mensaje de `/api/chat` acepta `images`, una lista de imágenes codificadas en base64 (sin prefijo `data:`). El harness adjunta al turno las rutas de imagen que el usuario escribe en el mensaje. Las imágenes son efímeras: acompañan a ese turno y no se guardan en el historial.
- **Concurrencia:** como las sesiones comparten un único modelo cargado, sus respuestas se serializan por orden de llegada. El turno de inferencia se toma **por petición**, no por ejecución completa, para que una sesión esperando una aprobación de herramienta no retenga el modelo. Es una consecuencia del hardware, no un defecto del diseño. Ver [[backend/DECISIONS]].

### Búsqueda en internet

- **Propósito:** consultar documentación de librerías mientras se planifica, y verificar versiones y APIs. Es la única integración que hace salir información de la máquina.
- **Qué sale:** solo la consulta que redacta el modelo. Nunca contenido del proyecto, documentación, historial ni tareas. Si la búsqueda necesita el proyecto, el agente te lo pide a ti.
- **Qué vuelve:** `buscar_en_internet` devuelve resultados (título, dirección, fragmento); `abrir_pagina` devuelve el contenido de una página. Ese contenido entra al presupuesto de contexto, se recorta y se audita igual que el resto.
- **Cómo se habilita:** no viene activada; el usuario la habilita. Ver [[backend/04-infrastructure/CONFIGURATION]].

## 2. Webhooks

No aplican. No hay servidor, ni entradas HTTP, ni notificaciones entrantes. LocalCli no expone nada que un tercero pueda llamar.

## 3. SDKs y librerías

| Dependencia | Para qué | Nota |
|---|---|---|
| Cliente HTTP de la biblioteca estándar | Hablar con Ollama e internet | Sin SDK de terceros |
| Landlock (syscall del kernel) | Bloqueo estructural de escritura en la terminal | Linux 5.13+; en otros sistemas no está |
| `modernc.org/sqlite` | Acceso a SQLite sin cgo | Solo lo usa `store` |
| Bubble Tea + Lip Gloss | La TUI | Los usa `tui`, no el motor |

## 4. Credenciales y configuración requerida

Ninguna. Ollama corre en local sin credenciales, y la búsqueda por internet no pide clave de API. LocalCli no guarda ni pide secretos. Si algún día una integración los pidiera, se documentaría aquí.

Lo que sí se necesita en la máquina, pero no es una credencial: Ollama instalado y corriendo, y Landlock disponible si se quiere el aislamiento fuerte. Ver [[backend/04-infrastructure/CONFIGURATION]].

## 5. Contratos externos

### Contrato con Ollama

- El harness espera respuestas en streaming, con el razonamiento distinguible del texto final. De ahí sale el razonamiento en vivo de la interfaz.
- El harness detecta el hardware y valida el modelo antes de cargarlo. Si un modelo no cabe en la VRAM, avisa y lo carga en RAM, más lento, sin fallar en silencio.
- El harness lee la ficha del modelo (`/api/show`) para conocer sus capacidades. La ausencia de `tools` cambia el comportamiento —el agente conversa sin herramientas— y la ausencia de `vision` solo produce un aviso. Ninguna impide usar el modelo.
- **Las herramientas viajan por el canal estructurado de `/api/chat`, no en el prompt.** La petición lleva un objeto por herramienta y la respuesta puede traer `tool_calls` con nombre y argumentos. Los resultados vuelven como mensajes de rol `tool`. Si ese contrato cambiara, el único módulo afectado sería `ollama`: `agent` y `tools` no deberían enterarse. Ver [[backend/02-interfaces/TOOLS]].
- **Un turno puede abrir varias peticiones.** El harness no da por terminado un turno cuando el modelo deja de escribir: sigue mientras el modelo pida herramientas, hasta un máximo. Quien decide cuándo se acaba es el bucle de `agent`, no el stream. Ver [[specs/SPEC-AGENTE-BASE]].
- Los turnos de herramienta **no se persisten**. El historial guardado guarda el mensaje del usuario y la respuesta final del agente; lo intermedio existe solo en memoria mientras dura el turno. Ver [[database/02-rules/DATA_FLOW]].
- Las imágenes de un turno de chat viajan en `images` (base64) del mensaje de `/api/chat`; si el contrato de Ollama para imágenes cambiara, el módulo afectado es `ollama` y el resto no debería enterarse.
- **La ventana de contexto se declara por petición.** Cada `/api/chat` lleva `options.num_ctx`, la menor entre la ventana que reporta el modelo en `/api/tags` (`details.context_length`) y el tope (`LOCALCLI_CONTEXT_LIMIT`, 16384 por defecto). Sin declararla, Ollama usa su valor de servidor (≈4096), recorta la lista de mensajes y devuelve 500 `no user query found in messages` en cuanto el turno encadena herramientas. El nodo de contexto y el presupuesto del historial se derivan de esa misma ventana. Ver [[backend/DECISIONS]].
- El límite de contexto del modelo manda: el nodo de contexto recorta para que lo entregado quepa. Ver [[backend/01-domain/DOMAIN]].
- Un status HTTP distinto de 200 trae el cuerpo de Ollama (`{"error": …}`) en el mensaje del error tipado: el motivo real, no solo el código.

Si el contrato de streaming de Ollama cambiara, el módulo afectado es `ollama`, y el resto no debería enterarse.

### Contrato con la búsqueda

- La búsqueda devuelve solo lo público de internet.
- El contenido que vuelve es texto sin trust: entra al contexto como cualquier otro documento, sujeto al mismo recorte y auditoría. Ningún agente lo vuelca sin procesarlo.

## Referencias

- [[specs/SPEC-OLLAMA-PERFIL]] — modelo, hardware, límites y concurrencia.
- [[specs/SPEC-TOOLS]] — el contrato funcional de las herramientas.
- [[backend/02-interfaces/TOOLS]] — la capa universal y las herramientas del usuario.
- [[backend/03-security/SECURITY]] — qué sale y qué no.
- [[backend/04-infrastructure/CONFIGURATION]] — cómo se habilita internet y dónde viven las herramientas del usuario.
- [[backend/04-infrastructure/EVENTS]] — cómo se notifica el streaming y las herramientas.
