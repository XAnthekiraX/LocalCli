# 012-task-pegado-bracketed.md

> T-F012 — Cableado del pegado (bracketed paste) y resaltado de tokens en la caja de entrada.
> Acción de esta descomposición: `actualizar`.

## Referencias

- [[frontend/FRONTEND]] §2 Estructura del Proyecto (`components/input/input.go`, `components/input/adjuntos.go`)
- [[frontend/01-domain/DOMAIN]] §1 Componentes (`input`), §2 Reglas de presentación (tokens de adjunto)
- [[frontend/02-interfaces/INTERFACES]] §4 Teclado
- [[specs/SPEC-INTERFAZ]] §2 Entrada de texto
- [[frontend/05-quality/TESTING]] §2 Qué debe probarse (pegado, tokens resaltados)

## Contexto

El motor de tokens (`Anotar`, `Expandir`) y su resaltado (`Estilizar`) ya existían, pero
no estaban cableados al flujo real de la TUI: `input.Update` trataba todo `tea.KeyRunes`
como escritura normal y nunca miraba `Paste`, así que un pegado mostraba la ruta o el
texto crudos en vez de su token; y el render pintaba el texto sin distinguir los tokens.

## Tareas pequeñas

| ID         | Acción     | Tarea                                                                                                   | Estado    | Archivos                                       | Verificación                                                                       |
| ---------- | ---------- | ------------------------------------------------------------------------------------------------------- | --------- | ---------------------------------------------- | --------------------------------------------------------------------------------- |
| T-F012-01  | actualizar | Enrutar el pegado en `Update`: un `tea.KeyRunes` con `Paste` en alto se inserta con `InsertarPegado` (tokens); lo tecleado sigue con `Escribir` | completada | `internal/tui/components/input/input.go`       | Pegar una ruta o varias líneas muestra su token; escribir letra a letra no lo activa |
| T-F012-02  | actualizar | Resaltar los tokens al pintar cada fila: sustituir `Resaltar` (línea completa) por `Estilizar` (segmentos) y usarlo en `filasInteriores` con `EstiloAdjunto` para los tokens y `EstiloBlanco` para el resto | completada | `internal/tui/components/input/adjuntos.go`, `internal/tui/components/input/input.go` | Los tokens se ven en el color de adjunto; el texto visible coincide con lo escrito |
| T-F012-03  | actualizar | Pruebas: pegado multilínea, ruta y carpeta pegadas producen su token y se envían expandidas; el estilo de adjunto tiñe el token | completada | `internal/tui/components/input/input_test.go`  | `go test ./internal/tui/components/input/` en verde                              |

**Dependencias:** `01`→`02`; `01`–`02`→`03`.

## Notas

- La vista de bienvenida usaba su propio modelo (`components/welcome`), sin tokens: un
  pegado ahí seguía escribiéndose crudo. Quedó fuera del alcance de esta tarea; lo
  resuelve `013-task-pegado-bienvenida.md`.
