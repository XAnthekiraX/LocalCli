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
  - "[[specs/SPEC-MODELO-PROVEEDOR]]"
---
# TOOLS-DTO — Payloads de las herramientas

Los cuerpos de petición y respuesta de cada herramienta. `LocalCli` no tiene HTTP, así que esto no son cuerpos JSON: son los argumentos que el modelo pasa a la herramienta y lo que esta devuelve. Los tipos son los de Go en la frontera. Ver [[backend/02-interfaces/TOOLS]] para el comportamiento y [[backend/05-quality/VALIDATION]] para las reglas de validación.

## 1. Recurso

Pertenece al recurso **herramientas del agente**, que es la superficie que consume el modelo a través de `agent` y `tools`. Ver [[specs/SPEC-TOOLS]].

## 2. Los tipos son la fuente del esquema

Cada herramienta declara su petición como un tipo de Go, y **el JSON Schema que ve el modelo se deriva de ese tipo por reflexión**. No hay un esquema escrito a mano al lado: si los dos existieran, divergirían.

```go
type PeticionLeerArchivo struct {
    Ruta string `json:"ruta" desc:"Ruta relativa a la carpeta del proyecto"`
}
```

derivaría en:

```json
{
  "type": "object",
  "properties": {
    "ruta": {
      "type": "string",
      "description": "Ruta relativa a la carpeta del proyecto"
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
| `leer_archivo` | `ruta` | string | Sí | Ruta relativa a la carpeta del proyecto |
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
| `escribir_archivo` | `ruta` | string | Sí | Ruta del archivo a sobrescribir |
| `escribir_archivo` | `contenido` | string | Sí | Texto que reemplaza el contenido entero |
| `editar_archivo` | `ruta` | string | Sí | Ruta del archivo a modificar |
| `editar_archivo` | `cambio` | string | Sí | La edición a aplicar |
| `eliminar_archivo` | `ruta` | string | Sí | Ruta del archivo a borrar; pide confirmación explícita |
| `crear_carpeta` | `ruta` | string | Sí | Ruta de la carpeta; falla si ya existe |
| `eliminar_carpeta` | `ruta` | string | Sí | Ruta de la carpeta; pide confirmación explícita |
| *(las seis)* | `motivo` | string | No | Solo si la ruta sale de la carpeta del proyecto: explica en una línea por qué la buscas |

### Terminal

| Herramienta | Campo | Tipo | Obligatorio | `desc` |
|---|---|---|---|---|
| `ejecutar_comando` | `comando` | string | Sí | Comando a correr; se valida contra la lista blanca, no contra su texto |
| `ejecutar_comando` | `carpeta` | string | No | Carpeta de trabajo; por defecto, la del proyecto |

### Internet

| Herramienta | Campo | Tipo | Obligatorio | `desc` |
|---|---|---|---|---|
| `buscar_en_internet` | `consulta` | string | Sí | Qué buscar; es lo único que sale de la máquina |
| `abrir_pagina` | `direccion` | string | Sí | Dirección de la página a abrir |

### Sesión

| Herramienta | Campo | Tipo | Obligatorio | `desc` |
|---|---|---|---|---|
| `crear_todo` | `contenido` | string | Sí | Qué hay que hacer; una acción concreta |
| `crear_todo` | `estado` | string | Sí | `pendiente` \| `en_progreso` \| `completada` \| `cancelada` |
| `crear_todo` | `prioridad` | string | No | `alta` \| `media` \| `baja` (opcional; por defecto `media`) |
| `actualizar_todo` | `elementos` | array de objeto | Sí | La lista completa de pasos; reemplaza la anterior |
| `actualizar_todo` | `elementos[].contenido` | string | Sí | Qué hay que hacer; una acción concreta |
| `actualizar_todo` | `elementos[].estado` | string | Sí | `pendiente` \| `en_progreso` \| `completada` \| `cancelada` |
| `actualizar_todo` | `elementos[].prioridad` | string | No | `alta` \| `media` \| `baja` (opcional; por defecto `media`) |

`crear_todo` lleva un paso, no una lista: el esquema es un objeto plano y **no tiene campo `elementos`**. Su argumento es deliberadamente más pequeño que el de `actualizar_todo`, porque no puede tocar lo que ya está: añadir y sustituir son contratos distintos, no el mismo con un atajo.

`elementos` es un `array` cuyo `items` expone el objeto de cada paso: el esquema anida `contenido`, `estado` y `prioridad`, no un `object` vacío. Lo que llega es la lista entera, no un delta: una lista vacía deja la lista en blanco. La capa universal comprueba que el estado y la prioridad estén en su vocabulario, en las dos herramientas, y un valor fuera vuelve al modelo como `E_BAD_ARGS` corregible. En `crear_todo` la comprobación incluye que no haya ya un paso `en_progreso` —regla que hasta ahora solo era instrucción en la descripción—: el error nombra `actualizar_todo` como la vía para cambiarlo. Ver [[specs/SPEC-TOOLS]].

### Del usuario

El tipo de petición de una herramienta del usuario es genérico, no declarado: acepta cualquier objeto. El modelo ve `{"type": "object"}` y lo que el ejecutable entienda.

Esto es una consecuencia honesta de que la herramienta sea código del usuario y no de LocalCli: **el harness no puede saber qué argumentos espera**. Los valida contra la forma de un objeto, nada más.

| Herramienta | Campo | Tipo | Obligatorio | `desc` |
|---|---|---|---|---|
| `<cualquiera>` | — | object | — | Lo que la herramienta espere; LocalCli no lo valida |

Es el precio de que el usuario escriba la herramienta en vez de elegirla de un catálogo. Queda dicho aquí para que nadie lo descubra en producción: el esquema no describe los argumentos de una herramienta del usuario, solo que el argumento es un objeto.

## 4. Response Schemas

Lo que cada herramienta devuelve al agente. El agente lo ve en su respuesta, así que el contenido importa: va al mismo presupuesto de contexto que el resto.

Todas las respuestas comparten envoltura: `Resultado{Salida, Meta, Truncado, Error}`. Lo que las tablas de abajo describen es el contenido de `Salida`.

El tamaño de `Salida` no viaja al modelo, pero sí a la línea del chat: cada herramienta declara su **unidad** ([[backend/02-interfaces/TOOLS]] §1) y el harness cuenta sus líneas con contenido. `leer_archivo`, `ejecutar_comando` y `abrir_pagina` se miden en líneas; `listar_carpeta` en entradas; `buscar_archivos` y `buscar_en_archivos` en coincidencias; `buscar_en_internet` en resultados; `actualizar_todo` en pasos. Las escrituras no tienen medida.

**La duración de la ejecución tampoco viaja al modelo, y por eso no está en `Resultado`.** Es un dato de pantalla: lo mide la capa universal y sale en el evento `herramienta_resultado` —campo `duracion`— y de ahí a la línea del chat y a `chat_evento.duration_ms`. Que no forme parte del payload que el modelo ve es deliberado: al modelo le importa si la herramienta funcionó y qué devolvió, no cuánto tardó, y un número de más en cada resultado es contexto que se paga en cada turno. Ver [[backend/04-infrastructure/EVENTS]] y [[database/01-schema/TABLES]] §`chat_evento`.

### Lectura de archivos

| Herramienta | Devuelve | Tipo |
|---|---|---|
| `leer_archivo` | El contenido del archivo | string |
| `listar_carpeta` | Las entradas de un nivel | lista de nombres |
| `buscar_archivos` | Las rutas que coinciden | lista de rutas |
| `buscar_en_archivos` | Las coincidencias | lista de (archivo, línea, fragmento) |

### Escritura de archivos

| Herramienta | Devuelve | Tipo |
|---|---|---|
| `crear_archivo` | Confirmación y la ruta | string |
| `escribir_archivo` | Confirmación y la ruta | string |
| `editar_archivo` | Confirmación y la ruta | string |
| `eliminar_archivo` | Confirmación | string |
| `crear_carpeta` | Confirmación y la ruta | string |
| `eliminar_carpeta` | Confirmación | string |

Toda escritura aprobada queda registrada en `change_history` con el antes y el después. La herramienta no devuelve ese historial al agente; vive en la base. Ver [[database/01-schema/TABLES]].

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

Las dos devuelven lo mismo a propósito: la lista entera, en el mismo formato. El modelo no tiene que distinguir «esto es lo que había más lo que añadí» de «esto es la lista nueva», y una herramienta de añadir no necesita inventar un formato de respuesta propio.

El modelo ve la lista resultante en el propio resultado, así que no necesita una herramienta de lectura. La lista además se persiste por sesión y se anuncia al panel con el evento `todo_actualizada`. Ver [[backend/04-infrastructure/EVENTS]] y [[database/01-schema/TABLES]].

### Del usuario

| Campo | Tipo | Notas |
|---|---|---|
| `salida` | string | Lo que el ejecutable escribió en su salida estándar |
| `codigo` | int | Código de salida del ejecutable |
| `truncado` | bool | Si la salida se cortó por tamaño o tiempo |
| `termino` | bool | Si el ejecutable terminó |

La salida de error del ejecutable **no** se distingue de la normal: se concatena a `Salida`, igual que hace la terminal. Es el mismo mecanismo, y el modelo no necesita saber de dónde vino.

## Referencias

- [[backend/02-interfaces/TOOLS]] — el catálogo, la capa universal y su comportamiento.
- [[backend/05-quality/VALIDATION]] — qué campos son obligatorios y cómo se validan.
- [[specs/SPEC-TOOLS]] — la especificación funcional y las herramientas del usuario.
- [[specs/SPEC-MODELO-PROVEEDOR]] — el canal por el que viajan los esquemas.
- [[database/01-schema/TABLES]] — dónde queda lo que las herramientas de escritura registran.

