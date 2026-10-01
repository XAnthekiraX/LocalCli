> T-B034 — el agente deja de ser un JSON: pasa a ser una carpeta con `agent.yaml` (identidad y permisos) y `prompt.md` (instrucciones), con permisos de tres categorías (`read`, `write`, `edit`), sin `skills` y sin `mode`.
> Acción de esta descomposición: `actualizar`.

## Referencias

- [[specs/SPEC-AGENTE-BASE]] — §Dónde se definen: la carpeta, el `agent.yaml`, el `prompt.md` y la tabla de los tres permisos; §Los dos agentes: qué es exactamente el `plan` incluido.
- [[specs/SPEC-SKILLS]] — capacidad no implementada: el contrato ya no tiene dónde declarar una skill.
- [[specs/SPEC-TOOLS]] — §El reparto: de dónde sale el catálogo de un agente; §La lista de pasos: por qué la lista de la sesión cae en `read`.
- [[specs/SPEC-AGENTE-PERSONALIZADO]] — §Flujo principal: crear la carpeta es «definir un agente propio».
- [[backend/01-domain/BUSINESS_RULES]] — §Agentes: las reglas del contrato, con `default` solo `deny` y los campos desconocidos rechazados.
- [[backend/02-interfaces/TOOLS]] — §1 y §2: la columna de permiso de cada herramienta y la comparación que sostiene la garantía.
- [[backend/02-interfaces/dto/TOOLS-DTO]] — §Estructuras: el `Permiso` que el catálogo expone.
- [[backend/04-infrastructure/CONFIGURATION]] — §Agentes del proyecto: la ruta y el ejemplo que tiene que existir en disco.
- [[backend/03-security/SECURITY]] — §1: de dónde sale lo que el agente puede pedir.

## Contexto

El agente es hoy un único archivo JSON de cinco campos (`nombre`, `descripcion`, `prompt`, `permisos`, `skills`), con el prompt y la configuración mezclados en la misma estructura:

```go
// internal/agent/model.go
var CamposDelAgente = []string{"nombre", "descripcion", "prompt", "permisos", "skills"}

type Agente struct {
    Nombre        string    `json:"nombre"`
    Descripcion   string    `json:"descripcion"`
    Prompt        string    `json:"prompt"`
    Permisos      []Permiso `json:"permisos"`
    Skills        []string  `json:"skills"`
    Herramientas  []string  `json:"-"`
}
```

Y los permisos se declaran por una lista de parejas acción × efecto, sobre cinco acciones:

```go
// internal/tools/catalog.go
type Permiso struct { Accion string; Efecto string } // en agent/model.go
AccionLeer, AccionEditar, AccionEjecutar, AccionInternet, AccionTareas
```

De ahí salen tres cosas que la documentación nueva ya no reconoce:

1. **El prompt viaja en el archivo de configuración.** El contrato de [[specs/SPEC-AGENTE-BASE]] los separa: `agent.yaml` gobierna el harness y `prompt.md` gobierna al modelo, y cada uno se lee de su archivo. El prompt literal no cambia; cambia dónde vive y quién lo carga.
2. **Las acciones no son la clasificación que se documentó.** La documentación ([[specs/SPEC-TOOLS]] §El reparto, [[backend/02-interfaces/TOOLS]] §2) ya dice que son tres permisos —`read`, `write`, `edit`— y que la lista de pasos de la sesión es `read`. En el código, la lista de pasos tiene su propia acción (`tareas`) y la escritura va entera a `editar`, así que `editar_archivo` —edición parcial— no se distingue de `crear_archivo`. Con tres permisos la distinción se vuelve real: `edit` es solo parchear.
3. **`skills` sigue en el contrato.** Es el campo muerto de AUD-014: se declara, se puebla con `[]` en `arranque.go:1363` y no lo lee nadie. La documentación ya lo retiró del contrato y declara la capacidad no implementada; el código todavía lo acepta como campo válido.

Y hay un cuarto punto, de carga: `agenteBase` recorre `.localcli/agents/` buscando archivos `*.json` (`arranque.go:1335-1347`), mientras que el contrato nuevo es una carpeta por agente. Además, un `*.json` suelto —el formato antiguo— debe avisarse nombrándolo en vez de ignorarse en silencio, porque si no el usuario ve que su agente desapareció sin explicación.

El agente base no cambia de comportamiento: `plan` sigue sin ninguna herramienta de escritura y `build` sigue con el catálogo completo. Lo que cambia es de dónde sale esa lista.

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-B034-01 | actualizar | `agent`: el contrato YAML de `Agente` pasa a tres campos —`Nombre`, `Descripcion`, `Permissions`— y **deja de aceptar `Prompt` y `Skills`**. `Prompt` sigue existiendo en la estructura, pero como valor que rellena la carga leyendo `prompt.md`, no como campo decodificado; `Skills` desaparece | completada | `internal/agent/model.go` | `TestAgenteSinSkills` en verde |
| T-B034-02 | actualizar | `agent`: decodificar `agent.yaml` con YAML estricto —rechaza `prompt`, `herramientas`, `skills`, `mode`, `hereda_de` y cualquier otro campo desconocido, nombrando el campo; `permissions` es un **mapa** `string` → `string`, no una lista de parejas | completada | `internal/agent/model.go` | `TestYamlConCampoDesconocidoSeRechaza`, `TestPromptEnElYamlSeRechaza` en verde |
| T-B034-03 | actualizar | `tools`: los cinco `Accion` pasan a tres `Permiso` (`read`, `write`, `edit`); `Herramienta` declara su `Permiso` como dato del catálogo y `Accion()` —que lo deducía de categoría y modo— desaparece. `editar_archivo` es la única `edit`; crear, escribir, eliminar y carpetas son `write`; lectura, terminal, internet y las dos de lista de pasos son `read` | completada | `internal/tools/catalog.go` | `TestCadaHerramientaDeclaraSuPermiso` en verde |
| T-B034-04 | actualizar | `agent`: `HerramientasDe` deriva el catálogo comparando el `Permiso` de la herramienta contra el `permissions` del agente; `default` es opcional y **solo admite `deny`** (ausente = deny, `allow` se rechaza con el motivo); un permiso o un efecto fuera de vocabulario se rechaza; sin `permissions` el agente es de solo conversación, sin error | completada | `internal/agent/catalog.go` | `TestDefaultAllowSeRechaza`, `TestPermisoDesconocidoSeRechaza`, `TestAgenteSinPermissionsSoloConversa` en verde |
| T-B034-05 | actualizar | `agent`: `Cargar` lee una **carpeta**: `agent.yaml` y `prompt.md` son los dos obligatorios; el nombre del agente sale de `name`, no del nombre de la carpeta; el `prompt.md` se lee íntegro, sin recortar, y un archivo vacío hace que el agente no cargue | completada | `internal/agent/loader.go` | `TestCarpetaSinPromptNoCarga`, `TestElNombreVieneDeName` en verde |
| T-B034-06 | actualizar | `arranque`: `agenteBase` recorre **subcarpetas** de `.localcli/agents/` en vez de `*.json`; un `*.json` suelto —el formato antiguo— produce un aviso que **nombra el archivo** y dice cuál es el formato nuevo, y no se carga; el resto de la semántica no cambia: un agente inválido se ignora con aviso y `plan`/`build` caen al respaldo si faltan | completada | `arranque.go` | `TestAgenteBaseCargaCarpetas`, `TestUnJsonAntiguoSeAvisaNombrandolo` en verde |
| T-B034-07 | actualizar | Migrar los dos agentes base del repositorio a la carpeta nueva: `.localcli/agents/plan/agent.yaml` + `prompt.md` y `.localcli/agents/build/agent.yaml` + `prompt.md`, con el **prompt actual literalmente idéntico** y `plan` con `permissions: {default: deny, read: allow}`; borrar los dos `*.json` antiguos. Es la migración de T-B034-05, no una excepción al contrato | completada | `.localcli/agents/plan/`, `.localcli/agents/build/` | `TestElPlanDelRepoNoTieneHerramientasDeEscritura`, `TestElBuildDelRepoTieneElCatalogoCompleto` en verde |
| T-B034-08 | actualizar | `agent`: `agenteDeRespaldo` deja de rellenar `Skills` y su comentario deja de decir «escribe su `.localcli/agents`» sin más: dice qué dos archivos hay que crear | completada | `arranque.go` | El respaldo sigue siendo un agente de solo conversación |
| T-B034-09 | actualizar | tests: fixtures de `agent/testdata` al formato nuevo, más los casos del contrato que no existían —carpeta sin `prompt.md`, `default: allow`, campo desconocido, permisos que no conceden nada y `plan` sin escritura | completada | `internal/agent/*_test.go`, `arranque_test.go` | `go test ./internal/agent/ ./internal/tools/ . -count=1` en verde |

Dependencias: T-B034-03 antes de T-B034-04 (la comparación es contra el `Permiso` del catálogo). T-B034-01, -02 y -05 antes de T-B034-06. T-B034-07 depende de T-B034-05 y va con T-B034-06: si los archivos base no están migrados, el arranque no tiene agentes y los tests del contrato no significan nada.

## Fuera de alcance

- **No se implementan las skills.** El campo desaparece del contrato y de `Agente`; la capacidad entera sigue sin existir. Ver [[specs/SPEC-SKILLS]].
- **No se añade `mode`.** Sigue aplazado a propósito; cuando exista un modo con efecto real se añade como clave, y no rompe nada porque `default` es `deny`.
- **No se leen herramientas del YAML.** No hay `herramientas:`: el catálogo se deriva de `permissions`. Un archivo con ese campo se rechaza, no se ignora.
- **No cambian los flujos ni las herramientas del usuario.** Los flujos siguen siendo JSON en `.localcli/flows/` y las herramientas del usuario JSON en `.localcli/tools/`: el cambio es solo de los agentes.
- **No se toca la TUI.** `Adaptador.Agentes() []string` sigue exponiendo nombres; la TUI no sabe de dónde viene un agente.
- **No hay migración de base de datos.** Ni definiciones de agentes ni permisos se persisten en SQLite.
- **No hay carga en caliente.** Un agente editado se ve al reiniciar, como hasta ahora.
- **No se recortan los prompts.** `prompt.md` se inyecta íntegro; el recorte de contexto sigue siendo del nodo de contexto y no toca esto.
