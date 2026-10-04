---
title: LocalCli — configuración del backend
tags: [backend, infraestructura]
depende_de:
  - "[[PROJECT]]"
  - "[[backend/DECISIONS]]"
  - "[[specs/SPEC-MODELO-MOTOR]]"
  - "[[specs/SPEC-SESIONES]]"
relacionado:
  - "[[backend/03-security/SECURITY]]"
  - "[[backend/04-infrastructure/INTEGRATIONS]]"
---
# CONFIGURATION — Configuración del backend

`LocalCli` no tiene configuración obligatoria. Arranca y funciona. Casi todo se deriva solo: dónde está el proyecto, qué modelos hay disponibles y qué modelos caben en tu máquina.

## 1. El comando

`localcli` es el comando para arrancar la aplicación. Se instala una vez y queda disponible en todo el ordenador, así que lo ejecutas desde cualquier carpeta:

```
localcli
```

La carpeta desde la que lo ejecutas **es** el proyecto. No hay que pasar la ruta como argumento ni elegirla en un menú.

## 2. Variables de entorno

Ninguna es obligatoria. No hay variables que apliques por omisión.

| Variable | Propósito | Por defecto | Obligatoria |
|---|---|---|---|
| `LOCALCLI_DB_PATH` | Sobrescribe la ruta del archivo SQLite del proyecto | Derivada de la carpeta de ejecución | No |
| `LOCALCLI_MOTOR` | Motor con el que arrancan las sesiones nuevas: el `id` de una entrada de `motores.json` | La última entrada activa si hay alguna. **Sin ninguna activa, no hay motor de arranque** | No |
| `LOCALCLI_CONTEXT_LIMIT` | Tope de la ventana de contexto y de los tokens que se respetan al recortar | `16384`, o la ventana del modelo si es menor | No |
| `LOCALCLI_ALLOW_INTERNET` | Permite las herramientas de internet | Desactivado | No |

`LOCALCLI_DB_PATH` existe solo para casos raros, como trabajar con la base en otro sitio. En el uso normal no hace falta.

`LOCALCLI_MOTOR` nombra el motor con el que arrancan las sesiones nuevas, y es una comodidad de arranque, no el mecanismo: la autoridad sobre qué motores existen y cuál usa cada sesión está en `motores.json` y en la sesión. Sin declararlo se usa la última entrada activa del registro. **Si no hay ninguna activa, no se elige ningún motor**: es el estado de partida de una instalación sin runtimes, y lo propio es abrir el modal de motores. Un `LOCALCLI_MOTOR` que no exista en el registro se ignora y se avisa: no se inventa un motor con ese nombre. Ver [[specs/SPEC-MODELO-MOTOR]] y [[backend/04-infrastructure/INTEGRATIONS]].

**No existe `LOCALCLI_LLAMACPP_URL`.** La URL de cada motor es un dato suyo, va en su entrada de `motores.json` y no depende del tipo: varios `llamacpp` pueden apuntar a direcciones distintas, igual que varios `ollama`.

`LOCALCLI_CONTEXT_LIMIT` es el tope de la ventana de contexto y del presupuesto con que se recorta: la ventana efectiva nunca lo supera y el nodo de contexto y el historial se derivan de ahí. Bajarlo en máquinas muy justas de memoria es lo habitual. Ver [[backend/DECISIONS]].

Sobre `LOCALCLI_ALLOW_INTERNET`: es la única integración que saca información de la máquina, así que no viene activada. El usuario la habilita. Ver [[backend/03-security/SECURITY]].

## 3. Los motores: se declaran, se administran

**El modelo no se configura por variable de entorno y no hay modelo por defecto.** El motor tampoco: se declara en un archivo. El flujo es:

1. `LocalCli` carga `~/.config/localcli/motores.json`, el registro de motores del usuario.
2. Se conecta al motor de cada sesión y extrae los modelos que declara su servidor.
3. La interfaz muestra esa lista para que el usuario elija, con el nombre del motor a la vista.
4. El harness comprueba si el modelo elegido cabe en la máquina y avisa si no.

El motor se administra, no se configura: se añade, se desactiva y se elimina desde el modal de motores (`Ctrl+X i`), y el registro es un archivo global de la máquina, no del proyecto. El modelo lo elige el usuario, siempre. La lista es la del motor de la sesión, no la de otro. Ver [[specs/SPEC-MODELO-MOTOR]] y [[backend/04-infrastructure/INTEGRATIONS]].

El tamaño de contexto no se configura por modelo. Con la extensión `num_ctx` (Ollama) se declara por petición, como la menor entre lo que reporta el modelo en `/api/tags` y el tope (`LOCALCLI_CONTEXT_LIMIT`, 16384 por defecto). Con la extensión `props` (llama.cpp) se lee del servidor, que lo fijó quien lo arrancó con `-c`. Sin extensión que la declare queda **desconocida**, y entonces manda el tope con el aviso de que el límite es del harness. En cualquier caso **el recorte lo hace el harness**, no el servidor: si no alcanza para un turno con herramientas, se avisa con la instrucción de levantar el servidor con más contexto y el turno se recorta de más nuevo a más viejo. Ver [[backend/DECISIONS]].

## 4. Rutas

Todo se deriva de la carpeta desde la que se ejecuta `localcli`:

| Qué | Dónde |
|---|---|
| Carpeta del proyecto | La carpeta desde la que se ejecuta `localcli` |
| Archivo SQLite | `.localcli/state.db` dentro del proyecto, o `LOCALCLI_DB_PATH` si se sobrescribe |
| Documentación | `ai/docs/` dentro del proyecto |
| TODO de trabajo | `ai/tasks/` dentro del proyecto |
| Herramientas del usuario | `.localcli/tools/*.json` dentro del proyecto |
| Agentes del proyecto | `.localcli/agents/<carpeta>/` dentro del proyecto, con `agent.yaml` y `prompt.md` |
| Archivos y carpetas | Todo lo que cuelgue de la carpeta del proyecto |

El proyecto es la carpeta abierta. El chat de una carpeta nunca aparece en otra. Ver [[specs/SPEC-SESIONES]].

El archivo SQLite va en `.localcli/state.db` dentro del proyecto, y ese archivo está fuera de git. La ruta se deriva de la carpeta abierta y no necesita variables; `LOCALCLI_DB_PATH` existe solo para los casos raros, como trabajar con la base en otro sitio. Ver [[backend/DECISIONS]].

De `.localcli/` solo se ignora el estado: `state.db` y sus archivos `-shm` y `-wal` son de la máquina. `tools/` y `agents/` son contenido del proyecto y se versionan con él, que es lo que permite que cada clon traiga los mismos agentes base.

### Agentes del proyecto

Cada **carpeta** de `.localcli/agents/` declara un agente, y dentro de ella los dos archivos son obligatorios:

```
.localcli/agents/build/
├── agent.yaml
└── prompt.md
```

- **`agent.yaml`** — la configuración: `name`, `description` y `permissions`. El nombre del agente lo decide `name`; el de la carpeta es solo donde vive.
- **`prompt.md`** — las instrucciones del modelo, que se inyectan íntegras como mensaje de sistema.

Los dos base, `plan` y `build`, se versionan con el proyecto. No hay nada que activar: si la carpeta está, el agente existe y `Tab` lo recorre. Una carpeta mal formada se avisa y se salta, y `plan` y `build` quedan disponibles con un prompt de respaldo aunque falten, porque los flujos oficiales los referencian. Un `*.json` suelto en `.localcli/agents/` —el formato antiguo— se avisa nombrándolo, sin cargar: la migración es manual. Ver [[specs/SPEC-AGENTE-BASE]] y [[backend/02-interfaces/INTERFACES-GENERAL]].

El ejemplo, tal cual está en disco:

```yaml
name: build
description: Implementa los cambios propuestos y verificados por el usuario.
permissions:
  default: deny
  read: allow
  write: allow
  edit: allow
```

Lo que no se nombre queda en `deny`, y `default: allow` no se admite: un `allow` donde se quería un `deny` concedería todo en silencio. Un campo desconocido —`skills`, `herramientas`, `prompt`— hace que el agente no cargue.

### Herramientas del usuario

El usuario puede añadir herramientas sin tocar el código: cada archivo `.json` de `.localcli/tools/` declara una. La carpeta es del proyecto, así que se versiona con él y cada uno lleva las suyas. No hay nada que activar: si el archivo está, la herramienta existe.

```json
{
  "nombre": "contar_lineas",
  "descripcion": "Cuenta las líneas de un archivo del proyecto.",
  "modo": "lee",
  "equipo": ["wc", "-l"]
}
```

- **Se declaran, no se ejecutan al leerlas.** Un archivo mal formado se avisa y se salta; no rompe el arranque ni las demás herramientas.
- **No se confiables por estar en la carpeta.** Toda ejecución pide permiso, sin excepción, aunque el agente ya tenga permiso para herramientas de lectura. Ver [[backend/03-security/SECURITY]].
- **Solo lectura.** `modo` tiene un único valor admisible, `lee`. Declarar `escribe` se rechaza con un aviso: Landlock impide que un proceso escriba en el proyecto, así que una herramienta de escritura no podría cumplir su promesa y fallaría de forma confusa. Si algún día se admite, será una decisión de seguridad nueva, no un cambio de configuración. Ver [[backend/02-interfaces/TOOLS]].
- **`equipo` es argv, no una línea de shell.** El primer elemento es el programa y los siguientes sus argumentos. No hay shell, no hay comillas que interpretar y no hay encadenado con `|`, `>` ni `&&`.
- **Se ejecutan como un subproceso** con la misma lista blanca y el mismo aislamiento que la terminal, y su salida se recorta antes de volver al modelo. Ver [[specs/SPEC-TOOLS]].

## 5. Configuración por ambiente

No hay ambientes de servidor. Hay tres formas de ejecutar:

| Modo | Para qué | Base de datos | Internet |
|---|---|---|---|
| Normal | Uso diario | Un archivo por proyecto | Solo si se habilita |
| Pruebas | Tests automatizados | Base temporal, aislada, que se destruye al terminar | Desactivado |
| Sin aislamiento fuerte | Sistema sin Landlock | Igual que el normal | Igual que el normal |

El tercer modo no es un flag: depende de si Landlock está disponible en el sistema. Si no lo está, el harness lo avisa y la garantía de la terminal es más débil, sin bloquear el uso. Ver [[backend/04-infrastructure/INTEGRATIONS]].

## 6. Servicios requeridos

| Servicio | Necesario para | Si no está |
|---|---|---|
| Al menos un motor corriendo en local —`ollama` o `llama-server`— | Generar respuestas y listar los modelos disponibles | El harness no puede generar nada; la interfaz sigue viva y avisa de cómo levantarlo |
| Landlock (Linux 5.13+) | El bloqueo estructural de escritura en la terminal | La terminal sigue funcionando, con garantía más débil, y el harness lo dice |

No hay más servicios. No hay base de datos que levantar, ni migraciones que aplicar a mano, ni cuentas que crear. Ver [[specs/SPEC-MODELO-MOTOR]].

## 7. Preferencias del usuario

Además de `~/.config/localcli/keys.json` (el mapa de teclas), el usuario tiene sus preferencias en `~/.config/localcli/config.json`:

```json
{
  "ultimo_motor": "ollama-local",
  "ultimo_modelo": "llama3.2",
  "ultimo_agente": "build",
  "historial_tokens": 4096
}
```

- Son **globales del usuario**, no de un proyecto: viven fuera de la carpeta y no se versionan.
- `ultimo_motor` guarda el `id` de la última entrada de `motores.json` que se usó, y es de qué motor era cuando se eligió `ultimo_modelo`. Los dos van juntos: **el mismo nombre de modelo no significa lo mismo en runtimes distintos**, así que el modelo recordado solo se reutiliza si el motor recordado es el que tiene la sesión. Si uno de los dos no cuadra con el motor de la sesión, se vuelve a autodetectar.
- El par motor/modelo de una sesión se guarda **en la sesión** (`sessions`), no en este archivo. Estas preferencias son el punto de partida de las sesiones nuevas, no el estado de las existentes.
- `historial_tokens` es el presupuesto de tokens del historial de conversación que se le entrega al modelo; sin él se usa `LOCALCLI_CONTEXT_LIMIT` y, en su defecto, un valor por defecto. Ver [[specs/SPEC-HISTORIAL-CONVERSACION]].
- Un archivo ausente o ilegible no rompe el arranque: se usan los valores por defecto (`plan` como agente, autodetección como modelo y motor).
- Un `ultimo_motor` que ya no esté registrado o activo no se reutiliza, y **no se sustituye por otro automático**: es el caso del registro sin motores activos, que se resuelve abriendo el modal.
- La interfaz los escribe al elegir motor o modelo, o al cambiar de agente; no hay que editar el archivo a mano. Ver [[specs/SPEC-MODELO-MOTOR]].

## 8. El registro de motores

Los motores viven en `~/.config/localcli/motores.json`, un archivo global de la máquina:

```json
{
  "motores": [
    {
      "id": "ollama-local",
      "nombre": "Ollama local",
      "tipo": "ollama",
      "url": "http://localhost:11434",
      "activo": true,
      "extensiones": ["num_ctx", "show", "tags"]
    },
    {
      "id": "router-13b",
      "nombre": "Router 13B",
      "tipo": "llamacpp",
      "url": "http://localhost:8080",
      "activo": true,
      "extensiones": ["props"]
    },
    {
      "id": "gpu-remota",
      "nombre": "GPU de casa",
      "tipo": "ollama",
      "url": "http://192.168.1.20:11434",
      "activo": false,
      "extensiones": ["num_ctx", "show", "tags"]
    }
  ],
  "modelos_por_motor": {
    "ollama-local": ["qwen3:8b", "gemma3:12b"],
    "router-13b": ["qwen3-14b-q4"]
  }
}
```

- **`id`** — identidad estable del motor, con la que las sesiones lo nombran. No cambia aunque se le edite el nombre o la URL, para que las sesiones que lo usaban no se queden huérfanas.
- **`nombre`** — etiqueta para el usuario, la que se ve en el modal y en la línea de estado.
- **`tipo`** — `ollama` o `llamacpp`. Es cerrado: un tipo desconocido se rechaza al cargar y se avisa con el nombre.
- **`url`** — dirección base del servidor. Es del motor, no del tipo, así que admite varias instancias de cada tipo.
- **`activo`** — si el motor se puede usar. Desactivar no borra nada: el registro se queda y las sesiones que lo usaban siguen existiendo, solo que avisan hasta que se elija otro o se reactive.
- **`modelos_por_motor`** — el último catálogo que se vio de cada motor, para que el modal abra al instante. Es una caché de lo último conocido, no una verdad: al abrir se pregunta al motor y lo que responda manda.

El archivo es del usuario y se administra desde el modal de motores (`Ctrl+X i`), pero se puede editar a mano. **Si el archivo está ausente, el registro queda vacío y no se siembra nada**: LocalCli se puede instalar en una máquina sin ningún runtime de inferencia, así que no se presupone que haya uno. Es un estado válido y estable, no un error; lo propio es abrir el modal de motores y activar o dar de alta el que haya. Si está mal formado, se avisa el error y el registro queda igualmente vacío. Ver [[specs/SPEC-MODELO-MOTOR]].

**El registro se entrega escrito.** LocalCli trae un `motores.json` de ejemplo con las dos entradas ya configuradas —tipo, dirección y extensiones— y **`activo: false`**, para que el usuario solo tenga que activar la que tenga instalada en vez de escribir las direcciones de referencia:

```json
{
  "motores": [
    {
      "id": "ollama-local",
      "nombre": "Ollama local",
      "tipo": "ollama",
      "url": "http://localhost:11434",
      "activo": false,
      "extensiones": ["num_ctx", "show", "tags"]
    },
    {
      "id": "llamacpp-local",
      "nombre": "llama.cpp local",
      "tipo": "llamacpp",
      "url": "http://localhost:8080",
      "activo": false,
      "extensiones": ["props"]
    }
  ],
  "modelos_por_motor": {}
}
```

Que ninguna entre activa es lo que hace que no haya proveedor por defecto: **el default que no existe es el activo**, no los datos. Ver [[specs/SPEC-MODELO-MOTOR]].

El campo `extensiones` es opcional y ausente se lee como «solo núcleo común». Un motor sin él funciona y el harness declara desconocido lo que no informa, en vez de suponerlo.

**Editar un motor ya aplicado se aplica al reiniciar.** Añadir, desactivar, reactivar y eliminar sí se aplican al momento. La razón está en [[backend/DECISIONS]]: el adaptador vive en memoria mientras corre el harness, y recrearlo en caliente dejaría dos versiones del mismo motor sirviendo turnos a la vez.

## Referencias

- [[backend/02-interfaces/TOOLS]] — el contrato de una herramienta del usuario y cómo se ejecuta.
- [[specs/SPEC-TOOLS]] — la especificación funcional del catálogo.
- [[backend/04-infrastructure/INTEGRATIONS]] — cómo se habla con los motores de inferencia y cómo se listan los modelos.
- [[backend/03-security/SECURITY]] — los límites de lo que puede hacer una herramienta.
- [[backend/DECISIONS]] — decisiones de configuración ya cerradas.
- [[PROJECT]] — requisitos en la máquina y despliegue.
