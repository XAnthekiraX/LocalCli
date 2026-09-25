# PROJECT — LocalCli

## Nombre del proyecto

LocalCli

## Descripción

Harness de terminal en Go que planifica y ejecuta desarrollo de software con un modelo local de Ollama, entregando a cada etapa solo el contexto que necesita y pidiendo tu aprobación antes de aplicar cualquier cambio.

## Alcance funcional aprobado

Definido en 17 especificaciones funcionales bajo `ai/docs/specs/`.

### Núcleo (P0)

- [[specs/SPEC-AGENTE-BASE]] — los dos agentes incluidos y el relevo entre ellos.
- [[specs/SPEC-TOOLS]] — catálogo de herramientas y reparto por agente.
- [[specs/SPEC-ARCHIVOS]] — reglas de acceso y permiso sobre archivos y carpetas.
- [[specs/SPEC-OLLAMA-PERFIL]] — conexión a Ollama y perfil de hardware.
- [[specs/SPEC-NODO-CONTEXTO]] — selección y recorte del contexto por etapa.
- [[specs/SPEC-SESIONES]] — varias sesiones con contexto independiente.
- [[specs/SPEC-MOTOR-FLUJOS]] — motor de etapas y flujos oficiales.
- [[specs/SPEC-COLA-TAREAS]] — cola por capa que se ejecuta sola.
- [[specs/SPEC-CICLO-PLANIFICACION]] — ciclo de planificación desde cero.
- [[specs/SPEC-CICLO-TRABAJO]] — ciclo de trabajo con sus tres entradas.
- [[specs/SPEC-INTERFAZ]] — disposición de la pantalla y zonas.

### Importante (P1)

- [[specs/SPEC-PANEL-CONTEXTO]] — conteo de tokens y ocupación del contexto.
- [[specs/SPEC-RESOLVER]] — ciclo aparte para arreglar lo que está roto.

### Interfaz (P2)

- [[specs/SPEC-INTERFAZ-ATAJOS]] — atajos de teclado y panel de aprobaciones.

### Personalización (P3)

- [[specs/SPEC-AGENTE-PERSONALIZADO]]
- [[specs/SPEC-FLUJO-PERSONALIZADO]]
- [[specs/SPEC-SKILLS]]

## Stack tecnológico

| Área | Elección |
|---|---|
| Lenguaje | Go |
| Interfaz de terminal | Bubble Tea + Lip Gloss (charmbracelet) |
| Aislamiento de terminal | Landlock (mecanismo del kernel en Linux, sin privilegios) |
| Persistencia | SQLite con driver puro Go (`modernc.org/sqlite`), sin cgo |
| Modelo | Ollama local por API HTTP; el modelo lo elige el usuario |
| Distribución | Un único binario, sin runtime externo |

No hay Node, ni Python, ni gestor de procesos. La herramienta arranca sin terminal multiplexer y sin dependencias de sistema, aparte de Ollama.

## Arquitectura general

Un solo proceso. Los módulos se comunican por canales de Go, no por red.

```
cmd/localcli/      punto de entrada
internal/
  tui/             chat, panel de datos, selector de sesiones, aprobaciones
  session/         creación, cambio y ejecución en segundo plano de sesiones
  agent/           definiciones de plan y build, prompts, permisos y relevo
  ollama/          cliente HTTP, streaming, razonamiento, perfil de hardware
  context/         grafo de frontmatter, selección, recorte y auditoría
  flow/            motor de etapas y encadenamiento
  queue/           cola por capa y orden por dependencias
  task/            archivos de tarea
  tools/           registro de herramientas y comprobación de permisos
  fileops/         validación de rutas, acceso a archivos e historial de cambios
  exec/            terminal: lista blanca y bloqueo estructural de escritura
  store/           persistencia SQLite
  docs/            carga de documentación y lectura de frontmatter
```

### Recorrido de una petición

1. `tui` recibe lo que escribes y lo envía a la sesión activa.
2. `session` decide si la petición es un chat o el arranque de un flujo.
3. `context` arma el contexto de la etapa: lee el grafo de dependencias, filtra, deja que el modelo elija y recorta hasta el límite.
4. `agent` construye la llamada con el prompt de `plan` o de `build`, respetando qué herramientas tiene cada uno.
5. `ollama` envía la petición y devuelve el token a token.
6. `tui` muestra el razonamiento en vivo y después la respuesta.
7. Si el agente pide una herramienta, `tools` comprueba el permiso y despacha a `fileops` o a `exec`.
8. El resultado de la etapa pasa a `flow`, que decide si sigue, si se detiene o si espera tu aprobación.
9. `store` persiste el estado; `queue` toma la siguiente tarea cuando toca.

### Concurrencia

Una goroutine por sesión en ejecución, con el motor de etapas encima. Los canales de Go hacen de cola entre la TUI y el trabajo de fondo. SQLite en modo WAL para que la interfaz pueda leer mientras las sesiones de fondo escriben.

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

## Integraciones

- **Ollama.** API HTTP en local. Peticiones en streaming para poder mostrar el razonamiento mientras llega.
- **Búsqueda en internet.** Es la única integración que hace salir información de la máquina, y solo sale la consulta. El contenido que vuelve entra al mismo presupuesto de contexto que el resto y queda registrado.

## Seguridad

- **Aprobación de toda escritura.** Sin excepción, ni dentro de un flujo en curso.
- **Bloqueo estructural de la terminal.** Landlock impide que un proceso hijo escriba en los archivos del proyecto, sin depender de revisar el texto del comando. En sistemas sin Landlock, la garantía es más débil y queda documentada como tal.
- **Frontera de rutas.** Fuera de la carpeta del proyecto hace falta permiso y además la explicación del agente.
- **Sin operaciones destructivas de git.** Ni `checkout`, ni `reset`, ni `clean`, ni commits, ni push.
- **Historial de cambios.** Cada cambio aplicado guarda lo que había antes y lo que quedó, para poder revertir y auditar.

## Pruebas

- **Unitarias.** Grafo de dependencias y selección de contexto, orden de la cola por dependencias, comprobación de permisos, reparto de herramientas por agente, validación de rutas.
- **De integración.** Conexión y streaming con Ollama, ciclo completo de una etapa con las herramientas de lectura, aplicación de un cambio con su aprobación, ciclo de cola.
- **Del aislamiento.** Un caso que intente escribir en el proyecto a través de la terminal y debe fallar. Es la prueba que sostiene la garantía de Landlock.
- **De contexto.** Que el recorte reduzca de forma medible frente a leer el proyecto completo, y que lo entregado quepa en el límite del modelo.

## Despliegue

- Compilación cruzada a un binario único con `go build`.
- Requisitos en la máquina: Ollama instalado y corriendo, y Landlock disponible si se quiere el aislamiento fuerte (Linux 5.13 o superior).
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

## Límites conocidos

- **Las sesiones concurrentes comparten un solo modelo.** Ollama con 4 GB de VRAM solo tiene un modelo cargado. Varias sesiones pueden estar activas a la vez, pero sus respuestas se serializan: mientras una genera, la otra espera. Es una consecuencia del hardware, no un defecto del diseño.
- **El aislamiento fuerte es de Linux.** Landlock solo existe ahí. En otros sistemas la terminal tiene una garantía más débil y la documentación lo dice.
- **Un 7B no cabe en 4 GB de VRAM.** Si eliges uno, el harness avisa y no lo carga en silencio; irá a RAM, que es más lento y aprieta los 16 GB.
- **La inferencia no se puede paralelizar** entre sesiones, por lo mismo.

## Decisiones pendientes

Las decisiones de backend están centralizadas en [[backend/DECISIONS]]. Las globales que siguen abiertas:

- Formato exacto del frontmatter con el que se declaran las dependencias entre documentos.
- Formato definitivo del archivo de tarea (`NNN-task-<nombre>.md`) y su bloque de contexto.
- Comandos concretos de la lista blanca de la terminal, con su límite de tiempo y de salida.
- Modelo y parámetros de inferencia por defecto para el perfil recomendado.

## Referencias

- [[IDEA]]
