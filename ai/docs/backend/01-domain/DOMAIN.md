---
title: LocalCli — módulos y entidades del orquestador
tags: [backend, dominio]
depende_de:
  - "[[backend/BACKEND]]"
  - "[[backend/DECISIONS]]"
relacionado:
  - "[[database/01-schema/TABLES]]"
  - "[[database/01-schema/RELATIONSHIPS]]"
  - "[[database/01-schema/SCHEMA]]"
  - "[[database/02-rules/DATA_FLOW]]"
  - "[[backend/01-domain/BUSINESS_RULES]]"
  - "[[backend/03-security/SECURITY]]"
---
# DOMAIN — Módulos y entidades

Los quince módulos del orquestador y las entidades sobre las que trabajan. Los datos no se repiten aquí: la definición de cada tabla está en [[database/01-schema/TABLES]].

## 1. Módulos

Cada módulo tiene una responsabilidad y un límite. Si un módulo necesita hacer el trabajo de otro, es una señal de que sobra un módulo o de que el límite está mal puesto.

| Módulo | Responsabilidad | Lo que no hace |
|---|---|---|
| `tui` | La pantalla: chat, panel de datos, selector de sesiones, aprobaciones, razonamiento en vivo | No decide nada ni habla con el modelo; pinta lo que llega |
| `session` | Ciclo de vida de las sesiones y su estado; ejecución en segundo plano | No ejecuta tareas; eso es `flow` y `queue` |
| `flow` | Orquestador de etapas: encadenar, decidir si sigue, parar o esperar permiso; lanzar el flujo pedido por comando del usuario o por la herramienta `ejecutar_flujo` | No sabe de herramientas ni de SQL; `ejecutar_flujo` vive en el catálogo de `tools` y su handler se le inyecta a `flow` |
| `queue` | Cola de la ejecución en curso: consume el TODO de la ejecución que el usuario arrancó, en orden, y marca lo bloqueado | No inventa trabajo ni arranca solo; el TODO lo propuso el flujo y el usuario lo ejecuta |
| `context` | Grafo de frontmatter, selección de lo relevante, recorte y auditoría | No llama al modelo por su cuenta; pide la decisión y aplica |
| `agent` | Cargar las definiciones de agente desde carpeta: `agent.yaml` y `prompt.md`. Corre el ciclo conversacional del agente. El agente base son datos en disco, no código | No ejecuta herramientas; las despacha a `tools` |
| `llm` | Frontera neutra con el modelo: tipos (`Mensaje`, `Herramienta`, `Evento`, `Modelo`), capacidades de tres estados, una cola de inferencia única y regla de la ventana | No conoce ningún endpoint; no sabe de tareas; no instancia adaptadores: el motor es único (llama.cpp) |
| `openai`, `llamacpp` | El motor, en dos piezas: el **núcleo común** (`openai`) habla el protocolo OpenAI con `llama-server`; la extensión (`llamacpp`) aporta **solo** lo que el protocolo no cubre (`/props`) | `openai` no sabe de runtimes; la extensión no decide ni traduce la conversación |
| `tools` | Registro de las diecisiete herramientas, comprobación de permiso, validación, recorte, medición y eventos; envolver toda ejecución | No inventa herramientas ni las aplica |
| `fileops` | Operaciones de archivo y carpeta, frontera de rutas, historial de cambios | No decide permisos; comprueba y aplica |
| `exec` | Terminal: confirmación selectiva y bloqueo estructural de escritura | No es una puerta trasera a los archivos del proyecto |
| `store` | Único acceso a SQLite: esquema, WAL, transacciones | No decide nada de negocio |
| `docs` | Cargar documentación, leer frontmatter, construir el grafo | No guarda nada |
| `task` | Leer y escribir el TODO de la ejecución: sus elementos, su orden y su estado | No detecta qué hay que hacer; eso es `flow`, y no lo consume, eso es `queue` |

### Los tres que sostienen la garantía de escritura

`agent` reparte permisos, `tools` los comprueba y `fileops` los aplica. Separar las tres cosas es lo que hace auditable la garantía: puedes leer en un solo sitio quién podía pedir, quién autorizó y quién ejecutó. `plan` no tiene herramientas de escritura en su catálogo, así que la garantía no depende de que nadie olvide comprobar un permiso. Ver [[backend/03-security/SECURITY]].

## 2. Entidades del Dominio

Dos planos. Ver [[database/02-rules/DATA_FLOW]].

### En archivos (fuente de verdad)

| Entidad | Dónde vive | Qué la define |
|---|---|---|
| Documento del proyecto | `ai/docs/**/*.md` | Frontmatter con `depende_de` (lo que hay que leer) y `relacionado` (lo que toca). Convención en [[PROJECT]] |
| TODO de la ejecución | `ai/tasks/**/MAIN-TASKS.md` | Elementos en orden: fases, tareas o pasos. Es la cola |
| Agente base | `~/.config/localcli/agents/<carpeta>/` | `agent.yaml` con `name`, `description` y `permissions`, más `prompt.md`. Sembrado desde el binario |
| Agentes y flujos propios | `<proyecto>/.localcli/agents/` y `<proyecto>/.localcli/flows/` | Misma forma; se cargan de forma aditiva y no sobrescriben los oficiales |

**El TODO no viene dado: se propone.** Cuando una petición tuya implica una lista ordenada de trabajo, el sistema la detecta y **propone** el TODO; no lo ejecuta hasta que tú lo confirmas o escribes `/ejecutar`. Por eso la cola no es global: cada petición ordenada tiene el suyo. Ver [[backend/01-domain/BUSINESS_RULES]].

**El agente base existe, pero no está hardcodeado.** No está en Go: se siembra desde el binario en `~/.config/localcli/agents/` y vive en archivos de texto que el usuario puede modificar, y de los que puede derivar otros agentes (en su carpeta de configuración o en `<proyecto>/.localcli/agents/`). La configuración y las instrucciones van separadas: del `agent.yaml` salen el nombre y las herramientas disponibles, y del `prompt.md` el mensaje de sistema. Ver [[specs/SPEC-AGENTE-BASE]].

**No hay skills.** No es que no haya ninguna por defecto: es que la capacidad no existe. Un agente no declara skills y no hay orquestador que las lea. Ver [[specs/SPEC-SKILLS]].

El grafo de dependencias entre documentos no se guarda: se construye en memoria al arrancar, leyendo el frontmatter. Ver [[backend/01-domain/BUSINESS_RULES]].

### En SQLite (estado de ejecución)

| Entidad | Para qué | Definición completa |
|---|---|---|
| `projects` | La carpeta abierta; es la raíz de las sesiones de ese directorio | [[database/01-schema/TABLES]] |
| `sessions` | La unidad de trabajo; lleva nombre, proyecto, capa y estado | [[database/01-schema/TABLES]] |
| `messages` | Turnos de conversación, con los tokens que consumieron | [[database/01-schema/TABLES]] |
| `reasoning` | Razonamiento del modelo, ligado a su mensaje | [[database/01-schema/TABLES]] |
| `approvals` | Lo que espera tu decisión | [[database/01-schema/TABLES]] |
| `context_audit` | Qué documentación entró y salió de cada etapa, y por qué | [[database/01-schema/TABLES]] |
| `change_history` | Qué se cambió en tus archivos, con el antes y el después | [[database/01-schema/TABLES]] |

## 3. Relaciones de Dominio

A nivel de negocio, la cadena de un turno es siempre la misma:

```
tui → session → flow → context → agent → llm → openai / llamacpp
                                    ↓
                            agent pide herramienta
                                    ↓
                        tools → fileops / exec
                                ↓
                  tools → flow (ejecutar_flujo)
                                    ↓
                        fileops → store (change_history)
```

- **`ejecutar_flujo` es una herramienta de `tools` cuyo handler se inyecta a `flow`.** Una tool más para el modelo y para la capa universal (permiso, validación, recorte, eventos), pero su ejecución invoca al orquestador: corre las etapas como sub-proceso y devuelve la entrega más el bloque resumido. Por eso `plan` puede lanzar flujos que no escriben y no tener jamás un flujo que escriba a la vista: el permiso se deriva del flujo en el catálogo, antes de llegar a `flow`.

- **`session` contiene `messages`, `messages` puede tener `reasoning`.** El razonamiento pertenece a un mensaje de agente, y como máximo hay uno. Ver [[database/01-schema/RELATIONSHIPS]].
- **`session` contiene `approvals`.** Una sesión esperando permiso tiene al menos una aprobación pendiente, y esa fila es la que se ve en el panel.
- **`session` produce filas en `context_audit`.** Una por documento y etapa: lo que entró, lo que salió y el motivo.
- **`change_history` referencia a `session`, pero sobrevive a la sesión.** Si borras la sesión, `session_id` queda a NULL y el registro sigue. Es la relación más importante del modelo porque separa lo desechable de lo que no se puede perder.
- **`queue` no tiene tabla propia.** La cola es una proyección del TODO de trabajo, reconstruida al arrancar. `queue` lee con `task` y guarda su vista en memoria, no en la base. Ver [[backend/01-domain/BUSINESS_RULES]].

### Relaciones que no existen y no deben aparecer

- **No hay relación entre sesiones.** Dos sesiones del mismo proyecto no se ven, ni comparten contexto, ni comparten cola. Cada una es un mundo cerrado.
- **No hay tabla de cola.** Si aparece una tabla con el estado de la cola, es un error: contradice la decisión de derivarla de los archivos. Ver [[backend/DECISIONS]].
- **No hay tabla de grafo.** Las dependencias de la documentación viven en el frontmatter de los archivos, no en filas.
- **No hay tabla de usuarios ni de permisos.** El permiso es una propiedad del agente (`plan` o `build`) y un caso de uso, no una fila.

## Referencias

- [[backend/BACKEND]] — mapa de la capa.
- [[backend/01-domain/BUSINESS_RULES]] — reglas que gobiernan estos módulos.
- [[database/01-schema/SCHEMA]] — el esquema completo.
- [[database/01-schema/RELATIONSHIPS]] — relaciones a nivel de datos.
