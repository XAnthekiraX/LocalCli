# 039-task-eliminar-validacion-proyecto.md

> T-B039 — store: eliminar la validación de proyecto al abrir (`isProjectReported` / `E_NOT_A_PROJECT`), limpiar el centinela muerto y alinear comentarios, seeding y tests.
> Acción de esta descomposición: `actualizar`.

## Referencias

- [[backend/05-quality/ERRORS]] — §2 errores conocidos, §3 códigos internos y §5 errores por operación: ya no existe `E_NOT_A_PROJECT`.
- [[database/03-operations/SEEDING]] — §4 orden de arranque: ya no incluye la comprobación de proyecto.
- [[backend/04-infrastructure/CONFIGURATION]] — §4 Rutas: todo se deriva de la carpeta abierta, sin validar su contenido.
- [[database/DATABASE]] — Almacenamiento: un archivo SQLite por proyecto, derivado de la carpeta abierta.
- [[backend/DECISIONS]] — la fila de `.localcli/state.db` dentro del proyecto.
- [[PROJECT]] — persistencia y "archivos como fuente de verdad".

## Contexto

`ai/changes/Cambios.md` registra que en `internal/store/db.go` se **eliminó** la función
`isProjectReported`, que comprobaba si la carpeta abierta era un proyecto LocalCli
(la existencia de `ai/docs/PROJECT.md` o de algún `ai/tasks/<capa>/MAIN-TASKS.md`) y hacía
fallar `Open` con `E_NOT_A_PROJECT` cuando no lo era. Esa validación **no la respalda
ninguna spec** de `ai/docs/specs/`: fue una decisión de código sin requisito aprobado.

La eliminación quedó a medias y deja el árbol inconsistente:

- El bloque de `isProjectReported` sigue **comentado** en `db.go` (código muerto) y el
  comentario de `Open` aún describe una comprobación que ya no ocurre.
- El centinela `ErrNotAProject` y su alias `ErrNoProyecto` quedan declarados y **sin uso**.
- Dos tests siguen afirmando el comportamiento eliminado y hoy **fallan**:
  `internal/store/store_test.go` y `tests/integration_errores_test.go`.
- La documentación ya se alineó por separado (`ERRORS.md` y `SEEDING.md`).

Comportamiento resultante (documentado por este comando): `store.Open` crea `.localcli/state.db`
en cualquier carpeta, sin validar que sea un proyecto. El alcance se limita a eso: no cambia
el esquema, ni las migraciones, ni la ruta de la base.

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-B039-01 | actualizar | En `db.go`, borrar el bloque comentado de `isProjectReported` y corregir el comentario de `Open` (ya no comprueba que la carpeta sea un proyecto) | completada | `internal/store/db.go` | `go build ./...` compila sin código muerto ni comentarios obsoletos |
| T-B039-02 | actualizar | En `db.go`, eliminar el centinela `ErrNotAProject` y el import `errors` si queda sin uso | completada | `internal/store/db.go` | `go build ./...`; `rg "ErrNotAProject" internal/` no devuelve resultados |
| T-B039-03 | actualizar | En `errors.go`, eliminar el alias `ErrNoProyecto`, que queda sin usos al desaparecer `ErrNotAProject` | completada | `internal/store/errors.go` | `go build ./...`; `rg "ErrNoProyecto"` no devuelve resultados |
| T-B039-04 | actualizar | Actualizar `store_test.go`: eliminar o reemplazar `TestOpenRechazaCarpetaQueNoEsProyecto` para reflejar que `Open` no valida el proyecto (abrir en carpeta vacía crea `.localcli/state.db`) | completada | `internal/store/store_test.go` | `go test ./internal/store/...` pasa |
| T-B039-05 | actualizar | Actualizar `integration_errores_test.go`: quitar `E_NOT_A_PROJECT` de los dos catálogos de códigos, `store.ErrNotAProject` de los centinelas y la aserción que esperaba el error al abrir una carpeta sin proyecto | completada | `tests/integration_errores_test.go` | `go test ./tests/...` pasa; el catálogo coincide con `ERRORS.md` §3 |
| T-B039-06 | actualizar | Verificación final de consistencia: ninguna referencia a la validación eliminada sobrevive en código ni en documentación | completada | `internal/store/`, `tests/`, `ai/docs/` | `rg -i "E_NOT_A_PROJECT\|isProjectReported\|ErrNotAProject\|ErrNoProyecto"` solo aparece en `ai/changes/Cambios.md` |

**Dependencias:** 01→02→03; {02,03}→04; {04}→05; {05}→06. Requiere T-B002 completada.

## Fuera de alcance

- **Sin cambios de esquema ni de migraciones**: la base sigue siendo `.localcli/state.db` por proyecto.
- **Sin tocar la ruta derivada de la carpeta abierta** ([[backend/04-infrastructure/CONFIGURATION]] §4): la validación era lo único que se elimina.
- **Sin reinstalar el comportamiento en otro sitio**: no se añade una comprobación sustituta.
- **Sin modificar las tareas completadas T-B002 ni T-B015**: se planifica en una tarea nueva con acción `actualizar`.
