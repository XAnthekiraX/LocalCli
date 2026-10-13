# 006-task-cliente-inferencia.md

> T-B006 — Cliente de inferencia HTTP contra `llama-server`.
> Acción de esta descomposición: `crear`.

## Referencias

- [[specs/SPEC-MODELOS-LLAMACPP]] §7 Comunicación con el harness, §8 Validaciones
- [[backend/04-infrastructure/CONFIGURATION]] §2 Variables de entorno, §3 El proveedor
- [[backend/DECISIONS]] — núcleo OpenAI-compatible; extensión nativa por runtime
- [[backend/02-interfaces/INTERFACES-GENERAL]] — la frontera `llm`

## Contexto

El cliente habla con `llama-server` en la dirección base de `localcli.json` (`api`). El transporte es el núcleo común OpenAI-compatible contra `/v1/chat/completions`, en streaming para recibir la respuesta token a token; lo que el núcleo no cubre —la ventana de contexto y las capacidades— sale por la extensión nativa `props` de llama.cpp. La lógica de comunicación permanece **separada** de la interfaz y del procesamiento de las respuestas.

No se envía ninguna petición si el servidor no está disponible (T-B005): el error de conexión se gestiona sin romper la sesión.

## Tareas pequeñas

| ID        | Acción | Tarea                                                                                          | Estado    | Archivos                    | Verificación                                                                 |
| --------- | ------ | ---------------------------------------------------------------------------------------------- | --------- | --------------------------- | ---------------------------------------------------------------------------- |
| T-B006-01 | crear  | Transporte HTTP + SSE contra `/v1/chat/completions` con lectura token a token                   | completada | `internal/llm/openai`       | Una generación simple devuelve los tokens conforme llegan                     |
| T-B006-02 | crear  | Extensión `props` de llama.cpp para ventana de contexto y capacidades                           | completada | `internal/llm/llamacpp`     | La ventana efectiva se lee del servidor cuando la informa                     |
| T-B006-03 | crear  | Manejo de errores de conexión (servidor no disponible, corte a mitad de stream)                 | completada | `internal/llm/openai`       | Cortar el servidor produce un error claro, no un cuelgue                      |
| T-B006-04 | crear  | Frontera `llm`: tipos del motor, capacidades en tres estados y regla de la ventana              | completada | `internal/llm`              | El recorte respeta el tope y la ventana conocida o desconocida                |
| T-B006-05 | crear  | Separar transporte de la interfaz y del procesamiento de la respuesta                           | completada | `internal/llm`              | El paquete `llm` no importa `tui` ni lógica de presentación                   |

**Dependencias:** `01`→`02`,`03`; `01`→`04`→`05`.