# 003-task-chat.md

> T-F003 — Chat: globos, líneas de herramienta y razonamiento.
> Acción de esta descomposición: `crear`.

## Referencias

- [[frontend/FRONTEND]] §2 Estructura del Proyecto (`chat.go`, `reasoning.go`), §3 Contratos
- [[frontend/01-domain/DOMAIN]] §1 Componentes (`chat`), §2 Reglas de presentación
- [[frontend/02-interfaces/INTERFACES]] §1 Eventos que consume, §1.1 La línea de herramienta, §1.2 La línea de sub-proceso, §5 Estados de espera
- [[specs/SPEC-INTERFAZ]] §1 Chat
- [[frontend/05-quality/TESTING]] §2 Qué debe probarse (orden del hilo, tiempos, indicador en vivo)

## Tareas pequeñas

| ID         | Acción | Tarea                                                                                                    | Estado    | Archivos                    | Verificación                                                                       |
| ---------- | ------ | -------------------------------------------------------------------------------------------------------- | --------- | --------------------------- | ---------------------------------------------------------------------------------- |
| T-F003-01  | crear  | Modelo del historial: segmentos y mensajes de la sesión activa                                            | completada | `internal/tui/components/chat/chat.go` | El historial se compone de segmentos de texto y líneas de sistema                  |
| T-F003-02  | crear  | Bloque de mensaje: franja de color por hablante (`#606CD5` usuario derecha, `#4CEE75` agente izquierda), fondo `#1E2030`, texto blanco y celda de aire; sin icono | completada | `internal/tui/components/chat/chat.go` | Cada mensaje ocupa una fila de ancho completo con su franja y aire, sin bordes (icono en conflicto: SPEC §1/DOMAIN §2 «sin icono» vs Disposición/DOMAIN §1 «con icono») |
| T-F003-03  | crear  | Respuesta del modelo con su tiempo de llegada                                                            | completada | `internal/tui/components/chat/turno.go`, `internal/tui/components/chat/vista.go` | La respuesta se pinta y su tiempo se muestra atenuado                            |
| T-F003-04  | crear  | Indicador en vivo (`[⠋ Pensando]`, `[⠋ Usando herramienta: X]`, `[⠋ Generando]`) con tiempo                 | completada | `internal/tui/components/chat/turno.go` | El indicador corre mientras el turno está en curso y se detiene al cerrarse        |
| T-F003-05  | crear  | Línea de herramienta: nace al invocar (verbo + tema) y se completa (marca, medida, recorte, tiempo)      | completada | `internal/tui/components/chat/lineas.go` | Una sola línea que se cierra con `· medida · tiempo`; no vuelca la salida cruda     |
| T-F003-06  | crear  | Orden cronológico del hilo: cerrar globo antes de una línea de sistema y abrir uno nuevo                | completada | `internal/tui/components/chat/chat.go`, `internal/tui/components/chat/lineas.go` | El texto previo a una herramienta queda arriba y el posterior abre un globo nuevo  |
| T-F003-07  | crear  | Línea de sub-proceso `[Sub Proceso] <nombre>`                                                            | completada | `internal/tui/components/chat/formato.go` | Cada etapa que empieza deja una sola línea con su nombre                           |
| T-F003-08  | crear  | Líneas del sistema sueltas (avisos, notificaciones)                                                      | completada | `internal/tui/components/chat/lineas.go`, `internal/tui/components/chat/vista.go` | Las líneas de sistema no llevan bloque de color de hablante                        |
| T-F003-09  | crear  | Recorrido del historial: `↑`/`↓`, `pgup`/`pgdown` y rueda, siguiendo el final                           | completada | `internal/tui/components/chat/vista.go` | Sin subir sigue el final; al subir respeta la posición; indica «↑ N líneas arriba»  |
| T-F003-10  | crear  | Estado del razonamiento en vivo (texto revelado u oculto)                                                | completada | `internal/tui/components/chat/razonamiento.go` | El razonamiento no se vuelca por defecto y crece en vivo sin detener la generación |
| T-F003-11  | crear  | `Ctrl+R` revela u oculta el razonamiento                                                                  | completada | `internal/tui/components/chat/razonamiento.go`, `internal/tui/keymap.go` | La revelación alterna sin detener la generación                |
| T-F003-12  | crear  | Formateadores puros: tiempo de respuesta, medida de herramienta y conteo                                  | completada | `internal/tui/components/chat/formato.go` | Función pura: ms, décimas de segundo y min:seg; una medida ausente no pinta hueco   |
| T-F003-13  | crear  | Integrar el chat en `app` como vista principal                                                           | completada | `internal/tui/app.go` | La vista principal pinta el historial de la sesión activa                           |
| T-F003-14  | actualizar | El chat expone su rectángulo como zona de selección (`zonaChat`)                                    | completada | `internal/tui/app.go`       | Un arrastre iniciado en el chat obtiene su zona                                     |
| T-F003-15  | actualizar | El sidebar expone su rectángulo como zona de selección (`zonaPanel`)                                | completada | `internal/tui/app.go`       | Un arrastre iniciado en el sidebar obtiene su zona                                  |
| T-F003-16  | actualizar | `zonaSeleccion` resuelve la zona por el punto (entrada, chat o sidebar)                              | completada | `internal/tui/selection.go` | Devuelve el rect de la zona que contiene el punto                                   |
| T-F003-17  | actualizar | Corregir el comentario de `componerColumnas`: la selección no cruza chat y sidebar                   | completada | `internal/tui/app.go`       | El comentario deja de afirmar que la selección recorre chat y sidebar a la vez      |
| T-F003-18  | actualizar | Prueba: un arrastre del chat no alcanza el sidebar (y al revés)                                      | completada | `internal/tui/selection_test.go` | El realce no cruza la banda de separación                                       |

**Dependencias:** `01`→`02`,`09`,`12`; `02`→`03`,`04`,`05`,`07`,`08`; `03`–`08`→`06`; `10`→`11`; `02`–`09`→`13`; `13`→`14`,`15`; `14`,`15`→`16`; `16`→`17`,`18`.
