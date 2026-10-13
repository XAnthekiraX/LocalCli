# 002-task-teclas.md

> T-F002 — Mapa y resolución de teclas (keymap, keyresolver, keys, config).
> Acción de esta descomposición: `crear`.

## Referencias

- [[frontend/FRONTEND]] §2 Estructura del Proyecto (`keymap.go`, `keyresolver.go`, `keys.go`, `config.go`)
- [[frontend/01-domain/DOMAIN]] §1 Componentes (`keys`)
- [[frontend/02-interfaces/INTERFACES]] §4 Teclado
- [[specs/SPEC-KEYBINDS]] §Modelo de datos, §Tecla líder, §Resolución por contexto, §Configuración
- [[specs/SPEC-INTERFAZ-ATAJOS]] §Reglas de negocio (atajos)
- [[frontend/05-quality/TESTING]] §2 Qué debe probarse (duplicados, persistencia)

## Tareas pequeñas

| ID         | Acción | Tarea                                                                                          | Estado    | Archivos                              | Verificación                                                                          |
| ---------- | ------ | ---------------------------------------------------------------------------------------------- | --------- | ------------------------------------- | ------------------------------------------------------------------------------------- |
| T-F002-01  | crear  | Catálogo de acciones con ID estable y atajos por defecto                                       | completada | `internal/tui/keymap.go`              | Cada ID y su atajo por defecto coinciden con la spec (sin acciones de motor)          |
| T-F002-02  | crear  | Formato de binding: tecla simple y secuencia con prefijo `<leader>`                            | completada | `internal/tui/keymap.go`              | `<leader>` se expande con la tecla líder configurada                                  |
| T-F002-03  | crear  | Regla de duplicados por contexto: un literal no puede pertenecer a dos acciones del mismo ámbito | completada | `internal/tui/keymap.go`              | El duplicado en el mismo contexto se rechaza al cargar y al guardar                   |
| T-F002-04  | crear  | Máquina de estados del resolver (`NORMAL` / `LEADER`)                                          | completada | `internal/tui/keyresolver.go`         | La líder entra en `LEADER`; se vuelve a `NORMAL` al completar, cancelar o expirar     |
| T-F002-05  | crear  | Tecla líder `ctrl+x` con timeout (2000 ms, configurable 500–10000) y cancelación               | completada | `internal/tui/keyresolver.go`         | Sin segunda tecla en el timeout vuelve a `NORMAL`; `esc` y re-líder cancelan la espera |
| T-F002-06  | crear  | Resolución por contexto: modal → input → vista → global                                        | completada | `internal/tui/keyresolver.go`         | Con modal abierto sus teclas no llegan a la vista; escribir no dispara acciones        |
| T-F002-07  | crear  | Entrega de la tecla cruda al componente cuando ninguna acción coincide                         | completada | `internal/tui/keyresolver.go`         | Una tecla sin binding llega al componente con el foco                                  |
| T-F002-08  | crear  | Indicador discreto de la líder mientras se espera la segunda tecla                            | completada | `internal/tui/keyresolver.go`, `internal/tui/app.go` | El indicador aparece al pulsar la líder y desaparece al expirar          |
| T-F002-09  | crear  | Carga de `~/.config/localcli/keys.json` con relleno de defaults y rechazo de inválidos        | completada | `internal/tui/keys.go`                | Archivo ausente/incompleto usa los defaults; lo inválido se rechaza y avisa           |
| T-F002-10  | crear  | Persistencia y reasignación de atajos sin reiniciar                                           | completada | `internal/tui/keys.go`                | Reasignar escribe el archivo y surte efecto sin reiniciar                             |
| T-F002-11  | crear  | Preferencias en `~/.config/localcli/config.json`: último agente y último modelo               | completada | `internal/tui/config.go`              | Se leen al arrancar y se escriben al cambiar de agente o de modelo                    |
| T-F002-12  | crear  | Integrar el resolver en `app`: los componentes consumen acciones, nunca teclas                | completada | `internal/tui/app.go`                 | Ningún componente de la TUI compara literales de tecla                                |

**Dependencias:** `01`→`02`→`03`; `04`→`05`,`06`,`08`; `06`→`07`; `03`→`09`→`10`; `01`→`12`.
