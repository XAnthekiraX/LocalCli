---
title: LocalCli — seguridad y permisos
tags: [backend, seguridad]
depende_de:
  - "[[backend/DECISIONS]]"
  - "[[specs/SPEC-ARCHIVOS]]"
  - "[[specs/SPEC-TOOLS]]"
relacionado:
  - "[[specs/SPEC-MODELO-PROVEEDOR]]"
  - "[[backend/01-domain/DOMAIN]]"
  - "[[backend/02-interfaces/TOOLS]]"
  - "[[backend/04-infrastructure/CONFIGURATION]]"
  - "[[backend/04-infrastructure/INTEGRATIONS]]"
---
# SECURITY — Seguridad y permisos

`LocalCli` es local y personal: no hay login, ni usuarios, ni sesiones de autenticación, ni endpoints que proteger de outsiders. La seguridad aquí es de otro tipo: se trata de que **nada de lo que hace el modelo toque tu proyecto ni tu máquina sin que tú lo hayas aprobado**, y de que un proceso hijo no pueda saltarse esa regla.

## 1. Autenticación

No hay autenticación, y no es una omisión. El proceso se abre en tu terminal, en tu máquina, con tus permisos. No hay red, así que no hay a quién autenticarse.

Lo que sí hay es **identidad de agente**: cada petición al motor viene de un agente (`plan` o `build`) y ese agente tiene un conjunto fijo de herramientas. La "autenticación" del modelo es saber qué agente es, no quién es el usuario. Ver [[backend/02-interfaces/TOOLS]].

Con el canal nativo de herramientas hay una pieza más de identidad que antes no hacía falta: **cada llamada va firmada con el agente que la pidió**. El nombre del agente no lo elige el modelo, sino quien construye la petición; el modelo solo elige la herramienta y sus argumentos. Por eso un modelo no puede pedir en nombre de otro agente: la capa de ejecución comprueba la firma contra el agente activo y, si no coincide, la deniega. No es criptografía —`tools.Peticion.Agente` es un campo de la petición en memoria y el modelo no lo controla—, sino atribución dentro del proceso.

## 2. Autorización

No hay roles de usuario. La autorización se resuelve en tres capas, y por eso es auditable: puedes leer en un solo sitio quién pidió, quién autorizó y quién ejecutó.

| Capa | Módulo | Qué decide |
|---|---|---|
| Qué puede pedir el agente | `agent` | Qué herramientas tiene `plan` (las del permiso `read`) y `build` (los tres permisos) |
| Si la petición está permitida | `tools` | Que la herramienta exista y que el agente la tenga |
| Si el efecto se aplica | `fileops`, `exec` | La aprobación concreta, la frontera de rutas, la lista blanca y Landlock |

El paso 2 ocurre en **la capa universal de `tools`**, antes de que se ejecute nada y antes de que se toque el disco. Es el mismo punto para las quince herramientas incluidas y para las del usuario: no hay un camino alternativo que se salte la comprobación.

La garantía central: **`plan` no tiene herramientas que escriban en el proyecto**. No es que las tenga bloqueadas, es que no existen para él, porque su `agent.yaml` concede solo `read` y todo lo demás cae en el `default: deny`. La lista de pasos de la sesión también es `read` —es estado de la sesión, no del proyecto—, así que la tienen los dos agentes sin debilitar la garantía. Ver [[specs/SPEC-AGENTE-BASE]] y [[backend/02-interfaces/TOOLS]] §2.

**Una aprobación vale para el cambio propuesto, no para lo que siga.** Si `build` necesita algo que `plan` no propuso, vuelve a preguntar.

**La aprobación es un mecanismo único.** Vive en `Contexto.Ask`, no dentro de los handlers. Eso significa que toda herramienta —incluida una escrita por el usuario— pasa por el mismo camino, y que no puede existir una herramienta cuya aprobación se comporte de otra manera. Una política de permiso repartida en quince sitios es una política que en algún sitio se olvidó.

## 3. Protección de las operaciones

No hay endpoints. Lo que hay son operaciones, y cada una tiene su regla.

### Toda escritura pasa por aprobación

Sin excepción: ni dentro de la carpeta del proyecto, ni fuera, ni dentro de un flujo en curso. Ver [[specs/SPEC-ARCHIVOS]].

### Frontera de rutas

- Toda ruta relativa es relativa a la carpeta abierta del proyecto; una absoluta se respeta y, si no cuelga de él, cuenta como «fuera».
- Dentro de la carpeta: accesible, pero escribir sigue pidiendo aprobación.
- Fuera de la carpeta: hace falta permiso **y** la explicación del agente sobre por qué busca eso. La explicación queda visible para ti.

### Terminal bloqueada estructuralmente

La terminal tiene dos controles independientes:

1. **Lista blanca:** solo compilar, probar, revisar estilo y tipos, y ver estado/diferencias/historial de git corren sin preguntar. Cualquier otro comando pide aprobación.
2. **Bloqueo de escritura:** la terminal no puede crear, editar ni borrar archivos del proyecto, y la garantía es del kernel (Landlock), no del texto del comando. Por eso `find -delete`, `tee`, una tubería hacia un archivo, `python3 -c` o `sh -c` con redirección no la esquivan.

Lo único con escritura concedida es el espacio propio del comando —temporales, caché de compilación y carpeta de trabajo por sesión— y los **dispositivos nulos** (`/dev/null`, `/dev/zero`, `/dev/full`): programas de la lista blanca como `git` abren `/dev/null` con `O_RDWR` aunque no guarden nada, y sin ese permiso `git status`, `git diff` y `git log` fallaban. La regla de Landlock se ancla en cada dispositivo, no en `/dev`, así que el resto del sistema sigue denegado.

La terminal tampoco puede descartar cambios del repositorio, hacer commits ni subir cambios.

## 3.1 Herramientas del usuario

Un `.json` en `.localcli/tools/` añade una herramienta. Eso abre una superficie que antes no existía —el usuario puede hacer que LocalCli ejecute lo que quiera— y por eso tiene reglas propias, todas en [[specs/SPEC-TOOLS]] y [[backend/02-interfaces/TOOLS]].

**No es código, es una declaración.** Un `.json` no puede ejecutar nada por sí mismo: LocalCli lo lee y lo ejecuta con su propio ejecutor. No hay intérprete, ni runtime, ni plugin compilado. La diferencia no es cosmética: no hay nada en el formato que permita ejecutar una función.

**El aislamiento es el de la terminal, no uno nuevo.** La herramienta corre sobre `exec.Ejecutor`, así que hereda Landlock, el límite de tiempo y la cola. La garantía de que no puede escribir en el proyecto no hay que volver a pensarla ni volver a probarla: es la misma.

**No hay shell.** `equipo` es una lista de argumentos que se pasa al ejecutable tal cual. Sin intérprete de shell no existen las tuberías, ni las redirecciones, ni `sh -c`. Un argumento con un `;` es un argumento con un punto y coma. Esto es lo que hace que la garantía estructural de Landlock se sostenga también para estas herramientas: un argumento que no pasa por un shell no puede convertir un argumento en una escritura.

**Siempre pide aprobación, y no se toca la lista blanca.** El ejecutable lo eligió el usuario, así que LocalCli no sabe qué hace aunque se llame como un comando que ya conoce. Que la lista blanca sea solo para los comandos que el harness conoce es precisamente lo que hace predecible el control: si dependiera del nombre, un binario con el mismo nombre y otro comportamiento se ejecutaría sin preguntar.

**`modo: escribe` se rechaza al arrancar.** La única vía sancionada para escribir en el proyecto son las herramientas de archivo: solo las tiene `build`, pasan por aprobación y quedan registradas en `change_history`. Aceptar una segunda vía de escritura con reglas distintas rompería la garantía central por la puerta de atrás.

**No puede escalar.** El handler de una herramienta del usuario no tiene acceso al registro, así que no puede pedir otras herramientas, no puede encadenar y no puede elevarse de permisos. No hay una ruta de una herramienta del usuario a una capacidad que no tuviera.

**Un JSON inválido no detiene nada.** Se ignora y se avisa, como con un agente o un flujo inválido. Un error de carga deja el catálogo como estaba, no lo deja a medias.

**El coste de aceptar esto.** El esquema que ve el modelo para una herramienta del usuario es genérico —un objeto, sin properties— porque el harness no puede saber qué argumentos espera un ejecutable que no es suyo. En la práctica eso significa que el modelo improvisa los argumentos de una herramienta del usuario, y que un error de nombre de argumento se descubre al ejecutar, no antes. Es un precio asumido a cambio de que añadir una herramienta no requiera recompilar; queda dicho aquí y en [[backend/02-interfaces/dto/TOOLS-DTO]] para que no sorprenda.

## 3.2 El servidor del proveedor de modelo

El harness trata al proveedor como una caja de la que solo consume inferencia; el estado del servidor —qué modelos tiene, con qué ventana— es de quien lo levanta. Eso fija dos requisitos:

- **`llama-server` tiene que levantarse sin las banderas que sirven el sistema de archivos por HTTP.** Con `--tools`, `--agent` o `--mcp-servers-json` activos, el servidor expone lectura o escritura de archivos por su propia API, al margen de la frontera de rutas y de la aprobación del harness. Un servidor así **no es un proveedor admisible**: la garantía de que nada toca el proyecto sin aprobación no se sostiene. Se documenta la instrucción de levantarlo sin esas banderas. Ver [[backend/04-infrastructure/INTEGRATIONS]].
- **No hay descarga automática de modelos.** El harness no llama a `POST /models` (descarga), ni a `POST /models/load` (carga), ni a `POST /props` (cambiar la ventana): no pide pesos a la red ni muta el servidor. Si el modelo no está, lo dice; no lo trae.

Que `llama-server` acepte `--api-key` no cambia la frontera: el servidor es de loopback y LocalCli no le pone clave, porque al que se le da una clave es al que hay que enseñarla. Ver [[backend/04-infrastructure/CONFIGURATION]].

## 4. Frontera de internet

Las dos herramientas de internet son las únicas que hacen salir información de la máquina, y ahí la regla es estrecha:

- **Solo sale la consulta que redacta el modelo.** Nunca el contenido de tus archivos, de la documentación, del historial ni de tus tareas.
- **Si una búsqueda necesita el proyecto para ser útil, el agente te lo pide a ti** en lugar de mandarlo.
- **Lo que vuelve entra al mismo presupuesto de contexto** que el resto y queda registrado en la auditoría, sujeto al mismo recorte. Ningún agente vierte el contenido de una página sin procesarlo. Ver [[specs/SPEC-TOOLS]].

## 5. Sanitización y datos sensibles

- El modelo solo recibe lo que el nodo de contexto decide. El contenido de los archivos que se entregan se lee antes, y lo que sale se registra. Ver [[backend/01-domain/DOMAIN]].
- No se envían datos personales del proyecto a internet por la vía de las herramientas de búsqueda.
- `change_history` guarda el antes y el después de cada cambio aplicado, lo que incluye contenido de tus archivos. Es un registro local en tu máquina, y existe para que puedas revertir y auditar, no para compartir.

## 6. Riesgos relevantes

- **Rate limiting:** no aplica. No hay red pública. Lo equivalente es el presupuesto de contexto, que recorta la entrada y evita que una respuesta o un comando desborden. Ver [[specs/SPEC-MODELO-PROVEEDOR]].
- **CORS:** no aplica. No hay navegador ni servidor.
- **Secrets:** los dos proveedores de modelo corren en local, sin credenciales (el `--api-key` opcional de `llama-server` no se usa). La búsqueda por internet es la única salida y solo lleva la consulta. LocalCli no guarda ni pide claves de API. Una herramienta del usuario puede llevar sus propias credenciales en sus argumentos, que viajan al modelo y al registro de auditoría: quien declare una sabe que el argumento queda visible ahí.
- **Aislamiento en sistemas sin Landlock:** en sistemas que no son Linux, o en Linux sin Landlock, la terminal no tiene el bloqueo estructural. La garantía es más débil y queda documentada como tal; el proyecto lo asume. Ver [[backend/04-infrastructure/CONFIGURATION]]. **Las herramientas del usuario heredan esta limitación tal cual**: sin Landlock no tienen el bloqueo de escritura, y su única defensa restante es la aprobación, que es una decisión del usuario y no una garantía.
- **El servidor del proveedor con acceso al sistema de archivos.** Un `llama-server` levantado con `--tools`, `--agent` o `--mcp-servers-json` sirve lectura o escritura de archivos por HTTP, fuera de la frontera de rutas y de la aprobación del harness; por eso no es un proveedor admisible. Ver §3.2.
- **El modelo no es de fiar para decidir permisos.** `plan` y `build` tienen herramientas fijas, y el permiso lo comprueban módulos, no el modelo. El modelo no puede concederse permisos.
- **Confiar en el texto del comando es un error.** Cualquier intento de validar la terminal leyendo el comando es frágil por diseño; por eso el bloqueo es estructural. Y por eso las herramientas del usuario pasan una lista de argumentos en vez de una línea de texto: quitar el shell es quitar esa clase de ataque entera, no cerrar sus casos sueltos.
- **Un error de herramienta no termina el turno.** El fallo se le devuelve al modelo para que lo corrija. Eso es intencionado, y es también un riesgo: un modelo puede insistir en una herramienta que falla. Por eso el máximo de pasadas por turno es un límite duro, no un detalle de implementación.

## Referencias

- [[specs/SPEC-ARCHIVOS]] — reglas de permiso sobre archivos.
- [[specs/SPEC-TOOLS]] — el catálogo, la terminal y las herramientas del usuario.
- [[backend/02-interfaces/TOOLS]] — el reparto `plan`/`build`, la capa universal y el punto de extensión.
- [[backend/02-interfaces/dto/TOOLS-DTO]] — los esquemas derivados y el caso genérico del usuario.
- [[backend/DECISIONS]] — por qué Landlock, por qué `plan` no escribe y por qué no hay shell en las herramientas del usuario.
- [[backend/04-infrastructure/INTEGRATIONS]] — la integración de los proveedores de modelo e internet.
