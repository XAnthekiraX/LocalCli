---
title: LocalCli — validación de entradas
tags: [backend, calidad]
depende_de:
  - "[[specs/SPEC-TOOLS]]"
  - "[[specs/SPEC-ARCHIVOS]]"
  - "[[backend/DECISIONS]]"
relacionado:
  - "[[backend/01-domain/DOMAIN]]"
  - "[[backend/02-interfaces/TOOLS]]"
  - "[[backend/02-interfaces/dto/TOOLS-DTO]]"
  - "[[backend/03-security/SECURITY]]"
  - "[[backend/04-infrastructure/EVENTS]]"
  - "[[backend/05-quality/ERRORS]]"
  - "[[database/01-schema/TABLES]]"
  - "[[database/02-rules/DATA_FLOW]]"
---
# VALIDATION — Validación de entradas

Qué se valida antes de que algo ocurra: rutas, argumentos de herramientas, peticiones del modelo y contenido de archivos. Las reglas funcionales están en [[specs/SPEC-TOOLS]] y [[specs/SPEC-ARCHIVOS]].

## 1. Validaciones de entrada

### Rutas (todas las herramientas de archivo)

- **Relativas a la carpeta abierta.** Una ruta absoluta se rechaza o se reinterpreta como relativa al proyecto; nunca sale de la carpeta por accidente.
- **Dentro de la carpeta:** accesible. Escribir sigue pidiendo aprobación.
- **Fuera de la carpeta:** requiere permiso **y** la explicación del agente. Sin las dos, se rechaza.
- **`crear_archivo` falla si el archivo ya existe.** No sobrescribe.
- **`escribir_archivo` sobrescribe**, y como pisa algo, la aprobación es explícita sobre el contenido que va a quedar.
- **Borrar** (`eliminar_archivo`, `eliminar_carpeta`) pide confirmación explícita, no solo aprobación genérica.
- **`crear_carpeta` falla si ya existe.**
- Se normaliza la ruta antes de validar, para que `../` o rutas equivalentes no esquiven la frontera de la carpeta. Ver [[backend/03-security/SECURITY]].

### Comandos de terminal

- El comando corre en la carpeta del proyecto.
- Solo los de la lista blanca pasan sin preguntar; el resto pide aprobación.
- El texto del comando no se usa para decidir si es seguro: el bloqueo es estructural. La validación no intenta adivinar si un comando escribe; Landlock lo impide. Ver [[backend/02-interfaces/TOOLS]].
- Un comando sin fin o con salida sin fin se corta por tiempo y por tamaño de salida, y se avisa.

### Herramientas de internet

- `buscar_en_internet` recibe solo la consulta. No admite parámetros que adjunten contenido del proyecto.
- `abrir_pagina` recibe una dirección. Lo que vuelve entra al presupuesto de contexto y se audita.

### Peticiones del modelo

- El modelo pide una herramienta por su nombre, y el nombre tiene que estar en el catálogo. Una herramienta inventada se rechaza.
- Los argumentos se validan contra el esquema de la herramienta antes de aplicarse. Una herramienta con argumentos que no encajan no se ejecuta.
- **El orden es: permiso, y solo después validación.** Si el agente no tiene la herramienta, no tiene sentido intentar entender sus argumentos: la respuesta es que no la tiene, no qué le falta. Ver [[backend/03-security/SECURITY]].

### Herramientas del usuario

- Un `.json` de `.localcli/tools/` se valida al cargar: `nombre`, `descripcion` y `equipo` obligatorios; `equipo` no vacío.
- `"modo": "escribe"` se rechaza al arrancar, con el motivo. No es un aviso: la herramienta no entra en el catálogo.
- Un `nombre` que choca con una herramienta incluida se rechaza, con el motivo.
- Un JSON inválido se ignora y se avisa. El arranque continúa y el resto de herramientas se carga.
- **Los argumentos de una herramienta del usuario no se pueden validar contra un esquema propio**, porque el harness no conoce el ejecutable. Solo se comprueba que lo que llega es un objeto. Un nombre de argumento equivocado se descubre al ejecutar, no antes. Ver [[backend/02-interfaces/dto/TOOLS-DTO]].

### Contexto

- Una etapa pide contexto para un objetivo. Si el modelo pide un documento que no existe, la etapa se detiene y avisa; no se continúa suponiendo.
- Lo entregado se recorta para que quepa en el límite del modelo. Nunca se entrega más.
- Un documento entra al contexto solo después de leerse.

## 2. Schemas y tipos

- **Herramientas:** cada una tiene un contrato de entrada y de salida. Los payloads están en [[backend/02-interfaces/dto/TOOLS-DTO]]. Los tipos son los de Go, con validación en la frontera: ruta (string), contenido (string), patrón (string), comando (string), dirección (string).
- **El esquema que ve el modelo se deriva del tipo de Go por reflexión**, con la descripción de cada campo en un tag `desc`. No hay un esquema escrito a mano al lado: si los dos existieran, divergirían sin que nada lo notase. Un campo sin `omitempty` es obligatorio. La equivalencia entre el esquema derivado y el documentado se comprueba con un test, no a revisión.
- **Eventos:** cada evento lleva identificadores y un payload mínimo. Los tipos están en [[backend/04-infrastructure/EVENTS]].
- **Datos:** los tipos de las tablas están en [[database/01-schema/TABLES]] (identificadores como texto UUID, fechas como texto ISO, booleanos como enteros). `store` no acepta tipos sueltos: usa los valores de la base.
- **Tareas y documentación:** el frontmatter se valida al leerlo. Si un campo declarado falta o no tiene el tipo esperado, se avisa. Ver [[backend/01-domain/DOMAIN]].

## 2.1 Un fallo de validación no es una avería

Es la diferencia más importante de esta capa, y la razón por la que se documenta aparte.

Un campo obligatorio que falta, un nombre de ruta que no existe, un archivo que ya está: **son cosas que el modelo puede arreglar**. Así que no cortan el turno. Vuelven al modelo como un resultado con el motivo y con lo que se esperaba, y el modelo reintenta en la misma pasada.

```
falta el campo obligatorio "ruta"
"editar_archivo" espera: ruta (obligatorio), cambio (obligatorio)
```

Un fallo que el modelo no puede arreglar —se le denegó el permiso, el sistema no responde, se canceló— **sí corta el turno**. Se le dice, pero no se le reintenta.

La distinción está en el tipo: `Resultado.Error` para lo corregible, un `error` de Go para lo que no lo es. Mezclarlos es lo que hace que un nombre de archivo mal escrito termine una sesión de trabajo.

El mensaje que vuelve al modelo tiene dos requisitos: **nombrar qué falta** y **no filtrar lo que el usuario no tiene que ver**. Una ruta absoluta de tu máquina en un mensaje de error es una fuga de información en la pantalla y en la auditoría, y también una pista innecesaria para el modelo.

## 3. Campos obligatorios y opcionales

| Entrada | Obligatorio | Opcional |
|---|---|---|
| Herramientas de lectura de archivo | Ruta (relativa al proyecto) | — |
| `crear_archivo` / `escribir_archivo` | Ruta y contenido | — |
| `editar_archivo` | Ruta y el cambio a aplicar | — |
| `eliminar_archivo` / `eliminar_carpeta` | Ruta, más confirmación explícita | — |
| `crear_carpeta` | Ruta | — |
| `ejecutar_comando` | Comando | Carpeta de trabajo (por defecto, la del proyecto) |
| `buscar_en_internet` | Consulta | — |
| `abrir_pagina` | Dirección | — |
| Herramienta del usuario | `nombre`, `descripcion`, `equipo` | `modo` (por defecto, `lee`); `timeout_segundos` |
| Objetivo de una etapa | Objetivo | — |

Una herramienta sin su campo obligatorio no se ejecuta; se rechaza con un error claro, que es un caso normal y no una avería. Ver [[backend/05-quality/ERRORS]].

## 4. Transformaciones

- **Rutas:** se normalizan antes de validar la frontera, para que las formas equivalentes de llegar al mismo archivo se traten igual.
- **Salida de herramienta:** se recorta al presupuesto de contexto, medido en tokens, y se marca que fue cortada. **Aplica a toda herramienta**, no solo a la terminal: una lectura enorme o el contenido de una página desbordan igual que un comando sin fin. Si el handler ya recortó por su cuenta, la capa universal respeta esa marca y no vuelve a recortar.
- **Salida de comando:** tiene además su propio límite, más estricto, porque un comando es la fuente más fácil de descontrolar. Ver [[backend/DECISIONS]].
- **Contenido de internet:** se recorta al presupuesto de contexto, igual que el resto, y se audita.
- **Razonamiento en streaming:** se acumula aparte del mensaje, y el mensaje se escribe cuando la respuesta termina. Ver [[database/02-rules/DATA_FLOW]].
- **Tokens:** se contabilizan por mensaje para poder ajustar el contexto en iteraciones siguientes. Ver [[database/01-schema/TABLES]].

## 5. Reglas de validación cruzadas

- **Una ruta que sale de la carpeta necesita permiso y explicación.** Son las dos cosas, no una.
- **Una escritura necesita aprobación, y borrar necesita confirmación explícita.** El borrado es un subcaso más estricto, no un caso paralelo.
- **Una herramienta de escritura la pide `build`.** Si la pide `plan`, se rechaza, y no por suerte: `plan` no la tiene. Ver [[backend/03-security/SECURITY]].
- **El modelo no puede concederse permisos.** La validación la hacen los módulos, no el modelo.
- **La terminal no valida el comando para decidir si es seguro.** Esa validación no existe a propósito; la seguridad la da Landlock.
- **Las herramientas del usuario se validan al cargar, no al ejecutar.** Un JSON inválido no produce un fallo en mitad de un turno: no llega a estar en el catálogo.
- **Una conexión de base de datos siempre tiene `foreign_keys` activado**, o el borrado en cascada no se sostiene. Ver [[backend/DECISIONS]].

## Referencias

- [[specs/SPEC-TOOLS]] y [[specs/SPEC-ARCHIVOS]] — reglas funcionales de permiso y rutas.
- [[backend/02-interfaces/TOOLS]] — el contrato de cada herramienta y la capa universal que valida.
- [[backend/02-interfaces/dto/TOOLS-DTO]] — los payloads que se validan y los esquemas derivados.
- [[backend/03-security/SECURITY]] — qué es garantía y qué es validación.
- [[backend/05-quality/ERRORS]] — qué error corta el turno y cuál vuelve corregible.
- [[database/01-schema/TABLES]] — tipos de los datos.
