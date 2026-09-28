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

El chat ocupa la pantalla completa. El panel de datos está plegado a la derecha y se abre y se cierra.

```
┌──────────────────────────────────┬──────────────┐
│                                  │  PANEL       │
│  CHAT                            │  DE DATOS    │
│                                  │              │
│   razonamiento del modelo        │  sesión      │
│   respuesta                      │  contexto    │
│   ...                            │  TODO        │
│                                  │  ruta        │
│                                  │  git         │
│                                  │  capa y cola │
│                                  │  aprobaciones│
│                                  │  agente      │
│                                  │  proyecto    │
├──────────────────────────────────┤  LocalCli    │
│  entrada de texto                │              │
└──────────────────────────────────┴──────────────┘

Con el panel cerrado, el chat ocupa todo el ancho.
```

El panel es de lectura. No se escribe nada desde ahí.

## Pantalla de bienvenida

Es la primera vista al ejecutar `localcli`, antes de que exista conversación: un logotipo en ASCII con el nombre de la herramienta y una única línea de entrada, como en opencode.

```
██╗      ██████╗  ██████╗  █████╗  ██╗ ██████╗ ██╗   ██╗
██║     ██╔═══██╗██╔════╝ ██╔══██╗ ██║██╔════╝ ██║   ██║
██║     ██║   ██║██║      ███████║ ██║██║      ██║   ██║
██║     ██║   ██║██║      ██╔══██╗ ██║██║      ██║   ██║
███████╗╚██████╔╝╚██████╗ ██║  ██║ ██║╚██████╗ ████║ ██║
╚══════╝ ╚═════╝  ╚═════╝ ╚═╝  ╚═╝ ╚═╝ ╚═════╝ ╚═══╝ ╚═╝

                   LocalCli · v0.1

             modelo: llama3.2  (Ctrl+X m cambiar)

             [plan] > En qué te ayudo hoy: █╚
```

A la izquierda del input figura el agente activo (`plan` o `build`); `Tab` lo alterna.

### Arte canónico del logotipo

El logotipo es fijo y esta es su definición, que sirve de salida dorada para las pruebas:

- Texto `LocalCli` en bloques de 6 filas × 53 columnas, con los caracteres de bloque `█ ╗ ╝ ║ ╔ ═ ╚` en UTF-8.
- La copia canónica byte a byte vive en `internal/tui/testdata/logo.txt`: seis líneas de exactamente 53 columnas, sin espacios finales. El bloque de arriba es su render fiel; la comparación de la prueba va contra el archivo.
- Debajo del arte van el nombre con versión, la línea con el modelo elegido y la línea de entrada, compuestos alrededor pero fuera del arte: no forman parte de la salida dorada.
- El arte no se reescala: las 6 filas × 53 columnas se pintan íntegras, sin recorte. Lo que sí hace la vista es centrar el bloque completo (logotipo + nombre con versión + línea de modelo + línea de entrada) dentro de la terminal, en horizontal y en vertical, según el tamaño reportado por la ventana. Si la terminal es más pequeña que el bloque, se alinea arriba a la izquierda sin recortar el arte.
- No entra al contexto del modelo ni a ningún historial: es arte de la aplicación.

- Solo hay logotipo, nombre con su versión, una línea que muestra el modelo en uso y una línea de entrada. Los modelos NO se listan en la bienvenida: se consultan y eligen desde un modal. Sin paneles ni datos.
- **Línea de modelo**: bajo el nombre con versión se muestra solo el modelo en uso (el detectado en el arranque o el último elegido en el modal) y el recordatorio del atajo para cambiarlo (`Ctrl+X m`). No hay lista de modelos visible en la bienvenida.
- **Modal de modelos**: la secuencia líder `Ctrl+X` seguida de `m` abre un modal centrado con la lista de modelos locales que reporta Ollama, con uno resaltado. `↑`/`↓` mueven la selección, `Enter` aplica el elegido y cierra el modal, `Esc` cierra sin cambiar nada. La lista se pide a Ollama al abrir el modal (no en el arranque); si Ollama no responde, el modal muestra el aviso «sin modelos» y se puede cerrar con `Esc` sin bloquear nada. Mientras el modal está abierto, las teclas son solo del modal: la escritura de la bienvenida no las recibe. Lo elegido pasa al motor como modelo de la sesión y se refleja en la línea de modelo de la bienvenida. La elección del usuario prevalece sobre la autodetección del arranque ([[specs/SPEC-OLLAMA-PERFIL]]: el modelo lo elige el usuario). El mecanismo de la secuencia está en [[specs/SPEC-KEYBINDS]].
- Lo que se escribe es la **primera petición de la sesión**: se envía tal cual, igual que se enviaría desde el chat.
- La línea de entrada de la bienvenida despliega también la **paleta de comandos de flujo** al escribir `/` (§Zonas 2): escribir filtra, `↑`/`↓` la recorren, `Tab` autocompleta y `Enter` ejecuta. Un comando crea la sesión y lleva a la vista principal igual que una primera petición.
- Al enviarla, la vista cambia a la interfaz principal y la petición aparece como primer mensaje del chat. La transición no repite la petición ni pide confirmación.
- No hay una sesión activa antes de usar la bienvenida: la primera petición **crea** una sesión nueva, con un nombre provisional que el modelo sustituye por un título generado a partir de esa misma petición. Ver [[specs/SPEC-SESIONES]].
- Desde la bienvenida no hay panel de contexto ni de aprobaciones: hay logotipo, línea de modelo y línea de entrada; con `Ctrl+X m` se abre el modal de modelos y con `Ctrl+X l` el de sesiones (al elegir una con `Enter`, la vista pasa a la principal con el historial de esa sesión). Se escribe y se envía —o se ejecuta un comando de flujo— y la salida es `Ctrl+C`.

## Zonas

### 1. Chat

- Ocupa el resto del ancho.
- Muestra el historial de la sesión activa.
- Cada mensaje se pinta en un globo con el color de quien habla: uno para lo que escribe el usuario y otro para lo que responde el agente (el «terminal»). Las líneas del sistema —herramientas, avisos— no son un turno: se pintan sueltas, sin globo.
- El hilo se pinta en el **orden en que ocurrió**: el texto que el modelo escribió antes de usar una herramienta queda arriba de su línea, y el que escribe después abre un globo nuevo. Los segmentos de texto de un mismo turno no se funden en uno solo por debajo de las líneas de herramienta.
- Cada intercambio muestra la respuesta del modelo; mientras genera, un **indicador en vivo** (`[⠋ Pensando]`) sustituye al volcado del razonamiento. El texto del razonamiento se revela con `Ctrl+R` (ver §Razonamiento del modelo).
- Cada respuesta del modelo muestra cuánto tardó en llegar, atenuado junto a ella. Mientras se espera, el tiempo corre en pantalla —junto al indicador en vivo— para saber que el modelo sigue trabajando.
- Muestra las propuestas pendientes de aprobación.
- El historial se puede recorrer: `↑`/`↓` suben y bajan línea a línea y `pgup`/`pgdown` por páginas; la rueda del ratón también desplaza. Mientras no se sube, la vista sigue el final y baja sola con cada respuesta nueva; al subir se respeta la posición. Cuando queda historial fuera de la ventana se indica con una línea discreta («↑ N líneas arriba» / «↓ N líneas abajo»).

### 2. Entrada de texto

- Una línea de escritura que crece en filas: el texto es un solo párrafo, pero al desbordar el ancho disponible salta de renglón (hasta un tope de filas; a partir de ahí se desplaza dentro de la ventana) en vez de recortarse. Al redimensionar la terminal, el reparto se reajusta. `Enter` envía la petición; no inserta saltos de línea.
- Escribe hacia la sesión activa.
- **Paleta de comandos de flujo.** Al escribir `/` se despliega encima de la línea la lista de comandos de flujo disponibles —los oficiales más los propios de `ai/flows/*.json`—, con el resaltado en el primero. Lo escrito filtra la lista; `↑`/`↓` la recorren, `Tab` autocompleta el comando resaltado dejando la línea lista para escribir la petición detrás (`/comando [petición]`) y `Enter` ejecuta. En cuanto se escribe un espacio (empieza la petición) la paleta se retira. Es un ayudante para descubrir los comandos, no una vía nueva de arranque: el flujo sigue arrancando solo con su comando explícito y el catálogo lo sirve el motor.
- El texto se edita en cualquier punto: las flechas mueven el cursor, `home`/`end` van al principio y al final y `ctrl+b`/`ctrl+e` son sus equivalentes. Los atajos que coincidan con una acción (por ejemplo `ctrl+a`) siguen resolviéndose como acción y no editan.
- Bajo la línea de entrada se muestra el modelo en uso y si tiene acceso a herramientas (`modelo: X · herramientas: sí/no/?`) y, cuando hay consumo que mostrar, el conteo de tokens del turno (`tokens: 54k`). La línea de entrada de la bienvenida se edita igual que esta.
- Sobre la línea de entrada, mientras el modelo trabaja, se pinta la **línea de actividad**: un glifo que gira y la etiqueta de lo que pasa —`[⠋ Pensando]` si aún no hay respuesta, `[⠋ Usando herramienta: X]` si corre una herramienta, `[⠋ Generando]` si ya llega respuesta— más el tiempo transcurrido. Sustituye al volcado crudo del razonamiento.
- **Indicador de agente a la izquierda del input**: justo al lado izquierdo de la línea de entrada se muestra el agente activo (p. ej. `[plan] > `). Cambia al instante con `Tab`, que recorre los agentes disponibles (los base `plan` y `build` más los que el usuario añada en `ai/agents/*.json`). El indicador es visible tanto en la interfaz principal como en la bienvenida.
- El agente activo responde con el catálogo derivado de sus permisos: `plan` solo lee y propone; `build` escribe con aprobación. Quién responde lo decide el indicador, no el texto escrito.

### 3. Panel de datos

| Dato | Qué muestra |
|---|---|
| Sesión | Nombre de la sesión activa |
| Contexto | Tokens usados y porcentaje ocupado |
| TODO | Elemento del TODO en curso y cuántos le quedan |
| Ruta | Carpeta actual del proyecto |
| Git | Rama activa y si hay cambios sin confirmar |
| Capa y cola | Capa en la que se está trabajando y cuántas tareas grandes le quedan |
| Aprobaciones | Cuántas hay esperando tu decisión |
| Agente | `plan` o `build` |
| Proyecto | Nombre y versión de LocalCli |

Los datos de contexto se calculan y se interpretan según [[specs/SPEC-PANEL-CONTEXTO]].

Debajo de los nueve datos, cuando el agente mantiene una lista de pasos (`actualizar_todo`) y le queda alguno por hacer, el panel muestra la sección «TODO DEL AGENTE» con `[•]` en curso, `[✓]` hecho, `[x]` cancelado y `[ ]` pendiente. Se oculta cuando no hay nada accionable. Ver [[specs/SPEC-TOOLS]].

## Razonamiento del modelo

Mientras el modelo trabaja se muestra un **indicador en vivo** (`[⠋ Pensando]`, `[⠋ Usando herramienta: X]`) con su glifo girando y el tiempo transcurrido. El texto crudo del razonamiento **no se vuelca por defecto**: se **revela con `Ctrl+R`**, arriba de la respuesta, y entonces se distingue visualmente de ella.

- El indicador no depende de que el texto esté revelado: se ve siempre que el turno está en marcha.
- Revelar u ocultar el texto no detiene la generación: el razonamiento se sigue acumulando y volver a revelarlo lo recupera entero.
- El texto revelado nunca se mezcla visualmente con la respuesta.
- Si el modelo no expone razonamiento, al revelarlo se indica que no está disponible.

## Modales

Tres modales centrados comparten el mismo comportamiento: uno abierto a la vez, sus teclas capturan el teclado (`↑`/`↓` navegan, `Enter` aplica y cierra), `Esc` cierra sin cambios y `Ctrl+C` sigue saliendo de la aplicación.

| Modal | Cómo se abre | Contenido | Al aplicar |
|---|---|---|---|
| **Modelos** | `Ctrl+X m` | Lista de modelos locales de Ollama, cargada al abrir; el resaltado arranca en el modelo en uso; los que no declaran capacidad de herramientas se marcan; aviso «sin modelos» si no responde | El modelo elegido pasa a ser el de la sesión y se ve en la línea de modelo; si no puede usar herramientas, se avisa sin bloquear |
| **Sesiones** | `Ctrl+X l` | Lista de las sesiones creadas anteriormente en el proyecto (nombre y estado, incluidas las de segundo plano) | Se abre esa sesión: el chat pasa a su historial sin detener lo que corre |
| **Atajos** | `Ctrl+P` | Tabla de los atajos existentes agrupada por categorías (General, Chat, Vista, Modales, Aprobaciones, Entrada), con la tecla y su acción alineadas, incluidas las secuencias con líder | No aplica nada: es solo lectura; `Esc` lo cierra |

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
- Cada respuesta muestra el tiempo que tardó el modelo en entregarla. Mientras el turno está en curso, ese tiempo corre en pantalla y se detiene al cerrarse el turno; el tiempo no se guarda en el historial, así que una sesión retomada no lo muestra.
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
- El agente activo se muestra siempre a la izquierda del input; `Tab` recorre los agentes disponibles, en bienvenida y en la vista principal. Con un modal abierto, `Tab` no cicla.
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
- La paleta de comandos de flujo se despliega al escribir `/` en la entrada —también en la bienvenida— y lista el catálogo que sirve el motor: los seis oficiales y los flujos propios de `ai/flows/*.json`. Es descubrimiento, no una vía nueva de arranque: no añade comandos que no existan.
- Con la paleta desplegada, `↑`/`↓` la recorren, `Tab` autocompleta el comando resaltado en vez de ciclar el agente y `Enter` ejecuta; escribir un espacio retira la paleta y deja paso a la petición.
- Si una petición parece trabajo ordenado, el sistema puede proponer un TODO, pero no lo ejecuta hasta que el usuario confirme o escriba el comando.
- El agente activo responde con las herramientas derivadas de sus permisos; el chat no añade ni quita herramientas.
- La bienvenida se pinta sin esperar a Ollama ni a la base: no depende de nada externo para mostrarse.

## Criterios de aceptación

- [ ] Con el panel cerrado, el chat ocupa todo el ancho.
- [ ] El panel se abre y se cierra sin interrumpir el trabajo.
- [ ] El panel muestra los nueve datos definidos.
- [ ] El panel muestra la lista de pasos del agente cuando queda alguno por hacer, y la oculta cuando no hay nada accionable.
- [ ] El panel muestra los datos de la sesión activa, no los de otra.
- [ ] Mientras el modelo trabaja se ve el indicador en vivo (`[⠋ Pensando]`, y `[⠋ Usando herramienta: X]` mientras corre una herramienta).
- [ ] El razonamiento no se vuelca por defecto; `Ctrl+R` revela su texto, arriba de la respuesta y distinguible visualmente, y volver a pulsarlo lo oculta sin perderlo.
- [ ] El conteo de tokens del turno se ve bajo la entrada (`tokens: 54k`).
- [ ] Cada respuesta muestra el tiempo que tardó en llegar, y mientras se espera el tiempo corre en pantalla y se detiene al cerrarse el turno.
- [ ] `Ctrl+X l` abre el modal de sesiones con nombre y estado de cada una; al seleccionar una con `Enter` se abre esa sesión y el chat muestra su historial.
- [ ] `Ctrl+P` abre el modal con la lista de atajos existentes (acción + tecla); es de solo lectura.
- [ ] El historial del chat se recorre con `↑`/`↓` y `pgup`/`pgdown`; mientras no se sube, la vista sigue el final.
- [ ] El texto del input se edita en cualquier punto con las flechas y `home`/`end`.
- [ ] Al escribir más de lo que cabe en el ancho, la línea salta de renglón en vez de recortarse; el reparto se reajusta al redimensionar la terminal.
- [ ] El historial del chat muestra lo escrito por el usuario y lo que responde el agente en globos de color distinto.
- [ ] El chat respeta el orden de ejecución: el texto previo a una herramienta queda arriba de su línea y el posterior abre un globo nuevo.
- [ ] Al abrir el modal de modelos, el resaltado está en el modelo en uso.
- [ ] Elegir un modelo sin capacidad de herramientas avisa sin bloquear.
- [ ] La rueda del ratón desplaza el historial y arrastrar con el ratón copia al portapapeles el texto seleccionado, que se resalta (video inverso) mientras se elige; la rueda no cancela esa selección (el resaltado se mantiene anclado al texto) y al soltar el resaltado desaparece con el aviso `[Copiado]` arriba a la derecha.
- [ ] Con la sesión trabajando, `esc` pide confirmación y un segundo `esc` cancela; otra tecla la descarta.
- [ ] Bajo la entrada se ve el modelo en uso y si tiene acceso a herramientas.
- [ ] Bajo la entrada se ve también si el modelo interpreta imágenes (visión).
- [ ] Escribir la ruta de una imagen existente en el mensaje la adjunta al turno; con un modelo que no declara visión avisa sin bloquear el envío.
- [ ] Pegar o arrastrar un archivo muestra `[nombre.ext]` resaltado y una carpeta `[CARPETA N elementos]`; el mensaje enviado usa su ruta real.
- [ ] Pegar un texto de varias líneas lo resume como `[PEGADO N líneas]` y al enviar llega el texto completo; si todas las líneas son rutas, se muestra un token por elemento.
- [ ] La línea de entrada de la bienvenida se edita en cualquier punto (flechas, `home`/`end`).
- [ ] El modal de atajos agrupa las acciones por categorías y alinea tecla y descripción.
- [ ] `Esc` cierra cualquier modal sin cambiar nada; con ninguno abierto no hace nada visible.
- [ ] `Tab` alterna el agente entre `plan` y `build`; el indicador del agente aparece a la izquierda del input tanto en la bienvenida como en la vista principal y se actualiza al instante.
- [ ] Elegir una sesión cambia el chat a esa sesión sin detener lo demás.
- [ ] `Ctrl+D` en el modal de sesiones elimina la sesión resaltada; si está trabajando, pide confirmación antes.
- [ ] `Ctrl+X n` crea una sesión nueva y la deja activa.
- [ ] Con el panel cerrado se ve cuántas aprobaciones hay pendientes.
- [ ] Una aprobación pendiente se ve sin bloquear el input; se resuelve con `Ctrl+A` + `a`/`d` o con un clic sobre «aprobar»/«declinar».
- [ ] El nombre y la versión de LocalCli aparecen en el panel.
- [ ] Cambiar de sesión no detiene ninguna ejecución.
- [ ] Al ejecutar `localcli` se ve la pantalla de bienvenida con logotipo, nombre, la línea del modelo en uso y una línea de entrada, sin lista de modelos visible.
- [ ] El bloque de la bienvenida (logotipo, nombre, línea de modelo y entrada) aparece centrado en horizontal y en vertical dentro de la terminal, sin recortar el arte.
- [ ] `Ctrl+X m` abre el modal de modelos con la lista de Ollama; `↑`/`↓` navegan, `Enter` aplica el resaltado y `Esc` cierra sin cambios. El modelo aplicado se usa para la primera petición y aparece en la línea de modelo.
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
