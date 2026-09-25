# 000-task-stack.md

> T-B000 — Configuración del stack Go: go.mod, dependencias fijas y binario localcli.
> Acción de esta descomposición: `crear`.

## Referencias

- [[backend/BACKEND]] — sección "Stack" y sección 2 "Estructura del Proyecto".
- [[backend/DECISIONS]] — decisiones confirmadas (sin ORM, modernc.org/sqlite, FIFO en ollama).
- [[backend/04-infrastructure/CONFIGURATION]] — comando `localcli`, rutas derivadas, variables de entorno.

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-B000-01 | crear | Módulo Go con nombre `localcli` y versión fijada en go.mod | pendiente | `go.mod` | `go mod tidy` sin errores |
| T-B000-02 | crear | Dependencias externas fijadas: `modernc.org/sqlite`, Bubble Tea + Lip Gloss (sin ORM ni HTTP externo) | pendiente | `go.mod`, `go.sum` | `go list -m all` muestra solo esas dependencias directas |
| T-B000-03 | crear | Punto de entrada del binario `localcli`: la carpeta de ejecución es el proyecto y se derivan las rutas (`ai/docs/`, `ai/tasks/`, SQLite) | pendiente | `main.go` | `go build -o localcli .` genera el ejecutable |
| T-B000-04 | crear | Lectura de las 3 variables opcionales (`LOCALCLI_DB_PATH`, `LOCALCLI_CONTEXT_LIMIT`, `LOCALCLI_ALLOW_INTERNET`) con sus valores por defecto | pendiente | `main.go` | Arrancar sin variables no produce error; con ellas se respetan |
| T-B000-05 | actualizar | `.gitignore`: excluir el binario `localcli` y `.localcli/` según DECISIONS | pendiente | `.gitignore` | `git status` no lista binario ni carpeta `.localcli/` |
| T-B000-06 | crear | Verificación integral del esqueleto del stack compilado | pendiente | — | `go vet ./...` y `go build ./...` sin errores |

Dependencias: 02 requiere 01; 03–05 requieren 01–02; 06 requiere 01–05.
