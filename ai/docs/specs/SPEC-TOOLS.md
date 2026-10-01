---
title: SPEC — Catálogo de herramientas
tags: [specs, requisito]
depende_de:
  - "[[IDEA]]"
  - "[[specs/SPEC-AGENTE-BASE]]"
  - "[[specs/SPEC-ARCHIVOS]]"
  - "[[specs/SPEC-NODO-CONTEXTO]]"
  - "[[specs/SPEC-RESOLVER]]"
  - "[[specs/SPEC-MODELO-PROVEEDOR]]"
relacionado:
  - "[[backend/DECISIONS]]"
  - "[[specs/SPEC-SKILLS]]"
---
# SPEC — Catálogo de herramientas

Prioridad: P0 (núcleo)

## Propósito

Definir qué herramientas tiene el agente para trabajar, cuáles puede usar para mirar y cuáles para escribir, en qué casos necesita permiso, cómo se le pide una herramienta al modelo y cómo puede el usuario añadir las suyas sin tocar el código.

## Alcance

Incluye el catálogo incluido, el reparto por agente, las reglas de la terminal y de internet, la capa universal que envuelve toda ejecución, el descubrimiento de herramientas del usuario en `.localcli/tools/` y la degradación cuando el modelo no sabe usarlas.
No incluye las reglas de permiso sobre archivos, que están en [[specs/SPEC-ARCHIVOS]], ni la selección de contexto, que está en [[specs/SPEC-NODO-CONTEXTO]], ni la conexión con el proveedor de modelo, que está en [[specs/SPEC-MODELO-PROVEEDOR]].

## Actores

- **Usuario**: aprueba lo que lo requiere y define sus propias herramientas.
- **Agente `plan`**: solo mira, propone y pide aprobación.
- **Agente `build`**: escribe, modifica y borra.
- **Sistema**: ejecuta la herramienta y devuelve el resultado.
- **Usuario como extensor**: declara herramientas propias en `.localcli/tools/`.

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

### Sesión

| Herramienta | Lee o escribe | Permiso | Agente |
|---|---|---|---|
| `crear_todo` | No toca el proyecto | `read` | Ambos |
| `actualizar_todo` | No toca el proyecto | `read` | Ambos |

Son las dos que mantienen la lista de pasos de la sesión: el plan de trabajo de varios pasos que el usuario ve en el panel. No escriben en el proyecto —es estado de la sesión—, así que la tienen los dos agentes.

**Una añade y la otra reemplaza, y no se confunden.** `crear_todo` **añade un paso al final** de la lista y nada más: no toca los que ya había. `actualizar_todo` **sustituye la lista entera** por la que mande. La primera es la vía normal para planning; la segunda, para reordenar, cancelar un paso o cambiar estados sin añadir.

La separación no es cosmética. Con una sola herramienta que sustituye, un modelo que Cree estar «actualizando un paso» reescribe la lista con lo que recuerda y **borra los demás sin saberlo**. Con `crear_todo` delante, añadir es un verbo que no puede perder nada, y el modelo solo llama a `actualizar_todo` cuando de verdad quiere reescribir todo.

Ninguna de las dos necesita saber si la lista existe: **las dos devuelven la lista resultante**, así que el modelo tiene el estado en su propio contexto y no tiene que preguntar. Y ninguna necesita un identificador de paso: el orden es la posición en la lista, y un identificador solo serviría para que un modelo lo cite mal.

Ver «La lista de pasos» para las reglas de estado.

### Del usuario

Las que el usuario declare en `.localcli/tools/`. Ver «Herramientas del usuario». No viven en este catálogo: se añaden al construir el registro y se reparten con las mismas reglas.

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

La medición va **en la capa universal, no en cada herramienta**: es el único punto por el que pasan todas las ejecuciones, incluidas las del usuario, así que ninguna tiene que llevar un reloj propio para que su línea diga cuánto tardó. El tiempo medido es el de la ejecución —lo que tardó el handler—, no el del turno: el turno incluye el modelo, y son dos cosas distintas.

### Errores que el modelo puede corregir

Un error de validación **no termina el turno**: vuelve al modelo como un resultado con el motivo y con lo que se esperaba, de forma que pueda corregir la llamada y reintentarlo en la misma pasada.

```
falta el campo obligatorio "ruta"
"editar_archivo" espera: ruta (obligatorio), contenido (obligatorio), reemplazos (opcional)
```

Esto vale también para un error de negocio —una ruta que no existe, un archivo que ya está— siempre que el mensaje sea entendible por el modelo y no exponga rutas absolutas del usuario.

Un fallo que el modelo no puede corregir por sí mismo —una denegación, un permiso del sistema, un fallo de red— se le dice igualmente, pero no se le reintenta: se lo informa y sigue.

### Recorte de la salida

**Toda** salida de herramienta se recorta antes de volver al modelo, no solo la de la terminal. Una lectura enorme, un volcado de tabla, una respuesta de API gigante.

El recorte no es un castigo: la herramienta que lo ha hecho avisa de que lo ha hecho, y el modelo lo sabe para no agir como si tuviera el resultado entero. Cuando se corta por la mitad, se dice por dónde.

## Herramientas del usuario

El usuario puede añadir herramientas suyas sin tocar el código: deja un `.json` en `.localcli/tools/` y aparecen junto a las incluidas.

Cada archivo declara una herramienta:

```json
{
  "nombre": "contar_lineas",
  "descripcion": "Cuenta las líneas de un archivo de texto.",
  "modo": "lee",
  "equipo": ["wc", "-l"]
}
```

| Campo | Qué es |
|---|---|
| `nombre` | Cómo se llama. No puede coincidir con una incluida. |
| `descripcion` | Lo que lee el modelo para decidir si la usa. |
| `modo` | `lee`. El valor `escribe` existe en el contrato pero se rechaza; ver más abajo. |
| `equipo` | El comando, como lista de argumentos. No es una línea de shell. |

`equipo` es una lista, no un texto, y no pasa por un shell. No hay tubería, ni redirección, ni `sh -c` que puedan colarse: el ejecutable y sus argumentos son exactamente los escritos, y un argumento con un `;` es un argumento con un punto y coma.

### Qué puede y qué no puede una herramienta del usuario

- **Siempre pide aprobación.** Los comandos que ejecuta son los que el usuario ha escrito, no los que la herramienta conoce, así que ninguno está en la lista blanca: se pide siempre. Sin excepción.
- **No puede escribir en el proyecto.** Corre bajo el mismo aislamiento que la terminal, así que el bloqueo es el mismo: aunque su comando intente escribir, no puede. Un archivo con `"modo": "escribe"` **se rechaza al arrancar** diciendo por qué: la única vía sancionada para escribir es la de las herramientas de archivo, que solo tiene `build` y que sí pasa por aprobación.
- **No puede tocar tus archivos** más allá de lo que permite la terminal.
- **No entra en la lista blanca** en ningún caso.
- **No puede definir herramientas nuevas**: no llama a otras herramientas ni amplía el catálogo.
- **Un JSON inválido se ignora** y se avisa, igual que un agente o un flujo inválido. El arranque no se detiene.

### Nombres

Una herramienta del usuario no puede llamarse igual que una incluida. Si el archivo se llama `contar_lineas.json` y declara `"nombre": "contar_lineas"`, el nombre que ve el modelo es ese. Si declarara `leer_archivo`, no se carga: choca con una incluida y se avisa.

## El reparto: `plan` mira, `build` escribe

El reparto sale de los `permissions` de cada agente, que se declaran **por permiso**, con tres: `read` (mirar el proyecto, la terminal de consulta, internet y la lista de pasos de la sesión), `write` (crear, sobrescribir o borrar archivos y carpetas) y `edit` (parchear contenido existente). El catálogo efectivo de herramientas se deriva de ahí contra el catálogo: no hay una lista de herramientas suelta que pueda contradecir los permisos. El detalle de qué herramienta cae en qué permiso está en «Quién puede usar qué» de [[specs/SPEC-AGENTE-BASE]].

`plan` permite solo `read` y deniega los otros dos. Nunca recibe una herramienta que escriba en el proyecto. Existe para entender el proyecto, decidir qué hay que hacer y proponerlo.

`build` permite los tres y recibe el catálogo completo. Es el único que crea, modifica y borra.

Esto no es una restricción de estilo: es la garantía de que nada cambia en tu proyecto sin que `plan` lo haya propuesto antes y tú lo hayas aprobado.

**Las herramientas del usuario respetan el mismo reparto.** Una declarada `"modo": "escribe"` nunca llega a `plan`, y las de lectura llegan a los dos. Extender el catálogo no abre una puerta trasera a la garantía.

## La lista de pasos

La lista de pasos de la sesión es estado de la sesión, no del proyecto, y por eso la tienen los dos agentes bajo el permiso `read`. La mantienen dos herramientas, y la diferencia entre las dos es qué le pasa a lo que ya había.

`crear_todo` **añade un paso al final**. Recibe `contenido` —qué hay que hacer— y `estado`; `prioridad` es opcional y por defecto es `media`. No recibe los pasos anteriores: no los necesita, porque no los toca. Lo que hace es insertar uno y devolver la lista completa, así que añadir un paso nunca puede perder los que ya estaban.

`actualizar_todo` **sustituye la lista entera** por la que mande, en un solo paso atómico. Es la herramienta para reordenar, cancelar un paso, cambiar estados de golpe o dejar la lista vacía: casos en los que «añadir un paso» no basta porque hay que quitar o cambiar algo de lo anterior.

El vocabulario es cerrado y lo comparten las dos: el estado de un paso es `pendiente`, `en_progreso`, `completada` o `cancelada`, y la prioridad es `alta`, `media` o `baja`. Un estado o una prioridad fuera de ese conjunto no se guarda: se devuelve al modelo el motivo y lo reintenta, como cualquier otro argumento mal formado.

Y se mantiene la regla de que **solo un paso puede estar `en_progreso` a la vez**, que hasta ahora era una instrucción en la descripción de la herramienta y no una comprobación. Añadir un paso ya en curso cuando otro lo está tiene que ser un error corregible, no un conflicto silencioso: `crear_todo` lo rechaza y le dice al modelo que use `actualizar_todo` para cambiar el que está en curso.

Ninguna de las dos herramientas necesita preguntar por el estado actual de la lista, y por eso devuelven la lista resultante. El modelo la ve y decide el siguiente paso sin una ronda extra. Ver [[database/01-schema/TABLES]] para cómo se guarda.

## Cuando el modelo no sabe usar herramientas

No todos los modelos que se pueden elegir en el proveedor saben pedir herramientas. La capacidad se puede preguntar antes de elegir, y hay modelos que responden que no.

Un modelo sin esa capacidad **no puede usar herramientas, ni las incluidas ni las del usuario**, porque no hay forma nativa de pedirlas y LocalCli no las presenta en texto para que las imite.

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
- Las ejecuciones son secuenciales, en el orden pedido.
- Un turno tiene un máximo de pasadas; al agotarlo, termina con lo conseguido.
- El catálogo incluido son quince herramientas y no se amplía desde fuera sin declararlas en `.localcli/tools/`.
- Las herramientas del usuario se cargan de `.localcli/tools/*.json`; un archivo inválido se ignora y el arranque sigue.
- Una herramienta del usuario se ejecuta como una lista de argumentos, sin shell.
- Una herramienta del usuario siempre pide aprobación y nunca entra en la lista blanca.
- Una herramienta del usuario no puede escribir en el proyecto: se ejecuta bajo el mismo aislamiento que la terminal.
- Un `"modo": "escribe"` en una herramienta del usuario se rechaza al arrancar, con el motivo.
- Una herramienta del usuario no puede llamar a otras herramientas ni ampliar el catálogo.
- El nombre de una herramienta del usuario no puede coincidir con una incluida; si coincide, no se carga.
- Las herramientas del usuario se reparten con las mismas reglas que las incluidas: `plan` nunca recibe una de escritura.
- El reparto sale de permisos (`read`, `write`, `edit`) declarados como `allow`/`deny`, con `default: deny`; el catálogo efectivo se deriva de ellos.
- Un `permissions` con `default: allow` no carga: el default solo admite `deny`.
- Un permiso o un efecto fuera de vocabulario no carga, no se ignora.
- Un modelo que no declara capacidad de herramientas no recibe ninguna y el agente cae a modo conversación, avisando en la interfaz. No se le impide usarlo.
- `plan` no tiene ninguna forma de escribir en el proyecto; sí mantiene la lista de pasos de la sesión.
- `build` tiene el catálogo completo.
- La lista de pasos de la sesión (`actualizar_todo`) la escriben los dos agentes: es estado de la sesión, no un cambio en el proyecto.
- La lista de pasos se reescribe entera: lo que llega reemplaza lo anterior, y una lista vacía la deja en blanco.
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
- [ ] `plan` no tiene ninguna herramienta que cree, modifique o borre.
- [ ] `build` tiene el catálogo completo.
- [ ] `plan` y `build` mantienen la lista de pasos de la sesión, y la lista se ve en el panel.
- [ ] `crear_todo` añade un paso al final y no cambia los que ya había.
- [ ] `crear_todo` con `contenido` vacío no añade nada y devuelve el motivo.
- [ ] `crear_todo` devuelve la lista resultante, sin importar si la lista estaba vacía.
- [ ] `crear_todo` con un `estado` o una `prioridad` fuera del vocabulario devuelve al modelo un error corregible y no guarda nada.
- [ ] `crear_todo` con un segundo paso `en_progreso` no lo añade y le dice al modelo que use `actualizar_todo`.
- [ ] `actualizar_todo` reemplaza la lista entera; una lista vacía la deja en blanco.
- [ ] Un estado o una prioridad fuera del vocabulario devuelve al modelo un error corregible y no guarda nada.
- [ ] Nada se escribe sin que `plan` lo haya propuesto y tú lo hayas aprobado.
- [ ] Una aprobación no habilita cambios distintos de los propuestos.
- [ ] `crear_archivo` falla si el archivo ya existe; `escribir_archivo` lo sobrescribe.
- [ ] Los comandos de la lista blanca se ejecutan sin pedir permiso.
- [ ] Un comando fuera de la lista blanca pide permiso antes de ejecutarse.
- [ ] La terminal no puede crear, editar ni borrar archivos del proyecto, ni con comandos indirectos.
- [ ] La terminal sí puede compilar, correr pruebas y revisar estilo y tipos.
- [ ] No puede descartar cambios del repositorio ni hacer commits.
- [ ] Un comando sin fin o con salida sin fin se corta y se avisa, sin colgar el harness.
- [ ] La búsqueda por internet envía solo la consulta, nunca contenido del proyecto.
- [ ] `abrir_pagina` devuelve el contenido de una página y ese contenido se audita.
- [ ] Un `.localcli/tools/<nombre>.json` válido añade una herramienta disponible sin tocar el código.
- [ ] La herramienta del usuario se ejecuta sin shell, con exactamente los argumentos declarados.
- [ ] La herramienta del usuario pide aprobación siempre, aunque su comando esté en la lista blanca.
- [ ] La herramienta del usuario no puede escribir en el proyecto, ni declarándolo ni con un comando indirecto.
- [ ] Un `"modo": "escribe"` en una herramienta del usuario se rechaza al arrancar, diciendo por qué.
- [ ] Una herramienta del usuario con el nombre de una incluida no se carga y se avisa.
- [ ] Un `.localcli/tools/*.json` inválido se ignora y el arranque sigue.
- [ ] Las herramientas del usuario siguen el reparto: `plan` no recibe ninguna de escritura.
- [ ] Un modelo que no declara capacidad de herramientas no recibe ninguna y el turno responde en modo conversación.
- [ ] Bajo la entrada se ve si el modelo en uso puede usar herramientas, y si no puede se dice que el agente va a conversar.
- [ ] En el chat, el agente activo usa exactamente las herramientas derivadas de sus permisos.
- [ ] El reparto sale de `read`, `write` y `edit`: `editar_archivo` va a `edit`, y crear, escribir, eliminar y las carpetas van a `write`.
- [ ] Mientras una herramienta se ejecuta, se ve en pantalla cuál es —su verbo y su objetivo (la ruta, el patrón, el comando)— y, al terminar, si fue bien y el tamaño del resultado («70 líneas»).

## Requisitos no funcionales

- Ninguna respuesta del modelo se agota por un comando descuidado.
- Ninguna respuesta del modelo se agota por una salida grande de cualquier herramienta.
- El contenido de una página web cuenta para el mismo presupuesto de contexto que el resto.
- Una espera de aprobación de una sesión no impide que otra sesión genere.

## Dependencias funcionales

- [[specs/SPEC-AGENTE-BASE]]
- [[specs/SPEC-ARCHIVOS]]
- [[specs/SPEC-NODO-CONTEXTO]]
- [[specs/SPEC-RESOLVER]]
- [[specs/SPEC-MODELO-PROVEEDOR]]

## Supuestos

- Los comandos concretos de la lista blanca, el límite de tiempo y el límite de salida se fijan en FASE 2 y FASE 3.
- El límite de recorte de una salida de herramienta se fija en FASE 2.
- El número máximo de pasadas por turno se fija en FASE 2.
- Si en el futuro hacen falta herramientas del usuario con capacidad de escritura, hará falta decidir explícitamente un nivel de privilegio superior al de la terminal. Hoy no existe.

## Referencias

- [[IDEA]]
