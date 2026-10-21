# 028-task-eliminar-modelos.md

> T-B028 — Eliminación de modelos: borrar la carpeta `models/<id>/` y su entrada de `models.json`, con el modelo en uso protegido.
> Acción de esta descomposición: `actualizar`.

## Referencias

- [[specs/SPEC-MODELOS-LLAMACPP]] §5 Catálogo de modelos (eliminación, modelo en uso y sesiones).
- [[backend/04-infrastructure/CONFIGURATION]] §8 El catálogo de modelos.
- [[backend/01-domain/BUSINESS_RULES]] §Modelos.
- [[backend/02-interfaces/INTERFACES-GENERAL]] §5 Contratos entre módulos.
- [[database/01-schema/TABLES]] §`sessions` — `modelo` no se valida contra el motor.

## Contexto

El catálogo sabe dar de alta, cargar, guardar y seleccionar modelos, pero no borrarlos.
La única eliminación admitida es la explícita del usuario desde el modal de modelos
(T-F017): `Ctrl+D`, confirmación `s/n` y, al confirmar, desaparecen la carpeta completa
`~/.config/localcli/models/<id>/` y la entrada de `models.json`.

Reglas que la implementación debe sostener:

- La ruta que se borra se construye con el directorio del catálogo y el `id`, **nunca**
  desde `ruta`: un catálogo editado a mano no puede llevar el borrado fuera de `models/`.
- Se borra primero la carpeta y después se guarda el catálogo. Si la carpeta no se puede
  borrar, el catálogo no se toca. Si la carpeta ya no existe, la entrada se limpia igual.
- El modelo en uso no se elimina: ni el de la sesión, ni el `seleccionado` del catálogo,
  ni el que tiene cargado `llama-server`.
- Las sesiones no se tocan: `sessions.modelo` conserva el nombre y la vista avisa al
  retomarlas.

## Tareas pequeñas

| ID        | Acción     | Tarea                                                                                                                                          | Estado    | Archivos                                        | Verificación                                                                                                       |
| --------- | ---------- | ---------------------------------------------------------------------------------------------------------------------------------------------- | --------- | ----------------------------------------------- | ------------------------------------------------------------------------------------------------------------------ |
| T-B028-01 | actualizar | `models.Gestor.Eliminar(id)`: buscar la entrada, borrar `models/<id>/` completo, quitarla del catálogo y guardar de forma atómica; si la carpeta falla, no se escribe el catálogo; si ya no existe, se limpia la entrada | completada | `internal/models/catalogo.go`, `internal/models/catalogo_test.go` | Borra la carpeta (GGUF + mmproj) y la entrada; un `id` inexistente o inválido devuelve error; una ruta de catálogo fuera de `models/` no se usa para borrar |
| T-B028-02 | actualizar | `session.Orquestador.EliminarModelo(nombre)`: rechazar el modelo activo comparando con `o.modelo`, `models.json.seleccionado` y `llama.Activo()`; delegar en `models.Gestor.Eliminar` en el resto de casos | completada | `internal/session/orquestador.go`, `internal/session/*_test.go` | El activo devuelve un error accionable y no toca disco ni catálogo; un modelo no activo se elimina |
| T-B028-03 | actualizar | Exponer `EliminarModelo(nombre string) error` en el `Puerto` de la TUI y en el cableado; el `app` recibe el error tal cual para pintarlo | completada | `internal/tui/wire.go`, `cmd/localcli/main.go`, dobles de prueba de la TUI | La TUI compila; el doble del puerto registra el nombre y devuelve el error del backend |
| T-B028-04 | actualizar | Pruebas del catálogo y de la sesión: carpeta completa, entrada huérfana, catálogo mal formado, modelo activo por cada una de las tres fuentes y modelo no activo | completada | `internal/models/catalogo_test.go`, `internal/session/*_test.go` | Cada regla tiene su prueba; ningún caso deja un GGUF fuera del catálogo ni una entrada apuntando a una carpeta borrada |

**Dependencias:** `01` → `02` → `03`; `04` valida las anteriores. T-B028 depende de T-B004
(catálogo y selección). T-F017 consume el puerto y no empieza hasta que `03` esté cerrado.

**Fuera de alcance:** borrar desde la CLI, papelera o recuperación, eliminar modelos que no
estén en el catálogo, borrar una sesión o su historial, y detener `llama-server` para
eliminar el modelo en uso (se bloquea en su lugar).

## Notas

- El catálogo mal formado no se sobrescribe: `Eliminar` falla antes de escribir, igual que
  `Alta` y `Seleccionar`.
- La operación no emite eventos: la lista del modal se recarga con la lectura que ya
  existe (`Modelos()`), y el aviso de error viaja como valor de retorno.
- `sessions.modelo` no se actualiza ni se limpia: la ausencia del modelo se resuelve al
  usarlo, como ya documenta [[database/01-schema/TABLES]].
