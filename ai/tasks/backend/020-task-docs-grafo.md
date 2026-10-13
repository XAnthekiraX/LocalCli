# 020-task-docs-grafo.md

> T-B020 — Documentación (`docs`) y grafo de frontmatter.
> Acción de esta descomposición: `crear`.

## Referencias

- [[specs/SPEC-NODO-CONTEXTO]] — la documentación y sus dependencias como mapa
- [[PROJECT]] §Convención del frontmatter — `depende_de` y `relacionado`, y el cierre que no arrastra el proyecto
- [[AGENTS.md]] §Convención de referencias — wiki link `[[carpeta/ARCHIVO]]` relativo a `ai/docs/`
- [[backend/BACKEND]] §3 (`docs`) — carga la documentación, lee el frontmatter y construye el grafo; no guarda nada
- [[backend/01-domain/DOMAIN]] §`docs` — el grafo se reconstruye en memoria al arrancar
- [[backend/02-interfaces/INTERFACES-GENERAL]] §5 — `docs` provee el grafo a `context`
- [[backend/04-infrastructure/CONFIGURATION]] §4 — la documentación vive en `ai/docs/`
- [[backend/05-quality/ERRORS]] §3/§5 — `E_DOC_PARSE`, `E_DOC_NOT_FOUND`, `E_STAGE_FAILED`
- [[backend/05-quality/TESTING]] §`docs` — qué cubren las pruebas del módulo

## Contexto

El módulo `docs` carga `ai/docs/**/*.md`, lee el frontmatter, extrae los
wiki-links del cuerpo y construye en memoria el grafo de dependencias; no
persiste nada. Las aristas salen **sobre todo** del frontmatter (`depende_de` es
dependencia fuerte; `relacionado` toca sin depender) y, de forma complementaria,
de los enlaces del cuerpo, que son menciones. El nodo de contexto consume este
módulo por la interfaz `context.Fuente` (`Documentos()` y `Leer(ruta)`), que
T-B017 dejó pendiente de su implementación real.

## Tareas pequeñas

| ID        | Acción | Tarea                                                                                                   | Estado    | Archivos                              | Verificación                                                                  |
| --------- | ------ | ------------------------------------------------------------------------------------------------------- | --------- | ------------------------------------- | ----------------------------------------------------------------------------- |
| T-B020-01 | crear  | Modelo del documento y su frontmatter (`Doc` y `Meta`): ruta, crudo, cuerpo y enlaces                   | completada | `internal/docs/documento.go`          | Un archivo sin frontmatter carga con meta vacía y sin error                   |
| T-B020-02 | crear  | Parseo del frontmatter YAML (escalar, lista inline y lista en bloque; comillas y comentarios)           | completada | `internal/docs/frontmatter.go`        | Un frontmatter roto devuelve error localizado, nunca pánico                   |
| T-B020-03 | crear  | Extracción de los wiki-links `[[...]]` del cuerpo, ignorando los que están en bloques o código en línea | completada | `internal/docs/enlaces.go`            | Un wiki-link dentro de ``` no cuenta como dependencia                         |
| T-B020-04 | crear  | Carga recursiva de `ai/docs/**/*.md` con errores localizados que no abortan la carga                     | completada | `internal/docs/carga.go`              | Una raíz que no es carpeta es `E_STAGE_FAILED`; un frontmatter roto se reporta y sigue |
| T-B020-05 | crear  | Grafo en memoria: aristas del frontmatter (dependencia fuerte y relación) y, complementarias, del cuerpo | completada | `internal/docs/grafo.go`              | Una etiqueta no crea arista; el grafo no queda vacío con frontmatter          |
| T-B020-06 | crear  | Consultas del grafo: por etiqueta, directas y cierre transitivo, documento por ruta o wiki-link y rotos | completada | `internal/docs/consulta.go`           | Pedir un documento inexistente devuelve `E_DOC_NOT_FOUND`                      |
| T-B020-07 | crear  | Errores del módulo (`E_DOC_PARSE` localizado, `E_STAGE_FAILED` y `E_DOC_NOT_FOUND`)                      | completada | `internal/docs/errores.go`            | `errors.Is(err, ErrStageFailed)` sin inspeccionar el mensaje                  |
| T-B020-08 | crear  | Fuente de contexto: implementar `context.Fuente` (`Documentos()` y `Leer(ruta)`) sobre el grafo          | completada | `internal/docs/fuente.go`             | El nodo de contexto recibe el catálogo con las dependencias declaradas         |
| T-B020-09 | crear  | Pruebas del módulo con testdata (carga, frontmatter, enlaces, grafo y rotos)                            | completada | `internal/docs/docs_test.go`, `internal/docs/testdata/` | `go test ./internal/docs/`                                     |
| T-B020-10 | crear  | Pruebas sobre la documentación real: frontmatter, sin enlaces rotos ni ciclos, y cierre que no arrastra  | completada | `internal/docs/ai_docs_test.go`       | `go test ./internal/docs/` sobre `ai/docs/`; `TestElCierreDeDependenciasNoArrastraElProyecto` |

**Dependencias:** `01`→`02`,`03`; `02`,`03`→`04`; `04`→`05`; `05`→`06`,`10`; `05`,`07`→`08`; `06`,`07`→`08`; `01`–`07`→`09`.
**Fuera de alcance (queda para sus tareas):** la selección, el recorte y la auditoría de contexto ya viven en `internal/context`; aquí solo se provee el grafo y la lectura. La inyección de esta fuente en el nodo de contexto por etapa es del cableado de contexto, no de este módulo.
