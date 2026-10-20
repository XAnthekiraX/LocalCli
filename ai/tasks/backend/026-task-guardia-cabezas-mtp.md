# 026-task-guardia-cabezas-mtp.md

> T-B026 — Rechazo de cabezas MTP y diagnóstico de caídas de `llama-server`.
> Acción de esta descomposición: `crear` (implementado).

## Referencias

- [[specs/SPEC-MODELOS-LLAMACPP]] §8 Validaciones
- [[backend/04-infrastructure/CONFIGURATION]] §8 El catálogo de modelos
- [[backend/04-infrastructure/INTEGRATIONS]] — notas del runtime
- [[backend/05-quality/ERRORS]] — `E_MODEL_DRAFT_HEAD`

## Contexto

Un GGUF puede ser una **cabeza MTP** (draft head): solo trae la capa de predicción
multi-token y se carga junto al modelo principal con `-md` y
`--spec-type draft-mtp`. Registrarla como modelo y pasarla a `llama-server` como
si fuera completa termina en una caída del motor (en la práctica, un `SIGSEGV`
que el aviso anterior ocultaba). Hay que rechazarla en la descarga y en el
arranque, y hacer que un fallo del motor diga cómo terminó y dónde está su
salida completa.

## Tareas pequeñas

| ID        | Acción | Tarea                                                                                              | Estado    | Archivos                            | Verificación                                                                 |
| --------- | ------ | -------------------------------------------------------------------------------------------------- | --------- | ----------------------------------- | ---------------------------------------------------------------------------- |
| T-B026-01 | crear  | Leer metadatos y tabla de tensores del GGUF sin cargar sus datos                                    | completada | `internal/models/gguf.go`           | Un GGUF de prueba expone arquitectura, contexto, capas, `nextn` y tensores   |
| T-B026-02 | crear  | Detectar cabezas MTP por nombre/ruta (`mtp-…`, carpeta `MTP/`) y por estructura                     | completada | `internal/models/draft.go`          | Un GGUF con solo la última capa y `nextn` se detecta; un modelo completo no   |
| T-B026-03 | crear  | Rechazar la URL de descarga de una cabeza MTP antes de bajarla                                      | completada | `internal/models/url.go`            | `ParsearURL` falla con una URL bajo `MTP/` y acepta el modelo principal       |
| T-B026-04 | crear  | No registrar cabezas MTP en el catálogo                                                             | completada | `internal/models/catalogo.go`       | `AltaDescarga` devuelve error y no crea la entrada                            |
| T-B026-05 | crear  | Avisar y no arrancar `llama-server` con una cabeza MTP ya catalogada                                | completada | `internal/llama/runtime.go`         | `Asegurar` devuelve un aviso accionable y no lanza el proceso                 |
| T-B026-06 | crear  | Incluir el código de salida o la señal en el error de arranque del motor                            | completada | `internal/llama/runtime.go`         | Un binario que sale con 3 o con `SIGSEGV` aparece nombrado en el error        |
| T-B026-07 | crear  | Persistir la salida completa del motor en `~/.config/localcli/logs/llama-server.log` y citarla       | completada | `internal/llama/runtime.go`, `cmd/localcli/main.go` | El registro conserva la salida y el aviso cita su ruta            |

**Dependencias:** `01`→`02`; `02`→`03`,`04`,`05`; `06` y `07` son independientes.

**Fuera de alcance:** arrancar cabezas MTP junto al modelo principal (`-md`,
`--spec-type draft-mtp`) y validar la integridad completa del GGUF más allá de su
cabecera.
