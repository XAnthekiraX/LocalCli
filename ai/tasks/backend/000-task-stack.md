# 000-task-stack.md

> T-B000 — Configuración del stack Go: go.mod, dependencias fijas y binario localcli.
> Acción de esta descomposición: `crear`.

## Referencias

- `ai/docs/backend/BACKEND.md` [83-90] (sección Stack)
- `ai/docs/backend/DECISIONS.md` [9-31] (decisiones confirmadas: sin ORM, driver puro Go, binario único)
- `ai/docs/backend/04-infrastructure/CONFIGURATION.md` [5-27] (el comando `localcli` y variables de entorno)
- `ai/docs/PROJECT.md` [49-51] (tecnologías aprobadas)

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-B000-01 | crear | Crear `go.mod` con el módulo `localcli` y la versión de Go fijada | pendiente | `go.mod` | `go mod tidy` corre sin errores |
| T-B000-02 | crear | Fijar las dependencias documentadas: charmbracelet/bubbletea, charmbracelet/lipgloss, modernc.org/sqlite | pendiente | `go.mod`, `go.sum` | `go list -m all` muestra solo esas tres como directas |
| T-B000-03 | crear | Crear el punto de entrada `main.go` del binario que arranca desde la carpeta actual | pendiente | `main.go` | `go build -o localcli .` genera el ejecutable |
| T-B000-04 | crear | Compilar el binario `localcli` en la raíz y verificar que arranca y se sale limpio | pendiente | `localcli` | `./localcli --help` o arranque+salida sin pánico |
| T-B000-05 | actualizar | Añadir al `.gitignore` los artefactos del stack: binario `localcli` y `.localcli/` | pendiente | `.gitignore` | `git status` no lista `localcli` ni `.localcli/` |
| T-B000-06 | crear | Escribir un test mínimo de humo que compile el paquete raíz | pendiente | `main_test.go` | `go test ./...` pasa |

**Dependencias:** T-B000-02 depende de T-B000-01; T-B000-03 depende de T-B000-02; T-B000-04 y T-B000-06 dependen de T-B000-03; T-B000-05 no tiene dependencias.
