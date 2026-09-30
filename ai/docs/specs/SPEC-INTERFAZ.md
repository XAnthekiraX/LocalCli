---
title: SPEC — Interfaz
tags: [specs, requisito]
depende_de:
    - "[[IDEA]]"
    - "[[specs/SPEC-INTERFAZ-ATAJOS]]"
    - "[[specs/SPEC-KEYBINDS]]"
    - "[[specs/SPEC-PANEL-CONTEXTO]]"
    - "[[specs/SPEC-SESIONES]]"
    - "[[specs/SPEC-COLA-TAREAS]]"
    - "[[specs/SPEC-TOOLS]]"
relacionado:
    - "[[specs/SPEC-ARCHIVOS]]"
---

# SPEC — Interfaz

Prioridad: P0 (núcleo)

## Propósito

Definir dónde vive cada cosa en la pantalla: qué se ve, dónde está y cómo se recorre.

## Alcance

Incluye la disposición de la pantalla, las zonas —incluida la paleta de comandos de flujo de la entrada—, los datos que muestra cada una, la pantalla de bienvenida y cómo se cambia de sesión.
No incluye los atajos de teclado, que están en [[specs/SPEC-INTERFAZ-ATAJOS]] ni el mecanismo de resolución de teclas y la tecla líder, que está en [[specs/SPEC-KEYBINDS]], ni el significado y el cálculo de los números de contexto, que están en [[specs/SPEC-PANEL-CONTEXTO]], ni las reglas de permiso, que están en [[specs/SPEC-ARCHIVOS]].

## Actores

- **Usuario**: lee, escribe y navega.

## Disposición

El chat ocupa la columna principal. El sidebar de datos está pegado a la derecha,
separado por **dos columnas** con el fondo de la pantalla. La entrada de texto vive en
una caja con fondo al pie de la columna principal, separada del chat por **dos filas**
(el divisor y el borde superior de la caja), también con el fondo de la pantalla. Cada
mensaje del chat va en un bloque con el fondo del color de quien habla y un icono a su
lado: el del agente delante y el del usuario detrás, pegado a la derecha.

Toda la pantalla es una **superficie rectangular continua**: cada fila ocupa el ancho
completo de la terminal y cada celda tiene fondo (lo garantiza la capa común de render,
`marco.go`), de modo que la selección con el ratón cubre el área completa y la copia trae
solo el texto. Las separaciones son celdas del propio fondo de la pantalla, no líneas ni
bandas de otro color.

```
  ▣   respuesta del agente              api de pedidos
                                       (trabajando)
         lo que escribes  ▣            CONTEXTO · TODO
                                       LISTA DE TAREAS
                                       ESTADO
 En qué te ayudo hoy: █                Capa y cola · Agente
 [plan] · * llama3.2 · herr …          Git · [~/ruta/proyecto]
                                       LocalCli · v0.1
```

Con el sidebar cerrado, el chat ocupa todo el ancho y la caja de entrada también.

El sidebar **arranca visible** y `Ctrl+D` lo pliega y lo despliega: es la disposición
normal, no un extra. Plegado, el chat recupera todo el ancho.

El sidebar es de lectura. No se escribe nada desde ahí. La entrada de texto es una caja
con fondo propio y, dentro, debajo del texto, la línea del agente activo y el modelo en
uso con sus capacidades.

## Pantalla de bienvenida

Es la primera vista al ejecutar `localcli`, antes de que exista conversación: el logotipo ASCII sobre el fondo, una caja de entrada con su línea de agente y modelo, la barra de pistas de teclado y la versión abajo a la derecha, como en opencode. El entorno del logotipo es la propia pantalla; solo la caja de entrada tiene superficie propia —un rectángulo sin glifos de borde—, de modo que la selección con el ratón cubre el área completa y la copia trae solo el texto.

```
              < LOGOTIPO 6×53 >


░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░
  "Escribe para iniciar la conversacion"
  [plan] • llama3.2
░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░

   TAB agentes   Ctrl+P comandos   Ctrl+X M modelos

                                        v0.1 alpha
```

Las `░` marcan las bandas de fondo que enmarcan la caja de entrada (arriba y abajo): no son caracteres que se pinten. La caja de entrada usa un fondo algo distinto al de la pantalla; el espacio alrededor del logotipo es el fondo. La barra de pistas se genera desde el mapa de teclas vigente; la versión firma la esquina inferior derecha. `Tab` alterna el agente activo (`plan` o `build`) y la línea de estado lo refleja al instante.

### Arte canónico del logotipo

El logotipo es fijo y esta es su definición, que sirve de salida dorada para las pruebas:

- Texto `LocalCli` en bloques de 6 filas × 53 columnas, con los caracteres de bloque `█ ╗ ╝ ║ ╔ ═ ╚` en UTF-8.
- La copia canónica byte a byte vive en `internal/tui/testdata/logo.txt`: seis líneas de exactamente 53 columnas, sin espacios finales. El bloque de arriba es su render fiel; la comparación de la prueba va contra el archivo.
- El arte va centrado sobre el fondo de la pantalla, sin caja ni borde; su entorno es el propio fondo.
- El arte no se reescala: las 6 filas × 53 columnas se pintan íntegras, sin recorte. El conjunto (logotipo, caja de entrada y barra de pistas) se centra en la terminal, en horizontal y en vertical. En una terminal estrecha el estilo no cambia: las cajas nunca se componen por debajo del arte (la terminal recorta lo que no quepa). El bloque llano —logotipo, nombre con versión, línea de modelo y línea de entrada— queda solo para antes de la primera `WindowSizeMsg`.
- No entra al contexto del modelo ni a ningún historial: es arte de la aplicación.

- Solo hay el logotipo sobre el fondo, la caja de entrada con su línea de agente y modelo, la barra de pistas de teclado y la versión. Ni paneles ni datos. Los modelos NO se listan en la bienvenida: se consultan y eligen desde un modal.
- **Caja de entrada**: mientras está vacía muestra el placeholder `"Escribe para iniciar la conversacion"` entre comillas; lo escrito se pinta con su cursor. Bajo el texto va la **línea de estado** con el agente activo y el modelo en uso (`[plan] • llama3.2`). La **barra de pistas** recuerda los atajos vigentes (`TAB agentes`, `Ctrl+P comandos`, `Ctrl+X M modelos`), generados desde el mapa de teclas. El placeholder y las etiquetas de la barra (`agentes`, `comandos`, `modelos`) van en un tono sutil propio (`#828BB8`); las teclas (`TAB`, `Ctrl+P`, `Ctrl+X M`) y el modelo en uso, en blanco. Una línea en blanco separa el texto escrito del modelo en la caja. No hay lista de modelos visible en la bienvenida.
- **Modal de modelos**: la secuencia líder `Ctrl+X` seguida de `m` abre un modal centrado con la lista de modelos locales que reporta Ollama, con uno resaltado. `↑`/`↓` mueven la selección, `Enter` aplica el elegido y cierra el modal, `Esc` cierra sin cambiar nada. La lista se pide a Ollama al abrir el modal (no en el arranque); si Ollama no responde, el modal muestra el aviso «sin modelos» y se puede cerrar con `Esc` sin bloquear nada. Mientras el modal está abierto, las teclas son solo del modal: la escritura de la bienvenida no las recibe. Lo elegido pasa al motor como modelo de la sesión y se refleja en la línea de modelo de la bienvenida. La elección del usuario prevalece sobre la autodetección del arranque ([[specs/SPEC-OLLAMA-PERFIL]]: el modelo lo elige el usuario). El mecanismo de la secuencia está en [[specs/SPEC-KEYBINDS]].
- Lo que se escribe es la **primera petición de la sesión**: se envía tal cual, igual que se enviaría desde el chat.
- La línea de entrada de la bienvenida despliega también la **paleta de comandos de flujo** al escribir `/` (§Zonas 2): escribir filtra, `↑`/`↓` la recorren, `Tab` autocompleta y `Enter` ejecuta. Un comando crea la sesión y lleva a la vista principal igual que una primera petición.
- Al enviarla, la vista cambia a la interfaz principal y la petición aparece como primer mensaje del chat. La transición no repite la petición ni pide confirmación.
- No hay una sesión activa antes de usar la bienvenida: la primera petición **crea** una sesión nueva, con un nombre provisional que el modelo sustituye por un título generado a partir de esa misma petición. Ver [[specs/SPEC-SESIONES]].
- Desde la bienvenida no hay panel de contexto ni de aprobaciones: hay el logotipo enmarcado, la caja de entrada y la barra de pistas; con `Ctrl+X m` se abre el modal de modelos y con `Ctrl+X l` el de sesiones (al elegir una con `Enter`, la vista pasa a la principal con el historial de esa sesión). Se escribe y se envía —o se ejecuta un comando de flujo— y la salida es `Ctrl+C`.

## Zonas

### 1. Chat

- Ocupa la columna principal (el ancho total menos el sidebar cuando está abierto).
- Muestra el historial de la sesión activa.
- Cada mensaje ocupa una fila de ancho completo: una franja de **una columna** con el color de quien habla —azul `#606CD5` el usuario en su borde derecho, verde `#4CEE75` el agente en el izquierdo— y el resto de la fila en el fondo del chat (`#1E2030`); el texto va en **blanco** dentro, con una celda de aire arriba, abajo y a cada lado, sin icono. Entre un mensaje y el siguiente va una fila en blanco. Sin bordes ni glifos de adorno. El área del chat entre mensajes mantiene el fondo de la pantalla; solo las filas de mensaje llevan `#1E2030`. Las líneas del sistema —herramientas, avisos— se pintan sueltas.
- El hilo se pinta en el **orden en que ocurrió**: el texto que el modelo escribió antes de usar una herramienta queda arriba de su línea, y el que escribe después abre un globo nuevo. Los segmentos de texto de un mismo turno no se funden en uno solo por debajo de las líneas de herramienta.
- Cada intercambio muestra la respuesta del modelo; mientras genera, un **indicador en vivo** (`[⠋ Pensando]`) sustituye al volcado del razonamiento. El texto del razonamiento se revela con `Ctrl+R` (ver §Razonamiento del modelo). El intercambio del turno en vivo —razonamiento y respuesta según llegan— es un bloque más del chat: crece dentro de la ventana del historial, así que se recorre con el scroll y nunca empuja la caja de entrada fuera de la pantalla.
- Cada respuesta del modelo muestra cuánto tardó en llegar, atenuado junto a ella. Mientras se espera, el tiempo corre en pantalla —junto al indicador en vivo— para saber que el modelo sigue trabajando.
- Cada línea de herramienta muestra cuánto tardó esa ejecución. Usa **la misma forma de medir el número y el mismo atenuado** que el tiempo de la respuesta, pero va unido con el separador de la línea (`· 0.4 s`) en vez de entre paréntesis, porque el paréntesis es lo que marca «esto es el tiempo de una respuesta» y ahí no lo es. Los dos tiempos son cosas distintas y se lucie por separado: el de la respuesta es del turno entero, y el de la herramienta es solo de su ejecución.
- Muestra las propuestas pendientes de aprobación.
- El historial se puede recorrer: `↑`/`↓` suben y bajan línea a línea y `pgup`/`pgdown` por páginas; la rueda del ratón también desplaza. Mientras no se sube, la vista sigue el final y baja sola con cada respuesta nueva; al subir se respeta la posición. Cuando queda historial fuera de la ventana se indica con una línea discreta («↑ N líneas arriba» / «↓ N líneas abajo»).

### 2. Entrada de texto

- Una línea de escritura que crece en filas: el texto es un solo párrafo, pero al desbordar el ancho disponible salta de renglón (hasta un tope de filas; a partir de ahí se desplaza dentro de la ventana) en vez de recortarse. Al redimensionar la terminal, el reparto se reajusta. `Enter` envía la petición; no inserta saltos de línea.
- Escribe hacia la sesión activa.
- **Paleta de comandos de flujo.** Al escribir `/` se despliega encima de la línea la lista de comandos de flujo disponibles —los oficiales más los propios de `.localcli/flows/*.json`—, con el resaltado en el primero. Lo escrito filtra la lista; `↑`/`↓` la recorren, `Tab` autocompleta el comando resaltado dejando la línea lista para escribir la petición detrás (`/comando [petición]`) y `Enter` ejecuta. En cuanto se escribe un espacio (empieza la petición) la paleta se retira. Es un ayudante para descubrir los comandos, no una vía nueva de arranque: el flujo sigue arrancando solo con su comando explícito y el catálogo lo sirve el motor.
- El texto se edita en cualquier punto: las flechas mueven el cursor, `home`/`end` van al principio y al final y `ctrl+b`/`ctrl+e` son sus equivalentes. Los atajos que coincidan con una acción (por ejemplo `ctrl+a`) siguen resolviéndose como acción y no editan.
- La **caja de la entrada** lleva el texto con una celda de aire arriba, abajo y a cada lado, y bajo él una línea de estado con el agente activo, el modelo en uso y sus chapas: `tool [*]` en verde si usa herramientas y en rojo si no (atenuada mientras no se sabe), `pensar [x]`/`pensar [ ]` —el interruptor de razonamiento, apagado por defecto y **pulsable con el ratón**; solo se enseña si el modelo declara la capacidad `thinking`—, `[v]` si acepta visión y `[T]` siempre (texto). El fondo de la caja es `#1E2030`. Cuando hay consumo que mostrar, el conteo de tokens del turno (`tokens: 54k`) se pinta justo debajo de la caja. La caja queda **siempre pegada al pie** de la columna, aunque el historial sea corto (el chat se rellena con celdas de fondo hasta la separación). La línea de entrada de la bienvenida se edita igual que esta.
- Justo encima de la caja, mientras el modelo trabaja, se pinta la **línea de actividad**: un glifo que gira y la etiqueta de lo que pasa —`[⠋ Pensando]` si aún no hay respuesta, `[⠋ Usando herramienta: X]` si corre una herramienta, `[⠋ Generando]` si ya llega respuesta— más el tiempo transcurrido. Sustituye al volcado crudo del razonamiento.
- **Indicador de agente en el pie de la caja**: en la línea de estado dentro de la caja de entrada se muestra el agente activo (p. ej. `[plan]`). Cambia al instante con `Tab`, que recorre los agentes disponibles (los base `plan` y `build` más los que el usuario añada en `.localcli/agents/*.json`). En la bienvenida el indicador va delante de la línea (`[plan] > `).
- El agente activo responde con el catálogo derivado de sus permisos: `plan` solo lee y propone; `build` escribe con aprobación. Quién responde lo decide el indicador, no el texto escrito.

### 3. Sidebar de datos

Es la columna derecha, separada de la principal por dos columnas con el fondo de la
pantalla; su texto deja dos columnas de separación a cada lado y una fila de aire
arriba del título y otra debajo del pie (esta última solo si la columna tiene sitio).
Si la terminal es estrecha, el sidebar cede ancho antes que la columna principal, que
nunca baja de 20 columnas. **Arranca visible** y se pliega y despliega con `Ctrl+D`. De arriba
abajo muestra:

1. El **título de la conversación**: el nombre de la sesión activa con su estado entre paréntesis.
2. **CONTEXTO**: los tokens usados y el porcentaje ocupado.
3. **▾ TODO**: el elemento del TODO en curso y cuántos le quedan.
4. **LISTA DE TAREAS**: los pasos del agente cuando queda alguno accionable.
5. **ESTADO**: capa y cola, aprobaciones y agente, en filas etiqueta + valor.
6. El **pie**, pegado al fondo de la columna y siempre visible, con tres filas de arriba abajo:
   el estado de **git**, la **ruta** del proyecto entre corchetes y la **firma** del
   harness con su nombre y versión.

Los nueve datos siguen siendo estos:

| Dato         | Qué muestra                                                          |
| ------------ | -------------------------------------------------------------------- |
| Sesión       | Nombre de la sesión activa                                           |
| Contexto     | Tokens usados y porcentaje ocupado                                   |
| TODO         | Elemento del TODO en curso y cuántos le quedan                       |
| Ruta         | Carpeta actual del proyecto                                          |
| Git          | Rama activa y número de cambios sin confirmar (0 si el árbol está limpio) |
| Capa y cola  | Capa en la que se está trabajando y cuántas tareas grandes le quedan |
| Aprobaciones | Cuántas hay esperando tu decisión                                    |
| Agente       | `plan` o `build`                                                     |
| Proyecto     | Nombre y versión de LocalCli                                         |

Git y proyecto viven en el pie, no en ESTADO: son los dos datos que se miran de un
vistazo y basta con llegar al fondo de la columna para verlos. La firma del harness va
suelta, sin etiqueta, igual que la ruta va entre corchetes: el nombre y la versión se
dicen solos.

**Git se lee una vez al arrancar.** Una sesión trabaja siempre en la misma rama, así que
el pie refleja el estado del repositorio en el momento del arranque y no se vuelve a
consultar. Por eso un cambio que hagas tú en la terminal con la sesión abierta no sale en
el pie hasta la siguiente: es el precio de no estar llamando a git sin parar.

Si el proyecto **no tiene git inicializado** —o no hay git instalado—, no se deja un hueco
ni se disfraza de árbol limpio: la fila dice `sin iniciar`. Es un dato, no un «no sé».

Los datos de contexto se calculan y se interpretan según [[specs/SPEC-PANEL-CONTEXTO]].

La sección «LISTA DE TAREAS» muestra la lista de pasos del agente (`actualizar_todo`) con `[•]` en curso, `[✓]` hecho, `[x]` cancelado y `[ ]` pendiente. Se oculta cuando no hay nada accionable. Ver [[specs/SPEC-TOOLS]].

## Razonamiento del modelo

Mientras el modelo trabaja se muestra un **indicador en vivo** (`[⠋ Pensando]`, `[⠋ Usando herramienta: X]`) con su glifo girando y el tiempo transcurrido. El texto crudo del razonamiento **no se vuelca por defecto**: se **revela con `Ctrl+R`**, arriba de la respuesta, y entonces se distingue visualmente de ella.

- El indicador no depende de que el texto esté revelado: se ve siempre que el turno está en marcha.
- Revelar u ocultar el texto no detiene la generación: el razonamiento se sigue acumulando y volver a revelarlo lo recupera entero.
- El texto revelado nunca se mezcla visualmente con la respuesta.
- Si el modelo no expone razonamiento, al revelarlo se indica que no está disponible.

## Modales

Tres modales centrados comparten el mismo comportamiento: uno abierto a la vez, sus teclas capturan el teclado (`↑`/`↓` navegan, `Enter` aplica y cierra), `Esc` cierra sin cambios y `Ctrl+C` sigue saliendo de la aplicación.

| Modal        | Cómo se abre | Contenido                                                                                                                                                                                   | Al aplicar                                                                                                                       |
| ------------ | ------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------- |
| **Modelos**  | `Ctrl+X m`   | Lista de modelos locales de Ollama, cargada al abrir; el resaltado arranca en el modelo en uso; los que no declaran capacidad de herramientas se marcan; aviso «sin modelos» si no responde | El modelo elegido pasa a ser el de la sesión y se ve en la línea de modelo; si no puede usar herramientas, se avisa sin bloquear |
| **Sesiones** | `Ctrl+X l`   | Lista de las sesiones creadas anteriormente en el proyecto (nombre y estado, incluidas las de segundo plano)                                                                                | Se abre esa sesión: el chat pasa a su historial sin detener lo que corre                                                         |
| **Atajos**   | `Ctrl+P`     | Tabla de los atajos existentes agrupada por categorías (General, Chat, Vista, Modales, Aprobaciones, Entrada), con la tecla y su acción alineadas, incluidas las secuencias con líder       | No aplica nada: es solo lectura; `Esc` lo cierra                                                                                 |

`Ctrl+P` abre la lista de atajos, no una paleta de comandos ejecutables: los comandos de flujo se descubren escribiendo `/` en la entrada (§Zonas 2). Ver [[specs/SPEC-SESIONES]] para el ciclo de vida de las sesiones.

## Cambiar de sesión

No hay una lista de sesiones siempre visible. El chat es para la sesión activa.

- `Ctrl+X l` abre el **modal de sesiones** con las sesiones del proyecto: nombre y estado de cada una.
- Al elegir una con `Enter`, el chat cambia a esa sesión.
- Las sesiones en segundo plano se ven desde cualquier otra en el modal, con su estado.
- Dentro del modal, `Ctrl+D` elimina la sesión resaltada. Si esa sesión está trabajando, se pide confirmación («seguro que deseas eliminar») antes de borrarla; al confirmar, su contenido se elimina en cascada.
- `Ctrl+X n` crea una sesión nueva desde la vista principal y la deja activa.
- Al borrar la última sesión del proyecto, la vista vuelve a la bienvenida de inmediato.

El modal aparece y desaparece. No ocupa espacio permanente.

## El aviso que no se puede ocultar

Con el panel cerrado, si alguna sesión está esperando tu aprobación, aparece una línea discreta indicando cuántas hay.

Es la única información que se muestra fuera del panel, porque es la única cuya ausencia detiene el trabajo: un flujo en segundo plano se queda parado hasta que decidas, y si no lo ves, no avanzas. El detalle de cada aprobación está en [[specs/SPEC-INTERFAZ-ATAJOS]].

## Reglas de negocio

- El panel de datos es de lectura. Nada se escribe desde él.
- El panel se abre y se cierra sin interrumpir ninguna sesión.
- Abrir el panel no ralentiza la generación de una respuesta.
- Mientras el modelo trabaja se muestra un indicador en vivo (`[⠋ Pensando]`, `[⠋ Usando herramienta: X]`, `[⠋ Generando]`); el texto del razonamiento se revela con `Ctrl+R` y se puede volver a ocultar.
- El razonamiento revelado nunca se mezcla visualmente con la respuesta final.
- El conteo de tokens del turno se muestra bajo la entrada, atenuado, cuando hay consumo (`tokens: 54k`).
- Cada respuesta muestra el tiempo que tardó el modelo en entregarla. Mientras el turno está en curso, ese tiempo corre en pantalla y se detiene al cerrarse el turno. Ese tiempo **se guarda** con la respuesta, así que una sesión retomada lo vuelve a mostrar; una línea anterior a esa medición se pinta sin él.
- Cada línea de herramienta muestra cuánto tardó su ejecución, y ese tiempo también se guarda: al volver a la sesión, el hilo se pinta con los tiempos que tuvo.
- Los dos tiempos se miden con las mismas reglas —milisegundos por debajo del segundo, décimas de segundo por debajo del minuto, minutos y segundos a partir de ahí— y los dos son opcionales: sin medición, no se pinta nada en vez de un cero.
- No hay lista de sesiones permanente: se acceden con `Ctrl+X l`, que abre el modal de sesiones.
- El historial del chat se recorre con `↑`/`↓` y `pgup`/`pgdown`; mientras no se sube, la vista sigue el final. La línea de entrada nunca queda fuera de pantalla.
- Al abrir el modal de modelos, el resaltado arranca en el modelo en uso. Los modelos que no declaran capacidad de herramientas se marcan; elegirlos avisa sin bloquear y deja al usuario cambiar de modelo.
- La rueda del ratón desplaza el historial. Arrastrar con el botón izquierdo selecciona texto, que se **resalta en video inverso** mientras se elige, y al soltar se copia al portapapeles —el realce desaparece y aparece el aviso transitorio `[Copiado]` arriba a la derecha, que se apaga solo—; al capturar el ratón, la selección nativa de la terminal queda disponible con `Shift`. La selección se mantiene **anclada al texto**: la rueda puede usarse mientras se selecciona y no la cancela, de modo que en un chat largo se puede seguir eligiendo al desplazarse. Con el panel de aprobaciones abierto, un clic sobre «aprobar» o «declinar» de una línea resuelve esa aprobación.
- Con la sesión activa trabajando, el primer `esc` pide confirmación («presiona esc otra vez para cancelar razonamiento») y el segundo cancela el trabajo; cualquier otra tecla la descarta.
- Bajo la entrada se muestra el modelo en uso y si tiene acceso a herramientas y a la visión (interpretar imágenes).
- Si el texto de una petición de chat incluye la ruta de un archivo de imagen existente, la imagen se adjunta a ese turno hacia el modelo. Las imágenes son efímeras: no se guardan en el historial, así que un turno posterior que no las vuelva a mencionar no las ve.
- Al pegar o arrastrar un archivo, la línea de entrada muestra `[nombre.ext]` resaltado; una carpeta muestra `[CARPETA N elementos]` con sus entradas de primer nivel. Al enviar se usa la ruta real, así que una imagen se adjunta igual y el mensaje conserva la referencia al archivo.
- Pegar texto de varias líneas se resume en la entrada como `[PEGADO N líneas]`; al enviar se inserta el texto completo. Si todas las líneas son rutas existentes (varios archivos o carpetas arrastrados a la vez), se muestra un token por elemento en vez del resumen.
- Adjuntar una imagen a un modelo que no declara visión avisa en el chat sin bloquear el envío; el usuario decide si cambia de modelo.
- El marco de la vista nunca excede el alto de la terminal: el historial se recorta a lo disponible para que la entrada no quede fuera.
- El modal de atajos agrupa las acciones por categorías y alinea la tecla con su descripción.
- Hay exactamente tres modales (modelos, sesiones, atajos); solo uno puede estar abierto a la vez y `Esc` cierra cualquiera.
- El agente activo se muestra siempre en el pie de la caja de entrada (y a la izquierda de la línea en la bienvenida); `Tab` recorre los agentes disponibles, en bienvenida y en la vista principal. Con un modal abierto, `Tab` no cicla.
- Cambiar de sesión no detiene lo que está corriendo.
- El panel refleja los datos de la sesión activa, no de otra.
- Con el modal de sesiones abierto, `Ctrl+D` elimina la sesión resaltada; si está trabajando, se confirma antes de borrarla.
- `Ctrl+X n` crea una sesión nueva y la deja activa, sin detener las demás.
- Con el panel cerrado, el número de aprobaciones pendientes siempre se ve.
- Una aprobación pendiente se muestra sin robar el teclado: el panel aparece con sus opciones y el input sigue escribiendo. `Ctrl+A` le da el foco (entonces `a`/`d` deciden); sin foco se resuelve con un clic sobre «aprobar» o «declinar».
- El nombre y la versión de LocalCli se ven siempre que el panel esté abierto.
- La pantalla de bienvenida es la primera vista al ejecutar `localcli` y muestra logotipo, nombre con versión, la línea con el modelo en uso y una línea de entrada, todo centrado en la terminal. Los modelos solo se ven dentro del modal.
- Lo escrito en la bienvenida es la primera petición: crea una sesión nueva y la envía, y la vista cambia a la principal sin repetir ni confirmar.
- El nombre de la sesión es su título: el generado por el modelo a partir de su primera petición o, mientras no lo haya, el provisional «Nueva sesión».
- Al borrar la última sesión del proyecto, la vista vuelve a la bienvenida.
- Desde la bienvenida no hay panel de contexto ni aprobaciones: se abre el modal de modelos (`Ctrl+X m`) o el de sesiones (`Ctrl+X l`), se escribe, se envía y se sale.
- La vista principal es un chat: cada petición se responde como conversación. Ningún texto arranca un flujo de trabajo por sí solo.
- Un flujo solo arranca con un comando explícito escrito en la entrada: `/planificar`, `/crear`, `/actualizar`, `/eliminar`, `/resolver` o `/ejecutar`.
- La paleta de comandos de flujo se despliega al escribir `/` en la entrada —también en la bienvenida— y lista el catálogo que sirve el motor: los seis oficiales y los flujos propios de `.localcli/flows/*.json`. Es descubrimiento, no una vía nueva de arranque: no añade comandos que no existan.
- Con la paleta desplegada, `↑`/`↓` la recorren, `Tab` autocompleta el comando resaltado en vez de ciclar el agente y `Enter` ejecuta; escribir un espacio retira la paleta y deja paso a la petición.
- Si una petición parece trabajo ordenado, el sistema puede proponer un TODO, pero no lo ejecuta hasta que el usuario confirme o escriba el comando.
- El agente activo responde con las herramientas derivadas de sus permisos; el chat no añade ni quita herramientas.
- La bienvenida se pinta sin esperar a Ollama ni a la base: no depende de nada externo para mostrarse.

## Criterios de aceptación

- [ ] El sidebar de datos arranca visible y `Ctrl+D` lo pliega y lo despliega.
- [ ] Con el panel cerrado, el chat ocupa todo el ancho.
- [ ] El panel se abre y se cierra sin interrumpir el trabajo.
- [ ] El panel muestra los nueve datos definidos.
- [ ] El pie del panel son tres filas —rama y número de cambios de git, ruta y firma del harness— y se mantienen visibles aunque la lista de tareas no quepa.
- [ ] En un proyecto sin git inicializado la fila de git dice `sin iniciar`.
- [ ] El panel muestra la lista de pasos del agente cuando queda alguno por hacer, y la oculta cuando no hay nada accionable.
- [ ] El panel muestra los datos de la sesión activa, no los de otra.
- [ ] Mientras el modelo trabaja se ve el indicador en vivo (`[⠋ Pensando]`, y `[⠋ Usando herramienta: X]` mientras corre una herramienta).
- [ ] El razonamiento no se vuelca por defecto; `Ctrl+R` revela su texto, arriba de la respuesta y distinguible visualmente, y volver a pulsarlo lo oculta sin perderlo.
- [ ] El conteo de tokens del turno se ve bajo la entrada (`tokens: 54k`).
- [ ] Cada respuesta muestra el tiempo que tardó en llegar, y mientras se espera el tiempo corre en pantalla y se detiene al cerrarse el turno.
- [ ] Cada línea de herramienta muestra cuánto tardó su ejecución, y al recargar la sesión el hilo conserva ese tiempo igual que el de las respuestas.
- [ ] `Ctrl+X l` abre el modal de sesiones con nombre y estado de cada una; al seleccionar una con `Enter` se abre esa sesión y el chat muestra su historial.
- [ ] `Ctrl+P` abre el modal con la lista de atajos existentes (acción + tecla); es de solo lectura.
- [ ] El historial del chat se recorre con `↑`/`↓` y `pgup`/`pgdown`; mientras no se sube, la vista sigue el final.
- [ ] El texto del input se edita en cualquier punto con las flechas y `home`/`end`.
- [ ] Al escribir más de lo que cabe en el ancho, la línea salta de renglón en vez de recortarse; el reparto se reajusta al redimensionar la terminal.
- [ ] El historial del chat muestra lo escrito por el usuario y lo que responde el agente en globos de color distinto.
- [ ] El chat respeta el orden de ejecución: el texto previo a una herramienta queda arriba de su línea y el posterior abre un globo nuevo.
- [ ] Al abrir el modal de modelos, el resaltado está en el modelo en uso.
- [ ] Elegir un modelo sin capacidad de herramientas avisa sin bloquear.
- [ ] La rueda del ratón desplaza el historial y arrastrar con el ratón copia al portapapeles el texto seleccionado, que se resalta (video inverso) mientras se elige; la rueda no cancela esa selección (el resaltado se mantiene anclado al texto) y al soltar el resaltado desaparece con el aviso `[Copiado]` arriba a la derecha. La selección no mezcla las zonas: la que empieza en el chat se queda en el chat, y la que empieza en el sidebar se queda en el sidebar.
- [ ] Con la sesión trabajando, `esc` pide confirmación y un segundo `esc` cancela; otra tecla la descarta.
- [ ] Bajo la entrada se ve el modelo en uso y si tiene acceso a herramientas.
- [ ] Bajo la entrada se ve también si el modelo interpreta imágenes (visión).
- [ ] Escribir la ruta de una imagen existente en el mensaje la adjunta al turno; con un modelo que no declara visión avisa sin bloquear el envío.
- [ ] Pegar o arrastrar un archivo muestra `[nombre.ext]` resaltado y una carpeta `[CARPETA N elementos]`; el mensaje enviado usa su ruta real.
- [ ] Pegar un texto de varias líneas lo resume como `[PEGADO N líneas]` y al enviar llega el texto completo; si todas las líneas son rutas, se muestra un token por elemento.
- [ ] La línea de entrada de la bienvenida se edita en cualquier punto (flechas, `home`/`end`).
- [ ] El modal de atajos agrupa las acciones por categorías y alinea tecla y descripción.
- [ ] `Esc` cierra cualquier modal sin cambiar nada; con ninguno abierto no hace nada visible.
- [ ] `Tab` alterna el agente entre `plan` y `build`; el indicador del agente aparece en el pie de la caja de entrada en la vista principal (y a la izquierda de la línea en la bienvenida) y se actualiza al instante.
- [ ] El chat y el sidebar van separados por dos columnas con el fondo de la pantalla, y el chat y la caja de entrada por dos filas; la caja tiene fondo propio y dentro lleva el texto, una línea en blanco y la línea del agente y el modelo.
- [ ] Elegir una sesión cambia el chat a esa sesión sin detener lo demás.
- [ ] `Ctrl+D` en el modal de sesiones elimina la sesión resaltada; si está trabajando, pide confirmación antes.
- [ ] `Ctrl+X n` crea una sesión nueva y la deja activa.
- [ ] Con el panel cerrado se ve cuántas aprobaciones hay pendientes.
- [ ] Una aprobación pendiente se ve sin bloquear el input; se resuelve con `Ctrl+A` + `a`/`d` o con un clic sobre «aprobar»/«declinar».
- [ ] El nombre y la versión de LocalCli aparecen en el panel.
- [ ] Cambiar de sesión no detiene ninguna ejecución.
- [ ] Al ejecutar `localcli` se ve la pantalla de bienvenida con el logotipo sobre el fondo, la caja de entrada (placeholder `"Escribe para iniciar la conversacion"` y línea `[plan] • modelo`), la barra de pistas de teclado y la versión en la esquina inferior derecha, sin lista de modelos visible.
- [ ] El conjunto de la bienvenida (logotipo, caja de entrada y barra de pistas) aparece centrado en horizontal y en vertical dentro de la terminal, sin recortar el arte; el entorno del logotipo es el fondo de la pantalla.
- [ ] La barra de pistas se genera desde el mapa de teclas vigente (`TAB agentes`, `Ctrl+P comandos`, `Ctrl+X M modelos`).
- [ ] `Ctrl+X m` abre el modal de modelos con la lista de Ollama; `↑`/`↓` navegan, `Enter` aplica el resaltado y `Esc` cierra sin cambios. El modelo aplicado se usa para la primera petición y aparece en la línea de estado de la caja.
- [ ] Sin Ollama disponible, la bienvenida se muestra igual (usa el modelo por defecto del arranque); al abrir el modal aparece el aviso «sin modelos» y se puede escribir y enviar sin él.
- [ ] La primera petición escrita en la bienvenida aparece como primer mensaje del chat al cambiar de vista.
- [ ] La primera petición desde la bienvenida crea una sesión nueva y su nombre pasa a ser un título generado por el modelo.
- [ ] Al borrar la última sesión, la vista vuelve a la bienvenida.
- [ ] La transición de bienvenida a interfaz principal no repite la petición ni pide confirmación.
- [ ] Desde la bienvenida no hay panel de contexto ni aprobaciones; se abre el modal de modelos o el de sesiones, se escribe, se envía y se sale.
- [ ] `Ctrl+X l` en la bienvenida abre el modal de sesiones; al elegir una con `Enter` la vista pasa a la principal con el historial de esa sesión.
- [ ] La vista principal responde como chat; ningún texto arranca un flujo de trabajo sin un comando explícito.
- [ ] `/planificar`, `/crear`, `/actualizar`, `/eliminar`, `/resolver` y `/ejecutar` arrancan su flujo.
- [ ] Escribir `/` despliega encima de la entrada la lista de comandos de flujo; escribir filtra, `↑`/`↓` la recorren y `Tab` autocompleta el comando resaltado.
- [ ] `Enter` con la paleta desplegada ejecuta el comando resaltado y el flujo se arranca por el motor; su eco aparece en el chat.
- [ ] La paleta también se despliega en la bienvenida; ejecutar un comando desde ahí crea la sesión y lleva a la vista principal.
- [ ] Una petición sin comando se responde en el chat con las herramientas del agente activo.
- [ ] La bienvenida se muestra aunque Ollama no esté disponible.
- [ ] Con la líder pulsada (`Ctrl+X`) pero sin segunda tecla, no se abre ningún modal y tras el timeout el indicador de líder desaparece.
- [ ] El logotipo coincide byte a byte con la salida dorada de `internal/tui/testdata/logo.txt`.

## Requisitos no funcionales

- Mostrar el panel y el razonamiento no debe retrasar la respuesta del modelo.
- Las sesiones en segundo plano se siguen avanzando con el panel cerrado.
- La bienvenida se pinta al instante: no espera a conexiones externas.

## Dependencias funcionales

- [[specs/SPEC-INTERFAZ-ATAJOS]]
- [[specs/SPEC-KEYBINDS]]
- [[specs/SPEC-PANEL-CONTEXTO]]
- [[specs/SPEC-SESIONES]]
- [[specs/SPEC-COLA-TAREAS]]
- [[specs/SPEC-TOOLS]]

## Supuestos

- La librería de interfaz de terminal y la biblioteca de TUI concreta se eligen en FASE 2.
- El ancho mínimo de terminal y el comportamiento en terminales estrechas se fijan en FASE 2 y FASE 3.

## Referencias

- [[IDEA]]
