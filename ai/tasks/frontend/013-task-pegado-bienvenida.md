# 013-task-pegado-bienvenida.md

> T-F013 — Tokenizar el pegado en la caja de la bienvenida con el motor de adjuntos compartido.
> Acción de esta descomposición: `actualizar`.

## Referencias

- [[frontend/FRONTEND]] §2 Estructura del Proyecto (`components/adjuntos/`, `components/welcome/`, `components/input/`)
- [[frontend/01-domain/DOMAIN]] §1 Componentes (`welcome`, `input`), §2 Reglas de presentación (tokens de adjunto)
- [[frontend/02-interfaces/INTERFACES]] §5 Estados de espera (pegado)
- [[frontend/05-quality/TESTING]] §2 Qué debe probarse (pegado, tokens resaltados)

## Contexto

La caja de la vista principal tokenizaba el pegado (T-F012), pero la de la **bienvenida**
usa su propio modelo (`components/welcome`) y seguía escribiendo el texto crudo: pegar un
archivo, una carpeta o un texto de varias líneas mostraba la ruta o el texto tal cual, sin
`[nombre.ext]`, `[CARPETA N elementos]` ni `[PEGADO N líneas]`. El motor de tokens vivía
dentro de `components/input`; se extrae a un paquete compartido para que ambas cajas usen
el mismo comportamiento sin duplicarlo.

## Tareas pequeñas

| ID         | Acción     | Tarea                                                                                                   | Estado    | Archivos                                                       | Verificación                                                                         |
| ---------- | ---------- | ------------------------------------------------------------------------------------------------------- | --------- | -------------------------------------------------------------- | ------------------------------------------------------------------------------------ |
| T-F013-01  | actualizar | Extraer el motor de adjuntos a un paquete compartido `components/adjuntos` (tipo `Adjuntos` exportado) y hacer que `input` lo consuma                                                            | completada | `internal/tui/components/adjuntos/adjuntos.go`, `internal/tui/components/input/input.go` | `go build ./...`; las pruebas de `input` siguen en verde                              |
| T-F013-02  | actualizar | Tokenizar el pegado en la bienvenida: `welcome.Model` con `Adjuntos`, `InsertarPegado`, `Peticion` expandida y `Estilizar` al pintar la caja; `editarBienvenida` enruta un `KeyRunes` con `Paste` en alto | completada | `internal/tui/components/welcome/welcome.go`, `internal/tui/components/welcome/caja.go`, `internal/tui/teclado.go` | Pegar en la bienvenida muestra el token y envía el valor real                         |
| T-F013-03  | actualizar | Pruebas: unitarias de `welcome` (multilínea, ruta, carpeta) y a nivel `app` (bienvenida y principal) que enrutan un pegado con `Paste`                                            | completada | `internal/tui/components/welcome/welcome_test.go`, `internal/tui/pegado_test.go` | `go test ./internal/tui/...` en verde |

**Dependencias:** `01`→`02`; `01`–`02`→`03`.

## Notas

- El paquete `components/adjuntos` es la única fuente del comportamiento de tokens
  (archivo/carpeta/multilínea) para las dos cajas de entrada.
- El pegado llega por la combinación de teclas (`Ctrl+Shift+V`) como bracketed paste:
  un `KeyRunes` con `Paste` en alto.
