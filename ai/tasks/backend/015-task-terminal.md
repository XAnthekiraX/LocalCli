# 015-task-terminal.md

> T-B015 — Terminal (`exec`): lista blanca y bloqueo estructural con Landlock.
> Acción de esta descomposición: `crear`.

## Referencias

- [[backend/01-domain/DOMAIN]] §1 Módulos (`exec`)
- [[backend/02-interfaces/TOOLS]] §1 El catálogo incluido (§Terminal), §5 Herramientas de terminal, §7 La forma de una herramienta, §8 La capa universal
- [[backend/02-interfaces/dto/TOOLS-DTO]] §3 Request Schemas (§Terminal), §4 Response Schemas (§Terminal)
- [[specs/SPEC-TOOLS]] §Herramientas de terminal
- [[backend/03-security/SECURITY]] §3 Protección de las operaciones (§Terminal bloqueada estructuralmente)
- [[backend/04-infrastructure/CONFIGURATION]] §5 Configuración por ambiente, §6 Servicios requeridos (Landlock)
- [[backend/05-quality/VALIDATION]] §1 Comandos de terminal
- [[backend/05-quality/ERRORS]] §3 Códigos internos, §5 Errores por operación (§Comando de terminal)
- [[backend/DECISIONS]] — Landlock para bloquear escritura, la aprobación en `Contexto.Ask`, los límites del comando

## Contexto

`exec` implementa la herramienta de terminal. Tiene **dos controles independientes**: una lista blanca que decide qué comandos corren sin preguntar y un bloqueo que garantiza que la terminal no puede crear, editar ni borrar archivos del proyecto.

Como cualquier herramienta, no decide permisos por su cuenta: el permiso lo comprueba la capa universal de `tools` (T-B012), y la aprobación de lo que no está en la lista blanca llega por `Contexto.Ask`. Aquí solo se decide la lista blanca, se aplica el aislamiento y se ejecuta.

Dos reglas que no admiten excepción: **el texto del comando no se usa para decidir si es seguro** —el bloqueo es estructural, del kernel (Landlock), no un filtro de texto— y **un comando que falla no detiene el trabajo**: el agente ve la salida de error y sigue.

## Tareas pequeñas

| ID         | Acción | Tarea                                                                                                                       | Estado    | Archivos                   | Verificación                                                                                       |
| ---------- | ------ | --------------------------------------------------------------------------------------------------------------------------- | --------- | -------------------------- | -------------------------------------------------------------------------------------------------- |
| T-B015-01  | crear  | Petición `ejecutar_comando` (`comando` requerido, `carpeta` opcional con el proyecto por defecto) y el contenido de `Salida` (`salida`, `error`, `codigo`, `truncado`, `termino`) | completada | `internal/exec/comando.go` | El esquema derivado coincide con el documentado en `TOOLS-DTO` §3/§4                                |
| T-B015-02  | crear  | Lista blanca por comando: compilar, probar, estilo y tipos, y `git` estado/diferencias/historial corren sin preguntar; lo demás marca aprobación (`E_CMD_NOT_WHITELISTED`) | completada | `internal/exec/lista.go`   | `go build`/`go test`/`git status` no piden permiso; un comando arbitrario pide aprobación          |
| T-B015-03  | crear  | Aprobación de lo no blanco vía `Contexto.Ask`, con el comando a la vista; nunca se ejecuta sin aprobar                       | completada | `internal/exec/aprobacion.go` | Un comando no blanco declinado devuelve `E_APPROVAL_DECLINED`                                      |
| T-B015-04  | crear  | Bloqueo estructural con Landlock: escritura denegada en el proyecto; concedido solo el espacio propio (temporales, caché, carpeta por sesión) y los dispositivos nulos `/dev/null`, `/dev/zero`, `/dev/full` | completada | `internal/exec/landlock.go` | `tee` y `sh -c '> archivo'` no escriben; `git status` (abre `/dev/null` con O_RDWR) funciona       |
| T-B015-05  | crear  | Detección de Landlock: si no está disponible, avisar con `E_NO_LANDLOCK` y seguir con garantía débil, sin bloquear el uso    | completada | `internal/exec/landlock.go` | En un sistema sin Landlock se emite `E_NO_LANDLOCK` y la terminal sigue funcionando                 |
| T-B015-06  | crear  | Límite de tiempo y de salida: cortar un comando sin fin o con salida sin fin y avisar (`E_CMD_TIMEOUT`, `E_CMD_OUTPUT_TRUNCATED`), sin colgar el harness | completada | `internal/exec/limites.go` | Un `yes`/`sleep` se corta y la salida marca `truncado=true`                                        |
| T-B015-07  | crear  | Ejecutor de subproceso reutilizable: correr en la carpeta del proyecto, capturar salida y salida de error y recoger el código; un comando que falla no corta el turno | completada | `internal/exec/ejecutor.go` | Un comando con código ≠ 0 devuelve su salida con `termino=true` y sin `error` de Go                |
| T-B015-08  | crear  | Handler `Ejecutar` de `ejecutar_comando` sobre el ejecutor, para inyectar en el catálogo de `tools`                          | completada | `internal/exec/handler.go`  | `Registro.Ejecutar` de `ejecutar_comando` devuelve el `Resultado` esperado                          |
| T-B015-09  | crear  | Cableado: registrar `ejecutar_comando` en el catálogo y montar el ejecutor con la lista blanca, el aislamiento y el `Contexto.Ask` | completada | cableado (`cmd/localcli`)   | El agente activo recibe `ejecutar_comando` con el ejecutor ya inyectado                            |
| T-B015-10  | crear  | Test: lista blanca sin permiso, comando fuera pide aprobación, Landlock impide escribir, timeout y truncado avisan          | completada | `internal/exec/exec_test.go` | Los cuatro casos se cubren                                                                          |

**Dependencias:** `01`→`02`; `02`→`03`; `03`,`04`,`05`→`06`; `01`→`07`; `06`→`07`; `07`→`08`; `08`→`09`; `07`→`10`.
**Fuera de alcance:** la capa universal, el permiso, el recorte, la medición, los eventos y la composición del tema de la línea (T-B012); el esquema que ve el modelo, derivado por reflexión en `tools` (T-B012); `fileops` (T-B014).
