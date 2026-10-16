---
title: SPEC — Catálogo de herramientas
tags: [specs, requisito]
depende_de:
  - "[[IDEA]]"
  - "[[specs/SPEC-ARCHIVOS]]"
  - "[[specs/SPEC-NODO-CONTEXTO]]"
  - "[[specs/SPEC-MODELOS-LLAMACPP]]"
relacionado:
  - "[[backend/DECISIONS]]"
  - "[[specs/SPEC-SKILLS]]"
  - "[[specs/SPEC-AGENTE-BASE]]"
  - "[[specs/SPEC-ORQUESTADOR-FLUJOS]]"
  - "[[specs/SPEC-RESOLVER]]"
  - "[[specs/SPEC-DIAGRAMA]]"
---
# SPEC — Catálogo de herramientas

Prioridad: P0 (núcleo)

## Propósito

Definir qué herramientas tiene el agente para trabajar, cuáles puede usar para mirar y cuáles para escribir, en qué casos necesita permiso y cómo se le pide una herramienta al modelo.

## Alcance

Incluye el catálogo incluido, el reparto por agente, las reglas de la terminal y de internet, la capa universal que envuelve toda ejecución y la degradación cuando el modelo no sabe usarlas.
No incluye las reglas de permiso sobre archivos, que están en [[specs/SPEC-ARCHIVOS]], ni la selección de contexto, que está en [[specs/SPEC-NODO-CONTEXTO]], ni la conexión con el motor de inferencia, que está en [[specs/SPEC-MODELOS-LLAMACPP]]. La herramienta `diagram` figura en el catálogo, pero su contrato funcional —sintaxis, layout y render— vive en [[specs/SPEC-DIAGRAMA]].

## Actores

- **Usuario**: aprueba lo que lo requiere.
- **Agente `plan`**: solo mira, propone y pide aprobación.
- **Agente `build`**: escribe, modifica y borra.
- **Sistema**: ejecuta la herramienta y devuelve el resultado.

## Catálogo

### Archivos

| Herramienta | Lee o escribe | Agente |
|---|---|---|
| `leer_archivo` | Lee | Ambos |
| `listar_carpeta` | Lee | Ambos |
| `buscar_archivos` | Lee | Ambos |
| `buscar_en_archivos` | Lee | Ambos |
| `crear_archivo` | Escribe | Solo `build` |
| `escribir_archivo` | Escribe | Solo `build` |
| `editar_archivo` | Escribe | Solo `build` |
| `eliminar_archivo` | Escribe | Solo `build` |
| `crear_carpeta` | Escribe | Solo `build` |
| `eliminar_carpeta` | Escribe | Solo `build` |

### Terminal

| Herramienta | Lee o escribe | Agente |
|---|---|---|
| `ejecutar_comando` | Lee | Ambos |

### Internet

| Herramienta | Lee o escribe | Agente |
|---|---|---|
| `buscar_en_internet` | Lee | Ambos |
| `abrir_pagina` | Lee | Ambos |

### Diagramas

| Herramienta | Lee o escribe | Permiso | Agente |
|---|---|---|---|
| `diagram` | Lee | `read` | Ambos |

`diagram` convierte código Mermaid de un flowchart en un diagrama Unicode que se muestra como bloque en el hilo del chat. No toca el proyecto ni la red —solo interpreta su argumento—, así que va en `read` y la tienen los dos agentes. Su contrato funcional completo está en [[specs/SPEC-DIAGRAMA]]; aquí solo figura su lugar en el catálogo.

### Sesión

| Herramienta | Lee o escribe | Permiso | Agente |
|---|---|---|---|
| `crear_todo` | No toca el proyecto | `read` | Ambos |
| `actualizar_todo` | No toca el proyecto | `read` | Ambos |

Son las dos que mantienen la lista de pasos de la sesión: el plan de trabajo de varios pasos que el usuario ve en el panel. Pueden ser pasos sueltos o una tarea principal con sus subtareas. No escriben en el proyecto —es estado de la sesión—, así que la tienen los dos agentes.

**Una añade y la otra reemplaza, y no se confunden.** `crear_todo` **añade al final** de la lista —un paso suelto o una tarea principal con sus subtareas— y nada más: no toca los que ya había. `actualizar_todo` **sustituye la lista entera** por la que mande. La primera es la vía normal para planning; la segunda, para reordenar, cancelar un paso o cambiar estados sin añadir.

La separación no es cosmética. Con una sola herramienta que sustituye, un modelo que Cree estar «actualizando un paso» reescribe la lista con lo que recuerda y **borra los demás sin saberlo**. Con `crear_todo` delante, añadir es un verbo que no puede perder nada, y el modelo solo llama a `actualizar_todo` cuando de verdad quiere reescribir todo.

Ninguna de las dos necesita saber si la lista existe: **las dos devuelven la lista resultante**, así que el modelo tiene el estado en su propio contexto y no tiene que preguntar. Y ninguna necesita un identificador de paso: el orden es la posición en la lista, y un identificador solo serviría para que un modelo lo cite mal.

Ver «La lista de pasos» para las reglas de estado.

### Flujos

| Herramienta | Lee o escribe | Permiso | Agente |
|---|---|---|---|
| `ejecutar_flujo` | Según el flujo | Derivado del flujo | Ambos, según el flujo |

`ejecutar_flujo` lanza un flujo y devuelve su entrega. Es una **sola herramienta genérica** —`flujo` elige el comando y `peticion` es el objetivo—, no una herramienta por flujo: el catálogo de flujos viaja en el esquema, no en el nombre. Ver «Los flujos como herramienta».

## Cómo se le pide una herramienta al modelo

El modelo **no lee un formato en prosa**. Recibe cada herramienta como una definición estructurada —nombre, descripción y esquema de sus argumentos— y responde pidiendo una por nombre con sus argumentos ya formados. La plataforma lo resuelve; el modelo no tiene que imitar una forma de escribir.

Esto tiene tres consecuencias:

1. **El contrato no depende de que el modelo obedezca.** Con un formato en texto, si el modelo se equivoca al escribir la llamada, la herramienta no se ejecuta y el turno se pierde. Con el contrato nativo, la llamada llega ya parseada.
2. **Los argumentos se validan antes de tocar nada.** Un esquema declarado significa que se puede rechazar una llamada con un campo que falta y explicarle al modelo exactamente qué falta.
3. **El sistema deja de gastar contexto en instrucciones.** El catálogo no viaja en el mensaje de sistema: viaja en el canal de herramientas.

Cada herramienta expone su esquema a partir de sus propios tipos de petición, con un esquema por campo: tipo, si es obligatorio y qué significa. La descripción que ve el modelo está escrita para que sepa cuándo usarla y cuándo no, no para resumir la implementación.

El resultado de una ejecución vuelve al modelo como un mensaje propio de herramienta, asociado al nombre con el que la pidió. Se hacen **varias pasadas**: el modelo pide, ve el resultado, y vuelve a pedir si le hace falta. Hay un máximo de pasadas por turno para que un bucle no se alargue sin fin. Al agotarlo, el turno pide una redacción final y **avisa**: si el modelo tampoco entrega nada, el turno falla con el motivo a la vista. Nunca termina en silencio. Ver [[specs/SPEC-AGENTE-BASE]] §El ciclo de un turno.

Las ejecuciones son **secuenciales**. Una tras otra, en el orden en que el modelo las pidió. No en paralelo: el orden importa —una carpeta tiene que existir antes de escribir dentro— y la aprobación de cada escritura es una interrupción que el usuario tiene que ver en orden.

## La capa universal

Toda ejecución de una herramienta pasa por la misma secuencia, sin excepciones. No la escribe cada herramienta: la escribe el sistema.

1. **Se busca** la herramienta en el registro.
2. **Se comprueba** el permiso del agente activo contra esa herramienta.
3. **Se validan** los argumentos contra su esquema.
4. **Se ejecuta** el handler de la herramienta.
5. **Se recorta** la salida.
6. **Se mide** cuánto tardó la ejecución.
7. **Se avisa** de lo que pasó, para que se vea en pantalla: el desenlace, la medida del resultado, si se recortó y cuánto tardó.

La medición va **en la capa universal, no en cada herramienta**: es el único punto por el que pasan todas las ejecuciones, así que ninguna tiene que llevar un reloj propio para que su línea diga cuánto tardó. El tiempo medido es el de la ejecución —lo que tardó el handler—, no el del turno: el turno incluye el modelo, y son dos cosas distintas.

### Errores que el modelo puede corregir

Un error de validación **no termina el turno**: vuelve al modelo como un resultado con el motivo y con lo que se esperaba, de forma que pueda corregir la llamada y reintentarlo en la misma pasada.

```
falta el campo obligatorio "ruta"
"editar_archivo" espera: ruta (obligatorio), reemplazos (obligatorio)
```

Esto vale también para un error de negocio —una ruta que no existe, un archivo que ya está— siempre que el mensaje sea entendible por el modelo y no exponga rutas absolutas del usuario.

Un fallo que el modelo no puede corregir por sí mismo —una denegación, un permiso del sistema, un fallo de red— se le dice igualmente, pero no se le reintenta: se lo informa y sigue.

### Recorte de la salida

**Toda** salida de herramienta se recorta antes de volver al modelo, no solo la de la terminal. Una lectura enorme, un volcado de tabla, una respuesta de API gigante.

El recorte no es un castigo: la herramienta que lo ha hecho avisa de que lo ha hecho, y el modelo lo sabe para no agir como si tuviera el resultado entero. Cuando se corta por la mitad, se dice por dónde.

## El reparto: `plan` mira, `build` escribe

El reparto sale de los `permissions` de cada agente, que se declaran **por permiso**, con tres: `read` (mirar el proyecto, la terminal de consulta, internet y la lista de pasos de la sesión), `write` (crear, sobrescribir o borrar archivos y carpetas) y `edit` (parchear contenido existente). El catálogo efectivo de herramientas se deriva de ahí contra el catálogo: no hay una lista de herramientas suelta que pueda contradecir los permisos. El detalle de qué herramienta cae en qué permiso está en «Quién puede usar qué» de [[specs/SPEC-AGENTE-BASE]].

`ejecutar_flujo` es la única herramienta sin permiso fijo: **lo hereda del flujo** que vaya a lanzar. Se arma por turno y solo llega al alcance de su permiso derivado, así que sigue siendo el reparto quien decide, no una lista suelta. Ver «Los flujos como herramienta».

`plan` permite solo `read` y deniega los otros dos. Nunca recibe una herramienta que escriba en el proyecto. Existe para entender el proyecto, decidir qué hay que hacer y proponerlo.

`build` permite los tres y recibe el catálogo completo. Es el único que crea, modifica y borra.

Esto no es una restricción de estilo: es la garantía de que nada cambia en tu proyecto sin que `plan` lo haya propuesto antes y tú lo hayas aprobado.

## La lista de pasos

La lista de pasos de la sesión es estado de la sesión, no del proyecto, y por eso la tienen los dos agentes bajo el permiso `read`. La mantienen dos herramientas, y la diferencia entre las dos es qué le pasa a lo que ya había.

`crear_todo` **añade al final, en un solo paso atómico: un paso suelto o una tarea principal con sus subtareas**. Un paso suelto recibe `contenido` —qué hay que hacer— y `estado`; `prioridad` es opcional y por defecto es `media`. Una tarea principal recibe `titulo` y `subtareas` —cada una con su `contenido` y su `estado`, y `prioridad` opcional— y se inserta como una fila raíz seguida de sus subtareas. No recibe los pasos anteriores: no los necesita, porque no los toca. Lo que hace es insertar y devolver la lista completa, así que añadir nunca puede perder los que ya estaban, ni una subtarea puede aparecer sin su principal. Las dos formas se excluyen: `contenido` junto a `titulo`/`subtareas` es un error corregible.

`actualizar_todo` **sustituye la lista entera** por la que mande, en un solo paso atómico. Es la herramienta para reordenar, cancelar un paso, cambiar estados de golpe o dejar la lista vacía: casos en los que «añadir» no basta porque hay que quitar o cambiar algo de lo anterior. Cada elemento de la lista puede llevar sus propios `subtareas`, en un solo nivel; sin ellos, cada elemento se comporta como hoy y la llamada es compatible con la que ya existía.

La jerarquía es de **un solo nivel**: una tarea principal puede tener subtareas, y una subtarea no tiene hijas. En la base, una fila raíz lleva `parent_position` nulo y cada subtarea apunta a la posición de su principal dentro de la misma sesión; el vínculo lo valida el harness al escribir. La jerarquía no introduce identificadores de fila: el orden sigue siendo la posición ([[database/01-schema/TABLES]] §`todos`). El panel anida las subtareas bajo su principal.

El vocabulario es cerrado y lo comparten las dos: el estado de un paso es `pendiente`, `en_progreso`, `completada` o `cancelada`, y la prioridad es `alta`, `media` o `baja`. Un estado o una prioridad fuera de ese conjunto no se guarda: se devuelve al modelo el motivo y lo reintenta, como cualquier otro argumento mal formado.

Y se mantiene la regla de que **solo un paso puede estar `en_progreso` a la vez**, que hasta ahora era una instrucción en la descripción de la herramienta y no una comprobación. Añadir un paso ya en curso cuando otro lo está tiene que ser un error corregible, no un conflicto silencioso: `crear_todo` lo rechaza y le dice al modelo que use `actualizar_todo` para cambiar el que está en curso.

Ninguna de las dos herramientas necesita preguntar por el estado actual de la lista, y por eso devuelven la lista resultante. El modelo la ve y decide el siguiente paso sin una ronda extra. Ver [[database/01-schema/TABLES]] para cómo se guarda.

## Los flujos como herramienta

`ejecutar_flujo` es la herramienta que usa el agente para arrancar el trabajo que pide etapas encadenadas: un informe, una implementación, una refactorización. Por fuera es una herramienta más —el modelo la pide por el canal nativo, con `flujo` (el comando del flujo) y `peticion`—, pero su handler no es un `Ejecutar` normal: **invoca al orquestador de flujos**, que corre las etapas como sub-proceso del turno.

De ahí salen tres diferencias con el resto de herramientas:

1. **Su permiso no es fijo: se deriva del flujo.** Si todas las etapas corren con `plan`, el flujo solo lee y lo lanza cualquier agente con `read`; si alguna etapa corre con `build`, el flujo escribe y exige `write`/`edit`. El esquema se arma **por turno con el agente activo**: solo lista los flujos que ese agente puede lanzar. Así `plan` no recibe a la vista un flujo que va a escribir, y la garantía «`plan` mira, `build` escribe» no depende de la disciplina del modelo.
2. **Es de ejecución larga e interactiva.** El flujo no termina en la pasada en que se pide: corre sus etapas, conservando sus pausas de `continuacion` y `aprobacion`, y el turno **espera** respuesta inline. Cuando la etapa `entrega` redacta, la ejecución cierra y el resultado vuelve al agente: la entrega más el **bloque de contexto resumido** —una aportación por etapa, condensada—. Lo que no cruza es el interior del flujo: ni etapas, ni herramientas internas, ni ventana. El agente sabe qué flujo lanzó y qué entregó, no lo que pasó dentro.
3. **Tiene profundidad uno.** `ejecutar_flujo` no está disponible dentro de una etapa de flujo: un flujo no invoca a otro.

La frontera de salida es deliberada: la sesión principal no debe heredar ruido del interior del flujo. Es la misma «frontera de sesión» que describe [[specs/SPEC-ORQUESTADOR-FLUJOS]] §Invocación del flujo.

En pantalla, el lanzamiento se anuncia con su verbo y su tema (ver el catálogo): `FLUJO [investigar] · "por qué el arranque cuelga"`, y a partir de ahí las líneas de sub-proceso de las etapas.

## Cuando el modelo no sabe usar herramientas

No todos los modelos que se pueden elegir en el motor de inferencia saben pedir herramientas. La capacidad se puede preguntar antes de elegir, y hay modelos que responden que no.

Un modelo sin esa capacidad **no puede usar ninguna herramienta**, porque no hay forma nativa de pedirlas y LocalCli no las presenta en texto para que las imite.

Cuando ocurre, el agente activo cae a **modo conversación**: sigue respondiendo con el contexto que recibe, pero sin nada que ejecutar. La interfaz lo dice bajo la línea de entrada, donde ya se muestra si el modelo tiene esa capacidad.

No se bloquea el modelo ni se esconde la opción, porque un modelo que no usa herramientas puede ser exactamente el que el usuario quiera para conversar. Lo que no puede es fingir que trabaja: si te pide abrir un archivo, te dirá que no puede.

## El relevo entre agentes

`plan` no escribe. Termina su trabajo proponiendo, y a partir de ahí el relevo es explícito.

```
plan lee  →  plan propone  →  tú apruebas  →  cambias a build  →  build aplica
```

1. `plan` investiga con herramientas de lectura.
2. `plan` propone el cambio concreto.
3. Tú lo apruebas.
4. Cambias a `build`.
5. `build` aplica exactamente lo aprobado.

**La aprobación vale para el cambio propuesto, no para lo que siga.** Si tras el relevo `build` necesita hacer algo que `plan` no propuso, vuelve a preguntar. Una aprobación no es un permiso general.

## Herramientas de archivo

Siguen las reglas de [[specs/SPEC-ARCHIVOS]]: dentro de la carpeta del proyecto son accesibles, fuera se pide permiso y el agente explica por qué, y **toda escritura pasa por aprobación**.

Una diferencia importante: `crear_archivo` falla si el archivo ya existe, y `escribir_archivo` lo sobrescribe. Están separadas a propósito. Así el agente no destruye algo por accidente cuando pretendía crear.

### `leer_archivo`

Lee el contenido de un archivo de texto. Si el archivo es pequeño, devuelve su contenido completo; si es grande, permite leerlo por rangos de líneas.

**Parámetros:**

- `ruta` (obligatoria): ruta del archivo. **Para archivos del proyecto, relativa al directorio de trabajo del harness**; se aceptan absolutas solo para archivos de fuera. Nunca se empieza por `~` ni `$HOME` (no se expanden).
- `inicio_linea` (opcional): línea inicial, contando desde 1.
- `fin_linea` (opcional): línea final, inclusive.

**Comportamiento:**

1. Validar que la ruta exista y corresponda a un archivo regular.
2. Comprobar que el archivo pueda leerse como texto.
3. Si no se especifica un rango y el archivo no supera el límite, devolver el contenido completo.
4. Si el archivo supera el límite, devolver un fragmento inicial e indicar que está truncado, diciendo cómo pedir otros rangos.
5. Si se especifica un rango, devolver las líneas solicitadas dentro del límite máximo de salida.
6. Si el rango excede el límite de salida, devolver el contenido disponible e indicar dónde terminó la lectura.
7. Informar el número total de líneas cuando pueda determinarse.
8. Devolver `ruta` en forma canónica: relativa al directorio de trabajo si el archivo está dentro de él, absoluta si queda fuera. El modelo ve esa forma corta en cada resultado y refuerza el uso de rutas relativas.

**Respuesta:** `ruta` (ruta canónica resuelta: relativa al directorio de trabajo si el archivo está dentro, absoluta si está fuera), `contenido`, `linea_inicio` (primera línea devuelta), `linea_fin` (última línea devuelta), `total_lineas` (si se conoce), `truncado` (si se omitió contenido) y `siguiente_linea` (línea desde la que puede continuar la lectura, si corresponde).

**Errores:** devolver un error claro cuando el archivo no existe, la ruta corresponde a un directorio, no hay permisos de lectura, el archivo no es compatible con la lectura de texto, o los parámetros o rangos son inválidos. Una ruta que empieza por `~` o contiene `$HOME` se rechaza con `E_BAD_ARGS` y un aviso de que esos prefijos no se expanden, para que el modelo corrija la forma en vez de chocar contra un «archivo no encontrado». Los errores de archivo inexistente y ruta que es un directorio recuerdan localizar el archivo con `buscar_archivos`/`listar_carpeta`: son corregibles y el turno no debe abandonarse por uno de ellos.

**Límites (fijos en esta versión):** lectura completa, 64 KiB; máximo por respuesta, 32 KiB; numeración de líneas desde 1. Se miden en bytes. El rango se aplica **sobre el archivo original**, nunca sobre un fragmento previamente truncado. Cuando una línea individual es demasiado grande, se devuelve lo que quepa y se señala.

**Restricciones:** no modifica ni elimina archivos, no ejecuta el contenido del archivo, no excede el límite máximo de salida y respeta las restricciones de acceso del harness.

### Resolución de rutas

Regla compartida por `leer_archivo` y `listar_carpeta`:

1. Las rutas relativas se resuelven desde el directorio de trabajo actual del harness.
2. Las rutas que contienen `..` permiten acceder a directorios superiores.
3. Las rutas absolutas se utilizan directamente.
4. No se busca automáticamente en otras ubicaciones si el archivo no existe.
5. No se expande `$HOME` ni `~`; una ruta que empieza por `~` o los contiene se rechaza con `E_BAD_ARGS` y un recordatorio de usar rutas relativas —es un error corregible, no un archivo no encontrado.
6. La normalización de rutas no debe utilizarse para eludir las restricciones de acceso del harness: una ruta que resuelve fuera de la carpeta del proyecto sigue siendo «fuera» y exige permiso y explicación ([[specs/SPEC-ARCHIVOS]]).
7. La respuesta devuelve `ruta` en forma canónica: relativa al directorio de trabajo cuando el archivo cae dentro de él, absoluta cuando queda fuera.

Para reducir el hábito del modelo de pegar la ruta completa (`leer [/home/usuario/proyecto/.agents/commands/actualizar.md]` en vez de `leer [.agents/commands/actualizar.md]`), el `desc` del parámetro `ruta` en el esquema instruye la ruta relativa conservando las subcarpetas, la respuesta la devuelve canónica y `~`/`$HOME` fallan con un aviso claro que enseña la convención.

## Herramientas de terminal

Tiene **dos controles independientes**: una lista blanca que decide qué comandos existen, y un bloqueo que garantiza que no puede tocar tus archivos.

### Control 1 — lista blanca

Estos comandos se ejecutan sin preguntar, porque son los que se repiten en cada iteración:

- Compilar el proyecto.
- Correr las pruebas.
- Revisar estilo y tipos.
- Ver el estado y las diferencias del repositorio: estado, diferencias, historial.

Cualquier otro comando pide aprobación antes de ejecutarse.

### Control 2 — no toca tus archivos

El bloqueo no depende de revisar el texto del comando. **La terminal no puede crear, editar ni borrar ningún archivo tuyo**, por más indirecto que sea el comando. Es una garantía del entorno, no una convención.

Un filtro de texto no sirve: `find -delete`, `tee`, una tubería hacia un archivo, `python3 -c` o un `sh -c` con redirección se cuelan sin esfuerzo. Por eso el bloqueo es estructural.

Lo que sí puede hacer:

- Usar su propio espacio: caché de compilación, temporales y una carpeta de trabajo por sesión.
- Correr los comandos de la lista blanca.

Lo que no puede hacer, aunque se le pida:

- Crear, editar o borrar archivos del proyecto. Para eso están las herramientas de escritura, que solo tiene `build` y que sí pasan por aprobación.
- Descartar cambios del repositorio.
- Hacer commits, subir cambios o publicar.

### Otras reglas

- El comando corre en la carpeta del proyecto.
- La salida está limitada. Un comando que no termina o que genera salida sin fin no puede colgar el harness ni llenar el contexto. Si se corta, se avisa de que se cortó.
- El resultado indica si el comando terminó bien o mal, con la salida de error cuando la hay.
- Un comando que falla no detiene el trabajo: el agente ve el error y sigue.

## Herramientas de internet

Son las únicas que hacen salir información de la máquina.

### Qué sale

**Solo la consulta que redacta el modelo.** Nada más.

Nunca sale:

- El contenido de tus archivos.
- El contenido de la documentación del proyecto.
- El historial de la conversación.
- Los datos de tus tareas.

Si una búsqueda necesita el contenido del proyecto para ser útil, el agente te lo pide a ti en lugar de mandarlo.

### Qué vuelve

`buscar_en_internet` devuelve resultados: títulos, direcciones y fragmentos. `abrir_pagina` devuelve el contenido de una página.

Ese contenido entra en el contexto, **ocupa el mismo espacio que el resto** y está sujeto al mismo recorte y al mismo registro de auditoría. Ningún agente vierte el contenido de una página sin procesarlo en su respuesta.

### A quién tiene acceso

Las usan los dos. `plan` las necesita para consultar documentación de librerías mientras planifica. `build` las necesita para verificar versiones y APIs.

## Reglas de negocio

- El modelo pide herramientas por su nombre con un esquema declarado, no imitando un formato escrito. No hay contrato en prosa.
- El catálogo llega por el canal de herramientas, no en el mensaje de sistema.
- Los argumentos se validan contra el esquema antes de ejecutar nada.
- Un error de validación vuelve al modelo con el motivo y lo que se esperaba, para que lo corrija en la misma pasada.
- Toda salida de herramienta se recorta, sea del tamaño que sea, y se avisa de que se ha recortado.
- `leer_archivo` devuelve el archivo completo si no supera el límite de lectura completa, y si lo supera devuelve un fragmento inicial, lo marca como truncado e indica la línea desde la que seguir.
- `leer_archivo` lee por rangos de línea contando desde 1; el rango se aplica sobre el archivo original y nunca sobre un fragmento ya truncado.
- Una lectura que supera el máximo por respuesta devuelve tantas líneas como quepan e indica dónde terminó, sin exceder nunca el límite.
- `leer_archivo` informa el total de líneas cuando puede determinarlo, y señala cuando una línea individual es demasiado grande para devolverla entera.
- Los límites de `leer_archivo` (lectura completa y máximo por respuesta) son fijos en esta versión y se miden en bytes.
- Las rutas relativas de `leer_archivo` y `listar_carpeta` se resuelven desde el directorio de trabajo del harness; `..` y las rutas absolutas son válidas, no se expande `~` ni `$HOME`, no se busca en otras ubicaciones y la normalización no elude las restricciones de acceso.
- Las búsquedas de archivo (`buscar_archivos` y `buscar_en_archivos`) recorren también las carpetas ocultas de contenido —`.agents`, `.localcli`, `.github`— porque ahí vive parte del proyecto; solo se saltan los sistemas de control de versiones (`.git`, `.hg`, `.svn`) y las dependencias (`node_modules`).
- El esquema instruye al modelo con la ruta relativa (conservando las subcarpetas, p. ej. `docs/specs/X.md`), la respuesta de `leer_archivo` devuelve la `ruta` canónica —relativa si el archivo está dentro del directorio de trabajo— y `~`/`$HOME` se rechazan con un error corregible que recuerda la convención; así el modelo no pega la ruta completa ni la reduce al nombre del archivo.
- `leer_archivo` no modifica ni elimina archivos y no ejecuta su contenido.
- Las ejecuciones son secuenciales, en el orden pedido.
- Un turno tiene un máximo de pasadas; al agotarlo, termina con lo conseguido.
- El catálogo incluido son diecisiete herramientas y **no se amplía desde fuera**: el catálogo está embebido en el código.
- `ejecutar_flujo` es una sola herramienta: `flujo` elige el comando y `peticion` el objetivo; su permiso se deriva del flujo, su esquema se arma por turno y solo lista los flujos que el agente activo puede lanzar.
- `ejecutar_flujo` corre el flujo como sub-proceso, conservando sus pausas de `continuacion` y `aprobacion`, y devuelve la entrega más el bloque resumido. El interior del flujo no cruza.
- `ejecutar_flujo` no está en el catálogo dentro de una etapa de flujo: un flujo no invoca a otro.
- El reparto sale de permisos (`read`, `write`, `edit`) declarados como `allow`/`deny`, con `default: deny`; el catálogo efectivo se deriva de ellos.
- Un `permissions` con `default: allow` no carga: el default solo admite `deny`.
- Un permiso o un efecto fuera de vocabulario no carga, no se ignora.
- Un modelo que no declara capacidad de herramientas no recibe ninguna y el agente cae a modo conversación, avisando en la interfaz. No se le impide usarlo.
- `plan` no tiene ninguna forma de escribir en el proyecto; sí mantiene la lista de pasos de la sesión.
- `build` tiene el catálogo completo.
- La lista de pasos de la sesión (la mantienen `crear_todo` y `actualizar_todo`) la escriben los dos agentes: es estado de la sesión, no un cambio en el proyecto.
- La lista de pasos se reescribe entera cuando llega `actualizar_todo`; lo que llega reemplaza lo anterior, y una lista vacía la deja en blanco.
- `crear_todo` solo añade: un paso suelto o una tarea principal con sus subtareas; no puede borrar ni tocar lo que ya había.
- La jerarquía es de un solo nivel: una tarea principal con sus subtareas, y una subtarea no tiene hijas. `parent_position` nulo es una fila raíz y, con valor, apunta a la posición de la principal en la misma sesión.
- `contenido` (paso suelto) y `titulo`+`subtareas` (tarea principal) se excluyen en `crear_todo`; mezclarlos es un error corregible.
- Solo un elemento de la lista puede estar `en_progreso` a la vez, y `completada` solo se marca tras verificar el paso.
- Un estado o una prioridad fuera del vocabulario de la lista vuelve al modelo como error corregible.
- El relevo de `plan` a `build` es explícito: propuesta, aprobación, cambio de agente, aplicación.
- Una aprobación vale para el cambio propuesto. No habilita nada más.
- Las herramientas de terminal nunca se saltan las reglas de permiso.
- La terminal no puede modificar nada del proyecto, y esa garantía no depende de revisar el comando.
- Por internet, solo sale la consulta, nunca nada del proyecto.
- El contenido que vuelve de internet se recorta y se audita igual que el resto.

## Criterios de aceptación

- [ ] El modelo pide herramientas por su nombre, con un esquema declarado, y no imitando un formato escrito.
- [ ] El catálogo no viaja en el mensaje de sistema.
- [ ] Una llamada con un campo obligatorio ausente no ejecuta nada y vuelve al modelo con el motivo.
- [ ] El modelo puede corregir la llamada y reintentarla en la misma pasada.
- [ ] La salida de toda herramienta se recorta, incluida la que no es un comando.
- [ ] Al recortar, el modelo sabe que se ha recortado y por dónde.
- [ ] Las ejecuciones de un turno son secuenciales y en el orden pedido.
- [ ] Un turno se detiene al llegar al máximo de pasadas, y avisa de que se agotaron.
- [ ] La línea de cada herramienta dice cuánto tardó su ejecución, y ese tiempo no viaja al modelo.
- [ ] `plan` puede leer, listar, buscar, ejecutar comandos de consulta, buscar en internet y mantener su lista de pasos.
- [ ] `diagram` figura en el catálogo bajo `read` y lo pueden usar los dos agentes; su contrato se valida contra [[specs/SPEC-DIAGRAMA]].
- [ ] `plan` no tiene ninguna herramienta que cree, modifique o borre.
- [ ] `build` tiene el catálogo completo.
- [ ] `plan` y `build` mantienen la lista de pasos de la sesión, y la lista se ve en el panel.
- [ ] `crear_todo` añade al final —un paso suelto o una tarea principal con sus subtareas— y no cambia los que ya había.
- [ ] `crear_todo` sin `subtareas` se comporta como un paso suelto, igual que antes.
- [ ] `crear_todo` con `titulo` y `subtareas` crea la tarea principal y todas sus subtareas, en orden y como hijas de la principal.
- [ ] `crear_todo` con `contenido` vacío no añade nada y devuelve el motivo.
- [ ] `crear_todo` con `contenido` junto a `titulo` o `subtareas` no añade nada y devuelve el motivo.
- [ ] `crear_todo` devuelve la lista resultante, sin importar si la lista estaba vacía.
- [ ] `crear_todo` con un `estado` o una `prioridad` fuera del vocabulario devuelve al modelo un error corregible y no guarda nada.
- [ ] `crear_todo` con un segundo paso `en_progreso` no lo añade y le dice al modelo que use `actualizar_todo`.
- [ ] `actualizar_todo` reemplaza la lista entera; una lista vacía la deja en blanco.
- [ ] Los elementos de `actualizar_todo` pueden llevar sus `subtareas`; sin ellas, la llamada se comporta como antes.
- [ ] Una subtarea no tiene hijas: la jerarquía se queda en un solo nivel.
- [ ] El panel anida las subtareas bajo su tarea principal.
- [ ] Un estado o una prioridad fuera del vocabulario devuelve al modelo un error corregible y no guarda nada.
- [ ] Nada se escribe sin que `plan` lo haya propuesto y tú lo hayas aprobado.
- [ ] Una aprobación no habilita cambios distintos de los propuestos.
- [ ] `crear_archivo` falla si el archivo ya existe; `escribir_archivo` lo sobrescribe.
- [ ] Un archivo que no supera el límite de lectura completa se devuelve entero, sin aviso de truncado.
- [ ] Un archivo que supera el límite de lectura completa se devuelve como fragmento inicial, marcado como truncado, con la línea desde la que continuar.
- [ ] Una lectura por rango devuelve exactamente las líneas solicitadas, numeradas desde 1 y tomadas del archivo original.
- [ ] Una lectura por rango que excede el máximo por respuesta devuelve hasta donde cabe e indica dónde terminó, sin pasar del límite.
- [ ] `leer_archivo` informa el total de líneas cuando puede determinarlo y señala una línea individual demasiado grande.
- [ ] Los errores de `leer_archivo` distinguen archivo inexistente, ruta que es un directorio, falta de permisos, archivo no legible como texto y parámetros o rangos inválidos.
- [ ] Las rutas relativas de `leer_archivo` y `listar_carpeta` se resuelven desde el directorio de trabajo; `..` y las absolutas se aceptan; `~` y `$HOME` no se expanden; no se busca en otras ubicaciones.
- [ ] Una ruta que empieza por `~` o contiene `$HOME` se rechaza con `E_BAD_ARGS` y un aviso que recuerda usar rutas relativas, no con «archivo no encontrado».
- [ ] La respuesta de `leer_archivo` devuelve `ruta` relativa al directorio de trabajo cuando el archivo está dentro de él, y absoluta cuando está fuera.
- [ ] Una ruta que resuelve fuera de la carpeta sigue exigiendo permiso y explicación, sin que la normalización lo evite.
- [ ] Los comandos de la lista blanca se ejecutan sin pedir permiso.
- [ ] Un comando fuera de la lista blanca pide permiso antes de ejecutarse.
- [ ] La terminal no puede crear, editar ni borrar archivos del proyecto, ni con comandos indirectos.
- [ ] La terminal sí puede compilar, correr pruebas y revisar estilo y tipos.
- [ ] No puede descartar cambios del repositorio ni hacer commits.
- [ ] Un comando sin fin o con salida sin fin se corta y se avisa, sin colgar el harness.
- [ ] La búsqueda por internet envía solo la consulta, nunca contenido del proyecto.
- [ ] `abrir_pagina` devuelve el contenido de una página y ese contenido se audita.
- [ ] Un modelo que no declara capacidad de herramientas no recibe ninguna y el turno responde en modo conversación.
- [ ] Bajo la entrada se ve si el modelo en uso puede usar herramientas, y si no puede se dice que el agente va a conversar.
- [ ] En el chat, el agente activo usa exactamente las herramientas derivadas de sus permisos.
- [ ] El reparto sale de `read`, `write` y `edit`: `editar_archivo` va a `edit`, y crear, escribir, eliminar y las carpetas van a `write`.
- [ ] `ejecutar_flujo` se pide por el canal nativo, con `flujo` y `peticion`, y su esquema se arma por turno con los flujos que el agente activo puede lanzar.
- [ ] Con `read`, el agente lanza los flujos cuyas etapas corren todas con `plan`; un flujo con alguna etapa `build` solo se lista para quien tiene `write`/`edit`.
- [ ] Al lanzar `ejecutar_flujo`, el turno espera mientras el flujo corre; las pausas de `continuacion` y `aprobacion` se conservan.
- [ ] El resultado de `ejecutar_flujo` es la entrega más el bloque resumido; ninguna etapa ni herramienta interna del flujo aparece en el contexto del agente que lo invocó.
- [ ] Dentro de una etapa de flujo, `ejecutar_flujo` no está en el catálogo: un flujo no invoca a otro.
- [ ] En pantalla, el lanzamiento se anuncia como `FLUJO [<comando>] · <peticion>` y después entran las líneas de sub-proceso de las etapas.
- [ ] Mientras una herramienta se ejecuta, se ve en pantalla cuál es —su verbo y su objetivo (la ruta, el patrón, el comando)— y, al terminar, si fue bien y el tamaño del resultado («70 líneas»).

## Requisitos no funcionales

- Ninguna respuesta del modelo se agota por un comando descuidado.
- Ninguna respuesta del modelo se agota por una salida grande de cualquier herramienta.
- El contenido de una página web cuenta para el mismo presupuesto de contexto que el resto.
- Una espera de aprobación de una sesión no impide que otra sesión genere.

## Dependencias funcionales

- [[specs/SPEC-ARCHIVOS]]
- [[specs/SPEC-NODO-CONTEXTO]]
- [[specs/SPEC-MODELOS-LLAMACPP]]

## Supuestos

- Los comandos concretos de la lista blanca, el límite de tiempo y el límite de salida se fijan en FASE 2 y FASE 3.
- El límite de recorte de una salida de herramienta se fija en FASE 2.
- El número máximo de pasadas por turno se fija en FASE 2.

## Referencias

- [[IDEA]]
- [[specs/SPEC-DIAGRAMA]] — el contrato de la herramienta `diagram`, incluida en el catálogo.
