---
title: LocalCli
tags: [proyecto, indice]
depende_de:
  - "[[IDEA]]"
relacionado:
  - "[[backend/DECISIONS]]"
---

# PROJECT — LocalCli

## Nombre del proyecto

LocalCli

## Descripción

Harness de terminal en Go que planifica y ejecuta desarrollo de software con un modelo local gestionado por el propio harness: descarga los modelos GGUF desde Hugging Face, los almacena y cataloga, y los sirve mediante `llama-server` (llama.cpp), entregando a cada etapa solo el contexto que necesita y pidiendo tu aprobación antes de aplicar cualquier cambio.

## Alcance funcional aprobado

Definido en 20 especificaciones funcionales bajo `ai/docs/specs/`.

### Núcleo (P0)

- [[specs/SPEC-AGENTE-BASE]] — los dos agentes incluidos y el relevo entre ellos.
- [[specs/SPEC-TOOLS]] — catálogo de herramientas y reparto por agente.
- [[specs/SPEC-ARCHIVOS]] — reglas de acceso y permiso sobre archivos y carpetas.
- [[specs/SPEC-MODELOS-LLAMACPP]] — gestión y ejecución de modelos con llama.cpp, descarga, catálogo y selección.
- [[specs/SPEC-NODO-CONTEXTO]] — selección y recorte del contexto por etapa.
- [[specs/SPEC-SESIONES]] — varias sesiones con contexto independiente.
- [[specs/SPEC-HISTORIAL-CONVERSACION]] — memoria del chat y compactación del historial.
- [[specs/SPEC-ORQUESTADOR-FLUJOS]] — orquestador de etapas y flujos oficiales.
- [[specs/SPEC-COLA-TAREAS]] — cola por capa que se ejecuta sola.
- [[specs/SPEC-CICLO-PLANIFICACION]] — ciclo de planificación desde cero.
- [[specs/SPEC-CICLO-TRABAJO]] — ciclo de trabajo con sus tres entradas.
- [[specs/SPEC-INTERFAZ]] — disposición de la pantalla y zonas.

### Importante (P1)

- [[specs/SPEC-PANEL-CONTEXTO]] — conteo de tokens y ocupación del contexto.
- [[specs/SPEC-RESOLVER]] — ciclo aparte para arreglar lo que está roto.

### Interfaz (P2)

- [[specs/SPEC-INTERFAZ-ATAJOS]] — atajos de teclado y panel de aprobaciones.
- [[specs/SPEC-VISTA-CODIGO]] — render del Markdown de la respuesta: bloques de código y botón de copiar.

### Personalización (P3)

- [[specs/SPEC-AGENTE-PERSONALIZADO]]
- [[specs/SPEC-FLUJO-PERSONALIZADO]]
- [[specs/SPEC-SKILLS]] — capacidad **no implementada**: no hay skills ni sitio donde declararlas.

## Stack tecnológico

| Área | Elección |
|---|---|
| Lenguaje | Go |
| Interfaz de terminal | Bubble Tea + Lip Gloss (charmbracelet) |
| Aislamiento de terminal | Landlock (mecanismo del kernel en Linux, sin privilegios) |
| Persistencia | SQLite con driver puro Go (`modernc.org/sqlite`), sin cgo |
| Modelo | Un solo proveedor, llama.cpp: LocalCli descarga los modelos GGUF desde Hugging Face, los almacena en `~/.config/localcli/models/`, los registra en su catálogo (`models.json`) y arranca `llama-server` con el elegido; la comunicación es HTTP sobre el protocolo OpenAI (`/v1`). La configuración global (`localcli.json`) guarda solo `proveedor` y la dirección base de la API |
| Distribución | Un único binario, sin runtime externo |

No hay Node, ni Python, ni gestor de procesos. La herramienta arranca sin terminal multiplexer, sin dependencias de sistema y sin modelos descargados: el binario es autosuficiente y descarga y gestiona los modelos él mismo.

## Arquitectura general

Un solo proceso. Los módulos se comunican por canales de Go, no por red.

```
cmd/localcli/      punto de entrada
internal/
  tui/             chat, panel de datos, selector de sesiones, aprobaciones
  session/         creación, cambio, memoria de conversación y ejecución en segundo plano de sesiones
  agent/           carga de agentes (los base sembrados en ~/.config/localcli/agents/ y los propios del proyecto en <proyecto>/.localcli/agents/; cada uno, una carpeta con agent.yaml y prompt.md)
  llm/             frontera del modelo: tipos, interfaz de motor, capacidades de tres estados y regla de la ventana
  llm/openai/      núcleo común: transporte HTTP y SSE contra /v1/chat/completions
  llm/llamacpp/    extensión nativa de llama-server: /props
  models/          gestión de modelos: descarga desde Hugging Face, catálogo (models.json), subdirectorios en models/ y selección
  llama/           ciclo de vida de llama-server: arranque y detención con el GGUF elegido, puerto y chequeo de disponibilidad
  context/         grafo de frontmatter, selección, recorte y auditoría
  flow/            orquestador de etapas y encadenamiento
  queue/           cola por capa y orden por dependencias
  task/            archivos de tarea
  tools/           catálogo y capa universal de ejecución
  fileops/         validación de rutas, acceso a archivos e historial de cambios
  exec/            terminal: confirmación selectiva y bloqueo estructural de escritura
  store/           persistencia SQLite
  docs/            carga de documentación y lectura de frontmatter
```

### Recorrido de una petición

1. `tui` recibe lo que escribes y lo envía a la sesión activa.
2. `session` responde la petición como chat; arranca un flujo cuando la línea es un comando explícito (`/planificar`, `/crear`, `/actualizar`, `/eliminar`, `/resolver` o `/ejecutar`), o cuando el agente lo pide con la herramienta `ejecutar_flujo`.
3. `context` arma el contexto de la etapa: lee el grafo de dependencias, filtra, deja que el modelo elija y recorta hasta el límite.
4. `agent` construye la llamada con el prompt de `plan` o de `build` y los esquemas de las herramientas de ese agente, sin escribir el catálogo en el prompt.
5. La frontera de motor (`llm`) recorta el turno para que quepa y lo envía por el núcleo común (`llm/openai`), que habla el protocolo OpenAI con `llama-server`; los puntos que ese núcleo no cubre salen por la extensión nativa (`props` de llama.cpp). Devuelve el token a token, junto con las peticiones de herramienta que el modelo formule.
6. `tui` muestra el razonamiento en vivo y después la respuesta.
7. Si el agente pide una herramienta, `tools` comprueba el permiso, valida los argumentos, ejecuta el handler, recorta la salida y emite el evento. Cada petición del modelo puede ir en una vuelta del bucle; hay un máximo de rondas.
8. El resultado de la etapa pasa a `flow`, que decide si sigue, si se detiene o si espera tu aprobación.
9. `store` persiste el estado; `queue` toma la siguiente tarea cuando toca.

Un turno de herramienta no se guarda en el historial: se resuelve en memoria y lo que persiste es el mensaje tuyo y la respuesta final del agente. Ver [[database/02-rules/DATA_FLOW]].

### Concurrencia

Una goroutine por sesión en ejecución, con el orquestador de etapas encima. Los canales de Go hacen de cola entre la TUI y el trabajo de fondo. SQLite en modo WAL para que la interfaz pueda leer mientras las sesiones de fondo escriben.

## Persistencia

Hay dos planos, y la separación es deliberada.

**Archivos: fuente de verdad.** La documentación del proyecto y los archivos de tarea se editan a mano y se versionan en git. Nada de eso se duplica en la base de datos.

**SQLite: estado de ejecución.** Solo lo que es efímero o lo que necesita consultas rápidas.

| Contenido | Plano |
|---|---|
| Documentación del proyecto | Archivos |
| Tareas (`NNN-task-<nombre>.md`) | Archivos |
| Sesiones y mensajes, incluido el razonamiento | SQLite |
| Estado y orden de la cola | Memoria, derivado de los archivos de tarea |
| Aprobaciones pendientes | SQLite |
| Registro de qué documentación entró y salió del contexto | SQLite |
| Historial de cambios aplicados | SQLite |

La cola se deriva de los archivos de tarea y se reconstruye al arrancar en memoria; no hay tabla de cola en SQLite. Si los dos divergen, manda el archivo.

**Configuración del harness: fuera del proyecto.** La carpeta `~/.config/localcli/` es la raíz de la máquina y concentra todo lo que no pertenece al proyecto: el mapa de teclas (`keys.json`), las preferencias (`config.json`), la configuración del proveedor (`localcli.json`), el catálogo de modelos (`models.json`), los modelos GGUF (`models/`), el registro del motor (`logs/llama-server.log`), la base de datos (`state.db`), los agentes base (`agents/`) y los flujos oficiales (`flows/`). Son globales del usuario, no del proyecto: no se versionan con el contenido del proyecto. Los agentes base y los flujos oficiales se embeben en el binario y se siembran en esa carpeta al ejecutar `localcli init`, sin sobrescribir lo que ya haya. El proyecto solo puede **añadir** agentes y flujos propios en `<proyecto>/.localcli/`, que se cargan de forma aditiva junto a los globales. Ver [[backend/04-infrastructure/CONFIGURATION]] y [[frontend/FRONTEND]].

**Catálogo de modelos: archivo global, no tabla.** Los modelos descargados viven en `~/.config/localcli/models/` (un GGUF original por modelo, dentro de su subdirectorio) y se registran en `~/.config/localcli/models.json`, no en SQLite. Son una característica de la máquina —qué modelos hay y dónde—, no del proyecto: guardarlos en la base ataría los archivos de la máquina a un repositorio. **El catálogo nace vacío y así se queda hasta que el usuario descarga un modelo: LocalCli se instala en una máquina sin modelos y eso es un estado válido**, no un error que haya que tapar con un modelo inventado. `localcli.json` no contiene el modelo ni los modelos descargados. Lo único que va a la base es el modelo de cada sesión, en `sessions.modelo`. Ver [[specs/SPEC-MODELOS-LLAMACPP]] y [[backend/DECISIONS]].

**Base de datos: un solo archivo global dentro de `~/.config/localcli/`.** No hay un archivo por proyecto. La base guarda las sesiones de todas las carpetas y cada una se ata a su proyecto por `sessions.project_id` (la ruta de la carpeta abierta): el aislamiento deja de ser físico y pasa a ser una consulta por proyecto. Ver [[database/DATABASE]] y [[database/01-schema/SCHEMA]].

## Integraciones

- **Inferencia (llama.cpp).** Un solo proveedor. LocalCli descarga modelos GGUF desde Hugging Face, los guarda en `~/.config/localcli/models/`, los registra en su catálogo y arranca `llama-server` con el modelo elegido, hablándole por HTTP bajo el protocolo OpenAI. Peticiones en streaming para poder mostrar el razonamiento mientras llega. No se usa Ollama ni se mantienen almacenes duplicados.
- **Búsqueda en internet.** Es la única integración que hace salir información de la máquina, y solo sale la consulta. El contenido que vuelve entra al mismo presupuesto de contexto que el resto y queda registrado.

## Seguridad

- **Confirmación de lo destructivo, lo de red y lo de fuera.** Borrar —un archivo, una carpeta—, sacar información a la red desde la terminal (`curl`, `wget`, `ssh`, `push`...) o tocar una ruta fuera de la carpeta del proyecto pide tu aprobación, también dentro de un flujo en curso. Crear, escribir entero un archivo vacío y editar dentro de la carpeta no interrumpen.
- **Bloqueo estructural de la terminal.** Landlock impide que un proceso hijo escriba en los archivos del proyecto, sin depender de revisar el texto del comando. En sistemas sin Landlock, la garantía es más débil y queda documentada como tal.
- **Frontera de rutas.** Fuera de la carpeta del proyecto hace falta permiso y además la explicación del agente.
- **Confirmación de los git destructivos y de la publicación.** `reset`, `clean`, `checkout` y `restore` piden aprobación; los commits los frena Landlock; `push` pide aprobación.
- **Historial de cambios.** Cada cambio aplicado guarda lo que había antes y lo que quedó, para poder revertir y auditar.

## Pruebas

- **Unitarias.** Grafo de dependencias y selección de contexto, orden de la cola por dependencias, comprobación de permisos, reparto de herramientas por agente, validación de rutas, generación de esquemas y recorte de la salida de una herramienta.
- **De integración.** Conexión y streaming con el motor, ciclo completo de una etapa con las herramientas de lectura, aplicación de un cambio con su aprobación, ciclo de cola, turno con varias herramientas encadenadas.
- **Del aislamiento.** Un caso que intente escribir en el proyecto a través de la terminal y debe fallar. Es la prueba que sostiene la garantía de Landlock.
- **De contexto.** Que el recorte reduzca de forma medible frente a leer el proyecto completo, y que lo entregado quepa en el límite del modelo.

## Despliegue

- Compilación cruzada a un binario único con `go build`.
- Requisitos en la máquina: **ningún modelo es requisito para instalar**. LocalCli arranca sin modelos en el catálogo; los descarga el propio harness desde Hugging Face y los sirve con `llama-server`. Hace falta el binario de `llama.cpp` disponible en el sistema y Landlock disponible si se quiere el aislamiento fuerte (Linux 5.13 o superior).
- Sin servicios adicionales, sin migraciones que aplicar a mano, sin configuración obligatoria.

## Decisiones técnicas

| Decisión | Por qué | Alternativa descartada |
|---|---|---|
| Go | Binario único, arranque inmediato, concurrencia con canales de Go y buen soporte para TUI | Node o Python, que arrastran runtime y peor ecosistema para CLI |
| Bubble Tea + Lip Gloss | Arquitectura orientada a eventos y streaming, que es justo lo que necesita el razonamiento en vivo | tview, con mejores widgets pero menos natural para texto que se actualiza en directo |
| Landlock | Garantía del kernel, sin privilegios y sin binarios externos | bubblewrap, igual de fuerte pero exige instalar un paquete; y el filtrado de texto, descartado porque `find -delete` o una redirección lo esquivan |
| SQLite con driver puro Go | Un archivo, sin servidor, y el binario sigue siendo portable porque no usa cgo | Postgres, innecesario para uso personal; un driver con cgo, que ataría la compilación a un toolchain de C |
| Archivos como fuente de verdad | La documentación se edita a mano, se versiona y se lee sin la herramienta | Guardarlo todo en SQLite, que haría la documentación inaccesible fuera del harness |
| El modelo lo elige el usuario | Es su máquina y su tradeoff entre calidad y velocidad | Fijar un modelo por defecto, que asumiría en su nombre qué Prioriza |
| Un solo proveedor: llama.cpp, gestionado por el harness | LocalCli controla de principio a fin la descarga, el almacenamiento y la ejecución del modelo, que se carga desde su GGUF original sin copias derivadas | Un registro multi-motor con Ollama, que reintroduce su almacén duplicado y una generalidad sin uso; hardcodear el modelo, que impide gestionarlo por catálogo |

## Límites conocidos

- **Las sesiones comparten servidor y, con él, el modelo.** Hay un solo `llama-server`, así que varias sesiones pueden estar abiertas a la vez, pero mientras una genera las demás esperan. Cambiar de modelo reinicia el servidor y desaloja el modelo anterior. Es una consecuencia del hardware, no un defecto del diseño.
- **Cambiar de modelo reinicia el servidor.** El harness arranca `llama-server` con el GGUF elegido y lo detiene; para pasar a otro modelo hay que reiniciarlo, y ese proceso lo gestiona LocalCli. Ver [[specs/SPEC-MODELOS-LLAMACPP]].
- **El aislamiento fuerte es de Linux.** Landlock solo existe ahí. En otros sistemas la terminal tiene una garantía más débil y la documentación lo dice.
- **Un 7B no cabe en 4 GB de VRAM.** Si eliges uno, el harness avisa y no lo carga en silencio; irá a RAM, que es más lento y aprieta los 16 GB.
- **La inferencia no se puede paralelizar** entre sesiones, por lo mismo.

## Convención del frontmatter

El formato de los enlaces entre documentos —wiki link
`[[carpeta/ARCHIVO]]` relativo a `ai/docs/`— está definido en `AGENTS.md` y no
se repite aquí. Lo que sigue es lo propio del frontmatter.

Cada documento de `ai/docs/` empieza con un bloque YAML que declara de qué
depende. Sin él, el grafo no sabe qué hay que leer para entender el documento.

```yaml
---
title: LocalCli — capa de datos
tags: [database, indice]
depende_de:
  - "[[PROJECT]]"
relacionado:
  - "[[backend/BACKEND]]"
---
```

- `title`: el nombre del documento. Una línea.
- `tags`: la capa (`proyecto`, `backend`, `database`, `frontend`, `specs`) y el
  tipo (`indice`, `arquitectura`, `esquema`, `reglas`, `dominio`, `interfaces`,
  `infraestructura`, `calidad`, `seguridad`, `decisiones`, `operaciones`,
  `requisito`, `origen`).
- `depende_de`: lo que hay que leer para entender o cambiar el documento.
- `relacionado`: a qué otros documentos toca, sin que este necesite de ellos.

**`depende_de` y `relacionado` no son sinónimos.** Un índice publica sus hijos
y no depende de ellos: `PROJECT.md` lista veinte specs, y declararlas haría
que cargar la raíz arrastrara el proyecto entero. La dirección real va al revés,
de la hoja a la raíz.

La distinción no es cosmética: el cierre transitivo de `depende_de` promedia
3,2 documentos de los 49 del proyecto, y el peor caso son 8. Tomando todas las
menciones del cuerpo como dependencia, el peor caso sube a 20. Esos números
están fijados en un test (`TestElCierreDeDependenciasNoArrastraElProyecto`)
porque un declarar-de-más no rompe el grafo: lo degrada en silencio.

En las specs, la sección `## Dependencias funcionales` del cuerpo es la que
manda: lo que ahí aparezca va en `depende_de`, y el frontmatter no puede
contradecirla. Un test lo comprueba.

Los enlaces que quedan en el cuerpo siguen valiendo, y el grafo los tiene en
cuenta, pero son menciones, no dependencias declaradas. Ver
[[backend/01-domain/DOMAIN]].

## Decisiones pendientes

Las decisiones de backend están centralizadas en [[backend/DECISIONS]]. Las globales que siguen abiertas:

- Formato definitivo del archivo de tarea (`NNN-task-<nombre>.md`) y su bloque de contexto.
- Los programas destructivos que exigen confirmación en la terminal (la lista cerrada está en `internal/exec/riesgo.go`), con su límite de tiempo y de salida.
- Parámetros de arranque de `llama-server` por defecto para el perfil recomendado.

## Referencias

- [[IDEA]]
