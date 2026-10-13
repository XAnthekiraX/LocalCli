# 005-task-llama-server.md

> T-B005 — Ciclo de vida de `llama-server`.
> Acción de esta descomposición: `crear`.

## Referencias

- [[specs/SPEC-MODELOS-LLAMACPP]] §6 Ejecución de modelos, §2 Configuración global, §8 Validaciones
- [[backend/04-infrastructure/CONFIGURATION]] §3 El proveedor, §6 Servicios requeridos
- [[backend/DECISIONS]] — el harness arranca y detiene el servidor

## Contexto

LocalCli usa `llama-server` para cargar el modelo seleccionado y exponerlo por HTTP. El harness lo **arranca y lo detiene**, configura el puerto (el de `api` en `localcli.json`), comprueba su disponibilidad y, al cambiar de modelo, lo reinicia. La ruta que se pasa a `llama-server` es la del **archivo GGUF original** en `models/`: no se copia a ninguna otra ubicación. Al arrancarlo se le nombra el modelo con `--alias <carpeta>` —el nombre del modelo es su carpeta— para que `/v1/models` y las peticiones del chat usen ese nombre y no la ruta ([[specs/SPEC-MODELOS-LLAMACPP]] §6).

No se presupone que el binario esté: si falta, o el modelo no existe, o no cabe en la máquina, se avisa sin romper el arranque.

## Tareas pequeñas

| ID        | Acción     | Tarea                                                                                             | Estado    | Archivos                    | Verificación                                                                 |
| --------- | ---------- | ------------------------------------------------------------------------------------------------- | --------- | --------------------------- | ---------------------------------------------------------------------------- |
| T-B005-01 | crear      | Localizar el binario `llama-server` (ruta configurable o descubrimiento en `PATH`)                  | completada | `internal/llama/runtime.go` | Con el binario en `PATH` se resuelve; sin él, se avisa y no se bloquea        |
| T-B005-02 | actualizar | Arrancar `llama-server` con el GGUF seleccionado, el puerto de `api`, `-c` (contexto) y `--alias <carpeta>` (nombre del modelo) | completada | `internal/llama/runtime.go` | El proceso queda escuchando en el puerto de `api`; `/v1/models` reporta el nombre del modelo, no la ruta |
| T-B005-03 | crear      | Detener el servidor de forma limpia al salir del harness o al deseleccionar                         | completada | `internal/llama/runtime.go` | No quedan procesos `llama-server` huérfanos al salir                          |
| T-B005-04 | crear      | Reiniciar el servidor al cambiar de modelo                                                          | completada | `internal/llama/runtime.go` | Cambiar de modelo detiene el anterior y arranca con el nuevo                  |
| T-B005-05 | crear      | Comprobar disponibilidad del servidor antes de usarlo                                              | completada | `internal/llama/runtime.go` | No se envía ninguna petición hasta que el servidor responde                   |
| T-B005-06 | crear      | Gestionar el puerto y detectar conflictos de ocupación                                             | completada | `internal/llama/runtime.go` | Un puerto ocupado se detecta y se avisa con claridad                          |
| T-B005-07 | crear      | Avisar cuando falta el binario, falta el modelo o el modelo no cabe en la máquina                   | completada | `internal/llama/runtime.go` | Cada caso produce un aviso distinto y accionable                              |
| T-B005-08 | crear      | Enlazar el ciclo de vida al arranque/salida del harness                                            | completada | `internal/llama/runtime.go` | Al iniciar no se arranca un segundo servidor si ya hay uno compatible         |

**Dependencias:** `01`→`02`→`03`,`04`,`05`; `02`→`06`,`07`,`08`.