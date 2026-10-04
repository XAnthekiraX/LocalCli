---
title: LocalCli — servicios externos
tags: [backend, infraestructura]
depende_de:
  - "[[backend/DECISIONS]]"
  - "[[backend/03-security/SECURITY]]"
  - "[[backend/05-quality/VALIDATION]]"
  - "[[specs/SPEC-MODELO-MOTOR]]"
  - "[[specs/SPEC-TOOLS]]"
relacionado:
  - "[[backend/01-domain/DOMAIN]]"
  - "[[backend/04-infrastructure/CONFIGURATION]]"
  - "[[backend/04-infrastructure/EVENTS]]"
  - "[[specs/SPEC-INTERFAZ]]"
---
# INTEGRATIONS — Servicios externos

Dos integraciones, de las que solo una sale de la máquina: los motores de inferencia —`ollama` y `llamacpp`, los dos en local— y la búsqueda por internet. Ninguna otra sale. Ver [[backend/03-security/SECURITY]] para las fronteras de seguridad.

## 1. APIs y servicios externos

### Motores de inferencia

- **Propósito:** es el modelo. Genera las respuestas, el razonamiento, las decisiones de contexto y las peticiones de herramientas. Sin motor no hay respuestas; el harness es un cliente de él.
- **Dónde:** API HTTP en local. Hay dos tipos cerrados y varias instancias de cada uno, declaradas en `~/.config/localcli/motores.json`; cada entrada trae su nombre, su tipo, su URL, si está activa y qué **extensiones nativas** declara. Ver [[backend/04-infrastructure/CONFIGURATION]] y [[specs/SPEC-MODELO-MOTOR]].
- **Núcleo común:** todos los motores hablan la misma superficie **OpenAI-compatible** —`POST /v1/chat/completions` y `GET /v1/models`—. Es el contrato mínimo y no hay diferencias contra él entre runtimes.
- **Streaming:** obligatorio, y siempre sobre el núcleo común. Peticiones en streaming para poder mostrar el razonamiento mientras llega, que es el requisito de la interfaz. Ver [[specs/SPEC-INTERFAZ]].
- **Perfil de hardware:** el harness detecta la máquina, avisa si el modelo elegido no cabe en la VRAM y limita el contexto. El modelo lo elige el usuario, no el harness. Solo se comprueba el encaje si el motor declara la extensión que da tamaño y familia.
- **Canal de herramientas:** es la parte de la que LocalCli depende por completo. Cada petición al modelo lleva las definiciones de las herramientas del agente activo —nombre, descripción y esquema de argumentos— y el modelo responde pidiendo una por su nombre con los argumentos ya formados. Un turno puede necesitar varias peticiones: pedir, ejecutar, volver a pedir. No hay formato en prosa de reserva, así que **un modelo sin esta capacidad no puede trabajar con herramientas**, y el agente cae a modo conversación. Ver [[specs/SPEC-TOOLS]].
- **Capacidades:** el harness pregunta al motor qué declara saber cada modelo y **normaliza la respuesta a tres** —herramientas, visión y razonamiento— en **tres estados cada una**: soportada, no soportada y **desconocida**. El estado `desconocida` aparece cuando el motor no expone la extensión que declara las capacidades, y es información sobre el motor, no sobre el modelo. Ninguna de las tres bloquea la elección del modelo. Ver [[backend/02-interfaces/TOOLS]].
- **Imágenes:** para los modelos multimodales, el harness adjunta al turno las rutas de imagen que el usuario escribe en el mensaje. Viajan por el núcleo común como `image_url` con data URI. Las imágenes son efímeras: acompañan a ese turno y no se guardan en el historial.
- **Concurrencia:** el harness serializa las respuestas **por orden de llegada por motor**, con una cola propia para cada uno, porque en el hardware objetivo solo cabe un modelo cargado a la vez. El turno de inferencia se toma **por petición**, no por ejecución completa, para que una sesión esperando una aprobación de herramienta no retenga el modelo. Dos motores distintos no se esperan entre sí. Es una consecuencia del hardware, no un defecto del diseño. Ver [[backend/DECISIONS]].

Los dos motores sirven los mismos pesos: lo que cambia es el runtime y cuánto declara cada uno, no el modelo. Ver [[specs/SPEC-MODELO-MOTOR]].

#### El núcleo común

Lo que **no** cambia entre motores, porque los dos runtimes lo exponen igual:

| Superficie | Uso |
|---|---|
| `POST /v1/chat/completions` | Petición y streaming. SSE con `data: {…}` y `data: [DONE]` que cierra |
| `delta.content` | Texto final |
| `delta.reasoning_content` | Razonamiento |
| `delta.tool_calls` | Peticiones de herramienta, **acumuladas por índice**: una misma llamada puede repartirse en varias deltas |
| `tools` / rol `tool` con `tool_call_id` | Definiciones de herramientas y sus resultados |
| `image_url` con data URI | Imágenes |
| `usage` | Conteo de tokens |
| `GET /v1/models` | Lista de modelos |

Las llamadas de herramienta llegan **por índice** y hay que acumularlas antes de poder ejecutarlas. Es el único detalle del núcleo que exige estado, y es el mismo para los dos.

#### Extensiones nativas

Lo que el estándar **no** cubre. Cada motor declara cuáles trae; el harness solo las consulta si están declaradas:

| Extensión | Motor | Endpoint | Qué aporta |
|---|---|---|---|
| `num_ctx` | `ollama` | `options.num_ctx` en la petición | Declarar la ventana por petición |
| `show` | `ollama` | `POST /api/show` | Capacidades del modelo (`completion`, `tools`, `vision`, `thinking`) |
| `tags` | `ollama` | `GET /api/tags` | Tamaño, familia y `details.context_length` del modelo |
| `props` | `llamacpp` | `GET /props` | Capacidades del servidor (`chat_template_caps`, `modalities.vision`) y `default_generation_settings.n_ctx` |

Una extensión ausente **no es un fallo**: el motor sigue hablando el núcleo común y lo que la extensión aportaba queda **desconocido**. El harness lo declara desconocido en vez de suponerlo, y lo dice en la interfaz con `?`.

#### Notas por motor

- **`ollama`**: expone el núcleo en `/v1`. Sus extensiones son `num_ctx`, `show` y `tags`. `/api/show` da las capacidades del modelo; `/api/tags` da tamaño, familia y la ventana que el modelo declara.
- **`llamacpp`**: expone el núcleo en `/v1`. Su extensión es `props`. Con la extensión declarada la ventana **se lee** —la fijó quien arrancó el servidor con `-c`—; sin ella, se desconoce y manda el tope.
- **`llama-server` tiene que ir en modo router**, con `--models-dir`. Un servidor por defecto no es admisible: el harness llama por nombre de modelo y el router es lo que atiende esa llamada. La autocarga se deja activada para no tener que pedirle que cargue nada.
- **Estado del servidor:** es de la persona que lo levanta. El harness **no** le pide descargar un modelo (`POST /models`), ni cargarlo o descargarlo (`POST /models/load`), ni ajustar su ventana (`POST /props`).
- **Banderas:** `llama-server` tiene que levantarse **sin** las opciones que sirven lectura o escritura de archivos (`--tools`, `--agent`, `--mcp-servers-json`). Un motor con acceso al sistema de archivos no es admisible. Ver [[backend/03-security/SECURITY]].

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
| Cliente HTTP de la biblioteca estándar | Hablar con los motores de inferencia e internet | Sin SDK de terceros; el SSE se lee con `bufio` de la estándar |
| Landlock (syscall del kernel) | Bloqueo estructural de escritura en la terminal | Linux 5.13+; en otros sistemas no está |
| `modernc.org/sqlite` | Acceso a SQLite sin cgo | Solo lo usa `store` |
| Bubble Tea + Lip Gloss | La TUI | Los usa `tui`, no el orquestador |

## 4. Credenciales y configuración requerida

Ninguna. Los dos motores corren en local sin credenciales, y la búsqueda por internet no pide clave de API. LocalCli no guarda ni pide secretos. Si algún día una integración los pidiera, se documentaría aquí.

Que `llama-server` acepte `--api-key` no cambia esto: el harness no lo usa, porque el servidor es de loopback y al que se le pone clave es al que hay que enseñarle la clave.

Lo que sí se necesita en la máquina, pero no es una credencial: al menos un motor corriendo —`ollama` o `llama-server`— y Landlock disponible si se quiere el aislamiento fuerte. Ver [[backend/04-infrastructure/CONFIGURATION]].

## 5. Contratos externos

### Contrato con el motor de inferencia

Lo que el harness da por cierto con cualquier motor:

- El harness espera respuestas en streaming, con el razonamiento distinguible del texto final. De ahí sale el razonamiento en vivo de la interfaz.
- **Todo motor habla el núcleo OpenAI-compatible.** Si un runtime no lo expone, no es admisible: no hay una segunda forma de hablarle al harness.
- El harness detecta el hardware y valida el modelo antes de cargarlo, **si el motor declara la extensión que da tamaño y familia**. Si no la declara, no afirma que el modelo quepa ni que no quepa.
- Las herramientas viajan por el **canal estructurado** del núcleo común, no en el prompt. La petición lleva un objeto por herramienta y la respuesta trae `tool_calls` con nombre y argumentos; los resultados vuelven como mensajes de rol `tool` con su `tool_call_id`. Si ese contrato cambiara, el único módulo afectado sería el adaptador del motor: `agent` y `tools` no deberían enterarse. Ver [[backend/02-interfaces/TOOLS]].
- **Un turno puede abrir varias peticiones.** El harness no da por terminado un turno cuando el modelo deja de escribir: sigue mientras el modelo pida herramientas, hasta un máximo. Quien decide cuándo se acaba es el bucle de `agent`, no el stream. Ver [[specs/SPEC-AGENTE-BASE]].
- Los turnos de herramienta **no se persisten**. El historial guardado guarda el mensaje del usuario y la respuesta final del agente; lo intermedio existe solo en memoria mientras dura el turno. Ver [[database/02-rules/DATA_FLOW]].
- **El recorte lo hace el harness.** El nodo de contexto recorta para que lo entregado quepa, y antes de enviar, el presupuesto del turno recorta la lista de mensajes de más nuevo a más viejo hasta que quepa. El harness **no** confía en que el servidor recorte por su cuenta. Ver [[backend/01-domain/DOMAIN]] y [[backend/DECISIONS]].
- Un status HTTP distinto de 200 trae el motivo real en el mensaje del error tipado, no solo el código. Cada adaptador normaliza el cuerpo de su motor —`ollama` devuelve `{"error": …}`, `llamacpp` JSON o texto plano— a ese mismo error.

Y lo que **sí** depende de las extensiones declaradas, no del tipo de motor:

- **La ventana de contexto.** Con la extensión `num_ctx` (Ollama) **se declara** en cada petición, y es la menor entre la que reporta el modelo en `/api/tags` (`details.context_length`) y el tope (`LOCALCLI_CONTEXT_LIMIT`, 16384 por defecto). Con la extensión `props` (llama.cpp) **se lee**: el harness consulta `/props` y trabaja con `default_generation_settings.n_ctx`, que fijó quien arrancó el servidor; si no se puede leer —porque el router todavía no tiene ningún modelo cargado— la ventana queda **desconocida**. Sin ninguna de las dos, también queda desconocida: manda el tope y el aviso dice que el límite es del harness, no del servidor. Ver [[backend/DECISIONS]] y [[specs/SPEC-MODELO-MOTOR]].
- **Las capacidades del modelo.** Con la extensión `show` (Ollama) salen de `/api/show`; con `props` (llama.cpp), de las capacidades del servidor. Sin extensión quedan **desconocidas**, y el harness no las convierte en «no las tiene» ni en «sí las tiene».
- **El tamaño y la familia del modelo.** Solo la extensión `tags` los da. Sin ella no se comprueba el encaje con el hardware.

Sobre una extensión ausente **nunca** se hereda nada: cada punto es independiente, y un motor puede declarar la ventana y no las capacidades, o al revés.

### Contrato con la búsqueda

- La búsqueda devuelve solo lo público de internet.
- El contenido que vuelve es texto sin trust: entra al contexto como cualquier otro documento, sujeto al mismo recorte y auditoría. Ningún agente lo vuelca sin procesarlo.

## Referencias

- [[specs/SPEC-MODELO-MOTOR]] — motor, modelo, hardware, límites y concurrencia.
- [[specs/SPEC-TOOLS]] — el contrato funcional de las herramientas.
- [[backend/02-interfaces/TOOLS]] — la capa universal y las herramientas del usuario.
- [[backend/03-security/SECURITY]] — qué sale y qué no.
- [[backend/04-infrastructure/CONFIGURATION]] — qué motores se declaran, cómo se habilita internet y dónde viven las herramientas del usuario.
- [[backend/04-infrastructure/EVENTS]] — cómo se notifica el streaming y las herramientas.