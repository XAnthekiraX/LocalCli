# 001-task-entrada.md

> T-F001 — Caja de entrada: edición, pegado y adjuntos.
> Acción de esta descomposición: `crear`.

## Referencias

- [[frontend/FRONTEND]] §2 Estructura del Proyecto (`input.go`, `adjuntos.go`), §3 Contratos
- [[frontend/01-domain/DOMAIN]] §1 Componentes (`input`), §2 Reglas de presentación
- [[frontend/02-interfaces/INTERFACES]] §4 Teclado, §5 Estados de espera
- [[specs/SPEC-INTERFAZ]] §2 Entrada de texto
- [[frontend/05-quality/TESTING]] §2 Qué debe probarse (envoltura, pegado, tokens)

## Tareas pequeñas

| ID         | Acción | Tarea                                                                                        | Estado    | Archivos                | Verificación                                                                          |
| ---------- | ------ | -------------------------------------------------------------------------------------------- | --------- | ----------------------- | ------------------------------------------------------------------------------------- |
| T-F001-01  | crear  | Modelo del componente `input`: texto, cursor, filas, agente, modelo y capacidades            | completada | `internal/tui/input.go` | El componente expone esos campos y `Init`/`Update`/`View` compilan                    |
| T-F001-02  | crear  | Render de la caja: fondo `#1E2030`, una celda de aire arriba/abajo/lados, sin glifos de borde | completada | `internal/tui/input.go` | La caja ocupa el ancho con fondo y sin caracteres de borde                            |
| T-F001-03  | crear  | Edición del texto en cualquier punto: flechas y `home`/`end`/`ctrl+b`/`ctrl+e`                | completada | `internal/tui/input.go` | El cursor se mueve a cualquier posición y edita ahí                                   |
| T-F001-04  | crear  | Crecimiento en filas al desbordar el ancho, con tope de filas y desplazamiento interno        | completada | `internal/tui/input.go` | El texto no se recorta; al pasar el tope, la ventana se desplaza                       |
| T-F001-05  | crear  | Envío con `Enter` hacia la sesión activa, sin insertar saltos de línea                        | completada | `internal/tui/input.go` | `Enter` emite la petición con el texto y no añade `\n`                                |
| T-F001-06  | crear  | Pie de la caja: agente activo, modelo y capacidades (`tool`, `pensar`, `[v]`, `[T]`)          | completada | `internal/tui/input.go` | El pie muestra `sí`/`no`/`?` por capacidad y el agente y modelo en uso                |
| T-F001-07  | crear  | Interruptor de razonamiento `pensar [x]`/`pensar [ ]`, pulsable con el ratón                  | completada | `internal/tui/input.go` | El interruptor se alterna con clic y solo se muestra si el modelo declara razonamiento |
| T-F001-08  | crear  | `Tab` alterna el agente activo (`plan`/`build`)                                              | completada | `internal/tui/input.go` | El indicador del agente cambia al instante                                            |
| T-F001-09  | crear  | Conteo de tokens del turno justo debajo de la caja (`tokens: 54k`)                            | completada | `internal/tui/input.go` | El conteo aparece solo con consumo y su formato es una función pura                   |
| T-F001-10  | crear  | Caja pegada al pie de la columna y reajuste del reparto al redimensionar                      | completada | `internal/tui/input.go` | Con historial corto la caja sigue al pie; un `WindowSizeMsg` reajusta el reparto       |
| T-F001-11  | crear  | Tokens de pegado/arrastre: `[nombre.ext]`, `[CARPETA N elementos]`, `[PEGADO N líneas]`       | completada | `internal/tui/adjuntos.go` | Cada elemento se muestra como token; un token por ruta si todas las líneas son rutas   |
| T-F001-12  | crear  | Expansión de los tokens al valor real (ruta o texto) al enviar                               | completada | `internal/tui/adjuntos.go` | Lo que llega a la sesión es la ruta o el texto real, no el token                       |
| T-F001-13  | crear  | Integrar la caja de entrada en `app` y pintarla en la vista principal                        | completada | `internal/tui/app.go`   | La vista principal muestra la caja y `Enter` envía a la sesión activa                 |

**Dependencias:** `01` antes de `02`–`10`; `02`→`04`; `03`→`05`; `06`→`07`; `11`→`12`; `13` tras `01` (y `T-F000-07`).
