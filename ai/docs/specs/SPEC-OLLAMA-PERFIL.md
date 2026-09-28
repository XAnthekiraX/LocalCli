---
title: SPEC — Integración con Ollama y perfil de hardware
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
# SPEC — Integración con Ollama y perfil de hardware

Prioridad: P0 (núcleo)

## Propósito

Hablar con el modelo local de forma que quepa y rinda en una máquina con 4 GB de VRAM y 16 GB de RAM.

## Alcance

Incluye conectarse a Ollama, ver los modelos disponibles, proponer un perfil que quepa en el hardware objetivo y avisar cuando no cabe. Incluye cómo se le pide al modelo que use herramientas y qué ocurre cuando el modelo elegido no sabe.
No incluye proveedores distintos de Ollama ni la visualización de tokens, que está en [[specs/SPEC-PANEL-CONTEXTO]].

## Actores

- **Usuario**: elige el modelo y acepta o cambia el perfil.
- **Sistema**: detecta, informa de qué cabe y avisa cuando no cabe.

## Pedir herramientas al modelo

Las herramientas se piden por el canal nativo de Ollama, no escribiéndolas en el mensaje. En cada petición al modelo viajan sus definiciones —nombre, descripción y esquema de argumentos— y el modelo responde pidiendo una por su nombre con los argumentos ya formados. Cuando pide varias, se ejecutan en el orden en que las pidió y sus resultados vuelven en la misma petición siguiente.

El sistema no le enseña al modelo a escribir una llamada: no tiene ningún formato que imitar. El resultado es que una llamada mal formada es un caso que no se da, y que los argumentos se pueden comprobar antes de ejecutar nada.

Esto tiene un coste que hay que asumir: **exige un modelo con capacidad de herramientas**. Se pregunta a Ollama antes de elegir y la respuesta se muestra en la interfaz.

### Cuando el modelo no tiene esa capacidad

Un modelo que no declara capacidad de herramientas **no puede usarlas**: ni las incluidas ni las que declare el usuario, porque no hay forma de pedírselas que no sea el canal nativo.

No se le impide elegirlo. La herramienta sigue funcionando, pero el agente activo cae a **modo conversación**: responde con el contexto que recibe y no ejecuta nada. La interfaz lo dice bajo la línea de entrada.

Un modelo así puede ser exactamente el adecuado para conversar, así que se avisa en vez de bloquear. Lo que no hace LocalCli es presentselo como si pudiera trabajar.

### Turnos de varias pasadas

Un turno puede necesitar más de una petición al modelo: pide una herramienta, ve el resultado y vuelve a pedir. Entre pasada y pasada hay un momento en que el modelo no está generando porque el usuario está contestando una aprobación. Ese momento **no debe bloquear a otras sesiones**: cada petición toma el turno de inferencia y lo suelta al terminar, de modo que una sesión esperando una aprobación deja libre el modelo para las demás.

## Varios modelos a la vez

Cada sesión pide por su cuenta y el sistema las va sirviendo **por orden de llegada**. No es arbitrario y no es "la última que pidió gana": la primera que llegó es la primera que se atiende.

La consecuencia de diseño que importa: una sesión que está esperando al usuario no está bloqueando a las demás. Si una herramienta pide aprobación y tardas medio minuto en contestar, otra sesión sigue generando con normalidad durante ese medio minuto.

## Flujo principal

1. La herramienta se conecta a Ollama.
2. Lee los modelos disponibles.
3. Muestra cuáles caben en el hardware detectado y cuáles no, y cuáles saben usar herramientas.
4. **El usuario elige el modelo.** La herramienta no decide por él.
5. Si el modelo elegido no cabe, avisa y no lo carga en silencio.
6. Si el modelo elegido no sabe usar herramientas, avisa de que el agente conversará sin ellas, y sigue.
7. El perfil queda guardado y se aplica a las sesiones nuevas.

## Flujos alternativos

- Ollama no está corriendo: avisa y explica cómo levantarlo.
- No hay ningún modelo que quepa en el hardware.
- El modelo elegido no cabe: lo dice y no lo carga.
- El modelo elegido no sabe usar herramientas: el agente conversa sin ellas y la interfaz lo avisa.
- La máquina tiene más recursos: se puede subir el tamaño de contexto.

## Reglas de negocio

- El modelo lo elige el usuario. La herramienta informa y avisa, no decide.
- La herramienta muestra qué modelos caben en el hardware detectado antes de que el usuario elija.
- El perfil por defecto se calcula para 4 GB de VRAM y 16 GB de RAM.
- El tamaño de contexto se limita para que quepa junto con el modelo cargado.
- Si el modelo elegido no cabe, la herramienta avisa y no lo carga en silencio.
- El perfil se aplica a las sesiones nuevas del proyecto.
- El último modelo y el último agente usados se recuerdan entre ejecuciones como preferencia global del usuario (fuera del proyecto). Al arrancar se reutiliza el modelo recordado si sigue instalado, y el agente recordado.
- Si el modelo no declara capacidad de herramientas, la interfaz lo marca y avisa sin bloquear al elegirlo, para que el usuario cambie de modelo si quiere. No se le impide usarlo: el agente activo cae a modo conversación, responde sin ejecutar nada y la interfaz lo dice. LocalCli no le presenta herramientas en texto para que las imite.
- Las herramientas se piden por el canal nativo de Ollama, con su nombre y su esquema. No hay formato en prosa que el modelo tenga que imitar.
- Las herramientas se ejecutan en el orden en que el modelo las pidió, nunca en paralelo.
- El turno de inferencia se toma por petición al modelo y se suelta al terminar, de manera que una sesión esperando una aprobación de herramienta no impide que otra sesión genere.
- Si el texto de un mensaje de chat incluye la ruta de una imagen existente, la imagen se adjunta a ese turno hacia el modelo: viaja codificada en base64 en el campo `images` de `/api/chat`, que es lo que aceptan los modelos multimodales. Solo el módulo `ollama` conoce ese formato.
- Las imágenes adjuntas son efímeras: no se persisten ni se replican en el historial. Un turno posterior que no vuelva a mencionar la imagen no la ve.
- Si el modelo en uso no declara capacidad de visión (`vision` en `/api/show`) y el turno lleva imágenes, la interfaz avisa sin bloquear el envío; no se le impide usarlo.
- Bajo la línea de entrada se muestra el modelo en uso y si tiene acceso a herramientas y a la visión (`sí`/`no`/`?` mientras se desconoce). Cuando el modelo no puede usar herramientas, se indica que el agente va a conversar sin ellas.
- Todo el modelo y toda la conversación ocurren en la máquina local: no se envía nada fuera.
- La única excepción es la búsqueda en internet de [[specs/SPEC-TOOLS]], y solo sale la consulta, nunca contenido del proyecto.
- El tamaño de contexto disponible se tiene en cuenta al decidir cuánto contexto entregar.

## Criterios de aceptación

- [ ] La herramienta se conecta a Ollama y lista los modelos disponibles.
- [ ] Indica cuáles caben en el hardware detectado antes de que el usuario elija.
- [ ] El usuario elige el modelo; la herramienta no lo decide por él.
- [ ] Si el modelo elegido no cabe, avisa y no lo carga.
- [ ] El perfil confirmado se aplica a las sesiones nuevas.
- [ ] Al reabrir LocalCli se parte del último modelo y el último agente usados.
- [ ] Un modelo que no declara capacidad de herramientas se marca y avisa sin bloquear.
- [ ] Un modelo que no declara capacidad de herramientas hace que el agente caiga a modo conversación, sin ejecutar nada.
- [ ] La interfaz dice bajo la entrada que el agente va a conversar sin herramientas cuando ese es el caso.
- [ ] Un turno que necesita varias herramientas hace una petición al modelo por cada paso, con los resultados de la anterior.
- [ ] Una sesión esperando una aprobación de herramienta no impide que otra sesión genere.
- [ ] Dos sesiones que piden a la vez se atienden por orden de llegada.
- [ ] Las imágenes de un turno de chat viajan al modelo en el campo `images` de `/api/chat`, codificadas en base64.
- [ ] Un modelo que no declara visión avisa, sin bloquear, al adjuntar una imagen.
- [ ] Bajo la entrada se ve el modelo en uso y si tiene acceso a herramientas y a la visión.
- [ ] La herramienta sigue funcionando con 16 GB de RAM sin agotar la memoria.
- [ ] Si Ollama no está disponible, avisa con una instrucción clara.
- [ ] La única información que sale de la máquina es la consulta de una búsqueda, nunca contenido del proyecto.

## Requisitos no funcionales

- Consumo de VRAM dentro de 4 GB en la configuración por defecto.
- No añadir esperas propias al tiempo de respuesta del modelo.
- Una sesión bloqueada esperando al usuario no retiene el turno de inferencia.

## Dependencias funcionales

- [[specs/SPEC-SESIONES]]
- [[specs/SPEC-NODO-CONTEXTO]]
- [[specs/SPEC-PANEL-CONTEXTO]]
- [[PROJECT]]

## Supuestos

- Los valores concretos del perfil (modelo, tamaño de contexto, parámetros de inferencia) se fijan en FASE 2 y FASE 3.

## Referencias

- [[IDEA]]
