# 006-task-agent.md

> T-B006 — agent: agentes JSON (plan/build), catálogo cerrado y despacho de herramientas a tools.
> Acción de esta descomposición: `crear`.

## Referencias

- `ai/docs/specs/SPEC-AGENTE-BASE.md` [24-110] (dónde se definen, los dos agentes, relevo, reglas)
- `ai/docs/backend/DECISIONS.md` [17, 29] (plan/build separados; JSON con 5 campos, sin herencia)
- `ai/docs/backend/01-domain/DOMAIN.md` [16, 39, 44] (agente en ai/agents/*.json, no código)
- `ai/docs/backend/01-domain/BUSINESS_RULES.md` [75-85] (reglas de agentes)

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-B006-01 | crear | Definir el tipo Agente con los 5 campos exactos: nombre, descripcion, prompt, herramientas, skills | pendiente | `internal/agent/model.go` | Test: round-trip JSON solo con esos campos; hereda_de rechaza |
| T-B006-02 | crear | Cargar los JSON de `ai/agents/*.json` validando estructura y tipos | pendiente | `internal/agent/loader.go` | Test: fixture válido carga; JSON roto da error localizado |
| T-B006-03 | crear | Validar cada herramienta declarada contra el catálogo cerrado de las 13 | pendiente | `internal/agent/catalog.go` | Test: herramienta inexistente rechaza el agente al cargar |
| T-B006-04 | crear | Crear los agentes base `plan.json` (solo lectura) y `build.json` (escritura) como datos | pendiente | `ai/agents/plan.json`, `ai/agents/build.json` | Test: plan no contiene ninguna herramienta de escritura |
| T-B006-05 | crear | Construir la llamada a ollama con el prompt del agente y sus mensajes | pendiente | `internal/agent/run.go` | Test con mock: el prompt enviado es el del JSON |
| T-B006-06 | crear | Despachar las peticiones de herramienta del modelo hacia tools (no ejecutar aquí) | pendiente | `internal/agent/dispatch.go` | Test: tool_call del modelo llega a un stub de tools con su payload |
| T-B006-07 | crear | Implementar el relevo plan→build según lo especificado | pendiente | `internal/agent/handoff.go` | Test: relevo solo ocurre tras aprobación del plan |
| T-B006-08 | crear | Soportar agente con `herramientas` vacío como agente de solo conversación | pendiente | `internal/agent/catalog.go` | Test: agente sin herramientas responde sin ruta de tools |
| T-B006-09 | crear | Escribir tests del módulo con fixtures de agentes | pendiente | `internal/agent/*_test.go`, `internal/agent/testdata/` | `go test ./internal/agent/...` pasa |

**Dependencias:** 01→{02,08}; 03 depende de T-B007-01 (catálogo); {02,03}→04; {01}→05; 05→06; 06→07; {07}→09. Requiere T-B001 y T-B005 completadas.
