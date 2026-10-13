# 014-task-fileops.md

> T-B014 — Acceso a archivos (`fileops`) y aprobaciones.
> Acción de esta descomposición: `crear`.

## Referencias

- [[backend/02-interfaces/TOOLS]] §1 Archivos, §4 Herramientas de archivo (§Lectura por rangos, §Resolución de rutas)
- [[backend/02-interfaces/dto/TOOLS-DTO]] §3 Request Schemas, §4 Response Schemas (§Lectura de archivos)
- [[specs/SPEC-TOOLS]] §Herramientas de archivo, §`leer_archivo`, §Resolución de rutas
- [[specs/SPEC-ARCHIVOS]] — acceso dentro/fuera de la carpeta y permiso
- [[backend/05-quality/VALIDATION]] §1 Rutas, §1 Lectura de archivo, §4 Transformaciones
- [[backend/05-quality/ERRORS]] §3, §5 Errores por operación
- [[backend/03-security/SECURITY]] — frontera de la carpeta y aprobación
- [[database/01-schema/TABLES]] §`change_history` — registro de escrituras

## Contexto

`fileops` implementa los handlers de las herramientas de archivo. Ninguna de ellas decide permisos por su cuenta: las reglas de acceso y la aprobación viven en la capa universal de `tools` (T-B012) y aquí solo se ejecutan y se devuelve el `Resultado`.

Una regla que no admite excepción: **`leer_archivo` no modifica ni elimina archivos y no ejecuta su contenido**. Lee texto, entrega lo que cabe y deja que el modelo continúe por rangos si el archivo es grande.

## Tareas pequeñas

| ID         | Acción | Tarea                                                                                                        | Estado    | Archivos                    | Verificación                                                                                        |
| ---------- | ------ | ------------------------------------------------------------------------------------------------------------ | --------- | --------------------------- | --------------------------------------------------------------------------------------------------- |
| T-B014-01  | crear  | Resolución de rutas compartida por `leer_archivo` y `listar_carpeta`: relativas desde el directorio de trabajo, `..`, absolutas; sin `~` ni `$HOME` —rechazarlas con `E_BAD_ARGS` y un recordatorio de usar la forma relativa—; sin buscar en otras ubicaciones | completada | `internal/fileops/rutas.go` | `../main.go` y `/tmp/x.txt` resuelven; `~` no se expande y se rechaza con `E_BAD_ARGS`  |
| T-B014-02  | crear  | Normalizar después de resolver sin eludir la frontera: una ruta fuera de la carpeta sigue siendo «fuera»     | completada | `internal/fileops/rutas.go` | `../` no esquiva la comprobación de acceso                                                            |
| T-B014-03  | crear  | Validación de acceso: dentro de la carpeta accesible; fuera exige permiso y `motivo` (`E_PATH_OUTSIDE`)       | completada | `internal/fileops/acceso.go` | Sin permiso y `motivo`, una ruta exterior se rechaza con `E_PATH_OUTSIDE`                             |
| T-B014-04  | crear  | `leer_archivo`: validar que la ruta sea un archivo regular y legible como texto                               | completada | `internal/fileops/leer.go` | Un directorio, un archivo sin permisos o un binario devuelven un error corregible                     |
| T-B014-05  | crear  | `leer_archivo`: sin rango, devolver el archivo completo si no supera 64 KiB                                  | completada | `internal/fileops/leer.go` | Un archivo pequeño se devuelve entero con `truncado=false`                                           |
| T-B014-06  | crear  | `leer_archivo`: si supera 64 KiB, devolver el fragmento inicial con `truncado=true` y `siguiente_linea`       | completada | `internal/fileops/leer.go` | El resultado indica por dónde continuar                                                               |
| T-B014-07  | crear  | `leer_archivo`: con rango, devolver las líneas `inicio_linea`-`fin_linea`, numeradas desde 1 y del archivo original | completada | `internal/fileops/leer.go` | El rango se aplica al original, nunca a un fragmento previamente truncado                              |
| T-B014-08  | crear  | `leer_archivo`: respetar el máximo de 32 KiB por respuesta y, si el rango no cabe, devolver hasta donde llega indicando dónde terminó | completada | `internal/fileops/leer.go` | Nunca se supera el límite; la salida dice dónde se cortó                                              |
| T-B014-09  | crear  | `leer_archivo`: señalar una línea individual demasiado grande (se devuelve lo que quepa)                      | completada | `internal/fileops/leer.go` | La línea aparece recortada y avisada                                                                  |
| T-B014-10  | crear  | `leer_archivo`: armar la respuesta con `ruta` (canónica: relativa al directorio de trabajo si está dentro, absoluta si está fuera), `contenido`, `linea_inicio`, `linea_fin`, `total_lineas`, `truncado`, `siguiente_linea` | completada | `internal/fileops/leer.go` | El modelo ve el tramo leído y la ruta en su forma corta cuando el archivo está dentro |
| T-B014-11  | crear  | `leer_archivo`: publicar el rango real y el truncado en `Resultado.Meta` para que la capa universal complete la línea | completada | `internal/fileops/leer.go` | Una lectura completa truncada deja el tema en `ruta:1-N`                                               |
| T-B014-12  | crear  | `listar_carpeta`: un nivel, con la resolución de rutas de T-B014-01                                               | completada | `internal/fileops/listar.go` | Lista las entradas de un nivel; una ruta exterior respeta el acceso                                    |
| T-B014-13  | crear  | Escrituras (`crear_archivo`, `escribir_archivo`, `editar_archivo`, `eliminar_archivo`, `crear_carpeta`, `eliminar_carpeta`) con aprobación vía `Contexto.Ask` | completada | `internal/fileops/escribir.go` | Toda escritura pasa por aprobación; borrar exige confirmación explícita                                |
| T-B014-14  | crear  | Registrar cada escritura aprobada en `change_history` (antes y después)                                        | completada | `internal/fileops/historial.go` | Una escritura aplicada queda con su `before_content`/`after_content`                                  |
| T-B014-15  | crear  | `buscar_archivos`: recorre el proyecto y devuelve las rutas cuyo nombre contiene el patrón, saltando las carpetas ocultas y las dependencias, con tope de coincidencias | completada | `internal/fileops/buscar.go` | Encuentra un archivo por su nombre; `.git`/`node_modules` no se recorren |
| T-B014-16  | crear  | `buscar_en_archivos`: busca el patrón dentro de los archivos de texto, acotado opcionalmente por `ruta`, con la misma frontera de acceso | completada | `internal/fileops/buscar.go` | Devuelve (archivo, línea, fragmento); una `ruta` exterior sin `motivo` da `E_PATH_OUTSIDE` |

**Dependencias:** `01`→`02`→`03`; `03`→`04`; `04`→`05`,`06`,`07`,`12`; `05`→`06`; `06`,`07`→`08`; `08`,`09`→`10`; `10`→`11`; `03`→`13`; `13`→`14`.
**Fuera de alcance:** la capa universal y la composición del tema de la línea (T-B012); el aislamiento de la terminal (T-B015); el esquema que ve el modelo, que se deriva en `tools` del tipo de petición.