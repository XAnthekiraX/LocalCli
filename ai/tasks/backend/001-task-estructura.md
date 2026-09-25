# 001-task-estructura.md

> T-B001 — Estructura de carpetas base: los 13 módulos de internal/ con límites por paquete.
> Acción de esta descomposición: `crear`.

## Referencias

- `ai/docs/backend/BACKEND.md` [25-40] (sección 2: estructura del proyecto)
- `ai/docs/backend/01-domain/DOMAIN.md` [9-23] (tabla de módulos: responsabilidad y lo que no hacen)
- `ai/docs/backend/DECISIONS.md` [11] (un solo proceso, canales de Go)

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-B001-01 | crear | Crear el paquete `internal/tui` con doc.go declarando su límite (solo pantalla) | completada | `internal/tui/doc.go` | `go build ./...` pasa |
| T-B001-02 | crear | Crear el paquete `internal/session` con doc.go (ciclo de vida de sesiones, no ejecuta tareas) | completada | `internal/session/doc.go` | `go build ./...` pasa |
| T-B001-03 | crear | Crear el paquete `internal/flow` con doc.go (motor de etapas, sin herramientas ni SQL) | completada | `internal/flow/doc.go` | `go build ./...` pasa |
| T-B001-04 | crear | Crear el paquete `internal/queue` con doc.go (cola derivada del TODO, sin tabla propia) | completada | `internal/queue/doc.go` | `go build ./...` pasa |
| T-B001-05 | crear | Crear el paquete `internal/context` con doc.go (grafo, selección, recorte, auditoría) | completada | `internal/context/doc.go` | `go build ./...` pasa |
| T-B001-06 | crear | Crear el paquete `internal/agent` con doc.go (agentes JSON, despacha a tools) | completada | `internal/agent/doc.go` | `go build ./...` pasa |
| T-B001-07 | crear | Crear el paquete `internal/ollama` con doc.go (único que habla con el modelo) | completada | `internal/ollama/doc.go` | `go build ./...` pasa |
| T-B001-08 | crear | Crear el paquete `internal/task` con doc.go (lee/escribe el TODO) | completada | `internal/task/doc.go` | `go build ./...` pasa |
| T-B001-09 | crear | Crear el paquete `internal/tools` con doc.go (registro, permiso y enrutado) | completada | `internal/tools/doc.go` | `go build ./...` pasa |
| T-B001-10 | crear | Crear el paquete `internal/fileops` con doc.go (frontera de rutas e historial) | completada | `internal/fileops/doc.go` | `go build ./...` pasa |
| T-B001-11 | crear | Crear el paquete `internal/exec` con doc.go (terminal, lista blanca, Landlock) | completada | `internal/exec/doc.go` | `go build ./...` pasa |
| T-B001-12 | crear | Crear el paquete `internal/store` con doc.go (único acceso a SQLite) | completada | `internal/store/doc.go` | `go build ./...` pasa |
| T-B001-13 | crear | Crear el paquete `internal/docs` con doc.go (carga de documentación y grafo, no guarda) | completada | `internal/docs/doc.go` | `go build ./...` pasa |
| T-B001-14 | crear | Escribir un test que verifique que existen los 13 paquetes bajo `internal/` | completada | `internal/estructura_test.go` | `go test ./internal/...` pasa y lista 13 paquetes |

**Dependencias:** ninguna entre las subtareas 01-13 (van en orden alfabético de módulo); T-B001-14 depende de T-B001-01..T-B001-13. Todas dependen de T-B000 (go.mod existente).
