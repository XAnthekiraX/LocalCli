---
title: LocalCli — configuración del backend
tags: [backend, infraestructura]
depende_de:
  - "[[PROJECT]]"
  - "[[backend/DECISIONS]]"
  - "[[specs/SPEC-OLLAMA-PERFIL]]"
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
| `LOCALCLI_CONTEXT_LIMIT` | Tope de la ventana de contexto (`num_ctx`) que se pide a Ollama y de los tokens que se respetan al recortar | `16384`, o la ventana del modelo si es menor | No |
| `LOCALCLI_ALLOW_INTERNET` | Permite las herramientas de internet | Desactivado | No |

`LOCALCLI_DB_PATH` existe solo para casos raros, como trabajar con la base en otro sitio. En el uso normal no hace falta.

`LOCALCLI_CONTEXT_LIMIT` es el tope de la ventana de contexto que cada petición declara a Ollama (`num_ctx`). Sin él, la ventana es la menor entre lo que declara el modelo y 16384; el nodo de contexto y el presupuesto del historial se derivan de ahí. Bajarlo en máquinas muy justas de memoria es lo habitual. Ver [[specs/SPEC-OLLAMA-PERFIL]] y [[backend/DECISIONS]].

Sobre `LOCALCLI_ALLOW_INTERNET`: es la única integración que saca información de la máquina, así que no viene activada. El usuario la habilita. Ver [[backend/03-security/SECURITY]].

## 3. El modelo: se detecta, no se configura

**No hay variable de entorno para el modelo, y no hay modelo por defecto.** El flujo es:

1. `LocalCli` ya está conectado a Ollama. No hay que configurarlo a mano.
2. Extrae los modelos disponibles con `ollama list`.
3. La interfaz muestra esa lista para que el usuario elija.
4. El harness comprueba si el modelo elegido cabe en la máquina y avisa si no.

El modelo lo elige el usuario, siempre. Ver [[specs/SPEC-OLLAMA-PERFIL]] y [[backend/04-infrastructure/INTEGRATIONS]].

El tamaño de contexto no se configura por modelo: cada petición lo declara (`num_ctx`) como la menor entre la ventana que reporta el modelo en `/api/tags` y el tope (`LOCALCLI_CONTEXT_LIMIT`, 16384 por defecto). Es lo que evita que Ollama use su valor de servidor (pequeño) y corte los turnos con herramientas. Ver [[backend/DECISIONS]].

## 4. Rutas

Todo se deriva de la carpeta desde la que se ejecuta `localcli`:

| Qué | Dónde |
|---|---|
| Carpeta del proyecto | La carpeta desde la que se ejecuta `localcli` |
| Archivo SQLite | `.localcli/state.db` dentro del proyecto, o `LOCALCLI_DB_PATH` si se sobrescribe |
| Documentación | `ai/docs/` dentro del proyecto |
| TODO de trabajo | `ai/tasks/` dentro del proyecto |
| Herramientas del usuario | `.localcli/tools/*.json` dentro del proyecto |
| Archivos y carpetas | Todo lo que cuelgue de la carpeta del proyecto |

El proyecto es la carpeta abierta. El chat de una carpeta nunca aparece en otra. Ver [[specs/SPEC-SESIONES]].

El archivo SQLite va en `.localcli/state.db` dentro del proyecto, y `.localcli/` está fuera de git. La ruta se deriva de la carpeta abierta y no necesita variables; `LOCALCLI_DB_PATH` existe solo para los casos raros, como trabajar con la base en otro sitio. Ver [[backend/DECISIONS]].

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
| Ollama, corriendo en local | Generar respuestas y listar los modelos disponibles | El harness no puede generar nada; la interfaz sigue viva |
| Landlock (Linux 5.13+) | El bloqueo estructural de escritura en la terminal | La terminal sigue funcionando, con garantía más débil, y el harness lo dice |

No hay más servicios. No hay base de datos que levantar, ni migraciones que aplicar a mano, ni cuentas que crear. Ver [[specs/SPEC-OLLAMA-PERFIL]].

## 7. Preferencias del usuario

Además de `~/.config/localcli/keys.json` (el mapa de teclas), el usuario tiene sus preferencias en `~/.config/localcli/config.json`:

```json
{
  "ultimo_modelo": "llama3.2",
  "ultimo_agente": "build",
  "historial_tokens": 4096
}
```

- Son **globales del usuario**, no de un proyecto: viven fuera de la carpeta y no se versionan.
- Al arrancar se reutiliza `ultimo_modelo` si sigue instalado en Ollama; si no, se autodetecta. El agente recordado lo aplica la vista.
- `historial_tokens` es el presupuesto de tokens del historial de conversación que se le entrega al modelo; sin él se usa `LOCALCLI_CONTEXT_LIMIT` y, en su defecto, un valor por defecto. Ver [[specs/SPEC-HISTORIAL-CONVERSACION]].
- Un archivo ausente o ilegible no rompe el arranque: se usan los valores por defecto (`plan` como agente, autodetección como modelo).
- La interfaz los escribe al elegir modelo o al cambiar de agente; no hay que editar el archivo a mano. Ver [[specs/SPEC-OLLAMA-PERFIL]].

## Referencias

- [[backend/02-interfaces/TOOLS]] — el contrato de una herramienta del usuario y cómo se ejecuta.
- [[specs/SPEC-TOOLS]] — la especificación funcional del catálogo.
- [[backend/04-infrastructure/INTEGRATIONS]] — cómo se habla con Ollama y cómo se listan los modelos.
- [[backend/03-security/SECURITY]] — los límites de lo que puede hacer una herramienta.
- [[backend/DECISIONS]] — decisiones de configuración ya cerradas.
- [[PROJECT]] — requisitos en la máquina y despliegue.
