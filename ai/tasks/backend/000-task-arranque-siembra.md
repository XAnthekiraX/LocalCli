# 000-task-arranque-siembra.md

> T-B000 — Comandos de arranque y siembra de la configuración. **Primera tarea del backend.**
> Acción de esta descomposición: `crear`.

## Referencias

- [[backend/04-infrastructure/CONFIGURATION]] §1 El comando y sus subcomandos, §4 Rutas, §8 El catálogo de modelos
- [[specs/SPEC-MODELOS-LLAMACPP]] §2 Configuración global, §3 Almacenamiento de modelos, §5 Catálogo de modelos
- [[frontend/02-interfaces/INTERFACES]] — la TUI ya resuelve `~/.config/localcli/config.json` y `keys.json`
- [[specs/SPEC-AGENTE-BASE]] §Dónde se definen
- [[specs/SPEC-FLUJO-PERSONALIZADO]] §Supuestos
- [[database/03-operations/MIGRATIONS]] — la BD global nace en v7, vacía

## Contexto

El harness expone un comando con subcomandos. Sin argumentos arranca sobre la carpeta actual (el proyecto); `-h` lista los comandos y `-v` muestra la versión.

**La raíz global no se crea sola.** Si `~/.config/localcli/` no existe, `localcli` no continúa: escribe en consola «localcli no iniciada, ejecuta el comando `localcli init`» y termina. La raíz la crea `localcli init`, que siembra en ella lo que falte sin sobrescribir nada. `localcli create` crea la configuración **del proyecto** en la carpeta actual (`<cwd>/.localcli/`).

Lista de siembra de `localcli init` (todo en `~/.config/localcli/`):

- `agents/{plan,build,chat}/{agent.yaml,prompt.md}` — agentes base, embebidos en el binario.
- `flows/*.json` — los flujos base (oficiales), embebidos en el binario.
- `state.db` — la base de datos global, nacida ya en v7 con `projects` y `sessions.project_id`, vacía.
- `localcli.json` — la configuración del proveedor: `{ "proveedor": "llama.cpp", "api": "http://localhost:8080" }`.
- `models.json` — el catálogo de modelos, vacío: `{ "modelos": [], "seleccionado": null }`.
- `models/` — el directorio de modelos GGUF, vacío.
- `config.json` — preferencias del usuario con sus valores por defecto.
- `keys.json` — el mapa de teclas con sus atajos por defecto.

`localcli create` crea `<cwd>/.localcli/` con `agents/` y `flows/` para los agentes y flujos propios del proyecto.

## Tareas pequeñas

| ID         | Acción | Tarea                                                                                              | Estado    | Archivos                          | Verificación                                                                              |
| ---------- | ------ | -------------------------------------------------------------------------------------------------- | --------- | --------------------------------- | ----------------------------------------------------------------------------------------- |
| T-B000-01  | crear  | Resolver la raíz de la máquina (`~/.config/localcli/` vía `os.UserConfigDir`/`XDG_CONFIG_HOME`)      | completada | `internal/arranque/rutas.go`      | La raíz se resuelve igual que ya hace `internal/tui/config.go`                             |
| T-B000-02  | crear  | Parsear el comando: sin argumentos arranca; reconocer `-h`, `-v`, `init`, `create`                   | completada | `cmd/localcli/main.go`            | Cada forma invoca la acción correcta; un subcomando desconocido avisa                      |
| T-B000-03  | crear  | `localcli -v` imprime la versión                                                                    | completada | `cmd/localcli/main.go`            | Muestra la versión y sale con código 0                                                     |
| T-B000-04  | crear  | `localcli -h` imprime la lista de comandos disponibles                                              | completada | `cmd/localcli/main.go`            | Lista `init`, `create`, `-h`, `-v` y sale con código 0                                     |
| T-B000-05  | crear  | Arranque sin raíz global: mensaje «localcli no iniciada, ejecuta el comando `localcli init`» y salida | completada | `internal/arranque/rutas.go`      | Sin `~/.config/localcli/`, `localcli` no arranca la TUI y muestra el mensaje               |
| T-B000-06  | crear  | `localcli init`: crear la raíz y sus subdirectorios (`agents/`, `flows/`, `models/`)                 | completada | `internal/arranque/siembra.go`    | Tras `init` existe `~/.config/localcli/` con `agents/`, `flows/` y `models/`               |
| T-B000-07  | crear  | Sembrar los agentes base embebidos (`plan`, `build`, `chat`), solo si faltan                         | completada | `internal/arranque/datos/agents`  | Tras `init` los tres agentes están; modificar uno no se sobrescribe                        |
| T-B000-08  | crear  | Sembrar los flujos base embebidos, solo si faltan                                                    | completada | `internal/arranque/datos/flows`   | Tras `init` los flujos base están; editar uno no se sobrescribe                            |
| T-B000-09  | crear  | Crear `state.db` global y dejarla en v7 con `projects` + `sessions.project_id`, vacía                | completada | `internal/store`                  | Abrir la BD nueva la deja en `user_version` 7 con las tablas                               |
| T-B000-10  | crear  | Sembrar `config.json` y `keys.json` con los valores por defecto, solo si faltan                      | completada | `internal/arranque/siembra.go`    | Sin archivos previos quedan con sus valores por defecto                                    |
| T-B000-11  | crear  | Sembrar `localcli.json` con `proveedor` y `api` por defecto, solo si falta                           | completada | `internal/arranque/siembra.go`    | Sin archivo previo, nace con `{ "proveedor": "llama.cpp", "api": "http://localhost:8080" }` |
| T-B000-12  | crear  | Sembrar `models.json` vacío (`"modelos": [], "seleccionado": null`), solo si falta                    | completada | `internal/arranque/siembra.go`    | Sin catálogo previo, nace vacío; un catálogo con modelos se conserva                       |
| T-B000-13  | crear  | `localcli create`: crear `<cwd>/.localcli/` con `agents/` y `flows/` en la carpeta actual            | completada | `internal/arranque/proyecto.go`   | Tras `create`, `./.localcli/` existe con `agents/` y `flows/`                              |
| T-B000-14  | crear  | La siembra nunca sobrescribe y no toca nada fuera de la raíz ni del `.localcli/` del proyecto        | completada | `internal/arranque/siembra.go`    | Un `localcli.json` o un `models.json` editados a mano se conservan al re-ejecutar `init`    |
| T-B000-15  | crear  | Un fallo o una ausencia no rompe el arranque: lo que no está, no aparece; se avisa                   | completada | `internal/arranque`               | Simular carpeta sin permisos: `init` avisa y no deja la raíz a medio hacer                 |

**Dependencias:** `01`→`02`→`03`,`04`,`05`; `02`→`06`→`07`,`08`,`09`,`10`,`11`,`12`; `02`→`13`; `06`→`14`,`15`.
**Nota de migración:** las bases antiguas de `<proyecto>/.localcli/state.db` se abandonan; no hay importación. Ver [[database/03-operations/MIGRATIONS]].