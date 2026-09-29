---
title: LocalCli — componentes de la TUI
tags: [frontend, dominio]
depende_de:
  - "[[frontend/FRONTEND]]"
  - "[[backend/01-domain/DOMAIN]]"
relacionado:
  - "[[backend/04-infrastructure/EVENTS]]"
  - "[[database/01-schema/ENUMS]]"
  - "[[specs/SPEC-INTERFAZ]]"
  - "[[specs/SPEC-PANEL-CONTEXTO]]"
---
# DOMAIN — Componentes de la TUI

Cada componente tiene una responsabilidad y un límite. Ninguno contiene reglas de negocio: si algo hay que decidir, la decisión pertenece al motor y el componente solo la pinta o la pregunta al usuario.

## 1. Componentes

| Componente | Responsabilidad | Lo que no hace |
|---|---|---|
| `app` | Modelo raíz: reparte los eventos entre componentes, mantiene qué vista está activa (bienvenida o principal) y decide la transición | No conoce el detalle de cada componente; solo enruta |
| `welcome` | Pantalla de bienvenida centrada: logotipo ASCII con nombre y versión, línea con el modelo en uso y una línea de entrada precedida del indicador de agente (`[plan]` / `[build]`). Su línea de entrada se edita en cualquier punto y despliega la paleta de comandos al escribir `/`, como la principal. Los modelos se listan solo dentro del modal (`Ctrl+X m`) y las sesiones en el suyo (`Ctrl+X l`) | No valida nada: envía lo escrito como primera petición (que crea una sesión nueva, resuelto por `app` con el puerto) junto con el modelo elegido, y la vista cambia a la principal |
| `chat` | Muestra el historial de la sesión activa. Cada mensaje va en un globo con el color de quien habla —uno para lo que escribe el usuario, otro para lo que responde el agente— y con un icono al lado: delante el agente, detrás el usuario (pegado a la derecha); las líneas del sistema (herramientas, avisos) se pintan sueltas. Muestra la respuesta de cada intercambio y, mientras el turno corre, un indicador en vivo (`[⠋ Pensando]`, `[⠋ Usando herramienta: X]`); el texto del razonamiento se revela con `Ctrl+R`. Las líneas de herramienta son compactas: nacen al invocar con el verbo y el tema (`LEER [ruta]`) y se completan al terminar (`✓ LEER [ruta] · 70 líneas`). El historial se recorre con `↑`/`↓`, `pgup`/`pgdown` y la rueda del ratón; sin subir, sigue el final | No gestiona sesiones; pinta lo que le llega de la activa |
| `input` | Una caja con borde que contiene una línea de texto —un solo párrafo que crece en filas al desbordar el ancho, sin recortar lo escrito— y, debajo, un pie con el agente activo (`[plan]` / `[build]`), el modelo en uso y sus capacidades; compone y envía la petición hacia la sesión activa. El texto se edita en cualquier punto (flechas, `home`/`end`). Al escribir `/` despliega encima la paleta de comandos. `Tab` alterna el agente. Un pegado o arrastre muestra cada archivo como token `[nombre.ext]`, cada carpeta como `[CARPETA N elementos]` y un texto de varias líneas como `[PEGADO N líneas]`; al enviar se expanden al valor real. | No valida reglas de negocio |
| `paleta` | Lista de comandos de flujo que se despliega encima del input al escribir `/`; filtra por lo escrito, `↑`/`↓` la recorren, `Tab` autocompleta el resaltado y `Enter` ejecuta. El catálogo lo ofrece el puerto: los flujos oficiales y los de `ai/flows/*.json` | No decide qué comandos existen ni arranca nada por su cuenta: entrega el comando resaltado al `app` |
| `panel` | El sidebar de datos, visible por defecto y plegable con `Ctrl+D`: el título de la conversación (sesión y estado), CONTEXTO (tokens y porcentaje), TODO, LISTA DE TAREAS, ESTADO (capa y cola, aprobaciones y agente) y un pie de tres filas con el estado de git, la ruta del proyecto entre corchetes y la firma del harness con su nombre y versión. El pie se lee al arrancar y no se vuelve a consultar, así que un cambio hecho en la terminal con la sesión abierta no sale hasta la siguiente; sin git inicializado la fila dice `sin iniciar` | Es de lectura: nada se escribe desde él |
| `modals` | Tres modales centrados con la misma mecánica (uno abierto a la vez, `↑`/`↓` navegan, `Enter` aplica y cierra, `Esc` descarta): `modelsmodal` (modelos de Ollama, `Ctrl+X m`; resalta el modelo en uso y marca los que no declaran herramientas), `sessionsmodal` (sesiones del proyecto con nombre y estado; al aplicar abre esa sesión, `Ctrl+X l`; `Ctrl+D` elimina la resaltada, con confirmación si está trabajando; refleja el título generado por `titulo_sesion`) y `keysmodal` (lista de atajos existentes agrupada por categorías, solo lectura, `Ctrl+P`) | No decide nada: entrega la elección al `app`; no mantiene listas permanentes |
| `approvals` | Panel de aprobaciones pendientes de todas las sesiones, resolvibles una a una. Se muestra solo (para que la decisión se vea) pero no roba el teclado: `Ctrl+A` le da el foco y entonces `a`/`d` deciden; sin foco se decide con un clic sobre «aprobar»/«declinar» | No decide: envía la decisión del usuario |
| `notify` | Línea discreta con el número de aprobaciones pendientes, visible con el panel cerrado | Es el único dato que se muestra fuera del panel |
| `keys` | Mapa de teclas con sus valores por defecto y su reasignación | Un atajo no cambia ninguna regla de permiso |

## 2. Reglas de presentación

- En el chat, lo escrito por el usuario y lo que responde el agente se pintan en globos con color propio (azul el usuario, verde el agente) y con un icono al lado —delante el agente, detrás el usuario (pegado a la derecha)—; las líneas del sistema —herramientas, avisos— van sueltas, sin globo.
- El chat y el sidebar se separan por una línea vertical, y el chat y la caja de entrada por una línea horizontal. Los globos dejan un margen a cada lado, la caja de la entrada queda siempre pegada al pie (aunque el historial sea corto) y su pie lleva el agente, el modelo en uso y sus capacidades; el conteo de tokens del turno va justo debajo de la caja.
- El chat respeta el orden de ejecución: el texto previo a una herramienta queda en su globo arriba de su línea, y el texto posterior abre un globo nuevo. Los segmentos de un turno no se funden en uno solo por debajo de las líneas de herramienta; cada globo conserva el razonamiento de su segmento.
- La línea de entrada es un solo párrafo que salta de renglón al desbordar el ancho, hasta un tope de filas; al pasar del tope se desplaza, y al redimensionar la terminal el reparto se reajusta. `Enter` envía; no inserta saltos de línea.
- Al pegar o arrastrar, cada archivo se muestra como `[nombre.ext]` resaltado, cada carpeta como `[CARPETA N elementos]` y un texto de varias líneas como `[PEGADO N líneas]`; si todas las líneas son rutas, va un token por elemento. Al enviar se usa el valor real (la ruta o el texto), no el token.
- Mientras el modelo trabaja se pinta un indicador en vivo (`[⠋ Pensando]`, `[⠋ Usando herramienta: X]`, `[⠋ Generando]`) con el tiempo transcurrido. El texto crudo del razonamiento no se vuelca: se revela con `Ctrl+R`, arriba de la respuesta y distinguible visualmente, sin detener la generación.
- El conteo de tokens del turno se muestra bajo la línea de entrada (`tokens: 54k`) cuando hay consumo, además de su fila en el panel.
- Cada respuesta del modelo muestra cuánto tardó en llegar; mientras el turno está en curso, ese tiempo corre en pantalla y se detiene al cerrarse. Es el tiempo que percibe quien espera, del envío al cierre, y no se persiste: un historial recargado no lo trae.
- El panel refleja siempre los datos de la sesión activa, no los de otra.
- Con el panel cerrado, el contador de aprobaciones pendientes sigue visible y se actualiza con cada evento.
- Los estados que se pintan son los de [[database/01-schema/ENUMS]]; la TUI no inventa estados ni transiciones.
- La estimación de tokens se marca como estimación cuando lo es. Ver [[specs/SPEC-PANEL-CONTEXTO]].
- `Ctrl+X n` crea una sesión nueva desde la vista principal y la deja activa, sin detener las demás. Nace con el nombre provisional «Nueva sesión»; el modelo lo sustituye por un título con su primera petición (`titulo_sesion`).
- El nombre de la sesión que se pinta (panel y modal) es su título; el identificador no cambia al renombrar.
- Al borrar la última sesión del proyecto, la vista vuelve a la bienvenida de inmediato.
- `Ctrl+D` con el modal de sesiones abierto elimina la sesión resaltada; si está trabajando, se pide confirmación antes de borrarla.
- Una aprobación pendiente se muestra sin bloquear la escritura: el panel aparece con sus opciones y el input sigue escribiendo. `Ctrl+A` le da el foco (entonces `a`/`d` deciden) y, sin foco, un clic sobre «aprobar»/«declinar» decide. Sin pendientes, el panel se cierra solo.
- Cambiar de sesión no interrumpe ninguna ejecución: solo cambia lo que se pinta.
- El historial del chat se puede recorrer (`↑`/`↓`, `pgup`/`pgdown`) sin que la línea de entrada quede fuera de pantalla; mientras no se sube, la vista sigue el final y baja sola con cada respuesta nueva.
- El último modelo y el último agente usados se recuerdan entre ejecuciones (`~/.config/localcli/config.json`). Al elegir un modelo que no declara capacidad de herramientas, la vista avisa sin bloquear y deja al usuario cambiar de modelo.
- La rueda del ratón recorre el historial. Arrastrar con el botón izquierdo selecciona texto —que se resalta en video inverso mientras se elige— y, al soltar, se copia al portapapeles (herramientas del sistema u OSC 52), el realce desaparece y se muestra el aviso transitorio `[Copiado]` arriba a la derecha; con el ratón capturado, la selección nativa queda disponible con `Shift`. La selección se mantiene anclada al texto al desplazar, así que la rueda no la cancela. Con el panel de aprobaciones abierto, un clic sobre «aprobar» o «declinar» de una línea resuelve esa aprobación.
- Con la sesión activa trabajando, el primer `esc` pide confirmación («presiona esc otra vez para cancelar razonamiento») y el segundo cancela el trabajo.
- La caja de la entrada lleva en su pie el agente activo, el modelo en uso y sus capacidades; el conteo de tokens del turno se ve justo debajo de la caja.
- La paleta de comandos de flujo se despliega encima de la entrada al escribir `/`; el catálogo lo ofrece el puerto y no decide nada: entrega el comando resaltado al `app`, que lo arranca por el motor. Con la paleta abierta, `Tab` autocompleta en vez de ciclar el agente y `↑`/`↓` la recorren en vez de desplazar el historial.

## 3. Reglas de la bienvenida

- Es la primera vista al ejecutar `localcli`. Hay logotipo, nombre con versión, una línea que muestra el modelo en uso y una línea de entrada con el indicador del agente a su izquierda (`[plan] > …`); el bloque completo va centrado en la terminal. Sin paneles ni aprobaciones y sin lista de modelos visible.
- `Tab` recorre los agentes disponibles (los base `plan` y `build` y los propios de `.localcli/agents/*.json`) también en la bienvenida; el indicador junto al input se actualiza al instante.
- La línea de entrada se edita en cualquier punto, como la de la vista principal: flechas, `home`/`end` y `ctrl+b`/`ctrl+e` mueven el cursor.
- La línea de entrada despliega también la paleta de comandos al escribir `/`; ejecutar un comando crea la sesión y lleva a la vista principal, igual que una primera petición.
- `Ctrl+X m` abre un modal con los modelos locales que reporta Ollama (se piden al abrir, no en el arranque); `↑`/`↓` cambian el resaltado, `Enter` aplica y cierra, `Esc` cierra sin cambios. Lo aplicado viaja con la primera petición y se ve en la línea de modelo. Si Ollama no responde, el modal muestra «sin modelos». La bienvenida nunca espera a nada externo y sin modal abierto no hay navegación de modelos: las flechas escriben/historial según su componente.
- `Ctrl+X l` abre el modal de sesiones del proyecto; al elegir una con `Enter`, la vista pasa a la principal con el historial de esa sesión.
- Lo escrito es la primera petición: crea una sesión nueva (con nombre provisional; el título lo genera el modelo) y la vista cambia a la principal, donde aparece como primer mensaje del chat. La transición no repite la petición ni pide confirmación. La vista principal es un chat: para arrancar un flujo hay que escribir su comando explícito.
- Se pinta sin esperar a Ollama ni a la base: no depende de nada externo. La única salida desde ella es `Ctrl+C`.
- El logotipo es un arte ASCII fijo de la aplicación, no contenido de sesión: vive en el código de la TUI y no entra al contexto del modelo.

## 4. Lo que el frontend nunca hace

- No llama a Ollama, no abre la base para escribir, no toca archivos del proyecto.
- No comprueba permisos ni aprueba nada por su cuenta: pinta la petición y envía la decisión del usuario.
- No calcula el contexto ni interpreta el TODO: recibe los datos hechos por el motor.

## Referencias

- [[frontend/FRONTEND]] — mapa de la capa.
- [[specs/SPEC-INTERFAZ]] — qué dato vive en cada zona, incluida la bienvenida.
- [[backend/04-infrastructure/EVENTS]] — los eventos que consumen estos componentes.
