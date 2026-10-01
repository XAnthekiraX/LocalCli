---
title: LocalCli — catálogo de herramientas
tags: [backend, interfaces]
depende_de:
  - "[[specs/SPEC-TOOLS]]"
  - "[[backend/DECISIONS]]"
relacionado:
  - "[[backend/02-interfaces/dto/TOOLS-DTO]]"
  - "[[backend/01-domain/DOMAIN]]"
  - "[[backend/02-interfaces/INTERFACES-GENERAL]]"
  - "[[backend/03-security/SECURITY]]"
  - "[[backend/05-quality/ERRORS]]"
  - "[[backend/05-quality/VALIDATION]]"
  - "[[backend/04-infrastructure/EVENTS]]"
  - "[[specs/SPEC-ARCHIVOS]]"
  - "[[specs/SPEC-MODELO-PROVEEDOR]]"
---
# TOOLS — Catálogo de herramientas

Las quince herramientas incluidas, su reparto y sus controles, la capa universal que envuelve toda ejecución, y el punto de extensión para las herramientas del usuario. La especificación funcional está en [[specs/SPEC-TOOLS]]; aquí está el contrato para quien implemente. Los payloads están en [[backend/02-interfaces/dto/TOOLS-DTO]].

## 1. El catálogo incluido

Cerrado. El agente no puede inventar herramientas fuera de esta lista.

### Archivos

| Herramienta | Lee o escribe | Permiso | Agente | Contrato |
|---|---|---|---|---|
| `leer_archivo` | Lee | `read` | Ambos | Ruta relativa al proyecto; devuelve el contenido como texto |
| `listar_carpeta` | Lee | `read` | Ambos | Ruta relativa; devuelve la lista de entradas de un nivel |
| `buscar_archivos` | Lee | `read` | Ambos | Patrón; devuelve las rutas que coinciden |
| `buscar_en_archivos` | Lee | `read` | Ambos | Patrón; devuelve las coincidencias con archivo y línea |
| `crear_archivo` | Escribe | `write` | Solo `build` | Ruta y contenido; falla si el archivo ya existe |
| `escribir_archivo` | Escribe | `write` | Solo `build` | Ruta y contenido; sobrescribe |
| `editar_archivo` | Escribe | `edit` | Solo `build` | Ruta y el cambio; aplica una edición parcial |
| `eliminar_archivo` | Escribe | `write` | Solo `build` | Ruta; pide confirmación explícita |
| `crear_carpeta` | Escribe | `write` | Solo `build` | Ruta; falla si ya existe |
| `eliminar_carpeta` | Escribe | `write` | Solo `build` | Ruta; pide confirmación explícita |

`editar_archivo` es la única herramienta de edición parcial, así que es la única que cae en `edit`; el resto de las de escritura van en `write`. Los tres permisos son independientes: un agente con `write: deny` y `edit: allow` puede parchear archivos pero no crearlos, sobrescribirlos ni borrarlos.

### Terminal

| Herramienta | Lee o escribe | Permiso | Agente | Contrato |
|---|---|---|---|---|
| `ejecutar_comando` | Lee | `read` | Ambos | Comando y carpeta de trabajo; devuelve salida, error y si terminó |

### Internet

| Herramienta | Lee o escribe | Permiso | Agente | Contrato |
|---|---|---|---|---|
| `buscar_en_internet` | Lee | `read` | Ambos | Consulta; devuelve título, dirección y fragmento de cada resultado |
| `abrir_pagina` | Lee | `read` | Ambos | Dirección; devuelve el contenido de la página |

### Sesión

| Herramienta | Lee o escribe | Permiso | Agente | Contrato |
|---|---|---|---|---|
| `crear_todo` | Sesión | `read` | Ambos | Un paso —contenido, estado y prioridad opcional—; lo añade al final y devuelve la lista resultante |
| `actualizar_todo` | Sesión | `read` | Ambos | La lista de pasos de la sesión, completa; reemplaza la anterior y devuelve el checklist |

`crear_todo` y `actualizar_todo` escriben estado de la sesión, no archivos del proyecto: por eso caen en el permiso `read` y la tienen los dos agentes, sin tocar la garantía de escritura de §2.

Una **añade** y la otra **sustituye**, y la diferencia es el contrato, no una preferencia. `crear_todo` no recibe los pasos anteriores porque no los toca: no hay forma de que añadir uno borre los que ya estaban. `actualizar_todo` recibe la lista entera y la sustituye, que es lo que hace falta para reordenar, cancelar un paso o dejar la lista vacía. Las dos devuelven la lista resultante, para que el modelo tenga el estado sin preguntar.

Ninguna de las dos recibe un identificador de paso: el orden es la posición y un identificador solo daría material para que un modelo cite mal.

`crear_archivo` y `escribir_archivo` están separadas a propósito: el agente no destruye algo por accidente cuando pretendía crear.

### Lo que se ve en la TUI

Cada herramienta declara, además de su contrato, tres datos de presentación que la TUI usa en la línea del chat ([[frontend/02-interfaces/INTERFACES]] §1.1): su **verbo** (la etiqueta corta que la nombra), su **tema** (el campo de la petición cuyo valor es el objetivo que se enseña) y su **unidad** (cómo se mide el resultado). No viajan al modelo: son presentación.

| Herramienta | Verbo | Tema | Unidad |
|---|---|---|---|
| `leer_archivo` | `LEER` | `ruta` | línea |
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
| `crear_todo` | `TODO` | `contenido` | paso |
| `actualizar_todo` | `TODO` | — | paso |

Solo se muestra el **tema**: el resto de argumentos —el contenido de un archivo, un cambio— no se expone, porque la línea va a pantalla y a auditoría. Una herramienta del usuario no declara estos datos: cae en su propio nombre como verbo y no tiene tema ni unidad.

## 2. El reparto: `plan` mira, `build` escribe

El reparto sale de los `permissions` del agente ([[specs/SPEC-AGENTE-BASE]]), no de una lista de herramientas declarada. Cada herramienta pertenece a un **permiso** fijo, el de la columna de las tablas de §1, y no hay forma de declararlo de otro modo: `leer_archivo`, `listar_carpeta`, `buscar_archivos`, `buscar_en_archivos`, `ejecutar_comando`, `buscar_en_internet`, `abrir_pagina`, `crear_todo` y `actualizar_todo` son `read`; `crear_archivo`, `escribir_archivo`, `eliminar_archivo`, `crear_carpeta` y `eliminar_carpeta` son `write`; `editar_archivo` es `edit`.

- `plan` permite solo `read` y deniega `write` y `edit`. No tiene ninguna herramienta que escriba en el proyecto.
- `build` permite los tres y recibe el catálogo completo. Es el único que crea, modifica y borra archivos.

El catálogo efectivo de cada agente se deriva de sus permisos contra el catálogo. No hay dos listas que puedan contradecirse.

**El reparto se aplica también a las herramientas del usuario** (§9). Una declarada con `modo: lee` entra como lectura y nunca llega a `plan`; una declarada con `modo: escribe` no entra en absoluto, porque se rechaza. Extender el catálogo no abre una puerta trasera a la garantía.

## 3. El relevo entre agentes

```
plan lee  →  plan propone  →  tú apruebas  →  cambias a build  →  build aplica
```

1. `plan` investiga con herramientas de lectura.
2. `plan` propone el cambio concreto.
3. Tú lo apruebas.
4. Cambias a `build`.
5. `build` aplica exactamente lo aprobado.

**La aprobación vale para el cambio propuesto, no para lo que siga.** Si tras el relevo `build` necesita hacer algo que `plan` no propuso, vuelve a preguntar. Una aprobación no es un permiso general.

## 4. Herramientas de archivo

Siguen [[specs/SPEC-ARCHIVOS]]: dentro de la carpeta del proyecto son accesibles, fuera se pide permiso y el agente explica por qué, y **toda escritura pasa por aprobación**, incluso dentro de un flujo en curso. El borrado exige confirmación explícita, no solo aprobación genérica. Cada ruta es relativa a la carpeta abierta. Ver [[backend/02-interfaces/dto/TOOLS-DTO]] para las reglas de validación de rutas.

## 5. Herramientas de terminal

Dos controles independientes.

### Control 1 — lista blanca

Se ejecutan sin preguntar, porque son los que se repiten en cada iteración:

- Compilar el proyecto.
- Correr las pruebas.
- Revisar estilo y tipos.
- Ver el estado y las diferencias del repositorio: estado, diferencias, historial.

Cualquier otro comando pide aprobación antes de ejecutarse.

### Control 2 — no toca tus archivos

El bloqueo **no depende de revisar el texto del comando**. La terminal no puede crear, editar ni borrar ningún archivo del usuario, por más indirecto que sea: `find -delete`, `tee`, una tubería hacia un archivo, `python3 -c` o un `sh -c` con redirección se colarían por un filtro de texto, y por eso el bloqueo es estructural, con Landlock.

Lo que sí puede hacer: usar su propio espacio (caché de compilación, temporales, carpeta de trabajo por sesión) y los dispositivos nulos (`/dev/null`, `/dev/zero`, `/dev/full`), y correr los comandos de la lista blanca.

Lo que no puede hacer, aunque se le pida: crear, editar o borrar archivos del proyecto (para eso están las herramientas de escritura, que solo tiene `build` y que pasan por aprobación), descartar cambios del repositorio, hacer commits o subir cambios.

### Otras reglas de la terminal

- El comando corre en la carpeta del proyecto.
- La salida está limitada: un comando que no termina o genera salida sin fin se corta y se avisa, sin colgar el harness ni llenar el contexto.
- El resultado indica si el comando terminó bien o mal, con la salida de error cuando la hay.
- Un comando que falla no detiene el trabajo: el agente ve el error y sigue.

## 6. Herramientas de internet

Son las únicas que hacen salir información de la máquina.

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
    Categoria   Categoria     // archivos | terminal | internet | tareas | usuario
    Permiso     Permiso       // read | write | edit: lo que hay que conceder para usarla
    Modo        Modo          // lee | escribe
    Esquema     *Esquema      // se deriva del tipo de petición
    Ejecutar    Ejecutar      // handler propio; nil significa "sin implementar"
}
```

`Permiso` es lo que el agente concede: el reparto se decide comparando el `Permiso` de la herramienta contra los `permissions` del agente, y por eso es un dato del catálogo, no un `switch` en el enrutado. `Categoria` y `Modo` no deciden nada de eso: sirven para agrupar y para la interfaz, y `Modo` es lo que distingue una herramienta del usuario de una incluida. Los dos vivos conviven a propósito; lo que se añade es el permiso, no se sustituye el modo.

Tres cosas de este contrato que no existían antes:

- **`Ejecutar` es un handler propio**, no una categoría. Una herramienta nueva no necesita que nadie modifique un `switch`. El enrutado actual (`Destinos`, tres categorías fijas) desaparece.
- **`Ask` vive en el contexto**, no dentro del handler. La aprobación deja de estar acoplada a `fileops` y `exec`: cualquier herramienta —incluida una del usuario— la pide por el mismo camino, y el mecanismo queda en un solo sitio.
- **`Error` es un campo de `Resultado`, no un `error` de Go.** La distinción importa: un `error` es un fallo del harness y termina el turno; un `Error` populated es un resultado que el modelo puede leer y corregir.

### El esquema se deriva, no se escribe

`Esquema` se genera por reflexión sobre el tipo de petición de cada herramienta. Cada campo lleva su descripción en un tag `desc`:

```go
type PeticionEditarArchivo struct {
    Ruta        string  `json:"ruta"        desc:"Ruta relativa a la carpeta del proyecto"`
    Reemplazos  []Cambio `json:"reemplazos" desc:"Cambios a aplicar, en orden"`
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

La medición va aquí, y no dentro de cada handler, porque **esta es la única función por la que pasa toda ejecución**: la de un catálogo nativo y la de una herramienta del usuario, por igual. Medir en la herramienta obligaría a que cada una llevara su propio reloj, y bastaría con que una lo olvidara para que su línea saliera sin tiempo.

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

## 9. Herramientas del usuario

Se declaran en `.localcli/tools/*.json` y se cargan al arrancar. **No son código**: son declaraciones que LocalCli ejecuta.

```json
{
  "nombre": "contar_lineas",
  "descripcion": "Cuenta las líneas de un archivo de texto.",
  "modo": "lee",
  "equipo": ["wc", "-l"]
}
```

| Campo | Regla |
|---|---|
| `nombre` | Requerido. No puede coincidir con una incluida. |
| `descripcion` | Requerido. Es lo que lee el modelo. |
| `modo` | `lee`. `escribe` **se rechaza**, ver más abajo. |
| `equipo` | Requerido. Lista de argumentos. Nunca se pasa por un shell. |
| `timeout_segundos` | Opcional. Si falta, se usa el límite de la terminal. |

### Ejecución

El handler de una herramienta del usuario es un adaptador fino sobre `exec.Ejecutor`. No reimplementa la terminal: la usa. Por eso hereda, sin código nuevo:

- El aislamiento de Landlock: no puede escribir en el proyecto.
- El límite de tiempo, propio o el de la terminal.
- El recorte de la salida antes de volver al modelo.
- Los mismos eventos que cualquier otra herramienta.

**Los argumentos no viajan en la línea de comando.** El modelo manda un objeto JSON libre, porque el contrato de la herramienta del usuario es un objeto genérico y no tiene esquema con campos conocidos. El adaptador **inyecta** ese objeto en la entrada estándar del subproceso, como un único JSON, y no lo añade a `equipo`. Es lo que permite que `equipo` siga siendo fijo y verificable: los argumentos del modelo no pueden inyectar un programa o un argumento nuevo, porque no llegan a la lista. La salida se lee de la salida estándar y de la de error, y se recorta como la de cualquier herramienta. Ver [[specs/SPEC-TOOLS]] y [[backend/05-quality/VALIDATION]].

Y por encima, dos reglas que no se negocian:

- **Siempre pide aprobación.** El comando lo escribió el usuario, no la herramienta, así que no puede estar en la lista blanca. No hay excepción, ni siquiera si el ejecutable es `go test`.
- **Nunca entra en la lista blanca.** Ni por nombre, ni por prefijo, ni por parecido.

`equipo` es una lista, no una cadena, y no se concatena en un shell. No hay tubería, ni redirección, ni `sh -c`. Un argumento con un `;` es un argumento con un punto y coma. Esto es lo que hace que la garantía de la terminal siga valiendo para estas herramientas: la garantía de Landlock es estructural, y un argumento que no pasa por un shell no puede esquivarla.

### Qué se rechaza

- **`"modo": "escribe"` se rechaza al arrancar**, con el motivo: la única vía sancionada para escribir son las herramientas de archivo, que solo tiene `build` y que sí pasan por aprobación. Aceptarlo sería una segunda puerta de escritura que no respeta la garantía.
- **Un nombre que choca con una incluida** no se carga y se avisa.
- **Un JSON inválido o un campo obligatorio ausente** no se carga y se avisa. El arranque continúa, igual que con un agente o un flujo inválido.

### Aislamiento

Una herramienta del usuario no puede llamar a otras herramientas: su handler no tiene acceso al registro. No puede ampliar el catálogo, encadenar herramientas ni elevar sus permisos.

**No hay namespacing.** El nombre que declara es el nombre que ve el modelo, sin prefijo ni carpeta que lo califique. La consecuencia es una regla de colisión, no una convención: un nombre que coincida con el de una incluida —o con el de otra declarada— no se carga y se avisa, porque dos herramientas con el mismo nombre harían que la elección del modelo fuera ambigua. Ver [[backend/05-quality/VALIDATION]].

## 10. Hooks

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

## 11. El cable: cómo llegan al modelo

`tools` no habla con ningún proveedor. `agent` pide los esquemas y los pone en la petición; la capa de proveedor (`llm`) los entrega al adaptador, que los serializa en el formato de su runtime.

1. `agent` pide a `tools` el **esquema** de las herramientas del agente activo.
2. `agent` construye una petición neutra (`llm.Peticion`) con esa lista en el campo de herramientas.
3. La frontera `llm.Proveedor` la pasa al adaptador del proveedor elegido: `ollama` la serializa para `/api/chat` y `openai` para `/v1/chat/completions`.
4. El modelo responde pidiendo una por su nombre con los argumentos ya formados.
5. El adaptador entrega las peticiones acumuladas; `agent` itera y llama a `Registro.Ejecutar`.
6. El resultado vuelve al modelo como un mensaje propio de herramienta, con el formato que su proveedor espera.

El catálogo **no** viaja en el mensaje de sistema. `PromptDeSistema` se queda con el prompt del agente.

El formato exacto de cada proveedor está en [[specs/SPEC-MODELO-PROVEEDOR]] y en [[backend/04-infrastructure/INTEGRATIONS]]; el de los mensajes de herramienta que no se persisten, en [[database/01-schema/ENUMS]].

## Referencias

- [[specs/SPEC-TOOLS]] — la especificación funcional y los criterios de aceptación.
- [[specs/SPEC-ARCHIVOS]] — reglas de permiso sobre archivos.
- [[specs/SPEC-MODELO-PROVEEDOR]] — el canal de herramientas y la capacidad del modelo.
- [[backend/02-interfaces/INTERFACES-GENERAL]] — las superficies.
- [[backend/02-interfaces/dto/TOOLS-DTO]] — los payloads de cada herramienta y sus esquemas.
- [[backend/03-security/SECURITY]] — la garantía de escritura, el bloqueo de la terminal y el aislamiento de las herramientas del usuario.
- [[backend/04-infrastructure/EVENTS]] — `herramienta_invocada` y `herramienta_resultado`.
- [[backend/05-quality/VALIDATION]] — validación de rutas y argumentos, y el error autocorregible.
- [[backend/05-quality/ERRORS]] — errores que puede producir cada herramienta.
- [[backend/DECISIONS]] — decisiones de contrato que sostienen este módulo.
