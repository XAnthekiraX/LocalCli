---
title: LocalCli — payloads de las herramientas
tags: [backend, interfaces]
depende_de:
  - "[[backend/02-interfaces/TOOLS]]"
  - "[[specs/SPEC-TOOLS]]"
relacionado:
  - "[[backend/05-quality/VALIDATION]]"
  - "[[backend/01-domain/DOMAIN]]"
  - "[[backend/05-quality/ERRORS]]"
  - "[[database/01-schema/TABLES]]"
  - "[[specs/SPEC-MODELOS-LLAMACPP]]"
---
# TOOLS-DTO — Payloads de las herramientas

Los cuerpos de petición y respuesta de cada herramienta. `LocalCli` no tiene HTTP, así que esto no son cuerpos JSON: son los argumentos que el modelo pasa a la herramienta y lo que esta devuelve. Los tipos son los de Go en la frontera. Ver [[backend/02-interfaces/TOOLS]] para el comportamiento y [[backend/05-quality/VALIDATION]] para las reglas de validación.

## 1. Recurso

Pertenece al recurso **herramientas del agente**, que es la superficie que consume el modelo a través de `agent` y `tools`. Ver [[specs/SPEC-TOOLS]].

## 2. Los tipos son la fuente del esquema

Cada herramienta declara su petición como un tipo de Go, y **el JSON Schema que ve el modelo se deriva de ese tipo por reflexión**. No hay un esquema escrito a mano al lado: si los dos existieran, divergirían.

```go
type PeticionLeerArchivo struct {
    Ruta        string `json:"ruta"                   desc:"Ruta relativa al directorio de trabajo del harness; conserva las subcarpetas (docs/specs/X.md), no la ruta absoluta completa ni ~ o $HOME"`
    InicioLinea int    `json:"inicio_linea,omitempty" desc:"Línea inicial, contando desde 1"`
    FinLinea    int    `json:"fin_linea,omitempty"    desc:"Línea final, inclusive"`
    Motivo      string `json:"motivo,omitempty"       desc:"Solo si la ruta sale de la carpeta del proyecto: explica en una línea por qué la buscas"`
}
```

derivaría en:

```json
{
  "type": "object",
  "properties": {
    "ruta": {
      "type": "string",
      "description": "Ruta del archivo a leer"
    },
    "inicio_linea": {
      "type": "integer",
      "description": "Línea inicial, contando desde 1"
    },
    "fin_linea": {
      "type": "integer",
      "description": "Línea final, inclusive"
    },
    "motivo": {
      "type": "string",
      "description": "Solo si la ruta sale de la carpeta del proyecto: explica en una línea por qué la buscas"
    }
  },
  "required": ["ruta"]
}
```

Reglas de derivación:

- El nombre del campo es el del tag `json`.
- El tipo es el del campo: `string`, `bool`, número, `[]T` como arreglo, `map` como objeto.
- Un campo **sin** `omitempty` es obligatorio y va en `required`. Con `omitempty`, es opcional.
- `desc` es la descripción que ve el modelo. Si falta, el campo viaja sin descripción.

**`desc` está escrito para el modelo, no para quien lee el código.** Dice cuándo usar el argumento y qué significa, no cómo está implementado. Es la prosa más cara del sistema —cada definición viaja en cada petición al modelo— y por eso no se adorna.

Los tipos anidados siguen la misma regla en profundidad: un `[]Cambio` de `editar_archivo` expone el esquema de `Cambio` entero, no un `object` vacío.

La comprobación de que el esquema derivado coincide con el documentado aquí es un **test**, no una revisión manual. Si divergen, falla la suite.

## 3. Request Schemas

Los argumentos que el modelo envía. Todos obligatorios salvo lo marcado.

Todas las herramientas de archivos admiten además `motivo` (string, **opcional**): la explicación de por qué se busca esa ruta. Solo hace falta —y entonces es obligatoria— cuando la ruta sale de la carpeta del proyecto: la spec exige permiso **y** explicación, y sin ella la operación se rechaza con `E_PATH_OUTSIDE` corregible ([[specs/SPEC-ARCHIVOS]] §Reglas, [[backend/05-quality/ERRORS]] §3). El motivo queda visible en el panel de aprobaciones.

### Archivos de lectura

| Herramienta | Campo | Tipo | Obligatorio | `desc` |
|---|---|---|---|---|
| `leer_archivo` | `ruta` | string | Sí | Ruta del archivo; una relativa se resuelve desde el directorio de trabajo del harness. Conserva las subcarpetas (`docs/specs/X.md`), no la ruta absoluta completa; nunca empieces por `~` o `$HOME` |
| `leer_archivo` | `inicio_linea` | integer | No | Línea inicial, contando desde 1 |
| `leer_archivo` | `fin_linea` | integer | No | Línea final, inclusive |
| `leer_archivo` | `motivo` | string | No | Solo si la ruta sale de la carpeta del proyecto: explica en una línea por qué la buscas |
| `listar_carpeta` | `ruta` | string | Sí | Ruta relativa; un nivel |
| `listar_carpeta` | `motivo` | string | No | Solo si la ruta sale de la carpeta del proyecto: explica en una línea por qué la buscas |
| `buscar_archivos` | `patron` | string | Sí | Texto contra el que se comparan los nombres de archivo |
| `buscar_en_archivos` | `patron` | string | Sí | Texto a buscar dentro de los archivos |
| `buscar_en_archivos` | `ruta` | string | No | Limita la búsqueda a una subcarpeta |
| `buscar_en_archivos` | `motivo` | string | No | Solo si la ruta sale de la carpeta del proyecto: explica en una línea por qué la buscas |

### Archivos de escritura (solo `build`)

| Herramienta | Campo | Tipo | Obligatorio | `desc` |
|---|---|---|---|---|
| `crear_archivo` | `ruta` | string | Sí | Ruta del archivo a crear; falla si ya existe |
| `crear_archivo` | `contenido` | string | Sí | Texto completo del archivo |
| `escribir_archivo` | `ruta` | string | Sí | Archivo a escribir; debe estar vacío o no existir, y si ya tiene contenido usa `editar_archivo` |
| `escribir_archivo` | `contenido` | string | Sí | Texto completo del archivo |
| `editar_archivo` | `ruta` | string | Sí | Ruta del archivo a modificar |
| `editar_archivo` | `reemplazos` | array de `Cambio` | Sí | Lista no vacía de sustituciones aplicadas en orden; cada buscar debe coincidir exactamente una vez |
| `editar_archivo` · `reemplazos[]` | `buscar` | string | Sí | Fragmento copiado textualmente del archivo (misma sangría y espacios); debe aparecer una sola vez |
| `editar_archivo` · `reemplazos[]` | `reemplazo` | string | Sí | Texto que reemplaza al fragmento; usa finales de línea LF |
| `eliminar_archivo` | `ruta` | string | Sí | Ruta del archivo a borrar; pide confirmación explícita |
| `crear_carpeta` | `ruta` | string | Sí | Ruta de la carpeta; falla si ya existe |
| `eliminar_carpeta` | `ruta` | string | Sí | Ruta de la carpeta; pide confirmación explícita |
| *(las seis)* | `motivo` | string | No | Solo si la ruta sale de la carpeta del proyecto: explica en una línea por qué la buscas |

### Terminal

| Herramienta | Campo | Tipo | Obligatorio | `desc` |
|---|---|---|---|---|
| `ejecutar_comando` | `comando` | string | Sí | Comando a correr; se confirma si es destructivo, saca información a la red o toca una ruta de fuera, nunca por el texto del comando |
| `ejecutar_comando` | `carpeta` | string | No | Carpeta de trabajo; por defecto, la del proyecto |

### Internet

| Herramienta | Campo | Tipo | Obligatorio | `desc` |
|---|---|---|---|---|
| `buscar_en_internet` | `consulta` | string | Sí | Qué buscar; es lo único que sale de la máquina |
| `abrir_pagina` | `direccion` | string | Sí | Dirección de la página a abrir |

### Sesión

| Herramienta | Campo | Tipo | Obligatorio | `desc` |
|---|---|---|---|---|
| `crear_todo` | `contenido` | string | † | Qué hay que hacer; una acción concreta |
| `crear_todo` | `estado` | string | Sí | `pendiente` \| `en_progreso` \| `completada` \| `cancelada` |
| `crear_todo` | `prioridad` | string | No | `alta` \| `media` \| `baja` (opcional; por defecto `media`) |
| `crear_todo` | `titulo` | string | † | Título de la tarea principal |
| `crear_todo` | `subtareas` | array de objeto | † | Subtareas de la tarea principal; un solo nivel |
| `crear_todo` | `subtareas[].contenido` | string | Sí | Qué hay que hacer; una acción concreta |
| `crear_todo` | `subtareas[].estado` | string | Sí | `pendiente` \| `en_progreso` \| `completada` \| `cancelada` |
| `crear_todo` | `subtareas[].prioridad` | string | No | `alta` \| `media` \| `baja` (opcional; por defecto `media`) |
| `actualizar_todo` | `elementos` | array de objeto | Sí | La lista completa de pasos; reemplaza la anterior |
| `actualizar_todo` | `elementos[].contenido` | string | Sí | Qué hay que hacer; una acción concreta |
| `actualizar_todo` | `elementos[].estado` | string | Sí | `pendiente` \| `en_progreso` \| `completada` \| `cancelada` |
| `actualizar_todo` | `elementos[].prioridad` | string | No | `alta` \| `media` \| `baja` (opcional; por defecto `media`) |
| `actualizar_todo` | `elementos[].subtareas` | array de objeto | No | Subtareas del elemento; un solo nivel |

† Excluyentes: un paso usa `contenido`; una tarea principal usa `titulo` con `subtareas`. Los dos a la vez, o ninguno, es `E_BAD_ARGS`.

`crear_todo` lleva **un paso o una tarea principal, no una lista**: el esquema es un objeto plano y **no tiene campo `elementos`**. Un paso usa `contenido`; una tarea principal usa `titulo` con `subtareas` (los dos caminos son excluyentes y el `desc` instruye al modelo a elegir). El argumento es deliberadamente más pequeño que el de `actualizar_todo`, porque no puede tocar lo que ya está: añadir y sustituir son contratos distintos, no el mismo con un atajo.

`elementos` es un `array` cuyo `items` expone el objeto de cada paso: el esquema anida `contenido`, `estado` y `prioridad`, no un `object` vacío. Cada elemento puede llevar además `subtareas` (opcional, un solo nivel): los pasos que cuelgan de otro se declaran dentro de él, no uno junto al otro. Lo que llega es la lista entera, no un delta: una lista vacía deja la lista en blanco. La capa universal comprueba que el estado y la prioridad estén en su vocabulario, en las dos herramientas —y en las subtareas—, y un valor fuera vuelve al modelo como `E_BAD_ARGS` corregible. En `crear_todo` la comprobación incluye que no haya ya un paso `en_progreso` —regla que hasta ahora solo era instrucción en la descripción—: el error nombra `actualizar_todo` como la vía para cambiarlo. Ver [[specs/SPEC-TOOLS]].

### Flujos

| Herramienta | Campo | Tipo | Obligatorio | `desc` |
|---|---|---|---|---|
| `ejecutar_flujo` | `flujo` | string (enum) | Sí | El comando del flujo a lanzar; el enum se arma por turno y solo lista los flujos que el permiso derivado del agente activo admite |
| `ejecutar_flujo` | `peticion` | string | Sí | El objetivo del flujo, con tus palabras; es lo que viaja en la ventana como petición original |

El argumento `flujo` es un **enum en el esquema, no un nombre en el handler**: el catálogo de flujos se resuelve contra los cargados (oficiales y propios) y contra el permiso derivado de sus etapas —todas `plan` → `read`; alguna `build` → `write`/`edit`—. Fuera del enum, la llamada no se ejecuta: la herramienta no existe para ese agente. Ver [[specs/SPEC-ORQUESTADOR-FLUJOS]] §Invocación del flujo.

### Diagramas

| Herramienta | Campo | Tipo | Obligatorio | `desc` |
|---|---|---|---|---|
| `diagram` | `code` | string | Sí | Código Mermaid de un flowchart (`flowchart TD` o `LR`); nodos con ID y texto, formas `[rect]` `{decisión}` `((elipse))` y conexiones `A --> B` con etiqueta `-->|texto|` |

`diagram` tiene **un solo argumento**: no recibe título, orientación ni metadatos. La orientación sale de la cabecera del propio código (`TD` por defecto, `LR` horizontal) y el título, si se deriva, sale de la definición. El `desc` del campo acota la sintaxis admitida —el subconjunto documentado en [[specs/SPEC-DIAGRAMA]]— para que un modelo local pequeño no invente construcciones fuera de alcance.

La llamada tiene esta forma:

```json
{
  "code": "flowchart TD\n  A[Inicio] --> B{¿Sigo?}\n  B -->|Sí| C[Fin]"
}
```

`code` es un **string con Mermaid**, no un objeto con `nodes` y `edges`: el grafo se deduce del texto. No hay un esquema alternativo de nodos y conexiones.

## 4. Response Schemas

Lo que cada herramienta devuelve al agente. El agente lo ve en su respuesta, así que el contenido importa: va al mismo presupuesto de contexto que el resto.

Todas las respuestas comparten envoltura: `Resultado{Salida, Meta, Truncado, Error}`. Lo que las tablas de abajo describen es el contenido de `Salida`.

El tamaño de `Salida` no viaja al modelo, pero sí a la línea del chat: cada herramienta declara su **unidad** ([[backend/02-interfaces/TOOLS]] §1) y el harness cuenta sus líneas con contenido. `leer_archivo`, `ejecutar_comando` y `abrir_pagina` se miden en líneas; `listar_carpeta` en entradas; `buscar_archivos` y `buscar_en_archivos` en coincidencias; `buscar_en_internet` en resultados; `actualizar_todo` en pasos. Las escrituras no tienen medida.

**La duración de la ejecución tampoco viaja al modelo, y por eso no está en `Resultado`.** Es un dato de pantalla: lo mide la capa universal y sale en el evento `herramienta_resultado` —campo `duracion`— y de ahí a la línea del chat y a `chat_evento.duration_ms`. Que no forme parte del payload que el modelo ve es deliberado: al modelo le importa si la herramienta funcionó y qué devolvió, no cuánto tardó, y un número de más en cada resultado es contexto que se paga en cada turno. Ver [[backend/04-infrastructure/EVENTS]] y [[database/01-schema/TABLES]] §`chat_evento`.

### Lectura de archivos

| Herramienta | Devuelve | Tipo |
|---|---|---|
| `leer_archivo` | El contenido y los metadatos de la lectura | objeto: `ruta`, `contenido`, `linea_inicio`, `linea_fin`, `total_lineas`, `truncado`, `siguiente_linea` |

`leer_archivo` devuelve un objeto, no un string: además del texto, el modelo necesita saber qué tramo leyó y si quedó contenido fuera. `ruta` es la ruta ya resuelta en forma canónica —relativa al directorio de trabajo si el archivo está dentro, absoluta si está fuera—; `linea_inicio` y `linea_fin` son el rango realmente devuelto, numerado desde 1; `total_lineas` va cuando puede determinarse; `truncado` es `true` si se omitió contenido; y `siguiente_linea` es la línea desde la que continuar, cuando la hay. Devolver la forma corta en cada resultado refuerza el hábito del modelo de no pegar la ruta completa.

La lectura tiene dos límites **fijos en esta versión**, medidos en bytes: 64 KiB para devolver el archivo completo y 32 KiB como máximo por respuesta. Un archivo que no supera el primero se devuelve entero; uno que lo supera se devuelve como fragmento inicial, con `truncado` en `true` y `siguiente_linea` para continuar. Un rango se aplica **sobre el archivo original** —nunca sobre un fragmento ya leído— y, si no cabe en el máximo por respuesta, se devuelve hasta donde llega. Una línea individual demasiado grande se devuelve recortada y señalada.

La **resolución de rutas** es la misma que para `listar_carpeta`: las relativas se resuelven desde el directorio de trabajo del harness, `..` y las absolutas son válidas, no se expande `~` ni `$HOME` (una ruta que empieza por esos prefijos se rechaza con `E_BAD_ARGS` y un recordatorio de usar la forma relativa) y no se busca en otras ubicaciones. La normalización no elude las restricciones de acceso: una ruta que resuelve fuera de la carpeta sigue exigiendo permiso y explicación (`motivo`). La `ruta` de la respuesta es canónica: relativa cuando el archivo está dentro del directorio de trabajo, absoluta cuando queda fuera.

| `listar_carpeta` | Las entradas de un nivel | lista de nombres |
| `buscar_archivos` | Las rutas que coinciden | lista de rutas |
| `buscar_en_archivos` | Las coincidencias | lista de (archivo, línea, fragmento) |

### Escritura de archivos

| Herramienta | Devuelve | Tipo |
|---|---|---|
| `crear_archivo` | Confirmación y la ruta | string |
| `escribir_archivo` | Confirmación y la ruta; error corregible si el archivo ya tiene contenido | string |
| `editar_archivo` | Confirmación y la ruta | string |
| `eliminar_archivo` | Confirmación | string |
| `crear_carpeta` | Confirmación y la ruta | string |
| `eliminar_carpeta` | Confirmación | string |

Toda escritura aprobada queda registrada en `change_history` con el antes y el después. La herramienta no devuelve ese historial al agente; vive en la base. Ver [[database/01-schema/TABLES]].

Además de la confirmación, el resultado interno de `crear_archivo`, `escribir_archivo` y `editar_archivo` lleva en `Meta` el bloque de diferencia (`bloque` y `bloque_tipo = "diferencia"`) cuando el cambio se aplicó. No forma parte del contrato con el modelo: la capa universal lo adjunta al evento `herramienta_resultado` y no lo reenvía. Ver [[backend/02-interfaces/TOOLS]] §La diferencia que se ve y [[backend/04-infrastructure/EVENTS]] §3.

### Terminal

| Campo | Tipo | Notas |
|---|---|---|
| `salida` | string | Salida del comando, recortada al límite |
| `error` | string | Salida de error cuando la hay |
| `codigo` | int | Código de salida |
| `truncado` | bool | Si la salida se cortó por tamaño o tiempo |
| `termino` | bool | Si el comando terminó |

Un comando que sale con error no es un fallo de la herramienta: devuelve su salida de error y el agente sigue. Ver [[backend/05-quality/ERRORS]].

El `truncado` de aquí es el del handler, que tiene su propio límite. La capa universal respeta ese `true` y no vuelve a recortar.

### Internet

| Herramienta | Devuelve | Tipo |
|---|---|---|
| `buscar_en_internet` | Resultados | lista de (titulo, direccion, fragmento) |
| `abrir_pagina` | El contenido de la página | string |

Lo que vuelve entra al presupuesto de contexto, se recorta y se audita como cualquier otro documento. Ver [[backend/01-domain/DOMAIN]].

### Sesión

| Herramienta | Devuelve | Tipo |
|---|---|---|
| `crear_todo` | La lista de pasos resultante, como checklist de texto (`[ ]`, `[•]`, `[✓]`, `[x]`) | string |
| `actualizar_todo` | La lista de pasos resultante, como checklist de texto (`[ ]`, `[•]`, `[✓]`, `[x]`) | string |

Las dos devuelven lo mismo a propósito: la lista entera, en el mismo formato. El modelo no tiene que distinguir «esto es lo que había más lo que añadí» de «esto es la lista nueva», y una herramienta de añadir no necesita inventar un formato de respuesta propio. Los pasos que cuelgan de una tarea principal se listan anidados bajo su título, con sangría, para que el modelo lea la misma jerarquía que verá el panel.

El modelo ve la lista resultante en el propio resultado, así que no necesita una herramienta de lectura. La lista además se persiste por sesión y se anuncia al panel con el evento `todo_actualizada`. Ver [[backend/04-infrastructure/EVENTS]] y [[database/01-schema/TABLES]].

### Flujos

| Herramienta | Devuelve | Tipo |
|---|---|---|
| `ejecutar_flujo` | La respuesta de la etapa `entrega` más el bloque de contexto resumido | texto |

`ejecutar_flujo` devuelve **solo dos cosas**: la entrega del flujo —lo que la etapa `entrega` redactó, que es lo mismo que el chat mostraría como conversación en un flujo por comando— y una síntesis del bloque de contexto (una aportación por etapa, condensada). Es la **frontera de salida** deliberada: ni las etapas, ni las herramientas que corrieron dentro, ni la ventana del flujo llegan al agente que la invocó. El agente sabe qué flujo lanzó y qué entregó, no lo que pasó dentro. Ver [[specs/SPEC-ORQUESTADOR-FLUJOS]] §Invocación del flujo.

La duración que reporta la línea de chat es la de toda la ejecución del flujo —el `duracion` de `herramienta_resultado`—, no la de un handler corto.

### Diagramas

| Herramienta | Devuelve | Tipo |
|---|---|---|
| `diagram` | El diagrama renderizado, en texto | string |

`diagram` devuelve la **representación en texto del diagrama** —el mismo bloque Unicode que la TUI pinta en el hilo—, para que el modelo lo lea y pueda citarlo o corregirlo. Los nodos y las conexiones, con su ancho ya calculado, son lo único que vuelve: ni la `code` de entrada ni metadatos de layout viajan de vuelta. Un diagrama que no cabe en los límites de la spec, o que no parsea, no entra aquí: vuelve como error corregible con línea y ejemplo, igual que cualquier `E_BAD_ARGS`. Ver [[specs/SPEC-DIAGRAMA]].

La presentación como bloque visual es de la TUI, no del payload: el handler entrega texto y la interfaz lo dibuja ([[frontend/02-interfaces/INTERFACES]] §1.1).

## Referencias

- [[backend/02-interfaces/TOOLS]] — el catálogo, la capa universal y su comportamiento.
- [[backend/05-quality/VALIDATION]] — qué campos son obligatorios y cómo se validan.
- [[specs/SPEC-TOOLS]] — la especificación funcional del catálogo.
- [[specs/SPEC-DIAGRAMA]] — la especificación funcional de la herramienta de diagramas.
- [[specs/SPEC-MODELOS-LLAMACPP]] — el canal por el que viajan los esquemas.
- [[database/01-schema/TABLES]] — dónde queda lo que las herramientas de escritura registran.

