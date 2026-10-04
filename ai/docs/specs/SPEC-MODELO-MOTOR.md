---
title: SPEC — Proveedor de modelo y perfil de hardware
tags: [specs, requisito]
depende_de:
  - "[[IDEA]]"
  - "[[specs/SPEC-SESIONES]]"
  - "[[specs/SPEC-NODO-CONTEXTO]]"
  - "[[specs/SPEC-PANEL-CONTEXTO]]"
  - "[[PROJECT]]"
relacionado:
  - "[[specs/SPEC-TOOLS]]"
---
# SPEC — Proveedor de modelo y perfil de hardware

Prioridad: P0 (núcleo)

## Propósito

Hablar con el modelo local de forma que quepa y rinda en una máquina con 4 GB de VRAM y 16 GB de RAM, sea cual sea el runtime que lo sirva.

## Alcance

Incluye conectarse al proveedor de modelo, ver los modelos disponibles, proponer un perfil que quepa en el hardware objetivo y avisar cuando no cabe. Incluye la elección de proveedor, cómo se le pide al modelo que use herramientas y qué ocurre cuando el modelo elegido no sabe.
No incluye cambiar de proveedor con la sesión viva, ni la visualización de tokens, que está en [[specs/SPEC-PANEL-CONTEXTO]].

## Actores

- **Usuario**: elige el modelo y acepta o cambia el perfil, y al arrancar el proveedor que lo sirve.
- **Sistema**: detecta, informa de qué cabe y avisa cuando no cabe.

## Proveedores

El modelo se pide siempre a un **proveedor**: el runtime que lo sirve. Hay dos:

| Clave | Runtime | Dirección por defecto |
|---|---|---|
| `ollama` | Ollama | `http://localhost:11434` |
| `llamacpp` | `llama-server` de llama.cpp, en modo router | `http://localhost:8080` |

**El proveedor se elige al arrancar y no cambia mientras la sesión vive.** Cambiarlo es reiniciar LocalCli. No hay forma de pasar de uno a otro con el modal abierto: el modal elige modelo, no runtime.

Se elige con `LOCALCLI_PROVEEDOR`:

- Sin declarar: se usa `ollama`. El comportamiento por defecto no depende de lo que haya instalado.
- `ollama` o `llamacpp`: se usa ese, y solo ese. Si no responde, avisa; no cae al otro por su cuenta.
- `auto`: se usa el primero que responde, empezando por `ollama`. Es opcional a propósito: quien lo declara quiere esa conmutación.
- `llamacpp` acepta además `LOCALCLI_LLAMACPP_URL` para la dirección del servidor.

Un proveedor que no responde **no impide arrancar**: la interfaz sigue viva, avisa de qué falta y con qué instrucción levantarlo, y espera a que se escriba algo.

## Lo que cambia por proveedor

El harness habla **un** contrato con el modelo y cada proveedor lo traduce. Lo que cambia es cómo se llega a él:

| | Ollama | llama.cpp |
|---|---|---|
| Herramientas | canal nativo, nombre y esquema | canal nativo, nombre y esquema |
| Razonamiento | campo `think` por petición | `reasoning_content` en la respuesta |
| Imágenes | `images` en base64 por mensaje | `image_url` con data URI |
| Contar tokens | contadores del servidor | contadores de `usage` |
| **Ventana de contexto** | **se declara** en cada petición | **se lee**: el servidor la fijó al arrancar |
| Lista de modelos | la que declara el servidor | la que declara el servidor |

La fila de la ventana es la única diferencia que no se resuelve traduciendo: cambia lo que el harness tiene que hacer, y por eso tiene su propia sección.

## Pedir herramientas al modelo

Las herramientas se piden por el canal nativo del proveedor, no escribiéndolas en el mensaje. En cada petición al modelo viajan sus definiciones —nombre, descripción y esquema de argumentos— y el modelo responde pidiendo una por su nombre con los argumentos ya formados. Cuando pide varias, se ejecutan en el orden en que las pidió y sus resultados vuelven en la misma petición siguiente.

El sistema no le enseña al modelo a escribir una llamada: no tiene ningún formato que imitar. El resultado es que una llamada mal formada es un caso que no se da, y que los argumentos se pueden comprobar antes de ejecutar nada.

Esto tiene un coste que hay que asumir: **exige un modelo con capacidad de herramientas**. Se pregunta al proveedor antes de elegir y la respuesta se muestra en la interfaz.

### Cuando el modelo no tiene esa capacidad

Un modelo que no declara capacidad de herramientas **no puede usarlas**: ni las incluidas ni las que declare el usuario, porque no hay forma de pedírselas que no sea el canal nativo.

No se le impide elegirlo. La herramienta sigue funcionando, pero el agente activo cae a **modo conversación**: responde con el contexto que recibe y no ejecuta nada. La interfaz lo dice bajo la línea de entrada.

Un modelo así puede ser exactamente el adecuado para conversar, así que se avisa en vez de bloquear. Lo que no hace LocalCli es presentárselo como si pudiera trabajar.

### Turnos de varias pasadas

Un turno puede necesitar más de una petición al modelo: pide una herramienta, ve el resultado y vuelve a pedir. Entre pasada y pasada hay un momento en que el modelo no está generando porque el usuario está contestando una aprobación. Ese momento **no debe bloquear a otras sesiones**: cada petición toma el turno de inferencia y lo suelta al terminar, de modo que una sesión esperando una aprobación deja libre el modelo para las demás.

## La ventana de contexto

Un turno con herramientas —prompt del agente, esquemas del catálogo, historial y salidas de herramienta— necesita una ventana que no se recorta por el camino. Qué hace el harness con ella depende del proveedor:

- **Declarable** (Ollama): la ventana viaja en cada petición, derivada del modelo y acotada por el tope. El servidor la obedece.
- **Declarada por el usuario** (llama.cpp): la ventana la fijó quien arrancó el servidor y no se puede cambiar por petición. El harness **la lee** y trabaja con la que hay.

En ambos casos la ventana efectiva es la menor entre la que declara el modelo, la que declara el servidor y el tope (`LOCALCLI_CONTEXT_LIMIT`, 16384 por defecto). El presupuesto del historial del turno se deriva de esa misma ventana, no de la que el modelo anunciaría.

Cuando la ventana efectiva es menor que la que un turno con herramientas necesita, **no es un fallo del modelo**: se avisa con la instrucción de levantarlo con más contexto y el turno se recorta para que quepa. Un turno que se recorta es un turno corto; un turno que muere sin respuesta no es aceptable.

El harness **no muta el servidor**: no le cambia la ventana, no le descarga modelos y no le pide que cargue ni descargue ninguno. Ese estado es del usuario.

## Varios modelos a la vez

Cada sesión pide por su cuenta y el sistema las va sirviendo **por orden de llegada**. No es arbitrario y no es "la última que pidió gana": la primera que llegó es la primera que se atiende.

La consecuencia de diseño que importa: una sesión que está esperando al usuario no está bloqueando a las demás. Si una herramienta pide aprobación y tardas medio minuto en contestar, otra sesión sigue generando con normalidad durante ese medio minuto.

## Flujo principal

1. La herramienta elige el proveedor: el declarado en `LOCALCLI_PROVEEDOR`, u `ollama` si no hay ninguno.
2. Se conecta a él y lee los modelos disponibles.
3. Muestra cuáles caben en el hardware detectado y cuáles no, y cuáles saben usar herramientas.
4. **El usuario elige el modelo.** La herramienta no decide por él.
5. Si el modelo elegido no cabe, avisa y no lo carga en silencio.
6. Si el modelo elegido no sabe usar herramientas, avisa de que el agente conversará sin ellas, y sigue.
7. El perfil queda guardado y se aplica a las sesiones nuevas.

## Flujos alternativos

- El proveedor no está corriendo: avisa y explica cómo levantarlo.
- No hay ningún modelo que quepa en el hardware.
- El modelo elegido no cabe: lo dice y no lo carga.
- El modelo elegido no sabe usar herramientas: el agente conversa sin ellas y la interfaz lo avisa.
- El servidor del proveedor expone menos contexto del que necesita un turno con herramientas: avisa con la instrucción y recorta.
- La máquina tiene más recursos: se puede subir el tamaño de contexto.

## Reglas de negocio

- El modelo lo elige el usuario. La herramienta informa y avisa, no decide.
- El proveedor también se elige, y al arrancar: se declara en `LOCALCLI_PROVEEDOR` y su valor por defecto es `ollama`.
- El proveedor no cambia con la sesión viva. El modal de modelos elige modelo dentro del proveedor ya elegido.
- `auto` existe como valor opcional y, al declararlo, se usa el primer proveedor que responde empezando por `ollama`.
- Un proveedor que no responde avisa con una instrucción clara de cómo levantarlo, y no impide que la interfaz arranque.
- El harness mantiene un solo contrato con el modelo y cada proveedor lo traduce: quien llama al modelo no sabe en qué formato viaja la respuesta.
- La herramienta muestra qué modelos caben en el hardware detectado antes de que el usuario elija.
- El perfil por defecto se calcula para 4 GB de VRAM y 16 GB de RAM.
- El tamaño de contexto se limita para que quepa junto con el modelo cargado.
- Si el modelo elegido no cabe, la herramienta avisa y no lo carga en silencio.
- El perfil se aplica a las sesiones nuevas del proyecto.
- El nombre del proveedor se muestra en el modal de modelos y en la línea de modelo, para que se sepa contra qué runtime se está hablando.
- El último modelo, el último agente y el último proveedor usados se recuerdan entre ejecuciones como preferencia global del usuario (fuera del proyecto). Al arrancar se reutiliza el modelo recordado **solo si su proveedor es el elegido**; con otro proveedor se autodetecta, porque el mismo nombre no significa lo mismo en los dos runtimes. Y el agente recordado.
- Si el modelo no declara capacidad de herramientas, la interfaz lo marca y avisa sin bloquear al elegirlo, para que el usuario cambie de modelo si quiere. No se le impide usarlo: el agente activo cae a modo conversación, responde sin ejecutar nada y la interfaz lo dice. LocalCli no le presenta herramientas en texto para que las imite.
- Las capacidades se leen al proveedor y se normalizan a las tres que el harness entiende —herramientas, visión y razonamiento—, venga de donde vengan. Ninguna lógica del harness lee el formato del proveedor.
- Las herramientas se piden por el canal nativo del proveedor, con su nombre y su esquema. No hay formato en prosa que el modelo tenga que imitar.
- Las herramientas se ejecutan en el orden en que el modelo las pidió, nunca en paralelo.
- El turno de inferencia se toma por petición al modelo y se suelta al terminar, de manera que una sesión esperando una aprobación de herramienta no impide que otra sesión genere.
- La ventana de contexto se declara por petición cuando el proveedor la acepta y se lee del servidor cuando no. En los dos casos la efectiva es la menor entre lo que declara el modelo, lo que declara el servidor y el tope, y el presupuesto del historial se deriva de ella.
- Si la ventana efectiva no alcanza para un turno con herramientas, se avisa con la instrucción de levantarlo con más contexto y se recorta el turno; no se deja morir.
- El harness no cambia el estado del servidor del proveedor: ni la ventana, ni los modelos cargados o descargados.
- Si el texto de un mensaje de chat incluye la ruta de una imagen existente, la imagen se adjunta a ese turno hacia el modelo, en el formato de imágenes que declara el proveedor. Solo el adaptador del proveedor conoce ese formato.
- Las imágenes adjuntas son efímeras: no se persisten ni se replican en el historial. Un turno posterior que no vuelva a mencionar la imagen no la ve.
- Si el modelo en uso no declara capacidad de visión y el turno lleva imágenes, la interfaz avisa sin bloquear el envío; no se le impide usarlo.
- Bajo la línea de entrada se muestra el modelo en uso y si tiene acceso a herramientas y a la visión (`sí`/`no`/`?` mientras se desconoce). Cuando el modelo no puede usar herramientas, se indica que el agente va a conversar sin ellas.
- El **razonamiento** de un modelo que lo declara llega apagado: se le pide que no razone en cada petición. En Ollama se hace con `think: false`; el runtime lo deja encendido por defecto en esos modelos y, en local, razonar cuesta minutos hasta para lo trivial. El usuario lo enciende por turno con el interruptor del pie (`pensar [x]`), que se pulsa con el ratón y se recuerda entre ejecuciones. A un modelo que no declara la capacidad no se le pide razonamiento —no entiende la opción— y sin ficha tampoco: no saberlo no es «no puede».
- Un servidor de llama.cpp que exponga lectura o escritura de archivos no es un proveedor admisible: se documenta que tiene que levantarse sin esas opciones.
- Todo el modelo y toda la conversación ocurren en la máquina local: no se envía nada fuera.
- La única excepción es la búsqueda en internet de [[specs/SPEC-TOOLS]], y solo sale la consulta, nunca contenido del proyecto.
- El tamaño de contexto disponible se tiene en cuenta al decidir cuánto contexto entregar.

## Criterios de aceptación

- [ ] Sin declarar `LOCALCLI_PROVEEDOR`, se usa Ollama.
- [ ] Con `LOCALCLI_PROVEEDOR=llamacpp`, la lista de modelos y los turnos van contra el servidor de llama.cpp declarado en `LOCALCLI_LLAMACPP_URL`.
- [ ] Con `LOCALCLI_PROVEEDOR=auto`, se usa el primer proveedor que responde, empezando por Ollama.
- [ ] Un proveedor que no responde avisa con una instrucción clara y la interfaz sigue viva.
- [ ] El nombre del proveedor se ve en el modal de modelos y en la línea de modelo.
- [ ] El modelo recordado solo se reutiliza si su proveedor es el elegido; con otro, se autodetecta.
- [ ] La herramienta se conecta al proveedor y lista los modelos disponibles.
- [ ] Indica cuáles caben en el hardware detectado antes de que el usuario elija.
- [ ] El usuario elige el modelo; la herramienta no lo decide por él.
- [ ] Si el modelo elegido no cabe, avisa y no lo carga.
- [ ] El perfil confirmado se aplica a las sesiones nuevas.
- [ ] Al reabrir LocalCli se parte del último modelo, el último agente y el último proveedor usados.
- [ ] Un modelo que no declara capacidad de herramientas se marca y avisa sin bloquear.
- [ ] Un modelo que no declara capacidad de herramientas hace que el agente caiga a modo conversación, sin ejecutar nada.
- [ ] La interfaz dice bajo la entrada que el agente va a conversar sin herramientas cuando ese es el caso.
- [ ] Un turno que necesita varias herramientas hace una petición al modelo por cada paso, con los resultados de la anterior.
- [ ] Una sesión esperando una aprobación de herramienta no impide que otra sesión genere.
- [ ] Dos sesiones que piden a la vez se atienden por orden de llegada.
- [ ] Las herramientas viajan por el canal nativo del proveedor contra los dos, sin formato en prosa.
- [ ] Las imágenes de un turno de chat llegan al modelo en el formato de imágenes que declara el proveedor.
- [ ] Un modelo que no declara visión avisa, sin bloquear, al adjuntar una imagen.
- [ ] Un modelo que declara razonamiento no razona si el interruptor del pie está apagado, y razona si el usuario lo enciende.
- [ ] Un modelo que no declara razonamiento no recibe la petición de razonar.
- [ ] Las capacidades de herramientas, visión y razonamiento se resuelven igual contra los dos proveedores.
- [ ] Bajo la entrada se ve el modelo en uso y si tiene acceso a herramientas y a la visión.
- [ ] Con un proveedor que declara más contexto que el tope, el turno usa el tope.
- [ ] Con un servidor cuya ventana efectiva es menor que la que necesita un turno con herramientas, avisa con la instrucción de levantarlo con más contexto y el turno se recorta en vez de morir.
- [ ] El harness no cambia la ventana, ni los modelos cargados o descargados, del servidor del proveedor.
- [ ] La herramienta sigue funcionando con 16 GB de RAM sin agotar la memoria.
- [ ] La única información que sale de la máquina es la consulta de una búsqueda, nunca contenido del proyecto.

## Requisitos no funcionales

- Consumo de VRAM dentro de 4 GB en la configuración por defecto.
- No añadir esperas propias al tiempo de respuesta del modelo.
- Una sesión bloqueada esperando al usuario no retiene el turno de inferencia.
- Añadir un proveedor no añade dependencias de terceros: el transporte es el cliente HTTP de la biblioteca estándar.
- El harness no muta el estado del servidor del proveedor: es de la persona que lo levanta.

## Dependencias funcionales

- [[specs/SPEC-SESIONES]]
- [[specs/SPEC-NODO-CONTEXTO]]
- [[specs/SPEC-PANEL-CONTEXTO]]
- [[PROJECT]]

## Supuestos

- Los valores concretos del perfil (modelo, tamaño de contexto, parámetros de inferencia) se fijan en FASE 2 y FASE 3.
- Los dos proveedores sirven los mismos pesos: lo que cambia es el runtime, no el modelo.

## Referencias

- [[IDEA]]