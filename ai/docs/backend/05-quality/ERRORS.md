---
title: LocalCli — errores del backend
tags: [backend, calidad]
depende_de:
  - "[[PROJECT]]"
  - "[[backend/02-interfaces/INTERFACES-GENERAL]]"
relacionado:
  - "[[backend/01-domain/BUSINESS_RULES]]"
  - "[[backend/02-interfaces/TOOLS]]"
  - "[[backend/04-infrastructure/INTEGRATIONS]]"
  - "[[backend/05-quality/VALIDATION]]"
  - "[[database/01-schema/TABLES]]"
  - "[[database/02-rules/DATA_FLOW]]"
---
# ERRORS — Errores del backend

No hay respuestas HTTP ni códigos de estado. Los errores son valores que se propagan por canales, se registran en la base cuando dejan rastro, y se muestran en la TUI. Ver [[backend/02-interfaces/INTERFACES-GENERAL]] para el estilo de propagación.

## 1. Sistema de errores

- **Un error es un valor, no una excepción.** Se propaga por el canal que corresponde y cada módulo decide si lo maneja o lo sube.
- **Un error tiene un código interno y un mensaje para la persona.** El código sirve para el motor y los tests; el mensaje se muestra en la pantalla. El mensaje dice qué pasó y, cuando aplica, qué hacer.
- **Los errores persistentes se registran en la base** cuando dejan rastro: una etapa fallida, un cambio rechazado, una auditoría de contexto. Ver [[database/01-schema/TABLES]].
- **Un error no borra trabajo ya hecho.** Si una etapa falla, lo que ya quedó aplicado permanece; el flujo se detiene, no se deshace. Ver [[backend/01-domain/BUSINESS_RULES]].
- **Un comando que falla no detiene el trabajo.** El agente ve el error y sigue. Ver [[backend/02-interfaces/TOOLS]].

## 2. Errores conocidos

Los que el usuario puede encontrarse y merece la pena distinguir:

| Situación | Qué ve el usuario | Qué hace el sistema |
|---|---|---|
| Ollama no está corriendo | Aviso de que no se puede generar | El harness sigue vivo; la interfaz funciona |
| El modelo elegido no cabe en la VRAM | Aviso de que irá a RAM, más lento | Lo carga igual, sin fallar en silencio |
| Falta Landlock | Aviso de que la garantía de la terminal es más débil | Sigue funcionando con la garantía reducida |
| Una etapa pide un documento que no existe | Aviso de qué falta | La etapa se detiene; no se supone nada |
| El contexto no cabe en el modelo | El recorte ocurre, y queda registrado | Se entrega lo que cabe, y se registra qué salió |
| Una escritura se declina | El agente recibe el rechazo | Propone otra cosa |
| Un comando no termina o no para de imprimir | Aviso de que se cortó | Se corta; el harness no se cuelga ni llena el contexto |
| Una tarea está bloqueada | Aviso de qué dependencia falta | La cola sigue con otra si nada depende de ella |
| Se intenta borrar fuera de la carpeta | Pide permiso y explicación | Sin las dos, se rechaza |
| La carpeta no es un proyecto válido | Aviso al abrir | No se abre una base de datos en un sitio que no toca |
| La base del proyecto es de una versión anterior del binario | Aviso de que hay que borrar `.localcli/state.db` | No se escribe sobre un esquema que no reconoce; la base es estado del proyecto, no dato del usuario |

## 3. Códigos internos

| Código | Significado |
|---|---|
| `E_OLLAMA_UNAVAILABLE` | Ollama no responde |
| `E_MODEL_TOO_BIG` | El modelo no cabe en la VRAM; se avisa y va a RAM |
| `E_NO_LANDLOCK` | Landlock no disponible; la garantía es más débil |
| `E_TOOL_UNKNOWN` | El modelo pidió una herramienta fuera del catálogo |
| `E_TOOL_NOT_ALLOWED` | El agente pidió una herramienta que no tiene (`plan` pidiendo escritura) |
| `E_BAD_ARGS` | Argumentos que no encajan con el contrato de la herramienta |
| `E_PATH_OUTSIDE` | Ruta fuera de la carpeta del proyecto, sin permiso y explicación |
| `E_PATH_EXISTS` | `crear_archivo` o `crear_carpeta` sobre algo que ya existe |
| `E_NEEDS_CONFIRM` | Borrado sin confirmación explícita |
| `E_NEEDS_APPROVAL` | Escritura sin aprobación |
| `E_APPROVAL_DECLINED` | El usuario declinó la operación |
| `E_CMD_NOT_WHITELISTED` | Comando fuera de la lista blanca |
| `E_CMD_TIMEOUT` | El comando no terminó a tiempo; se cortó |
| `E_CMD_OUTPUT_TRUNCATED` | La salida superó el límite; se cortó y se avisa |
| `E_DOC_NOT_FOUND` | El modelo pidió un documento que no existe |
| `E_DOC_PARSE` | Un documento no se pudo leer o su frontmatter no se pudo parsear; el archivo concreto se queda fuera del grafo y el resto se carga |
| `E_CONTEXT_TOO_BIG` | Lo que se quería entregar no cabe ni tras recortar |
| `E_ELEMENTO_BLOQUEADO` | Un elemento del TODO no puede arrancar por dependencias |
| `E_STAGE_FAILED` | Una etapa falló; el flujo se detiene |
| `E_FLOW_CANCELLED` | El flujo fue cancelado |
| `E_NOT_A_PROJECT` | La carpeta abierta no es un proyecto válido |
| `E_DB_UNAVAILABLE` | No se pudo abrir o leer la base de datos del proyecto |
| `E_DB_SCHEMA_OUTDATED` | El archivo de base no corresponde a la versión del binario |
| `E_DB_CONSTRAINT` | Un `CHECK` o un `NOT NULL` rechazó la escritura |
| `E_DB_FOREIGN_KEY` | Se escribió un hijo sin su fila padre |
| `E_DB_CONFLICT` | La fila ya existe: clave primaria o unicidad repetida |
| `E_DB_LOCKED` | Otra escritura sostiene el bloqueo; es transitorio y reintentable |

## 4. Manejo de excepciones

Go no tiene excepciones, así que el "manejo" es explícito en cada punto donde puede fallar:

- **Se revisa el error en la frontera de cada módulo.** Un módulo que recibe un error decide si lo maneja o lo sube. Ninguno lo ignora en silencio.
- **Los errores de una etapa se convierten en `E_STAGE_FAILED` para el motor**, que detiene el flujo y deja que el usuario decida.
- **Los errores de una herramienta se devuelven al agente**, que los ve y sigue. Un fallo de herramienta no es un fallo de etapa por sí solo.
- **Los errores de la base se traducen antes de salir de `store`**, para que el resto del motor no sepa si fue un bloqueo, una restricción o una conexión.
- **Un error de la base tras un cambio de archivo no se traga.** Se revisa qué se aplicó y se deja constancia; ver [[database/02-rules/DATA_FLOW]] para el detalle de la relación entre archivo y registro.
- **Un error de Landlock al escribir por terminal es un resultado esperado, no una avería.** La terminal no puede escribir, y eso es la garantía funcionando.

## 5. Errores por operación

- **Escritura de archivo:** `E_PATH_OUTSIDE`, `E_PATH_EXISTS`, `E_NEEDS_APPROVAL`, `E_APPROVAL_DECLINED`, `E_NEEDS_CONFIRM` (al borrar), `E_BAD_ARGS`. Al aplicar, si falla la base, el archivo puede quedar sin registrar; se deja constancia. Ver [[database/02-rules/DATA_FLOW]].
- **Lectura de archivo:** `E_PATH_OUTSIDE` (fuera de la carpeta sin permiso), `E_BAD_ARGS`, y un error de archivo que no existe.
- **Comando de terminal:** `E_CMD_NOT_WHITELISTED`, `E_NEEDS_APPROVAL`, `E_CMD_TIMEOUT`, `E_CMD_OUTPUT_TRUNCATED`. Un comando que sale con error no es un `E_`: es un resultado con la salida de error, y el agente sigue.
- **Internet:** `E_BAD_ARGS` si la consulta o la dirección no valen. Lo que vuelve es contenido sin confianza, no un error.
- **Contexto:** `E_DOC_NOT_FOUND`, `E_CONTEXT_TOO_BIG`, `E_DOC_PARSE`.
- **Carga de documentación:** un frontmatter roto de UN archivo es un `E_DOC_PARSE` localizado:
  el archivo no entra al grafo, la carga continúa y el error queda reportado. Para el motor es un
  `E_STAGE_FAILED` (reconocible con `errors.Is`, sin leer el mensaje), porque la etapa no puede seguir
  sin avisar; el código propio queda en el mensaje como diagnóstico. Una raíz que no es carpeta de
  documentación es `E_STAGE_FAILED` directo.
- **Cola:** `E_ELEMENTO_BLOQUEADO`; la cola no es un error, sigue con otra.
- **Flujo:** `E_STAGE_FAILED`, `E_FLOW_CANCELLED`.
- **Sesiones:** `E_NOT_A_PROJECT` al abrir; los demás casos son operacionales, no de arranque.
- **Base de datos:** `E_DB_UNAVAILABLE`, `E_DB_SCHEMA_OUTDATED`, `E_DB_CONSTRAINT`, `E_DB_FOREIGN_KEY`, `E_DB_CONFLICT`, `E_DB_LOCKED`. Ninguno de ellos es `E_BAD_ARGS`: ese código es para el contrato de una herramienta, y un fallo de la base es de otra capa. Ver [[database/02-rules/DATA_FLOW]].
- **Ollama:** `E_OLLAMA_UNAVAILABLE`, `E_MODEL_TOO_BIG`. Ver [[backend/04-infrastructure/INTEGRATIONS]].

## Referencias

- [[backend/05-quality/VALIDATION]] — qué se valida antes de operar.
- [[backend/01-domain/BUSINESS_RULES]] — qué se detiene y qué sigue tras un fallo.
- [[backend/02-interfaces/TOOLS]] — errores por herramienta.
- [[database/02-rules/DATA_FLOW]] — errores al aplicar un cambio y registrarlo.
- [[PROJECT]] — límites conocidos del proyecto.
