---
title: LocalCli — entradas y salidas de la TUI
tags: [frontend, interfaces]
depende_de:
    - "[[frontend/FRONTEND]]"
    - "[[backend/04-infrastructure/EVENTS]]"
    - "[[specs/SPEC-INTERFAZ]]"
relacionado:
    - "[[specs/SPEC-SESIONES]]"
    - "[[specs/SPEC-INTERFAZ-ATAJOS]]"
    - "[[specs/SPEC-TOOLS]]"
    - "[[specs/SPEC-OLLAMA-PERFIL]]"
    - "[[database/03-operations/QUERIES]]"
---

# INTERFACES — Entradas y salidas de la TUI

La TUI no tiene red ni API: su frontera son dos entradas (teclado y eventos) y dos salidas (peticiones a `session` y lecturas a `store`).

No importa `tools`. Recibe lo que llega por el bus, que es lo que la prohibición de módulos exige. Ver [[backend/04-infrastructure/EVENTS]].

## 1. Eventos que consume

De [[backend/04-infrastructure/EVENTS]] llega cada evento y así reacciona la pantalla:

| Evento                                      | Qué hace la TUI                                                                                                                                                                         |
| ------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `token`                                     | Añade el fragmento al bloque de razonamiento o a la respuesta, en vivo                                                                                                                  |
| `herramienta_invocada`                      | Abre la línea de herramienta en el chat con su verbo y su tema (la ruta, el patrón o el comando)                                                                                        |
| `herramienta_resultado`                     | Completa esa misma línea: la marca, la medida del resultado, si se recortó y cuánto tardó la ejecución                                                                                    |
| `estado_sesion`                             | Actualiza el estado en el selector y en el panel                                                                                                                                        |
| `titulo_sesion`                             | Renombra la sesión en el panel (si es la activa) y en su fila del modal de sesiones                                                                                                     |
| `notificacion`                              | Marca el aviso de esa sesión aunque no sea la activa                                                                                                                                    |
| `peticion_aprobacion`                       | Añade la línea al panel de aprobaciones y actualiza el contador                                                                                                                         |
| `aprobacion_resuelta`                       | Retira o marca la línea y actualiza el contador                                                                                                                                         |
| `etapa_iniciada`                            | Pinta en el chat la línea `[Sub Proceso] <nombre>` de la etapa que empieza                                                                                                              |
| `etapa_terminada` / `etapa_fallida`         | Sin línea propia: la entrega final del flujo la escribe su último paso; `fallida` deja aviso                                                                                            |
| `flujo_pausado` / `reanudado` / `cancelado` | Refleja el estado del flujo                                                                                                                                                             |
| `cola_actualizada` / `elemento_bloqueado`   | Actualiza la zona de capa y cola del panel                                                                                                                                              |
| `todo_actualizada`                          | Repinta la lista de pasos de la sesión activa en el panel                                                                                                                               |
| `cambio_aplicado`                           | Hoy no cambia nada en pantalla: el estado de git se leyó al arrancar y el panel no lo vuelve a consultar. El evento queda reservado para cuando la lectura deje de ser solo de arranque |
| `contexto_auditado`                         | Queda disponible para consulta; no se pinta por defecto                                                                                                                                 |

**La vista pinta una sola sesión.** De los eventos que llevan `sesion`, la TUI aplica los de la sesión activa y descarta los de las demás, que siguen trabajando en segundo plano y no deben mezclarse ([[backend/01-domain/DOMAIN]] §1). `notificacion` es la excepción: se ve aunque sea de otra. Al activar una sesión se refresca el CONTEXTO con su historial, se conserva el `Ctrl+R` y, si la sesión está trabajando, se reanuda su contador y su glifo ([Sub Proceso] y las líneas de herramienta de otro turno no entran).

## 1.1 La línea de herramienta

Es lo que hace visible el trabajo del agente, y sin ella no habría nada que ver.

Con el contrato en prosa, la llamada a una herramienta viajaba dentro del texto del modelo y se veía sola. Con el canal nativo **la llamada sale del texto**: si la TUI no la pinta, mientras el agente lee un archivo o corre las pruebas la pantalla se queda quieta y de golpe aparece una respuesta como si nada.

La línea es una sola: **nace al invocar y se completa al terminar**. Al invocar lleva el verbo y el tema; al llegar el resultado se le añade la marca, la medida, el tiempo que tardó y —si se recortó— el aviso:

```
  LEER [internal/tools/catalog.go]
  ✓ LEER [internal/tools/catalog.go] · 70 líneas · 0.4 s
  ✓ EJEC [go test ./...] · 42 líneas · recortado · 1.2 s
  ✗ CREAR [nuevo.txt] · el archivo ya existe · 0.01 s
```

Cuatro reglas de lo que se pinta y lo que no:

- **El verbo, el tema y la medida.** El verbo es la etiqueta corta de la herramienta ([[backend/02-interfaces/TOOLS]]); el tema es el **argumento objetivo** que su catálogo declara (la ruta, el patrón, el comando, la consulta), colapsado y recortado. Es solo ese campo: el resto de argumentos —cuerpos de archivo, credenciales— no se expone, porque esto va a pantalla y a auditoría. La medida es el tamaño del resultado en su unidad (`líneas`, `coincidencias`, `entradas`).
- **La salida, nunca.** El resultado va al modelo, no a la pantalla. Si quieres ver qué devolvió una herramienta, lo vas a ver explicado en la respuesta del agente.
- **Si se recortó, se dice.** Un resultado recortado sin avisar es peor que no verlo: el agente puede actuar como si tuviera el resultado entero.
- **Cuánto tardó, siempre que se sepa.** La duración de la ejecución va al final de la línea cerrada, con el separador de la línea (`· 0.4 s`) y con el mismo atenuado que el tiempo de la respuesta. No va entre paréntesis: ahí los paréntesis marcan «esto es el tiempo de una respuesta», y una línea de herramienta no lo es. Se pinta también cuando la ejecución falló —«✗ … · 0.01 s»—, porque tardó igual. El número sale del mismo formateador que el tiempo de la respuesta, de modo que los dos se leen igual. **Si la línea no tiene duración, no se pinta nada y el espacio no se deja reservado**: un hueco de medidas desiguales delata más que una omisión.

La línea se pinta en vivo y se **recarga con su tiempo**: al volver a la sesión, la línea guardada llega ya cerrada y con su duración, así que el hilo recuperado se lee igual que el que se vio en directo. Ver [[database/02-rules/DATA_FLOW]] §Creación y [[specs/SPEC-INTERFAZ]] §El chat.

El agente que la pidió no se pinta en la línea (sigue en el evento, para la auditoría).

**El orden del hilo.** La línea de herramienta se inserta en su sitio cronológico, no en bloque al final del turno: el texto que el modelo escribió antes queda en un globo arriba de ella y el que escribe después abre un globo nuevo. Cada vez que una línea se interpone en el hilo —una herramienta, una notificación, una etapa de flujo, un cambio aplicado—, el segmento de texto en curso se cierra antes de escribirla, con su razonamiento. Así el chat no funde en un solo globo el texto anterior y el posterior a la herramienta.

Una herramienta que necesita aprobación **no** muestra nada extra: la línea de la herramienta aparece, y debajo la línea de aprobación que ya existía. El usuario ve que el agente está actuando y que se le está pidiendo permiso por ello.

## 1.2 La línea de sub-proceso

Un flujo corre sus etapas como sub-procesos: cada una trabaja sin arrastrar el contexto del chat y su resultado se encadena a la siguiente ([[specs/SPEC-MOTOR-FLUJOS]]). En el hilo, cada etapa que empieza deja una sola línea con su nombre:

```
[Sub Proceso] Entender el problema
[Sub Proceso] Buscar contexto
[Sub Proceso] Diagnosticar
```

El texto de un paso intermedio **no** se pinta: el motor lo corre en silencio y solo alimenta la cadena. Lo que el usuario lee al final es la **entrega** del último paso del flujo, en un solo globo —por ejemplo, el PLAN del resolver con su salida estándar ([[specs/SPEC-RESOLVER]])—. Una etapa que pide aprobación sí muestra su salida, porque el usuario tiene que ver lo que aprueba; y una etapa que falla deja su aviso.

## 2. Peticiones que envía

Todas van a `session`, la única puerta del motor:

| Petición                  | Cuándo                                                                                                                                                                                                                |
| ------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Enviar mensaje            | El usuario escribe y confirma, tanto en la bienvenida (primera petición) como en el chat                                                                                                                              |
| Ejecutar comando de flujo | El usuario escribe un comando explícito (`/planificar`, `/crear`, `/actualizar`, `/eliminar`, `/resolver` o `/ejecutar`) y confirma, o lo elige en la paleta y pulsa `Enter`; el motor lo reconoce y arranca el flujo |
| Crear sesión              | Al enviar desde la bienvenida (o con `Ctrl+X n`): nace una sesión nueva con nombre provisional «Nueva sesión» y va el mensaje; el modelo le pondrá título con esa primera petición                                    |
| Cambiar de sesión         | Elige en el selector momentáneo; también desde la bienvenida, donde al elegir la vista pasa a la principal con el historial de esa sesión                                                                             |
| Aprobar / declinar        | Resuelve una línea del panel de aprobaciones                                                                                                                                                                          |
| Cancelar flujo            | Lo pide con su atajo; si había flujo en marcha, `session` pregunta qué hacer, según [[specs/SPEC-SESIONES]]                                                                                                           |

## 3. Lecturas a `store`

Solo lectura, con las consultas de [[database/03-operations/QUERIES]]: historial de la sesión activa, la lista de pasos de la sesión activa, aprobaciones pendientes de todas las sesiones, auditoría de una etapa y datos del panel. Nunca escribe: si algo cambia, es el motor quien lo persiste y notifica.

### 3.1 Lecturas del arranque

Dos datos del panel son fijos mientras la sesión vive, así que se leen **una sola vez**, al arrancar, y se pintan en el pie del sidebar. La pantalla no los consulta ni los revalida:

| Lectura     | Qué trae                                                 | Nota                                                                                                                      |
| ----------- | -------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------- |
| `Carpeta()` | La carpeta del proyecto                                  | Se abrevia con `~` y se recorta por la izquierda                                                                          |
| `Git()`     | La rama activa y si el árbol tiene cambios sin confirmar | Se llama al binario `git`; rama vacía = proyecto sin git inicializado o sin git instalado, y el pie lo dice `sin iniciar` |

Que no se revalide es una decisión, no un descuido: una sesión trabaja siempre en la misma
rama y llamar a git en cada repintado costaría un proceso por frame. El precio es que un
cambio hecho en la terminal con la sesión abierta no se refleja en el pie hasta la
siguiente.

## 4. Teclado

Atajos por defecto, reasignables desde la ayuda y guardados en `~/.config/localcli/keys.json`. El último modelo y el último agente usados se recuerdan en `~/.config/localcli/config.json`. El teclado pasa por un resolver central (`KeyResolver`, ver [[specs/SPEC-KEYBINDS]]): acciones con ID estable, múltiples bindings por acción, tecla líder `Ctrl+X` con timeout 2000 ms y resolución por contexto (modal → input → vista → global). Los componentes reciben acciones, nunca teclas:

| Atajo                     | Acción                                                                                                                | Contexto                                  |
| ------------------------- | --------------------------------------------------------------------------------------------------------------------- | ----------------------------------------- |
| `Ctrl+C`                  | Salir (`app_exit`)                                                                                                    | global, también con modales abiertos      |
| `Ctrl+X m`                | Abrir el modal de modelos (`model_picker`)                                                                            | global                                    |
| `Ctrl+X l`                | Abrir el modal de sesiones; al elegir una con `Enter` se abre esa sesión (`session_picker`)                           | global                                    |
| `Ctrl+X n`                | Crear una sesión nueva y dejarla activa (`session_new`)                                                               | vista principal                           |
| `Ctrl+D`                  | Eliminar la sesión resaltada en el modal; si trabaja, pide confirmación (`session_delete`)                            | modal de sesiones                         |
| `Ctrl+P`                  | Abrir el modal con la lista de atajos existentes (`command_palette`)                                                  | global                                    |
| `Tab`                     | Cambiar de agente: recorre los disponibles (`agent_cycle`); el agente activo se pinta en el pie de la caja de entrada | vista y bienvenida (no con modal abierto) |
| `Esc`                     | Cerrar cualquier modal (`dismiss`)                                                                                    | modal                                     |
| `Esc`                     | Cancelar el trabajo en curso: el primero pide confirmación, el segundo cancela (doble `esc`)                          | vista, sesión trabajando                  |
| `↑` / `↓`                 | Navegar la lista del modal abierto                                                                                    | modal                                     |
| `↑` / `↓`                 | Recorrer el historial del chat (`chat_scroll_up`/`chat_scroll_down`)                                                  | vista principal                           |
| `PgUp` / `PgDn`           | Página arriba / abajo en el historial (`chat_page_up`/`chat_page_down`)                                               | vista principal                           |
| `←` / `→`, `Home` / `End` | Mover el cursor del input y editar en cualquier punto                                                                 | input                                     |
| `Enter`                   | Aplicar lo resaltado en el modal y cerrarlo                                                                           | modal                                     |
| `Enter`                   | Enviar la petición                                                                                                    | input                                     |
| `Ctrl+D`                  | Plegar o desplegar el sidebar de datos, que arranca visible (`panel_toggle`)                                          | vista                                     |
| `Ctrl+R`                  | Mostrar u ocultar el razonamiento                                                                                     | vista                                     |
| `Ctrl+A`                  | Abrir el panel de aprobaciones                                                                                        | vista                                     |
| `Ctrl+F`                  | Cancelar el flujo en curso (pide confirmación)                                                                        | vista                                     |
| `a` / `d`                 | Aprobar / declinar la línea seleccionada                                                                              | panel de aprobaciones                     |

Las sesiones se crean con `Ctrl+X n` desde la vista principal o enviando la primera petición desde la bienvenida. No hay ayuda por `?`: el listado de atajos es el modal de `Ctrl+P`.

**Paleta de comandos de flujo.** Al escribir `/` en la entrada —principal o bienvenida— se despliega encima la lista de comandos que sirve el motor (los oficiales y los propios de `.localcli/flows/*.json`). Mientras está desplegada, `↑`/`↓` la recorren, `Tab` autocompleta el comando resaltado dejando la línea lista para la petición (`/comando [petición]`) y `Enter` lo ejecuta; un espacio retira la paleta y lo escrito pasa a ser la petición. Lo que no coincide con ningún comando —aunque empiece por `/`— se responde como chat.

**Ratón.** La rueda desplaza el historial del chat. Arrastrar con el botón izquierdo selecciona texto, que se **resalta en video inverso** mientras se elige, y al soltar se copia al portapapeles —el realce desaparece y aparece el aviso transitorio `[Copiado]` arriba a la derecha, que se apaga solo—. Al capturar el ratón —necesario para poder copiar—, la selección nativa de la terminal queda disponible manteniendo `Shift`. La selección queda **anclada al texto**: la rueda puede usarse mientras se selecciona y no la cancela, así que en un chat largo se puede seguir eligiendo al desplazarse. Con el panel de aprobaciones visible, un clic sobre «aprobar» o «declinar» de una línea resuelve esa aprobación, sin necesidad de darle el foco con el teclado (un clic, no un arrastre: arrastrar sigue seleccionando texto).

Reglas, según [[specs/SPEC-INTERFAZ-ATAJOS]] y [[specs/SPEC-KEYBINDS]]:

- Un atajo no puede quedar asignado a dos acciones; el duplicado se rechaza al guardar.
- Una acción admite varios atajos y puede deshabilitarse con lista vacía en keys.json.
- Con un modal abierto, sus teclas (flechas/enter/esc) no llegan a la vista de abajo; con el input enfocado, las letras sueltas escriben y no activan acciones.
- Tras pulsar la líder sin segunda tecla, el estado vuelve a NORMAL al expirar el timeout.
- Reasignar no cambia reglas de permiso, solo la forma de invocar.
- El cambio se guarda sin reiniciar la aplicación.

## 5. Estados de espera

- Si la sesión activa está generando, la entrada sigue activa: escribir no bloquea ni cancela nada.
- La línea de entrada envuelve en varias filas lo que no cabe en el ancho, sin recortarlo, y reajusta el reparto al redimensionar la terminal; `Enter` envía y no inserta saltos. En el chat, lo del usuario y lo del agente se pintan en globos con color propio.
- Un pegado o arrastre se muestra como token: cada archivo `[nombre.ext]`, cada carpeta `[CARPETA N elementos]` y un texto de varias líneas `[PEGADO N líneas]` (un token por elemento si todas las líneas son rutas). Al enviar se expande al valor real: la ruta, el texto entero o, si es imagen, la imagen adjunta.
- La caja de la entrada lleva en su pie el agente activo, el modelo en uso y sus capacidades (`sí`/`no`, o `?` mientras se desconoce), incluido el **interruptor de razonamiento** (`pensar [x]`/`pensar [ ]`): se pulsa con el ratón, llega apagado y solo se enseña si el modelo declara que razona. El conteo de tokens del turno se ve justo debajo de la caja.
- **Si el modelo en uso no puede usar herramientas, se dice explícitamente** que el agente va a conversar sin ellas. Es una diferencia entre «todavía no lo sé» y «este modelo no puede», y confundirlas hace que el usuario espere un trabajo que no va a pasar. Ver [[specs/SPEC-OLLAMA-PERFIL]].
- Cada línea de herramienta dice cuánto tardó su ejecución, y esa línea se guarda con su tiempo: al recargar la sesión, el hilo conserva las duraciones además de las medidas.
- Mientras una herramienta se ejecuta, su línea está en el chat. Si la sesión espera permiso por una herramienta, la línea de la herramienta y la de aprobación coexisten.
- Con la sesión trabajando, el primer `esc` pide confirmación («presiona esc otra vez para cancelar razonamiento») y el segundo cancela; cualquier otra tecla la descarta.
- Si el usuario cierra una sesión con un flujo en marcha, la TUI muestra la pregunta de qué hacer con el flujo; la decisión la aplica `session`.
- Mientras una sesión espera permiso, su estado se ve en selector, panel y, si procede, en la línea de aviso.
- **Una sesión esperando tu aprobación no bloquea a las demás.** Si en otra pestaña hay una sesión generando, sigue generando mientras decides. Ver [[specs/SPEC-SESIONES]].

## Referencias

- [[frontend/FRONTEND]] — mapa de la capa.
- [[backend/04-infrastructure/EVENTS]] — productores y payloads, incluidos los de herramienta.
- [[backend/02-interfaces/TOOLS]] — la capa universal que emite esos eventos.
- [[database/03-operations/QUERIES]] — las consultas de lectura que puede ejecutar.
- [[specs/SPEC-INTERFAZ-ATAJOS]] — reglas funcionales de los atajos.
- [[specs/SPEC-TOOLS]] — el catálogo, y qué se pinta de una ejecución.
