---
title: LocalCli — servicios externos
tags: [backend, infraestructura]
depende_de:
  - "[[backend/DECISIONS]]"
  - "[[backend/03-security/SECURITY]]"
  - "[[backend/05-quality/VALIDATION]]"
  - "[[specs/SPEC-MODELO-PROVEEDOR]]"
  - "[[specs/SPEC-TOOLS]]"
relacionado:
  - "[[backend/01-domain/DOMAIN]]"
  - "[[backend/04-infrastructure/CONFIGURATION]]"
  - "[[backend/04-infrastructure/EVENTS]]"
  - "[[specs/SPEC-INTERFAZ]]"
---
# INTEGRATIONS — Servicios externos

Tres integraciones, de las que solo una sale de la máquina: los proveedores de modelo —Ollama y llama.cpp, los dos en local— y la búsqueda por internet. Ninguna otra sale. Ver [[backend/03-security/SECURITY]] para las fronteras de seguridad.

## 1. APIs y servicios externos

### Proveedores de modelo

- **Propósito:** es el modelo. Genera las respuestas, el razonamiento, las decisiones de contexto y las peticiones de herramientas. Sin proveedor no hay respuestas; el harness es un cliente de él.
- **Dónde:** API HTTP en local. La dirección la decide el proveedor elegido al arrancar: `LOCALCLI_PROVEEDOR` y, para llama.cpp, `LOCALCLI_LLAMACPP_URL`. Ver [[backend/04-infrastructure/CONFIGURATION]].
- **Streaming:** obligatorio. Peticiones en streaming para poder mostrar el razonamiento mientras llega, que es el requisito de la interfaz. Ver [[specs/SPEC-INTERFAZ]].
- **Perfil de hardware:** el harness detecta la máquina, avisa si el modelo elegido no cabe en la VRAM y limita el contexto. El modelo lo elige el usuario, no el harness. Ver [[specs/SPEC-MODELO-PROVEEDOR]].
- **Canal de herramientas:** es la parte de la que LocalCli depende por completo. Cada petición al modelo lleva las definiciones de las herramientas del agente activo —nombre, descripción y esquema de argumentos— y el modelo responde pidiendo una por su nombre con los argumentos ya formados. Un turno puede necesitar varias peticiones: pedir, ejecutar, volver a pedir. No hay formato en prosa de reserva, así que **un modelo que no declare esta capacidad no puede trabajar con herramientas**, y el agente cae a modo conversación. Ver [[specs/SPEC-TOOLS]].
- **Capacidades:** el harness pregunta al proveedor qué declara saber cada modelo y **normaliza la respuesta a tres** —herramientas, visión y razonamiento—, de manera que ninguna lógica del harness sabe de dónde salieron. La diferencia entre las tres es que **las herramientas deciden y las otras dos solo informan**: sin visión, el harness adjunta la imagen y avisa de que el modelo no la interpretará; sin razonamiento, no se le pide razonar; sin herramientas, el agente no recibe ninguna y conversa. Ninguna de las tres bloquea la elección del modelo. Ver [[backend/02-interfaces/TOOLS]].
- **Imágenes:** para los modelos multimodales, el harness adjunta al turno las rutas de imagen que el usuario escribe en el mensaje. El formato en el que viajan depende del proveedor. Las imágenes son efímeras: acompañan a ese turno y no se guardan en el historial.
- **Concurrencia:** el harness serializa las respuestas **por orden de llegada**, con una cola propia, porque en el hardware objetivo solo cabe un modelo cargado a la vez. El turno de inferencia se toma **por petición**, no por ejecución completa, para que una sesión esperando una aprobación de herramienta no retenga el modelo. Es una consecuencia del hardware, no un defecto del diseño. Ver [[backend/DECISIONS]].

Los dos proveedores sirven los mismos pesos: lo que cambia es el runtime, no el modelo. Ver [[specs/SPEC-MODELO-PROVEEDOR]].

#### Ollama

- **Streaming:** NDJSON, una línea JSON por evento. `message.thinking` es el razonamiento y `message.content` el texto final.
- **Herramientas:** la petición lleva un objeto por herramienta y la respuesta puede traer `tool_calls` con nombre y argumentos. Los resultados vuelven como mensajes de rol `tool`.
- **Imágenes:** cada mensaje acepta `images`, una lista de imágenes codificadas en base64 (sin prefijo `data:`).
- **Razonamiento:** el campo `think` viaja por petición. Apagado por defecto en cada petición y encendido por el interruptor del pie.
- **Capacidades:** `/api/show` las expone en `capabilities` (`completion`, `tools`, `vision`, `thinking`).
- **Lista de modelos:** `/api/tags`, que además reporta en `details.context_length` la ventana de cada modelo.

#### llama.cpp

- **Dónde:** `llama-server` en modo **router**, con `--models-dir`. Un servidor por defecto no es admisible: el harness llama por nombre de modelo y el router es lo que atiende esa llamada. La autocarga se deja activada para no tener que pedirle que cargue nada.
- **Streaming:** SSE, con `data: {…}` y un `data: [DONE]` que cierra. El razonamiento llega en `delta.reasoning_content` y el texto en `delta.content`. Las llamadas de herramienta llegan en `delta.tool_calls` **por índice**: una misma llamada puede repartirse en varias deltas, así que hay que acumularlas por su índice antes de poder ejecutarlas.
- **Herramientas:** la petición usa `tools` con el esquema OpenAI; los resultados vuelven con rol `tool` y su `tool_call_id`.
- **Imágenes:** `image_url` con data URI.
- **Razonamiento:** se pide con `reasoning_effort` cuando la plantilla del modelo lo soporta.
- **Capacidades:** `/props` las expone en `chat_template_caps` (`supports_tool_calls`, `supports_reasoning_effort`) y en `modalities.vision`.
- **Lista de modelos:** `/models` del router, con `/v1/models` como alternativa.
- **Ventana:** se **lee**, no se declara. La fijó quien arrancó el servidor con `-c` y no hay forma de cambiarla por petición. Ver más abajo.
- **Estado del servidor:** es de la persona que lo levanta. El harness **no** le pide descargar un modelo (`POST /models`), ni cargarlo o descargarlo (`POST /models/load`), ni ajustar su ventana (`POST /props`).
- **Banderas:** tiene que levantarse **sin** las opciones que sirven lectura o escritura de archivos (`--tools`, `--agent`, `--mcp-servers-json`). Un proveedor con acceso al sistema de archivos no es admisible. Ver [[backend/03-security/SECURITY]].

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
| Cliente HTTP de la biblioteca estándar | Hablar con los proveedores de modelo e internet | Sin SDK de terceros; el SSE se lee con `bufio` de la estándar |
| Landlock (syscall del kernel) | Bloqueo estructural de escritura en la terminal | Linux 5.13+; en otros sistemas no está |
| `modernc.org/sqlite` | Acceso a SQLite sin cgo | Solo lo usa `store` |
| Bubble Tea + Lip Gloss | La TUI | Los usa `tui`, no el motor |

## 4. Credenciales y configuración requerida

Ninguna. Los dos proveedores corren en local sin credenciales, y la búsqueda por internet no pide clave de API. LocalCli no guarda ni pide secretos. Si algún día una integración los pidiera, se documentaría aquí.

Que `llama-server` acepte `--api-key` no cambia esto: el harness no lo usa, porque el servidor es de loopback y al que se le pone clave es al que hay que enseñarle la clave.

Lo que sí se necesita en la máquina, pero no es una credencial: un proveedor de modelo corriendo —Ollama o `llama-server`— y Landlock disponible si se quiere el aislamiento fuerte. Ver [[backend/04-infrastructure/CONFIGURATION]].

## 5. Contratos externos

### Contrato con el proveedor de modelo

Lo que el harness da por cierto con cualquier proveedor:

- El harness espera respuestas en streaming, con el razonamiento distinguible del texto final. De ahí sale el razonamiento en vivo de la interfaz.
- El harness detecta el hardware y valida el modelo antes de cargarlo. Si un modelo no cabe en la VRAM, avisa y lo carga en RAM, más lento, sin fallar en silencio.
- Las herramientas viajan por el **canal estructurado** del proveedor, no en el prompt. La petición lleva un objeto por herramienta y la respuesta puede traer `tool_calls` con nombre y argumentos; los resultados vuelven como mensajes de rol `tool`. Si ese contrato cambiara, el único módulo afectado sería el adaptador del proveedor: `agent` y `tools` no deberían enterarse. Ver [[backend/02-interfaces/TOOLS]].
- **Un turno puede abrir varias peticiones.** El harness no da por terminado un turno cuando el modelo deja de escribir: sigue mientras el modelo pida herramientas, hasta un máximo. Quien decide cuándo se acaba es el bucle de `agent`, no el stream. Ver [[specs/SPEC-AGENTE-BASE]].
- Los turnos de herramienta **no se persisten**. El historial guardado guarda el mensaje del usuario y la respuesta final del agente; lo intermedio existe solo en memoria mientras dura el turno. Ver [[database/02-rules/DATA_FLOW]].
- El límite de contexto del modelo manda: el nodo de contexto recorta para que lo entregado quepa. Ver [[backend/01-domain/DOMAIN]].
- Un status HTTP distinto de 200 trae el motivo real en el mensaje del error tipado, no solo el código. Cada adaptador normaliza el cuerpo de su proveedor —Ollama devuelve `{"error": …}`, llama.cpp JSON o texto plano— a ese mismo error.

Y lo que **sí** cambia por proveedor:

- **La ventana de contexto, que es la única diferencia que no se resuelve traduciendo.** Con Ollama **se declara** en cada petición: `/api/chat` lleva `options.num_ctx`, la menor entre la ventana que reporta el modelo en `/api/tags` (`details.context_length`) y el tope (`LOCALCLI_CONTEXT_LIMIT`, 16384 por defecto). Sin declararla, Ollama usa su valor de servidor (≈4096), recorta la lista de mensajes y devuelve 500 `no user query found in messages` en cuanto el turno encadena herramientas. Con llama.cpp **se lee**: el harness consulta `/props` y trabaja con `default_generation_settings.n_ctx`, que fijó quien arrancó el servidor. Si no se puede leer —porque el router todavía no tiene ningún modelo cargado, por ejemplo— la ventana se trata como desconocida y el turno se recorta al tope. En los dos casos el nodo de contexto y el presupuesto del historial se derivan de la ventana efectiva. Ver [[backend/DECISIONS]] y [[specs/SPEC-MODELO-PROVEEDOR]].
- **El formato de las imágenes.** Con Ollama viajan en `images` (base64) del mensaje de `/api/chat`; con llama.cpp en `image_url` de la parte de usuario. Si el contrato de imágenes de un proveedor cambiara, el módulo afectado es su adaptador y el resto no debería enterarse.

Si el contrato de streaming de un proveedor cambiara, el módulo afectado es su adaptador —`ollama` u `openai`—, y el resto no debería enterarse.

### Contrato con la búsqueda

- La búsqueda devuelve solo lo público de internet.
- El contenido que vuelve es texto sin trust: entra al contexto como cualquier otro documento, sujeto al mismo recorte y auditoría. Ningún agente lo vuelca sin procesarlo.

## Referencias

- [[specs/SPEC-MODELO-PROVEEDOR]] — proveedor, modelo, hardware, límites y concurrencia.
- [[specs/SPEC-TOOLS]] — el contrato funcional de las herramientas.
- [[backend/02-interfaces/TOOLS]] — la capa universal y las herramientas del usuario.
- [[backend/03-security/SECURITY]] — qué sale y qué no.
- [[backend/04-infrastructure/CONFIGURATION]] — qué proveedor se usa, cómo se habilita internet y dónde viven las herramientas del usuario.
- [[backend/04-infrastructure/EVENTS]] — cómo se notifica el streaming y las herramientas.