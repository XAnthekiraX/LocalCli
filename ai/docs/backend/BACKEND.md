---
title: LocalCli — capa backend
tags: [backend, arquitectura]
depende_de:
  - "[[PROJECT]]"
  - "[[backend/DECISIONS]]"
relacionado:
  - "[[frontend/FRONTEND]]"
  - "[[database/DATABASE]]"
---

# BACKEND — LocalCli

Capa de motor de `LocalCli`: un proceso único de terminal, en Go, que ejecuta trabajos de varias etapas por ti. No es un servidor web: no expone HTTP, no tiene usuarios, no tiene login. La TUI (`tui`) es su único consumidor.

Este documento es el mapa de navegación del backend. Para los datos, mira [[database/DATABASE]]; para la pantalla, [[frontend/FRONTEND]].

## 1. Visión General

El backend se divide en dos planos, y la separación es deliberada:

- **Lo que vive en archivos.** La documentación del proyecto y los archivos de tarea. Se editan a mano, se versionan en git, se leen sin la herramienta. Fuentes de verdad. Ver [[backend/01-domain/DOMAIN]].
- **Lo que vive en SQLite.** El estado de ejecución: sesiones, mensajes, razonamiento, aprobaciones, auditoría de contexto e historial de cambios. Ver [[database/DATABASE]].

Sobre esos dos planos trabajan trece módulos, cada uno con una responsabilidad única:

| Grupo | Módulos | Qué hacen |
|---|---|---|
| Interfaz y ciclo | `tui`, `session`, `flow` | Reciben lo que escribes, encadenan etapas y muestran el resultado |
| Contexto e inteligencia | `context`, `agent`, `ollama` | Deciden qué se le entrega al modelo, quién es el agente y cómo se le habla |
| Ejecución | `tools`, `fileops`, `exec` | Aplican lo que el agente pide, dentro de las fronteras permitidas |
| Trabajo en serie | `queue`, `task` | Ordenan las tareas grandes y las ejecutan solas |
| Datos | `store`, `docs` | Persisten y leen lo que hace falta para saber qué está pasando |

## 2. Estructura del Proyecto

```
localcli           comando global para arrancar la aplicación
internal/
  tui/             pantalla Bubble Tea: chat, panel, selector, aprobaciones
  session/         ciclo de vida de sesiones y ejecución en segundo plano
  flow/            motor de etapas y encadenamiento
  queue/           cola de trabajo en forma de TODO y su orden
  context/         grafo de frontmatter, selección de contexto y auditoría
  agent/           agentes desde JSON: prompt, herramientas y relevo
  ollama/          cliente HTTP, streaming, razonamiento, perfil de hardware
  task/            archivos de tarea
  tools/           registro de herramientas y comprobación de permisos
  fileops/         validación de rutas, acceso a archivos e historial
  exec/            terminal: lista blanca y bloqueo estructural
  store/           persistencia SQLite
  docs/            carga de documentación y lectura de frontmatter
```

## 3. Responsabilidades de los Componentes

Cada módulo tiene un límite. Ver [[backend/01-domain/DOMAIN]] para el detalle.

- **`tui`** — Toda la pantalla. Recibe teclado, muestra chat, panel de datos, selector de sesiones, aprobaciones y razonamiento en vivo. No decide nada: solo pinta lo que le llega y manda lo que pulsas.
- **`session`** — Ciclo de vida de las sesiones: crear, cambiar, retomar, cerrar, y su estado. Corre en segundo plano aunque cambies de vista. No ejecuta tareas; eso es `flow` y `queue`.
- **`flow`** — Motor de etapas. Encadena las etapas de los ciclos oficiales (planificación, trabajo, resolver), decide si sigue, si para o si espera tu aprobación. No sabe de herramientas ni de SQL.
- **`queue`** — Cola de la ejecución en curso. Consume el TODO que creó esta solicitud, respeta su orden y marca lo bloqueado. No inventa trabajo: el TODO se lo creó `flow` al detectar tu petición.
- **`context`** — Nodo de contexto. Lee el grafo de dependencias, deja que el modelo decida qué es relevante, recorta hasta el límite y registra qué entró y qué salió. No llama al modelo por su cuenta.
- **`agent`** — Carga las definiciones de agente desde JSON: prompt, herramientas y relevo. El agente base es un JSON editable, no código. Las skills no vienen de fábrica; las crea el usuario en markdown. No ejecuta herramientas; las despacha a `tools`.
- **`ollama`** — Cliente de Ollama: HTTP en local, streaming token a token, extracción de razonamiento y perfil de hardware. Es el único que habla con el modelo.
- **`tools`** — Registro de las trece herramientas, comprobación de permiso y enrutado. No inventa herramientas y no aplica permisos: solo enruta.
- **`fileops`** — Aplica operaciones de archivo y carpeta, valida la frontera de rutas y guarda el historial de cambios. Donde vive la frontera de la carpeta del proyecto.
- **`exec`** — Terminal. Lista blanca de comandos y bloqueo estructural de escritura con Landlock. Lo único que puede lanzar procesos.
- **`store`** — Único acceso a SQLite: esquema, WAL y transacciones. Ningún otro módulo escribe en la base.
- **`docs`** — Carga la documentación del proyecto, lee el frontmatter y construye el grafo de dependencias. No guarda nada.
- **`task`** — Lee y escribe el TODO de la ejecución: elementos, orden y estado.

## 4. Relaciones entre Componentes

Los módulos se comunican por canales de Go, no por red. Un solo proceso. La forma exacta de cada contrato entre módulos está en [[backend/02-interfaces/INTERFACES-GENERAL]].

- `tui` → `session`: la TUI pide a la sesión activa que mande lo escrito; pide cambiar de sesión, aprobar o declinar.
- `session` → `flow`: al arrancar un flujo, la sesión pide al motor que encadene etapas.
- `flow` → `context`: cada etapa pide su contexto por un objetivo concreto.
- `flow` → `agent`: el motor dice qué agente corre (`plan` o `build`) en cada etapa.
- `context` → `ollama` y `store`: para pedir al modelo qué documentos necesita, y para registrar la auditoría.
- `agent` → `ollama`: el agente construye la llamada con su prompt y la envía.
- `agent` → `tools`: el agente pide una herramienta; `tools` comprueba permiso y enruta.
- `tools` → `fileops` o `exec`: las de archivo van a `fileops`; la de terminal, a `exec`.
- `fileops` → `store`: cada cambio aplicado se registra en `change_history`.
- `queue` → `flow`: la cola toma la siguiente tarea y el motor la ejecuta.
- `docs` → `context` y `task`: proveen el grafo y las tareas.
- `store` es transversal: lo usan `session`, `context`, `fileops` y `queue`.

### Concurrencia

Una goroutine por sesión en ejecución, con el motor de etapas encima. Los canales de Go hacen de cola entre la TUI y el trabajo de fondo. SQLite en WAL para que la interfaz pueda leer mientras las sesiones de fondo escriben. Las sesiones concurrentes comparten un único modelo en Ollama, así que sus respuestas se serializan: mientras una genera, la otra espera. Es una consecuencia del hardware, no del diseño. Ver [[backend/04-infrastructure/INTEGRATIONS]].

## Stack

- Lenguaje: Go, binario único, sin runtime externo.
- Framework: ninguno para el backend. La TUI usa Bubble Tea + Lip Gloss.
- ORM: ninguno. `store` habla SQL directo con el driver puro Go.
- Base de datos: SQLite en modo WAL, un archivo por proyecto, vía `modernc.org/sqlite`.
- Otras dependencias: cliente HTTP estándar para Ollama y para internet; Landlock del sistema para el bloqueo de escritura en la terminal.
- Aislamiento: Landlock en Linux da garantía fuerte; en otros sistemas la garantía es más débil y queda documentada como tal. Ver [[backend/03-security/SECURITY]].

## Referencia rápida

- Módulos y entidades → [[backend/01-domain/DOMAIN]]
- Reglas del motor → [[backend/01-domain/BUSINESS_RULES]]
- Contratos entre módulos y superficies → [[backend/02-interfaces/INTERFACES-GENERAL]]
- Las trece herramientas → [[backend/02-interfaces/TOOLS]]
- Seguridad y permisos → [[backend/03-security/SECURITY]]
- Configuración → [[backend/04-infrastructure/CONFIGURATION]]
- Integraciones externas → [[backend/04-infrastructure/INTEGRATIONS]]
- Eventos internos → [[backend/04-infrastructure/EVENTS]]
- Pruebas → [[backend/05-quality/TESTING]]
- Validación de entradas → [[backend/05-quality/VALIDATION]]
- Errores → [[backend/05-quality/ERRORS]]
- Decisiones de la capa → [[backend/DECISIONS]]

## Mapa de Navegación

Guía de lectura para la IA según la tarea:

- Implementar o cambiar una etapa → este documento, [[backend/01-domain/DOMAIN]], [[backend/02-interfaces/INTERFACES-GENERAL]].
- Añadir o revisar una herramienta → [[backend/02-interfaces/TOOLS]], [[backend/03-security/SECURITY]].
- Cambiar qué recibe el modelo → [[backend/01-domain/DOMAIN]], [[database/02-rules/DATA_FLOW]].
- Tocar la persistencia → [[database/DATABASE]], [[database/01-schema/SCHEMA]].
- Tocar la cola o el encadenamiento → [[backend/01-domain/BUSINESS_RULES]], [[backend/04-infrastructure/EVENTS]].
- Cambiar permisos, aprobación o aislamiento → [[backend/03-security/SECURITY]].

Al implementar una herramienta o una etapa, leer la subcadena necesaria:

```
BACKEND.md → DOMAIN → INTERFACES-GENERAL → TOOLS → BUSINESS_RULES
```

## Referencias

- [[PROJECT]] — decisiones técnicas ya aprobadas, no se reinventan.
- [[database/DATABASE]] — la capa de datos, no se duplica aquí.
