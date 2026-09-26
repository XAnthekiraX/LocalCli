# 005-task-ollama.md

> T-B005 — ollama: cliente HTTP streaming, razonamiento, perfil de hardware y serialización FIFO.
> Acción de esta descomposición: `crear`.

## Referencias

- `ai/docs/backend/04-infrastructure/INTEGRATIONS.md` [7-14, 43-50] (contrato con Ollama)
- `ai/docs/specs/SPEC-OLLAMA-PERFIL.md` [19-57] (flujo de detección de modelos y perfil)
- `ai/docs/backend/DECISIONS.md` [24-25] (canal FIFO capacidad 1; razonamiento persistido cada 200ms)
- `ai/docs/backend/01-domain/BUSINESS_RULES.md` [61-85] (nodo de contexto y agentes usan el modelo)

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-B005-01 | crear | Crear el cliente HTTP local de Ollama (base URL configurable internamente, sin clave) | completada | `internal/ollama/client.go` | Test con httptest: POST /api/generate responde |
| T-B005-02 | crear | Implementar streaming NDJSON token a token emitiendo eventos por canal | completada | `internal/ollama/stream.go` | Test: fixture NDJSON produce los tokens en orden |
| T-B005-03 | crear | Extraer el campo de razonamiento del stream y emitirlo como evento separado | completada | `internal/ollama/reasoning.go` | Test: respuesta con reasoning separa texto y razonamiento |
| T-B005-04 | crear | Listar modelos disponibles consultando Ollama (equivalente a `ollama list`) | completada | `internal/ollama/models.go` | Test: parsea la respuesta de /api/tags |
| T-B005-05 | crear | Calcular el perfil de hardware y comprobar si el modelo cabe (aviso si no) | completada | `internal/ollama/profile.go` | Test: modelo mayor a RAM disponible devuelve aviso documentado |
| T-B005-06 | crear | Serializar la inferencia con un canal FIFO de capacidad 1 entre sesiones | completada | `internal/ollama/fifo.go` | Test: dos llamadas concurrentes se ejecutan de a una, por orden |
| T-B005-07 | crear | Emitir estado "esperando al modelo" desde el punto único de la cola FIFO | completada | `internal/ollama/fifo.go` | Test: la segunda petición reporta espera mientras la primera genera |
| T-B005-08 | crear | Mapear fallos de conexión (Ollama apagado) al error documentado sin matar la UI | completada | `internal/ollama/errors.go` | Test: connection refused → error tipado de ERRORES.md |
| T-B005-09 | crear | Escribir tests del módulo con servidor httptest de Ollama simulado | completada | `internal/ollama/*_test.go`, `internal/ollama/testdata/` | `go test ./internal/ollama/...` pasa |

**Dependencias:** 01→{02,04,08}; 02→03; {04}→05; {02,03}→06; 06→07; {05,07}→09. Requiere T-B001 completada.
