# 017-task-eliminar-modelos-modal.md

> T-F017 — Modal de modelos: eliminar el resaltado con `Ctrl+D` y confirmación `s/n`, con recarga de la lista y bloqueo del modelo en uso.
> Acción de esta descomposición: `actualizar`.

## Referencias

- [[specs/SPEC-INTERFAZ]] §Modales y §Reglas de negocio (eliminación desde el modal de modelos).
- [[specs/SPEC-MODELOS-LLAMACPP]] §5 Catálogo de modelos.
- [[specs/SPEC-KEYBINDS]] §Acción (`model_delete`) y §Resolución por contexto.
- [[specs/SPEC-INTERFAZ-ATAJOS]] §Reglas de negocio.
- [[frontend/01-domain/DOMAIN]] §1 Componentes (`modals`) y §2 Reglas de presentación.
- [[frontend/02-interfaces/INTERFACES]] §4 Teclado y §5 Estados de espera.
- [[frontend/05-quality/TESTING]] §2 Qué debe probarse.

## Contexto

El modal de modelos lista y aplica, pero no borra. Esta tarea añade la eliminación desde
el modal con el mismo patrón que el modal de sesiones, cambiando la confirmación: el
usuario pulsa `Ctrl+D` sobre el resaltado, el modal pregunta
«¿seguro que deseas eliminar `<modelo>`? (s/n)» y solo `s` concede el borrado. `n` y `Esc`
cancelan la pregunta sin cerrar el modal.

La TUI no borra nada: entrega el modelo al `app`, que llama al puerto
(`EliminarModelo`, T-B028). El backend decide si el modelo está en uso y devuelve el
aviso; la vista lo pinta dentro del modal. Tras un borrado correcto la lista se vuelve a
leer y el resaltado queda en el elemento siguiente —o aparece «sin modelos» si no queda
ninguno—.

## Tareas pequeñas

| ID        | Acción     | Tarea                                                                                                                                                                   | Estado    | Archivos                                                                 | Verificación                                                                                                                   |
| --------- | ---------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------- | ------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------ |
| T-F017-01 | actualizar | Estado de confirmación en `Modelos`: `PedirEliminar()`, `Confirmando()`, `CancelarConfirmacion()` y repintado de la pregunta antes del pie; el aviso de error se pinta sin perder la lista | completada | `internal/tui/components/modals/modelosmodal.go`, `internal/tui/components/modals/modelosmodal_test.go` | `Ctrl+D` sobre un modelo pide confirmación y no entrega nada; `s`/`n` se resuelven fuera del modal; la pregunta desaparece al cancelar |
| T-F017-02 | actualizar | Añadir `model_delete` (`ctrl+d`) al keymap, categoría Modales y ámbito **solo modal de modelos**, con su descripción y su entrada en `keys.json` | completada | `internal/tui/keymap.go`, `internal/tui/keymap_test.go`, `internal/tui/keys.go` | El mapa por defecto valida sin duplicados: `ctrl+d` sigue siendo `panel_toggle` en la vista y `session_delete` en sesiones |
| T-F017-03 | actualizar | Cablear la eliminación en `app`: comando que llama al puerto con el modelo resaltado, mensaje de resultado, recarga de la lista conservando la posición y aviso del error del backend; `s`/`n` en `teclaDeModal` solo con el modal de modelos confirmando, y `Esc` cancela la pregunta antes de cerrar | completada | `internal/tui/app.go`, `internal/tui/teclado.go`, `internal/tui/modales_test.go` | `s` borra y la lista se recarga; `n`/`Esc` no borran; el error del modelo en uso se ve en el modal y la lista no cambia |
| T-F017-04 | actualizar | Pruebas de la TUI: confirmación, modelo en uso rechazado, borrado con recarga, último modelo (lista vacía) y `Esc` que cancela sin cerrar; actualizar los dorados del modal | completada | `internal/tui/components/modals/*_test.go`, `internal/tui/*_test.go`, `internal/tui/testdata/*.golden` | Cada regla tiene su prueba y los dorados existentes se revisan, no se regeneran a ciegas |

**Dependencias:** `01` → `03`; `02` → `03`; `04` valida las anteriores. T-F017 depende de
T-F006 (modal de modelos) y de T-B028 (borrado en el catálogo y puerto `EliminarModelo`).

**Fuera de alcance:** borrar modelos desde la paleta o desde la bienvenida sin abrir el
modal, deshacer un borrado, papelera, y cambiar el modelo activo automáticamente al
eliminar otro.

## Notas

- `s`/`n` no son atajos del mapa: son la respuesta al diálogo, como el texto del campo de
  URL del modal de descarga. Solo `Ctrl+D` dispara la eliminación.
- El modal no decide si un modelo se puede borrar: pinta el error que devuelve el puerto.
  Que el modelo esté en uso lo resuelve el backend comparando sesión, catálogo y
  `llama-server`.
- Con la confirmación pendiente, el pie del modal sigue anunciando las teclas de la lista;
  la pregunta y sus opciones van en una línea propia para que no se confundan con un
  elemento más.
