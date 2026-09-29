---
title: LocalCli — auditoría de código y cobertura documentación↔implementación
tags: [auditoria, backend, frontend, database]
depende_de:
  - "[[PROJECT]]"
  - "[[backend/BACKEND]]"
  - "[[frontend/FRONTEND]]"
  - "[[database/DATABASE]]"
relacionado:
  - "[[backend/DECISIONS]]"
  - "[[backend/03-security/SECURITY]]"
  - "[[specs/SPEC-TOOLS]]"
---
# Code Audit — LocalCli

> **Auditoría read-only del código.** Ningún archivo fuente del proyecto fue creado, modificado o eliminado como parte de este informe. Todos los hallazgos son observaciones documentadas con evidencia. El único archivo añadido por esta auditoría es el propio informe, en `ai/audit/`.

## 1. Resumen

### Proyecto

* **Nombre:** LocalCli
* **Stack detectado:** Go 1.21 · Bubble Tea + Lip Gloss (TUI) · SQLite vía `modernc.org/sqlite` (sin cgo) · Landlock vía `golang.org/x/sys/unix` · Ollama por API HTTP. Binario único.
* **Alcance:** proyecto completo — 138 archivos `.go` de producción (18.357 LOC), 83 de test (15.036 LOC, 524 `Test*`), 51 documentos en `ai/docs/`, 57 archivos de tareas en `ai/tasks/`.
* **Fecha:** 2026-09-28
* **Ruta de salida:** `ai/audit/`
* **Skills aplicadas:** `code-auditor` (calidad, arquitectura, seguridad, rendimiento, mantenibilidad) + verificación cruzada documentación↔código.

### Estado general

La base técnica es sólida: `go vet`, `gofmt`, `go build` y `go test -race` están limpios; no hay carreras de datos detectadas, ni inyección SQL, ni secretos en código, ni `TODO`/`FIXME` sin resolver. Los invariantes de seguridad locales (frontera de rutas, permisos por acción, approvals, keymap) están bien implementados y bien testados.

El proyecto tiene **dos problemas estructurales**, ninguno de ellos de calidad de código:

1. **La garantía de la terminal es evadible.** `internal/exec` identifica el programa por nombre base, no por ruta, y nunca inspecciona los flags posteriores al subcomando. Confirmado con PoC: 9 vectores ejecutan binarios arbitrarios **sin pedir aprobación**. Es el hallazgo más grave del proyecto porque anula la decisión central registrada en [[backend/DECISIONS]] ("Landlock para bloquear escritura en la terminal") y la regla de [[backend/03-security/SECURITY]].

2. **La documentación declara implementado lo que no está cableado.** 71,4% de los criterios de aceptación están implementados, pero el patrón de los 25 criterios faltantes es inequívoco: funciones escritas, probadas y **sin llamador productivo** (`flow.GenerarTODO`, `session.Rederrivar`, `ollama.DetectarHardware`, `ollama.AvisoNoCabe`, `ollama.ContextoLimitadoTokens`). Los 24 ítems de `ai/tasks/backend/MAIN-TASKS.md` figuran como "completada" sin excepción, incluidas las dos tareas cuyo contenido está desconectado. El tablero no es verificable y por tanto no cumple su función de fuente de verdad del progreso.

A esto se suma un test rojo (`internal/task`), el `.gitignore` literalmente escrito como `(empty)` con la base de datos de conversaciones versionada en git, y dos archivos de composición (`tui/app.go` 1.315 LOC, `arranque.go` 1.173 LOC) que erosionan la separación de responsabilidades que el propio proyecto declara.

### Resumen de hallazgos

| Severidad | Cantidad | IDs |
|-----------|---------:|-----|
| CRITICAL  |        2 | AUD-001, AUD-002 |
| HIGH      |        7 | AUD-003 … AUD-009 |
| MEDIUM    |        5 | AUD-010 … AUD-014 |
| LOW       |       22 | AUD-015 … AUD-036 |
| INFO      |        5 | AUD-037 … AUD-041 |
| INVESTIGATE |      2 | AUD-042, AUD-043 |
| **Total** |    **43** | |

### Las dos cifras de cobertura

| Pregunta | Resultado | Desglose |
|---|---|---|
| **¿Cuánta documentación está implementada?** | **71,4%** | 177/248 criterios de aceptación implementados · 38 parciales (15,3%) · 25 sin implementar (10,1%) · 8 no verificables (3,2%). Usable (impl. + parcial): **86,7%** |
| **¿Cuánto código está documentado?** | **21,5%** | 148/688 símbolos exportados denominativos citados en `ai/docs/` · 29,7% contando `ai/tasks/` · 13,3% con match estricto (símbolo entre backticks) |

Matiz que evita la lectura errónea de ambas cifras: la documentación es **arquitectónica y de dominio, no referencial**. Cubre el 100% de la superficie conceptual — 13/13 módulos, 6/6 tablas, 13/13 herramientas, 4/4 enums, 130/138 archivos fuente citados por nombre, 70% de los códigos de error. Si se mide documentación de *diseño* el resultado es alto (~85-90%); si se mide documentación de *API/símbolo* es 21,5%. Detalle completo en §15 (Anexo A) y §16 (Anexo B).

---

## 2. Hallazgos críticos

### [CRITICAL] La lista blanca de la terminal se evade con una ruta absoluta

**ID:** AUD-001

**Ubicación:**
`internal/exec/whitelist.go:42`

**Problema:**
`EnListaBlanca` identifica el programa permitido por `filepath.Base(argv[0])`, nunca por la ruta resuelta. Cualquier ruta absoluta o relativa cuyo *basename* sea `go`, `git` o `gofmt` casa con una regla de la lista y se ejecuta **sin pedir aprobación** (`internal/exec/run.go:115-129`).

**Impacto:**
Ejecución de código arbitrario sin consentimiento del usuario. El modelo puede pedir `ejecutar_comando` con `/tmp/cualquiera/go test ./...` y el motor ejecuta ese binario. Esto anula por completo la primera de las dos barreras de la terminal y deja la garantía del proyecto en manos de un `Base()`.

**Evidencia:**
```go
// internal/exec/whitelist.go:42-46
programa := filepath.Base(argv[0])
for _, r := range listaBlanca {
    if programa != r.programa {
        continue
    }
```

PoC ejecutado sobre una copia literal de `whitelist.go` en `/tmp/opencode/poc/`:

```
BYPASS CONFIRMADO [binario falso por ruta absoluta]: [/tmp/evil/go test ./...]  -> EnListaBlanca=true
BYPASS CONFIRMADO [binario falso en subdirectorio]: [./tools/go test]          -> EnListaBlanca=true
BYPASS CONFIRMADO [git falso por ruta absoluta]:     [/tmp/evil/git status]     -> EnListaBlanca=true
BYPASS CONFIRMADO [gofmt falso por ruta absoluta]:   [/tmp/evil/gofmt main.go]  -> EnListaBlanca=true
```

Los controles negativos pasan: `rm -rf /`, `sh -c id`, `go run`, `gofmt -w` y `git push` sí piden aprobación. El defecto es exclusivamente la identificación por nombre base.

**Recomendación:**
Resolver el ejecutable antes de comparar y validar la ruta canónica:
1. Rechazar `argv[0]` que contenga un separador de ruta (`strings.ContainsRune(argv[0], filepath.Separator)`) — el proyecto ya tiene `Validar` (`internal/exec/validate.go:35`) como punto natural para esa comprobación.
2. Para el resto, resolver con `exec.LookPath` + `filepath.EvalSymlinks` y comparar contra un conjunto de rutas absolutas de permitidos poblado en el arranque, no contra nombres.

**Justificación:**
El comentario del propio archivo (líneas 11-13) declara la intención correcta —"se identifica el programa por su ejecutable"— pero la implementación hace exactamente lo contrario. Comparar rutas canónicas cierra la clase completa: enlaces simbólicos, rutas relativas, directorios en `PATH` manipulados y copias del binario en cualquier ubicación.

**Categoría:** `Security`

---

### [CRITICAL] La lista blanca se evade con los flags posteriores al subcomando

**ID:** AUD-002

**Ubicación:**
`internal/exec/whitelist.go:57`

**Problema:**
`primerSubcomando` devuelve el primer argumento que **no** empieza por `-`, y a partir de ahí nada más se examina. Los flags que siguen al subcomando nunca se inspeccionan, aunque la lista blanca sea correcta por ruta. `go test -exec <binario>` ejecuta un programa arbitrario durante la fase de test de cada paquete; `go build -toolexec` hace lo mismo en compilación; `git log --output=<ruta>` y `git diff --ext-diff=<script>` escriben y ejecutan fuera de la garantía.

**Impacto:**
Segunda vía independiente hacia ejecución arbitraria sin aprobación, y la más cómoda de explotar para el modelo: solo necesita escribir en el repositorio. Corregir únicamente AUD-001 **no cierra** este hallazgo.

**Evidencia:**
```go
// internal/exec/whitelist.go:57-64
func primerSubcomando(args []string) string {
    for _, a := range args {
        if !strings.HasPrefix(a, "-") {
            return a
        }
    }
    return ""
}
```

Mismo PoC, resultado:

```
BYPASS CONFIRMADO [go test -exec]:        [go test -exec /tmp/evil/payload ./...]        -> true
BYPASS CONFIRMADO [go test -toolexec]:    [go test -toolexec=/tmp/evil/payload ./...]    -> true
BYPASS CONFIRMADO [go build -exec]:       [go build -o /tmp/out -exec /tmp/evil/p ...]  -> true
BYPASS CONFIRMADO [git log --output]:     [git log --output=/tmp/exfil.txt]              -> true
BYPASS CONFIRMADO [git diff --ext-diff]:  [git diff --ext-diff=/tmp/evil/payload]        -> true
```

Un caso Mukherjee Async: `git -C /etc status` **sí** se rechaza, pero por accidente — `/etc` es el primer argumento sin flag y no casa con `status`. No es una protección, es una coincidencia de la implementación.

**Recomendación:**
Dejar de validar subcomandos y validar *acciones*: permitir `go` solo con flags de solo lectura, y denegar explícitamente la lista de flags con capacidad de ejecutar o escribir fuera de las permitidas del proyecto: `-exec`, `-toolexec`, `-c`, `-C`, `--output`, `--ext-diff`, `--exec-path`, `-o` con ruta absoluta fuera del proyecto.

**Justificación:**
Es la misma corrección conceptual que aplicó el proyecto a `gofmt` (líneas 47-50: en vez de decidir por el subcomando, se comprueba si el flag concreto escribe). Extender ese patrón ya establecido al resto de la lista es coherente con el diseño existente y cubre los flags compuestos (`-w=`, `-lw`).

**Categoría:** `Security`

---

## 3. Hallazgos importantes

### [HIGH] Landlock degrada en silencio y solo restringe escritura

**ID:** AUD-003

**Ubicación:**
`internal/exec/landlock_linux.go:41`

**Problema:**
Tres degradaciones silenciosas en la garantía estructural:
1. `aplicarLandlock` falla → se escribe a stderr y **se ejecuta igual** (líneas 41-46).
2. `run.go:179-188`: si `os.Executable()` falla, se cae a `exec.CommandContext` sin Landlock y sin aviso.
3. `run.go:189`: si `soportaLandlock()` es falso, se ejecuta sin trampolín.

Además `leerPermitidos` (líneas 78-94) no maneja `LANDLOCK_ACCESS_FS_READ_FILE` ni `READ_DIR`: la política es solo de escritura.

**Impacto:**
La garantía declarada en [[backend/DECISIONS]] ("no se esquiva con `find -delete` ni redirecciones") desaparece sin que el usuario lo note, y en combinación con AUD-001/AUD-002 un binario arbitrario escribe en todo el sistema de ficheros. Sin restricción de lectura, `go test` compila y **ejecuta** el código de test del repositorio sin aprobación: ese código puede leer `~/.aws/credentials` y exfiltrarlo, ya que la red no está restringida.

**Evidencia:**
```go
// internal/exec/landlock_linux.go:41-46
if err := aplicarLandlock(leerPermitidos(os.Getenv(envPermitidos))); err != nil {
    // Degradación avisada: se ejecuta igual, con garantía más débil
    fmt.Fprintln(os.Stderr, err.Error())
}
```

La degradación **está documentada** en [[backend/05-quality/ERRORS]] §2 como decisión consciente ("Falta Landlock → aviso de que la garantía de la terminal es más débil; sigue funcionando"). Por eso este hallazgo es HIGH y no CRITICAL: el comportamiento es intencionado. Lo que no es intencionado es que laCombinación con la lista blanca evadible lo convierta en un fallo silencioso real.

**Recomendación:**
Mantener la degradación (es una decisión correcta para Portabilidad), pero: (a) propagar el error de `os.Executable()` en vez de caer al caminho sin protección; (b) añadir `READ_FILE`/`READ_DIR` a las rutas permitidas para que la frontera sea bidireccional; (c) exponer el estado de la garantía en la línea de estado de la TUI (ya existe una para modelo y herramientas, `T-F025`) en lugar de solo en stderr.

**Justificación:**
Convierte una degradación invisible en un estado observable, y cierra la lectura de credenciales sin retirar la flexibilidad de portability que la decisión documentada persigue.

**Categoría:** `Security`

---

### [HIGH] `.gitignore` no ignora nada: conversaciones y binario versionados

**ID:** AUD-004

**Ubicación:**
`.gitignore:1`

**Problema:**
El archivo contiene literalmente la cadena `(empty)`. En sintaxis de gitignore eso es un **patrón literal**, no un comentario: no ignora nada. `git ls-files` confirma que `.localcli/state.db` y el binario `localcli` (17.853.358 bytes) están versionados. `state.db-wal` y `state.db-shm` aparecen sin trackear pero tampoco ignorados.

**Impacto:**
Datos personales publicados en el repositorio. La base contiene 6 sesiones, 17 mensajes y 8 razonamientos con contenido de las conversaciones del usuario. La historia de git es irreversible sin reescritura. El binario de 17 MB satura el repositorio en cada cambio de versión.

**Evidencia:**
```bash
$ cat -A .gitignore
(empty)$          # sin salto de línea, sin '#'

$ git ls-files | grep -E 'state.db|^localcli$'
.localcli/state.db
localcli

$ python3 -c "import sqlite3; c=sqlite3.connect('.localcli/state.db'); \
    [print(t, c.execute('select count(*) from '+t).fetchone()[0]) \
     for t in ['sessions','messages','reasoning']]"
sessions 6
messages 17
reasoning 8
```

Contradicción directa con [[backend/DECISIONS]], fila *"Archivo SQLite en `.localcli/state.db`… y `.localcli/` fuera de git"*. La decisión está escrita; el `.gitignore` que debía ejecutarla no existe.

**Recomendación:**
```gitignore
.localcli/
/localcli
*.db-wal
*.db-shm
```
Luego limpiar el índice (`git rm --cached .localcli/state.db localcli`) y purgar la historia (`git filter-repo --path .localcli --path localcli --invert-paths`) si el repositorio ya fue publicado.

**Justificación:**
Restaura una decisión ya aprobada y detiene la fuga de datos. La purga de historia es irreversible, por lo que conviene decidir si el repositorio llegó a publicarse antes de ejecutarla.

**Categoría:** `Security`

---

### [HIGH] Escritura de archivo no atómica: un fallo a mitad deja el archivo vacío

**ID:** AUD-005

**Ubicación:**
`internal/fileops/write.go:96`

**Problema:**
`EscribirArchivo` usa `os.WriteFile`, que trunca el destino y escribe encima. Un fallo intermedio (disco lleno, EIO, SIGKILL) deja el archivo **vacío o corrupto** y devuelve error. El propio archivo declara en las líneas 10-11 que "escritura y registro son un solo paso lógico", y `CrearArchivo` (líneas 69-72) ya compensa con temporal + rename. `EscribirArchivo` no.

**Impacto:**
Pérdida de datos en el contenido de un proyecto. Es la operación que el agente usa en cada iteración de `build` sobre archivos existentes, y sobre un archivo que ya tenía contenido previo.

**Evidencia:**
```go
// internal/fileops/write.go:96-101
if err := os.WriteFile(abs, []byte(contenido), 0o644); err != nil {
    ...
if err := o.registrar(store.OpEscribirArchivo, mustRelativa(...), before, contenido); err != nil {
    return tools.RespuestaEscritura{}, err
}
```

**Recomendación:**
Reutilizar exactamente el patrón que `CrearArchivo` ya tiene: escribir a un temporal en el mismo directorio, `fsync`, y `rename` atómico. Aplicarlo también a `EditarArchivo`.

**Justificación:**
La solución ya existe a 25 líneas de distancia en el mismo archivo; no es una decisión de diseño nueva, es inconsistencia interna.

**Categoría:** `Correctness`

---

### [HIGH] `EscribirArchivo` no revierte si falla el registro en el historial

**ID:** AUD-006

**Ubicación:**
`internal/fileops/write.go:99`

**Problema:**
Si `o.registrar` falla, la función devuelve error **pero el archivo ya quedó sobrescrito**. `CrearArchivo` revierte correctamente con el contenido `before` justo dos líneas más arriba; `EscribirArchivo` no.

**Impacto:**
El contenido en disco cambió, la operación se reporta como fallida y no queda traza en `change_history`. El agente reintenta el mismo cambio y el usuario ve un error donde el cambio **sí ocurrió** — exactamente la situation que la tabla `change_history` existe para evitar ([[backend/DECISIONS]]: "el registro de qué se tocó en los archivos… no puede depender de que la sesión siga existiendo").

**Evidencia:**
`before` ya está leído en `write.go:86-89` y está disponible para revertir. La asimetría con la ruta de creación es visible en el mismo bloque.

**Recomendación:**
Igual que en la ruta de creación: ante error de `registrar`, restaurar `before` en el archivo y devolver un error que nombre las dos operaciones (escritura revertida + historial no registrado).

**Justificación:**
Deja disco e historial coherentes, que es la invariante que el módulo `fileops` declara sostener.

**Categoría:** `Correctness`

---

### [HIGH] Cierre forzado de sesión sin esperar a la goroutine en vuelo

**ID:** AUD-007

**Ubicación:**
`internal/session/bg.go:150`

**Problema:**
`Cerrar(..., forzar)` cancela el contexto y hace `Borrar(sesionID)` sin esperar a `t.hecho`. `bg.go:197-207` sigue ejecutando y `bg.go:206` descarta el error con `_ = correr(cctx)`. No existe ningún `WaitGroup` en producción (solo en los tests) para drenar el trabajo al salir. Al borrar la fila, `cerrarTurno` (`session/run.go:272`) falla con `ErrNoEncontrado` y los mensajes posteriores violan la FK.

**Impacto:**
Escrituras sobre filas eliminadas, errores silenciosos y, en el cierre del proceso, un WAL sin consolidar. Es el camino que se ejecuta al salir de la aplicación.

**Evidencia:**
```go
// internal/session/bg.go:206
_ = correr(cctx)   // el error se descarta
```

**Recomendación:**
Tras cancelar, esperar `t.hecho` con un timeout acotado antes de `Borrar`. Propagar el error de `correr` a un evento en lugar de descartarlo. Registrar el `WaitGroup` de las goroutines de sesión en el `Cerrar` de la aplicación.

**Justificación:**
Un cierre ordenado es la diferencia entre "el proceso terminó" y "el proceso terminó sin dejar basura". `go test -race` no detecta esto porque no hay carrera: es un problema de orden de ciclo de vida.

**Categoría:** `Correctness`

---

### [HIGH] Lectura arbitraria de ficheros al adjuntar imágenes

**ID:** AUD-008

**Ubicación:**
`internal/tui/adjuntos.go:57`

**Problema:**
`RutasImagen` solo comprueba extensión y existencia; `AdjuntosDe` hace `os.ReadFile` de **cualquier ruta** sin frontera de proyecto ni límite de tamaño. Cualquier `.png` alcanzable —incluido un symlink a un fichero de 2 GiB— entra en base64 y se envía a Ollama. El paquete `fileops`, que resuelve el mismo problema, sí tiene `LimiteLecturaBytes` y frontera de rutas.

**Impacto:**
Lectura de ficheros arbitrarios y agotamiento de memoria, con una vía que el resto del proyecto ya cerró para `leer_archivo` pero que el camino de las imágenes reopeninge.

**Evidencia:**
```go
// internal/tui/adjuntos.go:57-67
if _, err := os.Stat(ruta); err != nil { continue }
datos, err := os.ReadFile(ruta)
imagenes = append(imagenes, base64.StdEncoding.EncodeToString(datos))
```

**Recomendación:**
Validar la ruta con `fileops.Resolver` (el mismo que usa `boundary.go:23-45`) y aplicar `fileops.LimiteLecturaBytes` antes de leer. Rechazar symlinks que escapen de la raíz.

**Justificación:**
Elimina un segundo camino hacia un caso que el módulo de seguridad ya sabe cerrar; la consistencia con `fileops` es la defensa menos costosa.

**Categoría:** `Security`

---

### [HIGH] La herramienta de internet permite HTTP sin cifrar y destinos internos (SSRF)

**ID:** AUD-009

**Ubicación:**
`arranque.go:806`

**Problema:**
El handler de internet antepone `https://` **solo cuando falta `://`**, de modo que un `http://` explícito se acepta. Usa `http.Client` por defecto, que sigue redirecciones sin revalidar. No hay bloqueo de loopback, de `169.254.169.254` (metadata de nube) ni de puertos internos.

**Impacto:**
SSRF desde el agente: el modelo puede hacer que el proceso alcanzó servicios internos del host o endpoints de metadata, y además puede degradar a texto plano.

**Evidencia:**
```go
// arranque.go:806-809 — solo antepone https si falta el esquema
```

**Recomendación:**
Forzar `https` (rechazar o redirigir `http://` explícito), usar un `http.Client` propio con `CheckRedirect` que revalide cada salto, y denegar por resolución las IPs privadas, loopback y link-local.

**Justificación:**
El riesgo no es la red abierta —la herramienta ya requiere el flag `LOCALCLI_ALLOW_INTERNET` y permiso `internet`— sino que ese permiso se conceda sobre un cliente que ignora a dónde va realmente.

**Categoría:** `Security`

---

### [MEDIUM] El relevo `plan` → `build` no existe

**ID:** AUD-010

**Ubicación:**
`internal/flow/engine.go:147`

**Problema:**
El motor pasa el **mismo `objetivo`** a cada etapa (línea 147) y el resultado de una etapa solo se emite como evento (líneas 186-190): nunca viaja a la etapa siguiente. `build` nunca ve la propuesta de `plan`. El comentario de `internal/flow/work.go:9-12` afirma lo contrario. Además la etapa `documentacion` de `flow/work.go:37` usa `tools.AgentePlan`, que por construcción no tiene herramientas de escritura, así que **la documentación nunca se escribe**.

**Impacto:**
El ciclo de trabajo documentado en [[specs/SPEC-CICLO-TRABAJO]] no funciona: la fase de documentación es un no-op silencioso, y el agente de construcción no parte del análisis.

**Evidencia:**
```go
// internal/flow/engine.go:186-190 — el resultado solo se publica como evento
```

**Recomendación:**
Transportar el resultado de la etapa anterior al `objetivo` de la siguiente (o a un campo de contexto explícito del tipo `entradas`), y asignar `tools.AgenteBuild` a la etapa `documentacion`.

**Justificación:**
`Encontro` entre lo que la spec promete y lo que el código hace es el hallazgo de mayor alcance funcional del proyecto, y se cierra con un cambio de una línea en el motor más una corrección de agente.

**Categoría:** `Architecture`

---

### [MEDIUM] `Limite` del nodo de contexto nunca se fija: por defecto no recorta

**ID:** AUD-011

**Ubicación:**
`internal/context/model.go:119`

**Problema:**
`Nodo.Limite` nunca se asigna en producción. `arranque.go:944-949` construye el nodo sin `Limite`; `LimiteDeEntorno()` devuelve 0 si no hay `LOCALCLI_CONTEXT_LIMIT`; y `Recortar` con `limite <= 0` **devuelve todo sin recortar** (`context/trim.go:22-28`). El único uso de `Limite:` es la copia del mismo valor en `nodo.go:162`.

**Impacto:**
La garantía de [[specs/SPEC-NODO-CONTEXTO]] ("el contexto entregado nunca supera el límite de contexto del modelo") **no se sostiene por defecto**. Con un modelo local, es el escenario más probable: la petición crece, Ollama responde con un error de contexto y el usuario no tiene diagnóstico de por qué.

**Evidencia:**
```go
// internal/context/trim.go:22-28
if limite <= 0 {
    return todo, Recorte{Recortado: false}
}
```

**Recomendación:**
Derivar el límite por defecto del tamaño de contexto del modelo elegido (que ya se obtiene en `arranque.go:917` vía el perfil) en lugar de dejarlo a 0, y tratar 0 como «consultar el modelo» y no como «sin límite».

**Justificación:**
Un cero que desactiva silenciosamente la protección es peor que ningún parámetro. El dato para derivarlo ya existe en el arranque.

**Categoría:** `Correctness`

---

### [MEDIUM] El perfil de hardware está escrito pero nunca conectado

**ID:** AUD-012

**Ubicación:**
`internal/ollama/profile.go:43`

**Problema:**
`DetectarHardware` no se invoca en ningún sitio: el arranque usa `PerfilPorDefecto()` fijo (`arranque.go:917`). `AvisoNoCabe` (línea 107) y `ContextoLimitadoTokens` (línea 121) no tienen llamadores productivos — solo su test. `Adaptador.Modelos()` (`arranque.go:309-337`) solo consulta `/api/show` y nunca llama a `ClasificarModelos`. El comentario de `arranque.go:923-925` afirma "el aviso de tamaño ya se mostró al listar", lo cual es **falso**.

**Impacto:**
Los tres criterios de [[specs/SPEC-OLLAMA-PERFIL]] sobre cabida en hardware quedan sin efecto. El usuario elige un modelo que no cabe y no recibe ningún aviso; la consecuencia es un fallo de Ollama sin diagnóstico. Tres funciones probadas y muertas.

**Evidencia:**
`rg -n "DetectarHardware|AvisoNoCabe|ContextoLimitadoTokens" --include="*.go"` → declaración + test, ningún llamador de producción.

**Recomendación:**
Llamar a `DetectarHardware` en el arranque y pasar el perfil a `Adaptador.Modelos()` para clasificar; invocar `AvisoNoCabe` al seleccionar un modelo que no cabe; y usar `ContextoLimitadoTokens` para alimentar el `Limite` del nodo de contexto (cierra AUD-011 por el mismo camino).

**Justificación:**
Una función probada y sin llamador es trabajo ya pagado: conectarla cuesta una línea y entrega tres criterios de spec.

**Categoría:** `Architecture`

---

### [MEDIUM] Pausar y reanudar la cola no son alcanzables desde la interfaz

**ID:** AUD-013

**Ubicación:**
`internal/session/pause.go:111`

**Problema:**
`Pausar` y `Reanudar` existen, pero `arranque.go:544-546` solo expone `Pausar`/`Cancelar` al puerto, no hay `AccionReanudar`, y `AccionPausar` tiene binding `{}` (`internal/tui/keymap.go:183`). No hay ninguna tecla que pause ni reanude. `Rederrivar` (`session/pause.go:146`) y `queue/next.go:30` tampoco tienen consumidor.

**Impacto:**
Dos criterios de [[specs/SPEC-COLA-TAREAS]] inalcanzables. Contradice [[backend/DECISIONS]], fila *"Solo el motor lanza colas"*: "Pausar, cancelar y re-derivar ya cubren el 'quiero solo esto'". Ninguna de las dos acciones está conectada.

**Evidencia:**
`internal/tui/keymap.go:183` → binding vacío; `rg "Rederrivar"` → sin llamador fuera de tests.

**Recomendación:**
Exponer `Reanudar` y `Rederrivar` en el puerto, y asignar tecla a `AccionPausar` en el keymap por defecto. Alternativamente, si la decisión de "sin modo de tarea suelta" se mantiene, eliminar las funciones muertas y quitar esos criterios de la spec.

**Justificación:**
Las dos salidas son válidas; lo que no es válido es tener código, spec y decisión diciendo tres cosas distintas.

**Categoría:** `Architecture`

---

### [MEDIUM] `Skills` es un campo muerto: SPEC-SKILLS al 0%

**ID:** AUD-014

**Ubicación:**
`internal/agent/model.go:54`

**Problema:**
`Skills []string` se declara y se puebla con `[]` en `arranque.go:1091`. Es el único uso del símbolo en todo el código no-test. No hay carga desde `.localcli/agents/*.json`, ni ámbito de sesión, ni resolución de markdown, ni skills por defecto.

**Impacto:**
SPEC-SKILLS completo: **0 de 6 criterios implementados**. El campo `skills` forma parte del contrato de cinco campos que declara [[backend/DECISIONS]], así que la decisión le_TEMPLATE da por vigente algo que no hace nada. Un `plan` sin skills escribe un prompt que promete capacidades que no tiene.

**Evidencia:**
```go
// internal/agent/model.go:54
Skills []string `json:"skills"`

// arranque.go:1091
Skills: []string{},
```

**Recomendación:**
Dos caminos, y hay que elegir uno explícitamente: implementar la spec (carga de `ai/skills/`, ámbito,markdown), o declarar SPEC-SKILLS como no aprobada en la v1 y retirar el campo del JSON de agente y de [[backend/DECISIONS]]. La segunda es más barata y coherente con el resto de la v1.

**Justificación:**
Un campo que se serializa, se documenta como decisão y no se lee es peor que un campo ausente: induce a error al que lea la spec.

**Categoría:** `Architecture`


### [LOW] `gofmt` sin `-w` imprime el contenido de cualquier fichero Go alcanzable

**ID:** AUD-015

**Ubicación:**
`internal/exec/whitelist.go:47`

**Problema:**
`gofmt` está en la lista blanca sin subcomandos, y la única condición para pedir aprobación es `escribeGofmt(argv[1:])`. `gofmt -l` y `gofmt` a secas recorren el árbol y **escriben en stdout** el contenido de cada fichero que visita. La lista blanca da por sentado que «gofmt sin escribir es solo revisar», pero la salida de `gofmt` es el archivo completo.

**Impacto:**
Volcado masivo de contenido a stdout —incluidos ficheros con secretos incrustados— a través de un canal que se considera seguro por definición. Combinado con el límite de 10 KB, el daño se acota en volumen, pero el contenido filtrado es legible.

**Evidencia:**
```go
// internal/exec/whitelist.go:47-50
if len(r.subcomandos) == 0 {
    // gofmt escribe con -w/-i; cualquier otra forma solo revisa.
    return !escribeGofmt(argv[1:])
}
```

**Recomendación:**
Sustituir `gofmt` por `gofmt -l` en la lista blanca (lista de ficheros, no contenido), que es además lo que un ciclo agentic necesita de verdad para detectar estilo roto.

**Justificación:**
El comentario de la línea 48 describe una suposición que no se cumple para la salida estándar. `gofmt -l` da la misma información de estilo con una fracción del volumen.

**Categoría:** `Security`

---

### [LOW] `panic` en producción al construir el keymap por defecto

**ID:** AUD-016

**Ubicación:**
`internal/tui/keymap.go:256`

**Problema:**
Si el mapa interno de teclas resulta inválido, la función hace `panic(err)`. Es una invariante interna, pero un `keys.json` de usuario corrupto puede disparar la construcción con datos inválidos y tumbar la TUI entera en lugar de degradar a los atajos de fábrica.

**Impacto:**
Un archivo de configuración del usuario (fuera del proyecto) impide arrancar la aplicación. El `panic` no distingue invariante rota por bug de datos externos corruptos.

**Evidencia:**
```go
// internal/tui/keymap.go:256-263
if err := ...; err != nil {
    panic(err)
}
```

**Recomendación:**
Degradar a `KeymapPorDefecto()` con un aviso en la línea de estado, reservando el `panic` para el caso de que también falle la tabla interna de fábrica.

**Justificación:**
El proyecto ya eligió degradar en lugar de morir en el caso análogo de Landlock (AUD-003). Aplicar el mismo criterio aquí es coherente.

**Categoría:** `Maintainability`

---

### [LOW] El cliente HTTP de Ollama no tiene timeout pese a declarar uno

**ID:** AUD-017

**Ubicación:**
`internal/ollama/client.go:197`

**Problema:**
Existe `var _ = 5 * time.Second` con el comentario *"timeoutTransport deja constancia del límite defensivo"*, pero `NewClient` construye `&http.Client{}` sin `Timeout` ni timeouts de dial/TLS/response-header. La línea no tiene efecto: es código muerto que documenta una protección inexistente.

**Impacto:**
Un Ollama colgado deja la goroutine de sesión bloqueada indefinidamente. Como el FIFO de inferencia tiene capacidad 1, un modelo colgado **bloquea todas las demás sesiones** en lugar de solo la suya.

**Evidencia:**
```go
// internal/ollama/client.go:197
var _ = 5 * time.Second   // "timeoutTransport deja constancia del límite defensivo"
```

**Recomendación:**
Implementar el `http.Transport` con `DialContext` y `ResponseHeaderTimeout`, o eliminar la constante y documentar que el timeout vive exclusivamente en el contexto del llamador. Lo que no debe quedar es un comentario que promete una garantía que no existe.

**Justificación:**
Un comentario de seguridad que describe un límite inexistente es peor que la ausencia del límite: induce a revisar el código creyendo que hay red de seguridad.

**Categoría:** `Correctness`

---

### [LOW] Los errores de la goroutine de cola se descartan

**ID:** AUD-018

**Ubicación:**
`arranque.go:453`

**Problema:**
`consumirCola` se lanza con `context.Background()` y descarta varios errores con `_ =`. Un fallo de la cola —derivación, orden por dependencias, escritura de estado— no llega a la vista ni al usuario.

**Impacto:**
Fallo silencioso del motor de colas. El usuario ve la TUI aparentemente viva mientras la cola dejó de avanzar, sin ningún indicador.

**Evidencia:** `arranque.go:453-476`, varios `_ =` en el cuerpo de la goroutine.

**Recomendación:**
Publicar los errores como evento del bus (el mecanismo ya existe en `session`) para que la TUI pueda mostrarlos, y derivar la cancelación del contexto de la aplicación en lugar de `context.Background()`.

**Justificación:**
El sistema de eventos ya resuelve exactamente este problema para otros eventos del motor; es reutilización, no infraestructura nueva.

**Categoría:** `Maintainability`

---

### [LOW] `os.Exit` anula el cierre diferido de la base de datos

**ID:** AUD-019

**Ubicación:**
`main.go:40`

**Problema:**
`main.go:34` registra `defer a.Cerrar()` pero, si Bubble Tea falla, `main.go:40-42` hace `os.Exit(1)`. En Go, `os.Exit` **no ejecuta los `defer`**, así que la base de datos nunca se cierra.

**Impacto:**
Salida sin consolidar el WAL. Con `PRAGMA foreign_keys = ON` y escritura concurrente de razonamiento (decisión documentada: ~5 escrituras/s), un cierre abrupto deja trabajo a medias para la siguiente apertura.

**Evidencia:**
```go
// main.go:34 y 40-42
defer a.Cerrar()
...
os.Exit(1)   // el defer no se ejecuta
```

**Recomendación:**
Devolver el error desde `main` (patrón `func run() error` + `os.Exit` en el `main` real) para que todos los `defer` se ejecuten siempre.

**Justificación:**
Es el idioma correcto en Go y elimina una clase entera de cierres incompletos sin cambiar la lógica de la aplicación.

**Categoría:** `Maintainability`

---

### [LOW] `parseISO` y `strNull` no tienen ningún uso

**ID:** AUD-020

**Ubicación:**
`internal/store/util.go:42`

**Problema:**
`parseISO` y `strNull` están declarados y probados, pero ningún código de producción los invoca. Búsqueda de referencias sobre todo el módulo, incluidos tests: cero llamadas.

**Impacto:**
Código muerto que sugiere que el parseo de fechas ISO y el manejo de NULL se usan de forma distinta a la pensada. Riesgo bajo, pero es ruido en un paquete cuyo contrato es "único acceso a SQLite".

**Evidencia:** `rg -w parseISO --include="*.go"` → solo la declaración y su test.

**Recomendación:**
Eliminar ambos y sus tests, o bien conectarlos donde el formato de fecha de la base de datos lo requiera (verificar `SCHEMA.md` antes de decidir).

**Justificación:**
En `store` la ausencia de código muerto es una propiedad de la calidad del módulo; dos funciones huérfanas la erosionan.

**Categoría:** `Maintainability`

---

### [LOW] Números mágicos en los puntos de cálculo

**ID:** AUD-021

**Ubicación:**
`internal/agent/loop.go:110`

**Problema:**
Presupuesto de tokens, umbral de truncar y máximo de pasadas están escritos en los puntos de cálculo, no en constantes. El propio proyecto ya usa constantes con nombre en `internal/context/audit.go:29` y `internal/fileops/list.go:25`, y DECISIONS fija valores concretos ("cada 200 ms", "120 s", "10 KB") que en el código no aparecen con nombre.

**Impacto:**
Bajo. Dificulta el ajuste fino y contrasta con la convención que el propio proyecto ya aplica en otros paquetes.

**Evidencia:** límites repartidos entre `agent/loop.go`, `session/contexto.go` y `exec/run.go:60-82`.

**Recomendación:**
Extraer a constantes con nombre en el paquete correspondiente, siguiendo la convención ya establecida en `context/audit.go`.

**Justificación:**
Coherencia interna y legibilidad; la convención ya existe, solo falta aplicarla.

**Categoría:** `Style`

---

### [LOW] La reescritura de la línea de contexto puede no escribir

**ID:** AUD-022

**Ubicación:**
`internal/tui/adjuntos.go:57`

**Problema:**
Si `os.ReadFile` falla, el error se descarta y la imagen simplemente se salta. El usuario ve su imagen adjunta en pantalla pero el modelo nunca la recibe, sin ningún aviso.

**Impacto:**
Confusión silenciosa entre lo que el usuario ve y lo que el modelo procesa. En una sesión de análisis visual, la conclusión puede ser errónea sin señal de error.

**Evidencia:**
```go
// internal/tui/adjuntos.go:57
if _, err := os.Stat(ruta); err != nil { continue }
```
`RutasImagen` filtra por existencia, pero entre el `Stat` y el `ReadFile` hay una ventana TOCTOU y, además, no hay límite de tamaño: una imagen válida de 40 MB se acepta.

**Recomendación:**
Reportar a la línea de estado qué adjuntos se descartaron y por qué, en lugar de eliminarlos en silencio. Añadir el límite de tamaño que ya aplica `fileops`.

**Justificación:**
La regla del proyecto es que el usuario sepa qué ha ocurrido; un descarte invisible rompe esa expectativa en el único punto donde el modelo ve algo que el usuario no ve.

**Categoría:** `Correctness`

---

### [LOW] 12 códigos de error del backend no están documentados

**ID:** AUD-023

**Ubicación:**
`ai/docs/backend/05-quality/ERRORS.md`

**Problema:**
De 40 códigos `E_*` documentados, 12 no aparecen en la documentación: `E_FILE`, `E_DIR`, `E_SYM`, `E_SOCK`, `E_FIFO`, `E_REG`, `E_CHAR`, `E_ALGO`, `E_BLOCK`, `E_PATH_BENEATH`, `E_RULESET`, `E_RULESET_VERSION`. Son los errores de `fileops` y de Landlock, es decir, los de las dos fronteras de seguridad.

**Impacto:**
Bajo por sí solo, pero los códigos sin documentar son precisamente los de seguridad, y ERRORS.md es el documento que un consumidor necesita para distinguir «el usuario denegó» de «el sistema no lo permite».

**Evidencia:** 28/40 códigos documentados (70 %).

**Recomendación:**
Añadir los 12 códigos a ERRORS.md con su significado y su condición de retorno.

**Justificación:**
Cerrar la cobertura documental de la capa de seguridad, que es la de mayor coste de equivocarse.

**Categoría:** `Maintainability`

---

### [LOW] La documentación de validación está desfasada respecto al código

**ID:** AUD-024

**Ubicación:**
`ai/DOCS-VALIDATION-REPORT.md:1`

**Problema:**
El informe declara haber analizado 45 documentos (17 specs + 12 backend + 12 database + 4 frontend). Hoy hay 51 archivos `.md` en `ai/`: faltan 6 por validar. Entre los cambios posteriores a esa validación hay 7 commits que tocaron 28 archivos de `ai/docs/`, incluidos `PROJECT.md`, `BACKEND.md`, `DOMAIN.md`, `TOOLS.md` y 13 specs. Los 3 WARNING (W01 prosa en inglés, W02 typo "elrequisito", W03 `messages.created_at` ausente en TABLES) no se han re-verificado.

**Impacto:**
El informe da una confianza que ya no respalda. Quien lo lea asumirá que la documentación está validada cuando 6 documentos nunca lo han sido y el resto cambió después de la validación.

**Evidencia:** fecha declarada `2026-09-25T12:00:00Z`; último commit de docs 2026-09-28.

**Recomendación:**
Ejecutar de nuevo la skill `docs-consistency-validator` y regenerar el informe, incluyendo el recuento de documentos como dato de entrada en lugar de escrito a mano.

**Justificación:**
La validación automática solo sirve si se repite; un informe desactualizado es peor que no tener informe porque aparenta cobertura.

**Categoría:** `Maintainability`

---

### [LOW] Dos helpers duplicados y cinco tablas de centinelas repetidas

**ID:** AUD-025

**Ubicación:**
`internal/exec/whitelist.go:82`, `internal/tools/permission.go:67`, `internal/exec/errors.go:41`, `internal/tools/errors.go:54`, `internal/fileops/errors.go:43`, `internal/queue/errors.go:42`, `internal/flow/errors.go:36`

**Problema:**
El helper `contiene` está implementado de forma idéntica en `exec` y `tools`. El patrón `nuevoError` (switch que inicializa un centinela y devuelve un error tipado) está replicado en cinco paquetes, con el mismo riesgo: si el código no está en el switch, `Unwrap()` devuelve `nil` y `errors.Is` falla en silencio. Hoy todos los call sites usan códigos válidos, pero un código nuevo rompe el contrato sin error de compilación.

**Impacto:**
Bajo por volumen; el punto de riesgo es que el centinela `nil` es un fallo silencioso en cinco sitios.

**Evidencia:** mismo `switch` con `default` que deja el centinela sin asignar en los cinco `errors.go`.

**Recomendación:**
`contiene` → `slices.Contains` (Go ≥ 1.21 ya disponible). Para los centinelas, tabla `map[string]error` en un paquete común: la consulta falla de forma ruidosa en lugar de silenciosa.

**Justificación:**
Reduce cinco implementaciones del mismo patrón frágil a una, y convierte un fallo silencioso en uno visible.

**Categoría:** `Maintainability`

---

### [LOW] `resolverCarpeta` no resuelve symlinks, a diferencia de `fileops`

**ID:** AUD-026

**Ubicación:**
`internal/exec/run.go:218`

**Problema:**
Rechaza absolutas y `..` correctamente, pero no llama a `EvalSymlinks`, al contrario que `fileops.Resolver` (`internal/fileops/boundary.go:49`). Un symlink dentro del proyecto fija `cmd.Dir` (`run.go:139`) fuera de la carpeta abierta.

**Impacto:**
Inconsistencia entre dos comprobaciones de frontera del mismo proyecto: la de ficheros resuelve enlaces y la de terminal no. El working directory de un comando aprobado puede quedar fuera de la carpeta abierta.

**Evidencia:** contraste directo entre `exec/run.go:218-232` y `fileops/boundary.go:49`.

**Recomendación:**
Extraer la resolución de ruta a una función compartida y usarla en ambos paquetes.

**Justificación:**
El mismo concepto ("una ruta debe quedarse dentro de la carpeta abierta") no puede tener dos grados de exigencia distintos.

**Categoría:** `Security`

---

### [LOW] Un comando aprobado nunca puede escribir en el proyecto, y el diálogo no lo dice

**ID:** AUD-027

**Ubicación:**
`internal/exec/landlock_linux.go:19`, `internal/exec/run.go:120`

**Problema:**
`landlock_linux.go:19-20` declara que el proyecto **no** está entre las rutas con escritura; `run.go:192-193` solo concede temporales y caché. Un comando que el usuario aprueba tampoco puede escribir en la carpeta abierta, y el texto de aprobación (`run.go:120`) no lo menciona.

**Impacto:**
El usuario aprueba un comando esperando un efecto que nunca ocurre, sin aviso previo. No es peligroso, pero erosiona la confianza en el diálogo de aprobación.

**Evidencia:** `landlock_linux.go:19-20` (proyecto excluido) frente a `run.go:120` (texto de aprobación).

**Recomendación:**
Añadir una línea al texto de aprobación indicando que la escritura queda restringida a temporales, o conceder la carpeta abierta de forma explícita y separada.

**Justificación:**
El diálogo de aprobación debe describir el efecto real de lo que se aprueba.

**Categoría:** `Maintainability`

---

### [LOW] La frontera de permiso y la de aprobación se solapan sin quedar declarada

**ID:** AUD-028

**Ubicación:**
`internal/tools/permission.go:52`, `internal/fileops/approval.go:24`

**Problema:**
El permiso se decide en `tools` y se aplica en `fileops`/`exec`, que es la decisión documentada en [[backend/DECISIONS]]. Pero `fileops/approval.go:24-45` comprueba la aprobación **además** del permiso, sin que quede explícito en el código cuál de las dos es la frontera real: un lector no puede distinguir por la firma si `fileops` sabe qué herramienta pidió o solo que algo pidió aprobación.

**Impacto:**
Bajo, pero la garantía de "una sola fuente" queda repartida entre dos comprobaciones de apariencia similar, lo que dificulta razonar sobre ella.

**Evidencia:** `tools/permission.go:52-64` decide; `fileops/approval.go:24-45` vuelve a comprobar.

**Recomendación:**
Declarar en la firma de cada una si es defensa en profundidad o frontera primaria, y dejarlo escrito en [[backend/03-security/SECURITY]].

**Justificación:**
La duplicación puede ser intencionada; lo que falta es que el código lo declare para que no se lea como descuido.

**Categoría:** `Maintainability`

---
## 4. Mejoras menores

### [LOW] El tablero de progreso marca 57/57 tareas como completadas sin ser verificable

**ID:** AUD-029

**Ubicación:**
`ai/tasks/backend/MAIN-TASKS.md:7`, `ai/tasks/frontend/MAIN-TASKS.md:7`

**Problema:**
Las 57 filas declaran `completada`. La verificación contra el código da 71,4% de criterios implementados, con 25 ausentes y 38 parciales. `ai/tasks/` es la fuente de verdad del progreso, pero no distingue "función escrita" de "componente integrado": las tareas cerradas incluyen `T-F032` (paleta de comandos), cuyo test falla.

**Impacto:**
El tablero no cumple su función: transmite completitud donde hay 25 criterios sin implementar. Quien lo use para decidir qué falta se equivoca.

**Evidencia:**
`git status` muestra `ai/tasks/frontend/032-task-paleta-comandos.md` sin trackear, y `TestCargarElementosDeTodasLasCapas` espera 32 elementos de frontend cuando hay 33 (`task_test.go:225`).

**Recomendación:**
Añadir un estado intermedio (`en curso`, `verificada`) que exija un criterio comprobable —el criterio de aceptación de la tarea pasa— en lugar del cierre administrativo. Corregir el contador de `task_test.go:225` o derivarlo del archivo en vez de hardcodearlo.

**Justificación:**
Un tablero de progreso solo sirve si refleja la realidad; hoy la sobreestima en casi un 30%.

**Categoría:** `Architecture`

---

### [LOW] Test en rojo: el conteo de tareas de frontend está desactualizado

**ID:** AUD-030

**Ubicación:**
`internal/task/task_test.go:225`

**Problema:**
El test espera 32 elementos grandes en `ai/tasks/frontend/MAIN-TASKS.md` y hay 33 (T-F032). El propio comentario del test reconoce que el conteo por longitud de ID es frágil, porque las subtareas de frontend tienen 9 caracteres igual que el corte.

**Impacto:**
`go test ./...` falla. La suite roja oculta fallos reales, y este test es precisamente el que valida la fuente de verdad del progreso.

**Evidencia:**
```
--- FAIL: TestCargarElementosDeTodasLasCapas (0.00s)
    task_test.go:225: capa frontend: 33 elementos grandes de MAIN-TASKS, queremos 32
```

**Recomendación:**
Contar los elementos `| T-F...` de MAIN-TASKS directamente sobre el archivo en vez de por longitud de ID, o derivar el número esperado del propio contenido.

**Justificación:**
Un test que hay que actualizar a mano cada vez que se añade una tarea no protege nada.

**Categoría:** `Correctness`

---

### [LOW] `go vet` no incluye los archivos de test

**ID:** AUD-031

**Ubicación:**
`go.mod`, `ci` (ausente)

**Problema:**
`go vet ./...` pasa sin analizar los `_test.go`; hace falta `-tests=true` explícito. No hay CI: la validación depende de que alguien la ejecute a mano.

**Impacto:**
Los errores en código de test no se detectan hasta que se ejecutan, y no hay barrera automática que impida consolidar un commit con la suite en rojo.

**Evidencia:**
`go vet ./...` sin salida; `go vet -tests=true ./...` también limpio hoy, así que el riesgo es preventivo.

**Recomendación:**
Añadir un objetivo de test y `go vet -tests=true` a un script en `scripts/`, y ejecutarlo en pre-commit.

**Justificación:**
El proyecto ya tiene `scripts/`; es el lugar natural y evita que la verificación dependa de la memoria.

**Categoría:** `Maintainability`

---

### [LOW] `nuevoError` puede devolver un centinela nil

**ID:** AUD-032

**Ubicación:**
`internal/exec/errors.go:41`

**Problema:**
El `switch` inicializa `sentinel` solo para códigos conocidos; un código desconocido deja el centinela en su valor cero, y `Unwrap()` devuelve `nil`, con lo que `errors.Is` falla en silencio. El mismo patrón se repite en `tools/errors.go:54`, `fileops/errors.go:43`, `queue/errors.go:42` y `flow/errors.go:36`.

**Impacto:**
Un código de error mal escrito no se detecta en compilación ni en test: produce un error que no se puede identificar por tipo. En el motor de flujo eso significa que un fallo de Ollama y uno de permisos son indistinguibles.

**Evidencia:**
Cinco copias del mismo patrón de tabla de centinelas, cada una con su propia lista.

**Recomendación:**
Convertir la lista en un `map[string]error` inicializado en el `init`, y devolver un error de "código desconocido" en vez de `nil`. Eliminar de paso las cinco copias en favor de un generador compartido (ver §8).

**Justificación:**
Un mapa elimina la clase de bug por construcción y reduce cinco implementaciones a una.

**Categoría:** `Correctness`

---

### [LOW] Ocho sitios envuelven con `%v` y pierden la cadena de errores

**ID:** AUD-033

**Ubicación:**
`internal/flow/engine.go:150`

**Problema:**
`fmt.Errorf("...: %v", err)` sobre un error ya tipado. Sin `%w` no se puede usar `errors.Is`/`errors.As` sobre la causa. Ocurre en `flow/engine.go:150,156`, `store/errors.go:80`, `store/db.go:80,114,135,142`, `docs/frontmatter.go:113,134,143`.

**Impacto:**
La cadena de causas se pierde justo donde más importa: `engine.go:150,156` descarta `cErr` y `aErr` en `ErrEtapaFallida`, que es el error de la etapa que falló.

**Evidencia:**
`fmt.Errorf("%w: %v", ErrEtapaFallida, cErr)` — envuelve el centinela y descarta la causa.

**Recomendación:**
Usar `%w` en los ocho sitios, o doble `%w` (Go 1.20+) donde se quiera conservar centinela y causa. Añadir un test de `errors.Is` por cada familia.

**Justificación:**
Convierte ocho errores que hoy solo son texto en errores consultables, que es la base del modelo de centinelas que el proyecto ya eligió.

**Categoría:** `Correctness`

---

### [LOW] `BuscarArchivos` acumula coincidencias sin límite de resultados

**ID:** AUD-034

**Ubicación:**
`internal/fileops/list.go:47`

**Problema:**
El recorrido del árbol acumula coincidencias de nombre sin tope. El contenido sí está limitado a 200 (`list.go:25`).

**Impacto:**
Un patrón amplio materializa una lista de longitud no acotada, que va al contexto del modelo. Se contradice a sí mismo: limita el contenido pero no los nombres.

**Evidencia:**
`MaxCoincidencias` existe como constante pero el bucle de nombres no la aplica.

**Recomendación:**
Aplicar `MaxCoincidencias` también al recorrido de nombres, y marcar el resultado como truncado en la respuesta.

**Justificación:**
La constante ya existe; es aplicarla donde corresponde.

**Categoría:** `Performance`

---

### [LOW] Archivos fuente citados: 8 de 138 sin citar en la documentación

**ID:** AUD-035

**Ubicación:**
`ai/docs/`

**Problema:**
130 de 138 archivos fuente aparecen citados por nombre en `ai/`. Los 8 sin cita: `context/nodo.go`, `fileops/ops.go`, `flow/catalogo.go`, `flow/comando.go`, `flow/flujo.go`, `store/repos.go`, `store/util.go`, `store/verify.go`.

**Impacto:**
Bajo por sí mismo. Ninguno contiene lógica de negocio sin documentar; los peor cubiertos son de soporte o reconstrucción.

**Evidencia:**
Contraste entre los 138 archivos del proyecto y las citas en `ai/`.

**Recomendación:**
Ninguna acción requerida. Se registra para que el 94,2% de la cifra sea auditable.

**Categoría:** `Maintainability`

---

### [LOW] `DOCS-VALIDATION-REPORT` desactualizado: le faltan 6 documentos y 7 commits

**ID:** AUD-036

**Ubicación:**
`ai/DOCS-VALIDATION-REPORT.md:1`

**Problema:**
El reporte declara fecha `2026-09-25T12:00:00Z` y "45 documentos analizados (17 specs + 12 backend + 12 database + 4 frontend)". Hoy hay 51 archivos `.md` en `ai/docs/`: faltan 2 specs, 2 backend y 2 más. 7 commits y 28 archivos de `ai/docs/` han cambiado desde entonces, incluidos `PROJECT.md`, `BACKEND.md`, `DOMAIN.md`, `TOOLS.md` y 13 specs. El `.json` tampoco tiene fecha parseable.

**Impacto:**
El informe da una confianza caducada. Los 3 WARNING que registró (W01, W02, W03) llevan 7 commits sin re-verificarse.

**Evidencia:**
Fecha del reporte `2026-09-25` contra el commit base `6bcf8ae` de `2026-09-28`.

**Recomendación:**
Re-ejecutar la validación y sobrescribir el reporte, con la fecha en el `.json` en un campo parseable.

**Justificación:**
Un informe de validación desactualizado es peor que ninguno: se cita como garantía de coherencia cuando ya no la comprueba.

**Categoría:** `Maintainability`

## 5. Observaciones

### [INFO] God files en la capa de composición y en la TUI

**ID:** AUD-037

**Ubicación:**
`internal/tui/app.go` (1.315 LOC), `arranque.go` (1.173 LOC)

**Observación:**
Dos archivos concentran responsabilidades heterogéneas. `app.go` acumula estado raíz, `Update`, input, modales y aprobaciones. `arranque.go` se define como composition root pero contiene adaptadores reales: cliente HTTP de internet, carga de modelos y goroutine de cola. `internal/tui/*` aporta 9 de los 15 archivos mayores del proyecto.

El mayor coste no es el tamaño en sí (el proyecto es pequeño) sino que ambos archivos son los puntos donde un cambio toca más cosas: cualquier modificación de input, modal o arranque pasa por el mismo fichero.

**Recomendación:**
Extraer por preocupación —repositorio de aprobaciones, cliente HTTP, carga de modelos, estado de vista— hacia interfaces que ya existen en `internal/`. Prioridad baja: los límites entre paquetes son correctos, el problema es interno a dos archivos.

**Categoría:** `Architecture`

---

### [INFO] El tablero de progreso es optimistic-only

**ID:** AUD-038

**Ubicación:**
`ai/tasks/backend/MAIN-TASKS.md:7`, `ai/tasks/frontend/MAIN-TASKS.md:7`

**Observación:**
Las 57 tareas están marcadas "completada" sin excepción. El estado real es 71,4% de los criterios implementados. La discrepancia no es de detalle: T-B004 ("escritura del TODO") y T-B010 ("encadenamiento de ciclos") están marcadas completadas y son precisamente dos de las funciones sin llamador productivo (`flow.GenerarTODO`, `session.Rederrivar`).

**Recomendación:**
Introducir un estado intermedio y exigir un criterio verificable para cerrar una tarea —por ejemplo, que la función tenga un llamador de producción, no solo un test—. Añadir a `internal/task` un test de integración que falle si una tarea marcada "completada" tiene un símbolo muerto asociado.

**Justificación:**
Convertiría el tablero en una fuente de verdad verificable por máquina en lugar de una declaración.

**Categoría:** `Architecture`

---

### [INFO] La documentación cubre la arquitectura, no la API

**ID:** AUD-039

**Ubicación:**
`ai/docs/` (51 documentos, 7.441 LOC)

**Observación:**
La documentación es sólida a nivel conceptual —cobertura del 100% en módulos, tablas, herramientas y enums— y floja a nivel referencial: 21,5% de los símbolos exportados. La lectura correcta es que el proyecto documenta *qué hace cada capa* y no *qué símbolos existen*.

No es un defecto de estilo: [[PROJECT]] define un sistema orientado a que la documentación sea la fuente de verdad para los agentes, y ese objetivo se cumple para el diseño. El hueco está donde un agente necesitaría consultar la firma exacta de un tipo, que es la documentación de código conventional que aporta el IDE.

**Recomendación:**
Mantener el enfoque actual. Solo cerrar los dos huecos estructurales: el contrato de `task` y `docs` (7% y 12%, los dos paquetes que manipulan las fuentes de verdad en disco) y los 12 códigos de error pendientes.

**Categoría:** `Maintainability`

---

### [INFO] SPEC-SKILLS está aprobada con 0% de implementación

**ID:** AUD-040

**Ubicación:**
`ai/docs/specs/SPEC-SKILLS.md`, `internal/agent/model.go:54`

**Observación:**
La spec aparece en el índice de [[PROJECT]] entre las specs aprobadas, pero ninguno de sus 6 criterios tiene implementación. El campo `Skills` existe en el modelo del agente y se fija a `[]` en `arranque.go:1091`, lo que da la apariencia de unagger soporte parcial.

**Recomendación:**
Marcar la spec como "no implementada / fuera de alcance de la v1" en su frontmatter, o implementar la carga de skills. Mantener una spec aprobada sin implementación contamin cualquier cálculo de cobertura que se haga en el futuro.

**Categoría:** `Architecture`

---

### [INFO] SPEC-CICLO-PLANIFICACION es la brecha funcional más grande (14%)

**ID:** AUD-041

**Ubicación:**
`ai/docs/specs/SPEC-CICLO-PLANIFICACION.md`, `internal/flow/plan.go:25`

**Observación:**
14% de implementación, la más baja de las specs con contenido funcional (SPEC-SKILLS es 0% pero no tiene implementación de base). Los tres criterios no implementados son los que definen el ciclo: que `/planificar` produzca un TODO real, que `build` reciba la propuesta de `plan`, y que las etapas usen agentes distintos.

Es coherente con AUD-010: el motor de flujo existe y funciona como mecanismo, pero el ciclo documentado en la spec no está cableado. El motor es la infraestructura; el ciclo es el producto.

**Recomendación:**
Tratar el cierre de SPEC-CICLO-PLANIFICACION como el objetivo de desarrollo de la siguiente iteración. Los tres criterios están en AUD-010, AUD-029 y AUD-035.

**Categoría:** `Architecture`

---

## 6. Problemas que requieren investigación

### [INVESTIGATE] ¿El `go test` en lista blanca es decisión o descuido?

**ID:** AUD-042

**Ubicación:**
`internal/exec/whitelist.go:31`

**Motivo:**
`go test` está en la lista blanca junto a `build` y `vet`. Compile y ejecuta el código de test del proyecto, lo que —combinado con AUD-001/AUD-002— implica que un `go test` aprobado sin restricción de ruta ejecuta binarios arbitrarios. No está claro si el equipo lo considera una operación de solo lectura.

**Qué verificar:**
Si el diseño de `go test` compila y ejecuta los tests contra el repositorio actual (código del usuario) o contra código generado. En el primer caso el riesgo es real; en el segundo es solo un vector de escape de la lista blanca.

**Recomendación:**
Si ejecuta código del proyecto, considerar moverlo a la lista que requiere aprobación, o restringirlo a un patrón de comando concreto. Si es decisión deliberada, documentarla en [[backend/DECISIONS]] junto a las demás decisiones de la lista blanca.

**Categoría:** `Investigation`

---

### [INVESTIGATE] ¿La reescritura de `EscribirArchivo` es reachable en el flujo normal?

**ID:** AUD-043

**Ubicación:**
`internal/fileops/write.go:86`

**Motivo:**
`EscribirArchivo` distingue entre archivo existente y nuevo, pero el agente `build` podría usar una ruta distinta. Si la herramienta solo se usa para archivos nuevos, el riesgo de perder contenido es menor de lo que sugiere el hallazgo.

**Qué verificar:**
Qué herramienta usa el agente para editar archivos existentes: ¿`editar_archivo` llama a `EscribirArchivo` o a `EditarArchivo`? Si la edición usa un mecanismo distinto, el hallazgo se aplica solo a ese caso.

**Recomendación:**
Aclarar el flujo antes de priorizar. Si `EditarArchivo` tiene su propia implementación, auditar esa por separado.

**Categoría:** `Investigation`

---

## 7. Código muerto

> Registrado solo con evidencia de búsqueda sobre el módulo completo. No se marca nada que pueda tener uso dinámico, reflexión o imports condicionales.

| Ubicación | Elemento | Evidencia (sin referencias encontradas) | Acción sugerida | Confianza |
|---|---|---|---|---|
| `internal/task/model.go:151` | `CamposDevueltos` | `rg -w` → 1 aparición (la declaración) | Confirmar; eliminar si no es API reservada | Alta |
| `internal/store/messages.go:136` | `RazonamientoPersistido` | `rg -w` → 1 aparición | Confirmar; eliminar si no es API reservada | Alta |
| `internal/store/migrate.go:95` | `ErrMigrationFailed` | `rg -w` → 1 aparición | Confirmar; eliminar si no es API reservada | Alta |
| `internal/docs/errors.go:53` | `ErrDocNotFound` | `rg -w` → 1 aparición | Confirmar; eliminar si no es API reservada | Alta |
| `internal/store/messages.go:100` | `UltimoMensaje` | `rg -w` → 1 aparición, sin test | Eliminar | Alta |
| `arranque.go:618` | `esperarResolucion` | `rg -w` → 1 aparición | Eliminar | Alta |
| `internal/context/select.go:96` | `OrdenarSeleccion` | `rg -w` → 1 aparición | Eliminar | Alta |
| `internal/docs/query.go:36` | `Directas` | `rg -w` → 1 aparición | Eliminar | Alta |
| `internal/exec/run.go:92` | `AvisoGarantia` | `rg -w` → 1 aparición | Relacionar con `AvisoNoCabe` (AUD-012) o eliminar | Alta |
| `internal/ollama/fifo.go:106` | `SetNotificadorGlobal` | `rg -w` → 1 aparición | Eliminar o cablear | Alta |
| `internal/store/util.go:42` | `parseISO` | `rg -w` → 1 aparición | Eliminar o usar en el parseo de fechas de la BD | Alta |
| `internal/store/util.go:62` | `strNull` | `rg -w` → 1 aparición | Eliminar o usar al escribir NULL | Alta |
| `internal/tui/keymap.go:432` | `deshabilitar` | `rg -w` → 1 aparición | Eliminar | Alta |
| `internal/agent/model.go:54` | `Skills` | Único uso: escritura a `[]` en `arranque.go:1091` | Ver AUD-014: implementar o retirar de la v1 | Media |

**Distinción importante (ver AUD-022).** La tabla anterior es código muerto real: símbolos sin ninguna referencia. **No** incluye el conjunto mayor y más importante de símbolos *usados solo por tests* — unas 25 funciones con cobertura completa y cero llamadores de producción (`flow/todo.go:36`, `ollama/profile.go:43,107,121`, `agent/handoff.go:55,71,76,97`, `agent/catalog.go:69,81`, `docs/graph.go:148-211`, `flow/comando.go:57`, `ollama/client.go:51,185`, `ollama/fifo.go:91,98`, `exec/run.go:88`, entre otros). Eso no es código muerto sino **cableado ausente**, y se corrige conectando, no borrando. Es la diferencia entre los dos patrones de la auditoría.

---

## 8. Duplicación

| Ubicación(s) | Lógica duplicada | Impacto | Recomendación | Alcance afectado |
|---|---|---|---|---|
| `internal/exec/whitelist.go:82`, `internal/tools/permission.go:67` | Helper `contiene`: comprobación de pertenencia en una lista de cadenas | Bajo | `slices.Contains` (Go ≥ 1.21) | 2 paquetes |
| `internal/exec/errors.go:41`, `internal/tools/errors.go:54`, `internal/fileops/errors.go:43`, `internal/queue/errors.go:42`, `internal/flow/errors.go:36` | `nuevoError`: switch código→centinela con `default` que deja el centinela sin asignar (AUD-018) | Medio | Tabla `map[string]error` en un paquete común | 5 paquetes |
| `internal/exec/run.go:218`, `internal/fileops/boundary.go:49` | Resolución de ruta dentro de la carpeta del proyecto: `exec` no resuelve symlinks, `fileops` sí | Medio | Extraer a un validador compartido (AUD-017) | Seguridad |
| `internal/tui/adjuntos.go:57`, `internal/fileops/approval.go:24` | Comprobación de que una ruta es utilizable: `tui` no comprueba nada, `fileops` comprueba frontera y tamaño | Medio | Que los adjuntos usen `fileops.Resolver` (AUD-008) | Seguridad |
| `internal/agent/loop.go:110`, `internal/session/contexto.go:55` | Poda de contexto: `session` compacta por presupuesto, `agent` no poda nada | Medio | Reutilizar la compactación existente (AUD-016) | Performance |

**Nota:** no se fuerza abstracción sobre duplicaciones triviales. Las cinco se listan porque o bien corrigen un defecto (las tres últimas) o bien eliminan una fragilidad compartida (las dos primeras), no solo porque repitan líneas.

---

## 9. Arquitectura

### Observaciones

**Responsabilidades.** Los límites entre los 13 paquetes de `internal/` son correctos y se sostienen. `store` es el único que escribe en SQLite, `tools` decide el permiso y `fileops`/`exec` lo aplican, `session` gestiona el ciclo de vida y `tui` no toca la base de datos directamente. Esa separación es lo mejor de la arquitectura del proyecto y [[backend/DECISIONS]] la razona explícitamente en cinco filas distintas.

**Acoplamiento y cohesión.** El acoplamiento entre paquetes es bajo: `fileops`, `tools` y `session` no se importan entre sí más de lo necesario, y la mayoría de la comunicación va por canales o por el bus de eventos de `session`. La cohesión interna es alta en los paquetes pequeños (`queue`, `context`, `exec`) y baja en los dos grandes (`tui/app.go`, `arranque.go`), que es donde aparecen todos los hallazgos de mantenibilidad.

**El problema está dentro de dos archivos, no entre paquetes.** `arranque.go` se describe en `internal/tui/wire.go` y en su propio comentario como composition root, pero contiene un cliente HTTP con su política de destino (AUD-009), un cargador de modelos con su N+1 de HTTP (AUD-013) y una goroutine de cola que descarta errores (AUD-018). `tui/app.go` acumula seis responsabilidades de presentación.

**El patrón dominante es «función correcta sin cableado».** No es un defecto de diseño: las funciones están bien escritas y probadas. Es un defecto de proceso, y se describe en AUD-022.

**Separación de capas.** Se respeta sin erosión en el sentido dirección (las capas bajas no importan a las altas: `store` no conoce a `session`, `fileops` no conoce a `tui`). La erosión es de *tamaño*, no de dirección.

### Recomendaciones

1. **Extraer por preocupación** desde `app.go` y `arranque.go` hacia interfaces ya existentes en `internal/` (AUD-020). No urgente: los límites aguantan.
2. **Cambiar el criterio de completado** de las tareas (AUD-021). Es la recomendación de mayor impacto de toda la auditoría y la más barata: una línea de criterio.
3. **Cablear las 6 capacidades desconectadas** antes de añadir otras nuevas (AUD-022).

---

## 10. Seguridad

| ID | Problema | Ubicación | Impacto | Evidencia | Recomendación | Categoría |
|---|---|---|---|---|---|---|
| AUD-001 | Lista blanca por `filepath.Base`: rutas absolutas arbitrarias sin aprobación | `internal/exec/whitelist.go:42` | **CRÍTICO** — ejecución arbitraria | PoC 4/4 vectores | Resolver ruta canónica con `EvalSymlinks` | `Security` |
| AUD-002 | Flags de subcomando nunca inspeccionados (`-exec`, `--output`, `--ext-diff`) | `internal/exec/whitelist.go:57` | **CRÍTICO** — ejecución arbitraria | PoC 5/6 vectores | Allowlist de flags, no de subcomandos | `Security` |
| AUD-003 | Landlock degrada en silencio y no restringe lectura | `internal/exec/landlock_linux.go:41` | Alto | `fmt.Fprintln` y continúa | Propagar error; añadir `READ_FILE`/`READ_DIR` | `Security` |
| AUD-004 | `.gitignore` con contenido inválido: BD de conversaciones versionada | `.gitignore:1` | Alto | `git ls-files` → `.localcli/state.db` | Sustituir por reglas reales; purgar historia | `Security` |
| AUD-008 | Adjuntos leen cualquier ruta sin frontera ni tope de tamaño | `internal/tui/adjuntos.go:57` | Alto | `os.ReadFile` sin validar | `fileops.Resolver` + `LimiteLecturaBytes` | `Security` |
| AUD-009 | Herramienta de internet acepta `http://` y no valida destino | `arranque.go:806` | Alto | Antepone `https://` solo si falta `://` | Forzar https; cliente con redirecciones revalidadas | `Security` |
| AUD-017 | `resolverCarpeta` no resuelve symlinks | `internal/exec/run.go:218` | Bajo | Contraste con `fileops/boundary.go:49` | Validador de ruta compartido | `Security` |
| AUD-025 | `go test` en lista blanca: efecto sobre código del proyecto sin decidir | `internal/exec/whitelist.go:31` | Por decidir | `listaBlanca` incluye `test` | Decidir y documentar | `Investigation` |

**Áreas de seguridad sin incidencia.** Verificadas y limpias:

- **Inyección SQL**: no hay concatenación de valores en consultas; se usan parámetros `?` en todo `store` (`internal/store/messages.go:39-42`). Ningún `SELECT *` en producción.
- **Secretos**: sin `.env`, `.pem`, `.key` ni `id_rsa` en ficheros versionados; sin valores de `api_key`, `secret` o `token` en código.
- **Validación de argumentos**: `tools.Decodificar` usa `DisallowUnknownFields` y rechaza contenido residual (`internal/tools/validate.go:90-100`).
- **Modelo de permisos**: la garantía de que `plan` no escribe es **estructural** (`tools/permission.go:52-64` deriva el catálogo de los permisos declarados en el JSON del agente), no una comprobación en tiempo de ejecución. Es la forma correcta.
- **Concurrencia**: `go test -race` pasa en los 14 paquetes.


## 11. Rendimiento

| ID | Problema | Ubicación | Impacto | Evidencia | Recomendación | Categoría |
|---|---|---|---|---|---|---|
| AUD-016 | Contexto del modelo crece sin poda entre pasadas del agente | `internal/agent/loop.go:110` | Medio | `leer_archivo` puede aportar 2 MiB por llamada; 3 pasadas | Truncar resultados de herramientas; reutilizar compactación de `session/contexto.go` | `Performance` |
| AUD-027 | `BuscarArchivos` acumula coincidencias sin límite de resultados | `internal/fileops/list.go:47` | Bajo | Contenido limitado a 200 (`list.go:25`), nombres sin tope | Aplicar `MaxCoincidencias` al recorrido de nombres | `Performance` |
| AUD-013 | Un INSERT y una transacción por documento en la auditoría de contexto | `internal/context/audit.go:33-55` | Medio | Bucle de `db.Exec`, cada uno su propia transacción | Una transacción con `PrepareStatement` y lotes | `Performance` |
| AUD-014 | N+1 de HTTP al cargar la lista de modelos | `arranque.go:309-334` | Medio | Una llamada `/api/show` por cada modelo de `/api/tags` | Pedir solo campos necesarios en una llamada, o cargar en segundo plano | `Performance` |

**Nota de contexto.** Estos cuatro hallazgos no tienen impacto demostrado en el uso real: el proyecto es un binario local para proyectos personales, no un servidor. Se listan porque el primero (AUD-016) puede causing fallos observables con modelos locales (error de contexto de Ollama) y los otros tres son trabajo acotado con un coste de ejecución desproporcionado para la escala. No se han medido perfiles de ejecución; los impactos son inferidos del código, no medidos.

**Sin incidencia de rendimiento verificada:**

- **Índices**: los de `internal/store/schema.sql` coinciden con [[database/01-schema/INDEXES]].
- **Consultas**: todas parametrizadas; sin `SELECT *`; sin bucles de consulta dentro de bucles de datos detectados.
- **Concurrencia de E/S**: la carga de modelos usa `semaphore` + `WaitGroup` correctamente (`arranque.go:319-334`); la captura de salida de comandos está protegida por mutex (`internal/exec/output.go:20-50`).

---

## 12. Mantenibilidad

| ID | Problema | Ubicación | Impacto | Evidencia | Recomendación | Categoría |
|---|---|---|---|---|---|---|
| AUD-010 | El relevo `plan` → `build` no existe | `internal/flow/engine.go:147` | Medio | El motor pasa el mismo objetivo a cada etapa; `documentacion` usa `AgentePlan` | Propagar el resultado; `documentacion` con `AgenteBuild` | `Architecture` |
| AUD-011 | `Nodo.Limite` nunca se fija: por defecto el contexto no recorta | `internal/context/model.go:119` | Medio | `arranque.go:944-949` construye el nodo sin `Limite`; `trim.go:22-28` con `limite<=0` devuelve todo | Derivarlo del modelo elegido; 0 = consultar, no "sin límite" | `Correctness` |
| AUD-012 | Perfil de hardware escrito pero nunca conectado | `internal/ollama/profile.go:43` | Medio | `DetectarHardware`, `AvisoNoCabe`, `ContextoLimitadoTokens` sin llamador | Conectar en el arranque y en la selección de modelo | `Architecture` |
| AUD-013 | Pausar y reanudar la cola no son alcanzables desde la interfaz | `internal/session/pause.go:111` | Medio | `AccionPausar` con binding `{}`; sin `AccionReanudar` | Exponer en el puerto y asignar tecla, o retirar de la spec | `Architecture` |
| AUD-021 | El tablero de progreso marca 57/57 tareas completadas sin ser verificable | `ai/tasks/backend/MAIN-TASKS.md:7` | Medio | 25 criterios sin implementar; 1 test en rojo | Estado intermedio + criterio de aceptación verificable | `Architecture` |
| AUD-015 | Test en rojo: el conteo de tareas de frontend está desactualizado | `internal/task/task_test.go:225` | Bajo | `33 elementos ... queremos 32` | Contar filas `\| T-F` directamente del archivo | `Correctness` |
| AUD-016 | `go vet` no incluye los archivos de test; no hay CI | `go.mod`, `ci` (ausente) | Bajo | `go vet ./...` vs. `go vet -tests=true ./...` | `go test ./...` + `go vet -tests=true` en `scripts/` | `Maintainability` |
| AUD-018 | Cinco tablas de centinelas con default que deja el centinela sin asignar | `internal/exec/errors.go:41` y 4 más | Bajo | `Unwrap()` puede devolver `nil` | Tabla `map[string]error` compartida | `Correctness` |
| AUD-019 | 8 sitios envuelven con `%v` y pierden la cadena de errores | `internal/flow/engine.go:150` y 7 más | Bajo | `fmt.Errorf("%w: %v", ...)` descarta la causa | `%w` en los 8 sitios | `Correctness` |
| AUD-020 | God files en TUI y composition root | `internal/tui/app.go` (1.315), `arranque.go` (1.173) | Bajo | 9 de los 15 archivos mayores en `tui` | Extraer por preocupación hacia interfaces existentes | `Architecture` |
| AUD-021 | `panic` en producción al construir el keymap | `internal/tui/keymap.go:256` | Bajo | Un `keys.json` corrupto tumba la TUI | Log + vuelta a `KeymapPorDefecto()` | `Maintainability` |
| AUD-024 | Cifras mágicas en puntos de cálculo | `internal/exec/run.go:60` y 3 más | Bajo | 120 s y 10 KB sin constante nombrada | Constantes con nombre, derivadas de DECISIONS | `Style` |
| AUD-023 | La reescritura de la línea de contexto puede no escribir | `internal/tui/adjuntos.go:57` | Bajo | `continue` silencioso si falla la lectura; sin tope de tamaño | Avisar en línea de estado; aplicar `LimiteLecturaBytes` | `Correctness` |
| AUD-025 | 12 códigos de error del backend no documentados | `ai/docs/backend/05-quality/ERRORS.md` | Bajo | 28/40 códigos documentados | Documentar los 12 (todos de fronteras de seguridad) | `Maintainability` |
| AUD-026 | `DOCS-VALIDATION-REPORT` desactualizado: 6 documentos y 7 commits | `ai/DOCS-VALIDATION-REPORT.md:1` | Bajo | Fecha `2026-09-25` vs. base `2026-09-28` | Re-ejecutar la validación | `Maintainability` |

**Cobertura de tests.** 524 `Test*` en 15.036 LOC de test, ratio global 0.82 (test/prod). Reparto: `tui` 1.05, `session` 0.90, `agent` 0.92, `docs` 0.84, `flow` 0.75, `context` 0.73, `queue` 0.69, `store` 0.64, `tools` 0.64, `task` 0.61, `ollama` 0.59, `exec` 0.37, `fileops` 0.30.

Los dos ratios más bajos son `fileops` (0.30) y `exec` (0.37) — **los dos módulos con garantía de escritura sobre el sistema de ficheros y con ejecución de procesos**. Es la inversión de esfuerzo correcta y merece atención: el resto del proyecto está por encima del promedio y estos dos están por debajo.

## 13. Recomendaciones priorizadas

Ordenadas por severidad e impacto técnico, no por facilidad de corrección. Cada entrada remite al hallazgo con su ID canónico.

### Cerrar primero — los dos CRITICAL

1. `[CRITICAL]` **AUD-001** — Lista blanca por `filepath.Base`: una ruta absoluta ejecuta un binario arbitrario sin aprobación. Resolver por ruta canónica (`filepath.EvalSymlinks` + comparación de prefijo con la raíz permitida), no por nombre base.
2. `[CRITICAL]` **AUD-002** — Flags de subcomando sin inspeccionar: `-exec`, `-toolexec`, `--output`, `--ext-diff` evaden la lista blanca. Rechazar todo argv que contenga flags con valor fuera de la tabla blanca.

Ambos comparten raíz: la lista blanca no es una lista de rutas sino de nombres, y no valida la segunda mitad de `argv`. Corregir solo uno deja el otro abierto. ~2,5 h en total, y son la primera prioridad del proyecto.

### Siguiente — seguridad e integridad de datos

3. `[HIGH]` **AUD-003** — Landlock degrada en silencio y no restringe lectura. Distinguir «no disponible» de «degradado» en el log y declarar explícitamente la pérdida de garantía de escritura.
4. `[HIGH]` **AUD-004** — `.gitignore` inválido: `.localcli/state.db` y el binario `localcli` versionados. Escribir un `.gitignore` real y decidir si el repositorio llegó a publicarse; si es así, purgar el historial. ~1 h.
5. `[HIGH]` **AUD-005** — Escritura no atómica: un fallo a mitad deja el archivo vacío. Escribir a temporal en la misma carpeta y `os.Rename`.
6. `[HIGH]` **AUD-006** — `EscribirArchivo` no revierte si falla el registro en el historial. Revertir el archivo si el segundo paso falla, o invertir el orden de las operaciones.
7. `[HIGH]` **AUD-007** — Cierre de sesión sin esperar a la goroutine en vuelo. Canal de cierre + `sync.WaitGroup` en `Cerrar`.
8. `[HIGH]` **AUD-008** — Adjuntos de imagen: lectura arbitraria de ficheros. Reutilizar `fileops.Resolver` (que ya valida la frontera) y un tope de tamaño antes de leer.
9. `[HIGH]` **AUD-009** — Herramienta de internet: HTTP sin cifrar y sin validar destino. Anteponer `https://` y rechazar rangos privados y de loopback.

### El cambio de mayor impacto — proceso

10. `[MEDIUM]` **AUD-029** — Cambiar el criterio de «completada» de las tareas: exigir llamador productivo, no solo test verde. Es la recomendación que desbloquea la visibilidad del proyecto entero.
11. `[MEDIUM]` **AUD-010** — Relevo `plan` → `build` inexistente y etapa `documentacion` con `AgentePlan`. Cablear el resultado de una etapa a la siguiente en `flow/engine.go:147`.
12. `[MEDIUM]` **AUD-012** — Perfil de hardware escrito pero nunca conectado. Llamar a `DetectarHardware` en el arranque y clasificar con `Adaptador.Modelos()`.
13. `[MEDIUM]` **AUD-011** — `Nodo.Limite` nunca se fija: por defecto el contexto no recorta. Derivarlo de las capacidades del modelo en el arranque.
14. `[MEDIUM]` **AUD-013** — Pausar/Reanudar inalcanzables desde la interfaz. Exponerlos en el puerto, o retirar los criterios de la spec.
15. `[MEDIUM]` **AUD-014** — `Skills` es un campo muerto: SPEC-SKILLS al 0%. Decidir el destino de la spec: implementar o marcar fuera de alcance.
16. `[LOW]` **AUD-018** — Los errores de la goroutine de cola se descartan. Propagarlos a la sesión.
17. `[LOW]` **AUD-022** — La reescritura de la línea de contexto puede no escribir. Eliminar el `continue` que salta el render.
18. `[LOW]` **AUD-030** — Test en rojo por el conteo de tareas de frontend. Derivar el número del archivo en vez de hardcodearlo.

### Ahora — bajo coste, riesgo de seguridad residual

19. `[LOW]` **AUD-015** — `gofmt` sin `-w` imprime el contenido de ficheros alcanzables. Sustituir por `gofmt -l` en la lista blanca.
20. `[LOW]` **AUD-016** — `panic` en producción al construir el keymap. Degradar a `KeymapPorDefecto()`.
21. `[LOW]` **AUD-017** — El cliente HTTP de Ollama declara un timeout que no se aplica. Pasarlo a `http.Client{Timeout: ...}`.
22. `[LOW]` **AUD-019** — `os.Exit` anula el cierre diferido de la base de datos. Cerrar explícitamente antes de salir.
23. `[LOW]` **AUD-026** — `resolverCarpeta` no resuelve symlinks, a diferencia de `fileops`. Unificar en un único helper.
24. `[LOW]` **AUD-027** — Un comando aprobado nunca puede escribir en el proyecto, y el diálogo no lo dice. Declararlo en el diálogo o retirar la promesa de escritura.
25. `[LOW]` **AUD-028** — La frontera de permiso y la de aprobación se solapan sin quedar declaradas. Documentar cuál manda.
26. `[LOW]` **AUD-033** — Ocho sitios envuelven con `%v` y pierden la cadena de errores. Usar `%w` y añadir tests de `errors.Is`.
27. `[LOW]` **AUD-032** — `nuevoError` puede devolver un centinela nil. Asignar el centinela en el `default` de las cinco tablas.
28. `[LOW]` **AUD-034** — `BuscarArchivos` acumula coincidencias sin límite. Aplicar la constante `MaxCoincidencias` ya existente al recorrido de nombres.
29. `[LOW]` **AUD-025** — Dos helpers duplicados y cinco tablas de centinelas repetidas. Consolidar en un solo punto.
30. `[LOW]` **AUD-021** — Números mágicos en los puntos de cálculo. Extraer constantes.
31. `[LOW]` **AUD-020** — `parseISO` y `strNull` no tienen ningún uso. Eliminar o cubrir con test.
32. `[LOW]` **AUD-031** — `go vet` no incluye los archivos de test en el flujo documentado. Ejecutar `go vet -tests=true ./...` en el guion de verificación.
33. `[LOW]` **AUD-024** y **AUD-036** — Documentación desfasada: el informe de validación no cubre 6 documentos nuevos y la lista de errores está incompleta. Re-ejecutar la validación.

### Cierre documental

34. `[INFO]` **AUD-037** — Extraer `app.go` y `arranque.go` por preocupación hacia interfaces ya existentes.
35. `[INFO]` **AUD-039** — Documentar el contrato de `task` y `docs`: son los dos paquetes con peor cobertura (7% y 12%) y gobiernan las fuentes de verdad en disco.
36. `[INFO]` **AUD-035** — Cerrar los 8 archivos fuente sin citar en la documentación.
37. `[INFO]` **AUD-040** y **AUD-041** — Marcar SPEC-SKILLS (0%) y SPEC-CICLO-PLANIFICACION (14%) con su estado real en el índice de specs.
38. `[INFO]` **AUD-038** — El tablero de progreso solo puede estar «optimistic-only» mientras el criterio de completado no cambie (ver recomendación 10).

### Decisiones pendientes

39. `[INVESTIGATE]` **AUD-042** — Decidir si `go test` en la lista blanca es decisión deliberada o descuido: compila y ejecuta el código de test del proyecto.
40. `[INVESTIGATE]` **AUD-043** — Aclarar si la reescritura de `EscribirArchivo` es alcanzable en el flujo de edición normal.

### Orden de ataque

```
10. [MEDIUM] AUD-029  criterio de completado
  └── 11. AUD-010  relevo plan → build
        └── 12. AUD-012  perfil de hardware
              └── 13. AUD-011  límite de contexto
                    └── 38. AUD-038  visibilidad del tablero
```

Ese camino no requiere refactor. Son seis puntos de conexión entre funciones que ya existen, probadas y documentadas. Es la observación de mayor valor de toda la auditoría: **el proyecto tiene ~25 funciones terminadas y desconectadas, y ~6 puntos de conexión pendientes.**

---
## 14. Conclusión técnica

**Calidad del código.** El proyecto está en mejor estado del que sugiere su tablero. `go vet`, `gofmt`, `go build` y `go test -race` pasan limpios. No hay inyección SQL, ni secretos en código, ni carreras de datos, ni `TODO` pendientes. Hay 524 tests para 18.357 LOC de producción y una separación de responsabilidades entre paquetes que se sostiene: `store` es el único que escribe en SQLite, `tools` decide permisos y `fileops`/`exec` los aplican. La zona realmente limpia es la base de datos: sin concatenación de consultas, índices alineados con la documentación y traducción de errores en un solo punto.

**Riesgos más relevantes.** Dos de criticidad inmediata. El primero es la **terminal**: la lista blanca compara `filepath.Base(argv[0])` y nunca inspecciona los flags, así que un modelo puede ejecutar cualquier binario sin aprobación por dos vías independientes —una ruta absoluta, o `go test -exec <payload>`. El PoC confirma 9 vectores. Esto deja sin efecto la decisión central del proyecto en [[backend/DECISIONS]] y es lo primero que hay que cerrar. El segundo es la **fuga de datos**: `.gitignore` contiene literalmente `(empty)`, así que el historial del repositorio incluye 6 sesiones de conversación, 17 mensajes, 8 razonamientos y un binario de 17 MB.

**Problema de fondo.** Ninguno de los dos es un descuido de código. El terminal se implementó con una heurística plausible —«el programa se identifica por su nombre»— que resulta insuficiente bajo un adversario que controla los argumentos, y el `.gitignore` fue sobrescrito por un placeholder. Son fallos de verificación: el PoC de la lista blanca no existía, y nadie comprobó que `.gitignore` ignorara algo.

**El hallazgo de mayor alcance, sin embargo, es de proceso.** `ai/tasks/*/MAIN-TASKS.md` declara 57/57 tareas completadas cuando la cobertura real de la documentación es del 71,4%. La brecha no son 25 funcionalidades ausentes: son funciones **escritas, probadas y desconectadas** —`flow.GenerarTODO`, `session.Rederrivar`, `ollama.DetectarHardware`, `ollama.AvisoNoCabe`, `ollama.ContextoLimitadoTokens`, `agent.Skills`, `Nodo.Limite`— más dos criterios de proceso rotos: el relevo `plan` → `build` no existe (`flow/engine.go:147` pasa el mismo objetivo a cada etapa) y la etapa de documentación usa un agente sin herramientas de escritura. El ciclo agentic que el producto vende no funciona de extremo a extremo.

El patrón es consistente y diagnóstico: **la infraestructura profunda está sólida y el pegamento de proceso falta.** Todo lo que es una regla local con invariantes —permisos, fronteras, Landlock, teclado, historial, sesiones, tools— está implementado y bien cubierto por sus tests. Todo lo que exige **orquestación entre componentes** —generar el TODO, escribir la documentación con `build`, pasar el resultado de una etapa a la siguiente, re-derivar la cola, detectar ciclos, medir la cabida de modelos, recortar el contexto por defecto— son funciones escritas, probadas y **sin llamador productivo**.

Es la firma clásica de «suite verde, producto incompleto»: cada test pasa porque verifica la unidad, y ninguna suite verifica el cableado. De ahí que 24/24 tareas backend figuren como completadas con SPEC-CICLO-PLANIFICACION al 14% y SPEC-SKILLS al 0%.

**Áreas que requieren atención, por orden de prioridad.**

1. **Terminal** — cerrar los dos CRITICAL antes de cualquier otra cosa. ~2,5 h.
2. **Repositorio** — `.gitignore` y purga de historia. ~1 h. Decidir si el repositorio se publicó.
3. **Proceso de tareas** — cambiar el criterio de completado y reconectar las ~25 funciones sueltas. El cambio de mayor impacto y menor coste de toda la auditoría.
4. **Fronteras de datos** — escritura atómica, reversión ante fallo de registro, drenaje de goroutines.
5. **Límites de la capa TUI** — `app.go` y `arranque.go` son el siguiente objetivo estructural, no urgente porque los límites entre paquetes aguantan.

**Quick wins frente a mejoras estructurales.**

- *Quick wins:* `.gitignore` (1 h, elimina un HIGH de privacidad); lista blanca por ruta canónica y por flags (2,5 h, elimina dos CRITICAL de ejecución); test de regresión del contador en `task_test.go` (10 min, devuelve la suite a verde).
- *Estructurales:* cambiar el criterio de completado y cablear el motor de etapas; documentar el contrato de `task` y `docs`, que son los dos paquetes con peor cobertura (7% y 12%) y los que gobiernan las fuentes de verdad en disco.

**Nota.** Esta conclusión se basa únicamente en los hallazgos documentados en las secciones 2 a 12. No introduce hallazgos nuevos.

---

## 15. Anexo A — Cobertura de la documentación (implementación)

**Método.** Unidad de medida: los **248 criterios de aceptación** (`- [ ]`) de los 19 archivos `ai/docs/specs/SPEC-*.md`, cruzados con las 210 reglas de negocio declaradas en los documentos por capa. Cada estado se verificó buscando el símbolo, la tabla, el keybinding o el evento correspondiente en el código Go. `NO VERIFICABLE` = el criterio depende únicamente de la salida del modelo y no es comprobable por inspección de código. **Los estados «completada» de `ai/tasks/*/MAIN-TASKS.md` se ignoraron como evidencia**, por si acaso.

| Spec | Criterios | Impl. | Parcial | No impl. | No verif. | % impl. |
|---|---:|---:|---:|---:|---:|---:|
| SPEC-SKILLS | 6 | 0 | 0 | 6 | 0 | **0%** |
| SPEC-CICLO-PLANIFICACION | 14 | 2 | 4 | 3 | 5 | **14%** |
| SPEC-CICLO-TRABAJO | 17 | 8 | 5 | 4 | 0 | **47%** |
| SPEC-COLA-TAREAS | 18 | 9 | 5 | 4 | 0 | **50%** |
| SPEC-AGENTE-PERSONALIZADO | 6 | 3 | 3 | 0 | 0 | **50%** |
| SPEC-FLUJO-PERSONALIZADO | 6 | 4 | 1 | 1 | 0 | **67%** |
| SPEC-AGENTE-BASE | 16 | 11 | 4 | 0 | 1 | **69%** |
| SPEC-OLLAMA-PERFIL | 13 | 9 | 2 | 2 | 0 | **69%** |
| SPEC-NODO-CONTEXTO | 7 | 5 | 1 | 1 | 0 | **71%** |
| SPEC-MOTOR-FLUJOS | 11 | 8 | 2 | 1 | 0 | **73%** |
| SPEC-RESOLVER | 8 | 6 | 0 | 0 | 2 | **75%** |
| SPEC-ARCHIVOS | 10 | 8 | 1 | 1 | 0 | **80%** |
| SPEC-TOOLS | 17 | 14 | 3 | 0 | 0 | **82%** |
| SPEC-INTERFAZ-ATAJOS | 11 | 9 | 2 | 0 | 0 | **82%** |
| SPEC-INTERFAZ | 49 | 42 | 5 | 2 | 0 | **86%** |
| SPEC-HISTORIAL-CONVERSACION | 6 | 6 | 0 | 0 | 0 | **100%** |
| SPEC-KEYBINDS | 16 | 16 | 0 | 0 | 0 | **100%** |
| SPEC-PANEL-CONTEXTO | 6 | 6 | 0 | 0 | 0 | **100%** |
| SPEC-SESIONES | 11 | 11 | 0 | 0 | 0 | **100%** |
| **TOTAL** | **248** | **177** | **38** | **25** | **8** | **71,4%** |

**Desglose de la columna «no implementados» (25 criterios).**

- **SPEC-SKILLS (6/6)** — sin carga desde `.localcli/agents/*.json`, sin ámbito de sesión, sin resolución de markdown, sin skills por defecto. `agent/model.go:54` declara el campo y `arranque.go:1091` lo fija a `[]`. → AUD-014.
- **SPEC-CICLO-PLANIFICACION (3)** — el flujo `/planificar` tiene 8 etapas **todas con `tools.AgentePlan`** (`flow/plan.go:25-34`), y su propio comentario admite que no escribe. `flow.GenerarTODO` no tiene llamador productivo. El orden de capas BD→backend→frontend, el máximo de 5 preguntas por tanda, la regla «un archivo = una responsabilidad», el prefijo numérico y el idioma forzado no tienen ninguna comprobación en código. → AUD-010, AUD-029.
- **SPEC-CICLO-TRABAJO (4)** — etapa `documentacion` con `Agente: tools.AgentePlan` (`flow/work.go:37`) pese a su comentario «con `build` escribiendo tras aprobar». La entrada `verificar` existe como acción (`task/model.go:38`) pero no hay comando ni flujo (`flow/work.go:28` solo genera `crear|actualizar|eliminar`). La coincidencia de acción entre elemento y tarea principal, y la búsqueda de referencias antes de eliminar, no tienen código.
- **SPEC-COLA-TAREAS (4)** — reanudación inalcanzable (`session/pause.go:111` existe; `arranque.go:544-546` solo expone `Pausar`/`Cancelar`; `AccionPausar` tiene binding `{}` en `keymap.go:183`). `Rederrivar` no se llama desde arranque ni TUI. «Una petición ordenada crea su propio TODO» solo produce un mensaje de sistema con una sugerencia de `/ejecutar` (`session/run.go:130-136`). «El estado de la cola se ve desde cualquier sesión» no se cumple: el panel muestra la sesión activa. → AUD-013.
- **SPEC-NODO-CONTEXTO (1)** — `Nodo.Limite` nunca se fija en producción; `LimiteDeEntorno()` devuelve 0 sin `LOCALCLI_CONTEXT_LIMIT`; `Recortar` con `limite <= 0` devuelve todo sin recortar. → AUD-011.
- **SPEC-OLLAMA-PERFIL (2)** — `Adaptador.Modelos()` solo consulta `/api/show`, nunca llama a `ClasificarModelos`. `AvisoNoCabe` y `ContextoLimitadoTokens` no tienen llamadores productivos. `DetectarHardware` nunca se invoca: el arranque usa `PerfilPorDefecto()` fijo. → AUD-012.
- **Otros (5)** — SPEC-MOTOR-FLUJOS: sin opciones de reintentar/saltar/cancelar (`flow/engine.go:154-157`). SPEC-FLUJO-PERSONALIZADO: `Flujo.Validar()` no detecta ciclos ni IDs repetidos. SPEC-ARCHIVOS: «operación fuera de la carpeta: además de la aprobación exige la explicación» implementado como rechazo duro, sin vía de permiso. SPEC-AGENTE-PERSONALIZADO: crear/editar/borrar agente solo editando JSON a mano. SPEC-INTERFAZ: reasignación de atajos inexistente en la vista.

**Contradicciones entre documentación y código.**

1. **El relevo `plan` → `build` no existe**, y contradice [[specs/SPEC-AGENTE-BASE]], [[backend/DECISIONS]] y el comentario de `flow/work.go:9-12`. → AUD-010.
2. **`MAIN-TASKS.md` declara 24/24 tareas backend «completada»**, incluidas T-B004 («escritura del TODO») y T-B010 («encadenamiento de ciclos»), precisamente las que están desconectadas. → AUD-021.
3. **`internal/agent/loader.go:40-43` vs. `arranque.go:1052-1081`**: el loader dice que un JSON roto detiene la carga; el arranque lo ignora. El comportamiento real cumple la spec, la documentación interna no.
4. **Estandarización entre specs**: SPEC-INTERFAZ-ATAJOS describe `Ctrl+A` como «enfoca»; SPEC-KEYBINDS y el código lo modelan como acción con estado alternable (`tui/app.go:484-494`).
5. **`ollama/profile.go:7-9`** interpreta «no lo carga en silencio» como «lo carga igual con aviso». La lectura literal de la spec es más fuerte, y la implementación de hecho no avisa nunca.

---

## 16. Anexo B — Cobertura del código en la documentación

Símbolos exportados (`func`/`type`/`const`/`var` con nombre en mayúscula) contados sobre el módulo, excluidos del recuento los valores de la lista **NOISE** (boilerplate Go: `String`, `Error`, `Update`, `Render`, `View`…). Se marcan como documentados los símbolos cuyo nombre aparece en al menos un `.md` de `ai/docs/`.

| Paquete | LOC prod | Símbolos exp. | Denominativos | En `ai/docs` | **% docs** | En `ai/docs`+`ai/tasks` | Ratio test/prod |
|---|---:|---:|---:|---:|---:|---:|---:|
| `tui` | 5.299 | 185 | 180 | 37 | **21%** | 30% | 1.05 |
| `store` | 1.618 | 68 | 67 | 12 | **18%** | 19% | 0.64 |
| `tools` | 953 | 65 | 62 | 8 | **13%** | 18% | 0.64 |
| `session` | 1.515 | 55 | 55 | 15 | **27%** | 53% | 0.90 |
| `flow` | 1.180 | 49 | 46 | 14 | **30%** | 37% | 0.75 |
| `ollama` | 1.087 | 46 | 43 | 8 | **19%** | 26% | 0.59 |
| `task` | 1.258 | 43 | 43 | 3 | **7%** | **7%** | 0.61 |
| `agent` | 706 | 37 | 36 | 9 | **25%** | 33% | 0.92 |
| `docs` | 778 | 35 | 33 | 4 | **12%** | **12%** | 0.84 |
| `fileops` | 888 | 29 | 27 | 4 | **15%** | 15% | **0.30** |
| `context` | 655 | 29 | 29 | 6 | **21%** | 31% | 0.73 |
| `queue` | 496 | 27 | 25 | 9 | **36%** | 44% | 0.69 |
| `exec` | 707 | 18 | 15 | 4 | **27%** | 33% | **0.37** |
| raíz (`arranque.go`, `main.go`) | 1.217 | 27 | 27 | 15 | **56%** | 78% | 0.06 |
| **TOTAL** | **18.357** | **713** | **688** | **148** | **21,5%** | **29,7%** | **0,82** |

**Métricas derivadas.**

| Medición | Valor |
|---|---:|
| Símbolo → `ai/docs/` (match laxo, por substring) | 20,9% |
| Símbolo → `ai/docs/` (match estricto, entre backticks) | 13,3% |
| LOC ponderado por cobertura de paquete | 22,8% |
| Módulos de `internal/` documentados | 13/13 = 100% |
| Archivos fuente citados por nombre en `ai/` | 130/138 = 94,2% |
| Códigos de error `E_*` documentados | 28/40 = 70% |
| Símbolos exportados muertos | 4 = 0,56% |

**Interpretación.** El punto de partida de la cifra del 21,5% es que un símbolo se cuenta como documentado si su **nombre** aparece en un `.md` de `ai/docs/`, con coincidencia por substring. Con el criterio más estricto (el símbolo entre backticks, es decir citado explícitamente) la cifra baja al 13,3%. La diferencia entre ambos no es discrepancia, sino ruido de coincidencia parcial: no todos los 148 son citas deliberadas.

**La asimetría es el dato relevante.** El resultado no es uniforme. Tres paquetes superan el 40% (`raíz` 56%, `queue` 36%, `flow` 30%) y dos caen por debajo del 15% (`task` 7%, `docs` 12%). La correlación es interpretable:

- `queue` y `flow` tienen specs densas ([[specs/SPEC-COLA-TAREAS]] y [[specs/SPEC-MOTOR-FLUJOS]]) que nombran sus tipos, estados y reglas al describirlos.
- `task` y `docs` no tienen equivalente. [[backend/DECISIONS]] los describe en una frase ("los archivos del TODO son la fuente de verdad") y [[backend/BACKEND]] §3 los lista como módulos del grafo. No hay documento que capture su estructura.

Esos dos paquetes son los que **manipulan las fuentes de verdad en disco** —el TODO de progreso y los documentos de arquitectura—, y son precisamente los que presentan el peor riesgo de sincronización entre código y documentación. Es coherente con el diagnóstico de la sección 14: donde la documentación de comportamiento es densa, la cobertura de implementación es alta; donde no la hay, ambos lados adolecen.

**El punto ciego de la documentación de símbolos no es `tui`** (21%, el segundo más alto en LOC) sino el par `task`+`docs` (7% y 12%, 138 símbolos denominativos sin documentar entre ambos). Cerrar ese hueco es más valioso que extender la documentación de `tui`, que ya tiene [[frontend/FRONTEND]] más [[frontend/02-interfaces/INTERFACES]] más [[frontend/05-quality/TESTING]] más [[specs/SPEC-INTERFAZ]] más [[specs/SPEC-INTERFAZ-ATAJOS]] más [[specs/SPEC-KEYBINDS]] cubriéndola desde seis ángulos.

### Archivos fuente sin citar por nombre en `ai/`

8 de 138 archivos (94,2% de cobertura). Ninguno contiene lógica no documentada:

| Archivo | Por qué no aparece |
|---|---|
| `internal/context/nodo.go` | Cubierto por [[specs/SPEC-NODO-CONTEXTO]], que lo describe por comportamiento |
| `internal/fileops/ops.go` | Cubierto por [[specs/SPEC-ARCHIVOS]] y [[backend/03-security/SECURITY]] |
| `internal/flow/catalogo.go` | Cubierto por [[specs/SPEC-MOTOR-FLUJOS]] |
| `internal/flow/comando.go` | Cubierto por [[specs/SPEC-MOTOR-FLUJOS]] |
| `internal/flow/flujo.go` | Cubierto por [[specs/SPEC-FLUJO-PERSONALIZADO]] |
| `internal/store/repos.go` | Cubierto por [[database/01-schema/RELATIONSHIPS]] |
| `internal/store/util.go` | Utilidades internas de conversion; [[database/01-schema/ENUMS]] cubre los formatos |
| `internal/store/verify.go` | Verificación de invariantes; [[database/01-schema/CONSTRAINTS]] |

Que no se citen no significa que estén sin documentar: la documentación cubre el comportamiento y esos ocho archivos son la implementación de ese comportamiento. Se registran para que la cifra del 94,2% sea auditable contra una lista concreta.

---

## Referencias

Documentos de `ai/docs/` citados en este informe:

- [[PROJECT]] — stack aprobado y decisiones globales del proyecto.
- [[backend/BACKEND]] — mapa de la capa backend, estructura de `internal/`.
- [[backend/DECISIONS]] — decisiones técnicas; contradicha por el código en AUD-001, AUD-004 y AUD-013.
- [[backend/03-security/SECURITY]] — modelo de permisos, fronteras de ruta y lista blanca de terminal.
- [[backend/05-quality/ERRORS]] — códigos de error `E_*`; 12 de 40 sin documentar (AUD-025).
- [[backend/04-infrastructure/EVENTS]] — eventos que produce el motor, consumidos por la TUI.
- [[database/01-schema/SCHEMA]] / [[database/01-schema/TABLES]] / [[database/01-schema/INDEXES]] / [[database/01-schema/CONSTRAINTS]] / [[database/01-schema/ENUMS]] / [[database/01-schema/RELATIONSHIPS]] — capa de datos; los índices verificados como coincidentes con `store/schema.sql`.
- [[database/03-operations/QUERIES]] / [[database/03-operations/MIGRATIONS]] — consultas y versionado del esquema.
- [[frontend/FRONTEND]] / [[frontend/01-domain/DOMAIN]] / [[frontend/02-interfaces/INTERFACES]] / [[frontend/05-quality/TESTING]] — capa TUI.
- [[specs/SPEC-TOOLS]] / [[specs/SPEC-ARCHIVOS]] / [[specs/SPEC-AGENTE-BASE]] / [[specs/SPEC-AGENTE-PERSONALIZADO]] — agentes y herramientas.
- [[specs/SPEC-CICLO-TRABAJO]] / [[specs/SPEC-CICLO-PLANIFICACION]] / [[specs/SPEC-MOTOR-FLUJOS]] / [[specs/SPEC-COLA-TAREAS]] / [[specs/SPEC-FLUJO-PERSONALIZADO]] — motor de flujo y cola; SPEC-CICLO-PLANIFICACION es la peor cobertura (14%, AUD-024).
- [[specs/SPEC-OLLAMA-PERFIL]] / [[specs/SPEC-NODO-CONTEXTO]] / [[specs/SPEC-PANEL-CONTEXTO]] / [[specs/SPEC-HISTORIAL-CONVERSACION]] / [[specs/SPEC-SESIONES]] / [[specs/SPEC-RESOLVER]] — modelo, contexto y sesiones.
- [[specs/SPEC-INTERFAZ]] / [[specs/SPEC-INTERFAZ-ATAJOS]] / [[specs/SPEC-KEYBINDS]] — superficie de TUI y teclado; 100% y 82% de implementación.
- [[specs/SPEC-SKILLS]] — **0% de implementación** (AUD-014, AUD-023).

Fuentes de proceso auditadas: `ai/tasks/backend/MAIN-TASKS.md`, `ai/tasks/frontend/MAIN-TASKS.md`, `ai/DOCS-VALIDATION-REPORT.md`.

Método y reproducibilidad: análisis estático (`go vet ./...`, `gofmt -l .`, `go build ./...`, `go test ./...`, `go test -race ./...`), conteo de símbolos por patrón, búsqueda de referencias con `rg -w` sobre el módulo completo, y un PoC de función pura ejecutado sobre una copia de `internal/exec/whitelist.go` en `/tmp/opencode/poc/`. Ningún archivo fuente del proyecto fue modificado.
