# DOMAIN — Módulos y entidades

Los trece módulos del motor y las entidades sobre las que trabajan. Los datos no se repiten aquí: la definición de cada tabla está en [[database/01-schema/TABLES]].

## 1. Módulos

Cada módulo tiene una responsabilidad y un límite. Si un módulo necesita hacer el trabajo de otro, es una señal de que sobra un módulo o de que el límite está mal puesto.

| Módulo | Responsabilidad | Lo que no hace |
|---|---|---|
| `tui` | La pantalla: chat, panel de datos, selector de sesiones, aprobaciones, razonamiento en vivo | No decide nada ni habla con el modelo; pinta lo que llega |
| `session` | Ciclo de vida de las sesiones y su estado; ejecución en segundo plano | No ejecuta tareas; eso es `flow` y `queue` |
| `flow` | Motor de etapas: encadenar, decidir si sigue, parar o esperar permiso | No sabe de herramientas ni de SQL |
| `queue` | Cola de la ejecución en curso: consume el TODO que creó esta solicitud, en orden, y marca lo bloqueado | No inventa trabajo; el TODO lo creó el flujo a partir de lo que pediste |
| `context` | Grafo de frontmatter, selección de lo relevante, recorte y auditoría | No llama al modelo por su cuenta; pide la decisión y aplica |
| `agent` | Cargar las definiciones de agente desde JSON: prompt, herramientas y relevo. El agente base es un JSON, no código | No ejecuta herramientas; las despacha a `tools` |
| `ollama` | Cliente HTTP con streaming, extracción de razonamiento, perfil de hardware | Es el único que habla con el modelo; no sabe de tareas |
| `tools` | Registro de las trece herramientas, comprobación de permiso, enrutado | No inventa herramientas ni las aplica |
| `fileops` | Operaciones de archivo y carpeta, frontera de rutas, historial de cambios | No decide permisos; comprueba y aplica |
| `exec` | Terminal: lista blanca y bloqueo estructural de escritura | No es una puerta trasera a los archivos del proyecto |
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
| Documento del proyecto | `ai/docs/**/*.md` | Frontmatter con etiquetas y enlaces que declaran dependencias |
| TODO de la ejecución | `ai/tasks/**/MAIN-TASKS.md` | Elementos en orden: fases, tareas o pasos. Es la cola |
| Agente base y derivados | `ai/agents/*.json` | Estructura fija, aún por confirmar |
| Skill | `ai/skills/**/*.md` | Instrucciones en markdown con frontmatter |

**El TODO no viene dado: se crea.** Cuando una petición tuya implica una lista ordenada de trabajo, el sistema la detecta y genera el TODO. Por eso la cola no es global: cada petición ordenada crea el suyo. Ver [[backend/01-domain/BUSINESS_RULES]].

**El agente base existe, pero no está hardcodeado.** No está en Go: vive en un JSON que el usuario puede modificar, y del que puede derivar otros agentes. Esa es la decisión tomada; los campos concretos del JSON se definirán más adelante. Ver [[backend/DECISIONS]].

**No hay skills por defecto.** Ni una. El usuario crea las suyas en markdown, como en opencode. El motor solo sabe leerlas y respetar lo que declaran. Ver [[backend/01-domain/BUSINESS_RULES]].

El grafo de dependencias entre documentos no se guarda: se construye en memoria al arrancar, leyendo el frontmatter. Ver [[backend/01-domain/BUSINESS_RULES]].

### En SQLite (estado de ejecución)

| Entidad | Para qué | Definición completa |
|---|---|---|
| `sessions` | La unidad de trabajo; lleva nombre, capa y estado | [[database/01-schema/TABLES]] |
| `messages` | Turnos de conversación, con los tokens que consumieron | [[database/01-schema/TABLES]] |
| `reasoning` | Razonamiento del modelo, ligado a su mensaje | [[database/01-schema/TABLES]] |
| `approvals` | Lo que espera tu decisión | [[database/01-schema/TABLES]] |
| `context_audit` | Qué documentación entró y salió de cada etapa, y por qué | [[database/01-schema/TABLES]] |
| `change_history` | Qué se cambió en tus archivos, con el antes y el después | [[database/01-schema/TABLES]] |

## 3. Relaciones de Dominio

A nivel de negocio, la cadena de un turno es siempre la misma:

```
tui → session → flow → context → agent → ollama
                                    ↓
                            agent pide herramienta
                                    ↓
                        tools → fileops / exec
                                    ↓
                          fileops → store (change_history)
```

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
