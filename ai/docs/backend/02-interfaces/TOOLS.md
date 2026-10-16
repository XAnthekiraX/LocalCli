---
title: LocalCli — catálogo de herramientas
tags: [backend, interfaces]
depende_de:
  - "[[specs/SPEC-TOOLS]]"
  - "[[backend/DECISIONS]]"
relacionado:
  - "[[specs/SPEC-DIAGRAMA]]"
  - "[[backend/02-interfaces/dto/TOOLS-DTO]]"
  - "[[backend/01-domain/DOMAIN]]"
  - "[[backend/02-interfaces/INTERFACES-GENERAL]]"
  - "[[backend/03-security/SECURITY]]"
  - "[[backend/05-quality/ERRORS]]"
  - "[[backend/05-quality/VALIDATION]]"
  - "[[backend/04-infrastructure/EVENTS]]"
  - "[[specs/SPEC-ARCHIVOS]]"
  - "[[specs/SPEC-MODELOS-LLAMACPP]]"
---
# TOOLS — Catálogo de herramientas

Las diecisiete herramientas incluidas, su reparto y sus controles, y la capa universal que envuelve toda ejecución. La especificación funcional está en [[specs/SPEC-TOOLS]] y la de la herramienta de diagramas en [[specs/SPEC-DIAGRAMA]]; aquí está el contrato para quien implemente. Los payloads están en [[backend/02-interfaces/dto/TOOLS-DTO]].

## 1. El catálogo incluido

Cerrado. El agente no puede inventar herramientas fuera de esta lista.

### Archivos

| Herramienta | Lee o escribe | Permiso | Agente | Contrato |
|---|---|---|---|---|
| `leer_archivo` | Lee | `read` | Ambos | Ruta, y rango opcional (`inicio_linea`, `fin_linea`); devuelve el contenido, el rango leído y si se truncó |
| `listar_carpeta` | Lee | `read` | Ambos | Ruta relativa; devuelve la lista de entradas de un nivel |
| `buscar_archivos` | Lee | `read` | Ambos | Patrón; devuelve las rutas que coinciden |
| `buscar_en_archivos` | Lee | `read` | Ambos | Patrón; devuelve las coincidencias con archivo y línea |
| `crear_archivo` | Escribe | `write` | Solo `build` | Ruta y contenido; falla si el archivo ya existe |
| `escribir_archivo` | Escribe | `write` | Solo `build` | Ruta y contenido; sobrescribe |
| `editar_archivo` | Escribe | `edit` | Solo `build` | Ruta y una lista no vacía de reemplazos; cada `buscar` debe coincidir exactamente una vez |
| `eliminar_archivo` | Escribe | `write` | Solo `build` | Ruta; pide confirmación explícita |
| `crear_carpeta` | Escribe | `write` | Solo `build` | Ruta; falla si ya existe |
| `eliminar_carpeta` | Escribe | `write` | Solo `build` | Ruta; pide confirmación explícita |

`editar_archivo` es la única herramienta de edición parcial, así que es la única que cae en `edit`; el resto de las de escritura van en `write`. Los tres permisos son independientes: un agente con `write: deny` y `edit: allow` puede parchear archivos pero no crearlos, sobrescribirlos ni borrarlos.

### Semántica de la edición parcial

`editar_archivo` aplica una lista de sustituciones sobre el texto del archivo, en orden. Cada sustitución exige **coincidencia única**: si el fragmento no está o aparece más de una vez, la herramienta falla con un error corregible —qué fragmento, cuántas ocurrencias, en qué línea— y **no escribe nada**; nunca reemplaza en bloque silenciosamente.

Para que el modelo no falle por diferencias de formato al copiar texto, la comparación es tolerante, en este orden:

1. **Subcadena exacta** — admite ediciones a media línea.
2. **Finales de línea** — el modelo escribe LF; un archivo con CRLF o CR sueltos matchea igual, y el resultado conserva el estilo original.
3. **Unicode tipográfico** — comillas curvas, guiones largos, la elipsis y los espacios sin separación (NBSP) se comparan contra su forma ASCII; el `…` del archivo matchea los `...` del modelo.
4. **Bloques de línea** — con tolerancia creciente a espacios finales, sangría (tab vs espacios) y, otra vez, Unicode.

El BOM UTF-8 se preserva al escribir. El `reemplazo` se entrega en LF y se convierte al estilo del archivo. Un `reemplazos` vacío, un `buscar` vacío o un reemplazo idéntico al fragmento son `E_BAD_ARGS`. La implementación vive en `internal/fileops/reemplazo.go`.

### Terminal

| Herramienta | Lee o escribe | Permiso | Agente | Contrato |
|---|---|---|---|---|
| `ejecutar_comando` | Lee | `read` | Ambos | Comando y carpeta de trabajo; devuelve salida, error y si terminó |

### Internet

| Herramienta | Lee o escribe | Permiso | Agente | Contrato |
|---|---|---|---|---|
| `buscar_en_internet` | Lee | `read` | Ambos | Consulta; devuelve título, dirección y fragmento de cada resultado |
| `abrir_pagina` | Lee | `read` | Ambos | Dirección; devuelve el contenido de la página |

### Diagramas

| Herramienta | Lee o escribe | Permiso | Agente | Contrato |
|---|---|---|---|---|
| `diagram` | Lee | `read` | Ambos | `code` (código Mermaid flowchart); devuelve el diagrama renderizado como bloque Unicode en la TUI y su representación en texto para el modelo |

`diagram` solo interpreta el subconjunto de Mermaid documentado en [[specs/SPEC-DIAGRAMA]] (flowchart `TD`/`LR`, formas básicas, conexiones etiquetadas). El código de entrada es dato no confiable: no se ejecuta, no toca archivos ni red, y un error de sintaxis o de límites vuelve al modelo como error corregible, con línea y ejemplo de sintaxis válida. El parser, el layout y el renderizador son partes separadas y testeables por separado; la presentación del bloque en el hilo es de la TUI, no del handler (ver [[frontend/02-interfaces/INTERFACES]] §1.1).

### Sesión

| Herramienta | Lee o escribe | Permiso | Agente | Contrato |
|---|---|---|---|---|
| `crear_todo` | Sesión | `read` | Ambos | Un paso —contenido, estado y prioridad opcional— o una tarea principal —título y sus subtareas—; lo añade al final y devuelve la lista resultante |
| `actualizar_todo` | Sesión | `read` | Ambos | La lista de pasos de la sesión, completa, con las subtareas opcionales de cada paso; reemplaza la anterior y devuelve el checklist |

`crear_todo` y `actualizar_todo` escriben estado de la sesión, no archivos del proyecto: por eso caen en el permiso `read` y la tienen los dos agentes, sin tocar la garantía de escritura de §2.

Una **añade** y la otra **sustituye**, y la diferencia es el contrato, no una preferencia. `crear_todo` no recibe los pasos anteriores porque no los toca: no hay forma de que añadir uno borre los que ya estaban. `actualizar_todo` recibe la lista entera y la sustituye, que es lo que hace falta para reordenar, cancelar un paso o dejar la lista vacía. Las dos devuelven la lista resultante, para que el modelo tenga el estado sin preguntar.

La jerarquía es de un nivel: un paso puede colgar de una **tarea principal**. `crear_todo` la añade con su `titulo` y sus `subtareas` como un solo grupo atómico —se inserta completa o no se inserta nada—, y `actualizar_todo` admite `subtareas` en cada elemento que las lleve. Los pasos sin padre son raíz; la referencia la resuelve la capa de datos ([[database/01-schema/TABLES]] §`todos` y [[backend/01-domain/BUSINESS_RULES]]).

Ninguna de las dos recibe un identificador de paso: el orden es la posición y un identificador solo daría material para que un modelo cite mal.

### Flujos

| Herramienta | Lee o escribe | Permiso | Agente | Contrato |
|---|---|---|---|---|
| `ejecutar_flujo` | Según el flujo | Derivado del flujo | Ambos, según el flujo | `flujo` (el comando) y `peticion`; devuelve la entrega más el bloque resumido |

`ejecutar_flujo` es una **sola herramienta genérica**: el catálogo de flujos viaja en su esquema, que se arma por turno, y **no hay una tool por flujo**. Su handler no es un `Ejecutar` normal: se le inyecta a `flow`, que corre las etapas como sub-proceso del turno. Ver «Los flujos como herramienta» más abajo.

### Los flujos como herramienta

Tres cosas hacen de `ejecutar_flujo` una herramienta distinta del resto:

1. **Su permiso no es fijo: se deriva del flujo.** Si todas las etapas corren con `plan`, el flujo solo lee y lo lanza cualquier agente con `read`; si alguna corre con `build`, exige `write`/`edit`. El esquema se arma **por turno con el agente activo** y solo lista los flujos que puede lanzar, así que `plan` no ve a la vista un flujo que va a escribir. Es la excepción a «cada herramienta pertenece a un permiso fijo» de §2, y la única.
2. **Es de ejecución larga e interactiva.** El turno **espera** mientras el flujo corre sus etapas, conservando sus pausas de `continuacion` y `aprobacion`. Cuando la etapa `entrega` redacta, la ejecución cierra y el resultado vuelve al agente: la entrega más el **bloque de contexto resumido**. El interior del flujo no cruza: ni etapas, ni herramientas internas, ni ventana. Ver [[specs/SPEC-ORQUESTADOR-FLUJOS]] §Invocación del flujo.
3. **Tiene profundidad uno.** `ejecutar_flujo` no está disponible dentro de una etapa de flujo: un flujo no invoca a otro.

En pantalla se anuncia su verbo y su tema —`FLUJO [investigar] · <peticion>`— y las líneas de sub-proceso de las etapas entran después, como en un flujo lanzado por comando.

`crear_archivo` y `escribir_archivo` están separadas a propósito: el agente no destruye algo por accidente cuando pretendía crear.

### Lo que se ve en la TUI

Cada herramienta declara, además de su contrato, tres datos de presentación que la TUI usa en la línea del chat ([[frontend/02-interfaces/INTERFACES]] §1.1): su **verbo** (la etiqueta corta que la nombra), su **tema** (el campo de la petición cuyo valor es el objetivo que se enseña) y su **unidad** (cómo se mide el resultado). No viajan al modelo: son presentación.

| Herramienta | Verbo | Tema | Unidad |
|---|---|---|---|
| `leer_archivo` | `LEER` | `ruta` (o `ruta:inicio-fin`) | línea |
| `listar_carpeta` | `LISTAR` | `ruta` | entrada |
| `buscar_archivos` | `BUSCAR` | `patron` | coincidencia |
| `buscar_en_archivos` | `GREP` | `patron` | coincidencia |
| `crear_archivo` | `CREAR` | `ruta` | — |
| `escribir_archivo` | `ESCRIBIR` | `ruta` | — |
| `editar_archivo` | `EDITAR` | `ruta` | — |
| `eliminar_archivo` | `BORRAR` | `ruta` | — |
| `crear_carpeta` | `MKDIR` | `ruta` | — |
| `eliminar_carpeta` | `RMDIR` | `ruta` | — |
| `ejecutar_comando` | `EJEC` | `comando` | línea |
| `buscar_en_internet` | `WEB` | `consulta` | resultado |
| `abrir_pagina` | `ABRIR` | `direccion` | línea |
| `crear_todo` | `TODO` | `titulo` o `contenido` | paso |
| `actualizar_todo` | `TODO` | — | paso |
| `ejecutar_flujo` | `FLUJO` | `flujo` | etapa |
| `diagram` | `DIAGRAMAR` | `code` (primera línea) | diagrama |

Solo se muestra el **tema**: el resto de argumentos —el contenido de un archivo, un cambio— no se expone, porque la línea va a pantalla y a auditoría. En `leer_archivo` el tema es la ruta cuando se leyó el archivo entero y `ruta:inicio-fin`, con el rango real devuelto, cuando la lectura fue parcial —el modelo pidió un rango o el límite truncó el resultado—.

## 2. El reparto: `plan` mira, `build` escribe

El reparto sale de los `permissions` del agente ([[specs/SPEC-AGENTE-BASE]]), no de una lista de herramientas declarada. Cada herramienta pertenece a un **permiso** fijo, el de la columna de las tablas de §1, y no hay forma de declararlo de otro modo: `leer_archivo`, `listar_carpeta`, `buscar_archivos`, `buscar_en_archivos`, `ejecutar_comando`, `buscar_en_internet`, `abrir_pagina`, `crear_todo`, `actualizar_todo` y `diagram` son `read`; `crear_archivo`, `escribir_archivo`, `eliminar_archivo`, `crear_carpeta` y `eliminar_carpeta` son `write`; `editar_archivo` es `edit`.

La única herramienta sin permiso fijo es `ejecutar_flujo`: **lo hereda del flujo** que vaya a lanzar —todas sus etapas `plan` → `read`; alguna `build` → `write`/`edit`—, y su esquema se arma por turno con solo los flujos que ese permiso deja. Sigue siendo el reparto quien decide, no una lista suelta. Ver §1 «Los flujos como herramienta».

- `plan` permite solo `read` y deniega `write` y `edit`. No tiene ninguna herramienta que escriba en el proyecto.
- `build` permite los tres y recibe el catálogo completo. Es el único que crea, modifica y borra archivos.

El catálogo efectivo de cada agente se deriva de sus permisos contra el catálogo. No hay dos listas que puedan contradecirse.

## 3. El relevo entre agentes

```
plan lee  →  plan propone  →  tú apruebas  →  cambias a build  →  build aplica
```

1. `plan` investiga con herramientas de lectura.
2. `plan` propone el cambio concreto.
3. Tú lo apruebas.
4. Cambias a `build`.
5. `build` aplica exactamente lo aprobado.

**La aprobación vale para el cambio propuesto, no para lo que siga.** Si tras el relevo `build` necesita algo que `plan` no propuso —un borrado o una ruta fuera de la carpeta—, vuelve a preguntar. Una aprobación no es un permiso general.

## 4. Herramientas de archivo

Siguen [[specs/SPEC-ARCHIVOS]]: dentro de la carpeta del proyecto son accesibles y se aplican sin interrumpir; fuera se pide permiso y el agente explica por qué. **La confirmación solo la piden dos cosas: lo que queda fuera de la carpeta y lo destructivo —los borrados—**, también dentro de un flujo en curso. Las rutas se resuelven desde el directorio de trabajo (las absolutas son válidas para archivos de fuera) y la respuesta de `leer_archivo` devuelve la forma canónica relativa. Ver [[backend/02-interfaces/dto/TOOLS-DTO]] para las reglas de validación de rutas.

Las búsquedas (`buscar_archivos` y `buscar_en_archivos`) recorren también las carpetas ocultas de contenido —`.agents`, `.localcli`, `.github`— porque ahí vive parte del proyecto; solo se saltan los sistemas de control de versiones (`.git`, `.hg`, `.svn`) y las dependencias (`node_modules`).

### Lectura por rangos

`leer_archivo` devuelve el archivo entero si no supera el límite de lectura completa (64 KiB) y, si lo supera, un fragmento inicial marcado como truncado, con la línea desde la que seguir (`siguiente_linea`). Con `inicio_linea` y `fin_linea` lee un rango concreto, numerado desde 1 y tomado del archivo original —nunca de un fragmento ya leído—. La salida nunca supera el máximo por respuesta (32 KiB): si el rango no cabe, devuelve hasta donde llega e indica dónde terminó. Cuando una línea individual es demasiado grande, se devuelve lo que quepa y se señala. Los límites son fijos en esta versión y se miden en bytes. Como la herramienta ya recorta, la capa universal respeta su `Truncado` (§8).

### Resolución de rutas

Compartida con `listar_carpeta`: las rutas relativas se resuelven desde el directorio de trabajo del harness; `..` y las rutas absolutas son válidas; no se expande `~` ni `$HOME` —una ruta que empieza por esos prefijos se rechaza con `E_BAD_ARGS` y un recordatorio de usar la forma relativa—; no se busca en otras ubicaciones; y la normalización no elude las restricciones de acceso —una ruta que resuelve fuera de la carpeta sigue contando como «fuera» y exige permiso y explicación—. La respuesta devuelve `ruta` en forma canónica (relativa si el archivo está dentro del directorio de trabajo, absoluta si está fuera), y el `desc` del parámetro instruye al modelo a usar la ruta relativa conservando las subcarpetas: dos levers para que el modelo no pegue la ruta completa en `leer [.agents/commands/actualizar.md]`. Ver [[backend/05-quality/VALIDATION]].

## 5. Herramientas de terminal

Dos controles independientes: cuándo te interrumpe y qué no puede tocar.

### Control 1 — lo destructivo, lo de red y lo de fuera preguntan

La confirmación solo la piden tres casos; todo lo demás corre sin preguntar:

- **Comandos destructivos** (`internal/exec/riesgo.go`, lista cerrada): los que borran, machacan o apagan —`rm`, `rmdir`, `mv`, `dd`, `truncate`, `kill`, `shutdown`, `mkfs*`— y los subcomandos de git que descartan trabajo o lo publican (`reset`, `clean`, `checkout`, `restore`, `branch -D`, `push`).
- **Programas de egreso** (`curl`, `wget`, `nc`, `ncat`, `netcat`, `telnet`, `ssh`, `scp`, `sftp`, `rsync`): la red solo sale por ellos y por las herramientas de internet, y aquí la revisas tú.
- **Rutas fuera de la carpeta del proyecto:** un argumento con ruta absoluta que queda fuera de la raíz.

Lo que se repite en cada iteración —compilar, correr pruebas, revisar estilo, ver estado y diferencias— y cualquier otro comando inocuo se ejecuta directo. `EjecutarEquipo` (herramientas del usuario) aplica exactamente la misma regla.

### Control 2 — no toca tus archivos

El bloqueo **no depende de revisar el texto del comando**. La terminal no puede crear, editar ni borrar ningún archivo del usuario, por más indirecto que sea: `find -delete`, `tee`, una tubería hacia un archivo, `python3 -c` o un `sh -c` con redirección se colarían por un filtro de texto, y por eso el bloqueo es estructural, con Landlock.

Lo que sí puede hacer: usar su propio espacio (caché de compilación, temporales, carpeta de trabajo por sesión) y los dispositivos nulos (`/dev/null`, `/dev/zero`, `/dev/full`), y correr los comandos que no son destructivos, no salen a la red ni tocan rutas de fuera.

Lo que no puede hacer, aunque se le pida: crear, editar o borrar archivos del proyecto (para eso están las herramientas de escritura, que solo tiene `build` y donde los borrados sí piden confirmación), descartar cambios del repositorio —que además piden confirmación y Landlock impide—, hacer commits —los frena Landlock—, y publicar sin confirmación.

### Otras reglas de la terminal

- El comando corre en la carpeta del proyecto.
- La salida está limitada: un comando que no termina o genera salida sin fin se corta y se avisa, sin colgar el harness ni llenar el contexto.
- El resultado indica si el comando terminó bien o mal, con la salida de error cuando la hay.
- Un comando que falla no detiene el trabajo: el agente ve el error y sigue.
- Un comando declinado no llega a ejecutarse, y el modelo ve el rechazo.

## 6. Herramientas de internet

Son la vía de salida pensada para la red. La otra es la terminal, y ahí cada programa que pueda sacar información —`curl`, `wget`, `nc`, `ssh`, `scp`, `rsync`...— pide confirmación (Control 1).

- **Qué sale:** solo la consulta que redacta el modelo. Nunca el contenido de tus archivos, de la documentación, del historial de la conversación ni de tus tareas. Si una búsqueda necesita el proyecto para ser útil, el agente te lo pide a ti.
- **Qué vuelve:** `buscar_en_internet` devuelve resultados; `abrir_pagina` devuelve el contenido de una página. Ese contenido entra al contexto, ocupa el mismo espacio que el resto, y está sujeto al mismo recorte y al mismo registro de auditoría que en [[backend/01-domain/DOMAIN]].
- **Quién las usa:** los dos agentes. `plan` las necesita para consultar documentación de librerías mientras planifica; `build` para verificar versiones y APIs.

## 7. La forma de una herramienta

`tools` define el contrato. **No importa `fileops`, `exec` ni el cliente de internet**: sus tipos son la frontera, y las implementaciones se inyectan en el cableado. Es lo que permite que el enrutado deje de ser un `switch` de tres categorías.

```go
type Contexto struct {
    Ctx      context.Context   // cancelación: lo que en opencode es ctx.abort
    SesionID string
    Agente   string
    Permisos []Accion          // los permisos del agente activo, ya resueltos
    Ask      func(ctx context.Context, s Solicitud) (Decision, error)
    Meta     func(titulo string, meta map[string]any)
}

type Resultado struct {
    Salida   string           // lo que ve el modelo
    Meta     map[string]any
    Truncado bool             // si la herramienta ya recortó, la capa no repite
    Error    string           // error de negocio: el modelo puede corregirlo
}

type Ejecutar func(ctx context.Context, args any, c Contexto) (Resultado, error)

type Herramienta struct {
    Nombre      string
    Descripcion string
    Categoria   Categoria     // archivos | terminal | internet | tareas | flujos
    Permiso     Permiso       // read | write | edit: lo que hay que conceder para usarla; "derivado" solo en ejecutar_flujo
    Esquema     *Esquema      // se deriva del tipo de petición
    Ejecutar    Ejecutar      // handler propio; nil significa "sin implementar"
}
```

`Permiso` es lo que el agente concede: el reparto se decide comparando el `Permiso` de la herramienta contra los `permissions` del agente, y por eso es un dato del catálogo, no un `switch` en el enrutado. `Categoria` no decide nada de eso: sirve para agrupar y para la interfaz.

`ejecutar_flujo` es la excepción al reparto por `Permiso` fijo. En el registro su `Permiso` es «derivado»: el esquema que ve el agente se arma en **construcción del catálogo por turno** —de los flujos cargados contra los permisos del agente—, no comparando una columna contra los `permissions`. El paso 2 de la capa universal (§8) se resuelve con lo que quedó en ese esquema.

Tres cosas de este contrato que no existían antes:

- **`Ejecutar` es un handler propio**, no una categoría. Una herramienta nueva no necesita que nadie modifique un `switch`. El enrutado actual (`Destinos`, tres categorías fijas) desaparece.
- **`Ask` vive en el contexto**, no dentro del handler. La aprobación deja de estar acoplada a `fileops` y `exec`: cualquier herramienta la pide por el mismo camino, y el mecanismo queda en un solo sitio.
- **`Error` es un campo de `Resultado`, no un `error` de Go.** La distinción importa: un `error` es un fallo del harness y termina el turno; un `Error` populated es un resultado que el modelo puede leer y corregir.

### El esquema se deriva, no se escribe

`Esquema` se genera por reflexión sobre el tipo de petición de cada herramienta. Cada campo lleva su descripción en un tag `desc`:

```go
type PeticionEditarArchivo struct {
    Ruta        string   `json:"ruta"        desc:"Archivo a modificar"`
    Reemplazos  []Cambio `json:"reemplazos"  desc:"Lista no vacía de sustituciones aplicadas en orden; cada buscar debe coincidir exactamente una vez"`
}
```

De ahí sale un JSON Schema: tipo, obligatoriedad y descripción por campo. Un campo sin `omitempty` es obligatorio; uno que lo lleva es opcional.

La descripción se escribe **para el modelo**, no para quien lee el código. Dice cuándo usar la herramienta y qué significa el argumento, no cómo está implementado. La comprobación cruzada de que el esquema generado coincide con el documentado en [[backend/02-interfaces/dto/TOOLS-DTO]] es un test: si divergen, falla la suite.

`tools` **no** reutiliza el estimador de `internal/context`. `context` ya depende de `llm`, `store` y `docs`; que `tools` lo importara convertiría un módulo de capa baja en uno que arrastra capa alta. El truncado (§8) lleva su propio estimador, que es pequeño.

## 8. La capa universal

Una sola función envuelve **toda** ejecución, sin excepciones. No la escribe cada herramienta.

```go
func (r *Registro) Ejecutar(ctx context.Context, peticion Peticion) Resultado
```

En orden:

1. **Buscar** la herramienta en el registro. Si no existe, resultado con error: el modelo ha pedido algo que no hay.
2. **Comprobar el permiso.** El `Permiso` de la herramienta tiene que estar concedido en los `permissions` del agente (`peticion.Agente` los resuelve). Este paso es el que sostiene §2, y por eso vive **antes** de ejecutar nada.
3. **Validar** los argumentos contra `Esquema`. Aquí se decodifica el JSON que el modelo envió.
4. **Emitir** `herramienta_invocada` con el verbo y el tema (el campo objetivo que declara el catálogo: la ruta, el patrón, el comando).
5. **Ejecutar** el handler, con el `Contexto` montado, y medir cuánto tarda.
6. **Recortar** la salida.
7. **Emitir** `herramienta_resultado` con el nombre, si terminó bien, la medida del resultado, si hubo recorte y la duración de la ejecución.

### Duración

La medición va aquí, y no dentro de cada handler, porque **esta es la única función por la que pasa toda ejecución**. Medir en la herramienta obligaría a que cada una llevara su propio reloj, y bastaría con que una lo olvidara para que su línea saliera sin tiempo.

La duración medida es la del **handler**, no la de la ejecución completa: el paso 2 pide permiso y puede esperar a que el usuario conteste, y ese tiempo no es de la herramienta, es de la persona. Se mide en el paso 5, y se reporta tanto si el handler terminó bien como si falló —una ejecución que tardó cinco minutos y falló también tardó cinco minutos—. Los pasos 1 a 4 no se miden, y por eso `herramienta_invocada` no lleva tiempo: el modelo sabe cuándo pidió, no cuánto lleva esperando.

El dato es de **pantalla, no de modelo**: viaja en `herramienta_resultado` y desde ahí a la línea del chat y a `chat_evento.duration_ms`, y no entra en `Resultado` ni en el contexto. Ver [[backend/04-infrastructure/EVENTS]] y [[database/01-schema/TABLES]] §`chat_evento`.

### Errores que el modelo puede corregir

Si el paso 3 falla, **no se lanza nada**: se devuelve un `Resultado` cuyo `Error` describe qué falta y qué se esperaba.

```
falta el campo obligatorio "ruta"
"editar_archivo" espera: ruta (obligatorio), reemplazos (obligatorio)
```

Lo mismo con un error de negocio entendible —una ruta que no existe, un archivo que ya está—. El turno sigue; el modelo ve el motivo y puede reintentar en la misma pasada.

Un fallo no corregible —permiso denegado, fallo de red— también se le dice, pero como error duro: el turno no reintenta. La distinción entre "esto lo arreglas tú corrigiendo la llamada" y "esto no va a funcionar" es explícita, no se deduce del texto.

### Recorte

**Toda** salida se recorta antes de volver al modelo, no solo la de la terminal. Se mide en tokens, contra el presupuesto de contexto, con un estimador propio de `tools`.

El recorte no es silencioso: `Resultado.Truncado` va a `true`, la salida incluye la marca de que se cortó y por dónde, y el modelo lo ve. Si el handler ya recortó por su cuenta —la terminal tiene su propio límite— la capa no vuelve a recortar y respeta `Truncado`.

Una herramienta que ya produce texto corto nunca paga el coste de esta capa más de una comparación de longitud.

### Un solo sitio para engancharse

Como el punto de permiso, el de validación, el de medición, el de recorte y el de eventos están todos aquí, no hace falta tocar ninguna herramienta para añadir una regla nueva ni para que su línea diga cuánto tardó.

## 9. Hooks

Tres puntos de enganche, en `tools`, sin dependencias:

```go
type Hooks struct {
    AntesDeEjecutar     func(nombre string, args any, meta map[string]any)
    DespuesDeEjecutar   func(nombre string, r Resultado, err error, meta map[string]any)
    DefinirHerramienta  func(nombre, descripcion string, e *Esquema) (string, *Esquema)
}
```

- **`AntesDeEjecutar` / `DespuesDeEjecutar`** — auditoría y métricas. El único sitio donde se registra "se ejecutó esta herramienta con estos argumentos y terminó así", sin instrumentar quince handlers.
- **`DefinirHerramienta`** — puede ajustar nombre, descripción y esquema antes de que lleguen al modelo. Es el punto de extensión para adaptar el catálogo a un modelo concreto sin tocar el catálogo.

Se conectan en el cableado. Si no hay ninguno, la capa universal funciona igual: los hooks son opcionales por diseño, no un punto de fallo.

## 10. El cable: cómo llegan al modelo

`tools` no habla con ningún motor. `agent` pide los esquemas y los pone en la petición; la frontera neutra (`llm`) los entrega al núcleo común (`llm/openai`), que es quien los serializa —el protocolo OpenAI-compatible que habla `llama-server`— y la extensión `llamacpp` solo añade lo suyo al lado.

1. `agent` pide a `tools` el **esquema** de las herramientas del agente activo.
2. `agent` construye una petición neutra (`llm.Peticion`) con esa lista en el campo de herramientas.
3. La frontera `llm` la pasa al **núcleo común** (`llm/openai`), que la serializa para `/v1/chat/completions`. La extensión `llamacpp` no cambia esto: puede añadir campos propios de su runtime, nunca alterar el canal de herramientas.
4. El modelo responde pidiendo una por su nombre con los argumentos ya formados.
5. El núcleo común entrega las peticiones acumuladas; `agent` itera y llama a `Registro.Ejecutar`.
6. El resultado vuelve al modelo como un mensaje de rol `tool` con su `tool_call_id`, tal como lo espera el núcleo común.

**Qué pasa si el modelo no admite herramientas.** Esa capacidad puede estar en tres estados y solo uno permite este camino: `soportada` usa el esquema estructurado; `no soportada` hace que `agent` caiga a modo conversación, sin herramientas en la petición; `desconocida` —una extensión que no informa, no una capacidad ausente— **también cae a modo conversación**, y no se finge que un canal estructurado que quizá no existe dé resultado. Ver [[specs/SPEC-MODELOS-LLAMACPP]].

El catálogo **no** viaja en el mensaje de sistema. `PromptDeSistema` se queda con el prompt del agente.

El formato exacto del núcleo común y de la extensión está en [[specs/SPEC-MODELOS-LLAMACPP]] y en [[backend/04-infrastructure/INTEGRATIONS]]; el de los mensajes de herramienta que no se persisten, en [[database/01-schema/ENUMS]].

## Referencias

- [[specs/SPEC-TOOLS]] — la especificación funcional y los criterios de aceptación.
- [[specs/SPEC-DIAGRAMA]] — la herramienta `diagram` y su subconjunto de Mermaid.
- [[specs/SPEC-ARCHIVOS]] — reglas de permiso sobre archivos.
- [[specs/SPEC-MODELOS-LLAMACPP]] — el canal de herramientas y la capacidad del modelo.
- [[backend/02-interfaces/INTERFACES-GENERAL]] — las superficies.
- [[backend/02-interfaces/dto/TOOLS-DTO]] — los payloads de cada herramienta y sus esquemas.
- [[backend/03-security/SECURITY]] — la garantía de escritura y el bloqueo de la terminal.
- [[backend/04-infrastructure/EVENTS]] — `herramienta_invocada` y `herramienta_resultado`.
- [[backend/05-quality/VALIDATION]] — validación de rutas y argumentos, y el error autocorregible.
- [[backend/05-quality/ERRORS]] — errores que puede producir cada herramienta.
- [[backend/DECISIONS]] — decisiones de contrato que sostienen este módulo.
