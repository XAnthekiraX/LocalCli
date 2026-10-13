# 012-task-catalogo-herramientas.md

> T-B012 — Catálogo de herramientas y capa universal de ejecución.
> Acción de esta descomposición: `crear`.

## Referencias

- [[backend/02-interfaces/TOOLS]] §1 El catálogo incluido, §2 El reparto, §7 La forma de una herramienta, §8 La capa universal, §9 Hooks, §10 El cable
- [[backend/02-interfaces/dto/TOOLS-DTO]] §2 Los tipos son la fuente del esquema, §3 Request Schemas, §4 Response Schemas
- [[specs/SPEC-TOOLS]] §Cómo se le pide una herramienta al modelo, §La capa universal, §Errores que el modelo puede corregir, §Recorte de la salida
- [[backend/04-infrastructure/EVENTS]] §3 Payloads (`herramienta_invocada`, `herramienta_resultado`)
- [[backend/03-security/SECURITY]] — reparto de permisos y capa universal
- [[backend/05-quality/VALIDATION]] §2 Schemas y tipos, §2.1 Un fallo de validación no es una avería
- [[backend/05-quality/ERRORS]] §3 Códigos internos, §5 Errores por operación
- [[backend/01-domain/DOMAIN]] — límites entre `tools`, `agent` y el núcleo del motor

## Contexto

`tools` define el contrato de las herramientas y la **capa universal** por la que pasa toda ejecución, sin excepciones. No importa `fileops`, `exec` ni el cliente de internet: las implementaciones se inyectan en el cableado, y una herramienta nueva es un `Ejecutar` propio, no un caso en un `switch`.

Aquí vive lo que no pertenece a ninguna herramienta: los tipos del contrato, la derivación del esquema por reflexión, el registro cerrado del catálogo, el reparto por permisos del agente, la validación, el recorte, la medición de la duración, la emisión de los dos eventos de herramienta y los hooks. El handler concreto de cada herramienta de archivo se implementa en T-B014.

Dos reglas que no admiten excepción: **el catálogo no viaja en el mensaje de sistema** —va por el canal nativo del motor— y **el orden de la capa universal es permiso antes que validación**: si el agente no tiene la herramienta, no se intenta entender sus argumentos.

Tercera particularidad: `ejecutar_flujo` se registra en el catálogo (dieciséis herramientas) pero con **permiso derivado**: su esquema, con el enum de flujos, lo arma `tools` por turno con el agente activo y la lista de flujos que `flow` cargó; su `Ejecutar` se inyecta en el cableado y llama al orquestador (T-B017). Dentro de una etapa de flujo, la herramienta no existe.

## Tareas pequeñas

| ID         | Acción | Tarea                                                                                                          | Estado    | Archivos                       | Verificación                                                                          |
| ---------- | ------ | -------------------------------------------------------------------------------------------------------------- | --------- | ------------------------------ | ------------------------------------------------------------------------------------- |
| T-B012-01  | crear  | Tipos del contrato: `Contexto`, `Resultado`, `Herramienta`, `Categoria`, `Permiso`, `Modo`, `Esquema`, `Ejecutar`, `Hooks` | completada | `internal/tools/modelo.go`     | El registro se construye con una lista de `Herramienta`                               |
| T-B012-02  | crear  | Derivación del JSON Schema por reflexión del tipo de petición (tag `json`, `desc`; `omitempty` = opcional); el `desc` de `ruta` instruye la forma corta relativa, sin `~` | completada | `internal/tools/esquema.go`    | Un campo sin `omitempty` queda en `required`; con él, aparece en `properties`         |
| T-B012-03  | crear  | Registro cerrado del catálogo: las dieciséis herramientas con su permiso, categoría, modo y datos de presentación; `ejecutar_flujo` es la única sin permiso fijo (derivado del flujo) y su handler se inyecta desde el orquestador (T-B017) | completada | `internal/tools/catalogo.go`   | Una herramienta fuera del catálogo devuelve `E_TOOL_UNKNOWN`; `ejecutar_flujo` no está en el registro dentro de una etapa de flujo |
| T-B012-04  | crear  | Reparto por permisos del agente (`read`/`write`/`edit`) contra el catálogo                                       | completada | `internal/tools/reparto.go`    | `plan` no recibe ninguna herramienta de escritura; `build` recibe el catálogo completo |
| T-B012-05  | crear  | Capa universal `Registro.Ejecutar`: buscar, permiso, validar, emitir invocada, ejecutar y medir, recortar, emitir resultado | completada | `internal/tools/registro.go`   | El permiso se comprueba antes de validar los argumentos                               |
| T-B012-06  | crear  | Recorte propio de `tools`, medido en bytes, con su propio estimador pequeño                                      | completada | `internal/tools/recorte.go`    | Una salida ya recortada por el handler respeta su `Truncado` y no se vuelve a cortar  |
| T-B012-07  | crear  | Emisión de `herramienta_invocada` (verbo y tema) y `herramienta_resultado` (ok/error, medida, recorte, duración) | completada | `internal/tools/eventos.go`    | La duración mide solo el handler, no la espera de aprobación                          |
| T-B012-08  | crear  | Composición del tema: para `leer_archivo`, `ruta` o `ruta:inicio-fin` con el rango real devuelto, siempre en forma canónica (relativa si está dentro del directorio de trabajo)  | completada | `internal/tools/tema.go`       | Una lectura completa deja el tema en `ruta`; una parcial, en `ruta:inicio-fin`; el tema nunca muestra el path absoluto del proyecto |
| T-B012-09  | crear  | Completar la línea al cerrar: al llegar el resultado de `leer_archivo`, aplicar el rango real de `Resultado.Meta` | completada | `internal/tools/eventos.go`    | Una lectura completa truncada por el límite cierra la línea con `ruta:1-N`             |
| T-B012-10  | crear  | Medida del resultado según la unidad declarada (líneas, entradas, coincidencias, resultados, pasos)             | completada | `internal/tools/medida.go`     | La medida no viaja al modelo, solo al evento y a `chat_evento`                        |
| T-B012-11  | crear  | Errores corregibles: `Resultado.Error` (el modelo reintenta) frente a `error` de Go (corta el turno)            | completada | `internal/tools/errores.go`    | Un campo obligatorio ausente no termina el turno                                       |
| T-B012-12  | crear  | Hooks `AntesDeEjecutar`, `DespuesDeEjecutar` y `DefinirHerramienta`, opcionales por diseño                      | completada | `internal/tools/hooks.go`      | Sin hooks, la capa universal funciona igual                                            |
| T-B012-13  | crear  | Entrega de los esquemas del agente activo en la petición neutra (`llm.Peticion`), sin tocar el mensaje de sistema | completada | cableado (`cmd/localcli`)      | El catálogo no aparece en el prompt de sistema                                         |

**Dependencias:** `01`→`02`→`03`; `03`,`04`→`05`; `05`→`06`,`07`,`11`; `07`→`08`,`10`; `08`→`09`; `07`–`12`→`13`.
**Fuera de alcance:** el handler de cada herramienta incluida (archivos en T-B014, terminal en T-B015, internet); el handler de `ejecutar_flujo`, que es el orquestador de flujos y se inyecta desde T-B017; la persistencia de las líneas en `chat_evento` (T-B010/T-B011); el adaptador nativo del motor (T-B006).
