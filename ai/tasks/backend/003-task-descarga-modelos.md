# 003-task-descarga-modelos.md

> T-B003 — Descarga de modelos GGUF desde URL directa (Hugging Face u otro).
> Acción de esta descomposición: `crear` (implementado).

## Referencias

- [[specs/SPEC-MODELOS-LLAMACPP]] §4 Descarga de modelos, §3 Almacenamiento de modelos, §8 Validaciones
- [[backend/04-infrastructure/CONFIGURATION]] §4 Rutas, §8 El catálogo de modelos
- [[backend/DECISIONS]] — un solo proveedor gestionado por el harness
- [[specs/SPEC-INTERFAZ]] — modal «DESCARGAR MODELOS» (`/descargar`, `<leader>d`)
- [[specs/SPEC-KEYBINDS]] — acción `model_download`

## Contexto

LocalCli permite descargar un modelo GGUF desde una URL directa (p. ej. `https://huggingface.co/<org>/<repo>/resolve/main/modelo.gguf`). El modal «DESCARGAR MODELOS» (`/descargar` o `<leader>d`) muestra un campo de URL, el nombre derivado del archivo (sin `.gguf`) y una barra de progreso en vivo. La descarga usa reanudación (`.part`), validación de tamaño y, si la URL es de Hugging Face, resolución del hash SHA256 vía la API del repositorio. El archivo se guarda en `models/<nombre>/<archivo>` y se registra en `models.json`; una descarga en curso se cancela al lanzar otra o al cerrar el modal con `Esc`.

## Tareas pequeñas

| ID        | Acción | Tarea                                                                                             | Estado    | Archivos                       | Verificación                                                                 |
| --------- | ------ | ------------------------------------------------------------------------------------------------- | --------- | ------------------------------ | ---------------------------------------------------------------------------- |
| T-B003-01 | crear  | Cliente contra la API de Hugging Face para listar los archivos `.gguf` de un repositorio           | completada | `internal/models/hf.go`        | Dado un repo conocido, devuelve sus variantes GGUF con nombre y tamaño        |
| T-B003-02 | crear  | Selección de repositorio y variante GGUF a descargar (por nombre de archivo)                        | completada | `internal/models/hf.go`        | Se elige una variante concreta y se resuelve su URL de descarga               |
| T-B003-03 | crear  | Descarga con progreso hacia `models/<modelo>/<archivo>.gguf`                                        | completada | `internal/models/descarga.go`  | El archivo aparece en su subdirectorio con el tamaño completo                 |
| T-B003-04 | crear  | Comprobar existencia previa antes de descargar (no duplicar)                                        | completada | `internal/models/descarga.go`  | Un archivo ya presente no se vuelve a descargar                               |
| T-B003-05 | crear  | Reanudar una descarga interrumpida usando el rango parcial (`Range`) cuando se pueda              | completada | `internal/models/descarga.go`  | Reanudar continúa desde el punto previo en vez de empezar de cero             |
| T-B003-06 | crear  | Validar que la descarga terminó correctamente (tamaño, y hash si el repo lo publica)               | completada | `internal/models/descarga.go`  | Una descarga truncada se detecta y no se marca como completa                  |
| T-B003-07 | crear  | Un fallo de red no borra modelos existentes ni sobrescribe la configuración                         | completada | `internal/models`              | Cortar la red a mitad de descarga deja intactos los modelos ya presentes      |
| T-B003-08 | crear  | `ParsearURL` + `NombreDeURL` para extraer `repo`/`archivo` y derivar el nombre del modelo          | completada | `internal/models/url.go`       | URL de HF resuelve repo; URL genérica extrae nombre; casos límite tolerados   |
| T-B003-09 | crear  | `Variante.URL` y `URLDescarga` que la respeta; `TamanoURL` por HEAD                                | completada | `internal/models/hf.go`        | URL directa se usa; HEAD devuelve Content-Length                              |
| T-B003-10 | crear  | `DescargarURL` que completa tamaño/hash antes de bajar                                             | completada | `internal/models/descarga.go`  | HF resuelve hash; genérico resuelve tamaño por HEAD                           |
| T-B003-11 | crear  | `Orquestador.DescargarModelo` / `CancelarDescarga` / `NombreDeURL` + eventos de descarga           | completada | `internal/session/orquestador.go` | Descarga en goroutine, progreso cada 1%, cancelación por generación         |
| T-B003-12 | crear  | `Puerto` ampliado con métodos de descarga + test double                                            | completada | `internal/tui/wire.go`, `modales_test.go` | Métodos firmados; doble implementado                                      |

**Dependencias:** `01`→`02`→`03`→`04`,`05`→`06`; `03`→`07`; `08`→`09`→`10`; `11`→`12`.

**Fuera de alcance:** selector de repositorio/variante en la TUI (el flujo ahora es URL directa, no navegación por repo). La implementación cubre descarga desde URL, validación, reanudación, registro en catálogo y cancelación.