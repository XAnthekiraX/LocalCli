---
title: LocalCli — configuración del backend
tags: [backend, infraestructura]
depende_de:
  - "[[PROJECT]]"
  - "[[backend/DECISIONS]]"
  - "[[specs/SPEC-MODELO-PROVEEDOR]]"
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
| `LOCALCLI_PROVEEDOR` | Runtime del modelo: `ollama`, `llamacpp` o `auto` | `ollama` | No |
| `LOCALCLI_LLAMACPP_URL` | Dirección del servidor `llama-server` cuando el proveedor es `llamacpp` | `http://localhost:8080` | No |
| `LOCALCLI_CONTEXT_LIMIT` | Tope de la ventana de contexto y de los tokens que se respetan al recortar | `16384`, o la ventana del modelo si es menor | No |
| `LOCALCLI_ALLOW_INTERNET` | Permite las herramientas de internet | Desactivado | No |

`LOCALCLI_DB_PATH` existe solo para casos raros, como trabajar con la base en otro sitio. En el uso normal no hace falta.

`LOCALCLI_PROVEEDOR` elige el runtime del modelo y no se cambia con la sesión viva. Sin declararlo se usa `ollama`, para que el comportamiento por defecto no dependa de lo que haya instalado. `ollama` y `llamacpp` fijan uno y solo uno: si no responde, se avisa y no se cae al otro. `auto` es opcional a propósito: quien lo declara quiere que se use el primero que responde, empezando por `ollama`. `LOCALCLI_LLAMACPP_URL` solo se lee con `llamacpp` y da la dirección del servidor. Ver [[specs/SPEC-MODELO-PROVEEDOR]] y [[backend/04-infrastructure/INTEGRATIONS]].

`LOCALCLI_CONTEXT_LIMIT` es el tope de la ventana de contexto y del presupuesto con que se recorta: la ventana efectiva nunca lo supera y el nodo de contexto y el historial se derivan de ahí. Bajarlo en máquinas muy justas de memoria es lo habitual. Ver [[backend/DECISIONS]].

Sobre `LOCALCLI_ALLOW_INTERNET`: es la única integración que saca información de la máquina, así que no viene activada. El usuario la habilita. Ver [[backend/03-security/SECURITY]].

## 3. El modelo: se detecta, no se configura

**No hay variable de entorno para el modelo, y no hay modelo por defecto.** El flujo es:

1. `LocalCli` elige el proveedor: el declarado en `LOCALCLI_PROVEEDOR`, u `ollama` si no hay ninguno.
2. Se conecta a él y extrae los modelos disponibles que declara el servidor.
3. La interfaz muestra esa lista para que el usuario elija, con el nombre del proveedor a la vista.
4. El harness comprueba si el modelo elegido cabe en la máquina y avisa si no.

El modelo lo elige el usuario, siempre. La lista es la del proveedor activo, no la de Ollama. Ver [[specs/SPEC-MODELO-PROVEEDOR]] y [[backend/04-infrastructure/INTEGRATIONS]].

El tamaño de contexto no se configura por modelo. Con Ollama se declara por petición (`num_ctx`) como la menor entre lo que reporta el modelo en `/api/tags` y el tope (`LOCALCLI_CONTEXT_LIMIT`, 16384 por defecto). Con llama.cpp se lee del servidor, que lo fijó quien lo arrancó con `-c`, y la efectiva es la menor entre ese valor y el tope. Si no alcanza para un turno con herramientas, se avisa con la instrucción de levantar el servidor con más contexto y el turno se recorta. Ver [[backend/DECISIONS]].

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
| Un proveedor de modelo corriendo en local —Ollama o `llama-server`— | Generar respuestas y listar los modelos disponibles | El harness no puede generar nada; la interfaz sigue viva y avisa de cómo levantarlo |
| Landlock (Linux 5.13+) | El bloqueo estructural de escritura en la terminal | La terminal sigue funcionando, con garantía más débil, y el harness lo dice |

No hay más servicios. No hay base de datos que levantar, ni migraciones que aplicar a mano, ni cuentas que crear. Ver [[specs/SPEC-MODELO-PROVEEDOR]].

## 7. Preferencias del usuario

Además de `~/.config/localcli/keys.json` (el mapa de teclas), el usuario tiene sus preferencias en `~/.config/localcli/config.json`:

```json
{
  "ultimo_modelo": "llama3.2",
  "ultimo_agente": "build",
  "ultimo_proveedor": "ollama",
  "historial_tokens": 4096
}
```

- Son **globales del usuario**, no de un proyecto: viven fuera de la carpeta y no se versionan.
- Al arrancar se reutiliza `ultimo_modelo` **solo si su proveedor es el elegido**: el mismo nombre no significa lo mismo en los dos runtimes, así que con otro proveedor se autodetecta. El agente recordado lo aplica la vista.
- `historial_tokens` es el presupuesto de tokens del historial de conversación que se le entrega al modelo; sin él se usa `LOCALCLI_CONTEXT_LIMIT` y, en su defecto, un valor por defecto. Ver [[specs/SPEC-HISTORIAL-CONVERSACION]].
- Un archivo ausente o ilegible no rompe el arranque: se usan los valores por defecto (`plan` como agente, autodetección como modelo).
- La interfaz los escribe al elegir modelo o al cambiar de agente; no hay que editar el archivo a mano. Ver [[specs/SPEC-MODELO-PROVEEDOR]].

## Referencias

- [[backend/02-interfaces/TOOLS]] — el contrato de una herramienta del usuario y cómo se ejecuta.
- [[specs/SPEC-TOOLS]] — la especificación funcional del catálogo.
- [[backend/04-infrastructure/INTEGRATIONS]] — cómo se habla con los proveedores de modelo y cómo se listan los modelos.
- [[backend/03-security/SECURITY]] — los límites de lo que puede hacer una herramienta.
- [[backend/DECISIONS]] — decisiones de configuración ya cerradas.
- [[PROJECT]] — requisitos en la máquina y despliegue.
