# 023-task-ventana-contexto-real.md

> T-B023 — Ventana de contexto real del modelo y recorte efectivo del turno.
> Acción de esta descomposición: `actualizar`.

## Referencias

- [[backend/DECISIONS]] §1 — la decisión de la ventana desde el modelo; refina la 75 (recorte del turno en el harness)
- [[backend/04-infrastructure/CONFIGURATION]] §2, §3, §7, §8 — `LOCALCLI_CONTEXT_LIMIT` como override, `-c` de arranque, `historial_tokens` y `ventana` del catálogo
- [[backend/04-infrastructure/INTEGRATIONS]] §1, §5 — `-c` desde el GGUF y ventana efectiva por `props`
- [[specs/SPEC-MODELOS-LLAMACPP]] §6 — arranque del servidor con el contexto del modelo
- [[backend/01-domain/DOMAIN]] §recorte — el presupuesto del turno

## Contexto

Un turno que leía dos documentos grandes hacía que `llama-server` respondiera
`400 Bad Request: request (26619 tokens) exceeds the available context`: la
ventana que creía el harness era ficticia. Se arrancaba el servidor con
`-c = 16384` fijo y la ventana se releía de `props` (`default_generation_settings.n_ctx`),
que es ese mismo `-c` —circular—; nunca se consultaba el contexto real del GGUF.
Además, nada acotaba el prompt del turno: el historial se resumía a 4096 tokens,
pero los resultados de herramientas que acumula `agent.Ejecutor` no se contaban.
Esto contradice la decisión 75 (el recorte lo hace el harness), aprobada pero sin
implementar.

## Tareas pequeñas

| ID         | Acción     | Tarea                                                                                                                                                                                                    | Estado     | Archivos                                                        | Verificación                                                                                     |
| ---------- | ---------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------- | --------------------------------------------------------------- | ------------------------------------------------------------------------------------------------ |
| T-B023-01  | crear      | Lector de cabecera GGUF: `ContextoDeGGUF` lee `general.context_length` sin tocar tensores                                                                                                                | completada | `internal/models/gguf.go`, `internal/models/gguf_test.go`       | Fixture con y sin la clave, magic inválido y archivo truncado                                 |
| T-B023-02  | actualizar | `Entrada.ContextoReal()` y alta desde el GGUF cuando llega `0`; `LOCALCLI_CONTEXT_LIMIT` pasa a override (`LimiteContextoOverride`)                                                                      | completada | `internal/models/catalogo.go`, `internal/arranque/config.go`    | Un modelo sin `ventana` en el catálogo la lee del GGUF                                            |
| T-B023-03  | actualizar | `llama.Modelo.Contexto` decide el `-c` de arranque; `llm.VentanaEfectiva` (override > modelo > defecto) y `llm.EstimarTokens`                                                                            | completada | `internal/llama/runtime.go`, `internal/llm/ventana.go`          | `argsDeArranque` usa el contexto del modelo; `VentanaEfectiva` respeta la precedencia              |
| T-B023-04  | actualizar | `main`: resolver el contexto antes de arrancar el servidor y fijar la ventana efectiva por `props`                                                                                                       | completada | `cmd/localcli/main.go`                                          | El servidor arranca con `-c` del modelo; la ventana es la que reporta `props`                     |
| T-B023-05  | actualizar | Recorte por ronda en `agent.Ejecutor` (`Ventana` + `ajustarAlContexto`): descarta lo más antiguo, trunca el resultado que no quepa, quita el `tool` huérfano y avisa                                          | completada | `internal/agent/ejecutor.go`, `internal/agent/recorte_test.go`  | Con ventana pequeña el motor recibe menos mensajes; sistema + último usuario conservados           |
| T-B023-06  | actualizar | Cableado de sesión: `fijarVentana`, relectura de `props` al cambiar de modelo, ventana al `Ejecutor` de chat y de etapa; presupuesto del historial derivado de la ventana                                  | completada | `internal/session/orquestador.go`, `internal/session/flujo.go`, `internal/session/historial.go` | Cambiar de modelo reajusta la ventana; el historial usa la mitad de la ventana si no hay preferencia |

**Fuente de verdad:** [[backend/DECISIONS]] (decisión de la ventana desde el modelo)
y [[specs/SPEC-MODELOS-LLAMACPP]]. No se implementa nada que no esté documentado ahí.

**Dependencias:** `01`→`02`→`04`; `03`→`04`,`05`; `05`,`06` cierran sobre el resto.
**Fuera de alcance:** estimar tokens de imágenes adjuntas; medir tokens con un
tokenizador real (se sigue estimando por longitud).
