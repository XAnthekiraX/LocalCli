---
title: SPEC — Modelo y motor de inferencia
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
# SPEC — Modelo y motor de inferencia

Prioridad: P0 (núcleo)

## Propósito

Hablar con el modelo local de forma que quepa y rinda en una máquina con 4 GB de VRAM y 16 GB de RAM, sea cual sea el runtime que lo sirva, y poder cambiar de runtime sin que eso signifique perder el trabajo.

## Alcance

Incluye administrar los motores de inferencia —agregarlos, configurarlos, desactivarlos y eliminarlos—, ver los modelos que cada uno ofrece, elegir modelo, proponer un perfil que quepa en el hardware objetivo y avisar cuando no cabe. Incluye el motor y el modelo que usa cada sesión, y qué ocurre cuando una sesión cambia de motor a mitad de conversación.

No incluye la visualización de tokens, que está en [[specs/SPEC-PANEL-CONTEXTO]], ni la estructura de las peticiones al modelo por etapa, que está en [[specs/SPEC-AGENTE-BASE]].

## Actores

- **Usuario**: administra los motores, elige el modelo de cada sesión y decide cuándo cambiar de uno a otro.
- **Sistema**: guarda el registro de motores, consulta sus modelos, normaliza lo que declaran y avisa cuando no cabe o no se puede usar.

## Motores

### Qué es un motor

Un **motor** es el runtime que ejecuta el modelo. No es un servicio al que uno se suscribe con una clave, ni una fuente distinta de modelos: es un proceso local que carga pesos y los ejecuta.

| Runtime | Qué es |
|---|---|
| `llama.cpp` | Motor de inferencia. Lee pesos GGUF y los ejecuta. |
| `ollama` | Runtime y servidor que gestiona modelos y expone una API. Por debajo termina ejecutando un motor como llama.cpp. |

La distinción importa porque **los dos sirven los mismos pesos**. Lo que cambia no es el modelo ni de dónde sale: es el motor que lo ejecuta y la API con la que se habla. Por eso el término correcto es *motor* y no *proveedor* — un proveedor sugiere una empresa de la que dependes, y aquí no hay tal cosa.

### Núcleo común y extensiones nativas

El harness habla **un solo protocolo** con todos los motores: la superficie **OpenAI-compatible**. Es el mínimo que todo motor debe hablar, y por eso un runtime que solo hable esto —un router nuevo, un servidor de inferencia de terceros— funciona sin que el harness sepa nada de él.

```text
┌──────────────────────────┐
│  Harness                 │   contrato neutro, sin endpoints
└────────────┬─────────────┘
             │  núcleo común obligatorio
             ▼
   /v1/chat/completions   SSE: delta.content, delta.reasoning_content,
                          delta.tool_calls, usage, [DONE]
   /v1/models             id, object, owned_by
             │
   ┌─────────┴──────────────┐
┌──▼────────┐          ┌────▼───────┐
│ ollama    │          │ llamacpp   │
└──┬────────┘          └────┬───────┘
   │ extensiones          │ extensiones
   ├ options.num_ctx      └ /props
   ├ /api/show
   └ /api/tags
```

Sobre ese núcleo, cada motor puede declarar **extensiones nativas**: endpoints propios de su runtime que el estándar no cubre y que aportan información que de otro modo se pierde. Son una declaración opcional por instancia, no un supuesto:

| Extensión | Motor | Qué aporta |
|---|---|---|
| `num_ctx` | `ollama` | Declarar la ventana en cada petición |
| `show` | `ollama` | Capacidades del modelo (`tools`, `vision`, `thinking`) |
| `tags` | `ollama` | Tamaño y familia del modelo, para el encaje con el hardware |
| `props` | `llamacpp` | Ventana que fijó el servidor y capacidades del servidor |

Una extensión ausente no es un fallo: el motor sigue funcionando por el núcleo común, y lo que la extensión aportaba pasa a **desconocido** —que es una tercera cosa, distinta de «no lo tiene» (ver *Capacidades*).

### El catálogo es cerrado

Hay dos tipos de motor y ninguno más:

| Tipo | Runtime | Dirección de referencia |
|---|---|---|
| `ollama` | Ollama | `http://localhost:11434` |
| `llamacpp` | `llama-server` de llama.cpp, en modo router | `http://localhost:8080` |

La dirección de referencia es **documentación, no configuración**: LocalCli no la aplica por su cuenta. Alguien instala LocalCli sin Ollama ni llama.cpp en la máquina, así que ninguna dirección se presupone alcanzable.

El catálogo de **tipos** es cerrado a propósito: el harness solo sabe qué extensiones nativas declara cada tipo. La alternativa abierta obligaría a que interpretara un protocolo desconocido en caliente, que es justo lo que no puede hacer un motor de este tamaño. El catálogo de **extensiones**, en cambio, está abierto por motor: cualquiera puede declararse y el núcleo la ignora si no la conoce.

### Varias instancias del mismo tipo

El **tipo** es cerrado; las **instancias** no. Cada motor registrado es una instancia con nombre propio:

```text
Ollama local            → tipo ollama    → http://localhost:11434
Ollama en la máquina grande → tipo ollama → http://portatil.local:11434
llama.cpp router 8080   → tipo llamacpp  → http://localhost:8080
llama.cpp router 8081   → tipo llamacpp  → http://localhost:8081
```

Esto no es una holgura de diseño: es lo que hace que «tengo el mismo modelo en dos máquinas» y «tengo dos routers con ventanas distintas» sean cosas expresables. Un modelo concreto vive **en un motor concreto**: `qwen3` en Ollama local y `qwen3` en el router 8080 son configuraciones distintas, aunque se llamen igual, y por eso nunca se pueden confundir al leer su ficha, su ventana o sus capacidades.

Cada motor tiene:

| Campo | Qué es |
|---|---|
| `id` | Identificador estable y único. Es la clave con la que se habla de él; no cambia al renombrarlo |
| `nombre` | Lo que ve el usuario. `Ollama local`, `llama.cpp 8080` |
| `tipo` | `ollama` o `llamacpp`. Determina qué extensiones nativas se pueden declarar |
| `url` | La dirección del servidor. Es la configuración de conexión y **no tiene valor por defecto**: hay que escribirla |
| `activo` | Si sus modelos están disponibles |
| `extensiones` | Extensiones nativas declaradas para esta instancia. Vacío significa «solo núcleo común» |

El nombre visible es el **de la instancia**, no el del tipo: dos motores `ollama` se distinguen por su nombre. Es lo que permite que el rotulado sea `Ollama local / qwen3` y `llama.cpp 8080 / qwen3`, y no dos filas indistinguibles.

## Lo que cambia por motor

Contra el núcleo común **no cambia nada**: los dos motores viajan igual. Transporte SSE por `/v1/chat/completions`, herramientas por `tool_calls`, razonamiento por `delta.reasoning_content`, imágenes por `image_url` con data URI, tokens por `usage`, modelos por `/v1/models`. Quien llama no sabe contra qué motor está hablando.

Lo que cambia es **qué extensiones trae declaradas**:

| Extensión | `ollama` | `llamacpp` | Sin la extensión |
|---|---|---|---|
| Ventana de contexto | `num_ctx` por petición | `props.n_ctx` del servidor | Se toma el tope y el harness recorta |
| Capacidades del modelo | `show.capabilities` | `props` | Desconocidas, no «no las tiene» |
| Tamaño y familia | `tags` | — | Desconocidos: no se puede comprobar el encaje |

Esto es lo que hace que el contrato neutro siga siendo neutro: la diferencia entre los dos motores no está en cómo se habla, sino en **cuánto se declara**. Un motor con menos extensiones no es peor motor; es un motor del que se sabe menos, y el harness lo dice en vez de suponer.

## Capacidades

Las capacidades del modelo —herramientas, visión, razonamiento— tienen **tres** estados, no dos. La razón es que *no saber* y *no tener* no son lo mismo, y confundirlos produce dos fallos distintos según el sentido:

- Confundir «desconocido» con «soportado» ofrece herramientas a un modelo que no las entiende, y la llamada llega mal formada.
- Confundir «desconocido» con «no soportado» apaga una capacidad real sin motivo, y el usuario no ve por qué.

| | `soportada` | `no soportada` | `desconocida` |
|---|---|---|---|
| **Herramientas** | se envían | no se ofrecen | **no se envían** y la interfaz muestra `?` |
| **Visión** | habilitada | no se ofrece | no se ofrece y la interfaz muestra `?` |
| **Razonamiento** | habilitada | desactivado | **desactivado y aviso al usuario** |

La asimetría es deliberada. Las herramientas y las imágenes viajan en la petición: mandarlas a ciegas hace que el modelo intente algo que no puede hacer y devuelva una llamada mal formada o ignore la imagen. El razonamiento viaja en una opción que el modelo entiende o no: activarla sin saber es pedirle algo que quizá no sepa hacer, así que se desactiva y **se avisa**, porque un interruptor que no hace nada observable sin explicación parece un fallo.

El estado `desconocida` se aplica cuando el motor no expone la extensión que declara las capacidades. La interfaz lo muestra con `?` bajo la línea de entrada, junto al nombre del motor, para que el `?` sea atribuible a la falta de información y no al modelo.

Cuando el estado es `desconocida` y el agente cae a modo conversación por no poderle ofrecer herramientas, **la interfaz lo dice**, igual que lo dice cuando la capacidad se sabe ausente: la degradación nunca es muda.

## Gestión de motores

Los motores se administran: se agregan, se editan, se desactivan, se reactivan y se eliminan. El registro vive en `~/.config/localcli/motores.json`, **fuera del proyecto**: un motor es un proceso de la máquina del usuario, no contenido del proyecto, igual que el mapa de teclas y el resto de preferencias.

### Nadie da por hecho qué hay instalado

LocalCli se instala en una máquina que puede no tener **ningún** runtime de inferencia. Nadie presupone que Ollama esté en `localhost:11434` ni que `llama-server` esté en `localhost:8080`, porque la instalación de LocalCli no instala motores.

Por eso hay dos reglas que van juntas:

- **El harness no siembra motores.** Si `motores.json` no existe, el registro queda vacío y eso es un estado válido y estable, no un fallo.
- **Ninguna dirección tiene valor por defecto.** `url` se escribe. Un motor sin dirección no se puede registrar.

Un registro ilegible tampoco rompe el arranque: se avisa y el registro queda vacío, igual que si no existiera.

### El registro se entrega escrito, no elegido

Como la configuración de los dos motores es siempre la misma —tipo y dirección de referencia—, LocalCli la entrega **ya escrita** en un `motores.json` de ejemplo, y lo que el usuario tiene que hacer es **activar** el que tenga instalado.

El archivo se entrega con las dos entradas y `activo: false`:

```json
{
  "motores": [
    { "id": "ollama-local", "nombre": "Ollama local", "tipo": "ollama",
      "url": "http://localhost:11434", "activo": false,
      "extensiones": ["num_ctx", "show", "tags"] },
    { "id": "llamacpp-local", "nombre": "llama.cpp local", "tipo": "llamacpp",
      "url": "http://localhost:8080", "activo": false,
      "extensiones": ["props"] }
  ],
  "modelos_por_motor": {}
}
```

Ninguna entra activa porque **no se sabe cuál está instalado**. El usuario activa la que corresponda con el modal de motores, y a partir de ahí esa entrada se comporta como cualquier motor agregado a mano: se le consultan sus modelos, se le guardan y se le puede editar o desactivar.

Esto no contradice que no haya proveedor por defecto: el default que no existe es el **activo**. Las direcciones y los tipos son datos, y están escritos para que el usuario no tenga que buscarlos. Lo que el harness no hace es elegir, activar ni presuponer.

### Agregar

Se configura el motor —nombre, tipo y dirección— y el harness consulta al servidor los modelos que ofrece y los registra a su nombre. El alta no adivina: si el servidor no responde, el motor se avisa pero no se registra a medias.

Los modelos registrados son el **último conocido**: se refrescan cada vez que el harness conecta con el motor. Conservarlos sirve para que un motor desactivado siga mostrando qué tenía, y para que la lista no desaparezca cuando el servidor está caído.

Al agregar, el nombre de la instancia es lo que la distingue: si se registra una segunda `ollama`, el usuario le da un nombre que las separe en pantalla.

### Editar

Cambiar nombre, tipo o dirección **no afecta a las sesiones activas**. El motor se sigue usando con la configuración con la que ya se conectó y el cambio se aplica cuando se reinicia el harness.

La razón es que el adaptador ya está construido y en uso: cambiar su dirección debajo dejaría a la mitad de un turno hablando con un sitio y a la otra mitad con otro. Editar es una decisión para el próximo arranque, no para el actual.

### Desactivar

El motor **permanece registrado y visible**, pero sus modelos dejan de estar disponibles. Desactivar es reversible y no es borrar: es decir «este motor existe, ahora no lo uses».

La interfaz lo muestra sin ocultarlo, y el motivo importa —un motor apagado y uno borrado significan cosas distintas:

```text
Ollama local
Modelos desactivados
```

Una sesión que estuviera usando ese motor no muere: avisa de que su motor está desactivado y se queda esperando, con su historial intacto. Es un aviso, no un error.

### Reactivar

El motor vuelve a estar disponible junto con los modelos que tenía registrados. No hace falta volver a agregarlo ni a consultarlos: se conserva el catálogo.

### Eliminar

El motor se elimina **completo**: su configuración y sus modelos registrados. Es definitivo y por eso pide confirmación.

Eliminar un motor **no elimina ni invalida las sesiones que lo usaban**. Una sesión sobrevive con su historial; lo que pasa es que su motor ya no existe, así que avisa de que falta y espera a que el usuario elija otro. Borrar un motor no es borrar conversaciones.

## Modelos

Los modelos pertenecen al motor que los ofrece:

```text
Ollama local
└── qwen3:8b

llama.cpp router 8080
└── qwen3
```

Aunque se llamen igual, son configuraciones distintas porque son motores distintos. La interfaz los muestra siempre con el nombre del motor delante, para que sea imposible dudar de contra cuál se está hablando:

```text
Ollama local / qwen3:8b
llama.cpp router 8080 / qwen3
```

Al agregar un motor se consultan sus modelos disponibles y quedan registrados a su nombre.

**No se recuerda qué motor respondió cada turno.** Una sesión guarda el motor y el modelo que está usando, no el historial de los que fue usando: puede cambiar de motor varias veces y ninguna fila del chat lleva la marca de quién la escribió. Se decidió así porque el par actual ya está guardado y la trazabilidad por mensaje no aporta nada a la vista —la línea de estado y el rotulado del modal dicen contra qué motor se está hablando ahora—, mientras quecostaría una columna más en cada mensaje ya persistido.

## Motor y modelo por sesión

El motor y el modelo son **recursos que la sesión utiliza**, no parte de su identidad. Una sesión tiene un historial, un agente y un motor con su modelo, y ese motor puede cambiar.

```text
Sesión
│
├── Historial
├── Agente
└── Motor/modelo actual
       │
       ├── Ollama local / qwen3:8b
       │
       └── cambiar a
           llama.cpp router 8080 / qwen3
```

Una sesión puede cambiar de motor, cambiar de modelo, cambiar ambos a la vez, y **continuar la misma conversación** después del cambio. El identificador de la sesión no cambia, su nombre no cambia, su historial no cambia y ninguna referencia interna se entera.

Cada sesión guarda su par, así que el motor es una propiedad del trabajo, no de la ventana: al volver a una sesión se retoma contra el motor que tenía.

Esto no significa que dos sesiones hablen a la vez. Es para **trabajar por separado**: una sesión contra Ollama y otra contra llama.cpp, yendo de una a otra. Dos motores distintos pueden generar simultáneamente porque son procesos distintos; dentro de uno, las respuestas se atienden por orden de llegada.

### Cambiar a mitad de conversación

Cambiar de motor con el historial ya escrito no es un clic inocuo, y el harness lo trata como lo que es:

| Cambio | Qué hace el harness |
|---|---|
| La ventana efectiva del motor nuevo es menor | El presupuesto del historial se recalcula a la ventana nueva y el chat se recorta más. El mismo historial puede entrar y no entrar |
| El modelo nuevo no declara herramientas | El agente cae a **modo conversación** y la interfaz lo dice bajo la línea de entrada |
| La ventana no alcanza para un turno con herramientas | Avisa con la instrucción de levantarlo con más contexto y recorta el turno. Un turno que se recorta es corto; uno que muere no |
| El modelo nuevo no declara visión | Avisa sin bloquear si se adjunta una imagen |

Las imágenes de turnos anteriores **no** se arrastran al motor nuevo: son efímeras y no se persisten, así que un turno posterior que no vuelva a mencionar la imagen no la ve, tampoco después de cambiar de motor.

## Pedir herramientas al modelo

Las herramientas se piden por el canal `tool_calls` del núcleo común, no escribiéndolas en el mensaje. En cada petición al modelo viajan sus definiciones —nombre, descripción y esquema de argumentos— y el modelo responde pidiendo una por su nombre con los argumentos ya formados. Cuando pide varias, se ejecutan en el orden en que las pidió y sus resultados vuelven en la misma petición siguiente.

El sistema no le enseña al modelo a escribir una llamada: no tiene ningún formato que imitar. El resultado es que una llamada mal formada es un caso que no se da, y que los argumentos se pueden comprobar antes de ejecutar nada.

Esto tiene un coste que hay que asumir: **exige un modelo con capacidad de herramientas**. Se pregunta al motor antes de elegir y la respuesta se muestra en la interfaz.

### Cuando el modelo no tiene esa capacidad

Un modelo que no declara capacidad de herramientas **no puede usarlas**: ni las incluidas ni las que declare el usuario, porque no hay forma de pedírselas que no sea el canal nativo.

No se le impide elegirlo. La herramienta sigue funcionando, pero el agente activo cae a **modo conversación**: responde con el contexto que recibe y no ejecuta nada. La interfaz lo dice bajo la línea de entrada.

Un modelo así puede ser exactamente el adecuado para conversar, así que se avisa en vez de bloquear. Lo que no hace LocalCli es presentárselo como si pudiera trabajar.

### Turnos de varias pasadas

Un turno puede necesitar más de una petición al modelo: pide una herramienta, ve el resultado y vuelve a pedir. Entre pasada y pasada hay un momento en que el modelo no está generando porque el usuario está contestando una aprobación. Ese momento **no debe bloquear a otras sesiones**: cada petición toma el turno de inferencia de su motor y lo suelta al terminar, de modo que una sesión esperando una aprobación deja libre el modelo para las demás.

## La ventana de contexto

Un turno con herramientas —prompt del agente, esquemas del catálogo, historial y salidas de herramienta— necesita una ventana que no se recorta por el camino.

La ventana efectiva es la menor entre la que declara el modelo, la que declara el servidor y el tope (`LOCALCLI_CONTEXT_LIMIT`, 16384 por defecto). De dónde sale cada una depende de las extensiones declaradas:

| Extensión | Qué aporta |
|---|---|
| `num_ctx` | La ventana se **declara** en cada petición y el servidor la obedece |
| `props` | La ventana **se lee**: la fijó quien arrancó el servidor y no se cambia por petición |
| ninguna | La ventana se **desconoce** y manda el tope |

### El recorte es del harness, no del servidor

El harness **nunca confía en que el servidor recorte por su cuenta**. Aunque la extensión `num_ctx` esté declarada y el servidor obedezca, el recorte previo es el mismo: se hace en el harness, sobre los mensajes del turno, antes de enviar nada.

La razón es concreta y no hipotética: un servidor de Ollama al que no se le declara la ventana trabaja con la suya propia, que puede ser de 4096tokens. Con un prompt largo y una herramienta llamada, el servidor recorta la lista de mensajes por su cuenta y termina **`500 no user query found in messages`**: el turno muere con un error que no viene de la herramienta y que el manejo de errores no sabe colocar. Ese fallo deja de existir cuando el recorte lo hace quien conoce el presupuesto.

Así que el presupuesto del turno se aplica siempre:

1. Se calcula la ventana efectiva.
2. El presupuesto es una fracción de esa ventana, no la ventana entera.
3. Los mensajes se ordenan **de más nuevo a más viejo** y se van añadiendo mientras quepan.
4. El primero que no quepa corta la lista; los siguientes no se consideran.

Lo que se descarta es lo más viejo, porque un turno se entiende por su contexto reciente y porque las salidas de herramienta ya están resumidas en el historial.

Cuando la ventana efectiva es menor que la que un turno con herramientas necesita, **no es un fallo del modelo**: se avisa con la instrucción de levantarlo con más contexto y el turno se recorta para que quepa.

Cuando la ventana se **desconoce** —motor sin extensión que la declare— el tope manda y el aviso aparece siempre, porque trabajar sobre un tope con un servidor configurado para menos es la forma de convertir un recorte en un fallo. El aviso dice que el límite es del harness, no del servidor.

El harness **no muta el servidor**: no le cambia la ventana, no le descarga modelos y no le pide que cargue ni descargue ninguno. Ese estado es del usuario.

## Varios modelos a la vez

Cada sesión pide por su cuenta y **dentro de un mismo motor** el sistema las va sirviendo **por orden de llegada**. No es arbitrario y no es «la última que pidió gana»: la primera que llegó es la primera que se atiende.

La consecuencia de diseño que importa: una sesión que está esperando al usuario no está bloqueando a las demás. Si una herramienta pide aprobación y tardas medio minuto en contestar, otra sesión sigue generando con normalidad durante ese medio minuto.

Entre motores distintos no hay turno compartido: son procesos separados, y cada uno lleva su propia cola.

## Flujo principal

1. El harness carga el registro de motores. Si no hay registro, el registro queda vacío y se abre en el modal de motores para dar de alta el primero.
2. El usuario activa el motor que tenga instalado, o agrega otro con su dirección.
3. Se conecta a los motores activos y lee los modelos que declara cada uno.
4. Muestra la lista de todos ellos, rotulada `motor / modelo`, marcando cuáles caben en el hardware detectado y cuáles no, y cuáles saben usar herramientas —o mostrando `?` cuando el motor no lo declara.
5. **El usuario elige el motor y el modelo.** La herramienta no decide por él.
6. Si el modelo elegido no cabe, avisa y no lo carga en silencio.
7. Si el modelo elegido no sabe usar herramientas —porque no las declara o porque el motor no lo dice— avisa de que el agente conversará sin ellas, y sigue.
8. El par queda guardado en la sesión y se aplica a las sesiones nuevas por omisión.

## Flujos alternativos

- **No hay ningún motor registrado**: es el estado de partida de una instalación recién hecha. Se abre el modal de motores y el resto de la interfaz funciona; no se puede hablar con un modelo hasta que haya uno. No es un error y no se avisa como tal.
- **No hay ningún motor activo**: avisa y se puede trabajar con lo que hubiera, esperando.
- El motor no está corriendo: avisa y explica cómo levantarlo. El resto de motores siguen disponibles.
- El modelo elegido no cabe: lo dice y no lo carga.
- El modelo elegido no sabe usar herramientas, o el motor no declara si las sabe: el agente conversa sin ellas y la interfaz lo avisa.
- El servidor expone menos contexto del que necesita un turno con herramientas: avisa con la instrucción y recorta.
- El motor no declara la ventana: manda el tope, el aviso lo dice y el recorte lo hace el harness.
- La máquina tiene más recursos: se puede subir el tamaño de contexto.
- Se desactiva el motor de una sesión en marcha: la sesión avisa y espera, con su historial intacto.
- Se elimina el motor de una sesión en marcha: la sesión avisa de que falta y espera a que se elija otro.
- Se edita un motor: el cambio espera al reinicio y las sesiones siguen con la configuración actual.

## Reglas de negocio

- El modelo lo elige el usuario. La herramienta informa y avisa, no decide.
- **El motor es el runtime que ejecuta el modelo, no un servicio externo.** Ollama es un runtime que expone una API; llama.cpp es un motor de inferencia. Los dos sirven los mismos pesos, y por eso lo que se administra es el motor y no el proveedor.
- Hay **dos tipos de motor y el catálogo de tipos es cerrado**: `ollama` y `llamacpp`. Añadir un runtime es una tarea de código.
- **Todo motor habla el mismo núcleo OpenAI-compatible**: `/v1/chat/completions` y `/v1/models`. Contra el núcleo no cambia nada entre motores.
- **Cada motor declara sus extensiones nativas** (`num_ctx`, `show`, `tags`, `props`) y el harness solo las usa si están declaradas. Una extensión ausente deja lo que aportaba en **desconocido**, nunca en «no lo tiene».
- **Un motor sin extensiones funciona igual**: habla por el núcleo común y el harness declara desconocido lo que no sabe, en vez de suponerlo.
- **El harness no siembra motores ni presupone direcciones.** No hay proveedor por defecto: sin registro, el registro queda vacío. La instalación de LocalCli no instala runtimes de inferencia.
- **Ninguna `url` tiene valor por defecto**: hay que escribirla.
- **El registro se entrega escrito con las dos entradas y `activo: false`.** El usuario activa la que tenga instalada; el harness no elige ni activa por su cuenta.
- Se pueden registrar **varias instancias del mismo tipo**, cada una con nombre, dirección, extensiones y estado propios.
- Un modelo pertenece a un motor. Dos modelos con el mismo nombre en motores distintos son configuraciones distintas y nunca comparten ficha, ventana ni capacidades.
- El nombre visible de un motor es el de su instancia, no el del tipo, para que dos motores del mismo tipo se distingan en pantalla.
- Los motores se registran en `~/.config/localcli/motores.json`, fuera del proyecto: son un recurso de la máquina del usuario, no contenido del proyecto.
- **Un registro de motores ausente o ilegible no impide arrancar y no siembra nada**: el registro queda vacío y se abre el modal de motores.
- Los modelos de un motor se consultan al agregarlo y se refrescan cada vez que el harness conecta. Se conservan como último conocido para que la lista sobreviva a un motor desactivado o a un servidor caído.
- **Editar** un motor no afecta a las sesiones activas: se aplica al reiniciar el harness. **Agregar, desactivar, reactivar y eliminar** son inmediatos.
- Desactivar deja el motor registrado y visible, con sus modelos no disponibles. No es borrarlo, y la interfaz lo distingue.
- Una sesión cuyo motor se desactiva o se elimina no muere: avisa, conserva su historial y espera a que se elija otro.
- Eliminar un motor borra su configuración y sus modelos registrados. **No elimina ni invalida ninguna sesión.**
- El motor y el modelo son recursos de la sesión, no parte de su identidad: se pueden cambiar sin que cambie el identificador, el nombre ni el historial, y la conversación continúa.
- El par motor/modelo vive **en la sesión**. Permite trabajar por separado con un motor en una sesión y otro en otra; no significa que dos sesiones generen a la vez.
- Cambiar de motor a mitad de conversación recalcula el presupuesto del historial a la ventana efectiva del motor nuevo, que puede ser menor. El chat se recorta más si hace falta.
- Un modelo sin herramientas —o con las herramientas desconocidas— que se elige en una sesión con historial hace que el agente caiga a modo conversación, y la interfaz lo avisa.
- Las imágenes de turnos anteriores no se arrastran al cambiar de motor: son efímeras y no se persisten.
- **No se recuerda qué motor respondió cada turno.** El par actual de la sesión es lo que se guarda.
- `LOCALCLI_MOTOR` elige con qué motor arranca una sesión que no tiene par propio: el `id` de un motor, o `auto` para el primero activo que responde empezando por el primero del registro. Sin declararla se usa `ultimo_motor` y, en su defecto, `auto`. **Si no hay ningún motor activo, no hay motor de arranque**: no se elige ninguno y se abre el modal de motores.
- **El harness habla un solo protocolo con todos los motores** —el núcleo OpenAI-compatible— y las diferencias entre runtimes viven solo en el adaptador, como extensiones. Ninguna lógica fuera de los adaptadores consulta un endpoint.
- **El recorte del turno lo hace el harness, no el servidor.** El presupuesto es una fracción de la ventana efectiva y los mensajes se añaden de más nuevo a más viejo hasta que quepan; el primero que no quepa corta la lista.
- El perfil por defecto se calcula para 4 GB de VRAM y 16 GB de RAM, y es del harness: no es asunto de un adaptador.
- El tamaño de contexto se limita para que quepa junto con el modelo cargado.
- Si el modelo elegido no cabe, la herramienta avisa y no lo carga en silencio. **El encaje con el hardware solo se comprueba si el motor declara la extensión que da tamaño y familia**; sin ella no se afirma que quepa ni que no quepa.
- El par elegido se aplica a las sesiones nuevas por omisión y a la sesión en la que se elige.
- El nombre del motor se muestra en el modal de modelos y en la línea de modelo, para que se sepa contra qué runtime se está hablando.
- El último modelo, el último agente y el último motor usados se recuerdan entre ejecuciones como preferencia global del usuario (fuera del proyecto) y son el punto de partida de una sesión nueva. El motor recordado solo se reutiliza si sigue registrado y activo.
- La herramienta muestra qué modelos caben en el hardware detectado antes de que el usuario elija, cuando el motor lo permite saberlo.
- **Las capacidades tienen tres estados —soportada, no soportada y desconocida— y `false` no significa `desconocida`.** La ausencia de dato nunca se convierte en permiso.
- **Herramientas, visión y razonamiento obedecen la tríada**: `desconocida` desactiva las tres, y el razonamiento desactivado por falta de dato **avisa**. Las herramientas y las imágenes no se envían sin dato porque van en la petición; el razonamiento se apaga sin daño.
- Si el modelo en uso no declara capacidad de visión y el turno lleva imágenes, la interfaz avisa sin bloquear el envío. Con la capacidad **desconocida** avisa igual, porque no se afirma lo que no se sabe.
- Las herramientas viajan por el canal `tool_calls` del núcleo común contra los dos motores, sin formato en prosa que el modelo tenga que imitar.
- Las herramientas se ejecutan en el orden en que el modelo las pidió, nunca en paralelo.
- El turno de inferencia se toma por petición al modelo y se suelta al terminar, de manera que una sesión esperando una aprobación de herramienta no impide que otra sesión del mismo motor genere.
- La ventana, la ficha de capacidades y la que declara el modelo se cachean **por motor**: dos motores del mismo tipo pueden declarar cosas distintas del mismo nombre de modelo.
- El harness no cambia el estado del servidor del motor: ni la ventana, ni los modelos cargados o descargados.
- Si el texto de un mensaje de chat incluye la ruta de una imagen existente, la imagen se adjunta a ese turno hacia el modelo, en el formato de imágenes que declara el motor. Solo el adaptador conoce ese formato.
- Las imágenes adjuntas son efímeras: no se persisten ni se replican en el historial.
- **La degradación a modo conversación nunca es muda**: se avisa tanto si la capacidad se sabe ausente como si es desconocida.
- Bajo la línea de entrada se muestra el motor y el modelo en uso y si el modelo tiene acceso a herramientas y a la visión: `sí`, `no` o `?` cuando se desconoce.
- El **razonamiento** de un modelo que lo declara llega apagado: se le pide que no razone en cada petición. El usuario lo enciende por turno con el interruptor del pie (`pensar [x]`), que se pulsa con el ratón y se recuerda entre ejecuciones. A un modelo que no declara la capacidad no se le pide razonamiento —no entiende la opción— y sin ficha tampoco: no saberlo no es «no puede», es la razón de que el interruptor quede apagado y avise.
- Un servidor de llama.cpp que exponga lectura o escritura de archivos no es un motor admisible: se documenta que tiene que levantarse sin esas opciones.
- Todo el modelo y toda la conversación ocurren en la máquina local: no se envía nada fuera.
- La única excepción es la búsqueda en internet de [[specs/SPEC-TOOLS]], y solo sale la consulta, nunca contenido del proyecto.

## Criterios de aceptación

- [ ] **Sin registro de motores, el registro queda vacío, no se siembra ninguno y el arranque no falla.**
- [ ] **Ninguna dirección de motor se aplica por defecto: `url` vacía es un error de registro, no una dirección implícita.**
- [ ] El `motores.json` de ejemplo se entrega con las dos entradas escritas y **`activo: false`**, y activating la que corresponda la deja operativa.
- [ ] Un registro de motores ilegible no impide arrancar y no siembra nada.
- [ ] **Los dos motores viajan por el mismo núcleo OpenAI-compatible** (`/v1/chat/completions`, `/v1/models`), y ninguna diferencia de runtime aparece fuera de los adaptadores.
- [ ] Un motor al que se le declaran las cuatro extensiones y otro al que no se le declara ninguna funcionan ambos, y el segundo declara desconocido lo que no informa.
- [ ] Una extensión no declarada deja la información en **desconocido**, no en «no la tiene».
- [ ] Las capacidades se resuelven en **tres estados**: `soportada`, `no soportada` y `desconocida`, y `false` nunca significa `desconocida`.
- [ ] Con las herramientas **desconocidas**, el harness **no las envía** y la interfaz muestra `?`.
- [ ] Con el razonamiento **desconocido**, el razonamiento queda apagado y **la interfaz avisa** de por qué.
- [ ] Con la visión **desconocida**, la interfaz muestra `?` y avisa al adjuntar una imagen sin bloquear el envío.
- [ ] La degradación a modo conversación se avisa tanto si la capacidad se sabe ausente como si es desconocida.
- [ ] Se pueden registrar dos motores del mismo tipo, con nombre y dirección distintos, y ambos aparecen con su propio nombre.
- [ ] La lista de modelos muestra todos los motores activos y rotula cada fila como `motor / modelo`.
- [ ] Al agregar un motor se consultan sus modelos y quedan registrados a su nombre.
- [ ] Los modelos registrados se refrescan al conectar con el motor.
- [ ] Un motor desactivado permanece visible en la lista, con sus modelos no disponibles, y no se puede seleccionar.
- [ ] Un motor desactivado se puede reactivar y recupera sus modelos registrados sin volver a agregarlo.
- [ ] Editar la dirección de un motor no cambia el comportamiento hasta reiniciar el harness.
- [ ] Agregar, desactivar, reactivar y eliminar son inmediatos.
- [ ] Eliminar un motor pide confirmación.
- [ ] Eliminar un motor no elimina ni invalida las sesiones que lo usaban.
- [ ] Una sesión cuyo motor se elimina sigue existiendo, con su historial, y avisa de que su motor falta.
- [ ] Una sesión puede cambiar de motor, de modelo, o de ambos a la vez, y sigue con la misma conversación.
- [ ] Al cambiar el motor de una sesión no cambian su identificador, su nombre ni su historial.
- [ ] El par motor/modelo de una sesión se conserva al salir y volver.
- [ ] Al volver a una sesión se retoma contra el motor que tenía.
- [ ] Cambiar de motor recalcula el presupuesto del historial a la ventana efectiva del motor nuevo.
- [ ] Un modelo sin herramientas elegido en una sesión con historial hace caer al agente a modo conversación, y la interfaz lo avisa.
- [ ] Las imágenes de turnos anteriores no aparecen al cambiar de motor.
- [ ] El nombre del motor se ve en el modal de modelos y en la línea de modelo.
- [ ] No se recuerda qué motor respondió cada turno: los mensajes no llevan esa marca.
- [ ] `LOCALCLI_MOTOR` con el `id` de un motor arranca las sesiones nuevas contra él; con `auto`, contra el primero activo que responde.
- [ ] Si el motor recordado ya no está registrado, se autodetecta otro y el arranque no falla.
- [ ] La herramienta se conecta al motor y lista los modelos disponibles.
- [ ] Indica cuáles caben en el hardware detectado antes de que el usuario elija.
- [ ] El usuario elige el motor y el modelo; la herramienta no lo decide por él.
- [ ] Si el modelo elegido no cabe, avisa y no lo carga.
- [ ] El par elegido se aplica a la sesión en la que se elige y es el punto de partida de las nuevas.
- [ ] Un modelo que no declara capacidad de herramientas se marca y avisa sin bloquear.
- [ ] La interfaz dice bajo la entrada que el agente va a conversar sin herramientas cuando ese es el caso.
- [ ] Un turno que necesita varias herramientas hace una petición al modelo por cada paso, con los resultados de la anterior.
- [ ] Una sesión esperando una aprobación de herramienta no impide que otra del mismo motor genere.
- [ ] Dos sesiones del mismo motor que piden a la vez se atienden por orden de llegada.
- [ ] Dos motores distintos pueden generar a la vez, porque cada uno lleva su cola.
- [ ] Las herramientas viajan por `tool_calls` contra los dos motores, sin formato en prosa.
- [ ] Las imágenes de un turno de chat llegan al modelo en el formato de imágenes que declara el motor.
- [ ] Un modelo que declara razonamiento no razona si el interruptor del pie está apagado, y razona si el usuario lo enciende.
- [ ] Bajo la entrada se ve el motor y el modelo en uso, y si el modelo tiene acceso a herramientas y a la visión.
- [ ] Con un motor que declara más contexto que el tope, el turno usa el tope.
- [ ] **El recorte del turno lo hace el harness antes de enviar**, con los mensajes de más nuevo a más viejo, aunque la extensión `num_ctx` esté declarada.
- [ ] **Con un motor sin extensión de ventana, el tope manda, el aviso lo dice y el turno se recorta**: un turno largo no muere con un error del servidor.
- [ ] Con un servidor cuya ventana efectiva es menor que la que necesita un turno con herramientas, avisa con la instrucción de levantarlo con más contexto y el turno se recorta en vez de morir.
- [ ] El harness no cambia la ventana, ni los modelos cargados o descargados, del servidor.
- [ ] Dos motores del mismo tipo no comparten ficha de capacidades ni ventana, aunque sirvan un modelo del mismo nombre.
- [ ] Ninguna lógica fuera de los adaptadores decide por tipo de motor.
- [ ] La herramienta sigue funcionando con 16 GB de RAM sin agotar la memoria.
- [ ] La única información que sale de la máquina es la consulta de una búsqueda, nunca contenido del proyecto.

## Requisitos no funcionales

- Consumo de VRAM dentro de 4 GB en la configuración por defecto.
- No añadir esperas propias al tiempo de respuesta del modelo.
- Una sesión bloqueada esperando al usuario no retiene el turno de inferencia de su motor.
- Añadir un motor no añade dependencias de terceros: el transporte es el cliente HTTP de la biblioteca estándar, y el del núcleo común lo comparte **todos** los motores.
- El harness no muta el estado del servidor del motor: es de la persona que lo levanta.
- Cambiar de motor no reconstruye el contexto del turno: solo recalcula el presupuesto con la ventana del motor nuevo.
- El recorte del turno no añade una ida y vuelta al modelo: es una operación local sobre la lista de mensajes, sin coste de red.

## Dependencias funcionales

- [[specs/SPEC-SESIONES]]
- [[specs/SPEC-NODO-CONTEXTO]]
- [[specs/SPEC-PANEL-CONTEXTO]]
- [[PROJECT]]

## Supuestos

- Los valores concretos del perfil (modelo, tamaño de contexto, parámetros de inferencia) se fijan en FASE 2 y FASE 3.
- Los dos motores sirven los mismos pesos: lo que cambia es el runtime y sus extensiones, no el modelo.
- Dos motores distintos pueden atender a la vez porque son procesos separados; no se asume que haya GPU para los dos.
- **Los dos runtimes de referencia implementan el núcleo OpenAI-compatible**: Ollama lo expone en `/v1` desde hace tiempo y `llama-server` lo expone en `/v1`. Es el supuesto sobre el que se apoya toda la arquitectura.
- LocalCli se puede instalar en una máquina sin ningún runtime de inferencia; por eso nada se presupone instalado.

## Referencias

- [[IDEA]]