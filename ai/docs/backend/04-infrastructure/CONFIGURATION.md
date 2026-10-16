---
title: LocalCli — configuración del backend
tags: [backend, infraestructura]
depende_de:
  - "[[PROJECT]]"
  - "[[backend/DECISIONS]]"
  - "[[specs/SPEC-MODELOS-LLAMACPP]]"
  - "[[specs/SPEC-SESIONES]]"
relacionado:
  - "[[backend/03-security/SECURITY]]"
  - "[[backend/04-infrastructure/INTEGRATIONS]]"
---
# CONFIGURATION — Configuración del backend

`LocalCli` necesita su raíz de máquina para arrancar. Si no está creada, no arranca solo: lo avisa y pide el comando que la crea. Una vez creada, casi todo se deriva solo: dónde está el proyecto, qué modelos hay descargados y qué modelos caben en tu máquina.

## 1. El comando y sus subcomandos

`localcli` es el comando. Se instala una vez y queda disponible en todo el ordenador, así que lo ejecutas desde cualquier carpeta. **La carpeta desde la que lo ejecutas es el proyecto.**

| Comando | Qué hace |
|---|---|
| `localcli` | Arranca el harness sobre la carpeta actual (el proyecto) |
| `localcli -h` | Lista los comandos disponibles |
| `localcli -v` | Muestra la versión |
| `localcli init` | Crea la carpeta global `~/.config/localcli/` y siembra lo que falta |
| `localcli create` | Crea la configuración del proyecto (`.localcli/`) en la carpeta actual |

**Requisito de arranque: la raíz global.** El harness necesita `~/.config/localcli/`. Si no existe, `localcli` **no la crea**: escribe en consola «localcli no iniciada, ejecuta el comando `localcli init`» y termina. `init` es la única forma de crearla.

- **`localcli init`** — crea `~/.config/localcli/` y siembra en ella lo que falte, sin sobrescribir nada de lo que ya haya.
- **`localcli create`** — crea la configuración **del proyecto** en la carpeta donde estás: `<carpeta actual>/.localcli/`, que es donde viven los agentes y flujos propios.

## 2. Variables de entorno

Ninguna es obligatoria. No hay variables que apliques por omisión.

| Variable | Propósito | Por defecto | Obligatoria |
|---|---|---|---|
| `LOCALCLI_DB_PATH` | Sobrescribe la ruta del archivo SQLite global | `~/.config/localcli/state.db` | No |
| `LOCALCLI_CONTEXT_LIMIT` | Fuerza (override) la ventana de contexto; sin ella, se usa el contexto real del modelo con margen 80% | Sin definir: el contexto del modelo (su GGUF, preferentemente `<arch>.context_length`) o `16384` si no lo declara | No |
| `LOCALCLI_ALLOW_INTERNET` | Permite las herramientas de internet | Desactivado | No |

`LOCALCLI_DB_PATH` existe solo para casos raros, como trabajar con la base en otro sitio. En el uso normal no hace falta: la base es un único archivo global en `~/.config/localcli/state.db`, y no depende de la carpeta desde la que se ejecute.

**El proveedor y la dirección de la API no son variables de entorno.** Viven en `~/.config/localcli/localcli.json`, y son la base con la que el harness arranca `llama-server` y con la que el cliente habla con él. Ver [[specs/SPEC-MODELOS-LLAMACPP]].

`LOCALCLI_CONTEXT_LIMIT` es un **override**: cuando está definido, manda él (sin margen). Sin él, la ventana sale del **contexto declarado del modelo** (su GGUF, preferentemente `<arch>.context_length`) con un **margen del 80 %**, y, si el modelo no lo declara, de `16384`. Esa ventana se usa para arrancar `llama-server` (`-c`), para el presupuesto del historial y para el recorte por ronda. Bajarlo fuerza menos contexto; no definirlo deja que el modelo use el suyo. Ver [[backend/DECISIONS]].

Sobre `LOCALCLI_ALLOW_INTERNET`: es la única integración que saca información de la máquina, así que no viene activada. El usuario la habilita. Ver [[backend/03-security/SECURITY]].

## 3. El proveedor: llama.cpp, administrado por el harness

**El modelo no se configura por variable de entorno ni se declara en la configuración global.** El proveedor tampoco se administra: la decisión está cerrada. Ver [[specs/SPEC-MODELOS-LLAMACPP]].

1. `LocalCli` lee `~/.config/localcli/localcli.json`, que guarda el proveedor (`llama.cpp`) y la dirección base de la API (`api`). Nada más: ni el modelo ni los modelos descargados.
2. El usuario elige un modelo del catálogo ([[#8. El catálogo de modelos]]) por su `id` (el nombre de su carpeta). La selección vive en el catálogo y en el estado de la aplicación, no en la configuración global.
3. El harness arranca `llama-server` con la **ruta del archivo GGUF** del modelo elegido, **identificándolo a `llama-server` por su nombre (carpeta)** para que `/v1/models` y las peticiones del chat usen ese nombre y no la ruta, y con el puerto correspondiente a `api`; comprueba que el servidor responde antes de usarlo.
4. El harness comprueba si el modelo cabe en la máquina y avisa si no.

El servidor no se declara en ninguna parte: **quien lo levanta es el harness**, con el modelo seleccionado, y quien lo detiene también. Cambiar de modelo reinicia el servidor; LocalCli gestiona ese proceso ([[specs/SPEC-MODELOS-LLAMACPP]]). No hay otros runtimes: no se usa Ollama ni se mantiene su almacén duplicado.

El tamaño de contexto se fija **al arrancar** `llama-server` con `-c`, tomándolo del **contexto declarado del modelo** (su GGUF, preferentemente `<arch>.context_length` sobre `general.context_length`) con un **margen del 80 %**, o de `LOCALCLI_CONTEXT_LIMIT` si se define (sin margen); si el modelo no declara contexto, se usa `16384`. La ventana que el turno respeta es la que el servidor acepta **de verdad**, releída con la extensión `props` de llama.cpp. **El recorte lo hace el harness**, no el servidor: antes de cada ronda se descartan los mensajes más antiguos —y se trunca el resultado de herramienta que por sí solo no quepa— hasta que el prompt cabe en la ventana menos la reserva de salida, avisando del recorte. Ver [[backend/DECISIONS]].

## 4. Rutas

Todo se deriva de dos raíces: la carpeta desde la que se ejecuta `localcli` (el proyecto) y `~/.config/localcli/` (la raíz de la máquina).

| Qué | Dónde |
|---|---|
| Carpeta del proyecto | La carpeta desde la que se ejecuta `localcli` |
| Archivo SQLite (global) | `~/.config/localcli/state.db`, o `LOCALCLI_DB_PATH` si se sobrescribe |
| Proyecto (dimensión lógica) | Una fila de `projects` con el `path` de la carpeta abierta |
| Documentación | `ai/docs/` dentro del proyecto |
| TODO de trabajo | `ai/tasks/` dentro del proyecto |
| Configuración del proveedor | `~/.config/localcli/localcli.json` (solo `proveedor` y `api`) |
| Otras preferencias del harness | `~/.config/localcli/config.json` y `~/.config/localcli/keys.json` |
| Modelos GGUF | `~/.config/localcli/models/`, un subdirectorio por modelo |
| Catálogo de modelos | `~/.config/localcli/models.json` |
| Agentes base | `~/.config/localcli/agents/`, sembrados desde el binario |
| Flujos oficiales | `~/.config/localcli/flows/`, sembrados desde el binario |
| Agentes y flujos propios | `<proyecto>/.localcli/agents/` y `<proyecto>/.localcli/flows/`, aditivos |
| Archivos y carpetas | Todo lo que cuelgue de la carpeta del proyecto |

El proyecto es la carpeta abierta. La base es única y global, pero el chat de una carpeta nunca aparece en otra: cada carpeta es una fila de `projects` con su `path` único, y toda sesión cuelga de ella por `project_id`. Ver [[specs/SPEC-SESIONES]].

El archivo SQLite vive en `~/.config/localcli/state.db`, fuera del proyecto, junto al resto de la configuración del harness, y por eso no se versiona ni se ignora por proyecto. La ruta se deriva de la raíz de la máquina y no necesita variables; `LOCALCLI_DB_PATH` existe solo para los casos raros, como trabajar con la base en otro sitio. Ver [[backend/DECISIONS]].

El proyecto solo aporta contenido **aditivo** en `.localcli/`: agentes y flujos propios. No contiene estado de la máquina, así que todo `.localcli/` es contenido del proyecto y se versiona con él.

**El arranque exige la raíz global.** El primer paso del harness es comprobar `~/.config/localcli/`. Si no existe, **no continúa y no la crea**: avisa «localcli no iniciada, ejecuta el comando `localcli init`» y termina. La raíz la crea y siembra `localcli init` —los agentes base, los flujos base, la base de datos (`state.db`, nacida en su versión actual) y la configuración por defecto (`localcli.json`, `config.json`, `keys.json`, `models.json`) y crea `models/`—, sin sobrescribir nunca lo que ya haya; lo que no está en archivos simplemente no aparece: no hay datos de respaldo dentro del código. `localcli create` hace lo propio para el proyecto, creando `<carpeta actual>/.localcli/`. Ver [[backend/DECISIONS]] y [[database/03-operations/SEEDING]].

### Agentes del proyecto (aditivos)

Los agentes **base** (`plan`, `build`, `chat`) se siembran desde el binario en `~/.config/localcli/agents/` al ejecutar `localcli init`. Un proyecto solo **añade** agentes propios en `<proyecto>/.localcli/agents/`; la carga es aditiva y nunca sobrescribe un agente oficial. Si un agente propio declara un `name` que ya existe en los base, se avisa y se conserva el global.

Cada **carpeta** de un directorio `agents/` declara un agente, y dentro de ella los dos archivos son obligatorios:

```
.localcli/agents/build/
├── agent.yaml
└── prompt.md
```

- **`agent.yaml`** — la configuración: `name`, `description` y `permissions`. El nombre del agente lo decide `name`; el de la carpeta es solo donde vive.
- **`prompt.md`** — las instrucciones del modelo, que se inyectan íntegras como mensaje de sistema.

Como los base ya vienen sembrados, no hay que versionarlos en cada proyecto. No hay nada que activar: si la carpeta está, el agente existe y `Tab` lo recorre. Una carpeta mal formada se avisa y se salta, y su disponibilidad es la de cualquier dato: si no está, no aparece —no hay agentes de respaldo en el código—. Un `*.json` suelto en un directorio `agents/` —el formato antiguo— se avisa nombrándolo, sin cargar: la migración es manual. Ver [[specs/SPEC-AGENTE-BASE]] y [[backend/02-interfaces/INTERFACES-GENERAL]].

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

## 5. Configuración por ambiente

No hay ambientes de servidor. Hay tres formas de ejecutar:

| Modo | Para qué | Base de datos | Internet |
|---|---|---|---|
| Normal | Uso diario | Un único archivo global | Solo si se habilita |
| Pruebas | Tests automatizados | Base temporal, aislada, que se destruye al terminar | Desactivado |
| Sin aislamiento fuerte | Sistema sin Landlock | Igual que el normal | Igual que el normal |

El tercer modo no es un flag: depende de si Landlock está disponible en el sistema. Si no lo está, el harness lo avisa y la garantía de la terminal es más débil, sin bloquear el uso. Ver [[backend/04-infrastructure/INTEGRATIONS]].

## 6. Servicios requeridos

| Servicio | Necesario para | Si no está |
|---|---|---|
| `llama-server` (lo arranca el propio harness) | Generar respuestas | El harness no puede generar nada; lo arranca con el modelo elegido y, si falta el binario o el modelo no está descargado, avisa de cómo resolverlo |
| Landlock (Linux 5.13+) | El bloqueo estructural de escritura en la terminal | La terminal sigue funcionando, con garantía más débil, y el harness lo dice |

`llama-server` no es un servicio externo que haya que mantener: forma parte del ciclo de vida que gestiona LocalCli. No hay más servicios. No hay base de datos que levantar, ni migraciones que aplicar a mano, ni cuentas que crear. Ver [[specs/SPEC-MODELOS-LLAMACPP]].

## 7. Preferencias del usuario

Además de `~/.config/localcli/keys.json` (el mapa de teclas), el usuario tiene sus preferencias en `~/.config/localcli/config.json`:

```json
{
  "ultimo_agente": "build",
  "historial_tokens": 4096,
  "compactacion": {
    "auto": true,
    "umbral": 0.6,
    "conservar_tokens": 15000,
    "formato": "opencode"
  }
}
```

- Son **globales del usuario**, no de un proyecto: viven fuera de la carpeta y no se versionan.
- **El modelo elegido no está aquí.** La selección del modelo se gestiona por el catálogo y el estado de la aplicación ([[#8. El catálogo de modelos]]), no en la configuración global: `localcli.json` solo guarda `proveedor` y `api`. Ver [[specs/SPEC-MODELOS-LLAMACPP]].
- `ultimo_agente` es el punto de partida de las sesiones nuevas.
- `historial_tokens` es el presupuesto de tokens del historial de conversación que se le entrega al modelo; sin él se usa `LOCALCLI_CONTEXT_LIMIT` y, en su defecto, la fracción de la ventana efectiva que fija el umbral de compactación (60 % por defecto) y `4096` si no se conoce la ventana. Ver [[specs/SPEC-HISTORIAL-CONVERSACION]].
- `compactacion` gobierna la compactación del historial ([[specs/SPEC-HISTORIAL-CONVERSACION]]):
  - `auto` — si la compactación automática está activa. Por defecto `true`.
  - `umbral` — fracción de la ventana efectiva a la que se dispara la compactación. Por defecto `0.6` (60 %).
  - `conservar_tokens` — tokens recientes que viajan literales tras el resumen. Por defecto `15000`; nunca supera el presupuesto del historial, al que además se le reserva un margen para el propio resumen.
  - `modelo` — modelo que resume; sin él, el modelo activo. Pensado para uno ligero.
  - `formato` — `opencode` (por defecto), `legacy` o `custom`.
  - `plantilla` — prompt literal del resumen, solo con `formato: "custom"`.
- Un archivo ausente o ilegible no rompe el arranque: se usan los valores por defecto (`plan` como agente).
- La interfaz los escribe al cambiar de agente; no hay que editar el archivo a mano. El bloque `compactacion` se edita a mano (la interfaz aún no lo expone).

## 8. El catálogo de modelos

Los modelos descargados viven en `~/.config/localcli/models/`, un archivo GGUF original **por modelo dentro de su subdirectorio**, sin copias de ejecución. **La carpeta de cada modelo es su nombre** (`id`): con ese nombre se le lista en la interfaz, se le selecciona y se le nombra al servidor; la ruta del GGUF es solo el detalle interno de arranque:

```
~/.config/localcli/models/
└── qwen3-14b/
    ├── qwen3-14b-q4_k_m.gguf
    └── info.json
```

El registro de lo descargado es el catálogo `~/.config/localcli/models.json`, **independiente de la configuración global**:

```json
{
  "modelos": [
    {
      "id": "qwen3-14b",
      "ruta": "~/.config/localcli/models/qwen3-14b/qwen3-14b-q4_k_m.gguf",
      "repo": "QuantFactory/Qwen3-14B-GGUF",
      "archivo": "qwen3-14b-q4_k_m.gguf",
      "mmproj": "mmproj-f16.gguf",
      "ventana": 4096,
      "tamano": 9123456789
    }
  ],
  "seleccionado": "qwen3-14b"
}
```

- **`id`** — identidad del modelo en el catálogo y su **nombre**: el nombre de su subdirectorio en `models/`. Es con lo que las sesiones lo nombran, la interfaz lo muestra y el servidor lo identifica al arrancarlo. Nunca es una ruta.
- **`ruta`** — ruta del archivo GGUF original en `models/`. Es la que se pasa a `llama-server` al arrancar; no se copia el archivo a ningún otro sitio y no se expone como nombre del modelo.
- **`repo`** / **`archivo`** — repositorio de Hugging Face de origen y nombre de la variante GGUF, para identificar el modelo y reanudar o redescargarlo.
- **`mmproj`** — nombre del proyector multimodal dentro de la carpeta del modelo, si es multimodal. Es opcional: si falta, el harness usa por convención el primer `mmproj*.gguf` de la carpeta (`models/<id>/`). El proyector se pasa a `llama-server` con `--mmproj`; sin él, el modelo no procesa imágenes. Ver [[specs/SPEC-MODELOS-LLAMACPP]].
- **`ventana`** / **`tamano`** — metadatos del modelo: **contexto declarado del modelo** (leído de su GGUF, preferentemente `<arch>.context_length`, al registrarlo) y tamaño, con los que el harness fija el `-c` de `llama-server` (aplicando el margen del 80 %) y comprueba si el modelo cabe. Si `ventana` es `0` el harness avisa.
- **`seleccionado`** — el modelo activo, por su `id` (nombre de carpeta). La selección se gestiona por el catálogo y el estado de la aplicación, **nunca** en `localcli.json`. Cambiar de modelo reinicia `llama-server`. Ver [[specs/SPEC-MODELOS-LLAMACPP]].

El catálogo **no se siembra con modelos**: nace vacío (`"modelos": [], "seleccionado": null`), porque los modelos los descarga el usuario. Sí se siembra el archivo y se crea `models/`. LocalCli escribe el catálogo al **descargar** un modelo (antes de empezar, comprueba si el archivo ya existe para no descargarlo dos veces) y al **seleccionarlo**, y nunca borra un modelo existente aunque haya errores. Una descarga interrumpida se reanuda si el servidor lo permite. Ver [[specs/SPEC-MODELOS-LLAMACPP]].

La descarga se inicia desde la interfaz con el modal «DESCARGAR MODELOS» (`/descargar` o `<leader>d`): el usuario pega la URL de un archivo `.gguf`, el nombre se deriva del archivo (sin la extensión) y se muestra en directo; al confirmar, la descarga corre en segundo plano con reanudación, validación de tamaño y, si la URL es de Hugging Face, resolución del hash SHA256 vía la API del repositorio. El archivo se guarda en `models/<nombre>/<archivo>` y se registra en `models.json`; una descarga en curso se cancela al lanzar otra o al cerrar el modal con `Esc`. Ver [[specs/SPEC-INTERFAZ]] y [[specs/SPEC-KEYBINDS]].

Un archivo ausente se trata como catálogo vacío; uno mal formado se avisa y se trata igual, sin sobrescribirlo. Los errores de validación **no sobrescriben** la configuración ni el catálogo.

## Referencias

- [[backend/02-interfaces/TOOLS]] — el contrato del catálogo de herramientas y su capa universal.
- [[specs/SPEC-TOOLS]] — la especificación funcional del catálogo.
- [[backend/04-infrastructure/INTEGRATIONS]] — cómo se habla con llama-server y cómo se listan los modelos.
- [[backend/03-security/SECURITY]] — los límites de lo que puede hacer una herramienta.
- [[backend/DECISIONS]] — decisiones de configuración ya cerradas.
- [[PROJECT]] — requisitos en la máquina y despliegue.